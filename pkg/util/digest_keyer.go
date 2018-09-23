package util

import (
	"errors"
	"fmt"
	"strings"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

// DigestKeyer is a function that converts a pair of instance and digest
// into a string that may be used to key an object. This may be used for
// internal identification (e.g., map keys) or external identification
// (e.g., keys of objects in Redis).
type DigestKeyer func(instance string, digest *remoteexecution.Digest) (string, error)

// KeyDigestWithInstance creates a key based on both the instance and
// the digest. This is generally needed for the Action Cache (AC), as
// identical operations may have different outcomes based on the
// instance.
func KeyDigestWithInstance(instance string, digest *remoteexecution.Digest) (string, error) {
	if strings.ContainsRune(digest.Hash, '|') {
		return "", errors.New("Blob hash cannot contain pipe character")
	}
	if strings.ContainsRune(instance, '|') {
		return "", errors.New("Instance name cannot contain pipe character")
	}
	return fmt.Sprintf("%s|%d|%s", digest.Hash, digest.SizeBytes, instance), nil
}

// KeyDigestWithoutInstance creates a key based on just the digest,
// ignoring the instance. This is acceptable for the Content Addressable
// Storage (CAS), as it allows identical blobs to be merged across
// instances.
func KeyDigestWithoutInstance(_ string, digest *remoteexecution.Digest) (string, error) {
	if strings.ContainsRune(digest.Hash, '|') {
		return "", errors.New("Blob hash cannot contain pipe character")
	}
	return fmt.Sprintf("%s|%d", digest.Hash, digest.SizeBytes), nil
}
