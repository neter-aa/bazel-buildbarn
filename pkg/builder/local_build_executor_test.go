package builder

import (
	"context"
	"os"
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/EdSchouten/bazel-buildbarn/pkg/mock"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mustStatus(s *status.Status, err error) *status.Status {
	if err != nil {
		panic("Failed to create status")
	}
	return s
}

func TestLocalBuildExecutorFailure(t *testing.T) {
	// Set up mocks.
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()
	contentAddressableStorage := mock.NewMockContentAddressableStorage(ctrl)
	environmentManager := mock.NewMockManager(ctrl)
	logsDirectory := mock.NewMockDirectory(ctrl)
	localBuildExecutor := NewLocalBuildExecutor(contentAddressableStorage, environmentManager, logsDirectory)

	// Missing action digest.
	executeResponse, mayBeCached := localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "debian8",
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.InvalidArgument, "Failed to extract digest for action: No digest provided").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Malformed action digest.
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "windows10",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "This is a malformed hash",
			SizeBytes: 123,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.InvalidArgument, "Failed to extract digest for action: Unknown digest hash length: 24 characters").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Action cannot be obtained from storage.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("freebsd12", &remoteexecution.Digest{
			Hash:      "64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c",
			SizeBytes: 11,
		})).Return(nil, mustStatus(status.New(codes.FailedPrecondition, "Blob not found").WithDetails(
		&errdetails.PreconditionFailure{
			Violations: []*errdetails.PreconditionFailure_Violation{
				{
					Type:    "MISSING",
					Subject: "blobs/64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c/11",
				},
			},
		})).Err())
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "freebsd12",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c",
			SizeBytes: 11,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: mustStatus(status.New(codes.FailedPrecondition, "Failed to obtain action: Blob not found").WithDetails(
			&errdetails.PreconditionFailure{
				Violations: []*errdetails.PreconditionFailure_Violation{
					{
						Type:    "MISSING",
						Subject: "blobs/64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c/11",
					},
				},
			})).Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Invalid command digest.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("macos", &remoteexecution.Digest{
			Hash:      "1234567890123456789012345678901234567890123456789012345678901234",
			SizeBytes: 42,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "0219780857348957032483209484095803948034980394803948091823092382",
			SizeBytes: -123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000000",
			SizeBytes: 42,
		},
	}, nil)
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "macos",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "1234567890123456789012345678901234567890123456789012345678901234",
			SizeBytes: 42,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.InvalidArgument, "Failed to extract digest for command: Invalid digest size: -123 bytes").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Command cannot be obtained from storage.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("macos", &remoteexecution.Digest{
			Hash:      "3333333333333333333333333333333333333333333333333333333333333333",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "4444444444444444444444444444444444444444444444444444444444444444",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000000",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("macos", &remoteexecution.Digest{
			Hash:      "4444444444444444444444444444444444444444444444444444444444444444",
			SizeBytes: 123,
		})).Return(nil, status.Error(codes.Internal, "Storage unavailable"))
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "macos",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "3333333333333333333333333333333333333333333333333333333333333333",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.Internal, "Failed to obtain command: Storage unavailable").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Environment manager rejecting the build.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		})).Return(&remoteexecution.Command{
		Arguments: []string{"touch", "foo"},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "PATH", Value: "/bin:/usr/bin"},
		},
		OutputFiles: []string{"foo"},
	}, nil)
	environmentManager.EXPECT().Acquire(nil).Return(nil, status.Error(codes.InvalidArgument, "Platform requirements not provided"))
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "netbsd",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.InvalidArgument, "Failed to acquire build environment: Platform requirements not provided").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Missing digest in input directory.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		})).Return(&remoteexecution.Command{
		Arguments: []string{"touch", "foo"},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "PATH", Value: "/bin:/usr/bin"},
		},
		OutputFiles: []string{"foo"},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		})).Return(&remoteexecution.Directory{
		Directories: []*remoteexecution.DirectoryNode{
			{
				Name: "Hello",
				Digest: &remoteexecution.Digest{
					Hash:      "8888888888888888888888888888888888888888888888888888888888888888",
					SizeBytes: 123,
				},
			},
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "8888888888888888888888888888888888888888888888888888888888888888",
			SizeBytes: 123,
		})).Return(&remoteexecution.Directory{
		Directories: []*remoteexecution.DirectoryNode{
			{
				Name: "World",
			},
		},
	}, nil)
	environment := mock.NewMockEnvironment(ctrl)
	environmentManager.EXPECT().Acquire(nil).Return(environment, nil)
	buildDirectory := mock.NewMockDirectory(ctrl)
	buildDirectory.EXPECT().Mkdir("Hello", os.FileMode(0777)).Return(nil)
	helloDirectory := mock.NewMockDirectory(ctrl)
	buildDirectory.EXPECT().Enter("Hello").Return(helloDirectory, nil)
	helloDirectory.EXPECT().Close()
	helloDirectory.EXPECT().Mkdir("World", os.FileMode(0777)).Return(nil)
	worldDirectory := mock.NewMockDirectory(ctrl)
	helloDirectory.EXPECT().Enter("World").Return(worldDirectory, nil)
	worldDirectory.EXPECT().Close()
	environment.EXPECT().GetBuildDirectory().Return(buildDirectory)
	environment.EXPECT().Release()
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "netbsd",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.InvalidArgument, "Failed to extract digest for input directory \"Hello/World\": No digest provided").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Unfetchable root input directory.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		})).Return(&remoteexecution.Command{
		Arguments: []string{"touch", "foo"},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "PATH", Value: "/bin:/usr/bin"},
		},
		OutputFiles: []string{"foo"},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("netbsd", &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		})).Return(nil, status.Error(codes.Internal, "Storage is offline"))
	environmentManager.EXPECT().Acquire(nil).Return(environment, nil)
	environment.EXPECT().GetBuildDirectory().Return(buildDirectory)
	environment.EXPECT().Release()
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "netbsd",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.Internal, "Failed to obtain input directory \".\": Storage is offline").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Failure to create an output directory.
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("fedora", &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("fedora", &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		})).Return(&remoteexecution.Command{
		Arguments: []string{"touch", "foo"},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "PATH", Value: "/bin:/usr/bin"},
		},
		OutputFiles: []string{"foo/bar/baz"},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("fedora", &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		})).Return(&remoteexecution.Directory{}, nil)
	environmentManager.EXPECT().Acquire(nil).Return(environment, nil)
	buildDirectory.EXPECT().Mkdir("foo", os.FileMode(0777)).Return(status.Error(codes.Internal, "Out of disk space"))
	environment.EXPECT().GetBuildDirectory().Return(buildDirectory)
	environment.EXPECT().Release()
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "fedora",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.Internal, "Failed to create output directory \"foo\": Out of disk space").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)

	// Failure to read a symlink in an output directory.
	stdout := mock.NewMockFile(ctrl)
	logsDirectory.EXPECT().OpenFile("stdout", os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0)).Return(stdout, nil)
	stdout.EXPECT().Close()
	stderr := mock.NewMockFile(ctrl)
	logsDirectory.EXPECT().OpenFile("stderr", os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0)).Return(stderr, nil)
	stderr.EXPECT().Close()
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("nintendo64", &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("nintendo64", &remoteexecution.Digest{
			Hash:      "6666666666666666666666666666666666666666666666666666666666666666",
			SizeBytes: 123,
		})).Return(&remoteexecution.Command{
		Arguments: []string{"touch", "foo"},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "PATH", Value: "/bin:/usr/bin"},
		},
		OutputDirectories: []string{"foo"},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("nintendo64", &remoteexecution.Digest{
			Hash:      "7777777777777777777777777777777777777777777777777777777777777777",
			SizeBytes: 42,
		})).Return(&remoteexecution.Directory{}, nil)
	contentAddressableStorage.EXPECT().PutFile(ctx, logsDirectory, "stdout", gomock.Any()).Return(
		util.MustNewDigest("nintendo64", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000005",
			SizeBytes: 567,
		}), nil)
	contentAddressableStorage.EXPECT().PutFile(ctx, logsDirectory, "stderr", gomock.Any()).Return(
		util.MustNewDigest("nintendo64", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000006",
			SizeBytes: 678,
		}), nil)
	environmentManager.EXPECT().Acquire(nil).Return(environment, nil)
	environment.EXPECT().GetBuildDirectory().Return(buildDirectory)
	environment.EXPECT().Run(ctx, []string{"touch", "foo"}, map[string]string{"PATH": "/bin:/usr/bin"}, "", stdout, stderr).Return(0, nil)
	environment.EXPECT().Release()
	buildDirectory.EXPECT().Lstat("foo").Return(filesystem.NewSimpleFileInfo("foo", 0777|os.ModeDir), nil)
	fooDirectory := mock.NewMockDirectory(ctrl)
	buildDirectory.EXPECT().Enter("foo").Return(fooDirectory, nil)
	fooDirectory.EXPECT().ReadDir().Return([]filesystem.FileInfo{
		filesystem.NewSimpleFileInfo("bar", 0777|os.ModeSymlink),
	}, nil)
	fooDirectory.EXPECT().Readlink("bar").Return("", status.Error(codes.Internal, "Cosmic rays caused interference"))
	fooDirectory.EXPECT().Close()
	executeResponse, mayBeCached = localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "nintendo64",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "5555555555555555555555555555555555555555555555555555555555555555",
			SizeBytes: 7,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Status: status.New(codes.Internal, "Failed to read output symlink \"foo/bar\": Cosmic rays caused interference").Proto(),
	}, executeResponse)
	require.False(t, mayBeCached)
}

// TestLocalBuildExecutorSuccess tests a full invocation of a simple
// build step, equivalent to compiling a simple C++ file.
func TestLocalBuildExecutorSuccess(t *testing.T) {
	ctrl, ctx := gomock.WithContext(context.Background(), t)
	defer ctrl.Finish()

	// File system operations that should occur against the build directory.
	// Creation of bazel-out/k8-fastbuild/bin/_objs/hello.
	buildDirectory := mock.NewMockDirectory(ctrl)
	buildDirectory.EXPECT().Mkdir("bazel-out", os.FileMode(0777)).Return(nil)
	bazelOutDirectory := mock.NewMockDirectory(ctrl)
	buildDirectory.EXPECT().Enter("bazel-out").Return(bazelOutDirectory, nil)
	bazelOutDirectory.EXPECT().Close()
	bazelOutDirectory.EXPECT().Mkdir("k8-fastbuild", os.FileMode(0777)).Return(nil)
	k8FastbuildDirectory := mock.NewMockDirectory(ctrl)
	bazelOutDirectory.EXPECT().Enter("k8-fastbuild").Return(k8FastbuildDirectory, nil)
	k8FastbuildDirectory.EXPECT().Close()
	k8FastbuildDirectory.EXPECT().Mkdir("bin", os.FileMode(0777)).Return(nil)
	binDirectory := mock.NewMockDirectory(ctrl)
	k8FastbuildDirectory.EXPECT().Enter("bin").Return(binDirectory, nil)
	binDirectory.EXPECT().Close()
	binDirectory.EXPECT().Mkdir("_objs", os.FileMode(0777)).Return(nil)
	objsDirectory := mock.NewMockDirectory(ctrl)
	binDirectory.EXPECT().Enter("_objs").Return(objsDirectory, nil)
	objsDirectory.EXPECT().Close()
	objsDirectory.EXPECT().Mkdir("hello", os.FileMode(0777)).Return(nil)
	helloDirectory := mock.NewMockDirectory(ctrl)
	objsDirectory.EXPECT().Enter("hello").Return(helloDirectory, nil)
	helloDirectory.EXPECT().Close()
	helloDirectory.EXPECT().Lstat("hello.pic.d").Return(filesystem.NewSimpleFileInfo("hello.pic.d", 0666), nil)
	helloDirectory.EXPECT().Lstat("hello.pic.o").Return(filesystem.NewSimpleFileInfo("hello.pic.o", 0777), nil)

	// Creation of log files.
	logsDirectory := mock.NewMockDirectory(ctrl)
	stdout := mock.NewMockFile(ctrl)
	logsDirectory.EXPECT().OpenFile("stdout", os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0)).Return(stdout, nil)
	stdout.EXPECT().Close()
	stderr := mock.NewMockFile(ctrl)
	logsDirectory.EXPECT().OpenFile("stderr", os.O_APPEND|os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(0)).Return(stderr, nil)
	stderr.EXPECT().Close()

	// Read operations against the Content Addressable Storage.
	contentAddressableStorage := mock.NewMockContentAddressableStorage(ctrl)
	contentAddressableStorage.EXPECT().GetAction(
		ctx, util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000001",
			SizeBytes: 123,
		})).Return(&remoteexecution.Action{
		CommandDigest: &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000002",
			SizeBytes: 234,
		},
		InputRootDigest: &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000003",
			SizeBytes: 345,
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetCommand(
		ctx, util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000002",
			SizeBytes: 234,
		})).Return(&remoteexecution.Command{
		Arguments: []string{
			"/usr/local/bin/clang",
			"-MD",
			"-MF",
			"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.d",
			"-c",
			"hello.cc",
			"-o",
			"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.o",
		},
		EnvironmentVariables: []*remoteexecution.Command_EnvironmentVariable{
			{Name: "BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN", Value: "1"},
			{Name: "PATH", Value: "/bin:/usr/bin"},
			{Name: "PWD", Value: "/proc/self/cwd"},
		},
		OutputFiles: []string{
			"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.d",
			"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.o",
		},
		Platform: &remoteexecution.Platform{
			Properties: []*remoteexecution.Platform_Property{
				{
					Name:  "container-image",
					Value: "docker://gcr.io/cloud-marketplace/google/rbe-debian8@sha256:4893599fb00089edc8351d9c26b31d3f600774cb5addefb00c70fdb6ca797abf",
				},
			},
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetDirectory(
		ctx, util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000003",
			SizeBytes: 345,
		})).Return(&remoteexecution.Directory{
		Files: []*remoteexecution.FileNode{
			{
				Name: "hello.cc",
				Digest: &remoteexecution.Digest{
					Hash:      "0000000000000000000000000000000000000000000000000000000000000004",
					SizeBytes: 456,
				},
			},
		},
	}, nil)
	contentAddressableStorage.EXPECT().GetFile(
		ctx, util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000004",
			SizeBytes: 456,
		}), buildDirectory, "hello.cc", false).Return(nil)

	// Write operations against the Content Addressable Storage.
	contentAddressableStorage.EXPECT().PutFile(ctx, logsDirectory, "stdout", gomock.Any()).Return(
		util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000005",
			SizeBytes: 567,
		}), nil)
	contentAddressableStorage.EXPECT().PutFile(ctx, logsDirectory, "stderr", gomock.Any()).Return(
		util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000006",
			SizeBytes: 678,
		}), nil)
	contentAddressableStorage.EXPECT().PutFile(ctx, helloDirectory, "hello.pic.d", gomock.Any()).Return(
		util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000007",
			SizeBytes: 789,
		}), nil)
	contentAddressableStorage.EXPECT().PutFile(ctx, helloDirectory, "hello.pic.o", gomock.Any()).Return(
		util.MustNewDigest("ubuntu1804", &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000008",
			SizeBytes: 890,
		}), nil)

	// Command execution.
	environmentManager := mock.NewMockManager(ctrl)
	environment := mock.NewMockEnvironment(ctrl)
	environmentManager.EXPECT().Acquire(&remoteexecution.Platform{
		Properties: []*remoteexecution.Platform_Property{
			{
				Name:  "container-image",
				Value: "docker://gcr.io/cloud-marketplace/google/rbe-debian8@sha256:4893599fb00089edc8351d9c26b31d3f600774cb5addefb00c70fdb6ca797abf",
			},
		},
	}).Return(environment, nil)
	environment.EXPECT().GetBuildDirectory().Return(buildDirectory)
	environment.EXPECT().Run(ctx, []string{
		"/usr/local/bin/clang",
		"-MD",
		"-MF",
		"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.d",
		"-c",
		"hello.cc",
		"-o",
		"bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.o",
	}, map[string]string{
		"BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN": "1",
		"PATH": "/bin:/usr/bin",
		"PWD":  "/proc/self/cwd",
	}, "", stdout, stderr).Return(0, nil)
	environment.EXPECT().Release()

	localBuildExecutor := NewLocalBuildExecutor(contentAddressableStorage, environmentManager, logsDirectory)
	executeResponse, mayBeCached := localBuildExecutor.Execute(ctx, &remoteexecution.ExecuteRequest{
		InstanceName: "ubuntu1804",
		ActionDigest: &remoteexecution.Digest{
			Hash:      "0000000000000000000000000000000000000000000000000000000000000001",
			SizeBytes: 123,
		},
	})
	require.Equal(t, &remoteexecution.ExecuteResponse{
		Result: &remoteexecution.ActionResult{
			OutputFiles: []*remoteexecution.OutputFile{
				{
					Path: "bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.d",
					Digest: &remoteexecution.Digest{
						Hash:      "0000000000000000000000000000000000000000000000000000000000000007",
						SizeBytes: 789,
					},
				},
				{
					Path: "bazel-out/k8-fastbuild/bin/_objs/hello/hello.pic.o",
					Digest: &remoteexecution.Digest{
						Hash:      "0000000000000000000000000000000000000000000000000000000000000008",
						SizeBytes: 890,
					},
					IsExecutable: true,
				},
			},
			StdoutDigest: &remoteexecution.Digest{
				Hash:      "0000000000000000000000000000000000000000000000000000000000000005",
				SizeBytes: 567,
			},
			StderrDigest: &remoteexecution.Digest{
				Hash:      "0000000000000000000000000000000000000000000000000000000000000006",
				SizeBytes: 678,
			},
		},
	}, executeResponse)
	require.True(t, mayBeCached)
}

// TODO(edsch): Test aspects of execution not covered above (e.g., output directories, symlinks).
