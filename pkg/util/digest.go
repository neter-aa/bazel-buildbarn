package util

import (
	"crypto/sha256"
	"encoding/hex"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
)

func DigestFromData(data []byte) *remoteexecution.Digest {
	hash := sha256.Sum256(data)
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hash[:]),
		SizeBytes: int64(len(data)),
	}
}

func DigestFromMessage(pb proto.Message) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return DigestFromData(data), nil
}
