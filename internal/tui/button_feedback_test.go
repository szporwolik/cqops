package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// Button feedback regression: every config-menu test button reports through
// toasts (or a parent-consumed toast message field), never through inline
// result lines. Validation failures must be visible the same way.

func TestIntegrationMenuGPSTestValidationToast(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.gpsEnabled = true
	im.gpsService = 0 // serial
	im.fm.row = imGPSTest

	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	im = m.(*IntegrationMenu)
	if im.gpsToast != "GPS: port and baud rate required" {
		t.Errorf("gpsToast = %q, want port/baud validation", im.gpsToast)
	}
}

func TestIntegrationMenuGPSTestResultToast(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)

	m, _ := im.Update(gpsTestMsg{ok: true})
	im = m.(*IntegrationMenu)
	if im.gpsToast != "GPS: connection verified" {
		t.Errorf("gpsToast = %q, want verified result", im.gpsToast)
	}
	if !im.gpsNeedsPoll {
		t.Error("gpsNeedsPoll should be set after a passing GPS test")
	}
}

func TestIntegrationMenuAPRSTestValidationToast(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.aprsEnabled = true
	im.aprsService = 0 // APRS-IS
	im.inetOnline = true
	im.aprsServer.SetValue("") // no default server in the test config
	im.fm.row = imAPRSTest

	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	im = m.(*IntegrationMenu)
	if im.aprsToast != "APRS: server is required" {
		t.Errorf("aprsToast = %q, want server validation", im.aprsToast)
	}
}

func TestCallbookMenuTestValidationToast(t *testing.T) {
	cfg := config.DefaultConfig()
	cm := NewCallbookMenu(cfg)
	cm.inetOnline = true
	cm.fm.row = cmQRZTest

	m, _ := cm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	cm = m.(*CallbookMenu)
	if cm.TestToast != "QRZ: username and password required" {
		t.Errorf("TestToast = %q, want credentials validation", cm.TestToast)
	}
}

// TestHandleIntegrationUpdateShowsGpsToast pins the parent-side consumption:
// a pending GPS toast is shown and cleared on the next integration update.
func TestHandleIntegrationUpdateShowsGpsToast(t *testing.T) {
	m := newTestModel()
	m.ui.integrationMenu = NewIntegrationMenu(m.App.Config)
	m.ui.integrationMenu.gpsToast = "GPS: connection verified"

	_, _ = m.handleIntegrationUpdate(nil, nil)

	found := false
	for _, item := range m.toasts.Active() {
		if item.Message == "GPS: connection verified" && item.Level == ToastSuccess {
			found = true
		}
	}
	if !found {
		t.Error("GPS success toast not shown")
	}
	if m.ui.integrationMenu.gpsToast != "" {
		t.Error("gpsToast should be cleared after the toast is shown")
	}
}
