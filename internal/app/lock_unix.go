//go:build !windows

package app

import (
	"os"
	"syscall"
)

// tryLockOS attempts a non-blocking exclusive flock on the lock file.
// Returns locked=false with a nil error when another process holds the
// lock. The kernel releases the lock automatically when the owning process
// exits, even after a crash.
func tryLockOS(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if err == syscall.EWOULDBLOCK || err == syscall.EAGAIN {
		return false, nil
	}
	return false, err
}

func unlockOS(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
