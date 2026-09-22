package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// PSK Reporter is opt-in: the F5 panel is off by default and is enabled from
// the Integrations menu.

func TestPSKDisabledBlocksF5(t *testing.T) {
	m := newTestModel()
	m.inetOnline = true

	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF5})
	if !handled {
		t.Error("F5 should be handled (warning toast) even when disabled")
	}
	if m.screen == screenPSKReporter {
		t.Fatal("F5 must not open the PSK screen while disabled")
	}

	m.App.Config.Integrations.PSK.Enabled = true
	_, handled = m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF5})
	if !handled || m.screen != screenPSKReporter {
		t.Fatalf("F5 with PSK enabled: handled=%v screen=%v, want PSK screen", handled, m.screen)
	}
}

func TestPSKFetchGatedOnEnable(t *testing.T) {
	m := newTestModel()
	m.screen = screenPSKReporter
	m.inetOnline = true

	// Disabled: no fetch is dispatched even though we are online.
	_, cmd := m.handlePSKReporterUpdate(nil, nil)
	if cmd != nil || m.psk.fetching {
		t.Fatalf("disabled PSK must not fetch: cmd=%v fetching=%v", cmd, m.psk.fetching)
	}

	// Enabled: the initial fetch is dispatched.
	m.App.Config.Integrations.PSK.Enabled = true
	_, cmd = m.handlePSKReporterUpdate(nil, nil)
	if cmd == nil || !m.psk.fetching {
		t.Fatalf("enabled PSK should fetch: cmd=%v fetching=%v", cmd, m.psk.fetching)
	}
}

func TestIntegrationMenuPSKDefaultOffAndToggles(t *testing.T) {
	im := NewIntegrationMenu(config.DefaultConfig())
	if im.pskEnabled {
		t.Fatal("PSK Reporter should default to off")
	}

	im.fm.row = imPSKChk
	im.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if !im.pskEnabled {
		t.Fatal("Space should enable PSK Reporter")
	}
	if !strings.Contains(im.View().Content, "PSK Reporter") {
		t.Fatal("menu should render the PSK Reporter row")
	}

	im.Update(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if im.pskEnabled {
		t.Fatal("second Space should disable PSK Reporter")
	}
}
