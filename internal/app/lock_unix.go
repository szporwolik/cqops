//go:build !windows

package app

import (
	"syscall"
)

func processExists(pid int) bool {
	// Signal 0 is the null signal — it performs error checking but
	// doesn't actually send a signal. ESRCH means the process doesn't
	// exist; EPERM means it exists but we can't signal it (which still
	// counts as "exists" for our purposes).
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
