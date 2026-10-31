package quality

import (
	"fmt"
	"os"
	"syscall"
)

// Keep the lock file after releasing it: unlinking a locked inode could allow
// another process to acquire a different inode for the same baseline.
func lockBaseline(path string) (func(), error) {
	if err := rejectSymlinkComponents(path + ".lock"); err != nil {
		return nil, err
	}
	file, err := openOutputFile(path+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		file.Close()
		return nil, fmt.Errorf("baseline lock must be a regular file")
	}
	if err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("baseline update already in progress: %w", err)
	}
	return func() { _ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN); _ = file.Close() }, nil
}
