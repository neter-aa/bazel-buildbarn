package cas

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/failure"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// ContentAddressableStorage provides typed access to a Bazel Content
// Addressable Storage (CAS).
type ContentAddressableStorage interface {
	GetAction(ctx context.Context, digest *util.Digest) (*remoteexecution.Action, error)
	GetActionFailure(ctx context.Context, digest *util.Digest) (*failure.ActionFailure, error)
	GetCommand(ctx context.Context, digest *util.Digest) (*remoteexecution.Command, error)
	GetDirectory(ctx context.Context, digest *util.Digest) (*remoteexecution.Directory, error)
	GetFile(ctx context.Context, digest *util.Digest, outputPath string, isExecutable bool) error
	GetTree(ctx context.Context, digest *util.Digest) (*remoteexecution.Tree, error)

	PutActionFailure(ctx context.Context, failure *failure.ActionFailure, parentDigest *util.Digest) (*util.Digest, error)
	PutFile(ctx context.Context, path string, parentDigest *util.Digest) (*util.Digest, bool, error)
	PutLog(ctx context.Context, log []byte, parentDigest *util.Digest) (*util.Digest, error)
	PutTree(ctx context.Context, tree *remoteexecution.Tree, parentDigest *util.Digest) (*util.Digest, error)
}
