package cas

import (
	"context"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type ContentAddressableStorage interface {
	GetAction(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Action, error)
	GetCommand(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Command, error)
	GetDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Directory, error)
	GetFile(ctx context.Context, instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error
	PutFile(ctx context.Context, instance string, path string) (*remoteexecution.Digest, bool, error)
	PutTree(ctx context.Context, instance string, tree *remoteexecution.Tree) (*remoteexecution.Digest, error)
}
