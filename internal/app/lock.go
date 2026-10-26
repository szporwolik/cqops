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
		f.Close()
		// The OS lock is held, so a live instance owns this configuration
		// directory: refuse to start. The lock file is never deleted here —
		// unlinking it would leave the holder locked on an orphaned inode
		// while this process locks a freshly created one, and both instances
		// would then share the same logbook and database.
		//
		// The PID is read for the message only. It must not decide ownership:
		// it can be stale, reused by an unrelated process, or belong to a
		// binary named anything at all (cqops-linux-amd64, a renamed copy, an
		// AppImage), and a name mismatch would otherwise be treated as proof
		// that the lock is orphaned. A leftover file from a crashed instance
		// needs no special handling either: the kernel released its lock, so
		// the next tryLockOS on that same file succeeds and reuses it.
		if owner := readLockOwner(path); owner != "" {
			return nil, fmt.Errorf("another CQOps instance is already running (PID %s) — close it before starting", owner)
		}
		return nil, fmt.Errorf("another CQOps instance is already running — close it before starting")
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

// readLockOwner returns the PID recorded in the lock file, or "" when it is
// missing or not a plain number. Diagnostics only — a stale or reused PID can
// never decide ownership.
func readLockOwner(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	owner := strings.TrimSpace(string(data))
	if _, err := strconv.Atoi(owner); err != nil {
		return ""
	}
	return owner
}

// PromptYN asks the user a yes/no question on the terminal. Used by the CLI
// reset commands; the single-instance lock never prompts — an OS lock that is
// held always means a live instance, and a leftover file is reused as-is.
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
