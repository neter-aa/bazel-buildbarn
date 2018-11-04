package builder

import (
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type BuildQueue interface {
	remoteexecution.CapabilitiesServer
	remoteexecution.ExecutionServer
}
