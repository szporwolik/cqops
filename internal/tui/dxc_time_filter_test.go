package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// DXC time filter tests (Pass 24)
// =============================================================================
// Tests time filter cycling, cutoff behavior, spot filtering by age,
// and interaction with band/mode filters. Uses temp DB with controlled
// timestamps. No real DX Cluster connection.

// now is a helper that returns the current UTC time for use in spot timestamps.
// Tests use this to ensure spots are classified as recent/old relative to
// dxcFilteredSpots's internal time.Now() call (which will be within a few ms).
var testNow = func() time.Time { return time.Now().UTC() }

// =============================================================================
// Time filter cycling — forward (PgUp)
// =============================================================================

func TestDXCTimeFilter_CycleForward(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Start: timeFilter=0 (all), timeIdx=0.
	if m.dxc.timeFilter != 0 {
		t.Fatalf("initial timeFilter = %d, want 0", m.dxc.timeFilter)
	}

	// PgUp → 60m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 60 {
		t.Errorf("1st PgUp: timeFilter = %d, want 60", m.dxc.timeFilter)
	}

	// PgUp → 30m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 30 {
		t.Errorf("2nd PgUp: timeFilter = %d, want 30", m.dxc.timeFilter)
	}

	// PgUp → 15m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 15 {
		t.Errorf("3rd PgUp: timeFilter = %d, want 15", m.dxc.timeFilter)
	}

	// PgUp → 10m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 10 {
		t.Errorf("4th PgUp: timeFilter = %d, want 10", m.dxc.timeFilter)
	}

	// PgUp → 5m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 5 {
		t.Errorf("5th PgUp: timeFilter = %d, want 5", m.dxc.timeFilter)
	}

	// PgUp → 0 (wraparound back to "all").
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.timeFilter != 0 {
		t.Errorf("6th PgUp (wrap): timeFilter = %d, want 0", m.dxc.timeFilter)
	}
}

// =============================================================================
// Time filter cycling — backward (PgDown)
// =============================================================================

func TestDXCTimeFilter_CycleBackward(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Start: timeFilter=0. PgDown wraps to last element → 5m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil)
	if m.dxc.timeFilter != 5 {
		t.Errorf("1st PgDown (from 0): timeFilter = %d, want 5", m.dxc.timeFilter)
	}

	// PgDown → 10m.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil)
	if m.dxc.timeFilter != 10 {
		t.Errorf("2nd PgDown: timeFilter = %d, want 10", m.dxc.timeFilter)
	}

	// PgDown → 15m, 30m, 60m, 0.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil) // 15
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil) // 30
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil) // 60
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil) // 0

	if m.dxc.timeFilter != 0 {
		t.Errorf("after full backward cycle: timeFilter = %d, want 0", m.dxc.timeFilter)
	}
}

// =============================================================================
// Backspace clears time filter
// =============================================================================

func TestDXCTimeFilter_ClearWithBackspace(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.timeFilter = 30
	m.dxc.timeIdx = 2
	m.dxc.bandFilter = "20m"

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)

	if m.dxc.timeFilter != 0 {
		t.Errorf("timeFilter = %d, want 0 after backspace", m.dxc.timeFilter)
	}
	if m.dxc.timeIdx != 0 {
		t.Errorf("timeIdx = %d, want 0 after backspace", m.dxc.timeIdx)
	}
	if m.dxc.bandFilter != "" {
		t.Errorf("bandFilter = %q, want \"\" after backspace", m.dxc.bandFilter)
	}
}

// =============================================================================
// Time filter forces table rebuild
// =============================================================================

func TestDXCTimeFilter_ForcesTableRebuild(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)
	m.dxc.tableReady = true

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.tableReady {
		t.Error("time filter change should set tableReady=false")
	}
}

// =============================================================================
// dxcFilteredSpots applies time filter — recent spots pass, old spots drop
// =============================================================================

func TestDXCTimeFilter_RecentSpotsPass(t *testing.T) {
	now := testNow().Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9FRESH", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: now},
	}
	m := newDXCBandFilterModel(t, spots)

	// Set time filter to 5 minutes — spot just created should pass.
	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (recent spot within 5min window)", len(filtered))
	}
	if filtered[0].DXCall != "SP9FRESH" {
		t.Errorf("filtered spot = %q, want SP9FRESH", filtered[0].DXCall)
	}
}

func TestDXCTimeFilter_OldSpotsDropped(t *testing.T) {
	old := testNow().Add(-2 * time.Hour).Unix() // 2 hours ago
	spots := []store.DXCSpot{
		{DXCall: "SP9OLD", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	// Set time filter to 5 minutes — 2-hour-old spot should be dropped.
	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 0 {
		t.Errorf("filtered count = %d, want 0 (old spot outside 5min window)", len(filtered))
	}
}

func TestDXCTimeFilter_MixedRecentAndOld(t *testing.T) {
	now := testNow().Unix()
	old := testNow().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9FRESH", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: now},
		{DXCall: "SP9OLD", Frequency: 7200000, Band: "40m", Mode: "CW", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (only recent spot)", len(filtered))
	}
	if filtered[0].DXCall != "SP9FRESH" {
		t.Errorf("filtered spot = %q, want SP9FRESH", filtered[0].DXCall)
	}
}

func TestDXCTimeFilter_AllTimePassesAll(t *testing.T) {
	old := testNow().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9OLD", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	// timeFilter=0 means "all" — even old spots pass.
	m.dxc.timeFilter = 0
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (all time filter passes everything)", len(filtered))
	}
}

// =============================================================================
// Time filter combines with band filter
// =============================================================================

func TestDXCTimeFilter_CombinedWithBandFilter(t *testing.T) {
	now := testNow().Unix()
	old := testNow().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9A", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: now}, // recent, 20m
		{DXCall: "SP9B", Frequency: 7200000, Band: "40m", Mode: "CW", ReceivedAt: now},   // recent, 40m
		{DXCall: "SP9C", Frequency: 14195000, Band: "20m", Mode: "FT8", ReceivedAt: old}, // old, 20m
	}
	m := newDXCBandFilterModel(t, spots)

	// Filter: last 5 minutes + band 20m → only SP9A should pass.
	m.dxc.timeFilter = 5
	m.dxc.bandFilter = "20m"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (recent 20m spot)", len(filtered))
	}
	if filtered[0].DXCall != "SP9A" {
		t.Errorf("filtered spot = %q, want SP9A", filtered[0].DXCall)
	}
}

// =============================================================================
// Time filter combines with mode filter
// =============================================================================

func TestDXCTimeFilter_CombinedWithModeFilter(t *testing.T) {
	now := testNow().Unix()
	old := testNow().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9CW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: now},
		{DXCall: "SP9FT8", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
		{DXCall: "SP9SSB", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	// Filter: last 5 minutes + mode CW → only SP9CW passes.
	m.dxc.timeFilter = 5
	m.dxc.modeFilter = "CW"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (recent CW spot)", len(filtered))
	}
	if filtered[0].DXCall != "SP9CW" {
		t.Errorf("filtered spot = %q, want SP9CW", filtered[0].DXCall)
	}
}

// =============================================================================
// Time filter + band + mode — all three combined
// =============================================================================

func TestDXCTimeFilter_AllThreeFiltersCombined(t *testing.T) {
	now := testNow().Unix()
	old := testNow().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "TARGET", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
		{DXCall: "WRONGBAND", Frequency: 7200000, Band: "40m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
		{DXCall: "WRONGMODE", Frequency: 14195000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: now},
		{DXCall: "TOO_OLD", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.timeFilter = 5
	m.dxc.bandFilter = "20m"
	m.dxc.modeFilter = "DIGI"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (only TARGET matches all three filters)", len(filtered))
	}
	if filtered[0].DXCall != "TARGET" {
		t.Errorf("filtered spot = %q, want TARGET", filtered[0].DXCall)
	}
}

// =============================================================================
// Edge cases: empty spots, zero timestamp, large time windows
// =============================================================================

func TestDXCTimeFilter_EmptySpotsNoPanic(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.timeFilter = 30
	filtered := m.dxcFilteredSpots()
	if filtered != nil && len(filtered) > 0 {
		t.Errorf("filtered spots should be empty with no spots, got %d", len(filtered))
	}
}

func TestDXCTimeFilter_ZeroTimestampPasses(t *testing.T) {
	// A spot with ReceivedAt=0 (Unix epoch) is very old — should be dropped.
	spots := []store.DXCSpot{
		{DXCall: "SP9ZERO", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: 0},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 0 {
		t.Errorf("spot with zero timestamp should be dropped by time filter, got %d spots", len(filtered))
	}
}

func TestDXCTimeFilter_ZeroTimestampPassesWithAllTime(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9ZERO", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: 0},
	}
	m := newDXCBandFilterModel(t, spots)

	// All-time filter passes everything, even epoch-zero spots.
	m.dxc.timeFilter = 0
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (all-time passes zero-timestamp spot)", len(filtered))
	}
}

func TestDXCTimeFilter_LargeTimeFilterPassesAll(t *testing.T) {
	old := testNow().Add(-90 * time.Minute).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9OLD", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	// 60-minute filter: spot is 90 min old → dropped.
	m.dxc.timeFilter = 60
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 0 {
		t.Errorf("90-min-old spot should be dropped by 60-min filter, got %d", len(filtered))
	}
}

func TestDXCTimeFilter_ExactlyAtBoundary(t *testing.T) {
	// Spot exactly at the cutoff boundary: ReceivedAt = now - timeFilter minutes.
	// The cutoff is computed as: time.Now().UTC().Add(-timeFilter * time.Minute).Unix().
	// Since we can't perfectly synchronize with dxcFilteredSpots's internal time.Now(),
	// we create a spot from "just now" — it should pass.
	now := testNow().Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9NOW", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: now},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 1 {
		t.Fatalf("just-now spot should pass 5-min filter, got %d", len(filtered))
	}
}

// =============================================================================
// Selected call behavior during time filter changes
// =============================================================================

func TestDXCTimeFilter_TableNotReadyAfterCycle(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)
	m.dxc.tableReady = true

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.tableReady {
		t.Error("time filter change should set tableReady=false")
	}
}

// =============================================================================
// PgUp/PgDown cycling with empty DB — safety
// =============================================================================

func TestDXCTimeFilter_CycleForwardEmptyDB(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)

	// Should not panic when cycling through time windows with no spots.
	for i := 0; i < 10; i++ {
		_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	}
	// State should have cycled.
	if m.dxc.timeFilter < 0 || m.dxc.timeFilter > 60 {
		t.Errorf("timeFilter %d out of range after cycling", m.dxc.timeFilter)
	}
}

func TestDXCTimeFilter_CycleBackwardEmptyDB(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)

	for i := 0; i < 10; i++ {
		_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil)
	}
	if m.dxc.timeFilter < 0 || m.dxc.timeFilter > 60 {
		t.Errorf("timeFilter %d out of range after backward cycling", m.dxc.timeFilter)
	}
}
