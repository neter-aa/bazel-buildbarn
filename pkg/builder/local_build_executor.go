package builder

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	"github.com/prometheus/client_golang/prometheus"

	remoteexecution "google.golang.org/genproto/googleapis/devtools/remoteexecution/v1test"
)

const (
	pathTempRoot  = "/tmp"
	pathBuildRoot = "/build"
	pathStdout    = "/stdout"
	pathStderr    = "/stderr"
)

var (
	localBuildExecutorDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "buildbarn",
			Subsystem: "builder",
			Name:      "local_build_executor_duration_seconds",
			Help:      "Amount of time spent per build execution step, in seconds.",
			Buckets:   prometheus.ExponentialBuckets(0.001, math.Pow(10.0, 1.0/3.0), 6*3+1),
		},
		[]string{"step"})
)

func init() {
	prometheus.MustRegister(localBuildExecutorDurationSeconds)
}

func joinPathSafe(elem ...string) (string, error) {
	joined := path.Join(elem...)
	if joined != path.Clean(joined) {
		return "", fmt.Errorf("Attempted to access non-clean path %s", joined)
	}
	return joined, nil
}

type localBuildExecutor struct {
	contentAddressableStorage cas.ContentAddressableStorage
}

func NewLocalBuildExecutor(contentAddressableStorage cas.ContentAddressableStorage) BuildExecutor {
	return &localBuildExecutor{
		contentAddressableStorage: contentAddressableStorage,
	}
}

func (be *localBuildExecutor) createInputDirectory(ctx context.Context, instance string, digest *remoteexecution.Digest, base string) error {
	if err := os.Mkdir(base, 0777); err != nil {
		return err
	}

	directory, err := be.contentAddressableStorage.GetDirectory(ctx, instance, digest)
	if err != nil {
		return err
	}

	for _, file := range directory.Files {
		childPath, err := joinPathSafe(base, file.Name)
		if err != nil {
			return err
		}
		if err := be.contentAddressableStorage.GetFile(ctx, instance, file.Digest, childPath, file.IsExecutable); err != nil {
			return err
		}
	}
	for _, directory := range directory.Directories {
		childPath, err := joinPathSafe(base, directory.Name)
		if err != nil {
			return err
		}
		if err := be.createInputDirectory(ctx, instance, directory.Digest, childPath); err != nil {
			return err
		}
	}
	return nil
}

func (be *localBuildExecutor) prepareFilesystem(ctx context.Context, request *remoteexecution.ExecuteRequest) error {
	// Copy input files into build environment.
	os.RemoveAll(pathBuildRoot)
	if err := be.createInputDirectory(ctx, request.InstanceName, request.Action.InputRootDigest, pathBuildRoot); err != nil {
		return err
	}

	// Ensure that directories where output files are stored are present.
	for _, outputFile := range request.Action.OutputFiles {
		outputPath, err := joinPathSafe(pathBuildRoot, outputFile)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(path.Dir(outputPath), 0777); err != nil {
			return err
		}
	}

	// Provide a clean temp directory.
	os.RemoveAll(pathTempRoot)
	return os.Mkdir(pathTempRoot, 0777)
}

func (be *localBuildExecutor) runCommand(ctx context.Context, request *remoteexecution.ExecuteRequest) error {
	// Fetch command.
	command, err := be.contentAddressableStorage.GetCommand(ctx, request.InstanceName, request.Action.CommandDigest)
	if err != nil {
		return err
	}
	if len(command.Arguments) < 1 {
		return errors.New("Insufficent number of command arguments")
	}

	// Prepare the command to run.
	cmd := exec.CommandContext(ctx, command.Arguments[0], command.Arguments[1:]...)
	cmd.Dir = pathBuildRoot
	cmd.Env = []string{"HOME=" + pathTempRoot}
	for _, environmentVariable := range command.EnvironmentVariables {
		cmd.Env = append(cmd.Env, environmentVariable.Name+"="+environmentVariable.Value)
	}

	// Output streams.
	stdout, err := os.OpenFile(pathStdout, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer stdout.Close()
	cmd.Stdout = stdout
	stderr, err := os.OpenFile(pathStderr, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return err
	}
	defer stderr.Close()
	cmd.Stderr = stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: 1,
			Gid: 1,
		},
	}
	return cmd.Run()
}

func (be *localBuildExecutor) uploadDirectory(ctx context.Context, instance string, basePath string, permitNonExistent bool, children map[string]*remoteexecution.Directory) (*remoteexecution.Directory, error) {
	files, err := ioutil.ReadDir(basePath)
	if err != nil {
		if permitNonExistent && os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var directory remoteexecution.Directory
	for _, file := range files {
		name := file.Name()
		fullPath := path.Join(basePath, name)
		switch file.Mode() & os.ModeType {
		case 0:
			digest, isExecutable, err := be.contentAddressableStorage.PutFile(ctx, instance, fullPath)
			if err != nil {
				return nil, err
			}
			directory.Files = append(directory.Files, &remoteexecution.FileNode{
				Name:         name,
				Digest:       digest,
				IsExecutable: isExecutable,
			})
		case os.ModeDir:
			child, err := be.uploadDirectory(ctx, instance, fullPath, false, children)
			if err != nil {
				return nil, err
			}
			digest, err := util.DigestFromMessage(child)
			if err != nil {
				return nil, err
			}
			children[digest.Hash] = child
			directory.Directories = append(directory.Directories, &remoteexecution.DirectoryNode{
				Name:   name,
				Digest: digest,
			})
		default:
			return nil, fmt.Errorf("Path %s has an unsupported file type", basePath)
		}
	}
	return &directory, nil
}

func (be *localBuildExecutor) uploadTree(ctx context.Context, instance string, path string) (*remoteexecution.Digest, error) {
	// Gather all individual directory objects and turn them into a tree.
	children := map[string]*remoteexecution.Directory{}
	root, err := be.uploadDirectory(ctx, instance, path, true, children)
	if root == nil || err != nil {
		return nil, err
	}
	tree := &remoteexecution.Tree{
		Root: root,
	}
	for _, child := range children {
		tree.Children = append(tree.Children, child)
	}
	return be.contentAddressableStorage.PutTree(ctx, instance, tree)
}

func (be *localBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) *remoteexecution.ExecuteResponse {
	timeStart := time.Now()

	// Set up inputs.
	if err := be.prepareFilesystem(ctx, request); err != nil {
		return convertErrorToExecuteResponse(err)
	}
	timeAfterPrepareFilesytem := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("prepare_filesystem").Observe(
		timeAfterPrepareFilesytem.Sub(timeStart).Seconds())

	// Invoke command.
	exitCode := 0
	if err := be.runCommand(ctx, request); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			exitCode = waitStatus.ExitStatus()
		} else {
			return convertErrorToExecuteResponse(err)
		}
	}
	timeAfterRunCommand := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("run_command").Observe(
		timeAfterRunCommand.Sub(timeAfterPrepareFilesytem).Seconds())

	// Upload command output.
	stdoutDigest, _, err := be.contentAddressableStorage.PutFile(ctx, request.InstanceName, pathStdout)
	if err != nil {
		return convertErrorToExecuteResponse(err)
	}
	stderrDigest, _, err := be.contentAddressableStorage.PutFile(ctx, request.InstanceName, pathStderr)
	if err != nil {
		return convertErrorToExecuteResponse(err)
	}

	response := &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			ExitCode:     int32(exitCode),
			StdoutDigest: stdoutDigest,
			StderrDigest: stderrDigest,
		},
	}

	// Upload output files.
	for _, outputFile := range request.Action.OutputFiles {
		outputPath, err := joinPathSafe(pathBuildRoot, outputFile)
		if err != nil {
			return convertErrorToExecuteResponse(err)
		}
		digest, isExecutable, err := be.contentAddressableStorage.PutFile(ctx, request.InstanceName, outputPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return convertErrorToExecuteResponse(err)
		}
		response.Result.OutputFiles = append(response.Result.OutputFiles, &remoteexecution.OutputFile{
			Path:         outputFile,
			Digest:       digest,
			IsExecutable: isExecutable,
		})
	}

	// Upload output directories.
	for _, outputDirectory := range request.Action.OutputDirectories {
		outputPath, err := joinPathSafe(pathBuildRoot, outputDirectory)
		if err != nil {
			return convertErrorToExecuteResponse(err)
		}
		digest, err := be.uploadTree(ctx, request.InstanceName, outputPath)
		if err != nil {
			return convertErrorToExecuteResponse(err)
		}
		if digest != nil {
			response.Result.OutputDirectories = append(response.Result.OutputDirectories, &remoteexecution.OutputDirectory{
				Path:       outputDirectory,
				TreeDigest: digest,
			})
		}
	}
	timeAfterUpload := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("upload_output").Observe(
		timeAfterUpload.Sub(timeAfterRunCommand).Seconds())

	return response
}
