package blobstore

import (
	"context"
	"io"
	"sort"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type shardingBlobAccess struct {
	backends           []BlobAccess
	cumulativeWeights  []uint64
	digestKeyFormat    util.DigestKeyFormat
	hashInitialization uint64
}

// NewShardingBlobAccess is an adapter for BlobAccess that partitions
// requests across backends by hashing the digest. Every backend has a
// weight that acts as a ratio on how much of the key space needs to end
// up at that backend.
//
// Backends may be drained by making them nil. This causes their keys to
// be redistributed over other backends. Draining a backend should not
// cause keys to be redistributed that would normally end up at other
// backends.
func NewShardingBlobAccess(backends []BlobAccess, weights []uint32, digestKeyFormat util.DigestKeyFormat, hashInitialization uint64) BlobAccess {
	// Compute cumulative weights for binary searching.
	var cumulativeWeights []uint64
	totalWeight := uint64(0)
	for _, weight := range weights {
		totalWeight += uint64(weight)
		cumulativeWeights = append(cumulativeWeights, totalWeight)
	}

	return &shardingBlobAccess{
		backends:           backends,
		cumulativeWeights:  cumulativeWeights,
		digestKeyFormat:    digestKeyFormat,
		hashInitialization: hashInitialization,
	}
}

func (ba *shardingBlobAccess) getBackend(digest *util.Digest) BlobAccess {
	// Hash the key using FNV-1a.
	h := ba.hashInitialization
	for _, c := range digest.GetKey(ba.digestKeyFormat) {
		h ^= uint64(c)
		h *= 1099511628211
	}

	cumulativeWeights := make([]uint64, len(ba.cumulativeWeights))
	copy(cumulativeWeights, ba.cumulativeWeights)
	for {
		// Perform binary search to find corresponding backend.
		slot := h % cumulativeWeights[len(cumulativeWeights)-1]
		idx := sort.Search(len(cumulativeWeights), func(i int) bool {
			return slot < cumulativeWeights[i]
		})
		if backend := ba.backends[idx]; backend != nil {
			return backend
		}

		// Ended up at a drained backend. Remove this slot from the
		// table and retry. Repeating this loop will cause another
		// slot to be computed without fully rehashing the key.
		for i := idx; i < len(cumulativeWeights); i++ {
			cumulativeWeights[i]--
		}
	}
}

func (ba *shardingBlobAccess) Get(ctx context.Context, digest *util.Digest) (int64, io.ReadCloser, error) {
	return ba.getBackend(digest).Get(ctx, digest)
}

func (ba *shardingBlobAccess) Put(ctx context.Context, digest *util.Digest, sizeBytes int64, r io.ReadCloser) error {
	return ba.getBackend(digest).Put(ctx, digest, sizeBytes, r)
}

func (ba *shardingBlobAccess) Delete(ctx context.Context, digest *util.Digest) error {
	return ba.getBackend(digest).Delete(ctx, digest)
}

func (ba *shardingBlobAccess) FindMissing(ctx context.Context, digests []*util.Digest) ([]*util.Digest, error) {
	// Determine which backends to contact.
	digestsPerBackend := map[BlobAccess][]*util.Digest{}
	for _, digest := range digests {
		backend := ba.getBackend(digest)
		digestsPerBackend[backend] = append(digestsPerBackend[backend], digest)
	}

	// Asynchronously call FindMissing() on backends.
	resultsChan := make(chan findMissingResults, len(digestsPerBackend))
	for backend, digests := range digestsPerBackend {
		go func(backend BlobAccess, digests []*util.Digest) {
			resultsChan <- callFindMissing(ctx, backend, digests)
		}(backend, digests)
	}

	// Recombine results.
	var missingDigests []*util.Digest
	var err error
	for i := 0; i < len(digestsPerBackend); i++ {
		results := <-resultsChan
		if results.err == nil {
			missingDigests = append(missingDigests, results.missing...)
		} else {
			err = results.err
		}
	}
	return missingDigests, err
}
