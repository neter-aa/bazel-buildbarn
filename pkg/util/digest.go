package util

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"log"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DigestFormat acts as a factory for constructing hashers of a certain type.
type DigestFormat func() hash.Hash

// DigestFormatFromLength returns a hasher of a type (MD5, SHA-1 or
// SHA-256) based on the length of an already existent hash. This
// assumes that there cannot be two hashing algorithms that create
// hashes of the same length.
func DigestFormatFromLength(length int) (DigestFormat, error) {
	switch length {
	case md5.Size * 2:
		return md5.New, nil
	case sha1.Size * 2:
		return sha1.New, nil
	case sha256.Size * 2:
		return sha256.New, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "Unknown digest hash length: %d characters", length)
	}
}

// DigestFromData computes a digest for the data in a slice of bytes.
func DigestFromData(data []byte, digestFormat DigestFormat) *remoteexecution.Digest {
	hasher := digestFormat()
	if _, err := hasher.Write(data); err != nil {
		log.Fatal(err)
	}
	return &remoteexecution.Digest{
		Hash:      hex.EncodeToString(hasher.Sum(nil)),
		SizeBytes: int64(len(data)),
	}
}

// DigestFromReader computes a digest for the data in a reader.
func DigestFromReader(r io.Reader, digestFormat DigestFormat) (*remoteexecution.Digest, error) {
	hasher := digestFormat()
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
func DigestFromMessage(pb proto.Message, digestFormat DigestFormat) (*remoteexecution.Digest, error) {
	data, err := proto.Marshal(pb)
	if err != nil {
		return nil, err
	}
	return DigestFromData(data, digestFormat), nil
}
