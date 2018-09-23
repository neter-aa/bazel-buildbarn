package builder

import (
	"io"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc"
)

type forwardingBuildQueue struct {
	client remoteexecution.ExecutionClient
}

// NewForwardingBuildQueue creates a GRPC service for the Execution
// service that simply forwards all requests to a GRPC client. This may
// be used by the frontend processes to forward execution requests to
// scheduler processes in unmodified form.
func NewForwardingBuildQueue(client *grpc.ClientConn) remoteexecution.ExecutionServer {
	return &forwardingBuildQueue{
		client: remoteexecution.NewExecutionClient(client),
	}
}

func forwardOperations(client remoteexecution.Execution_ExecuteClient, server remoteexecution.Execution_ExecuteServer) error {
	for {
		operation, err := client.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := server.Send(operation); err != nil {
			return err
		}
	}
}

func (bq *forwardingBuildQueue) Execute(in *remoteexecution.ExecuteRequest, out remoteexecution.Execution_ExecuteServer) error {
	client, err := bq.client.Execute(out.Context(), in)
	if err != nil {
		return err
	}
	return forwardOperations(client, out)
}

func (bq *forwardingBuildQueue) WaitExecution(in *remoteexecution.WaitExecutionRequest, out remoteexecution.Execution_WaitExecutionServer) error {
	client, err := bq.client.WaitExecution(out.Context(), in)
	if err != nil {
		return err
	}
	return forwardOperations(client, out)
}
