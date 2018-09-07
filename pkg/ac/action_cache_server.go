package ac

import (
	"context"

	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type actionCacheServer struct {
	actionCache  ActionCache
	allowUpdates bool
}

func NewActionCacheServer(actionCache ActionCache, allowUpdates bool) remoteexecution.ActionCacheServer {
	return &actionCacheServer{
		actionCache:  actionCache,
		allowUpdates: allowUpdates,
	}
}

func (s *actionCacheServer) GetActionResult(ctx context.Context, in *remoteexecution.GetActionResultRequest) (*remoteexecution.ActionResult, error) {
	return s.actionCache.GetActionResult(ctx, in.InstanceName, in.ActionDigest)
}

func (s *actionCacheServer) UpdateActionResult(ctx context.Context, in *remoteexecution.UpdateActionResultRequest) (*remoteexecution.ActionResult, error) {
	if !s.allowUpdates {
		return nil, status.Error(codes.Unimplemented, "This service can only be used to get action results")
	}
	return in.ActionResult, s.actionCache.PutActionResult(ctx, in.InstanceName, in.ActionDigest, in.ActionResult)
}
