package util

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
)

// DigestFromData computes a digest for the data in a slice of bytes.
func DigestFromData(data []byte) *remoteexecution.Digest {
	hash := sha256.Sum256(data)
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hash[:]),
		SizeBytes: int64(len(data)),
	}
}

// DigestFromReader computes a digest for the data in a reader.
func DigestFromReader(r io.Reader) (*remoteexecution.Digest, error) {
	hasher := sha256.New()
	n, err := io.Copy(hasher, r)
	if err != nil {
		return nil, err
	}
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: int64(n),
	}, nil
}

// DigestFromMessage computes a digest for a Protobuf message.
func DigestFromMessage(pb proto.Message) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return DigestFromData(data), nil
}
