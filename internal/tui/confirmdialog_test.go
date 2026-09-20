package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestDialogRender(t *testing.T) {
	dlg := NewDialog("Test Title", "Are you sure?",
		Option{Label: "Yes", Value: "yes"},
		Option{Label: "No", Value: "no"},
	)
	dlg.width = 80
	dlg.height = 24

	view := dlg.View()
	content := view.Content
	if content == "" {
		t.Error("Dialog render returned empty content")
	}
	if !strings.Contains(content, "Test Title") {
		t.Error("Dialog render missing title")
	}
	if !strings.Contains(content, "Are you sure?") {
		t.Error("Dialog render missing message")
	}
	if !strings.Contains(content, "Yes") {
		t.Error("Dialog render missing Yes button")
	}
	if !strings.Contains(content, "No") {
		t.Error("Dialog render missing No button")
	}

	// Output should not be excessively wide (capped at 56 modal + padding)
	w := lipgloss.Width(content)
	if w > 65 {
		t.Errorf("Dialog render width %d > 65 (should be compact)", w)
	}
}

func TestDialogRenderNarrow(t *testing.T) {
	dlg := NewDialog("T", "M",
		Option{Label: "Ok", Value: "ok"},
	)
	dlg.width = 30 // narrow terminal
	dlg.height = 10

	view := dlg.View()
	content := view.Content
	if content == "" {
		t.Error("Dialog render on narrow terminal returned empty")
	}
	// Should not panic; content can be anything non-empty
}

func TestDialogESCCancels(t *testing.T) {
	dlg := NewDialog("Quit", "Exit?",
		Option{Label: "Quit", Value: "quit"},
		Option{Label: "Cancel", Value: "cancel"},
	)
	dlg.width = 80
	dlg.height = 24

	// Press ESC
	updated, _ := dlg.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	dlg = updated.(DialogModel)

	if !dlg.Done() {
		t.Error("Dialog should be done after ESC")
	}
	if !dlg.Result.Cancelled {
		t.Error("Dialog should be cancelled after ESC")
	}
	if dlg.Result.Confirmed {
		t.Error("Dialog should not be confirmed after ESC")
	}
}

// TestConfirmDialogDoesNotKillTickChain covers the stuck-toast regression:
// a tick arriving while a confirm dialog is open used to return without
// re-arming the next tick, which stopped toast expiry (and the clock, GPS,
// and all other per-tick housekeeping) for the rest of the session.
func TestConfirmDialogDoesNotKillTickChain(t *testing.T) {
	m := newLifecycleTestModel(t)
	dlg := NewDialog("Quit", "Exit CQOps?",
		Option{Label: "Quit", Value: "quit"},
		Option{Label: "Cancel", Value: "cancel"},
	)
	m.confirm = &dlg

	// A tick arriving while the dialog is open must re-arm the next tick.
	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("tick chain broken while confirm dialog open — toasts would stick forever")
	}
	// tea.Batch wraps the re-armed tick together with other periodic
	// commands — at least one of them must be the next tick.
	bm, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("re-armed command returned %T, want tea.BatchMsg", cmd())
	}
	found := false
	for _, c := range bm {
		if inner := c(); inner != nil {
			if _, ok := inner.(tickMsg); ok {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("re-armed batch does not contain the next tickMsg")
	}

	// The tick must not mutate or dismiss the dialog.
	if m.confirm == nil || m.confirm.Done() {
		t.Fatal("tickMsg must not dismiss the confirm dialog")
	}
}

func TestDialogEnterConfirms(t *testing.T) {
	dlg := NewDialog("Quit", "Exit?",
		Option{Label: "Quit", Value: "quit"},
		Option{Label: "Cancel", Value: "cancel"},
	)
	dlg.width = 80
	dlg.height = 24

	// Press Enter (default selects first option)
	updated, _ := dlg.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	dlg = updated.(DialogModel)

	if !dlg.Done() {
		t.Error("Dialog should be done after Enter")
	}
	if !dlg.Result.Confirmed {
		t.Error("Dialog should be confirmed after Enter")
	}
	if dlg.Result.Value != "quit" {
		t.Errorf("Dialog value = %q; want 'quit'", dlg.Result.Value)
	}
}

func TestDialogSelectionChange(t *testing.T) {
	dlg := NewDialog("Test", "Choose",
		Option{Label: "A", Value: "a"},
		Option{Label: "B", Value: "b"},
		Option{Label: "C", Value: "c"},
	)
	dlg.width = 80
	dlg.height = 24

	// Press Tab to move selection
	updated, _ := dlg.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	dlg = updated.(DialogModel)

	// Confirm with Enter — should select second option
	updated, _ = dlg.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	dlg = updated.(DialogModel)

	if !dlg.Done() {
		t.Error("Dialog should be done after Enter")
	}
	if dlg.Result.Value != "b" {
		t.Errorf("Dialog value = %q; want 'b' (second option after Tab)", dlg.Result.Value)
	}
}

func TestDialogDangerOption(t *testing.T) {
	dlg := NewDialog("Delete", "Really?",
		Option{Label: "Delete", Value: "delete", Danger: true},
		Option{Label: "Cancel", Value: "cancel"},
	)
	dlg.width = 80
	dlg.height = 24

	// Danger option renders without panic
	view := dlg.View()
	content := view.Content
	if content == "" {
		t.Error("Dialog with danger option rendered empty")
	}
	if !strings.Contains(content, "Delete") {
		t.Error("Dialog render missing danger button")
	}

	// Press Enter (default is first option, which is danger)
	updated, _ := dlg.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	dlg = updated.(DialogModel)

	if dlg.Result.Value != "delete" {
		t.Errorf("Dialog value = %q; want 'delete'", dlg.Result.Value)
	}
}

func TestNoOldConfirmReferences(t *testing.T) {
	// Verify DialogModel and its types compile and are the canonical dialog types.
	// The old Confirm struct, ConfirmKind, NewConfirm, RenderConfirmOverlay
	// from confirm.go were all deleted in Phase 1.
	dlg := NewDialog("Test", "Body", Option{Label: "OK", Value: "ok"})
	_ = dlg.Title
	_ = dlg.Message
	_ = dlg.Options
}
