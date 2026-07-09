//go:build windows

package app

import (
	"os"
)

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Windows, FindProcess returns an error only if the PID is
	// invalid. Release the handle immediately — we just need existence.
	defer proc.Release()
	return true
}
