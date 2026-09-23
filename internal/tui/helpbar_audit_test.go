package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// =============================================================================
// Help overlay / bottom bar consistency regression tests (audit fixes)
// =============================================================================

// TestDXCHelpShowsRealFilterKeys: the DXC filter keys are t/b/m/c — the
// help overlay must show those, not the old placeholder keys.
func TestDXCHelpShowsRealFilterKeys(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.screen = screenDXC

	var keysSeen []string
	for _, b := range m.ActiveBindings() {
		keysSeen = append(keysSeen, b.Keys()...)
	}
	joined := strings.Join(keysSeen, "|")
	for _, want := range []string{"t", "b", "m", "c", "esc"} {
		if !strings.Contains("|"+joined+"|", "|"+want+"|") {
			t.Errorf("DXC help missing key %q; got %v", want, keysSeen)
		}
	}
	if strings.Contains(joined, "\\") {
		t.Errorf("DXC help still advertises stale backslash binding: %v", keysSeen)
	}
}

// TestCallbookMenuHasHelpEntries: the callbook menu was absent from the
// overlay and the bottom bar.
func TestCallbookMenuHasHelpEntries(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.screen = screenCallbook

	var keysSeen []string
	for _, b := range m.ActiveBindings() {
		keysSeen = append(keysSeen, b.Keys()...)
	}
	joined := strings.Join(keysSeen, "|")
	for _, want := range []string{"esc", " "} {
		if !strings.Contains("|"+joined+"|", "|"+want+"|") {
			t.Errorf("callbook help missing key %q; got %v", want, keysSeen)
		}
	}
	// Saving goes through the in-form [ Save & Back ] button — Ctrl+S is no
	// longer advertised, matching the General menu.
	if strings.Contains(joined, "ctrl+s") {
		t.Errorf("callbook help still advertises Ctrl+S: %v", keysSeen)
	}

	bar := m.minimalBarBindings()
	var barKeys []string
	for _, b := range bar {
		barKeys = append(barKeys, b.Keys()...)
	}
	barJoined := strings.Join(barKeys, "|")
	if strings.Contains(barJoined, "ctrl+s") || !strings.Contains(barJoined, "esc") {
		t.Errorf("callbook bottom bar should show Esc but not Ctrl+S: %v", barKeys)
	}
}

// TestConfigMenusDoNotAdvertiseEnterSave: every config menu and submenu form
// renders its own [ Save & Back ] button, so neither the ? overlay nor the
// bottom bar should advertise Enter as a separate save key.
func TestConfigMenusDoNotAdvertiseEnterSave(t *testing.T) {
	m := newLifecycleTestModel(t)

	m.screen = screenConfig
	assertNoEnterAdvertised(t, m)

	c := NewLogbookChooser(m.App, NewToastQueue())
	c.mode = chooserEdit
	m.ui.chooser = c
	m.screen = screenChooser
	assertNoEnterAdvertised(t, m)

	rc := NewRigChooser(m.App, NewToastQueue())
	rc.mode = rigChooserEdit
	m.ui.rigChooser = rc
	m.screen = screenRigEdit
	assertNoEnterAdvertised(t, m)

	cc := NewContestChooser(m.App, NewToastQueue())
	cc.mode = contestEdit
	m.ui.contestChooser = cc
	m.screen = screenContest
	assertNoEnterAdvertised(t, m)

	oc := NewOperatorChooser(m.App, NewToastQueue())
	oc.mode = operatorEdit
	m.ui.operatorChooser = oc
	m.screen = screenOperator
	assertNoEnterAdvertised(t, m)
}

// assertNoEnterAdvertised fails when the ? overlay or the bottom bar still
// advertises Enter on a screen that renders its own save button.
func assertNoEnterAdvertised(t *testing.T, m *Model) {
	t.Helper()
	var overlay []string
	for _, b := range m.ActiveBindings() {
		overlay = append(overlay, b.Keys()...)
	}
	if strings.Contains("|"+strings.Join(overlay, "|")+"|", "|enter|") {
		t.Errorf("screen %v: ? overlay still advertises Enter: %v", m.screen, overlay)
	}
	bar := barKeys(m.minimalBarBindings())
	if strings.Contains("|"+bar+"|", "|enter|") {
		t.Errorf("screen %v: bottom bar still advertises Enter: %s", m.screen, bar)
	}
}

// TestLogViewerEscExits: the log viewer previously had no working exit key.
func TestLogViewerEscExits(t *testing.T) {
	lv := NewLogViewer("Test Log")
	if lv.done {
		t.Fatal("new viewer must not be done")
	}
	upd, _ := lv.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	lv = upd.(*LogViewer)
	if !lv.done {
		t.Fatal("Esc should mark the log viewer done")
	}
}

// TestLogViewBottomBarHasBackEntry.
func TestLogViewBottomBarHasBackEntry(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.screen = screenLogView

	var keysSeen []string
	for _, b := range m.minimalBarBindings() {
		keysSeen = append(keysSeen, b.Keys()...)
	}
	if !strings.Contains(strings.Join(keysSeen, "|"), "esc") {
		t.Errorf("log viewer bottom bar missing Esc back entry: %v", keysSeen)
	}
}

// TestMainMenuEscExits: the bottom bar advertised Esc Back, which did nothing.
func TestMainMenuEscExits(t *testing.T) {
	mm := NewMainMenu()
	if mm.done {
		t.Fatal("new menu must not be done")
	}
	upd, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm = upd.(*MainMenu)
	if !mm.done {
		t.Fatal("Esc should mark the main menu done")
	}
}

// TestMainMenuDigitsJump: digits 1-7 select the menu entry directly.
func TestMainMenuDigitsJump(t *testing.T) {
	mm := NewMainMenu()
	upd, _ := mm.Update(tea.KeyPressMsg{Code: '1'})
	mm = upd.(*MainMenu)
	if mm.action != "logbook" {
		t.Errorf("digit 1: action = %q, want logbook", mm.action)
	}

	upd, _ = mm.Update(tea.KeyPressMsg{Code: '5'})
	mm = upd.(*MainMenu)
	if mm.action != "integration" {
		t.Errorf("digit 5: action = %q, want integration", mm.action)
	}
}

// TestMainMenuOrder pins the entry order: station identity and operation
// first, online services next, preferences last.
func TestMainMenuOrder(t *testing.T) {
	mm := NewMainMenu()
	want := []string{"Logbooks", "Operators", "Rigs", "Contests", "Integrations", "Callbook", "General"}
	if len(mm.items) != len(want) {
		t.Fatalf("menu has %d items, want %d", len(mm.items), len(want))
	}
	for i, label := range want {
		if mm.items[i].label != label {
			t.Errorf("item %d = %q, want %q", i, mm.items[i].label, label)
		}
	}
}

// TestKeyMapHasNoDeadBindings: removed bindings must not carry empty keys.
func TestKeyMapHasNoDeadBindings(t *testing.T) {
	k := DefaultKeyMap()
	for _, b := range []key.Binding{k.QSOForm, k.Partner, k.APRS, k.PSKReporter,
		k.DXC, k.LogEditor, k.Config, k.Logs, k.Ref, k.BPL, k.Delete, k.Lookup,
		k.NextField, k.PrevField, k.NextRow, k.PrevRow, k.CycleUp, k.CycleDown,
		k.Enter, k.Cancel, k.CycleLogbook, k.CycleRig, k.CycleContest,
		k.CycleOperator, k.Spot, k.Help, k.Quit} {
		if len(b.Keys()) == 0 {
			t.Errorf("binding %q has no keys", b.Help().Desc)
		}
	}
}

// barKeys flattens the key names of a binding list into a pipe-joined string.
func barKeys(bindings []key.Binding) string {
	var all []string
	for _, b := range bindings {
		all = append(all, b.Keys()...)
	}
	return strings.Join(all, "|")
}

// TestBottomBarsShowCoreActionsOnly: the bar keeps the primary action per
// screen; secondary actions live behind the ? overlay.
func TestBottomBarsShowCoreActionsOnly(t *testing.T) {
	m := newLifecycleTestModel(t)

	// Chooser list — Enter Edit + Esc only, no Create/Delete/Activate.
	m.screen = screenChooser
	keys := barKeys(m.minimalBarBindings())
	for _, want := range []string{"enter", "esc"} {
		if !strings.Contains("|"+keys+"|", "|"+want+"|") {
			t.Errorf("chooser list bar missing %q: %s", want, keys)
		}
	}
	for _, stale := range []string{"insert", "delete", "space"} {
		if strings.Contains("|"+keys+"|", "|"+stale+"|") {
			t.Errorf("chooser list bar still shows secondary key %q: %s", stale, keys)
		}
	}

	// Log editor list — Enter Edit + Esc.
	m.screen = screenLogbookEditor
	m.ui.logbookEditor = NewLogbookEditor(LogbookEditorConfig{})
	keys = barKeys(m.minimalBarBindings())
	if !strings.Contains("|"+keys+"|", "|enter|") {
		t.Errorf("editor list bar missing Enter: %s", keys)
	}
}
