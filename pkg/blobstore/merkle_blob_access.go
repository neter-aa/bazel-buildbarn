package blobstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// extractDigest validates the format of fields in a Digest object and returns them.
func extractDigest(digest *remoteexecution.Digest) ([]byte, int64, error) {
	checksum, err := hex.DecodeString(digest.Hash)
	if err != nil {
		return nil, 0, err
	}
	if len(checksum) != sha256.Size {
		return nil, 0, fmt.Errorf("Expected checksum to be %d bytes; not %d", sha256.Size, len(checksum))
	}
	if digest.SizeBytes < 0 {
		return nil, 0, fmt.Errorf("Invalid negative size: %d", digest.SizeBytes)
	}
	return checksum, digest.SizeBytes, nil
}

type merkleBlobAccess struct {
	blobAccess BlobAccess
}

func NewMerkleBlobAccess(blobAccess BlobAccess) BlobAccess {
	return &merkleBlobAccess{
		blobAccess: blobAccess,
	}
}

func (ba *merkleBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	checksum, size, err := extractDigest(digest)
	if err != nil {
		return &errorReader{err: err}
	}
	return &checksumValidatingReader{
		ReadCloser:       ba.blobAccess.Get(ctx, instance, digest),
		expectedChecksum: checksum,
		partialChecksum:  sha256.New(),
		sizeLeft:         size,
	}
}

func (ba *merkleBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
	checksum, digestSizeBytes, err := extractDigest(digest)
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
		partialChecksum:  sha256.New(),
		sizeLeft:         sizeBytes,
	})
}

func (ba *merkleBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	for _, digest := range digests {
		_, _, err := extractDigest(digest)
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
}

func (r *checksumValidatingReader) Read(p []byte) (int, error) {
	n, err := io.TeeReader(r.ReadCloser, r.partialChecksum).Read(p)
	nLen := int64(n)
	if nLen > r.sizeLeft {
		return 0, fmt.Errorf("Blob is %d bytes longer than expected", nLen-r.sizeLeft)
	}
	r.sizeLeft -= nLen

	if err == io.EOF {
		if r.sizeLeft != 0 {
			return 0, fmt.Errorf("Blob is %d bytes shorter than expected", r.sizeLeft)
		}

		actualChecksum := r.partialChecksum.Sum(nil)
		if bytes.Compare(actualChecksum, r.expectedChecksum) != 0 {
			return 0, fmt.Errorf(
				"Checksum of blob is %s, while %s was expected",
				hex.EncodeToString(actualChecksum),
				hex.EncodeToString(r.expectedChecksum))
		}
	}
	return n, err
}
