package tui

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/gps"
)

func TestTruncateGrid(t *testing.T) {
	tests := []struct {
		grid string
		prec int
		want string
	}{
		{"JO90aa11xx", 10, "JO90aa11xx"},
		{"JO90aa11xx", 8, "JO90aa11"},
		{"JO90aa11xx", 6, "JO90aa"},
		{"JO90", 6, "JO90"},
		{"JO90aa", 10, "JO90aa"},
	}
	for _, tt := range tests {
		if got := truncateGrid(tt.grid, tt.prec); got != tt.want {
			t.Errorf("truncateGrid(%q,%d) = %q, want %q", tt.grid, tt.prec, got, tt.want)
		}
	}
}

func TestGPSLogPrecision(t *testing.T) {
	// No app — default full precision.
	if got := (&Model{}).gpsLogPrecision(); got != 10 {
		t.Errorf("default precision = %d, want 10", got)
	}

	m := &Model{App: &app.App{Config: &config.Config{
		Integrations: config.IntegrationsConfig{GPS: config.GPSConfig{GridPrecision: 6}},
	}}}
	if got := m.gpsLogPrecision(); got != 6 {
		t.Errorf("configured precision = %d, want 6", got)
	}

	m.App.Config.Integrations.GPS.GridPrecision = 8
	if got := m.gpsLogPrecision(); got != 8 {
		t.Errorf("configured precision = %d, want 8", got)
	}

	// Invalid values fall back to full precision.
	m.App.Config.Integrations.GPS.GridPrecision = 4
	if got := m.gpsLogPrecision(); got != 10 {
		t.Errorf("invalid precision = %d, want 10 fallback", got)
	}
}

// scriptNMEAReader feeds pre-scripted NMEA lines to a real gps.Client for
// integration-style TUI tests. In block mode it waits for more lines instead
// of returning an error when the script is exhausted.
type scriptNMEAReader struct {
	mu     sync.Mutex
	lines  []string
	pos    int
	closed bool
	block  bool
}

func (r *scriptNMEAReader) ReadLine() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.pos >= len(r.lines) {
		if !r.block || r.closed {
			return "", fmt.Errorf("script exhausted")
		}
		r.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		r.mu.Lock()
	}
	line := r.lines[r.pos]
	r.pos++
	return line, nil
}

func (r *scriptNMEAReader) addLine(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lines = append(r.lines, s)
}

func (r *scriptNMEAReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *scriptNMEAReader) TryOpen() error { return nil }

// TestHandleGPSTickExpiresStaleFix verifies the freshness enforcement: a
// position whose last valid update is older than the TTL is treated as a
// lost fix even when no explicit void sentence ever arrived.
func TestHandleGPSTickExpiresStaleFix(t *testing.T) {
	orig := gpsFixTTL
	gpsFixTTL = 100 * time.Millisecond
	t.Cleanup(func() { gpsFixTTL = orig })

	r := &scriptNMEAReader{
		lines: []string{"$GPGGA,120000.000,5003.0000,N,01956.4000,E,1,12,1.0,201.3,M,42.0,M,,*45"},
		block: true,
	}

	// Create the model BEFORE starting the client goroutine: New() toggles
	// applog debug mode, which must not race the reader's log calls.
	m := newLifecycleTestModel(t)
	m.gps.client = gps.NewClient(r)
	m.gps.client.Start()
	defer m.gps.client.Stop()

	// Let the client parse the valid sentence, then tick.
	time.Sleep(30 * time.Millisecond)
	m.handleGPSTick()
	if !m.gps.hasFix {
		t.Fatal("expected a valid fix while fresh")
	}

	// No further sentences arrive — the fix ages out and must expire.
	time.Sleep(150 * time.Millisecond)
	m.handleGPSTick()
	if m.gps.hasFix {
		t.Fatal("stale fix must expire even without an explicit void sentence")
	}
}

// TestHandleGPSTickPropagatesMovementToEffectiveGrid verifies every position
// change reaches the app (APRS/effective grid) and that the effective grid
// follows movement — while the CONFIGURED station grid is never mutated.
// The effective grid derives the GPS value from live state, so an override
// can no longer write into (or leak across) logbook configuration.
func TestHandleGPSTickPropagatesMovementToEffectiveGrid(t *testing.T) {
	r := &scriptNMEAReader{
		lines: []string{"$GPGGA,120000.000,5003.0000,N,01956.4000,E,1,12,1.0,201.3,M,42.0,M,,*45"},
		block: true,
	}

	// Create the model BEFORE starting the client goroutine (see the
	// staleness test for why the order matters).
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.GPS.Enabled = true
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Logbook.Station.GPSGrid = true
	m.gps.client = gps.NewClient(r)
	m.gps.client.Start()
	defer m.gps.client.Stop()

	// First fix: effective grid is the GPS grid, the configured fallback
	// stays untouched.
	time.Sleep(30 * time.Millisecond)
	m.handleGPSTick()
	if !m.gps.hasFix || m.gps.lastGrid == "" {
		t.Fatalf("expected a fix, got hasFix=%v grid=%q", m.gps.hasFix, m.gps.lastGrid)
	}
	grid1 := m.gps.lastGrid
	if got := m.effectiveGrid(); got != grid1 {
		t.Errorf("effective grid = %q, want GPS grid %q", got, grid1)
	}
	if m.App.Logbook.Station.Grid != "JO90" {
		t.Errorf("configured station grid was mutated: %q, want JO90", m.App.Logbook.Station.Grid)
	}

	// The receiver moves to a different grid — the effective grid must
	// follow without replacing the configured fallback.
	r.addLine("$GPGGA,120001.000,5231.2000,N,01324.0000,E,1,12,1.0,201.3,M,42.0,M,,*45")
	time.Sleep(30 * time.Millisecond)
	m.handleGPSTick()
	if m.gps.lastGrid == grid1 {
		t.Fatal("grid did not move despite the new position")
	}
	if got := m.effectiveGrid(); got != m.gps.lastGrid {
		t.Errorf("effective grid did not follow movement: %q, want %q", got, m.gps.lastGrid)
	}
	if m.App.Logbook.Station.Grid != "JO90" {
		t.Errorf("configured station grid was mutated after movement: %q, want JO90", m.App.Logbook.Station.Grid)
	}
}

// TestGPSGridOverrideDoesNotCrossLogbookBoundary reproduces the reported
// bug: enabling the GPS grid in logbook A and switching to B used to leave
// A's saved fallback pending, so losing the fix wrote A's grid into B's
// active station state. The effective grid must never mutate configured
// station state, so each logbook keeps its own configured grid.
func TestGPSGridOverrideDoesNotCrossLogbookBoundary(t *testing.T) {
	orig := gpsFixTTL
	gpsFixTTL = 100 * time.Millisecond
	t.Cleanup(func() { gpsFixTTL = orig })

	r := &scriptNMEAReader{
		lines: []string{"$GPGGA,120000.000,5003.0000,N,01956.4000,E,1,12,1.0,201.3,M,42.0,M,,*45"},
		block: true,
	}

	// Logbook A: GPS grid enabled, configured fallback JO90.
	m := newLifecycleTestModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(m.App.StopAPRSTimer)
	m.App.Config.Integrations.GPS.Enabled = true
	m.App.Logbook.Station.Grid = "JO90"
	m.App.Logbook.Station.GPSGrid = true
	m.gps.client = gps.NewClient(r)
	m.gps.client.Start()
	defer m.gps.client.Stop()

	time.Sleep(30 * time.Millisecond)
	m.handleGPSTick()
	if !m.gps.hasFix {
		t.Fatal("expected a GPS fix on logbook A")
	}
	if m.effectiveGrid() == "JO90" {
		t.Fatal("A's effective grid should be GPS-derived, not the fallback")
	}

	// Switch to logbook B: its own configured grid, GPS grid disabled.
	lbB := config.Logbook{
		Station:      config.Station{Callsign: "SP9B", Grid: "KO00"},
		DatabasePath: filepath.Join(t.TempDir(), "b.db"),
	}
	m.App.Config.Logbooks["b"] = lbB
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { m.App.DB.Close() })

	// B has GPS grid disabled — its configured grid wins immediately.
	if got := m.effectiveGrid(); got != "KO00" {
		t.Errorf("B's effective grid = %q, want its configured KO00", got)
	}

	// The fix ages out while on B — B's configured grid must survive.
	time.Sleep(150 * time.Millisecond)
	m.handleGPSTick()
	if m.gps.hasFix {
		t.Fatal("fix should have aged out")
	}
	if m.App.Logbook.Station.Grid != "KO00" {
		t.Errorf("B's configured grid was replaced: %q, want KO00", m.App.Logbook.Station.Grid)
	}
	if got := m.effectiveGrid(); got != "KO00" {
		t.Errorf("B's effective grid after fix loss = %q, want KO00", got)
	}
	// A's configured fallback is also intact in the config map.
	if got := m.App.Config.Logbooks["test"].Station.Grid; got != "JO90" {
		t.Errorf("A's configured grid was mutated: %q, want JO90", got)
	}
}
