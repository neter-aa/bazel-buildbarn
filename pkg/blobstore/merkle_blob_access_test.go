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
			ctx, "fedora28", digest, digest.SizeBytes, gomock.Any(),
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
			ctx, "fedora28", digest, digest.SizeBytes,
			ioutil.NopCloser(bytes.NewBuffer(body))))

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
			ctx, "freebsd12", digest, digest.SizeBytes,
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

func TestMerkleBlobAccessMalformedData(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()

	testBadData := func(digest *remoteexecution.Digest, body []byte, errorMessage string) {
		bottomBlobAccess := mock.NewMockBlobAccess(ctrl)

		// A Get() call yielding corrupted data should also
		// trigger a Delete() call on the storage backend, so
		// that inconsistencies are automatically repaired.
		bottomBlobAccess.EXPECT().Get(
			ctx, "freebsd11", digest,
		).Return(ioutil.NopCloser(bytes.NewBuffer(body)))
		bottomBlobAccess.EXPECT().Delete(
			ctx, "freebsd11", digest,
		).Return(nil)

		// A Put() call for uploading broken data does not
		// trigger a Delete(). If broken data ends up being
		// stored, future Get() calls will repair it for us.
		bottomBlobAccess.EXPECT().Put(
			ctx, "fedora28", digest, digest.SizeBytes, gomock.Any(),
		).DoAndReturn(func(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
			_, err := ioutil.ReadAll(r)
			s := status.Convert(err)
			require.Equal(t, codes.InvalidArgument, s.Code())
			require.Equal(t, errorMessage, s.Message())
			require.NoError(t, r.Close())
			return err
		})

		blobAccess := NewMerkleBlobAccess(bottomBlobAccess)

		// A Get() call on corrupt data should trigger an
		// internal error on the server.
		r := blobAccess.Get(ctx, "freebsd11", digest)
		_, err := ioutil.ReadAll(r)
		s := status.Convert(err)
		require.Equal(t, codes.Internal, s.Code())
		require.Equal(t, errorMessage, s.Message())
		require.NoError(t, r.Close())

		// A Put() call for corrupt data should return an
		// invalid argument error instead.
		err = blobAccess.Put(
			ctx, "fedora28", digest, digest.SizeBytes,
			ioutil.NopCloser(bytes.NewBuffer(body)))
		s = status.Convert(err)
		require.Equal(t, codes.InvalidArgument, s.Code())
		require.Equal(t, errorMessage, s.Message())
		require.NoError(t, r.Close())
	}
	testBadData(
		&remoteexecution.Digest{
			Hash:      "3e25960a79dbc69b674cd4ec67a72c62",
			SizeBytes: 11,
		},
		[]byte("Hello"),
		"Blob is 6 bytes shorter than expected")
	testBadData(
		&remoteexecution.Digest{
			Hash:      "8b1a9953c4611296a827abf8c47804d7",
			SizeBytes: 5,
		},
		[]byte("Hello world"),
		"Blob is longer than expected")
	testBadData(
		&remoteexecution.Digest{
			Hash:      "8b1a9953c4611296a827abf8c47804d7",
			SizeBytes: 11,
		},
		[]byte("Hello world"),
		"Checksum of blob is 3e25960a79dbc69b674cd4ec67a72c62, while 8b1a9953c4611296a827abf8c47804d7 was expected")
	testBadData(
		&remoteexecution.Digest{
			Hash:      "f7ff9e8b7bb2e09b70935a5d785e0cc5d9d0abf0",
			SizeBytes: 11,
		},
		[]byte("Hello world"),
		"Checksum of blob is 7b502c3a1f48c8609ae212cdfb639dee39673f5e, while f7ff9e8b7bb2e09b70935a5d785e0cc5d9d0abf0 was expected")
	testBadData(
		&remoteexecution.Digest{
			Hash:      "185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969",
			SizeBytes: 11,
		},
		[]byte("Hello world"),
		"Checksum of blob is 64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c, while 185f8db32271fe25f561a6fc938b2e264306ec304eda518007d1764826381969 was expected")
}
