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
// PID, simulating a lock whose recorded owner no longer exists.
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

// TestAcquireLockLiveInstanceFailsWithoutPrompt: a genuine live CQOps holder
// fails fast — the operator can simply close that instance. No delete
// question is asked for a live process.
func TestAcquireLockLiveInstanceFailsWithoutPrompt(t *testing.T) {
	origPrompt := lockPrompt
	origOwner := lockOwnerIsCQOps
	t.Cleanup(func() { lockPrompt = origPrompt; lockOwnerIsCQOps = origOwner })
	prompted := false
	lockPrompt = func(string) bool { prompted = true; return true }
	lockOwnerIsCQOps = func(int) bool { return true }

	dir := t.TempDir()
	hold := holdLockFile(t, dir, strconv.Itoa(os.Getpid()))
	defer hold.Close()

	_, err := acquireLock(dir)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("err = %v, want 'already running'", err)
	}
	if prompted {
		t.Error("live instance must not trigger the delete-lock prompt")
	}
}

// TestAcquireLockReusedPIDPrompts: the lock is held and the recorded PID is
// alive, but it no longer belongs to a CQOps process (PID reuse) — the
// user must still get the delete prompt.
func TestAcquireLockReusedPIDPrompts(t *testing.T) {
	origPrompt := lockPrompt
	origOwner := lockOwnerIsCQOps
	t.Cleanup(func() { lockPrompt = origPrompt; lockOwnerIsCQOps = origOwner })
	var asked string
	lockPrompt = func(q string) bool { asked = q; return true }
	lockOwnerIsCQOps = func(int) bool { return false }

	dir := t.TempDir()
	hold := holdLockFile(t, dir, strconv.Itoa(os.Getpid()))
	defer hold.Close()

	lk, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("acquireLock with a reused PID and accepted prompt: %v", err)
	}
	defer lk.release()

	if asked == "" {
		t.Fatal("reused PID must trigger the prompt")
	}
	// The fresh lock is owned now.
	lockOwnerIsCQOps = func(int) bool { return true }
	_, err = acquireLock(dir)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Errorf("after accepted prompt the lock must be owned: %v", err)
	}
}

func TestAcquireLockStalePromptAccepted(t *testing.T) {
	origPrompt := lockPrompt
	origOwner := lockOwnerIsCQOps
	t.Cleanup(func() { lockPrompt = origPrompt; lockOwnerIsCQOps = origOwner })
	var asked string
	lockPrompt = func(q string) bool { asked = q; return true }

	dir := t.TempDir()
	hold := holdLockFile(t, dir, strconv.Itoa(deadPID))
	defer hold.Close()

	lk, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("acquireLock with stale lock and accepted prompt: %v", err)
	}
	defer lk.release()

	if !strings.Contains(asked, strconv.Itoa(deadPID)) {
		t.Errorf("prompt = %q, want the stale PID in it", asked)
	}
	if _, err := os.Stat(filepath.Join(dir, "cqops.lock")); err != nil {
		t.Errorf("lock file should exist after re-acquire: %v", err)
	}

	// We own the fresh lock — a further acquire must fail as "live".
	lockOwnerIsCQOps = func(int) bool { return true }
	_, err = acquireLock(dir)
	if err == nil || !strings.Contains(err.Error(), "already running") {
		t.Errorf("after accepted prompt the lock must be owned: %v", err)
	}
}

func TestAcquireLockStalePromptDeclined(t *testing.T) {
	orig := lockPrompt
	t.Cleanup(func() { lockPrompt = orig })
	lockPrompt = func(string) bool { return false }

	dir := t.TempDir()
	hold := holdLockFile(t, dir, strconv.Itoa(deadPID))
	defer hold.Close()

	_, err := acquireLock(dir)
	if err == nil {
		t.Fatal("declined prompt must fail the acquisition")
	}
	if !strings.Contains(err.Error(), "remove it manually") {
		t.Errorf("error = %q, want manual-removal hint", err)
	}
}
