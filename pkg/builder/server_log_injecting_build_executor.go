package builder

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"

	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
)

type serverLogInjectingBuildExecutor struct {
	base                      BuildExecutor
	contentAddressableStorage blobstore.BlobAccess
	browserURL                *url.URL
}

// NewServerLogInjectingBuildExecutor forwards execute requests to an underlying
// executor. For responses with a valid action result with a non-zero exit code,
// it generates a URL pointing to bbb_browser that contains information on the
// build failure. This URL is attached to the response as a server log, so that
// the user may access it by running 'bazel build --verbose_failures'.
func NewServerLogInjectingBuildExecutor(base BuildExecutor, contentAddressableStorage blobstore.BlobAccess, browserURL *url.URL) BuildExecutor {
	return &serverLogInjectingBuildExecutor{
		base: base,
		contentAddressableStorage: contentAddressableStorage,
		browserURL:                browserURL,
	}
}

func (be *serverLogInjectingBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, bool) {
	response, mayBeCached := be.base.Execute(ctx, request)
	if response.Result != nil && response.Result.ExitCode != 0 {
		// Generate a body for a log file to present to the user.
		data, err := proto.Marshal(response.Result)
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		browserURL, err := be.browserURL.Parse(
			fmt.Sprintf(
				"/action/%s/%s/%d/%s",
				request.InstanceName,
				request.ActionDigest.Hash,
				request.ActionDigest.SizeBytes,
				base64.URLEncoding.EncodeToString(data)))
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		logBody := []byte(fmt.Sprintf(
			"More details about this build action can be found here:\n" +
				"\n" +
				browserURL.String() + "\n"))

		// Compute a digest of the log file.
		// TODO(edsch): ContentAddressableStorage.PutLog()?
		actionDigest, err := util.NewDigest(request.InstanceName, request.ActionDigest)
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		digestGenerator := actionDigest.NewDigestGenerator()
		digestGenerator.Write(logBody)
		logDigest := digestGenerator.Sum()

		// Store the log in the Content Addressable Storage.
		if err := be.contentAddressableStorage.Put(ctx, logDigest, logDigest.GetSizeBytes(), ioutil.NopCloser(bytes.NewBuffer(logBody))); err != nil {
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
