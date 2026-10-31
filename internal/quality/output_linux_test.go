package quality

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestOutputDirectoryRetainsOriginalInode(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	path := filepath.Join(root, "reports")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	dir, err := openOutputDirectory(path)
	if err != nil {
		t.Fatal(err)
	}
	defer dir.Close()
	moved := filepath.Join(root, "moved")
	if err := os.Rename(path, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	fd, err := syscall.Openat(int(dir.Fd()), "proof", syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	syscall.Close(fd)
	if _, err := os.Stat(filepath.Join(moved, "proof")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(outside, "proof")); !os.IsNotExist(err) {
		t.Fatal("write redirected outside")
	}
	if f, err := openOutputFile(filepath.Join(path, "other"), os.O_CREATE|os.O_RDWR, 0600); err == nil {
		f.Close()
		t.Fatal("replacement symlink followed")
	}
}
