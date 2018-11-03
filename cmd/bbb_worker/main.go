package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/builder"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/scheduler"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/golang/protobuf/proto"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	var (
		browserURLString = flag.String("browser-url", "http://bbb-browser/", "URL of the Bazel Buildbarn Browser, accessible by the user through 'bazel build --verbose_failures'")
		blobstoreConfig  = flag.String("blobstore-config", "/config/blobstore.conf", "Configuration for blob storage")
		schedulerAddress = flag.String("scheduler", "", "Address of the scheduler to which to connect")
	)
	flag.Parse()

	browserURL, err := url.Parse(*browserURLString)
	if err != nil {
		log.Fatal("Failed to parse browser URL: ", err)
	}

	// Respect file permissions that we pass to os.OpenFile(), os.Mkdir(), etc.
	syscall.Umask(0)

	// Web server for metrics and profiling.
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	// Storage access.
	contentAddressableStorageBlobAccess, actionCacheBlobAccess, err := blobstore.CreateBlobAccessObjectsFromConfig(*blobstoreConfig)
	if err != nil {
		log.Fatal("Failed to create blob access: ", err)
	}

	// On-disk caching of content for efficient linking into build environments.
	if err := os.Mkdir("/cache", 0); err != nil {
		log.Fatal("Failed to create cache directory: ", err)
	}

	buildExecutor := builder.NewServerLogInjectingBuildExecutor(
		builder.NewCachingBuildExecutor(
			builder.NewLocalBuildExecutor(
				cas.NewDirectoryCachingContentAddressableStorage(
					cas.NewHardlinkingContentAddressableStorage(
						cas.NewBlobAccessContentAddressableStorage(
							blobstore.NewExistencePreconditionBlobAccess(
								contentAddressableStorageBlobAccess)),
						util.DigestKeyWithoutInstance, "/cache", 10000, 1<<30),
					util.DigestKeyWithoutInstance, 1000)),
			ac.NewBlobAccessActionCache(
				blobstore.NewMetricsBlobAccess(actionCacheBlobAccess, "ac_build_executor"))),
		contentAddressableStorageBlobAccess,
		browserURL)

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
		err := subscribeAndExecute(schedulerClient, buildExecutor, browserURL)
		log.Print("Failed to subscribe and execute: ", err)
		time.Sleep(time.Second * 3)
	}
}

func subscribeAndExecute(schedulerClient scheduler.SchedulerClient, buildExecutor builder.BuildExecutor, browserURL *url.URL) error {
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

		// Print URL in log at the start of the request.
		browserURL.Path = fmt.Sprintf(
			"/action/%s/%s/%d/",
			request.InstanceName,
			request.ActionDigest.Hash,
			request.ActionDigest.SizeBytes)
		log.Print("Action: ", browserURL.String())

		response, _ := buildExecutor.Execute(stream.Context(), request)

		if response.Result != nil {
			// Print the same URL, but with the ActionResult in
			// Base64 appended to it. This link works, even if the
			// ActionResult doesn't end up getting cached.
			// TODO(edsch): Remove duplication with
			// ServerLogInjectingBuildExecutor.
			data, err := proto.Marshal(response.Result)
			if err != nil {
				return err
			}
			browserURL.Path = fmt.Sprintf(
				"/action/%s/%s/%d/%s",
				request.InstanceName,
				request.ActionDigest.Hash,
				request.ActionDigest.SizeBytes,
				base64.URLEncoding.EncodeToString(data))
			log.Print("ActionResult: ", browserURL.String())
		}
		if s := status.FromProto(response.Status); s.Code() != codes.OK {
			log.Print("Error: ", s)
		}

		if err := stream.Send(response); err != nil {
			return err
		}
	}
}
