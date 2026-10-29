package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Alt+0 is the F10 fallback. Some terminals reserve F10 for their own menu and
// never deliver it (and the raw Linux console does not deliver function keys at
// all, which is why the tab bar already shows Alt+1…Alt+9 there), so a station
// could end up with no way to quit.

// TestAltZeroOpensTheQuitDialog: in the main view Alt+0 must do exactly what F10
// does — show the quit confirmation without switching screens.
func TestAltZeroOpensTheQuitDialog(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyF10}, altKey('0')} {
		m := newLifecycleTestModel(t)
		m.screen = screenQSO

		upd, _ := m.Update(key)
		m = upd.(*Model)
		if m.confirm == nil {
			t.Fatalf("%q did not open the quit dialog", key.String())
		}
		if !strings.Contains(m.confirm.Title, "Quit") {
			t.Errorf("%q opened %q, want the quit dialog", key.String(), m.confirm.Title)
		}
		if m.screen != screenQSO {
			t.Errorf("%q switched the screen to %v", key.String(), m.screen)
		}
	}
}

// TestAltZeroQuitIsAdvertised: the key has to be discoverable, so the binding
// that feeds both the bottom bar and the ? overlay names it.
func TestAltZeroQuitIsAdvertised(t *testing.T) {
	km := DefaultKeyMap()
	help := km.Quit.Help()
	if !strings.Contains(help.Key, "Alt+0") || !strings.Contains(help.Desc, "Quit") {
		t.Fatalf("Quit help = %q/%q, want Alt+0 listed", help.Key, help.Desc)
	}
	if !km.Quit.Enabled() {
		t.Fatal("the quit binding is disabled")
	}

	m := newLayoutTestModel(t, 80, 24)
	bar := stripANSI(m.renderHelpBar())
	if !strings.Contains(bar, "Alt+0") {
		t.Errorf("the bottom bar does not mention Alt+0: %q", bar)
	}
	// The longer label must still fit the bar.
	if w := len([]rune(bar)); w > 80 {
		t.Errorf("the bottom bar is %d cells wide in an 80-cell terminal", w)
	}
}

// TestAltZeroQuitsTheWizard: the first-run wizard has its own quit path (it runs
// before the main model exists), so the fallback has to be wired there too.
func TestAltZeroQuitsTheWizard(t *testing.T) {
	for _, key := range []tea.KeyPressMsg{{Code: tea.KeyF10}, altKey('0')} {
		w := newTestWizard(t, "SP9MOA", "JO90AA")
		upd, cmd := w.Update(key)
		if cmd == nil {
			t.Fatalf("%q produced no command in the wizard", key.String())
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%q did not quit the wizard", key.String())
		}
		if _, ok := upd.(*Wizard); !ok {
			t.Errorf("%q returned %T, want the wizard", key.String(), upd)
		}
	}
}
