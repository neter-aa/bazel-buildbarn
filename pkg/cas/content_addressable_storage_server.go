package cas

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type contentAddressableStorageServer struct {
	contentAddressableStorage blobstore.BlobAccess
}

func NewContentAddressableStorageServer(contentAddressableStorage blobstore.BlobAccess) remoteexecution.ContentAddressableStorageServer {
	return &contentAddressableStorageServer{
		contentAddressableStorage: contentAddressableStorage,
	}
}

func (s *contentAddressableStorageServer) FindMissingBlobs(ctx context.Context, in *remoteexecution.FindMissingBlobsRequest) (*remoteexecution.FindMissingBlobsResponse, error) {
	digests, err := s.contentAddressableStorage.FindMissing(ctx, in.InstanceName, in.BlobDigests)
	if err != nil {
		return nil, err
	}
	return &remoteexecution.FindMissingBlobsResponse{
		MissingBlobDigests: digests,
	}, nil
}

func (s *contentAddressableStorageServer) BatchReadBlobs(ctx context.Context, in *remoteexecution.BatchReadBlobsRequest) (*remoteexecution.BatchReadBlobsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "This service does not support batched reading of blobs")
}

func (s *contentAddressableStorageServer) BatchUpdateBlobs(ctx context.Context, in *remoteexecution.BatchUpdateBlobsRequest) (*remoteexecution.BatchUpdateBlobsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "This service does not support batched uploading of blobs")
}

func (s *contentAddressableStorageServer) GetTree(in *remoteexecution.GetTreeRequest, stream remoteexecution.ContentAddressableStorage_GetTreeServer) error {
	return status.Error(codes.Unimplemented, "This service does not support downloading directory trees")
}
