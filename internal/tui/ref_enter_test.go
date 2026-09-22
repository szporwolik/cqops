package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/ref"
)

// TestRefEnterCommitsAndReturnsToQSO: Enter on a result row commits the
// reference to the QSO form and jumps back — same flow as DXC.
func TestRefEnterCommitsAndReturnsToQSO(t *testing.T) {
	m := newTestModel()
	m.screen = screenRef
	m.ref.searched = true
	m.ref.rows = []ref.Row{{Ref: "SP-001", RefType: ref.RefSOTA}}
	m.ref.cursor = 0

	_, _ = m.handleRefUpdate(tea.KeyPressMsg{Code: tea.KeyEnter}, nil)
	if m.screen != screenQSO {
		t.Errorf("screen = %v, want screenQSO after commit", m.screen)
	}
	if got := m.fields[fieldSOTA].Value(); got != "SP-001" {
		t.Errorf("SOTA field = %q, want SP-001", got)
	}
}

// TestRefEnterWithoutResultsSearches: with no results, Enter runs the search
// and stays on the REF screen.
func TestRefEnterWithoutResultsSearches(t *testing.T) {
	m := newTestModel()
	m.screen = screenRef
	m.ref.searched = false
	m.ref.rows = nil

	_, _ = m.handleRefUpdate(tea.KeyPressMsg{Code: tea.KeyEnter}, nil)
	if m.screen != screenRef {
		t.Errorf("screen = %v, want screenRef when searching", m.screen)
	}
}

// TestRefBackspaceAfterSearchClearsWholeSearch: once a lookup is done,
// Backspace deletes the whole search and clears the filtering (same as the
// logbook editor search) instead of deleting one character.
func TestRefBackspaceAfterSearchClearsWholeSearch(t *testing.T) {
	m := newTestModel()
	m.screen = screenRef
	m.ref.searched = true
	m.ref.rows = []ref.Row{{Ref: "SP-001", RefType: ref.RefSOTA}}
	m.ref.input.SetValue("wolin")
	m.ref.cursor = 1
	m.ref.scroll = 3

	_, _ = m.handleRefUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)
	if m.ref.searched {
		t.Error("searched should clear after Backspace")
	}
	if len(m.ref.rows) != 0 {
		t.Errorf("rows = %d, want 0 after Backspace", len(m.ref.rows))
	}
	if got := m.ref.input.Value(); got != "" {
		t.Errorf("input = %q, want cleared", got)
	}
	if m.ref.cursor != 0 || m.ref.scroll != 0 {
		t.Errorf("cursor/scroll = %d/%d, want 0/0", m.ref.cursor, m.ref.scroll)
	}
}

// TestRefBackspaceBeforeSearchEditsInput: without an active lookup, Backspace
// keeps editing the query one character at a time.
func TestRefBackspaceBeforeSearchEditsInput(t *testing.T) {
	m := newTestModel()
	m.screen = screenRef
	m.ref.input.Focus()
	m.ref.input.SetValue("ab")
	m.ref.searched = false

	_, _ = m.handleRefUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)
	if got := m.ref.input.Value(); got != "a" {
		t.Errorf("input = %q, want %q (one character removed)", got, "a")
	}
	if m.ref.searched {
		t.Error("Backspace before searching must not mark as searched")
	}
}
