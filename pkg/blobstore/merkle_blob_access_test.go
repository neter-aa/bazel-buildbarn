package blobstore

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMerkleBlobAccessMalformedDigests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Perform a series of bad calls with invalid digests. None of
	// the calls should end up going to the backend to prevent
	// malformed requests.
	blobAccess := NewMerkleBlobAccess(mock.NewMockBlobAccess(ctrl))
	testBadDigest := func(digest *remoteexecution.Digest, errorMessage string) {
		r := blobAccess.Get(context.Background(), "windows10", digest)
		_, err := ioutil.ReadAll(r)
		require.Error(t, err)
		require.Equal(t, errorMessage, err.Error())
		require.NoError(t, r.Close())

		err = blobAccess.Put(
			context.Background(), "freebsd12", digest, 5,
			ioutil.NopCloser(bytes.NewBufferString("Hello")))
		require.Error(t, err)
		require.Equal(t, errorMessage, err.Error())

		err = blobAccess.Delete(context.Background(), "macos", digest)
		require.Error(t, err)
		require.Equal(t, errorMessage, err.Error())

		_, err = blobAccess.FindMissing(
			context.Background(), "debian8",
			[]*remoteexecution.Digest{digest})
		require.Error(t, err)
		require.Equal(t, errorMessage, err.Error())
	}
	testBadDigest(&remoteexecution.Digest{
		Hash: "cafebabe",
		SizeBytes: 0,
	}, "Unknown digest hash length: 8 characters")
	testBadDigest(&remoteexecution.Digest{
		Hash: "This is a sentence of 32 chars!!",
		SizeBytes: 0,
	}, "Non-hexadecimal character in digest hash: U+0054 'T'")
	testBadDigest(&remoteexecution.Digest{
		Hash: "89D5739BAABBBE65BE35CBE61C88E06D",
		SizeBytes: 0,
	}, "Non-hexadecimal character in digest hash: U+0044 'D'")
	testBadDigest(&remoteexecution.Digest{
		Hash: "e811818f80d9c3c22d577ba83d6196788e553bb408535bb42105cdff726a60ab",
		SizeBytes: -42,
	}, "Invalid digest size: -42 bytes")
}

// TODO(edsch): Validate successful requests.
// TODO(edsch): Test corrupted data.
