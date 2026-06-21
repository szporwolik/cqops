package tui

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

var isWindows = runtime.GOOS == "windows"

// =============================================================================
// Normal config save-boundary validation tests (Pass 13)
// =============================================================================

// newSaveTestModel creates a Model with a temp config path and a valid station.
func newSaveTestModel(t *testing.T) *Model {
	t.Helper()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := config.DefaultConfig()
	cfg.General.Timezone = "UTC"
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {
			ID:          "default",
			Name: "Test",
			Station: config.Station{
				Callsign: "SP9MOA",
				Operator: "Op",
				Grid:     "JO90",
			},
		},
	}
	cfg.Integrations.QRZ.Enabled = false

	a := &app.App{
		Config:     cfg,
		ConfigPath: cfgPath,
		Logbook:    &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Operator: "Op", Grid: "JO90"}},
	}

	return New(a, nil)
}

func setCallsign(cfg *config.Config, call string) {
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = call
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
}

func setLocator(cfg *config.Config, grid string) {
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Grid = grid
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
}

func setWavelog(cfg *config.Config, url, key string) {
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Wavelog = &config.WavelogConfig{
		Enabled:          true,
		URL:              url,
		APIKey:           key,
		StationProfileID: "1",
	}
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
}

func setQRZ(cfg *config.Config, user, pass string) {
	cfg.Integrations.QRZ.Enabled = true
	cfg.Integrations.QRZ.User = user
	cfg.Integrations.QRZ.Pass = pass
}

// =============================================================================
// Valid save tests
// =============================================================================

func TestSaveConfig_ValidStation(t *testing.T) {
	m := newSaveTestModel(t)
	m.saveConfig("")
	// Should have saved the file to temp path.
	_, err := os.Stat(m.App.ConfigPath)
	if err != nil {
		t.Errorf("config file should exist after save: %v", err)
	}
}

func TestSaveConfig_ValidPortableCallsign(t *testing.T) {
	for _, call := range []string{"SP9MOA/P", "9A/SP9MOA", "DL/SP9MOA/P"} {
		m := newSaveTestModel(t)
		setCallsign(m.App.Config, call)
		m.saveConfig("")
		_, err := os.Stat(m.App.ConfigPath)
		if err != nil {
			t.Errorf("portable callsign %q should save: %v", call, err)
		}
	}
}

func TestSaveConfig_ValidLocators(t *testing.T) {
	for _, grid := range []string{"JO90", "JO90aa", "FN31pr"} {
		m := newSaveTestModel(t)
		setLocator(m.App.Config, grid)
		m.saveConfig("")
		_, err := os.Stat(m.App.ConfigPath)
		if err != nil {
			t.Errorf("locator %q should save: %v", grid, err)
		}
	}
}

func TestSaveConfig_EmptyLocatorOK(t *testing.T) {
	m := newSaveTestModel(t)
	setLocator(m.App.Config, "")
	m.saveConfig("")
	_, err := os.Stat(m.App.ConfigPath)
	if err != nil {
		t.Errorf("empty locator should save: %v", err)
	}
}

// =============================================================================
// Invalid save tests — must NOT write config file
// =============================================================================

func testSaveConfigBlocked(t *testing.T, cfgSetup func(*config.Config), desc string) {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	cfg := config.DefaultConfig()
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks = map[string]config.Logbook{
		"default": {
			ID:          "default",
			Name: "Test",
			Station:     config.Station{Callsign: "SP9MOA", Grid: "JO90"},
		},
	}
	cfg.Integrations.QRZ.Enabled = false

	cfgSetup(cfg)

	a := &app.App{Config: cfg, ConfigPath: cfgPath}
	m := New(a, nil)

	// First save a valid config to ensure the file exists.
	origCfg := config.DefaultConfig()
	origCfg.State.ActiveLogbook = "default"
	origCfg.Logbooks = map[string]config.Logbook{
		"default": {ID: "default", Name: "Test", Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
	}
	if err := config.Save(cfgPath, origCfg); err != nil {
		t.Fatalf("pre-save valid config: %v", err)
	}

	// Now try the invalid save.
	m.saveConfig("")

	// Verify the file still contains the original valid config (not overwritten).
	loaded, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("%s: load config after blocked save: %v", desc, err)
	}
	lb := loaded.Logbooks[loaded.State.ActiveLogbook]
	if lb.Station.Callsign != "SP9MOA" && desc != "callsign" {
		t.Errorf("%s: original callsign should be preserved", desc)
	}
}

func TestSaveConfig_Blocked_InvalidCallsign(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setCallsign(cfg, "!!!!")
	}, "callsign")
}

func TestSaveConfig_Blocked_InvalidLocator(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setLocator(cfg, "XXXX")
	}, "locator")
}

func TestSaveConfig_Blocked_WavelogHTTP(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setWavelog(cfg, "http://log.example.com", "test-key")
	}, "wavelog-http")
}

func TestSaveConfig_Blocked_WavelogNoAPIKey(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setWavelog(cfg, "https://log.example.com", "")
	}, "wavelog-no-key")
}

func TestSaveConfig_Blocked_QRZNoUser(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setQRZ(cfg, "", "secret")
	}, "qrz-no-user")
}

func TestSaveConfig_Blocked_QRZNoPass(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		setQRZ(cfg, "user", "")
	}, "qrz-no-pass")
}

func TestSaveConfig_Blocked_InvalidDistanceUnit(t *testing.T) {
	testSaveConfigBlocked(t, func(cfg *config.Config) {
		cfg.General.DistanceUnit = "feet"
	}, "distance-unit")
}

// =============================================================================
// Permission check
// =============================================================================

func TestSaveConfig_PermissionsAre0600(t *testing.T) {
	m := newSaveTestModel(t)
	m.saveConfig("")

	info, err := os.Stat(m.App.ConfigPath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	// 0600 is a Unix permission; on Windows os.WriteFile does not apply
	// POSIX owner-only semantics, so the check is meaningless there.
	if isWindows {
		t.Skip("Windows does not enforce POSIX 0600 file permissions")
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("config file permissions = %04o, want 0600", info.Mode().Perm())
	}
}
