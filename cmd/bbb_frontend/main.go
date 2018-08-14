package main

import (
	"flag"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/genproto/googleapis/bytestream"
	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	watcher "google.golang.org/genproto/googleapis/watcher/v1"
	"google.golang.org/grpc"
)

type stringList []string

func (i *stringList) String() string {
	return "my string representation"
}

func (i *stringList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	var schedulersList stringList
	var (
		redisEndpoint = flag.String("redis-endpoint", "", "Redis endpoint for the Content Addressable Storage and the Action Cache")

		s3Endpoint        = flag.String("s3-endpoint", "", "S3 compatible object storage endpoint for the Content Addressable Storage and the Action Cache")
		s3AccessKeyId     = flag.String("s3-access-key-id", "", "Access key for the object storage")
		s3SecretAccessKey = flag.String("s3-secret-access-key", "", "Secret key for the object storage")
		s3Region          = flag.String("s3-region", "", "Region of the object storage")
		s3DisableSsl      = flag.Bool("s3-disable-ssl", false, "Whether to use HTTP for the object storage instead of HTTPS")

		remoteCache = flag.String("remote", "", "The address of the remote HTTP cache")
	)
	flag.Var(&schedulersList, "scheduler", "Backend capable of executing build actions. Example: debian9|hostname-of-debian9-scheduler:8981")
	flag.Parse()

	// Web server for metrics and profiling.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	// Create an S3 client. Set the uploader concurrency to 1 to drastically reduce memory usage.
	// TODO(edsch): Maybe the concurrency can be left alone for this process?
	session := session.New(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(*s3AccessKeyId, *s3SecretAccessKey, ""),
		Endpoint:         s3Endpoint,
		Region:           s3Region,
		DisableSSL:       s3DisableSsl,
		S3ForcePathStyle: aws.Bool(true),
	})
	s3 := s3.New(session)
	uploader := s3manager.NewUploader(session)
	uploader.Concurrency = 1

	var casBlobAccess blobstore.BlobAccess
	var actionCacheBlobAccess blobstore.BlobAccess

	if *remoteCache == "" {

		casBlobAccess = blobstore.NewSizeDistinguishingBlobAccess(
			blobstore.NewMetricsBlobAccess(
				blobstore.NewRedisBlobAccess(
					redis.NewClient(
						&redis.Options{
							Addr: *redisEndpoint,
							DB:   0,
						}),
					util.KeyDigestWithoutInstance),
				"cas_redis"),
			blobstore.NewMetricsBlobAccess(
				blobstore.NewS3BlobAccess(
					s3,
					uploader,
					aws.String("content-addressable-storage"),
					util.KeyDigestWithoutInstance),
				"cas_s3"),
			1<<20)

		actionCacheBlobAccess = blobstore.NewMetricsBlobAccess(
			blobstore.NewRedisBlobAccess(
				redis.NewClient(
					&redis.Options{
						Addr: *redisEndpoint,
						DB:   1,
					}),
				util.KeyDigestWithInstance),
			"ac_redis")
	} else {
		casBlobAccess = blobstore.NewMetricsBlobAccess(
			blobstore.NewRemoteBlobAccess(*remoteCache, "cas"),
			"cas_remote")
		actionCacheBlobAccess = blobstore.NewMetricsBlobAccess(
			blobstore.NewRemoteBlobAccess(*remoteCache, "ac"),
			"ac_remote")
	}

	// Storage of content and actions.
	contentAddressableStorageBlobAccess := blobstore.NewMetricsBlobAccess(
		blobstore.NewMerkleBlobAccess(casBlobAccess),
		"cas_merkle")

	actionCache := ac.NewBlobAccessActionCache(actionCacheBlobAccess)

	// Backends capable of compiling.
	schedulers := map[string]builder.BuildQueue{}
	for _, schedulerEntry := range schedulersList {
		components := strings.SplitN(schedulerEntry, "|", 2)
		if len(components) != 2 {
			log.Fatal("Invalid scheduler entry: ", schedulerEntry)
		}
		scheduler, err := grpc.Dial(
			components[1],
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
			grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor))
		if err != nil {
			log.Fatal("Failed to create scheduler RPC client: ", err)
		}
		schedulers[components[0]] = builder.NewForwardingBuildQueue(scheduler)
	}
	buildQueue := builder.NewDemultiplexingBuildQueue(schedulers)

	// RPC server.
	s := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	remoteexecution.RegisterActionCacheServer(s, ac.NewActionCacheServer(actionCache))
	remoteexecution.RegisterContentAddressableStorageServer(s, cas.NewContentAddressableStorageServer(contentAddressableStorageBlobAccess))
	bytestream.RegisterByteStreamServer(s, blobstore.NewByteStreamServer(contentAddressableStorageBlobAccess))
	remoteexecution.RegisterExecutionServer(s, buildQueue)
	watcher.RegisterWatcherServer(s, buildQueue)
	grpc_prometheus.EnableHandlingTimeHistogram()
	grpc_prometheus.Register(s)

	sock, err := net.Listen("tcp", ":8980")
	if err != nil {
		log.Fatal("Failed to create listening socket: ", err)
	}
	if err := s.Serve(sock); err != nil {
		log.Fatal("Failed to serve RPC server: ", err)
	}
}
