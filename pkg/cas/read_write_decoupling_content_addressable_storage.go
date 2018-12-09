package cas

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/failure"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type readWriteDecouplingContentAddressableStorage struct {
	reader ContentAddressableStorage
	writer ContentAddressableStorage
}

// NewReadWriteDecouplingContentAddressableStorage takes a pair of
// ContentAddressableStorage objects and forwards reads and write
// requests to them, respectively. It can, for example, be used to
// forward read requests to a process-wide cache, while write requests
// are sent to a worker/action-specific write cache.
func NewReadWriteDecouplingContentAddressableStorage(reader ContentAddressableStorage, writer ContentAddressableStorage) ContentAddressableStorage {
	return &readWriteDecouplingContentAddressableStorage{
		reader: reader,
		writer: writer,
	}
}

func (cas *readWriteDecouplingContentAddressableStorage) GetAction(ctx context.Context, digest *util.Digest) (*remoteexecution.Action, error) {
	return cas.reader.GetAction(ctx, digest)
}

func (cas *readWriteDecouplingContentAddressableStorage) GetActionFailure(ctx context.Context, digest *util.Digest) (*failure.ActionFailure, error) {
	return cas.reader.GetActionFailure(ctx, digest)
}

func (cas *readWriteDecouplingContentAddressableStorage) GetCommand(ctx context.Context, digest *util.Digest) (*remoteexecution.Command, error) {
	return cas.reader.GetCommand(ctx, digest)
}

func (cas *readWriteDecouplingContentAddressableStorage) GetDirectory(ctx context.Context, digest *util.Digest) (*remoteexecution.Directory, error) {
	return cas.reader.GetDirectory(ctx, digest)
}

func (cas *readWriteDecouplingContentAddressableStorage) GetFile(ctx context.Context, digest *util.Digest, directory filesystem.Directory, name string, isExecutable bool) error {
	return cas.reader.GetFile(ctx, digest, directory, name, isExecutable)
}

func (cas *readWriteDecouplingContentAddressableStorage) GetTree(ctx context.Context, digest *util.Digest) (*remoteexecution.Tree, error) {
	return cas.reader.GetTree(ctx, digest)
}

func (cas *readWriteDecouplingContentAddressableStorage) PutActionFailure(ctx context.Context, actionFailure *failure.ActionFailure, parentDigest *util.Digest) (*util.Digest, error) {
	return cas.writer.PutActionFailure(ctx, actionFailure, parentDigest)
}

func (cas *readWriteDecouplingContentAddressableStorage) PutFile(ctx context.Context, directory filesystem.Directory, name string, parentDigest *util.Digest) (*util.Digest, error) {
	return cas.writer.PutFile(ctx, directory, name, parentDigest)
}

func (cas *readWriteDecouplingContentAddressableStorage) PutLog(ctx context.Context, log []byte, parentDigest *util.Digest) (*util.Digest, error) {
	return cas.writer.PutLog(ctx, log, parentDigest)
}

func (cas *readWriteDecouplingContentAddressableStorage) PutTree(ctx context.Context, tree *remoteexecution.Tree, parentDigest *util.Digest) (*util.Digest, error) {
	return cas.writer.PutTree(ctx, tree, parentDigest)
}
