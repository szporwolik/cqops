package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

func TestIntegrationMenu_HTTPThemeDefaults(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}

	im := NewIntegrationMenu(cfg)
	if im.httpTheme != 0 {
		t.Errorf("default theme = %d, want 0 (Bright)", im.httpTheme)
	}
}

func TestIntegrationMenu_HTTPThemeToggle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}

	im := NewIntegrationMenu(cfg)
	im.focus = imHTTPTheme

	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if im.httpTheme != 1 {
		t.Errorf("after first Space: theme = %d, want 1 (Dark)", im.httpTheme)
	}

	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if im.httpTheme != 2 {
		t.Errorf("after second Space: theme = %d, want 2 (Orchid)", im.httpTheme)
	}

	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if im.httpTheme != 3 {
		t.Errorf("after third Space: theme = %d, want 3 (HighVis)", im.httpTheme)
	}

	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if im.httpTheme != 0 {
		t.Errorf("after fourth Space: theme = %d, want 0 (Bright)", im.httpTheme)
	}
}

func TestIntegrationMenu_HTTPThemeDarkFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Theme = "dark"

	im := NewIntegrationMenu(cfg)
	if im.httpTheme != 1 {
		t.Errorf("theme from config 'dark' = %d, want 1 (Dark)", im.httpTheme)
	}
}

func TestIntegrationMenu_HTTPThemeValues(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}

	im := NewIntegrationMenu(cfg)
	im.focus = imHTTPTheme

	// Default → Bright
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _, _, _, _ := im.Values()
	if theme != "bright" {
		t.Errorf("Values() theme = %q, want 'bright'", theme)
	}

	// Space → Dark
	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _, _, _, _ = im.Values()
	if theme != "dark" {
		t.Errorf("Values() theme = %q, want 'dark'", theme)
	}

	// Space → Orchid (yl)
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _, _, _, _ = im.Values()
	if theme != "yl" {
		t.Errorf("Values() theme = %q, want 'yl' (Orchid)", theme)
	}

	// Space → HighVis
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _, _, _, _ = im.Values()
	if theme != "hivis" {
		t.Errorf("Values() theme = %q, want 'hivis'", theme)
	}
}

func TestIntegrationMenu_HTTPThemeYlFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Theme = "yl"

	im := NewIntegrationMenu(cfg)
	if im.httpTheme != 2 {
		t.Errorf("theme from config 'yl' = %d, want 2 (Orchid)", im.httpTheme)
	}
}

func TestIntegrationMenu_HTTPThemeHighVisFromConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Theme = "hivis"
	im := NewIntegrationMenu(cfg)
	if im.httpTheme != 3 {
		t.Errorf("theme from config 'hivis' = %d, want 3 (HighVis)", im.httpTheme)
	}
}

func TestIntegrationMenu_HTTPThemeVisibleOnlyWhenEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}

	im := NewIntegrationMenu(cfg)

	im.focus = imHTTPTheme
	if im.isPositionVisible(im.focus) {
		t.Error("theme should not be visible when HTTP server is disabled")
	}

	im.focus = imHTTPChk
	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	im.focus = imHTTPTheme
	if !im.isPositionVisible(im.focus) {
		t.Error("theme should be visible when HTTP server is enabled")
	}
}

func TestIntegrationMenu_HTTPThemeRender(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Enabled = true

	im := NewIntegrationMenu(cfg)
	im.width = 100
	im.height = 40

	view := im.View()
	content := view.Content
	if content == "" {
		t.Fatal("view is empty")
	}
	if !strings.Contains(content, "Theme:") {
		t.Error("view should contain 'Theme:'")
	}
	if !strings.Contains(content, "Bright") {
		t.Error("view should contain 'Bright'")
	}
}

func TestIntegrationMenu_HTTPTLSToggle(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Enabled = true

	im := NewIntegrationMenu(cfg)
	im.focus = imHTTPTLS
	if im.httpTLS {
		t.Fatal("TLS should default to off")
	}

	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if !im.httpTLS {
		t.Fatal("Space should enable TLS")
	}

	_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, tlsOn, _, _ := im.Values()
	if !tlsOn {
		t.Error("Values() httpTLS = false, want true after toggle")
	}

	// Toggling off again.
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	if im.httpTLS {
		t.Error("second Space should disable TLS")
	}
}

func TestIntegrationMenu_HTTPTLSCertKeyPairValidation(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Enabled = true
	cfg.Integrations.HTTPServer.TLSEnabled = true

	im := NewIntegrationMenu(cfg)
	im.httpTLS = true
	im.httpTLSCert.SetValue("/tmp/cert.pem")

	im.focus = imHTTPPort
	_, _ = im.Update(tea.KeyPressMsg{Text: "\x13"})
	if im.SaveError == "" {
		t.Error("save with cert but no key should set SaveError")
	}

	// SaveError is cleared by the parent after showing the toast; reset it
	// here so it doesn't mask the second save.
	im.SaveError = ""
	im.httpTLSKey.SetValue("/tmp/key.pem")
	_, _ = im.Update(tea.KeyPressMsg{Text: "\x13"})
	if !im.saved {
		t.Errorf("save with cert+key should succeed, SaveError = %q", im.SaveError)
	}
}

// TestIntegrationMenu_HTTPTLSFocusOrder: the TLS row is rendered right
// after Theme, so tabbing down from Theme must land on it — the focus
// index order must match the visual row order.
func TestIntegrationMenu_HTTPTLSFocusOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {Name: "Default", Station: config.Station{Callsign: "SP9MOA", Grid: "KO00"}},
	}
	cfg.Integrations.HTTPServer.Enabled = true

	im := NewIntegrationMenu(cfg)
	im.focus = imHTTPTheme

	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	im = m.(*IntegrationMenu)
	if im.focus != imHTTPTLS {
		t.Errorf("tab from Theme landed on focus %d, want imHTTPTLS (%d)", im.focus, imHTTPTLS)
	}

	// With TLS off, the cert/key rows are hidden — tabbing must skip them.
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	im = m.(*IntegrationMenu)
	if im.focus != imHTTPHdr1 {
		t.Errorf("tab from TLS (disabled) landed on focus %d, want imHTTPHdr1 (%d)", im.focus, imHTTPHdr1)
	}

	// Next tab reaches the cert field (TLS enabled for the sub-fields).
	im.focus = imHTTPTLS
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace}) // enable TLS
	im = m.(*IntegrationMenu)
	im.focus = imHTTPTLS
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	im = m.(*IntegrationMenu)
	if im.focus != imHTTPTLSCert {
		t.Errorf("tab from TLS landed on focus %d, want imHTTPTLSCert (%d)", im.focus, imHTTPTLSCert)
	}
}

func TestIntegrationMenuPrefillsDXCLoginFromLogbook(t *testing.T) {
	// First available callsign (sorted by logbook id) prefills the login.
	cfg := config.DefaultConfig()
	cfg.Logbooks = map[string]config.Logbook{
		"b": {Station: config.Station{Callsign: "", Grid: "JO90"}},
		"a": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
	}
	im := NewIntegrationMenu(cfg)
	if im.dxcLogin.Value() != "SP9MOA" {
		t.Errorf("dxc login = %q, want SP9MOA", im.dxcLogin.Value())
	}

	// An already configured login always wins.
	cfg2 := config.DefaultConfig()
	cfg2.Integrations.DXC.Login = "SP9EGL"
	cfg2.Logbooks = map[string]config.Logbook{
		"a": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
	}
	im2 := NewIntegrationMenu(cfg2)
	if im2.dxcLogin.Value() != "SP9EGL" {
		t.Errorf("existing login should win, got %q", im2.dxcLogin.Value())
	}

	// No logbooks → no prefill.
	im3 := NewIntegrationMenu(config.DefaultConfig())
	if im3.dxcLogin.Value() != "" {
		t.Errorf("expected empty login, got %q", im3.dxcLogin.Value())
	}
}
