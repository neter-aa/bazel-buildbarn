package cas

import (
	"context"
	"math/rand"
	"os"
	"path"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type hardlinkingContentAddressableStorage struct {
	ContentAddressableStorage

	digestKeyer util.DigestKeyer
	path        string
	maxFiles    int
	maxSize     int64

	filesPresentList      []string
	filesPresentSize      map[string]int64
	filesPresentTotalSize int64
}

func NewHardlinkingContentAddressableStorage(base ContentAddressableStorage, digestKeyer util.DigestKeyer, path string, maxFiles int, maxSize int64) ContentAddressableStorage {
	return &hardlinkingContentAddressableStorage{
		ContentAddressableStorage: base,

		digestKeyer: digestKeyer,
		path:        path,
		maxFiles:    maxFiles,
		maxSize:     maxSize,

		filesPresentSize: map[string]int64{},
	}
}

func (cas *hardlinkingContentAddressableStorage) makeSpace(size int64) error {
	for len(cas.filesPresentList) > 0 && (len(cas.filesPresentList) >= cas.maxFiles || cas.filesPresentTotalSize+size > cas.maxSize) {
		// Remove random file from disk.
		idx := rand.Intn(len(cas.filesPresentList))
		key := cas.filesPresentList[idx]
		if err := os.Remove(path.Join(cas.path, key)); err != nil {
			return err
		}

		// Remove file from bookkeeping.
		cas.filesPresentTotalSize -= cas.filesPresentSize[key]
		delete(cas.filesPresentSize, key)
		last := len(cas.filesPresentList) - 1
		cas.filesPresentList[idx] = cas.filesPresentList[last]
		cas.filesPresentList = cas.filesPresentList[:last]
	}
	return nil
}

func (cas *hardlinkingContentAddressableStorage) GetFile(ctx context.Context, instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error {
	key, err := cas.digestKeyer(instance, digest)
	if err != nil {
		return err
	}
	if isExecutable {
		key += "+x"
	} else {
		key += "-x"
	}

	cachePath := path.Join(cas.path, key)
	if _, ok := cas.filesPresentSize[key]; !ok {
		if err := cas.makeSpace(digest.SizeBytes); err != nil {
			return err
		}
		if err := cas.ContentAddressableStorage.GetFile(ctx, instance, digest, cachePath, isExecutable); err != nil {
			return err
		}
		cas.filesPresentList = append(cas.filesPresentList, key)
		cas.filesPresentSize[key] = digest.SizeBytes
		cas.filesPresentTotalSize += digest.SizeBytes
	}
	return os.Link(cachePath, outputPath)
}
