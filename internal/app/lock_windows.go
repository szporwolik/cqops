//go:build windows

package app

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// processExists reports whether a process with the given PID is alive.
// On Windows FindProcess fails only for invalid PIDs; the handle is
// released immediately — existence is all we need.
func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	defer proc.Release()
	return true
}

// processIsCQOps conservatively reports true on Windows — the process-name
// lookup is not implemented, so any live PID counts as a real instance and
// startup fails fast instead of prompting.
func processIsCQOps(pid int) bool {
	return true
}

// tryLockOS attempts a non-blocking exclusive byte-range lock on the lock
// file. Returns locked=false with a nil error when another process holds
// the lock. Windows releases file locks when the owning handle closes, so
// a crashed instance cannot keep the lock.
func tryLockOS(f *os.File) (bool, error) {
	var ol windows.Overlapped
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, &ol)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return false, err
}

func unlockOS(f *os.File) error {
	var ol windows.Overlapped
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, &ol)
}
