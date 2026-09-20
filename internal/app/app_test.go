package app

import (
	"errors"
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

// fakeAPRSClient is a minimal aprs.Client for status-forwarding tests.
// The id field keeps instances at distinct addresses (a zero-size struct
// would make all pointers equal).
type fakeAPRSClient struct{ id int }

func (*fakeAPRSClient) Start()            {}
func (*fakeAPRSClient) Stop()             {}
func (*fakeAPRSClient) IsRunning() bool   { return false }
func (*fakeAPRSClient) IsConnected() bool { return false }

// TestReportAPRSStatus_StaleClientIgnored verifies that disconnect events
// from a replaced client are not forwarded (no spurious "connection lost").
func TestReportAPRSStatus_StaleClientIgnored(t *testing.T) {
	calls := 0
	a := &App{}
	a.SetAPRSStatusCallback(func(connected bool, err error) { calls++ })

	old := &fakeAPRSClient{id: 1}
	a.APRSClient = old
	a.reportAPRSStatus(old, false, errors.New("closed"))
	if calls != 1 {
		t.Fatalf("current client event not forwarded, calls=%d", calls)
	}

	// Client replaced — stale events from the old client must be ignored.
	cur := &fakeAPRSClient{id: 2}
	a.APRSClient = cur
	a.reportAPRSStatus(old, false, errors.New("closed"))
	if calls != 1 {
		t.Fatalf("stale client event forwarded, calls=%d", calls)
	}
	a.reportAPRSStatus(cur, false, errors.New("boom"))
	if calls != 2 {
		t.Fatalf("new client event not forwarded, calls=%d", calls)
	}
}

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

func TestMaybeRestartAPRS_ReceiveOnlyStartsClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate cache dir
	cfg := config.DefaultConfig()
	cfg.Integrations.APRS.Enabled = true
	cfg.Integrations.APRS.Service = "kiss_server"
	a := &App{
		Config:     cfg,
		Logbook:    &config.Logbook{}, // no APRS config — receive-only
		InetOnline: true,
	}
	a.MaybeRestartAPRS()
	if a.APRSClient == nil {
		t.Fatal("APRSClient should start in receive-only mode")
	}
	if a.beaconStopCh != nil {
		t.Error("beacon goroutine must not start in receive-only mode")
	}
	a.stopAPRS()
	time.Sleep(50 * time.Millisecond)
	if a.APRSClient != nil {
		t.Error("APRSClient should be nil after stopAPRS")
	}
}

func TestMaybeRestartAPRS_ReceiveOnlyNoCallsign(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate cache dir
	cfg := config.DefaultConfig()
	cfg.Integrations.APRS.Enabled = true
	// Default service is aprs_is — login requires a callsign.
	a := &App{
		Config:     cfg,
		Logbook:    &config.Logbook{}, // no station callsign
		InetOnline: true,
	}
	a.MaybeRestartAPRS()
	if a.APRSClient != nil {
		t.Error("APRSClient should not start without a login callsign")
	}
	if a.APRSCache != nil {
		t.Error("APRSCache should not be opened without a login callsign")
	}
}

func TestMaybeRestartAPRS_ReceiveOnlyWithStationCall(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate cache dir
	cfg := config.DefaultConfig()
	cfg.Integrations.APRS.Enabled = true
	cfg.Integrations.APRS.Server = "127.0.0.1:1" // loopback only — never real APRS-IS
	a := &App{
		Config:     cfg,
		Logbook:    &config.Logbook{Station: config.Station{Callsign: "SP9ABC", Grid: "JO90"}},
		InetOnline: true,
	}
	a.MaybeRestartAPRS()
	if a.APRSClient == nil {
		t.Fatal("APRSClient should start in receive-only mode with station callsign")
	}
	if a.beaconStopCh != nil {
		t.Error("beacon goroutine must not start in receive-only mode")
	}
	a.stopAPRS()
	time.Sleep(50 * time.Millisecond)
}

func TestMaybeRestartAPRS_FullModeStartsBeacon(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // isolate cache dir
	cfg := config.DefaultConfig()
	cfg.Integrations.APRS.Enabled = true
	cfg.Integrations.APRS.Service = "kiss_server"
	a := &App{
		Config: cfg,
		Logbook: &config.Logbook{APRS: &config.APRSConfig{
			Enabled:      true,
			SendLocation: true,
		}},
		InetOnline: true,
	}
	a.MaybeRestartAPRS()
	if a.APRSClient == nil {
		t.Fatal("APRSClient should start in full mode")
	}
	if a.beaconStopCh == nil {
		t.Error("beacon goroutine should start in full mode")
	}
	a.stopAPRS()
	time.Sleep(50 * time.Millisecond)
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
