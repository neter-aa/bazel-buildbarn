package filesystem

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type localDirectory struct {
	fd int
}

func validateFilename(name string) error {
	if name == "" || name == ".." || strings.ContainsRune(name, '/') {
		return fmt.Errorf("Invalid filename: %s", name)
	}
	return nil
}

// NewLocalDirectory creates a directory handle that corresponds to a
// local path on the system.
func NewLocalDirectory(path string) (Directory, error) {
	fd, err := unix.Openat(unix.AT_FDCWD, path, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	d := &localDirectory{
		fd: fd,
	}
	runtime.SetFinalizer(d, (*localDirectory).Close)
	return d, nil
}

func (d *localDirectory) Enter(name string) (Directory, error) {
	if err := validateFilename(name); err != nil {
		return nil, err
	}
	fd, err := unix.Openat(d.fd, name, unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	cd := &localDirectory{
		fd: fd,
	}
	runtime.SetFinalizer(cd, (*localDirectory).Close)
	return cd, nil
}

func (d *localDirectory) Close() error {
	fd := d.fd
	d.fd = -1
	runtime.SetFinalizer(d, nil)
	return unix.Close(fd)
}

func (d *localDirectory) Link(oldName string, newDirectory Directory, newName string) error {
	if err := validateFilename(oldName); err != nil {
		return err
	}
	if err := validateFilename(newName); err != nil {
		return err
	}
	d2, ok := newDirectory.(*localDirectory)
	if !ok {
		return errors.New("Source and target directory have different types")
	}
	err := unix.Linkat(d.fd, oldName, d2.fd, newName, 0)
	runtime.KeepAlive(d)
	runtime.KeepAlive(d2)
	return err
}

func (d *localDirectory) Mkdir(name string, perm os.FileMode) error {
	if err := validateFilename(name); err != nil {
		return err
	}
	err := unix.Mkdirat(d.fd, name, uint32(perm))
	runtime.KeepAlive(d)
	return err
}

func (d *localDirectory) OpenFile(name string, flag int, perm os.FileMode) (*os.File, error) {
	if err := validateFilename(name); err != nil {
		return nil, err
	}
	fd, err := unix.Openat(d.fd, name, flag|unix.O_NOFOLLOW, uint32(perm))
	runtime.KeepAlive(d)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), name), nil
}

// fileInfo implements the os.FileInfo interface on top of unix.Stat_t.
type fileInfo struct {
	name string
	stat unix.Stat_t
}

func (fi *fileInfo) Name() string {
	return fi.name
}

func (fi *fileInfo) Size() int64 {
	return fi.stat.Size
}

func (fi *fileInfo) Mode() os.FileMode {
	mode := os.FileMode(fi.stat.Mode & 0777)
	switch fi.stat.Mode & syscall.S_IFMT {
	case syscall.S_IFBLK:
		mode |= os.ModeDevice
	case syscall.S_IFCHR:
		mode |= os.ModeDevice | os.ModeCharDevice
	case syscall.S_IFDIR:
		mode |= os.ModeDir
	case syscall.S_IFIFO:
		mode |= os.ModeNamedPipe
	case syscall.S_IFLNK:
		mode |= os.ModeSymlink
	case syscall.S_IFREG:
		// Regular files have a mode of zero.
	case syscall.S_IFSOCK:
		mode |= os.ModeSocket
	}
	if fi.stat.Mode&syscall.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	if fi.stat.Mode&syscall.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if fi.stat.Mode&syscall.S_ISVTX != 0 {
		mode |= os.ModeSticky
	}
	return mode
}

func (fi *fileInfo) ModTime() time.Time {
	return time.Unix(int64(fi.stat.Mtim.Sec), int64(fi.stat.Mtim.Nsec))
}

func (fi *fileInfo) IsDir() bool {
	return fi.stat.Mode&syscall.S_IFMT == syscall.S_IFDIR
}

func (fi *fileInfo) Sys() interface{} {
	return &fi.stat
}

func (d *localDirectory) ReadDir() ([]os.FileInfo, error) {
	// Obtain filenames in current directory.
	fd, err := unix.Openat(d.fd, ".", unix.O_DIRECTORY|unix.O_RDONLY, 0)
	runtime.KeepAlive(d)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), ".")
	names, err := f.Readdirnames(-1)
	f.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	// Obtain file info.
	var list []os.FileInfo
	for _, name := range names {
		stat := &fileInfo{
			name: name,
		}
		if err := unix.Fstatat(d.fd, name, &stat.stat, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		list = append(list, stat)
	}
	runtime.KeepAlive(d)
	return list, nil
}

func (d *localDirectory) Readlink(name string) (string, error) {
	if err := validateFilename(name); err != nil {
		return "", err
	}
	for l := 128; ; l *= 2 {
		b := make([]byte, l)
		n, err := unix.Readlinkat(d.fd, name, b)
		runtime.KeepAlive(d)
		if err != nil {
			return "", err
		}
		if n < l {
			return string(b[0:n]), nil
		}
	}
}

func (d *localDirectory) Remove(name string) error {
	if err := validateFilename(name); err != nil {
		return err
	}
	// First try deleting it as a regular file.
	err1 := unix.Unlinkat(d.fd, name, 0)
	if err1 == nil {
		return nil
	}
	// Then try to delete it as a directory.
	err2 := unix.Unlinkat(d.fd, name, unix.AT_REMOVEDIR)
	runtime.KeepAlive(d)
	if err2 == nil {
		return nil
	}
	// Determine which error to return.
	if err1 != syscall.ENOTDIR {
		return err1
	}
	return err2
}

func (d *localDirectory) Symlink(oldName string, newName string) error {
	if err := validateFilename(newName); err != nil {
		return err
	}
	err := unix.Symlinkat(oldName, d.fd, newName)
	runtime.KeepAlive(d)
	return err
}
