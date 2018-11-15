package blobstore

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
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

func (ba *existencePreconditionBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	length, r, err := ba.BlobAccess.Get(ctx, digest)
	if s := status.Convert(err); s.Code() == codes.NotFound {
		s, err := status.New(codes.FailedPrecondition, s.Message()).WithDetails(
			&errdetails.PreconditionFailure{
				Violations: []*errdetails.PreconditionFailure_Violation{
					{
						Type:    "MISSING",
						Subject: fmt.Sprintf("blobs/%s/%d", hex.EncodeToString(digest.GetHash()), digest.GetSizeBytes()),
					},
				},
			})
		if err != nil {
			return 0, nil, err
		}
		return 0, nil, s.Err()
	}
	return length, r, err
}
