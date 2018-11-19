package builder

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"strings"
	"time"

	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/environment"
	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
	"github.com/prometheus/client_golang/prometheus"
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

type localBuildExecutor struct {
	contentAddressableStorage cas.ContentAddressableStorage
	environmentManager        environment.Manager
	logsDirectory             filesystem.Directory
}

// NewLocalBuildExecutor returns a BuildExecutor that executes build
// steps on the local system.
func NewLocalBuildExecutor(contentAddressableStorage cas.ContentAddressableStorage, environmentManager environment.Manager, logsDirectory filesystem.Directory) BuildExecutor {
	return &localBuildExecutor{
		contentAddressableStorage: contentAddressableStorage,
		environmentManager:        environmentManager,
		logsDirectory:             logsDirectory,
	}
}

func (be *localBuildExecutor) createInputDirectory(ctx context.Context, digest *util.Digest, inputDirectory filesystem.Directory) error {
	directory, err := be.contentAddressableStorage.GetDirectory(ctx, digest)
	if err != nil {
		return err
	}

	for _, file := range directory.Files {
		childDigest, err := digest.NewDerivedDigest(file.Digest)
		if err != nil {
			return err
		}
		if err := be.contentAddressableStorage.GetFile(ctx, childDigest, inputDirectory, file.Name, file.IsExecutable); err != nil {
			return err
		}
	}
	for _, directory := range directory.Directories {
		childDigest, err := digest.NewDerivedDigest(directory.Digest)
		if err != nil {
			return err
		}
		if err := inputDirectory.Mkdir(directory.Name, 0777); err != nil {
			return err
		}
		childDirectory, err := inputDirectory.Enter(directory.Name)
		if err != nil {
			return err
		}
		err = be.createInputDirectory(ctx, childDigest, childDirectory)
		childDirectory.Close()
		if err != nil {
			return err
		}
	}
	for _, symlink := range directory.Symlinks {
		if err := inputDirectory.Symlink(symlink.Target, symlink.Name); err != nil {
			return err
		}
	}
	return nil
}

func (be *localBuildExecutor) runCommand(ctx context.Context, command *remoteexecution.Command, environment environment.Environment) (int, error) {
	environmentVariables := map[string]string{}
	for _, environmentVariable := range command.EnvironmentVariables {
		environmentVariables[environmentVariable.Name] = environmentVariable.Value
	}

	stdout, err := be.logsDirectory.OpenFile("stdout", os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return 0, err
	}
	defer stdout.Close()

	stderr, err := be.logsDirectory.OpenFile("stderr", os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0)
	if err != nil {
		return 0, err
	}
	defer stderr.Close()

	return environment.Run(ctx, command.Arguments, environmentVariables, command.WorkingDirectory, stdout, stderr)
}

func (be *localBuildExecutor) uploadDirectory(ctx context.Context, outputDirectory filesystem.Directory, parentDigest *util.Digest, children map[string]*remoteexecution.Directory) (*remoteexecution.Directory, error) {
	files, err := outputDirectory.ReadDir()
	if err != nil {
		return nil, err
	}

	var directory remoteexecution.Directory
	for _, file := range files {
		name := file.Name()
		switch file.Mode() & os.ModeType {
		case 0:
			digest, isExecutable, err := be.contentAddressableStorage.PutFile(ctx, outputDirectory, name, parentDigest)
			if err != nil {
				return nil, err
			}
			directory.Files = append(directory.Files, &remoteexecution.FileNode{
				Name:         name,
				Digest:       digest.GetRawDigest(),
				IsExecutable: isExecutable,
			})
		case os.ModeDir:
			childDirectory, err := outputDirectory.Enter(name)
			if err != nil {
				return nil, err
			}
			child, err := be.uploadDirectory(ctx, childDirectory, parentDigest, children)
			childDirectory.Close()
			if err != nil {
				return nil, err
			}

			// Compute digest of the child directory. This requires serializing it.
			data, err := proto.Marshal(child)
			if err != nil {
				return nil, err
			}
			digestGenerator := parentDigest.NewDigestGenerator()
			if _, err := digestGenerator.Write(data); err != nil {
				return nil, err
			}
			digest := digestGenerator.Sum()

			children[digest.GetKey(util.DigestKeyWithoutInstance)] = child
			directory.Directories = append(directory.Directories, &remoteexecution.DirectoryNode{
				Name:   name,
				Digest: digest.GetRawDigest(),
			})
		case os.ModeSymlink:
			target, err := outputDirectory.Readlink(name)
			if err != nil {
				return nil, err
			}
			directory.Symlinks = append(directory.Symlinks, &remoteexecution.SymlinkNode{
				Name:   name,
				Target: target,
			})
		default:
			return nil, fmt.Errorf("File %s has an unsupported file type", name)
		}
	}
	return &directory, nil
}

func (be *localBuildExecutor) uploadTree(ctx context.Context, outputDirectory filesystem.Directory, parentDigest *util.Digest) (*util.Digest, error) {
	// Gather all individual directory objects and turn them into a tree.
	children := map[string]*remoteexecution.Directory{}
	root, err := be.uploadDirectory(ctx, outputDirectory, parentDigest, children)
	if err != nil {
		return nil, err
	}
	tree := &remoteexecution.Tree{
		Root: root,
	}
	for _, child := range children {
		tree.Children = append(tree.Children, child)
	}
	return be.contentAddressableStorage.PutTree(ctx, tree, parentDigest)
}

func (be *localBuildExecutor) createOutputParentDirectory(buildDirectory filesystem.Directory, path string) (filesystem.Directory, error) {
	components := strings.FieldsFunc(path, func(r rune) bool { return r == '/' })

	// Create and enter first component.
	if err := buildDirectory.Mkdir(components[0], 0777); err != nil && !os.IsExist(err) {
		return nil, err
	}
	d, err := buildDirectory.Enter(components[0])
	if err != nil {
		return nil, err
	}

	// Create and enter successive components, closing the former.
	for _, component := range components[1:] {
		if err := d.Mkdir(component, 0777); err != nil && !os.IsExist(err) {
			d.Close()
			return nil, err
		}
		d2, err := d.Enter(component)
		d.Close()
		if err != nil {
			return nil, err
		}
		d = d2
	}
	return d, nil
}

func (be *localBuildExecutor) Execute(ctx context.Context, request *remoteexecution.ExecuteRequest) (*remoteexecution.ExecuteResponse, bool) {
	timeStart := time.Now()

	// Fetch action and command.
	actionDigest, err := util.NewDigest(request.InstanceName, request.ActionDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	action, err := be.contentAddressableStorage.GetAction(ctx, actionDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	commandDigest, err := actionDigest.NewDerivedDigest(action.CommandDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	command, err := be.contentAddressableStorage.GetCommand(ctx, commandDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	timeAfterGetActionCommand := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("get_action_command").Observe(
		timeAfterGetActionCommand.Sub(timeStart).Seconds())

	environment, err := be.environmentManager.Acquire(command.Platform)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	defer environment.Release()

	// Set up inputs.
	inputRootDigest, err := actionDigest.NewDerivedDigest(action.InputRootDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	buildDirectory := environment.GetBuildDirectory()
	if err := be.createInputDirectory(ctx, inputRootDigest, buildDirectory); err != nil {
		return convertErrorToExecuteResponse(err), false
	}

	// Create and open parent directories of where we expect to see output.
	// Build rules generally expect the parent directories to already be
	// there. We later use the directory handles to extract output files.
	outputParentDirectories := map[string]filesystem.Directory{}
	for _, outputDirectory := range command.OutputDirectories {
		dirPath := path.Dir(outputDirectory)
		if _, ok := outputParentDirectories[dirPath]; !ok {
			dir, err := be.createOutputParentDirectory(buildDirectory, dirPath)
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}
			outputParentDirectories[dirPath] = dir
			defer dir.Close()
		}
	}
	for _, outputFile := range command.OutputFiles {
		dirPath := path.Dir(outputFile)
		if _, ok := outputParentDirectories[dirPath]; !ok {
			dir, err := be.createOutputParentDirectory(buildDirectory, dirPath)
			if err != nil {
				return convertErrorToExecuteResponse(err), false
			}
			outputParentDirectories[dirPath] = dir
			defer dir.Close()
		}
	}

	timeAfterPrepareFilesytem := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("prepare_filesystem").Observe(
		timeAfterPrepareFilesytem.Sub(timeAfterGetActionCommand).Seconds())

	// Invoke command.
	exitCode, err := be.runCommand(ctx, command, environment)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	timeAfterRunCommand := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("run_command").Observe(
		timeAfterRunCommand.Sub(timeAfterPrepareFilesytem).Seconds())

	response := &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			ExitCode: int32(exitCode),
		},
	}

	// Upload command output. In the common case, the files are
	// empty. If that's the case, don't bother setting the digest to
	// keep the ActionResult small.
	stdoutDigest, _, err := be.contentAddressableStorage.PutFile(ctx, be.logsDirectory, "stdout", inputRootDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	if stdoutDigest.GetSizeBytes() > 0 {
		response.Result.StdoutDigest = stdoutDigest.GetRawDigest()
	}
	stderrDigest, _, err := be.contentAddressableStorage.PutFile(ctx, be.logsDirectory, "stderr", inputRootDigest)
	if err != nil {
		return convertErrorToExecuteResponse(err), false
	}
	if stderrDigest.GetSizeBytes() > 0 {
		response.Result.StderrDigest = stderrDigest.GetRawDigest()
	}

	// Upload output files.
	for _, outputFile := range command.OutputFiles {
		digest, isExecutable, err := be.contentAddressableStorage.PutFile(ctx, outputParentDirectories[path.Dir(outputFile)], path.Base(outputFile), inputRootDigest)
		if err != nil {
			// TODO(edsch): Output file symlinks.
			if os.IsNotExist(err) {
				continue
			}
			return convertErrorToExecuteResponse(err), false
		}
		response.Result.OutputFiles = append(response.Result.OutputFiles, &remoteexecution.OutputFile{
			Path:         outputFile,
			Digest:       digest.GetRawDigest(),
			IsExecutable: isExecutable,
		})
	}

	// Upload output directories.
	for _, outputDirectory := range command.OutputDirectories {
		directory, err := outputParentDirectories[path.Dir(outputDirectory)].Enter(path.Base(outputDirectory))
		if err != nil {
			// TODO(edsch): Output directory symlinks.
			if os.IsNotExist(err) {
				continue
			}
			return convertErrorToExecuteResponse(err), false
		}
		digest, err := be.uploadTree(ctx, directory, inputRootDigest)
		directory.Close()
		if err != nil {
			return convertErrorToExecuteResponse(err), false
		}
		if digest != nil {
			response.Result.OutputDirectories = append(response.Result.OutputDirectories, &remoteexecution.OutputDirectory{
				Path:       outputDirectory,
				TreeDigest: digest.GetRawDigest(),
			})
		}
	}

	timeAfterUpload := time.Now()
	localBuildExecutorDurationSeconds.WithLabelValues("upload_output").Observe(
		timeAfterUpload.Sub(timeAfterRunCommand).Seconds())

	return response, !action.DoNotCache && response.Result.ExitCode == 0
}
