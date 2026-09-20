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
