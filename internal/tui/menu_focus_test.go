package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// fakeRows is a configurable focusableRows for testing the engine itself.
type fakeRows struct {
	visible []bool
	focused int // last row focused, -1 when none
	blurred int
}

func (f *fakeRows) rowCount() int { return len(f.visible) }
func (f *fakeRows) rowVisible(i int) bool {
	return i >= 0 && i < len(f.visible) && f.visible[i]
}
func (f *fakeRows) blurAll() { f.blurred++ }
func (f *fakeRows) focusRow(i int) tea.Cmd {
	f.focused = i
	return nil
}

func TestMenuFocusEngine(t *testing.T) {
	m := &fakeRows{visible: []bool{true, true, false, true}}
	f := menuFocus{row: 0}

	// next skips the hidden row and wraps.
	f.next(m)
	if f.row != 1 {
		t.Fatalf("next: row=%d, want 1", f.row)
	}
	f.next(m)
	if f.row != 3 {
		t.Fatalf("next skips hidden row: row=%d, want 3", f.row)
	}
	f.next(m)
	if f.row != 0 {
		t.Fatalf("next wraps: row=%d, want 0", f.row)
	}
	if m.focused != 0 || m.blurred != 3 {
		t.Fatalf("next side effects: focused=%d blurred=%d, want 0/3", m.focused, m.blurred)
	}

	// prev wraps and skips hidden rows.
	f.prev(m)
	if f.row != 3 {
		t.Fatalf("prev wraps: row=%d, want 3", f.row)
	}
	f.prev(m)
	if f.row != 1 {
		t.Fatalf("prev skips hidden row: row=%d, want 1", f.row)
	}

	// Button navigation from the last row.
	f.row = 3
	handled, _ := f.onKey(tea.KeyPressMsg{Code: tea.KeyTab}, m, nil)
	if !handled || !f.btn.Focus || f.row != -1 {
		t.Fatalf("tab on last row: handled=%v btn=%v row=%d", handled, f.btn.Focus, f.row)
	}
	if m.blurred != 6 {
		t.Fatalf("button grab must blur: blurred=%d", m.blurred)
	}

	// Space on the button calls save.
	saved := false
	handled, _ = f.onKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}, m, func() tea.Cmd {
		saved = true
		return nil
	})
	if !handled || !saved {
		t.Fatal("space on button should save")
	}

	// Esc on the button falls through to the caller.
	handled, _ = f.onKey(tea.KeyPressMsg{Code: tea.KeyEsc}, m, nil)
	if handled {
		t.Fatal("esc should fall through to the caller")
	}

	// Other keys are swallowed while the button is focused.
	handled, _ = f.onKey(tea.KeyPressMsg{Text: "x"}, m, nil)
	if !handled {
		t.Fatal("unrelated keys should be swallowed while the button is focused")
	}

	// Leaving the button: next → first visible row, prev → last visible row.
	handled, _ = f.onKey(tea.KeyPressMsg{Code: tea.KeyTab}, m, nil)
	if !handled || f.row != 0 {
		t.Fatalf("tab from button: row=%d, want 0", f.row)
	}
	f.focusButton(m)
	handled, _ = f.onKey(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, m, nil)
	if !handled || f.row != 3 {
		t.Fatalf("shift+tab from button: row=%d, want 3", f.row)
	}

	// Up on the first row reaches the button.
	f.row = 0
	handled, _ = f.onKey(tea.KeyPressMsg{Code: tea.KeyUp}, m, nil)
	if !handled || !f.btn.Focus || f.row != -1 {
		t.Fatal("up on first row should focus the button")
	}

	// reset returns to the first row with the button unfocused.
	f.focusButton(m)
	f.reset()
	if f.row != 0 || f.btn.Focus {
		t.Fatalf("reset: row=%d btn=%v, want 0/false", f.row, f.btn.Focus)
	}
}

// menuAdapter binds a real config menu to its menuFocus so the same
// navigation invariants can be checked against every menu.
type menuAdapter struct {
	name string
	m    focusableRows
	fm   *menuFocus
}

func newTestMenuAdapters(t *testing.T) []menuAdapter {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Logbooks = map[string]config.Logbook{"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}}}

	gm := NewGeneralMenu(cfg)
	nm := NewNotificationsMenu(cfg)
	im := NewIntegrationMenu(cfg)
	cm := NewCallbookMenu(cfg)
	cc := newTestContestChooser(t, nil)
	rc := newTestRigChooser(t)
	oc := newTestOperatorChooser(t)
	le := NewLogbookEditor(LogbookEditorConfig{DB: nil, StationOperator: "OP", StationGrid: "JO90", StationCall: "SP9MOA"})

	a := newChooserTestApp(t)
	lc := NewLogbookChooser(a, NewToastQueue())

	w := newTestWizard(t, "SP9MOA", "JO90")

	return []menuAdapter{
		{"general", gm, &gm.fm},
		{"notifications", nm, &nm.fm},
		{"integrations", im, &im.fm},
		{"callbook", cm, &cm.fm},
		{"contest", cc, &cc.fm},
		{"rig", rc, &rc.fm},
		{"operator", oc, &oc.fm},
		{"editor", le, &le.fm},
		{"logbook station form", lc, &lc.fm},
		{"wizard station step", w, &w.fm},
	}
}

// TestMenuFocusAllMenusWalk walks every menu's rows through the shared
// engine and asserts that only visible rows receive focus and that the walk
// wraps back to the first visible row.
func TestMenuFocusAllMenusWalk(t *testing.T) {
	for _, a := range newTestMenuAdapters(t) {
		t.Run(a.name, func(t *testing.T) {
			m := a.m
			n := m.rowCount()
			visible := 0
			for i := 0; i < n; i++ {
				if m.rowVisible(i) {
					visible++
				}
			}
			if visible == 0 {
				t.Fatal("menu has no visible rows")
			}
			a.fm.reset()

			start := a.fm.firstVisible(m)
			if !m.rowVisible(start) {
				t.Fatalf("firstVisible returned hidden row %d", start)
			}

			visited := make(map[int]bool)
			cur := start
			a.fm.row = cur
			steps := 0
			for !visited[cur] {
				if !m.rowVisible(cur) {
					t.Fatalf("engine landed on hidden row %d", cur)
				}
				visited[cur] = true
				if steps++; steps > n {
					t.Fatal("walk did not wrap within the row count")
				}
				if cmd := a.fm.next(m); cmd != nil {
					_ = cmd
				}
				cur = a.fm.row
			}
			if cur != start {
				t.Fatalf("walk wrapped to row %d, want start %d", cur, start)
			}
			if len(visited) != visible {
				t.Errorf("walk visited %d rows, want %d visible rows", len(visited), visible)
			}

			if a.fm.firstVisible(m) != start {
				t.Errorf("firstVisible = %d, want %d", a.fm.firstVisible(m), start)
			}
			if !m.rowVisible(a.fm.lastVisible(m)) {
				t.Errorf("lastVisible returned hidden row %d", a.fm.lastVisible(m))
			}
		})
	}
}
