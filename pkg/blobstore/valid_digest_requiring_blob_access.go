package blobstore

import (
	"context"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func validateDigest(digest *remoteexecution.Digest) error {
	_, err := util.DigestFormatFromLength(len(digest.Hash))
	if err != nil {
		return err
	}
	for _, c := range digest.Hash {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return status.Errorf(codes.InvalidArgument, "Non-hexadecimal character in digest hash: %#U", c)
		}
	}
	if digest.SizeBytes < 0 {
		return status.Errorf(codes.InvalidArgument, "Invalid digest size: %d bytes", digest.SizeBytes)
	}
	return nil
}

type validDigestRequiringBlobAccess struct {
	blobAccess BlobAccess
}

// NewValidDigestRequiringBlobAccess creates an adapter that validates
// the format of digests, ensuring that layers below don't have to deal
// with degenerate digests (e.g., containing negative sizes or digests
// of odd lengths or containing bad characters).
func NewValidDigestRequiringBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &validDigestRequiringBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *validDigestRequiringBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	if err := validateDigest(digest); err != nil {
		return util.NewErrorReader(err)
	}
	return ba.blobAccess.Get(ctx, instance, digest)
}

func (ba *validDigestRequiringBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	if err := validateDigest(digest); err != nil {
		r.Close()
		return err
	}
	return ba.blobAccess.Put(ctx, instance, digest, sizeBytes, r)
}

func (ba *validDigestRequiringBlobAccess) Delete(ctx context.Context, instance string, digest *remoteexecution.Digest) error {
	if err := validateDigest(digest); err != nil {
		return err
	}
	return ba.blobAccess.Delete(ctx, instance, digest)
}

func (ba *validDigestRequiringBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	for _, digest := range digests {
		if err := validateDigest(digest); err != nil {
			return nil, err
		}
	}
	return ba.blobAccess.FindMissing(ctx, instance, digests)
}
