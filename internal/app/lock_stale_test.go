//go:build !windows

package app

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// holdLockFile takes an OS flock on a fresh lock file containing the given
// PID, simulating an instance that holds the lock. The PID is deliberately
// independent of the holder: it may be dead, live, or reused.
func holdLockFile(t *testing.T, dir, pid string) *os.File {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(dir, "cqops.lock"), os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		t.Fatalf("flock: %v", err)
	}
	if pid != "" {
		if _, err := f.WriteString(pid + "\n"); err != nil {
			t.Fatalf("write pid: %v", err)
		}
	}
	return f
}

// deadPID is far beyond Linux pid_max — guaranteed not to exist.
const deadPID = 1 << 30

// TestAcquireLockHeldLockIsRefused: while the OS lock is held, a second
// instance refuses to start. No prompt is asked and the lock file is never
// removed — deleting it is exactly what let two instances run side by side.
func TestAcquireLockHeldLockIsRefused(t *testing.T) {
	dir := t.TempDir()
	hold := holdLockFile(t, dir, strconv.Itoa(os.Getpid()))
	defer hold.Close()

	if _, err := acquireLock(dir); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("err = %v, want 'already running'", err)
	}

	// The lock file must survive the refused start...
	if _, err := os.Stat(filepath.Join(dir, "cqops.lock")); err != nil {
		t.Fatalf("lock file was removed: %v", err)
	}
	// ...and the holder keeps owning it: another attempt still fails.
	if _, err := acquireLock(dir); err == nil {
		t.Fatal("second attempt acquired a lock that is still held")
	}
}

// TestAcquireLockCannotBeStolenWithForeignProcessName is the regression test
// for the reported bypass: the recorded PID is alive but its process name is
// not "cqops" (a release binary named cqops-linux-amd64, a renamed copy, an
// AppImage — or, as here, the test binary). The old code read that as "stale
// lock", offered to delete the file, and then locked a freshly created inode
// while the holder kept the old one: two instances, one logbook and database.
func TestAcquireLockCannotBeStolenWithForeignProcessName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cqops.lock")
	// os.Getpid() is alive; /proc/<pid>/comm is the test binary name, never
	// "cqops".
	hold := holdLockFile(t, dir, strconv.Itoa(os.Getpid()))
	defer hold.Close()

	for i := 0; i < 3; i++ {
		if _, err := acquireLock(dir); err == nil {
			t.Fatalf("attempt %d stole a held lock", i+1)
		}
	}

	// The file the holder locked must still be the file at the lock path: a
	// delete-and-recreate would leave the holder on an orphaned inode.
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat lock file: %v", err)
	}
	hfi, err := hold.Stat()
	if err != nil {
		t.Fatalf("stat held file: %v", err)
	}
	if !os.SameFile(fi, hfi) {
		t.Fatal("lock file was replaced while the lock was held")
	}
}

// TestAcquireLockReusesLeftoverFileAfterOwnerExit: a lock file left behind by
// a crashed instance is harmless. The kernel released its lock, so the next
// start acquires the same file and rewrites the PID — no deletion, no prompt,
// no stale-lock question.
func TestAcquireLockReusesLeftoverFileAfterOwnerExit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cqops.lock")
	if err := os.WriteFile(path, []byte(strconv.Itoa(deadPID)+"\n"), 0o644); err != nil {
		t.Fatalf("write leftover lock: %v", err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat leftover: %v", err)
	}

	lk, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("leftover lock file must be reusable: %v", err)
	}
	defer lk.release()

	after, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after acquire: %v", err)
	}
	if !os.SameFile(before, after) {
		t.Error("lock file was replaced instead of reused")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock contains %q, want our PID %d", got, os.Getpid())
	}

	// After a clean release the same file is reusable again.
	lk.release()
	lk2, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	lk2.release()
}
