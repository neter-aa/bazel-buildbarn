package blobstore

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestExistenceByteStreamServerRead(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()

	// Calls against underlying storage.
	blobAccess := mock.NewMockBlobAccess(ctrl)
	blobAccess.EXPECT().Get(gomock.Any(), "", &remoteexecution.Digest{
		Hash:      "09f7e02f1290be211da707a266f153b3",
		SizeBytes: 5,
	}).Return(ioutil.NopCloser(bytes.NewBufferString("Hello")))
	blobAccess.EXPECT().Get(gomock.Any(), "debian8", &remoteexecution.Digest{
		Hash:      "3538d378083b9afa5ffad767f7269509",
		SizeBytes: 22,
	}).Return(ioutil.NopCloser(bytes.NewBufferString("This is a long message")))

	// Create an RPC server/client pair.
	l := bufconn.Listen(1 << 20)
	s := grpc.NewServer()
	bytestream.RegisterByteStreamServer(s, NewByteStreamServer(blobAccess, 10))
	go func() {
		require.NoError(t, s.Serve(l))
	}()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithDialer(func(string, time.Duration) (net.Conn, error) {
		return l.Dial()
	}), grpc.WithInsecure())
	require.NoError(t, err)
	defer s.Stop()
	defer conn.Close()
	client := bytestream.NewByteStreamClient(conn)

	// Attempt to access a bad resource name.
	req, err := client.Read(ctx, &bytestream.ReadRequest{
		ResourceName: "This is an incorrect resource name",
	})
	require.NoError(t, err)
	_, err = req.Recv()
	status := status.Convert(err)
	require.Equal(t, codes.InvalidArgument, status.Code())
	require.Equal(t, "Invalid resource naming scheme", status.Message())

	// Attempt to fetch the small blob without an instance name.
	req, err = client.Read(ctx, &bytestream.ReadRequest{
		ResourceName: "blobs/09f7e02f1290be211da707a266f153b3/5",
	})
	require.NoError(t, err)
	readResponse, err := req.Recv()
	require.NoError(t, err)
	require.Equal(t, []byte("Hello"), readResponse.Data)
	_, err = req.Recv()
	require.Equal(t, io.EOF, err)

	// Attempt to fetch the large blob with an instance name.
	req, err = client.Read(ctx, &bytestream.ReadRequest{
		ResourceName: "debian8/blobs/3538d378083b9afa5ffad767f7269509/22",
	})
	require.NoError(t, err)
	readResponse, err = req.Recv()
	require.NoError(t, err)
	require.Equal(t, []byte("This is a "), readResponse.Data)
	readResponse, err = req.Recv()
	require.NoError(t, err)
	require.Equal(t, []byte("long messa"), readResponse.Data)
	readResponse, err = req.Recv()
	require.NoError(t, err)
	require.Equal(t, []byte("ge"), readResponse.Data)
	_, err = req.Recv()
	require.Equal(t, io.EOF, err)
}

// TODO(edsch): Add testing coverage for Write() and QueryWriteStatus().
