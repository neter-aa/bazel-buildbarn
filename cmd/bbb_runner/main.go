package main

import (
	"flag"
	"log"
	"net"

	"github.com/EdSchouten/bazel-buildbarn/pkg/environment"
	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/runner"

	"google.golang.org/grpc"
)

func main() {
	var (
		listenPath = flag.String("listen-path", "", "Path on which this process should bind its UNIX socket to wait for incoming requests through GRPC")
	)
	flag.Parse()

	rootDirectory, err := filesystem.NewLocalDirectory(".")
	if err != nil {
		log.Fatal("Failed to open current directory: ", err)
	}

	s := grpc.NewServer()
	runner.RegisterRunnerServer(s, environment.NewLocalExecutionEnvironment(rootDirectory, "."))

	sock, err := net.Listen("unix", *listenPath)
	if err != nil {
		log.Fatal("Failed to create listening socket: ", err)
	}
	if err := s.Serve(sock); err != nil {
		log.Fatal("Failed to serve RPC server: ", err)
	}
}
