package wsjtx

import (
	"testing"
)

// =============================================================================
// Listener lifecycle tests — no real UDP / WSJT-X required
// =============================================================================
//
// Start() calls applog which requires initialization. These tests focus on
// the lifecycle state machine (NewListener, Stop, IsActive) which are safe
// without a running application.

func TestNewListener(t *testing.T) {
	l := NewListener()
	if l == nil {
		t.Fatal("NewListener returned nil")
	}
	if l.Events == nil {
		t.Error("Events channel is nil")
	}
	if l.IsActive() {
		t.Error("new listener should not be active")
	}
}

func TestStopIdempotent(t *testing.T) {
	l := NewListener()
	// Double stop should not panic.
	l.Stop()
	l.Stop()
	if l.IsActive() {
		t.Error("listener should not be active after Stop")
	}
}

func TestStopBeforeStart(t *testing.T) {
	l := NewListener()
	// Stop before Start should be a no-op.
	l.Stop()
	l.Stop()
	if l.IsActive() {
		t.Error("Stop before Start should not activate the listener")
	}
}

func TestListenerMutexSafety(t *testing.T) {
	// Concurrent Stop calls should not race.
	l := NewListener()
	done := make(chan struct{})
	go func() {
		l.Stop()
		done <- struct{}{}
	}()
	go func() {
		l.Stop()
		done <- struct{}{}
	}()
	<-done
	<-done
	// If we get here without a race detector failure, the mutex is working.
}

func TestIsActiveInitialState(t *testing.T) {
	l := NewListener()
	if l.IsActive() {
		t.Error("IsActive should be false for new listener")
	}
	l.Stop()
	if l.IsActive() {
		t.Error("IsActive should be false after Stop on fresh listener")
	}
}

func TestCallbacksInitiallyNil(t *testing.T) {
	l := NewListener()
	if l.OnADIF != nil {
		t.Error("OnADIF callback should be nil on new listener")
	}
	if l.OnStatus != nil {
		t.Error("OnStatus callback should be nil on new listener")
	}
}

// =============================================================================
// Generation guard tests
// =============================================================================

func TestSnapshotOnADIF_RejectsStaleGeneration(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Simulate what Start() does: bump generation, snapshot it.
	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{}) // normally created by Start()
	l.mu.Unlock()

	// Current generation should return the callback.
	if cb := l.snapshotOnADIF(gen); cb == nil {
		t.Error("snapshotOnADIF returned nil for current generation")
	}

	// Simulate restart: deactivate and bump generation.
	l.mu.Lock()
	l.active = false
	l.generation++ // new generation starts, old gen is stale
	l.mu.Unlock()

	// Now the stale generation should be rejected.
	if cb := l.snapshotOnADIF(gen); cb != nil {
		t.Error("snapshotOnADIF returned non-nil for stale (stopped) generation")
	}
}

func TestSnapshotOnStatus_RejectsStaleGeneration(t *testing.T) {
	l := NewListener()
	l.OnStatus = func(call, grid string, freq uint64, mode, submode, report string) {}

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	if cb := l.snapshotOnStatus(gen); cb == nil {
		t.Error("snapshotOnStatus returned nil for current generation")
	}

	l.mu.Lock()
	l.active = false
	l.generation++
	l.mu.Unlock()

	if cb := l.snapshotOnStatus(gen); cb != nil {
		t.Error("snapshotOnStatus returned non-nil for stale (stopped) generation")
	}
}

func TestSnapshotOnADIF_RejectsWhenInactive(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Never activated — generation 0, not active.
	if cb := l.snapshotOnADIF(0); cb != nil {
		t.Error("snapshotOnADIF returned non-nil when listener is not active")
	}
}

func TestGenerationIncrements(t *testing.T) {
	l := NewListener()
	l.OnADIF = func(s string) {}

	// Simulate two starts.
	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen1 := l.generation
	l.mu.Unlock()

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen2 := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	if gen2 != gen1+1 {
		t.Errorf("generation did not increment: gen1=%d gen2=%d", gen1, gen2)
	}

	// gen1 should now be stale.
	if cb := l.snapshotOnADIF(gen1); cb != nil {
		t.Error("snapshotOnADIF returned non-nil for gen1 after gen2 was created")
	}
	if cb := l.snapshotOnADIF(gen2); cb == nil {
		t.Error("snapshotOnADIF returned nil for current gen2")
	}
}

func TestConcurrentSnapshotAndStop(t *testing.T) {
	// Concurrent deactivation and snapshot should not race.
	l := NewListener()
	l.OnADIF = func(s string) {}

	l.mu.Lock()
	l.stopLocked()
	l.generation++
	gen := l.generation
	l.active = true
	l.stop = make(chan struct{})
	l.mu.Unlock()

	done := make(chan struct{}, 2)
	go func() {
		l.snapshotOnADIF(gen)
		done <- struct{}{}
	}()
	go func() {
		l.mu.Lock()
		l.active = false
		l.generation++
		l.mu.Unlock()
		done <- struct{}{}
	}()
	<-done
	<-done
	// Race detector will catch any issues.
}
