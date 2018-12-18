package blobstore

import (
	"context"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type shardingBlobAccess struct {
	backends           []BlobAccess
	cumulativeWeights  []uint64
	digestKeyFormat    util.DigestKeyFormat
	hashInitialization uint64
}

// convertListToTree converts a list of backends and weights to a
// perfectly balanced binary search tree. The binary search tree is
// stored in an array, just like how a binary heap can be stored in an
// array. Elements are stored in the tree in the same order in which
// they are stored in the original list. Weights are accumulated, so
// that binary searching may be used to obtain the backend for a hashed
// digest.
func convertListToTree(backends []BlobAccess, weights []uint32, backendsTree []BlobAccess, cumulativeWeights []uint64, treeIndex int) {
	if len(backends) > 0 {
		// Determine which element in the list has to be the
		// root in the tree.
		completeTreeSizePlusOne := 2
		for completeTreeSizePlusOne < len(backends)+1 {
			completeTreeSizePlusOne *= 2
		}
		var pivot int
		if len(backends) >= 3*completeTreeSizePlusOne/4 {
			pivot = completeTreeSizePlusOne/2 - 1
		} else {
			pivot = len(backends) - completeTreeSizePlusOne/4
		}

		// Create left and right subtrees.
		leftIndex := treeIndex*2 + 1
		convertListToTree(backends[:pivot], weights[:pivot], backendsTree, cumulativeWeights, leftIndex)
		rightIndex := leftIndex + 1
		convertListToTree(backends[pivot+1:], weights[pivot+1:], backendsTree, cumulativeWeights, rightIndex)

		// Create root and accumulate weights.
		backendsTree[treeIndex] = backends[pivot]
		cumulativeWeights[treeIndex] = uint64(weights[pivot])
		if leftIndex < len(cumulativeWeights) {
			cumulativeWeights[treeIndex] += cumulativeWeights[leftIndex]
		}
		if rightIndex < len(cumulativeWeights) {
			cumulativeWeights[treeIndex] += cumulativeWeights[rightIndex]
		}
	}
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
	backendsTree := make([]BlobAccess, len(weights))
	cumulativeWeights := make([]uint64, len(weights))
	convertListToTree(backends, weights, backendsTree, cumulativeWeights, 0)

	return &shardingBlobAccess{
		backends:           backendsTree,
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
		slot := h % cumulativeWeights[0]
		index := 0
		for {
			indexLeft := index*2 + 1
			if indexLeft >= len(cumulativeWeights) {
				break
			}
			weightLeft := cumulativeWeights[indexLeft]
			if slot < weightLeft {
				index = indexLeft
			} else {
				indexRight := indexLeft + 1
				if indexRight >= len(cumulativeWeights) {
					break
				}
				weightLeftMiddle := cumulativeWeights[index] - cumulativeWeights[indexRight]
				if slot < weightLeftMiddle {
					break
				}
				index = indexRight
				slot -= weightLeftMiddle
			}
		}

		if backend := ba.backends[index]; backend != nil {
			return backend
		}

		// Ended up at a drained backend. Remove this slot from the
		// tree and retry. Repeating this loop will cause another
		// slot to be computed without fully rehashing the key.
		for {
			cumulativeWeights[index]--
			if index == 0 {
				break
			}
			index = (index - 1) / 2
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
