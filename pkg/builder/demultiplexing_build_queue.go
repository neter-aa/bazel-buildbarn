package builder

import (
	"fmt"
	"strings"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/genproto/googleapis/longrunning"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type demultiplexingBuildQueue struct {
	backends map[string]remoteexecution.ExecutionServer
}

// NewDemultiplexingBuildQueue creates an adapter for the Execution
// service to forward requests to different backends backed on the
// instance given in requests. Job identifiers returned by backends are
// prefixed with the instance name, so that successive requests may
// demultiplex the requests later on.
func NewDemultiplexingBuildQueue(backends map[string]remoteexecution.ExecutionServer) remoteexecution.ExecutionServer {
	return &demultiplexingBuildQueue{
		backends: backends,
	}
}

func (bq *demultiplexingBuildQueue) Execute(in *remoteexecution.ExecuteRequest, out remoteexecution.Execution_ExecuteServer) error {
	if strings.ContainsRune(in.InstanceName, '|') {
		return status.Errorf(codes.InvalidArgument, "Instance name cannot contain pipe character")
	}
	backend, ok := bq.backends[in.InstanceName]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Unknown instance name")
	}
	return backend.Execute(in, &operationNamePrepender{
		Execution_ExecuteServer: out,
		prefix:                  in.InstanceName,
	})
}

func (bq *demultiplexingBuildQueue) WaitExecution(in *remoteexecution.WaitExecutionRequest, out remoteexecution.Execution_WaitExecutionServer) error {
	target := strings.SplitN(in.Name, "|", 2)
	if len(target) != 2 {
		return status.Errorf(codes.InvalidArgument, "Unable to extract instance name from watch request")
	}
	backend, ok := bq.backends[target[0]]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "Unknown instance name")
	}
	requestCopy := *in
	requestCopy.Name = target[1]
	return backend.WaitExecution(in, &operationNamePrepender{
		Execution_ExecuteServer: out,
		prefix:                  target[1],
	})
}

type operationNamePrepender struct {
	remoteexecution.Execution_ExecuteServer
	prefix string
}

func (np *operationNamePrepender) Send(operation *longrunning.Operation) error {
	operationCopy := *operation
	operationCopy.Name = fmt.Sprintf("%s|%s", np.prefix, operation.Name)
	return np.Execution_ExecuteServer.Send(&operationCopy)
}
