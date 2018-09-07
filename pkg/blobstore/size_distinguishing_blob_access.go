package blobstore

import (
	"context"
	"io"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type sizeDistinguishingBlobAccess struct {
	smallBlobAccess BlobAccess
	largeBlobAccess BlobAccess
	cutoffSizeBytes int64
}

func NewSizeDistinguishingBlobAccess(smallBlobAccess BlobAccess, largeBlobAccess BlobAccess, cutoffSizeBytes int64) BlobAccess {
	return &sizeDistinguishingBlobAccess{
		smallBlobAccess: smallBlobAccess,
		largeBlobAccess: largeBlobAccess,
		cutoffSizeBytes: cutoffSizeBytes,
	}
}

func (ba *sizeDistinguishingBlobAccess) Get(ctx context.Context, instance string, digest *remoteexecution.Digest) io.ReadCloser {
	if digest.SizeBytes <= ba.cutoffSizeBytes {
		return ba.smallBlobAccess.Get(ctx, instance, digest)
	}
	return ba.largeBlobAccess.Get(ctx, instance, digest)
}

func (ba *sizeDistinguishingBlobAccess) Put(ctx context.Context, instance string, digest *remoteexecution.Digest, r io.ReadCloser) error {
	if digest.SizeBytes <= ba.cutoffSizeBytes {
		return ba.smallBlobAccess.Put(ctx, instance, digest, r)
	}
	return ba.largeBlobAccess.Put(ctx, instance, digest, r)
}

type findMissingResults struct {
	missing []*remoteexecution.Digest
	err     error
}

func callFindMissing(blobAccess BlobAccess, ctx context.Context, instance string, digests []*remoteexecution.Digest) findMissingResults {
	missing, err := blobAccess.FindMissing(ctx, instance, digests)
	return findMissingResults{missing: missing, err: err}
}

func (ba *sizeDistinguishingBlobAccess) FindMissing(ctx context.Context, instance string, digests []*remoteexecution.Digest) ([]*remoteexecution.Digest, error) {
	// Split up digests by size.
	var smallDigests []*remoteexecution.Digest
	var largeDigests []*remoteexecution.Digest
	for _, digest := range digests {
		if digest.SizeBytes <= ba.cutoffSizeBytes {
			smallDigests = append(smallDigests, digest)
		} else {
			largeDigests = append(largeDigests, digest)
		}
	}

	// Forward FindMissing() to both implementations.
	smallResultsChan := make(chan findMissingResults, 1)
	go func() {
		smallResultsChan <- callFindMissing(ba.smallBlobAccess, ctx, instance, smallDigests)
	}()
	largeResults := callFindMissing(ba.largeBlobAccess, ctx, instance, largeDigests)
	smallResults := <-smallResultsChan

	// Recombine results.
	if smallResults.err != nil {
		return nil, smallResults.err
	}
	if largeResults.err != nil {
		return nil, largeResults.err
	}
	return append(smallResults.missing, largeResults.missing...), nil
}
