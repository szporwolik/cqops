//go:build !windows

package app

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// processExists reports whether a process with the given PID is alive.
// Signal 0 is the null signal — it performs error checking but sends
// nothing. ESRCH means the process does not exist; EPERM means it exists
// but cannot be signaled (still counts as alive).
func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// processIsCQOps reports whether the process with the given PID is a CQOps
// instance (the kernel's task comm name, e.g. /proc/<pid>/comm). Used to
// distinguish a real running instance from a PID reused by an unrelated
// process after a crash. When the answer cannot be determined, it returns
// true — refusing to start is the safe default.
func processIsCQOps(pid int) bool {
	comm, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/comm")
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(comm)) == "cqops"
}

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
