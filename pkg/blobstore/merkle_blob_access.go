package blobstore

import (
	"bytes"
	"context"
	"encoding/hex"
	"hash"
	"io"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type merkleBlobAccess struct {
	BlobAccess
}

// NewMerkleBlobAccess creates an adapter that validates that blobs read
// from and written to storage correspond with the digest that is used
// for identification. It ensures that the size and the SHA-256 based
// checksum match. This is used to ensure clients cannot corrupt the CAS
// and that if corruption were to occur, use of corrupted data is prevented.
func NewMerkleBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &merkleBlobAccess{
		BlobAccess: blobAccess,
	}
}

func (ba *merkleBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	return newChecksumValidatingReader(
		digest,
		ba.BlobAccess.Get(ctx, instance, digest),
		func() {
			// Trigger blob deletion in case we detect data
			// corruption. This will cause future calls to
			// FindMissing() to indicate absence, causing clients to
			// re-upload them and/or build actions to be retried.
			if err := ba.BlobAccess.Delete(ctx, instance, digest); err == nil {
				log.Printf("Successfully deleted corrupted blob %s", digest)
			} else {
				log.Printf("Failed to delete corrupted blob %s: %s", digest, err)
			}
		},
		codes.Internal)
}

func (ba *merkleBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	if digest.SizeBytes != sizeBytes {
		log.Fatal("Called into CAS to store non-CAS object")
	}
	return ba.BlobAccess.Put(
		ctx, instance, digest, digest.SizeBytes,
		newChecksumValidatingReader(digest, r, func() {}, codes.InvalidArgument))
}

type checksumValidatingReader struct {
	io.ReadCloser

	expectedChecksum []byte
	partialChecksum  hash.Hash
	sizeLeft         int64

	// Called whenever size/checksum inconsistencies are detected.
	invalidator func()
	errorCode   codes.Code
}

func newChecksumValidatingReader(digest *remoteexecution.Digest, r io.ReadCloser, invalidator func(), errorCode codes.Code) io.ReadCloser {
	digestFormat, err := util.DigestFormatFromLength(len(digest.Hash))
	if err != nil {
		log.Fatal("Failed to obtain format of digest, even though its contents have already been validated")
	}
	checksum, _ := hex.DecodeString(digest.Hash)
	if err != nil {
		log.Fatal("Failed to decode digest hash, even though its contents have already been validated")
	}
	return &checksumValidatingReader{
		ReadCloser:       r,
		expectedChecksum: checksum,
		partialChecksum:  digestFormat(),
		sizeLeft:         digest.SizeBytes,
		invalidator:      invalidator,
		errorCode:        errorCode,
	}
}

func (r *checksumValidatingReader) Read(p []byte) (int, error) {
	n, err := io.TeeReader(r.ReadCloser, r.partialChecksum).Read(p)
	nLen := int64(n)
	if nLen > r.sizeLeft {
		r.invalidator()
		return 0, status.Error(r.errorCode, "Blob is longer than expected")
	}
	r.sizeLeft -= nLen

	if err == io.EOF {
		if r.sizeLeft != 0 {
			r.invalidator()
			return 0, status.Errorf(r.errorCode, "Blob is %d bytes shorter than expected", r.sizeLeft)
		}

		actualChecksum := r.partialChecksum.Sum(nil)
		if bytes.Compare(actualChecksum, r.expectedChecksum) != 0 {
			r.invalidator()
			return 0, status.Errorf(
				r.errorCode,
				"Checksum of blob is %s, while %s was expected",
				hex.EncodeToString(actualChecksum),
				hex.EncodeToString(r.expectedChecksum))
		}
	}
	return n, err
}
