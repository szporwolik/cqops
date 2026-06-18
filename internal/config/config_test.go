package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// =============================================================================
// Cross-platform path expectation helpers
// =============================================================================

// expectedConfigDir returns the ConfigDir() path expected when HOME is set to
// home and any XDG_CONFIG_HOME override is applied.
func expectedConfigDir(home, xdgConfigHome string) string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "cqops")
	}
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "cqops")
	}
	return filepath.Join(home, ".config", "cqops")
}

// expectedDataDir returns the DataDir() path expected when HOME is set to home.
func expectedDataDir(home string) string {
	switch {
	case runtime.GOOS == "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "cqops", "database")
	case runtime.GOOS == "darwin":
		return filepath.Join(home, "Library", "Application Support", "cqops", "database")
	default: // linux, etc.
		return filepath.Join(home, ".local", "share", "cqops", "database")
	}
}

// expectedLogDir returns the LogDir() path expected when HOME is set to home.
func expectedLogDir(home string) string {
	switch {
	case runtime.GOOS == "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "cqops", "logs")
	case runtime.GOOS == "darwin":
		return filepath.Join(home, "Library", "Application Support", "cqops", "logs")
	default:
		return filepath.Join(home, ".local", "share", "cqops", "logs")
	}
}

// isolateHome sets up an isolated $HOME (and platform-specific vars) under
// a temp directory. Returns the temp home path.
func isolateHome(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", filepath.Join(tmp, "AppData", "Roaming"))
	}
	return tmp
}

// =============================================================================
// DefaultConfig tests
// =============================================================================

func TestDefaultConfig_HasDefaults(t *testing.T) {
	cfg := DefaultConfig()

	// General
	if cfg.General.Timezone == "" {
		t.Error("DefaultConfig: timezone should not be empty")
	}
	if cfg.General.DistanceUnit != "km" {
		t.Errorf("DefaultConfig: distance_unit = %q; want km", cfg.General.DistanceUnit)
	}

	// State — ActiveLogbook should be a non-empty hash ID.
	if cfg.State.ActiveLogbook == "" {
		t.Error("DefaultConfig: active_logbook should not be empty")
	}

	// Logbooks — should have exactly one entry with hash ID key.
	if len(cfg.Logbooks) == 0 {
		t.Fatal("DefaultConfig: no logbooks")
	}
	if len(cfg.Logbooks) != 1 {
		t.Fatalf("DefaultConfig: expected 1 logbook, got %d", len(cfg.Logbooks))
	}
	var defLB Logbook
	for _, lb := range cfg.Logbooks {
		defLB = lb
		break
	}
	if defLB.ID == "" {
		t.Error("DefaultConfig: logbook ID should not be empty")
	}
	if defLB.Description == "" {
		t.Error("DefaultConfig: default logbook description is empty")
	}

	// Station defaults (should be zero-value — user fills them in)
	if defLB.Station.Callsign != "" {
		t.Error("DefaultConfig: station callsign should be empty")
	}
	if defLB.Station.Operator != "" {
		t.Error("DefaultConfig: station operator should be empty")
	}
	if defLB.Station.Grid != "" {
		t.Error("DefaultConfig: station grid should be empty")
	}

	// Rigs — should have exactly one entry with hash ID key.
	if len(cfg.Rigs) == 0 {
		t.Fatal("DefaultConfig: no rigs")
	}
	if len(cfg.Rigs) != 1 {
		t.Fatalf("DefaultConfig: expected 1 rig, got %d", len(cfg.Rigs))
	}
	for _, rp := range cfg.Rigs {
		if rp.ID == "" {
			t.Error("DefaultConfig: rig ID should not be empty")
		}
		break
	}

	// WSJT-X
	if cfg.WSJTX.Enabled != false {
		t.Error("DefaultConfig: WSJT-X should be disabled")
	}
	if cfg.WSJTX.UDPHost != "127.0.0.1" {
		t.Errorf("DefaultConfig: WSJT-X host = %q; want 127.0.0.1", cfg.WSJTX.UDPHost)
	}
	if cfg.WSJTX.UDPPort != 2233 {
		t.Errorf("DefaultConfig: WSJT-X port = %d; want 2233", cfg.WSJTX.UDPPort)
	}

	// QRZ
	if cfg.QRZ.Enabled != false {
		t.Error("DefaultConfig: QRZ should be disabled")
	}
}

func TestDefaultConfig_Validate(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("DefaultConfig should validate cleanly: %v", err)
	}
}

// =============================================================================
// Load / Save round-trip tests
// =============================================================================

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.General.Timezone = "Europe/Warsaw"
	cfg.WSJTX.Enabled = true
	cfg.WSJTX.UDPHost = "192.168.0.1"
	cfg.WSJTX.UDPPort = 2238
	cfg.QRZ.User = "testuser"
	cfg.State.ActiveLogbook = "default"
	cfg.Logbooks["default"] = Logbook{
		Description: "Test logbook",
		Station: Station{
			Callsign: "SP9MOA",
			Operator: "Szymon",
			Grid:     "KO00ca",
			RigName:  "myrig",
		},
	}
	cfg.Rigs["myrig"] = RigPreset{
		Model:   "Xiegu G90",
		Antenna: "HWEF",
		Power:   "20",
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify key values survived round-trip
	if loaded.General.Timezone != "Europe/Warsaw" {
		t.Errorf("round-trip timezone: got %q", loaded.General.Timezone)
	}
	if loaded.WSJTX.Enabled != true {
		t.Error("round-trip: WSJT-X not enabled")
	}
	if loaded.WSJTX.UDPHost != "192.168.0.1" {
		t.Errorf("round-trip WSJT-X host: got %q", loaded.WSJTX.UDPHost)
	}
	if loaded.WSJTX.UDPPort != 2238 {
		t.Errorf("round-trip WSJT-X port: got %d", loaded.WSJTX.UDPPort)
	}
	if loaded.QRZ.User != "testuser" {
		t.Errorf("round-trip QRZ user: got %q", loaded.QRZ.User)
	}

	lb := loaded.Logbooks["default"]
	if lb.Station.Callsign != "SP9MOA" {
		t.Errorf("round-trip callsign: got %q", lb.Station.Callsign)
	}
	if lb.Station.Operator != "Szymon" {
		t.Errorf("round-trip operator: got %q", lb.Station.Operator)
	}
	if lb.Station.Grid != "KO00ca" {
		t.Errorf("round-trip grid: got %q", lb.Station.Grid)
	}
	if lb.Station.RigName != "myrig" {
		t.Errorf("round-trip rig_name: got %q", lb.Station.RigName)
	}

	rp, ok := loaded.Rigs["myrig"]
	if !ok {
		t.Fatal("round-trip: rig preset 'myrig' not found")
	}
	if rp.Model != "Xiegu G90" {
		t.Errorf("round-trip rig model: got %q", rp.Model)
	}
	if rp.Antenna != "HWEF" {
		t.Errorf("round-trip rig antenna: got %q", rp.Antenna)
	}
	if rp.Power != "20" {
		t.Errorf("round-trip rig power: got %q", rp.Power)
	}
}

func TestLoad_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")

	_, err := Load(path)
	if err == nil {
		t.Error("Load should return error for nonexistent file")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("this: is: not: valid: [[["), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("Load should return error for malformed YAML")
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	os.WriteFile(path, []byte(""), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load empty file: %v", err)
	}
	// Empty YAML unmarshals to zero-value Config — Validate should catch it.
	if err := cfg.Validate(); err == nil {
		t.Error("empty config should fail Validate (no logbooks, no active logbook)")
	}
}

func TestSaveAndLoad_PreservesAPIKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.Logbooks["default"] = Logbook{
		Description: "test",
		Station:     Station{Callsign: "XX0XX", Operator: "Op", Grid: "JO90"},
		Wavelog: &WavelogConfig{
			Enabled:          true,
			URL:              "https://log.example.com",
			APIKey:           "secret-api-key-12345",
			StationProfileID: "SP-0001",
		},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	wl := loaded.Logbooks["default"].Wavelog
	if wl == nil {
		t.Fatal("Wavelog config lost after round-trip")
	}
	if wl.APIKey != "secret-api-key-12345" {
		t.Errorf("Wavelog API key: got %q, want secret-api-key-12345", wl.APIKey)
	}
	if wl.StationProfileID != "SP-0001" {
		t.Errorf("Wavelog station profile ID: got %q", wl.StationProfileID)
	}
}

// =============================================================================
// Validate tests
// =============================================================================

func TestValidate_NoLogbooks(t *testing.T) {
	cfg := &Config{
		State: StateConfig{ActiveLogbook: "default"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate should fail when no logbooks configured")
	}
}

func TestValidate_NoActiveLogbook(t *testing.T) {
	cfg := &Config{
		Logbooks: map[string]Logbook{"default": {}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate should fail when no active logbook is set")
	}
}

func TestValidate_ActiveLogbookNotFound(t *testing.T) {
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "missing"},
		Logbooks: map[string]Logbook{"default": {}},
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Validate should fail when active logbook is not in logbooks map")
	}
}

// =============================================================================
// Station.Rig* method tests
// =============================================================================

func TestStationRigMethods(t *testing.T) {
	rigs := map[string]RigPreset{
		"g90": {
			Model:        "Xiegu G90",
			Antenna:      "HWEF 20.5",
			Power:        "20",
			FlrigEnabled: true,
			FlrigHost:    "localhost",
			FlrigPort:    "12345",
		},
	}

	t.Run("Rig found", func(t *testing.T) {
		s := Station{RigName: "g90"}
		rp, ok := s.Rig(rigs)
		if !ok {
			t.Fatal("Rig should be found")
		}
		if rp.Model != "Xiegu G90" {
			t.Errorf("model = %q", rp.Model)
		}
	})

	t.Run("Rig not found", func(t *testing.T) {
		s := Station{RigName: "nonexistent"}
		_, ok := s.Rig(rigs)
		if ok {
			t.Error("Rig should not be found")
		}
	})

	t.Run("RigModel empty name", func(t *testing.T) {
		s := Station{RigName: ""}
		if m := s.RigModel(rigs); m != "" {
			t.Errorf("RigModel with empty RigName: got %q, want empty", m)
		}
	})

	t.Run("RigModel found", func(t *testing.T) {
		s := Station{RigName: "g90"}
		if m := s.RigModel(rigs); m != "Xiegu G90" {
			t.Errorf("RigModel: got %q", m)
		}
	})

	t.Run("RigAntenna", func(t *testing.T) {
		s := Station{RigName: "g90"}
		if a := s.RigAntenna(rigs); a != "HWEF 20.5" {
			t.Errorf("RigAntenna: got %q", a)
		}
	})

	t.Run("RigPower", func(t *testing.T) {
		s := Station{RigName: "g90"}
		if p := s.RigPower(rigs); p != "20" {
			t.Errorf("RigPower: got %q", p)
		}
	})

	t.Run("RigFlrig found", func(t *testing.T) {
		s := Station{RigName: "g90"}
		enabled, host, port := s.RigFlrig(rigs)
		if !enabled {
			t.Error("Flrig should be enabled")
		}
		if host != "localhost" {
			t.Errorf("Flrig host: got %q", host)
		}
		if port != "12345" {
			t.Errorf("Flrig port: got %q", port)
		}
	})

	t.Run("RigFlrig defaults when not found", func(t *testing.T) {
		s := Station{RigName: "nonexistent"}
		enabled, host, port := s.RigFlrig(rigs)
		if enabled {
			t.Error("Flrig should default to disabled")
		}
		if host != "localhost" {
			t.Errorf("Flrig host default: got %q", host)
		}
		if port != "12345" {
			t.Errorf("Flrig port default: got %q", port)
		}
	})
}

// =============================================================================
// ResolveLogbook tests
// =============================================================================

func TestResolveLogbook_UsesActiveLogbook(t *testing.T) {
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "home"},
		Logbooks: map[string]Logbook{"home": {Description: "Home QTH"}},
	}
	name, lb, err := ResolveLogbook(cfg, "")
	if err != nil {
		t.Fatalf("ResolveLogbook: %v", err)
	}
	if name != "home" {
		t.Errorf("name = %q; want home", name)
	}
	if lb.Description != "Home QTH" {
		t.Errorf("description = %q", lb.Description)
	}
}

func TestResolveLogbook_CLIFlagOverrides(t *testing.T) {
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "home"},
		Logbooks: map[string]Logbook{"field": {Description: "Field day"}},
	}
	name, _, err := ResolveLogbook(cfg, "field")
	if err != nil {
		t.Fatalf("ResolveLogbook: %v", err)
	}
	if name != "field" {
		t.Errorf("name = %q; want field", name)
	}
}

func TestResolveLogbook_EnvVarOverrides(t *testing.T) {
	t.Setenv("CQOPS_LOGBOOK", "contest")
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "home"},
		Logbooks: map[string]Logbook{"contest": {Description: "Contest"}},
	}
	name, _, err := ResolveLogbook(cfg, "")
	if err != nil {
		t.Fatalf("ResolveLogbook: %v", err)
	}
	if name != "contest" {
		t.Errorf("name = %q; want contest", name)
	}
}

func TestResolveLogbook_CLIFlagWinsOverEnv(t *testing.T) {
	t.Setenv("CQOPS_LOGBOOK", "envlog")
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "home"},
		Logbooks: map[string]Logbook{"cli": {}, "envlog": {}},
	}
	name, _, err := ResolveLogbook(cfg, "cli")
	if err != nil {
		t.Fatalf("ResolveLogbook: %v", err)
	}
	if name != "cli" {
		t.Errorf("name = %q; want cli (CLI flag should win over env)", name)
	}
}

func TestResolveLogbook_FallsBackToDefault(t *testing.T) {
	// Empty state with no logbooks should return an error.
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: ""},
		Logbooks: map[string]Logbook{},
	}
	_, _, err := ResolveLogbook(cfg, "")
	if err == nil {
		t.Error("ResolveLogbook should return error when no active logbook and no logbooks")
	}
}

func TestResolveLogbook_FindsFirstWhenActiveMissing(t *testing.T) {
	cfg := &Config{
		State: StateConfig{ActiveLogbook: "nonexistent"},
		Logbooks: map[string]Logbook{
			"abc123": {ID: "abc123", Station: Station{Callsign: "SP9MOA"}},
		},
	}
	id, lb, err := ResolveLogbook(cfg, "")
	if err != nil {
		t.Fatalf("ResolveLogbook: %v", err)
	}
	if id != "abc123" {
		t.Errorf("id = %q; want abc123", id)
	}
	if lb.Station.Callsign != "SP9MOA" {
		t.Errorf("callsign = %q", lb.Station.Callsign)
	}
}

func TestResolveLogbook_ErrorOnMissingWithEmptyLogbooks(t *testing.T) {
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "newone"},
		Logbooks: map[string]Logbook{},
	}
	_, _, err := ResolveLogbook(cfg, "")
	if err == nil {
		t.Error("ResolveLogbook should return error when active logbook not found and no logbooks")
	}
}

// =============================================================================
// IsFirstRun tests
// =============================================================================

func TestIsFirstRun_TrueForEmptySingleLogbook(t *testing.T) {
	cfg := &Config{
		State:    StateConfig{ActiveLogbook: "abc123"},
		Logbooks: map[string]Logbook{"abc123": {ID: "abc123"}},
	}
	if !IsFirstRun(cfg) {
		t.Error("IsFirstRun should be true for exactly one logbook with empty callsign/operator/grid")
	}
}

func TestIsFirstRun_FalseWhenCallsignSet(t *testing.T) {
	cfg := &Config{
		State: StateConfig{ActiveLogbook: "abc123"},
		Logbooks: map[string]Logbook{
			"abc123": {ID: "abc123", Station: Station{Callsign: "SP9MOA"}},
		},
	}
	if IsFirstRun(cfg) {
		t.Error("IsFirstRun should be false when callsign is set")
	}
}

func TestIsFirstRun_FalseWhenMultipleLogbooks(t *testing.T) {
	cfg := &Config{
		State: StateConfig{ActiveLogbook: "abc123"},
		Logbooks: map[string]Logbook{
			"abc123": {ID: "abc123"},
			"def456": {ID: "def456"},
		},
	}
	if IsFirstRun(cfg) {
		t.Error("IsFirstRun should be false when more than one logbook")
	}
}

func TestIsFirstRun_FalseWhenOperatorSet(t *testing.T) {
	cfg := &Config{
		State: StateConfig{ActiveLogbook: "abc123"},
		Logbooks: map[string]Logbook{
			"abc123": {ID: "abc123", Station: Station{Operator: "Szymon"}},
		},
	}
	if IsFirstRun(cfg) {
		t.Error("IsFirstRun should be false when operator is set")
	}
}

// =============================================================================
// DBPath tests
// =============================================================================

func TestDBPath_UsesExplicitPath(t *testing.T) {
	lb := &Logbook{DatabasePath: "/custom/path/db.sqlite"}
	path, err := DBPath("mylog", lb)
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}
	if path != "/custom/path/db.sqlite" {
		t.Errorf("DBPath = %q; want /custom/path/db.sqlite", path)
	}
}

func TestDBPath_GeneratesDefaultPath(t *testing.T) {
	tmp := isolateHome(t)

	lb := &Logbook{}
	path, err := DBPath("mylog", lb)
	if err != nil {
		t.Fatalf("DBPath: %v", err)
	}

	// Default path should be under the data dir.
	dataDir := expectedDataDir(tmp)
	expected := filepath.Join(dataDir, "mylog.db")
	if path != expected {
		t.Errorf("DBPath = %q; want %q", path, expected)
	}
}

// =============================================================================
// Path helper tests — cross-platform via t.Setenv
// =============================================================================

func TestConfigDir_UsesXdgConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is not used on Windows")
	}
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", "/should-not-be-used")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}

	expected := filepath.Join(tmp, "cqops")
	if dir != expected {
		t.Errorf("ConfigDir = %q; want %q", dir, expected)
	}
}

func TestConfigDir_FallsBackToHome(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}

	expected := expectedConfigDir(tmp, "")
	if dir != expected {
		t.Errorf("ConfigDir = %q; want %q", dir, expected)
	}
}

func TestConfigDir_DoesNotTouchRealHome(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}

	if !filepath.HasPrefix(dir, tmp) {
		t.Errorf("ConfigDir = %q; should be under temp dir %q", dir, tmp)
	}
}

func TestConfigPath_UnderIsolatedConfigDir(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}

	expectedDir := expectedConfigDir(tmp, "")
	expected := filepath.Join(expectedDir, "config.yaml")
	if path != expected {
		t.Errorf("ConfigPath = %q; want %q", path, expected)
	}
	if !filepath.HasPrefix(path, tmp) {
		t.Errorf("ConfigPath = %q; should be under temp dir %q", path, tmp)
	}
}

func TestDataDir_UnderIsolatedHome(t *testing.T) {
	tmp := isolateHome(t)

	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}

	expected := expectedDataDir(tmp)
	if dir != expected {
		t.Errorf("DataDir = %q; want %q", dir, expected)
	}
	if !filepath.HasPrefix(dir, tmp) {
		t.Errorf("DataDir = %q; should be under temp dir %q", dir, tmp)
	}
}

func TestLogDir_UnderIsolatedHome(t *testing.T) {
	tmp := isolateHome(t)

	dir, err := LogDir()
	if err != nil {
		t.Fatalf("LogDir: %v", err)
	}

	expected := expectedLogDir(tmp)
	if dir != expected {
		t.Errorf("LogDir = %q; want %q", dir, expected)
	}
	if !filepath.HasPrefix(dir, tmp) {
		t.Errorf("LogDir = %q; should be under temp dir %q", dir, tmp)
	}
}

// =============================================================================
// EnsureConfig tests — cross-platform isolated
// =============================================================================

func TestEnsureConfig_FirstRunCreatesDefault(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg, configPath, err := EnsureConfig()
	if err != nil {
		t.Fatalf("EnsureConfig: %v", err)
	}

	if !filepath.HasPrefix(configPath, tmp) {
		t.Errorf("configPath = %q; should be under temp dir %q", configPath, tmp)
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should validate: %v", err)
	}
	if cfg.State.ActiveLogbook == "" {
		t.Error("active logbook should not be empty")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file was not created at %q", configPath)
	}
}

func TestEnsureConfig_SecondCallLoadsExisting(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfg1, path1, err := EnsureConfig()
	if err != nil {
		t.Fatalf("first EnsureConfig: %v", err)
	}
	if !filepath.HasPrefix(path1, tmp) {
		t.Errorf("config path should be under temp dir: %q", path1)
	}

	cfg1.General.Timezone = "Europe/London"
	cfg1.WSJTX.UDPPort = 2238
	if err := Save(path1, cfg1); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cfg2, path2, err := EnsureConfig()
	if err != nil {
		t.Fatalf("second EnsureConfig: %v", err)
	}
	if path2 != path1 {
		t.Errorf("second call returned different path: %q vs %q", path2, path1)
	}
	if cfg2.General.Timezone != "Europe/London" {
		t.Errorf("timezone = %q; want Europe/London", cfg2.General.Timezone)
	}
	if cfg2.WSJTX.UDPPort != 2238 {
		t.Errorf("WSJTX port = %d; want 2238", cfg2.WSJTX.UDPPort)
	}
}

func TestEnsureConfig_MalformedConfigReturnsError(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := expectedConfigDir(tmp, "")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")

	os.WriteFile(configPath, []byte("state:\n  active_logbook: missing\nlogbooks: {}\n"), 0644)

	_, _, err := EnsureConfig()
	if err == nil {
		t.Error("EnsureConfig should return error for malformed config")
	}
}

func TestEnsureConfig_CreatesLogbooksMapIfNil(t *testing.T) {
	tmp := isolateHome(t)
	t.Setenv("XDG_CONFIG_HOME", "")

	configDir := expectedConfigDir(tmp, "")
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "config.yaml")
	os.WriteFile(configPath, []byte("state:\n  active_logbook: default\n"), 0644)

	_, _, err := EnsureConfig()
	if err == nil {
		t.Error("EnsureConfig should return error: empty logbooks map fails Validate")
	}
}

// =============================================================================
// Pass 11 — Station profile, config validation edge cases, permissions
// =============================================================================

func TestValidate_ValidCallsign(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
	if err := cfg.Validate(); err != nil {
		t.Errorf("valid callsign should pass: %v", err)
	}
}

func TestValidate_ValidPortableCallsign(t *testing.T) {
	for _, call := range []string{"SP9MOA/P", "SP9MOA/M", "9A/SP9MOA", "DL/SP9MOA/P"} {
		cfg := DefaultConfig()
		lb := cfg.Logbooks[cfg.State.ActiveLogbook]
		lb.Station.Callsign = call
		cfg.Logbooks[cfg.State.ActiveLogbook] = lb
		if err := cfg.Validate(); err != nil {
			t.Errorf("portable callsign %q should pass: %v", call, err)
		}
	}
}

func TestValidate_InvalidCallsign(t *testing.T) {
	for _, call := range []string{"!!!!", "12345", "XX", "A", "SP9 MOA"} {
		cfg := DefaultConfig()
		lb := cfg.Logbooks[cfg.State.ActiveLogbook]
		lb.Station.Callsign = call
		cfg.Logbooks[cfg.State.ActiveLogbook] = lb
		if err := cfg.Validate(); err == nil {
			t.Errorf("invalid callsign %q should fail validation", call)
		}
	}
}

func TestValidate_ValidLocator(t *testing.T) {
	for _, grid := range []string{"JO90", "JO90aa", "FN31pr", "KO00ca"} {
		cfg := DefaultConfig()
		lb := cfg.Logbooks[cfg.State.ActiveLogbook]
		lb.Station.Callsign = "SP9MOA"
		lb.Station.Grid = grid
		cfg.Logbooks[cfg.State.ActiveLogbook] = lb
		if err := cfg.Validate(); err != nil {
			t.Errorf("valid locator %q should pass: %v", grid, err)
		}
	}
}

func TestValidate_InvalidLocator(t *testing.T) {
	for _, grid := range []string{"XXXX", "INVALID", "JO90xxx", "12", "A"} {
		cfg := DefaultConfig()
		lb := cfg.Logbooks[cfg.State.ActiveLogbook]
		lb.Station.Callsign = "SP9MOA"
		lb.Station.Grid = grid
		cfg.Logbooks[cfg.State.ActiveLogbook] = lb
		if err := cfg.Validate(); err == nil {
			t.Errorf("invalid locator %q should fail validation", grid)
		}
	}
}

func TestValidate_EmptyLocatorIsOK(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	lb.Station.Grid = ""
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
	if err := cfg.Validate(); err != nil {
		t.Errorf("empty locator should pass (optional): %v", err)
	}
}

func TestValidate_QRZRequiresUserAndPass(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	cfg.QRZ.Enabled = true
	cfg.QRZ.User = ""
	cfg.QRZ.Pass = "secret"
	if err := cfg.Validate(); err == nil {
		t.Error("QRZ enabled with empty user should fail")
	}

	cfg.QRZ.User = "user"
	cfg.QRZ.Pass = ""
	if err := cfg.Validate(); err == nil {
		t.Error("QRZ enabled with empty pass should fail")
	}

	cfg.QRZ.User = "user"
	cfg.QRZ.Pass = "secret"
	if err := cfg.Validate(); err != nil {
		t.Errorf("QRZ enabled with user+pass should pass: %v", err)
	}
}

func TestValidate_WavelogRequiresHTTPS(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	lb.Wavelog = &WavelogConfig{
		Enabled:          true,
		URL:              "http://log.example.com",
		APIKey:           "test-key",
		StationProfileID: "1",
	}
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
	if err := cfg.Validate(); err == nil {
		t.Error("Wavelog URL without HTTPS should fail")
	}

	lb.Wavelog.URL = "https://log.example.com"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
	if err := cfg.Validate(); err != nil {
		t.Errorf("Wavelog URL with HTTPS should pass: %v", err)
	}
}

func TestValidate_WavelogRequiresAPIKey(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	lb.Wavelog = &WavelogConfig{
		Enabled:          true,
		URL:              "https://log.example.com",
		APIKey:           "",
		StationProfileID: "1",
	}
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb
	if err := cfg.Validate(); err == nil {
		t.Error("Wavelog enabled with empty API key should fail")
	}
}

func TestValidate_WSJTXPortRange(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	cfg.WSJTX.Enabled = true
	cfg.WSJTX.UDPHost = "127.0.0.1"

	cfg.WSJTX.UDPPort = 0
	if err := cfg.Validate(); err == nil {
		t.Error("WSJT-X port 0 should fail")
	}

	cfg.WSJTX.UDPPort = 65536
	if err := cfg.Validate(); err == nil {
		t.Error("WSJT-X port 65536 should fail")
	}

	cfg.WSJTX.UDPPort = 2233
	if err := cfg.Validate(); err != nil {
		t.Errorf("WSJT-X port 2233 should pass: %v", err)
	}
}

func TestValidate_DistanceUnitEnum(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	for _, unit := range []string{"km", "mi", ""} {
		cfg.General.DistanceUnit = unit
		if err := cfg.Validate(); err != nil {
			t.Errorf("distance_unit %q should pass: %v", unit, err)
		}
	}

	cfg.General.DistanceUnit = "feet"
	if err := cfg.Validate(); err == nil {
		t.Error("distance_unit 'feet' should fail")
	}
}

// =============================================================================
// Config file permissions and save/load roundtrip
// =============================================================================

func TestSave_PermissionsAre0600(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %04o, want 0600", perm)
	}
}

func TestSaveAndLoad_StationFieldsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := DefaultConfig()
	cfg.General.Timezone = "Europe/Warsaw"
	cfg.General.DistanceUnit = "mi"
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	lb.Station.Operator = "Szymon"
	lb.Station.Grid = "KO00ca"
	lb.Station.SOTARef = "SP/TA-001"
	lb.Station.POTARef = "SP-0001"
	lb.Station.WWFFRef = "SPFF-0001"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	lb2 := loaded.Logbooks[loaded.State.ActiveLogbook]
	if lb2.Station.Callsign != "SP9MOA" {
		t.Errorf("Callsign = %q", lb2.Station.Callsign)
	}
	if lb2.Station.Operator != "Szymon" {
		t.Errorf("Operator = %q", lb2.Station.Operator)
	}
	if lb2.Station.Grid != "KO00ca" {
		t.Errorf("Grid = %q", lb2.Station.Grid)
	}
	if lb2.Station.SOTARef != "SP/TA-001" {
		t.Errorf("SOTARef = %q", lb2.Station.SOTARef)
	}
	if lb2.Station.POTARef != "SP-0001" {
		t.Errorf("POTARef = %q", lb2.Station.POTARef)
	}
	if loaded.General.Timezone != "Europe/Warsaw" {
		t.Errorf("Timezone = %q", loaded.General.Timezone)
	}
	if loaded.General.DistanceUnit != "mi" {
		t.Errorf("DistanceUnit = %q", loaded.General.DistanceUnit)
	}
}

func TestValidate_DXCPort(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	cfg.DXC.Enabled = true
	cfg.DXC.Host = "dxspider.co.uk"

	cfg.DXC.Port = "abc"
	if err := cfg.Validate(); err == nil {
		t.Error("DXC port 'abc' should fail")
	}
	cfg.DXC.Port = "0"
	if err := cfg.Validate(); err == nil {
		t.Error("DXC port 0 should fail")
	}
	cfg.DXC.Port = "7300"
	if err := cfg.Validate(); err != nil {
		t.Errorf("DXC port 7300 should pass: %v", err)
	}
}

func TestValidate_RigFlrigPort(t *testing.T) {
	cfg := DefaultConfig()
	lb := cfg.Logbooks[cfg.State.ActiveLogbook]
	lb.Station.Callsign = "SP9MOA"
	cfg.Logbooks[cfg.State.ActiveLogbook] = lb

	cfg.Rigs["test-rig"] = RigPreset{
		ID:           "test-rig",
		Model:        "FT-891",
		FlrigEnabled: true,
		FlrigHost:    "localhost",
		FlrigPort:    "abc",
	}
	if err := cfg.Validate(); err == nil {
		t.Error("flrig port 'abc' should fail")
	}

	cfg.Rigs["test-rig"] = RigPreset{
		ID:           "test-rig",
		Model:        "FT-891",
		FlrigEnabled: true,
		FlrigHost:    "localhost",
		FlrigPort:    "12345",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("flrig port 12345 should pass: %v", err)
	}
}
