package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// TestGeneralMenuEnterSaves: Enter saves and closes the General settings
// menu — Space remains the toggle/cycle key, Ctrl+S no longer saves here.
func TestGeneralMenuEnterSaves(t *testing.T) {
	gm := NewGeneralMenu(config.DefaultConfig())
	upd, _ := gm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	gm = upd.(*GeneralMenu)
	if !gm.done || !gm.saved {
		t.Errorf("Enter should save and close the menu: done=%v saved=%v", gm.done, gm.saved)
	}
}

// TestGeneralMenuCtrlSNoLongerSaves: Ctrl+S was replaced by Enter.
func TestGeneralMenuCtrlSNoLongerSaves(t *testing.T) {
	gm := NewGeneralMenu(config.DefaultConfig())
	upd, _ := gm.Update(tea.KeyPressMsg{Text: "\x13"})
	gm = upd.(*GeneralMenu)
	if gm.done || gm.saved {
		t.Errorf("Ctrl+S should not save: done=%v saved=%v", gm.done, gm.saved)
	}
}

// TestGeneralMenuSpaceStillCycles: Space must remain the toggle key.
func TestGeneralMenuSpaceStillCycles(t *testing.T) {
	gm := NewGeneralMenu(config.DefaultConfig())
	gm.fm.row = 0 // Units
	before := gm.distanceUnit
	upd, _ := gm.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	gm = upd.(*GeneralMenu)
	if gm.distanceUnit == before {
		t.Error("Space should cycle the Units setting")
	}
}

// The Integration, Callbook and Notifications menus follow the General menu
// convention: saving goes through the in-form [ Save & Back ] button, so
// Ctrl+S no longer saves.

func TestIntegrationMenuCtrlSNoLongerSaves(t *testing.T) {
	im := NewIntegrationMenu(config.DefaultConfig())
	upd, _ := im.Update(tea.KeyPressMsg{Text: "\x13"})
	im = upd.(*IntegrationMenu)
	if im.done || im.saved {
		t.Errorf("Ctrl+S should not save: done=%v saved=%v", im.done, im.saved)
	}
}

func TestCallbookMenuCtrlSNoLongerSaves(t *testing.T) {
	cm := NewCallbookMenu(config.DefaultConfig())
	upd, _ := cm.Update(tea.KeyPressMsg{Text: "\x13"})
	cm = upd.(*CallbookMenu)
	if cm.done || cm.saved {
		t.Errorf("Ctrl+S should not save: done=%v saved=%v", cm.done, cm.saved)
	}
}

func TestNotificationsMenuCtrlSNoLongerSaves(t *testing.T) {
	nm := NewNotificationsMenu(config.DefaultConfig())
	upd, _ := nm.Update(tea.KeyPressMsg{Text: "\x13"})
	nm = upd.(*NotificationsMenu)
	if nm.done || nm.saved {
		t.Errorf("Ctrl+S should not save: done=%v saved=%v", nm.done, nm.saved)
	}
}
