package blobstore

import (
	"context"
	"io"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type existencePreconditionBlobAccess struct {
	BlobAccess
}

// NewExistencePreconditionBlobAccess wraps a BlobAccess into a version
// that returns GRPC status code "FAILED_PRECONDITION" instead of
// "NOT_FOUND" for Get() operations. This is used by worker processes to
// make Execution::Execute() comply to the protocol.
func NewExistencePreconditionBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &existencePreconditionBlobAccess{
		BlobAccess: blobAccess,
	}
}

func (ba *existencePreconditionBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	return &existencePreconditionReader{
		ReadCloser: ba.BlobAccess.Get(ctx, instance, digest),
	}
}

type existencePreconditionReader struct {
	io.ReadCloser
}

func (r *existencePreconditionReader) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if s := status.Convert(err); s.Code() == codes.NotFound {
		return n, status.Error(codes.FailedPrecondition, s.Message())
	}
	return n, err
}
