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
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _ := im.Values()
	if theme != "bright" {
		t.Errorf("Values() theme = %q, want 'bright'", theme)
	}

	// Space → Dark
	m, _ := im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _ = im.Values()
	if theme != "dark" {
		t.Errorf("Values() theme = %q, want 'dark'", theme)
	}

	// Space → Orchid (yl)
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _ = im.Values()
	if theme != "yl" {
		t.Errorf("Values() theme = %q, want 'yl' (Orchid)", theme)
	}

	// Space → HighVis
	m, _ = im.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	im = m.(*IntegrationMenu)
	_, _, _, _, _, _, _, _, _, _, theme, _, _, _, _, _ = im.Values()
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
