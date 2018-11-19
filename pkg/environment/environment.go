package environment

import (
	"context"
	"io"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
)

// Environment represents a context in which build commands may be
// invoked. Examples of environments may include Docker containers,
// simple chroots, local execution, etc., etc.
type Environment interface {
	// GetBuildDirectory returns a handle to a directory in which a
	// BuildExecutor may place the input files of the build step.
	GetBuildDirectory() filesystem.Directory

	// Run a command within the environment, with a given set of
	// arguments, environment variables, a working directory
	// relative to the build directory and a pair of writers for
	// diagnostics output.
	Run(ctx context.Context, arguments []string, environmentVariables map[string]string, workingDirectory string, stdout io.Writer, stderr io.Writer) (int, error)

	// Release the Environment back to the EnvironmentManager,
	// causing any input/output files to be discarded.
	Release()
}
