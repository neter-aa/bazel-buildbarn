package circular

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

// SimpleDigest is the on-disk format for digests that the circular
// storage backend uses.
//
// Digests are encoded by storing the hash, followed by the size. Enough
// space is left for a SHA-256 sum.
type SimpleDigest [sha256.Size + 8]byte

// NewSimpleDigest converts a Digest to a SimpleDigest.
func NewSimpleDigest(digest *util.Digest) SimpleDigest {
	var sd SimpleDigest
	copy(sd[:], digest.GetHashBytes())
	binary.LittleEndian.PutUint32(sd[sha256.Size:], uint32(digest.GetSizeBytes()))
	return sd
}
