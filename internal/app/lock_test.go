package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func lockPath(dir string) string {
	return filepath.Join(dir, "cqops.lock")
}

func TestAcquireLock_HoldsExclusiveOwnership(t *testing.T) {
	origOwner := lockOwnerIsCQOps
	t.Cleanup(func() { lockOwnerIsCQOps = origOwner })
	// Treat the test process as a real CQOps instance so the held lock
	// fails fast instead of prompting.
	lockOwnerIsCQOps = func(int) bool { return true }

	dir := t.TempDir()
	lk, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("acquireLock: %v", err)
	}

	data, err := os.ReadFile(lockPath(dir))
	if err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != strconv.Itoa(os.Getpid()) {
		t.Errorf("lock contains %q, want our PID %d", got, os.Getpid())
	}

	// The same process on a second file description must be refused — the
	// OS lock, not file existence, decides ownership.
	if _, err := acquireLock(dir); err == nil {
		t.Fatal("second acquireLock should fail while the lock is held")
	}

	lk.release()
	lk2, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("re-acquire after release: %v", err)
	}
	lk2.release()
}

// TestAcquireLock_AtomicUnderRace verifies the OS lock serializes racers:
// exactly one of N simultaneous instances can hold the lock.
func TestAcquireLock_AtomicUnderRace(t *testing.T) {
	origOwner := lockOwnerIsCQOps
	t.Cleanup(func() { lockOwnerIsCQOps = origOwner })
	// Losers must fail fast (real CQOps owner) instead of prompting.
	lockOwnerIsCQOps = func(int) bool { return true }

	dir := t.TempDir()

	const n = 8
	start := make(chan struct{})
	lks := make([]*lockFile, n)
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			lks[i], errs[i] = acquireLock(dir)
		}(i)
	}
	close(start)
	wg.Wait()

	successes := 0
	for i := 0; i < n; i++ {
		if errs[i] == nil {
			successes++
			lks[i].release()
		}
	}
	if successes != 1 {
		t.Fatalf("exactly one instance must win the race, got %d", successes)
	}
}

// TestAcquireLock_ReleasedAfterOwnerDeath verifies the OS-backed lifetime
// guarantee: the kernel must release the lock when the owning process dies,
// even without any cleanup — a crashed instance can never leave the app
// locked out.
func TestAcquireLock_ReleasedAfterOwnerDeath(t *testing.T) {
	if os.Getenv("CQOPS_LOCK_HELPER") == "1" {
		// Child: hold the lock until killed.
		lk, err := acquireLock(os.Getenv("CQOPS_LOCK_DIR"))
		if err != nil {
			os.Exit(2)
		}
		_ = lk
		time.Sleep(time.Minute)
		os.Exit(0)
	}

	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=TestAcquireLock_ReleasedAfterOwnerDeath")
	cmd.Env = append(os.Environ(), "CQOPS_LOCK_HELPER=1", "CQOPS_LOCK_DIR="+dir)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Wait until the child holds the lock.
	deadline := time.Now().Add(5 * time.Second)
	for {
		lk, err := acquireLock(dir)
		if err == nil {
			lk.release()
			if time.Now().After(deadline) {
				t.Fatal("helper never acquired the lock")
			}
			time.Sleep(20 * time.Millisecond)
			continue
		}
		break // child holds it
	}

	// Kill the child without any cleanup — the kernel must release the lock.
	cmd.Process.Kill()
	cmd.Wait()

	lk, err := acquireLock(dir)
	if err != nil {
		t.Fatalf("lock not released after owner death: %v", err)
	}
	lk.release()
}
