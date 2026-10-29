package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// Shared helpers for tests that drive real ticks through Model.Update and need
// to run the commands a tick issues while keeping one particular result.

// pumpCommands executes a command tree the way the Bubble Tea runtime does: a
// batch is expanded and each leaf runs on its own goroutine.
func pumpCommands(cmd tea.Cmd, out chan<- tea.Msg) {
	if cmd == nil {
		return
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		select {
		case out <- msg:
		default: // the test already has what it needs
		}
		return
	}
	for _, sub := range batch {
		go pumpCommands(sub, out)
	}
}

// awaitTickMessage runs the commands a tick issued and returns the first
// message of type T. Messages from the other commands a tick starts (the tick
// timer, read-only view fetches, the local callbook status probe) are ignored:
// the point is to control WHEN one specific result reaches the update loop.
func awaitTickMessage[T any](t *testing.T, cmd tea.Cmd, what string) T {
	t.Helper()
	msgs := make(chan tea.Msg, 32)
	go pumpCommands(cmd, msgs)
	deadline := time.After(30 * time.Second)
	for {
		select {
		case msg := <-msgs:
			if v, ok := msg.(T); ok {
				return v
			}
		case <-deadline:
			var zero T
			t.Fatalf("no %s finished; the tick marked the work as started but produced no result", what)
			return zero
		}
	}
}
