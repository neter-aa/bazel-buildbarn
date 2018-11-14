package circular

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
)

type SimpleDigest [sha256.Size + 4]byte

func NewSimpleDigest(digest *util.Digest) SimpleDigest {
	var sd SimpleDigest
	copy(sd[:], digest.GetHash())
	binary.LittleEndian.PutUint32(sd[sha256.Size:], uint32(digest.GetSizeBytes()))
	return sd
}
