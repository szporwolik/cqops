package tui

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/dashboard"
)

// =============================================================================
// Dashboard station event — the two-second push broadcast storm
//
// With APRS enabled and a resolvable position, every dashboard push built a
// fresh *GeoPoint for the station, and SetStation compared the station structs
// with `!=` — which compares that POINTER. Identical coordinates therefore
// counted as a change: an unchanged Station event every two seconds, and the
// browser rebuilt its map layers on each one.
// =============================================================================

// stationEventCounter subscribes to the dashboard hub and counts only Station
// events, ignoring the other traffic a push produces (display, app, APRS
// stations, today/recent).
type stationEventCounter struct {
	ch chan dashboard.Event
}

func countStationEvents(t *testing.T, m *Model) *stationEventCounter {
	t.Helper()
	ch := m.http.client.Hub().Subscribe()
	if ch == nil {
		t.Fatal("dashboard hub is full; cannot subscribe")
	}
	t.Cleanup(func() { m.http.client.Hub().Unsubscribe(ch) })
	return &stationEventCounter{ch: ch}
}

// drain returns how many Station events arrived within a short window.
func (c *stationEventCounter) drain() int {
	n := 0
	deadline := time.After(60 * time.Millisecond)
	for {
		select {
		case ev := <-c.ch:
			if ev.Type == string(dashboard.EventStation) {
				n++
			}
		case <-deadline:
			return n
		}
	}
}

// TestDashboardPushPublishesStationOnce reproduces the report: APRS enabled, a
// valid position, two pushes with identical inputs — one Station event, not two.
func TestDashboardPushPublishesStationOnce(t *testing.T) {
	m := dashboardCycleModel(t)
	m.App.Logbook.Station.Grid = "JO90WE"
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 25}

	counter := countStationEvents(t, m)
	// Every push goes through the 2-tick throttle.
	push := func() {
		m.tickCount += 2
		m.pushDashboardState()
	}

	push()
	if got := counter.drain(); got != 1 {
		t.Fatalf("first push published %d station events, want 1", got)
	}

	push()
	if got := counter.drain(); got != 0 {
		t.Errorf("an unchanged push published %d station events, want 0 — every two seconds the browser would rebuild its map layers", got)
	}

	// A real change still has to go out.
	m.App.Logbook.Station.Grid = "KO00CA"
	push()
	if got := counter.drain(); got != 1 {
		t.Errorf("a changed station grid published %d station events, want 1", got)
	}
}

// TestDashboardPushPublishesStationOnPositionChange keeps the other half of the
// contract honest: a moving position must still reach the browser, and its
// absence must be published too (APRS disabled, then re-enabled).
func TestDashboardPushPublishesStationOnPositionChange(t *testing.T) {
	m := dashboardCycleModel(t)
	m.App.Logbook.Station.Grid = "JO90WE"
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 25}

	counter := countStationEvents(t, m)
	push := func() {
		m.tickCount += 2
		m.pushDashboardState()
	}

	push()
	if got := counter.drain(); got != 1 {
		t.Fatalf("first push published %d station events, want 1", got)
	}

	// A pinned APRS position moves: same grid, different coordinates.
	m.App.Logbook.APRS.FixedLocation = "KO00CA"
	push()
	if got := counter.drain(); got != 1 {
		t.Errorf("a moved APRS position published %d station events, want 1", got)
	}

	// Position removed (APRS off): one more event, then silence.
	m.App.Logbook.APRS = nil
	push()
	if got := counter.drain(); got != 1 {
		t.Errorf("losing the APRS position published %d station events, want 1", got)
	}
	push()
	if got := counter.drain(); got != 0 {
		t.Errorf("a repeated push without a position published %d station events, want 0", got)
	}
}
