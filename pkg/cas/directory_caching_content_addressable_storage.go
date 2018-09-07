package cas

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type directoryCachingContentAddressableStorage struct {
	ContentAddressableStorage

	digestKeyer    util.DigestKeyer
	maxDirectories int

	directories map[string]*remoteexecution.Directory
}

func NewDirectoryCachingContentAddressableStorage(base ContentAddressableStorage, digestKeyer util.DigestKeyer, maxDirectories int) ContentAddressableStorage {
	return &directoryCachingContentAddressableStorage{
		ContentAddressableStorage: base,

		digestKeyer:    digestKeyer,
		maxDirectories: maxDirectories,

		directories: map[string]*remoteexecution.Directory{},
	}
}

func (cas *directoryCachingContentAddressableStorage) GetDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Directory, error) {
	key, err := cas.digestKeyer(instance, digest)
	if err != nil {
		return nil, err
	}
	if directory, ok := cas.directories[key]; ok {
		return directory, nil
	}
	directory, err := cas.ContentAddressableStorage.GetDirectory(ctx, instance, digest)
	if err != nil {
		return nil, err
	}
	// TODO(edsch): Respect maxDirectories.
	cas.directories[key] = directory
	return directory, nil
}
