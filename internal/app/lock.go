package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// lockFile is a simple single-instance guard. It writes the current PID into
// <dir>/cqops.lock. On startup, if a live process owns the lock, Init returns
// an error. If the lock is stale (crashed process), the user is prompted to
// delete it. The lock is cleaned up in Close.
type lockFile struct {
	path string
}

func acquireLock(dir string) (*lockFile, error) {
	path := filepath.Join(dir, "cqops.lock")

	data, err := os.ReadFile(path)
	if err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pidStr != "" {
			oldPID, convErr := strconv.Atoi(pidStr)
			if convErr == nil && oldPID != os.Getpid() {
				if processExists(oldPID) {
					return nil, fmt.Errorf("another CQOps instance is already running (PID %d)", oldPID)
				}
				// Stale lock from a crashed process — ask the user.
				if !PromptYN(fmt.Sprintf("Stale lock from PID %d found. Delete it?", oldPID)) {
					return nil, fmt.Errorf("lock file exists (%s) — remove it manually or restart", path)
				}
				os.Remove(path)
			}
		}
	}

	pidData := []byte(strconv.Itoa(os.Getpid()) + "\n")
	if err := os.WriteFile(path, pidData, 0644); err != nil {
		return nil, fmt.Errorf("cannot write lock file: %w", err)
	}

	return &lockFile{path: path}, nil
}

func PromptYN(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

func (l *lockFile) release() {
	if l == nil || l.path == "" {
		return
	}
	data, err := os.ReadFile(l.path)
	if err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pidStr == strconv.Itoa(os.Getpid()) {
			os.Remove(l.path)
		}
	}
}
