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

// lockPrompt is the test seam for the stale-lock confirmation prompt.
var lockPrompt = PromptYN

// lockOwnerIsCQOps is the test seam for the live-owner identity check.
var lockOwnerIsCQOps = processIsCQOps

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
		f.Close()
		// The OS lock is held — normally by a live instance: that fails fast
		// (the operator can just close it). When the PID recorded in the file
		// no longer exists, or was reused by an unrelated process (checked
		// via the kernel's process name), the lock looks orphaned and the
		// user is asked before removing it, like the pre-refactor guard did.
		owner := ""
		if data, rerr := os.ReadFile(path); rerr == nil {
			owner = strings.TrimSpace(string(data))
		}
		ownerPID, _ := strconv.Atoi(owner)
		if ownerPID > 0 && processExists(ownerPID) && lockOwnerIsCQOps(ownerPID) {
			return nil, fmt.Errorf("another CQOps instance is already running (PID %s) — close it before starting", owner)
		}
		stale := owner
		if stale == "" {
			stale = "unknown"
		}
		if !lockPrompt(fmt.Sprintf("Stale lock from PID %s found. Delete it?", stale)) {
			return nil, fmt.Errorf("lock file exists (%s) — remove it manually or restart", path)
		}
		if rerr := os.Remove(path); rerr != nil {
			return nil, fmt.Errorf("cannot remove stale lock file: %w", rerr)
		}
		// Retry once on a fresh inode. If the lock was actually held by a
		// live process (PID reuse), the retry still fails and we refuse to
		// start instead of running two instances.
		f2, oerr := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if oerr != nil {
			return nil, fmt.Errorf("cannot reopen lock file: %w", oerr)
		}
		locked2, lerr := tryLockOS(f2)
		if lerr != nil || !locked2 {
			f2.Close()
			if lerr != nil {
				return nil, fmt.Errorf("cannot lock lock file: %w", lerr)
			}
			return nil, fmt.Errorf("another CQOps instance is already running")
		}
		f = f2
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
// CLI reset commands and by the single-instance lock when it finds a stale
// (orphaned) lock file: the OS normally releases the lock automatically
// when the owner exits, but a stale PID means the file can be removed.
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
