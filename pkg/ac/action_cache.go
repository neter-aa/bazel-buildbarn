package ac

import (
	"context"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type ActionCache interface {
	GetActionResult(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.ActionResult, error)
	PutActionResult(ctx context.Context, instance string, digest *remoteexecution.Digest, result *remoteexecution.ActionResult) error
}
