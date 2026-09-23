package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// TestGeneralMenuNotificationsRowOpensSubmenu: Enter and Space on the
// Notifications row open the submenu without saving or closing General.
func TestGeneralMenuNotificationsRowOpensSubmenu(t *testing.T) {
	gm := NewGeneralMenu(config.DefaultConfig())
	gm.fm.row = generalRowNotifications

	upd, _ := gm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	gm = upd.(*GeneralMenu)
	if !gm.goNotifications {
		t.Error("Enter on Notifications row should request the submenu")
	}
	if gm.done || gm.saved {
		t.Errorf("opening the submenu must not save/close General: done=%v saved=%v", gm.done, gm.saved)
	}

	gm.goNotifications = false
	upd, _ = gm.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	gm = upd.(*GeneralMenu)
	if !gm.goNotifications {
		t.Error("Space on Notifications row should request the submenu")
	}
	if gm.done || gm.saved {
		t.Errorf("opening the submenu must not save/close General: done=%v saved=%v", gm.done, gm.saved)
	}
}

// TestNotificationsSubmenuRoundTrip: opening Notifications from General and
// closing it (back or saved) returns to the General menu with its state
// preserved.
func TestNotificationsSubmenuRoundTrip(t *testing.T) {
	m := newTestModel()
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	m.ui.configMenu = NewGeneralMenu(m.App.Config)
	m.ui.configMenu.width = 100
	m.ui.configMenu.height = 30
	m.screen = screenConfig

	// Open the submenu from General.
	m.ui.configMenu.goNotifications = true
	_, _ = m.handleConfigUpdate(nil, nil)
	if m.screen != screenNotifications {
		t.Fatalf("screen after open = %v, want screenNotifications", m.screen)
	}
	if !m.ui.notifMenu.fromGeneral {
		t.Error("notifMenu.fromGeneral should be set when opened from General")
	}
	if m.ui.configMenu.done {
		t.Error("General must stay open while Notifications is shown")
	}

	// Esc/back returns to General, not the main menu.
	m.ui.notifMenu.done = true
	m.ui.notifMenu.goBack = true
	_, _ = m.handleNotificationsUpdate(nil, nil)
	if m.screen != screenConfig {
		t.Errorf("back from Notifications should return to General, screen=%v", m.screen)
	}

	// Saving also returns to General.
	m.ui.notifMenu = NewNotificationsMenu(m.App.Config)
	m.ui.notifMenu.fromGeneral = true
	m.ui.notifMenu.done = true
	m.ui.notifMenu.saved = true
	_, _ = m.handleNotificationsUpdate(nil, nil)
	if m.screen != screenConfig {
		t.Errorf("saving Notifications should return to General, screen=%v", m.screen)
	}
}
