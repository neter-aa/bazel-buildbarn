package builder

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/proto/failure"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

type serverLogInjectingBuildExecutor struct {
	base                      BuildExecutor
	contentAddressableStorage cas.ContentAddressableStorage
	browserURL                *url.URL
}

// NewServerLogInjectingBuildExecutor forwards execute requests to an underlying
// executor. For responses with a valid action result with a non-zero exit code,
// it stores an ActionFailure entry in the Content Addressable storage,
// containing the Action digest and the ActionResult. It then generates an URL
// for the ActionFailure pointing to bbb_browser. This URL is attached to the
// response as a server log, so that the user may access it by running
// 'bazel build --verbose_failures'.
func NewServerLogInjectingBuildExecutor(base BuildExecutor, contentAddressableStorage cas.ContentAddressableStorage, browserURL *url.URL) BuildExecutor {
	return &serverLogInjectingBuildExecutor{
		base: base,
		contentAddressableStorage: contentAddressableStorage,
		browserURL:                browserURL,
	}
}

func (be *serverLogInjectingBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, bool) {
	response, mayBeCached := be.base.Execute(ctx, request)
	if response.Result != nil && response.Result.ExitCode != 0 {
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

		// Generate a body for a log file to present to the user.
		actionFailureURL, err := be.browserURL.Parse(
			fmt.Sprintf(
				"/actionfailure/%s/%s/%d/",
				actionFailureDigest.GetInstance(),
				hex.EncodeToString(actionFailureDigest.GetHash()),
				actionFailureDigest.GetSizeBytes()))
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		log.Print("ActionFailure: ", actionFailureURL.String())
		logDigest, err := be.contentAddressableStorage.PutLog(
			ctx,
			[]byte(fmt.Sprintf("Build failure details: %s\n", actionFailureURL.String())),
			actionDigest)
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}

		// Attach it to the execute response.
		response.ServerLogs = map[string]*remoteexecution.LogFile{
			"bbb_browser": {
				Digest:        logDigest.GetRawDigest(),
				HumanReadable: true,
			},
		}
	}
	return response, mayBeCached
}
