package blobstore

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type emptyBlobFilteringBlobAccess struct {
	blobAccess BlobAccess
}

// NewEmptyBlobFilteringBlobAccess creates a BlobAccess that filters out
// requests for blobs having a digest with size zero. In addition to
// preventing unnecessary requests on storage backends, some of them may
// even not be capable of storing empty blobs in the first place.
func NewEmptyBlobFilteringBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &emptyBlobFilteringBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *emptyBlobFilteringBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	if digest.SizeBytes == 0 {
		return ioutil.NopCloser(bytes.NewBuffer(nil))
	}
	return ba.blobAccess.Get(ctx, instance, digest)
}

func (ba *emptyBlobFilteringBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	if digest.SizeBytes == 0 {
		return r.Close()
	}
	return ba.blobAccess.Put(ctx, instance, digest, sizeBytes, r)
}

func (ba *emptyBlobFilteringBlobAccess) Delete(ctx context.Context, instance string, digest *remoteexecution.Digest) error {
	if digest.SizeBytes == 0 {
		return nil
	}
	return ba.blobAccess.Delete(ctx, instance, digest)
}

func (ba *emptyBlobFilteringBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	var nonEmptyDigests []*remoteexecution.Digest
	for _, digest := range digests {
		if digest.SizeBytes != 0 {
			nonEmptyDigests = append(nonEmptyDigests, digest)
		}
	}
	return ba.blobAccess.FindMissing(ctx, instance, nonEmptyDigests)
}
