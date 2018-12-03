package filesystem_test

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/EdSchouten/bazel-buildbarn/pkg/filesystem"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func openTmpDir(t *testing.T) filesystem.Directory {
	p := filepath.Join(os.Getenv("TEST_TMPDIR"), t.Name())
	require.NoError(t, os.Mkdir(p, 0777))
	d, err := filesystem.NewLocalDirectory(p)
	require.NoError(t, err)
	return d
}

func TestLocalDirectoryCreationFailure(t *testing.T) {
	_, err := filesystem.NewLocalDirectory("/nonexistent")
	require.True(t, os.IsNotExist(err))
}

func TestLocalDirectoryCreationSuccess(t *testing.T) {
	d := openTmpDir(t)
	require.NoError(t, d.Close())
}

func TestLocalDirectoryEnterBadName(t *testing.T) {
	d := openTmpDir(t)

	// Empty filename.
	_, err := d.Enter("")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"\""), err)
	// Attempt to bypass directory hierarchy.
	_, err = d.Enter(".")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \".\""), err)
	_, err = d.Enter("..")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"..\""), err)
	// Skipping of intermediate directory levels.
	_, err = d.Enter("foo/bar")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"foo/bar\""), err)

	require.NoError(t, d.Close())
}

func TestLocalDirectoryEnterNonExistent(t *testing.T) {
	d := openTmpDir(t)
	_, err := d.Enter("nonexistent")
	require.True(t, os.IsNotExist(err))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryEnterFile(t *testing.T) {
	d := openTmpDir(t)
	f, err := d.OpenFile("file", os.O_CREATE|os.O_WRONLY, 0666)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	_, err = d.Enter("file")
	require.Equal(t, syscall.ENOTDIR, err)
	require.NoError(t, d.Close())
}

func TestLocalDirectoryEnterSymlink(t *testing.T) {
	d := openTmpDir(t)
	require.NoError(t, d.Symlink("/", "symlink"))
	_, err := d.Enter("symlink")
	require.Equal(t, syscall.ENOTDIR, err)
	require.NoError(t, d.Close())
}

func TestLocalDirectoryEnterSuccess(t *testing.T) {
	d := openTmpDir(t)
	require.NoError(t, d.Mkdir("subdir", 0777))
	sub, err := d.Enter("subdir")
	require.NoError(t, err)
	require.NoError(t, sub.Close())
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLinkBadName(t *testing.T) {
	d := openTmpDir(t)

	// Invalid source name.
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"\""), d.Link("", d, "file"))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \".\""), d.Link(".", d, "file"))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"..\""), d.Link("..", d, "file"))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"foo/bar\""), d.Link("foo/bar", d, "file"))

	// Invalid target name.
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"\""), d.Link("file", d, ""))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \".\""), d.Link("file", d, "."))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"..\""), d.Link("file", d, ".."))
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"foo/bar\""), d.Link("file", d, "foo/bar"))

	require.NoError(t, d.Close())
}

func TestLocalDirectoryLinkNotFound(t *testing.T) {
	d := openTmpDir(t)
	require.Equal(t, syscall.ENOENT, d.Link("source", d, "target"))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLinkDirectory(t *testing.T) {
	d := openTmpDir(t)
	require.NoError(t, d.Mkdir("source", 0777))
	require.True(t, os.IsPermission(d.Link("source", d, "target")))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLinkTargetExists(t *testing.T) {
	d := openTmpDir(t)
	f, err := d.OpenFile("source", os.O_CREATE|os.O_WRONLY, 0666)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = d.OpenFile("target", os.O_CREATE|os.O_WRONLY, 0666)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.True(t, os.IsExist(d.Link("source", d, "target")))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLinkSuccess(t *testing.T) {
	d := openTmpDir(t)
	f, err := d.OpenFile("source", os.O_CREATE|os.O_WRONLY, 0666)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, d.Link("source", d, "target"))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLstatBadName(t *testing.T) {
	d := openTmpDir(t)
	_, err := d.Lstat("")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"\""), err)
	_, err = d.Lstat(".")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \".\""), err)
	_, err = d.Lstat("..")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"..\""), err)
	_, err = d.Lstat("foo/bar")
	require.Equal(t, status.Error(codes.InvalidArgument, "Invalid filename: \"foo/bar\""), err)
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLstatNonExistent(t *testing.T) {
	d := openTmpDir(t)
	_, err := d.Lstat("hello")
	require.True(t, os.IsNotExist(err))
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLstatFile(t *testing.T) {
	syscall.Umask(0)
	d := openTmpDir(t)
	f, err := d.OpenFile("file", os.O_CREATE|os.O_WRONLY, 0644)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	fi, err := d.Lstat("file")
	require.NoError(t, err)
	require.Equal(t, "file", fi.Name())
	require.Equal(t, os.FileMode(0644), fi.Mode())
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLstatSymlink(t *testing.T) {
	syscall.Umask(0)
	d := openTmpDir(t)
	require.NoError(t, d.Symlink("/", "symlink"))
	fi, err := d.Lstat("symlink")
	require.NoError(t, err)
	require.Equal(t, "symlink", fi.Name())
	require.Equal(t, 0777|os.ModeSymlink, fi.Mode())
	require.NoError(t, d.Close())
}

func TestLocalDirectoryLstatDirectory(t *testing.T) {
	syscall.Umask(0)
	d := openTmpDir(t)
	require.NoError(t, d.Mkdir("directory", 0700))
	fi, err := d.Lstat("directory")
	require.NoError(t, err)
	require.Equal(t, "directory", fi.Name())
	require.Equal(t, 0700|os.ModeDir, fi.Mode())
	require.NoError(t, d.Close())
}

// TODO(edsch): Make testing coverage more complete?
