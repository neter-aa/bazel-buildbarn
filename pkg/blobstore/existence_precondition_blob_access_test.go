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

func TestExistencePreconditionBlobAccessGetSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Let Get() return a reader from which we can read successfully.
	bottomBlobAccess := mock.NewMockBlobAccess(ctrl)
	bottomBlobAccess.EXPECT().Get(context.Background(), "debian8", &remoteexecution.Digest{
		Hash:      "8b1a9953c4611296a827abf8c47804d7",
		SizeBytes: 5,
	}).Return(ioutil.NopCloser(bytes.NewBufferString("Hello")))

	// Validate that the reader can still be read properly.
	r := NewExistencePreconditionBlobAccess(bottomBlobAccess).Get(
		context.Background(), "debian8", &remoteexecution.Digest{
			Hash:      "8b1a9953c4611296a827abf8c47804d7",
			SizeBytes: 5,
		})
	buf, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte("Hello"), buf)
	require.NoError(t, r.Close())
}

// TODO(edsch): Add more unit testing coverage.
