package blobstore

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"log"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// extractDigest validates the format of fields in a Digest object and returns them.
func extractDigest(digest *remoteexecution.Digest) ([]byte, int64, util.DigestFormat, error) {
	checksum, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, 0, nil, err
	}
	digestFormat, err := util.DigestFormatFromLength(len(digest.Hash))
	if err != nil {
		return nil, 0, nil, err
	}
	if digest.SizeBytes < 0 {
		return nil, 0, nil, fmt.Errorf("Invalid negative size: %d", digest.SizeBytes)
	}
	return checksum, digest.SizeBytes, digestFormat, nil
}

type merkleBlobAccess struct {
	blobAccess BlobAccess
}

// NewMerkleBlobAccess creates an adapter that validates that blobs read
// from and written to storage correspond with the digest that is used
// for identification. It ensures that the size and the SHA-256 based
// checksum match. This is used to ensure clients cannot corrupt the CAS
// and that if corruption were to occur, use of corrupted data is prevented.
func NewMerkleBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &merkleBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *merkleBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	checksum, size, digestFormat, err := extractDigest(digest)
	if err != nil {
		return &errorReader{err: err}
	}
	return &checksumValidatingReader{
		ReadCloser:       ba.blobAccess.Get(ctx, instance, digest),
		expectedChecksum: checksum,
		partialChecksum:  digestFormat(),
		sizeLeft:         size,
		invalidator: func() {
			// Trigger blob deletion in case we detect data
			// corruption. This will cause future calls to
			// FindMissing() to indicate absence, causing clients to
			// re-upload them and/or build actions to be retried.
			if err := ba.blobAccess.Delete(ctx, instance, digest); err == nil {
				log.Printf("Successfully deleted corrupted blob %s", digest)
			} else {
				log.Printf("Failed to delete corrupted blob %s: %s", digest, err)
			}
		},
	}
}

func (ba *merkleBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	checksum, digestSizeBytes, digestFormat, err := extractDigest(digest)
	if err != nil {
		r.Close()
		return err
	}
	if sizeBytes != digestSizeBytes {
		r.Close()
		return fmt.Errorf("Attempted to put object of size %d, whereas the digest contains size %d", sizeBytes, digestSizeBytes)
	}
	return ba.blobAccess.Put(ctx, instance, digest, sizeBytes, &checksumValidatingReader{
		ReadCloser:       r,
		expectedChecksum: checksum,
		partialChecksum:  digestFormat(),
		sizeLeft:         sizeBytes,
		invalidator:      func() {},
	})
}

func (ba *merkleBlobAccess) Delete(ctx context.Context, instance string, digest *remoteexecution.Digest) error {
	_, _, _, err := extractDigest(digest)
	if err != nil {
		return err
	}
	return ba.blobAccess.Delete(ctx, instance, digest)
}

func (ba *merkleBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	for _, digest := range digests {
		_, _, _, err := extractDigest(digest)
		if err != nil {
			return nil, err
		}
	}
	return ba.blobAccess.FindMissing(ctx, instance, digests)
}

type checksumValidatingReader struct {
	io.ReadCloser

	expectedChecksum []byte
	partialChecksum  hash.Hash
	sizeLeft         int64

	// Called whenever size/checksum inconsistencies are detected.
	invalidator func()
}

func (r *checksumValidatingReader) Read(p []byte) (int, error) {
	n, err := io.TeeReader(r.ReadCloser, r.partialChecksum).Read(p)
	nLen := int64(n)
	if nLen > r.sizeLeft {
		r.invalidator()
		return 0, fmt.Errorf("Blob is %d bytes longer than expected", nLen-r.sizeLeft)
	}
	r.sizeLeft -= nLen

	if err == io.EOF {
		if r.sizeLeft != 0 {
			r.invalidator()
			return 0, fmt.Errorf("Blob is %d bytes shorter than expected", r.sizeLeft)
		}

		actualChecksum := r.partialChecksum.Sum(nil)
		if bytes.Compare(actualChecksum, r.expectedChecksum) != 0 {
			r.invalidator()
			return 0, fmt.Errorf(
				"Checksum of blob is %s, while %s was expected",
				hex.EncodeToString(actualChecksum),
				hex.EncodeToString(r.expectedChecksum))
		}
	}
	return n, err
}
