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
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/bytestream"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestExistenceByteStreamServer(t *testing.T) {
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
	blobAccess.EXPECT().Get(gomock.Any(), "fedora28", &remoteexecution.Digest{
		Hash:      "09f34d28e9c8bb445ec996388968a9e8",
		SizeBytes: 7,
	}).Return(util.NewErrorReader(status.Error(codes.NotFound, "Blob not found")))

	blobAccess.EXPECT().Put(gomock.Any(), "", &remoteexecution.Digest{
		Hash:      "94876e5b1ce62c7b2b5ff6e661624841",
		SizeBytes: 14,
	}, int64(14), gomock.Any()).DoAndReturn(func(ctx context.Context, instance string, digest *remoteexecution.Digest, sizeBytes int64, r io.ReadCloser) error {
		buf, err := ioutil.ReadAll(r)
		require.NoError(t, err)
		require.Equal(t, []byte("LaputanMachine"), buf)
		require.NoError(t, r.Close())
		return nil
	})

	// Create an RPC server/client pair.
	l := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	bytestream.RegisterByteStreamServer(server, NewByteStreamServer(blobAccess, 10))
	go func() {
		require.NoError(t, server.Serve(l))
	}()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithDialer(func(string, time.Duration) (net.Conn, error) {
		return l.Dial()
	}), grpc.WithInsecure())
	require.NoError(t, err)
	defer server.Stop()
	defer conn.Close()
	client := bytestream.NewByteStreamClient(conn)

	// Attempt to access a bad resource name.
	req, err := client.Read(ctx, &bytestream.ReadRequest{
		ResourceName: "This is an incorrect resource name",
	})
	require.NoError(t, err)
	_, err = req.Recv()
	s := status.Convert(err)
	require.Equal(t, codes.InvalidArgument, s.Code())
	require.Equal(t, "Invalid resource naming scheme", s.Message())

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

	// Attempt to fetch a nonexistent blob.
	req, err = client.Read(ctx, &bytestream.ReadRequest{
		ResourceName: "///fedora28//blobs/09f34d28e9c8bb445ec996388968a9e8/////7/",
	})
	require.NoError(t, err)
	_, err = req.Recv()
	s = status.Convert(err)
	require.Equal(t, codes.NotFound, s.Code())
	require.Equal(t, "Blob not found", s.Message())

	// Attempt to write to a bad resource name.
	stream, err := client.Write(ctx)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&bytestream.WriteRequest{
		ResourceName: "This is an incorrect resource name",
		Data:         []byte("Bleep bloop!"),
	}))
	_, err = stream.CloseAndRecv()
	s = status.Convert(err)
	require.Equal(t, codes.InvalidArgument, s.Code())
	require.Equal(t, "Invalid resource naming scheme", s.Message())

	// Attempt to write a blob without an instance name.
	stream, err = client.Write(ctx)
	require.NoError(t, err)
	require.NoError(t, stream.Send(&bytestream.WriteRequest{
		ResourceName: "uploads/7de747e0-ab6b-4d83-90cb-11989f84c473/blobs/94876e5b1ce62c7b2b5ff6e661624841/14",
		Data:         []byte("Laputan"),
	}))
	require.NoError(t, stream.Send(&bytestream.WriteRequest{
		Data:        []byte("Machine"),
		WriteOffset: 7,
		FinishWrite: true,
	}))
	response, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.Equal(t, int64(14), response.CommittedSize)

	// TODO(edsch): Add testing coverage for invalid WriteOffset,
	// lack of FinishWrite, etc.
}

// TODO(edsch): Add testing coverage QueryWriteStatus()?
