package cas

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
)

type blobAccessContentAddressableStorage struct {
	blobAccess blobstore.BlobAccess
}

// NewBlobAccessContentAddressableStorage creates a
// ContentAddressableStorage that reads and writes Content Adressable
// Storage (CAS) objects from a BlobAccess based store.
func NewBlobAccessContentAddressableStorage(blobAccess blobstore.BlobAccess) ContentAddressableStorage {
	return &blobAccessContentAddressableStorage{
		blobAccess: blobAccess,
	}
}

func (cas *blobAccessContentAddressableStorage) GetAction(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Action, error) {
	r := cas.blobAccess.Get(ctx, instance, digest)
	data, err := ioutil.ReadAll(r)
	r.Close()
	if err != nil {
		return nil, err
	}
	var action remoteexecution.Action
	if err := proto.Unmarshal(data, &action); err != nil {
		return nil, err
	}
	return &action, nil
}

func (cas *blobAccessContentAddressableStorage) GetCommand(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Command, error) {
	r := cas.blobAccess.Get(ctx, instance, digest)
	data, err := ioutil.ReadAll(r)
	r.Close()
	if err != nil {
		return nil, err
	}
	var command remoteexecution.Command
	if err := proto.Unmarshal(data, &command); err != nil {
		return nil, err
	}
	return &command, nil
}

func (cas *blobAccessContentAddressableStorage) GetDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest) (*remoteexecution.Directory, error) {
	r := cas.blobAccess.Get(ctx, instance, digest)
	data, err := ioutil.ReadAll(r)
	r.Close()
	if err != nil {
		return nil, err
	}
	var directory remoteexecution.Directory
	if err := proto.Unmarshal(data, &directory); err != nil {
		return nil, err
	}
	return &directory, nil
}

func (cas *blobAccessContentAddressableStorage) GetFile(ctx context.Context, instance string, digest *remoteexecution.Digest, outputPath string, isExecutable bool) error {
	var mode os.FileMode = 0444
	if isExecutable {
		mode = 0555
	}
	w, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	defer w.Close()

	r := cas.blobAccess.Get(ctx, instance, digest)
	_, err = io.Copy(w, r)
	r.Close()

	// Ensure no traces are left behind upon failure.
	if err != nil {
		os.Remove(outputPath)
	}
	return err
}

func (cas *blobAccessContentAddressableStorage) PutFile(ctx context.Context, instance string, path string) (*remoteexecution.Digest, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}

	// Determine whether the file is executable.
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, false, err
	}

	// Walk through the file to compute the digest.
	digest, err := util.DigestFromReader(file)
	if err != nil {
		file.Close()
		return nil, false, err
	}

	// Rewind and store it.
	if _, err := file.Seek(0, 0); err != nil {
		file.Close()
		return nil, false, err
	}
	if err := cas.blobAccess.Put(ctx, instance, digest, digest.SizeBytes, file); err != nil {
		return nil, false, err
	}
	return digest, (info.Mode() & 0111) != 0, nil
}

func (cas *blobAccessContentAddressableStorage) PutTree(ctx context.Context, instance string, tree *remoteexecution.Tree) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(tree)
	if err != nil {
		return nil, err
	}
	digest := util.DigestFromData(data)
	if err := cas.blobAccess.Put(ctx, instance, digest, int64(len(data)), ioutil.NopCloser(bytes.NewBuffer(data))); err != nil {
		return nil, err
	}
	return digest, nil
}
