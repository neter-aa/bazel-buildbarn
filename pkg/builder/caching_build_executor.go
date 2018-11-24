package builder

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/url"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/failure"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type cachingBuildExecutor struct {
	base                      BuildExecutor
	contentAddressableStorage cas.ContentAddressableStorage
	actionCache               ac.ActionCache
	browserURL                *url.URL
}

// NewCachingBuildExecutor creates an adapter for BuildExecutor that
// stores action results in the Action Cache (AC) if they may be cached.
// If they may not be cached, they are stored in the Content Addressable
// Storage (CAS) instead.
//
// In both cases, a link to bbb_browser is added to the ExecuteResponse,
// so that the user may inspect the Action and ActionResult in detail.
func NewCachingBuildExecutor(base BuildExecutor, contentAddressableStorage cas.ContentAddressableStorage, actionCache ac.ActionCache, browserURL *url.URL) BuildExecutor {
	return &cachingBuildExecutor{
		base: base,
		contentAddressableStorage: contentAddressableStorage,
		actionCache:               actionCache,
		browserURL:                browserURL,
	}
}

func (be *cachingBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, bool) {
	response, mayBeCached := be.base.Execute(ctx, request)
	if response.Result != nil {
		actionDigest, err := util.NewDigest(request.InstanceName, request.ActionDigest)
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		if mayBeCached {
			// Store result in the Action Cache.
			if err := be.actionCache.PutActionResult(ctx, actionDigest, response.Result); err != nil {
				return convertErrorToExecuteResponse(err), false
			}

			actionURL, err := be.browserURL.Parse(
				fmt.Sprintf(
					"/action/%s/%s/%d/",
					actionDigest.GetInstance(),
					hex.EncodeToString(actionDigest.GetHash()),
					actionDigest.GetSizeBytes()))
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}
			response.Message = "Action details (cached): " + actionURL.String()
		} else {
			// Extension: store the result in the Content
			// Addressable Storage, so the user can at least inspect
			// it through bbb_browser.
			actionDigest, err := util.NewDigest(request.InstanceName, request.ActionDigest)
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}
			actionFailureDigest, err := be.contentAddressableStorage.PutActionFailure(
				ctx,
				&failure.ActionFailure{
					ActionDigest: request.ActionDigest,
					ActionResult: response.Result,
				},
				actionDigest)
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}

			actionFailureURL, err := be.browserURL.Parse(
				fmt.Sprintf(
					"/actionfailure/%s/%s/%d/",
					actionFailureDigest.GetInstance(),
					hex.EncodeToString(actionFailureDigest.GetHash()),
					actionFailureDigest.GetSizeBytes()))
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}
			response.Message = "Action details (uncached): " + actionFailureURL.String()
		}
	}
	return response, mayBeCached
}
