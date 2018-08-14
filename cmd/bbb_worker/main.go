package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/scheduler"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
)

func main() {
	var (
		redisEndpoint = flag.String("redis-endpoint", "", "Redis endpoint for the Content Addressable Storage and the Action Cache")

		s3Endpoint        = flag.String("s3-endpoint", "", "S3 compatible object storage endpoint for the Content Addressable Storage and the Action Cache")
		s3AccessKeyId     = flag.String("s3-access-key-id", "", "Access key for the object storage")
		s3SecretAccessKey = flag.String("s3-secret-access-key", "", "Secret key for the object storage")
		s3Region          = flag.String("s3-region", "", "Region of the object storage")
		s3DisableSsl      = flag.Bool("s3-disable-ssl", false, "Whether to use HTTP for the object storage instead of HTTPS")

		remoteCache = flag.String("remote", "", "The address of the remote HTTP cache")

		schedulerAddress = flag.String("scheduler", "", "Address of the scheduler to which to connect")
	)
	flag.Parse()

	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

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
		blobstore.NewMerkleBlobAccess(
			casBlobAccess),
		"cas_merkle")

	// On-disk caching of content for efficient linking into build environments.
	if err := os.Mkdir("/cache", 0); err != nil {
		log.Fatal("Failed to create cache directory: ", err)
	}

	buildExecutor := builder.NewCachingBuildExecutor(
		builder.NewLocalBuildExecutor(
			cas.NewDirectoryCachingContentAddressableStorage(
				cas.NewHardlinkingContentAddressableStorage(
					cas.NewBlobAccessContentAddressableStorage(
						contentAddressableStorageBlobAccess),
					util.KeyDigestWithoutInstance, "/cache", 10000, 1<<30),
				util.KeyDigestWithoutInstance, 1000)),
		ac.NewBlobAccessActionCache(
			blobstore.NewMetricsBlobAccess(actionCacheBlobAccess, "ac_build_executor")))

	// Create connection with scheduler.
	schedulerConnection, err := grpc.Dial(
		*schedulerAddress,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor))
	if err != nil {
		log.Fatal("Failed to create scheduler RPC client: ", err)
	}
	schedulerClient := scheduler.NewSchedulerClient(schedulerConnection)

	// Repeatedly ask the scheduler for work.
	for {
		err := subscribeAndExecute(schedulerClient, buildExecutor)
		log.Print("Failed to subscribe and execute: ", err)
		time.Sleep(time.Second * 3)
	}
}

func subscribeAndExecute(schedulerClient scheduler.SchedulerClient, buildExecutor builder.BuildExecutor) error {
	stream, err := schedulerClient.GetWork(context.Background())
	if err != nil {
		return err
	}
	defer stream.CloseSend()

	for {
		request, err := stream.Recv()
		if err != nil {
			return err
		}
		log.Print("Request: ", request)
		response := buildExecutor.Execute(stream.Context(), request)
		log.Print("Response: ", response)
		if err := stream.Send(response); err != nil {
			return err
		}
	}
}
