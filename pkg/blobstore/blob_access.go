package blobstore

import (
	"context"
	"io"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// BlobAccess is an abstraction for a data store that can be used to
// hold both a Bazel Action Cache (AC) and Content Addressable Storage
// (CAS).
type BlobAccess interface {
	Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser
	Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error
	Delete(ctx context.Context, instance string, digest *remoteexecution.Digest) error
	FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error)
}

type errorReader struct {
	err error
}

func (r *errorReader) Read(_ []byte) (int, error) {
	return 0, r.err
}

func (r *errorReader) Close() error {
	return r.err
}
