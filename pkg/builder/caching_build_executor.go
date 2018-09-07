package builder

import (
	"context"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type cachingBuildExecutor struct {
	base        BuildExecutor
	actionCache ac.ActionCache
}

func NewCachingBuildExecutor(base BuildExecutor, actionCache ac.ActionCache) BuildExecutor {
	return &cachingBuildExecutor{
		base:        base,
		actionCache: actionCache,
	}
}

func (be *cachingBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, bool) {
	response, mayBeCached := be.base.Execute(ctx, request)
	if mayBeCached {
		if err := be.actionCache.PutActionResult(ctx, request.InstanceName, request.ActionDigest, response.Result); err != nil {
			return convertErrorToExecuteResponse(err), false
		}
	}
	return response, mayBeCached
}
