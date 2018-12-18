package sharding

// ShardProposer is the callback type called by ShardSelector.GetShard.
// It is invoked until false is returned, providing a backend index
// number for every call.
type ShardProposer func(int) bool

// ShardSelector is an algorithm for turning a hash into a series of
// indices corresponding to backends capable of serving blobs
// corresponding with that hash.
//
// As backends may be unavailable (e.g., drained) or replication
// strategies may be applied to duplicate blobs, it is important that an
// actual permutation is returned to ensure every backend is given a
// chance. It is permitted to spuriously generate the same index
// multiple times.
type ShardSelector interface {
	GetShard(hash uint64, propose ShardProposer)
}
