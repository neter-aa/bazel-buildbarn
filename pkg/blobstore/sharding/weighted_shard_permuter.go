package sharding

type weightedShardPermuter struct {
	originalIndices   []int
	cumulativeWeights []uint64
}

// convertListToTree converts a list of weights to a perfectly balanced
// binary tree. The binary tree is stored in an array, just like how a
// binary heap can be stored in an array. Weights are stored in the
// tree in the same order in which they are stored in the original list,
// except that they are cumulative. This allows for logarithmic time
// lookups and adjustments.
func convertListToTree(weights []uint32, originalIndices []int, cumulativeWeights []uint64, listIndex int, treeIndex int) {
	if len(weights) > 0 {
		// Determine which element in the list has to be the
		// root in the tree.
		completeTreeSizePlusOne := 2
		for completeTreeSizePlusOne < len(weights)+1 {
			completeTreeSizePlusOne *= 2
		}
		var pivot int
		if len(weights) >= 3*completeTreeSizePlusOne/4 {
			pivot = completeTreeSizePlusOne/2 - 1
		} else {
			pivot = len(weights) - completeTreeSizePlusOne/4
		}

		// Create left and right subtrees.
		leftIndex := treeIndex*2 + 1
		convertListToTree(weights[:pivot], originalIndices, cumulativeWeights, 0, leftIndex)
		rightIndex := leftIndex + 1
		convertListToTree(weights[pivot+1:], originalIndices, cumulativeWeights, pivot+1, rightIndex)

		// Accumulate weights.
		originalIndices[treeIndex] = listIndex + pivot
		cumulativeWeights[treeIndex] = uint64(weights[pivot])
		if leftIndex < len(cumulativeWeights) {
			cumulativeWeights[treeIndex] += cumulativeWeights[leftIndex]
		}
		if rightIndex < len(cumulativeWeights) {
			cumulativeWeights[treeIndex] += cumulativeWeights[rightIndex]
		}
	}
}

// NewWeightedShardPermuter is a shard selection algorithm that
// generates a permutation of [0, len(weights)) for every hash, where
// every index i is returned weights[i] times. This makes it possible to
// have storage backends with different specifications in terms of
// capacity and throughput, giving them a proportional amount of
// traffic.
func NewWeightedShardPermuter(weights []uint32) ShardPermuter {
	s := &weightedShardPermuter{
		originalIndices:   make([]int, len(weights)),
		cumulativeWeights: make([]uint64, len(weights)),
	}
	convertListToTree(weights, s.originalIndices, s.cumulativeWeights, 0, 0)
	return s
}

func (s *weightedShardPermuter) GetShard(hash uint64, selector ShardSelector) {
	cumulativeWeights := make([]uint64, len(s.cumulativeWeights))
	copy(cumulativeWeights, s.cumulativeWeights)
	for {
		// Perform binary search to find corresponding backend.
		slot := hash % cumulativeWeights[0]
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

		if !selector(s.originalIndices[index]) {
			return
		}

		// Ended up at a drained backend. Remove this slot from the
		// tree and retry. Repeating this loop will cause another
		// slot to be computed without fully rehashing the key.
		//
		// TODO(edsch): Is there some kind of constant memory
		// trick we could use here instead? Using a power of two
		// cumulative weight and triangular numbers has the
		// downside of yielding the same sequence for every key
		// hasing to the same initial slot.
		for {
			cumulativeWeights[index]--
			if index == 0 {
				break
			}
			index = (index - 1) / 2
		}
	}
}
