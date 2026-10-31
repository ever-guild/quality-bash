package quality

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// Open every directory component without following links, retaining the final
// inode so later renames of path components cannot redirect an output write.
func openOutputDirectory(path string) (*os.File, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	flags := syscall.O_RDONLY | syscall.O_DIRECTORY | syscall.O_NOFOLLOW | syscall.O_CLOEXEC
	fd, err := syscall.Open("/", flags, 0)
	if err != nil {
		return nil, err
	}
	for _, component := range strings.Split(strings.TrimPrefix(absolute, "/"), "/") {
		if component == "" {
			continue
		}
		next, err := syscall.Openat(fd, component, flags, 0)
		syscall.Close(fd)
		if err != nil {
			return nil, fmt.Errorf("open output directory without symlinks: %w", err)
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), absolute), nil
}

func openOutputFile(path string, flags int, mode uint32) (*os.File, error) {
	directory, err := openOutputDirectory(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	defer directory.Close()
	fd, err := syscall.Openat(int(directory.Fd()), filepath.Base(path), flags|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, mode)
	if err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}
