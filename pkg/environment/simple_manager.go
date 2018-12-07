package environment

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
)

const (
	pathTempRoot  = "/tmp"
	pathBuildRoot = "/build"
)

type simpleManager struct{}

// NewSimpleManager returns an Manager capable of running commands as part of
// the bbb_worker container image.
func NewSimpleManager() Manager {
	return &simpleManager{}
}

func (em *simpleManager) Acquire(platform *remoteexecution.Platform) (Environment, error) {
	// Provide a clean temporary/home directory.
	os.RemoveAll(pathTempRoot)
	if err := os.Mkdir(pathTempRoot, 0777); err != nil {
		return nil, err
	}

	// Provide a clean build directory.
	os.RemoveAll(pathBuildRoot)
	if err := os.Mkdir(pathBuildRoot, 0777); err != nil {
		return nil, err
	}
	buildDirectory, err := filesystem.NewLocalDirectory(pathBuildRoot)
	if err != nil {
		return nil, err
	}

	return &simpleEnvironment{
		buildDirectory: buildDirectory,
	}, nil
}

type simpleEnvironment struct {
	buildDirectory filesystem.Directory
}

func (e *simpleEnvironment) GetBuildDirectory() filesystem.Directory {
	return e.buildDirectory
}

func (e *simpleEnvironment) Run(ctx context.Context, arguments []string, environmentVariables map[string]string, workingDirectory string, stdout io.Writer, stderr io.Writer) (int, error) {
	if len(arguments) < 1 {
		return 0, errors.New("Insufficient number of command arguments")
	}
	cmd := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
	cmd.Dir = filepath.Join(pathBuildRoot, workingDirectory)
	cmd.Env = []string{"HOME=" + pathTempRoot}
	for name, value := range environmentVariables {
		cmd.Env = append(cmd.Env, name+"="+value)
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 1,
			Gid: 1,
		},
	}
	err := cmd.Run()
	if exitError, ok := err.(*exec.ExitError); ok {
		waitStatus := exitError.Sys().(syscall.WaitStatus)
		return waitStatus.ExitStatus(), nil
	}
	return 0, err
}

func (e *simpleEnvironment) Release() {
	e.buildDirectory.Close()

	os.RemoveAll(pathTempRoot)
	os.RemoveAll(pathBuildRoot)
}
