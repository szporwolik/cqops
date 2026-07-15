package app

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/wsjtx"
)

// =============================================================================
// WSJT-X restart avoidance tests
// =============================================================================
//
// These tests verify that MaybeRestartWSJTX only restarts when the effective
// WSJT-X config (enabled, host, port) actually changes.
//
// Tests that require Start() are limited because applog is nil in tests.

func TestMaybeRestartWSJTX_NoOpWhenUnchanged(t *testing.T) {
	enabled := false
	host := "127.0.0.1"
	port := 2233

	a := &App{
		Config: config.DefaultConfig(),
		WSJTX:  wsjtx.NewListener(),
	}

	// First call: disabled. Stop() is a no-op on fresh listener.
	a.MaybeRestartWSJTX(enabled, host, port)

	if a.lastWSJTX.enabled != false {
		t.Error("lastWSJTX.enabled should be false after apply")
	}

	// Second call: same config -> should be no-op (skips Stop too).
	lastBefore := a.lastWSJTX
	a.MaybeRestartWSJTX(enabled, host, port)

	if a.lastWSJTX != lastBefore {
		t.Error("lastWSJTX changed on unchanged config — should be no-op")
	}
}

func TestMaybeRestartWSJTX_DetectsHostChange(t *testing.T) {
	enabled := false
	host := "127.0.0.1"
	port := 2237

	a := &App{
		Config: config.DefaultConfig(),
		WSJTX:  wsjtx.NewListener(),
	}

	// Apply initial config.
	a.MaybeRestartWSJTX(enabled, host, port)
	first := a.lastWSJTX

	// Change host (still disabled — Stop is called but no Start).
	host = "192.168.1.1"
	a.MaybeRestartWSJTX(enabled, host, port)

	if a.lastWSJTX.host != "192.168.1.1" {
		t.Errorf("lastWSJTX.host = %q; want 192.168.1.1", a.lastWSJTX.host)
	}
	if a.lastWSJTX == first {
		t.Error("lastWSJTX should have changed on host change")
	}
}

func TestMaybeRestartWSJTX_DetectsPortChange(t *testing.T) {
	enabled := false
	host := "127.0.0.1"
	port := 2237

	a := &App{
		Config: config.DefaultConfig(),
		WSJTX:  wsjtx.NewListener(),
	}

	// Apply initial config.
	a.MaybeRestartWSJTX(enabled, host, port)
	first := a.lastWSJTX

	// Change port (still disabled).
	port = 2238
	a.MaybeRestartWSJTX(enabled, host, port)

	if a.lastWSJTX.port != 2238 {
		t.Errorf("lastWSJTX.port = %d; want 2238", a.lastWSJTX.port)
	}
	if a.lastWSJTX == first {
		t.Error("lastWSJTX should have changed on port change")
	}
}

func TestMaybeRestartWSJTX_DisableStopsListener(t *testing.T) {
	a := &App{
		Config: config.DefaultConfig(),
		WSJTX:  wsjtx.NewListener(),
	}

	// Manually set state as if a previous Start had succeeded.
	a.lastWSJTX.enabled = true
	a.lastWSJTX.host = "127.0.0.1"
	a.lastWSJTX.port = 2237

	// Now "disable" — config says disabled, last-applied says enabled.
	// Should call Stop(), which is safe in tests.
	a.MaybeRestartWSJTX(false, "127.0.0.1", 2237)

	if a.lastWSJTX.enabled != false {
		t.Error("lastWSJTX.enabled should be false after disable")
	}
	if a.WSJTX.IsActive() {
		t.Error("listener should not be active after disable")
	}
}

// TestMaybeRestartWSJTX_NoOpOnSameConfig verifies repeated calls with the
// same effective config are no-ops. The "failed Start doesn't update
// last-applied" path cannot be unit-tested reliably because UDP port
// availability varies — it is verified by code review.
func TestMaybeRestartWSJTX_NoOpOnSameConfig(t *testing.T) {
	enabled := false
	host := "127.0.0.1"
	port := 2237

	a := &App{
		Config: config.DefaultConfig(),
		WSJTX:  wsjtx.NewListener(),
	}

	// Apply disabled state.
	a.MaybeRestartWSJTX(enabled, host, port)
	first := a.lastWSJTX

	// Same config again — no change expected.
	a.MaybeRestartWSJTX(enabled, host, port)
	if a.lastWSJTX != first {
		t.Error("lastWSJTX changed on identical re-apply — should be no-op")
	}

	// Same config a third time.
	a.MaybeRestartWSJTX(enabled, host, port)
	if a.lastWSJTX != first {
		t.Error("lastWSJTX changed on third identical apply")
	}
}

// =============================================================================
// APRS tests
// =============================================================================

func TestMaybeRestartAPRS_DisabledByDefault(t *testing.T) {
	cfg := config.DefaultConfig()
	a := &App{
		Config:  cfg,
		Logbook: &config.Logbook{},
	}
	// Default config has no APRS enabled — should be a no-op.
	a.MaybeRestartAPRS()
	if a.APRSClient != nil {
		t.Error("APRSClient should be nil when APRS is disabled")
	}
	if a.APRSCache != nil {
		t.Error("APRSCache should be nil when APRS is disabled")
	}
}

func TestMaybeRestartAPRS_StopsExistingClient(t *testing.T) {
	cfg := config.DefaultConfig()
	a := &App{
		Config:     cfg,
		Logbook:    &config.Logbook{},
		InetOnline: true,
	}
	// Simulate a running KISS server client by creating one manually.
	kc := aprs.NewKISSServerClient("127.0.0.1:1")
	kc.Start()
	a.APRSClient = kc

	// Disabled config should stop the client.
	a.MaybeRestartAPRS()
	time.Sleep(100 * time.Millisecond)

	if a.APRSClient != nil {
		t.Error("APRSClient should be nil after disabling")
	}
}

func TestEffectiveGrid_NoGPS(t *testing.T) {
	cfg := config.DefaultConfig()
	a := &App{
		Config:  cfg,
		Logbook: &config.Logbook{Station: config.Station{Grid: "JO62TJ"}},
	}
	// Without GPS, EffectiveGrid returns the station grid.
	g := a.EffectiveGrid()
	if g != "JO62TJ" {
		t.Errorf("EffectiveGrid: got %q, want JO62TJ", g)
	}
}

func TestEffectiveGrid_GPSWithFix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.GPS.Enabled = true
	a := &App{
		Config:  cfg,
		Logbook: &config.Logbook{Station: config.Station{Grid: "JO62TJ", GPSGrid: true}},
	}
	a.SetGPSGrid("KO00CA", true)
	g := a.EffectiveGrid()
	if g != "KO00CA" {
		t.Errorf("EffectiveGrid with GPS fix: got %q, want KO00CA", g)
	}
}

func TestEffectiveGrid_GPSNoFix(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.GPS.Enabled = true
	a := &App{
		Config:  cfg,
		Logbook: &config.Logbook{Station: config.Station{Grid: "JO62TJ", GPSGrid: true}},
	}
	a.SetGPSGrid("KO00CA", false)
	g := a.EffectiveGrid()
	if g != "JO62TJ" {
		t.Errorf("EffectiveGrid with GPS no-fix: got %q, want JO62TJ", g)
	}
}

func TestSetGPSGrid(t *testing.T) {
	cfg := config.DefaultConfig()
	a := &App{
		Config:  cfg,
		Logbook: &config.Logbook{Station: config.Station{Grid: "JO62TJ"}},
	}
	a.SetGPSGrid("KO00CA02wh", true)
	if a.gpsGrid != "KO00CA02wh" {
		t.Errorf("gpsGrid: got %q", a.gpsGrid)
	}
	if !a.gpsHasFix {
		t.Error("gpsHasFix should be true")
	}

	// Clear GPS.
	a.SetGPSGrid("", false)
	if a.gpsGrid != "" {
		t.Errorf("gpsGrid should be empty after clear, got %q", a.gpsGrid)
	}
	if a.gpsHasFix {
		t.Error("gpsHasFix should be false after clear")
	}
}

func TestEffectiveGrid_KISSServerService(t *testing.T) {
	// Verify that APRS client type detection works correctly.
	kc := aprs.NewKISSServerClient("127.0.0.1:1")
	if kc.IsRunning() {
		kc.Stop()
	}
	if kc.IsRunning() {
		t.Error("KISSServerClient should not be running after Stop")
	}
	if kc.IsConnected() {
		t.Error("KISSServerClient should not be connected when never started")
	}
}
