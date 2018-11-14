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
// space is left for a SHA-256 sum. For the size, only four bytes are
// allocated. This shouldn't be a problem in practice, as conversion of
// Digests to SimpleDigests is unidirectional. Furthermore, layers above
// (e.g., MerkleBlobAccess) already ensure integrity.
type SimpleDigest [sha256.Size + 4]byte

// NewSimpleDigest converts a Digest to a SimpleDigest.
func NewSimpleDigest(digest *util.Digest) SimpleDigest {
	var sd SimpleDigest
	copy(sd[:], digest.GetHash())
	binary.LittleEndian.PutUint32(sd[sha256.Size:], uint32(digest.GetSizeBytes()))
	return sd
}
