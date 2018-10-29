package blobstore

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMerkleBlobAccessSuccess(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()

	testSuccess := func(digest *remoteexecution.Digest, body []byte) {
		// All calls are expect to go to the backend.
		bottomBlobAccess := mock.NewMockBlobAccess(ctrl)
		bottomBlobAccess.EXPECT().Get(
			ctx, "windows10", digest,
		).Return(ioutil.NopCloser(bytes.NewBuffer(body)))
		bottomBlobAccess.EXPECT().Put(
			ctx, "fedora28", digest, int64(len(body)), gomock.Any(),
		).DoAndReturn(func(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
			buf, err := ioutil.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, body, buf)
			require.NoError(t, r.Close())
			return nil
		})
		bottomBlobAccess.EXPECT().Delete(
			ctx, "ubuntu1804", digest,
		).Return(nil)
		bottomBlobAccess.EXPECT().FindMissing(
			ctx, "solaris11", []*remoteexecution.Digest{digest},
		).Return([]*remoteexecution.Digest{digest}, nil)

		blobAccess := NewMerkleBlobAccess(bottomBlobAccess)

		r := blobAccess.Get(ctx, "windows10", digest)
		buf, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, body, buf)
		require.NoError(t, r.Close())

		require.NoError(t, blobAccess.Put(
			ctx, "fedora28", digest,
			int64(len(body)), ioutil.NopCloser(bytes.NewBuffer(body))))

		require.NoError(t, blobAccess.Delete(ctx, "ubuntu1804", digest))

		missing, err := blobAccess.FindMissing(
			ctx, "solaris11", []*remoteexecution.Digest{digest})
		require.NoError(t, err)
		require.Equal(t, []*remoteexecution.Digest{digest}, missing)
	}
	testSuccess(&remoteexecution.Digest{
		Hash:      "8b1a9953c4611296a827abf8c47804d7",
		SizeBytes: 5,
	}, []byte("Hello"))
	testSuccess(&remoteexecution.Digest{
		Hash:      "a54d88e06612d820bc3be72877c74f257b561b19",
		SizeBytes: 14,
	}, []byte("This is a test"))
	testSuccess(&remoteexecution.Digest{
		Hash:      "1d1f71aecd9b2d8127e5a91fc871833fffe58c5c63aceed9f6fd0b71fe732504",
		SizeBytes: 16,
	}, []byte("And another test"))
}

func TestMerkleBlobAccessMalformedDigests(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()

	// Perform a series of bad calls with invalid digests. None of
	// the calls should end up going to the backend to prevent
	// malformed requests.
	blobAccess := NewMerkleBlobAccess(mock.NewMockBlobAccess(ctrl))
	testBadDigest := func(digest *remoteexecution.Digest, errorMessage string) {
		r := blobAccess.Get(ctx, "windows10", digest)
		_, err := ioutil.ReadAll(r)
		s := status.Convert(err)
		require.Equal(t, codes.InvalidArgument, s.Code())
		require.Equal(t, errorMessage, s.Message())
		require.NoError(t, r.Close())

		err = blobAccess.Put(
			ctx, "freebsd12", digest, 5,
			ioutil.NopCloser(bytes.NewBufferString("Hello")))
		s = status.Convert(err)
		require.Equal(t, codes.InvalidArgument, s.Code())
		require.Equal(t, errorMessage, s.Message())

		err = blobAccess.Delete(ctx, "macos", digest)
		s = status.Convert(err)
		require.Equal(t, codes.InvalidArgument, s.Code())
		require.Equal(t, errorMessage, s.Message())

		_, err = blobAccess.FindMissing(
			ctx, "debian8", []*remoteexecution.Digest{digest})
		s = status.Convert(err)
		require.Equal(t, codes.InvalidArgument, s.Code())
		require.Equal(t, errorMessage, s.Message())
	}
	testBadDigest(&remoteexecution.Digest{
		Hash:      "cafebabe",
		SizeBytes: 0,
	}, "Unknown digest hash length: 8 characters")
	testBadDigest(&remoteexecution.Digest{
		Hash:      "This is a sentence of 32 chars!!",
		SizeBytes: 0,
	}, "Non-hexadecimal character in digest hash: U+0054 'T'")
	testBadDigest(&remoteexecution.Digest{
		Hash:      "89D5739BAABBBE65BE35CBE61C88E06D",
		SizeBytes: 0,
	}, "Non-hexadecimal character in digest hash: U+0044 'D'")
	testBadDigest(&remoteexecution.Digest{
		Hash:      "e811818f80d9c3c22d577ba83d6196788e553bb408535bb42105cdff726a60ab",
		SizeBytes: -42,
	}, "Invalid digest size: -42 bytes")
}

// TODO(edsch): Test corrupted data.
