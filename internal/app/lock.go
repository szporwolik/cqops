package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// lockFile is the single-instance guard. It holds an OS-backed exclusive
// lock on <dir>/cqops.lock — flock on Unix, LockFileEx on Windows. The
// kernel releases the lock automatically when the process exits, even
// after a crash, so the PID written into the file is informational only.
// Ownership is decided by the OS lock, which serializes racing instances
// at startup and survives for the whole process lifetime. The lock is
// released explicitly on initialization failures.
type lockFile struct {
	path string
	f    *os.File
}

func acquireLock(dir string) (*lockFile, error) {
	path := filepath.Join(dir, "cqops.lock")

	// Open without O_EXCL: the file may already exist from a previous
	// instance. Ownership is decided by the OS lock below — never by file
	// creation or a read-then-write PID check, which racing instances
	// could both pass.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("cannot open lock file: %w", err)
	}

	locked, err := tryLockOS(f)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("cannot lock lock file: %w", err)
	}
	if !locked {
		// Another live instance holds the OS lock. The file's PID content
		// is only for the error message.
		owner := ""
		if data, rerr := os.ReadFile(path); rerr == nil {
			owner = strings.TrimSpace(string(data))
		}
		f.Close()
		if owner != "" {
			return nil, fmt.Errorf("another CQOps instance is already running (PID %s)", owner)
		}
		return nil, fmt.Errorf("another CQOps instance is already running")
	}

	// We own the lock — record our PID for diagnostics. A stale PID left
	// behind by a crashed instance is harmless: the OS lock, not the
	// content, decides ownership.
	if err := f.Truncate(0); err == nil {
		if _, werr := f.WriteString(strconv.Itoa(os.Getpid()) + "\n"); werr != nil {
			unlockOS(f)
			f.Close()
			return nil, fmt.Errorf("cannot write lock file: %w", werr)
		}
	}
	return &lockFile{path: path, f: f}, nil
}

func (l *lockFile) release() {
	if l == nil {
		return
	}
	if l.f != nil {
		unlockOS(l.f)
		l.f.Close()
		l.f = nil
	}
	// The lock file itself is intentionally left in place: deleting a
	// locked file races a third process that may have already opened the
	// same inode, which could let two instances each hold a lock on a
	// different file.
}

// PromptYN asks the user a yes/no question on the terminal. Used by the
// CLI reset commands — the single-instance lock itself no longer prompts:
// the OS releases it automatically when the owner exits.
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
