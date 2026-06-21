package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// DXC mode filter tests (Pass 25)
// =============================================================================
// Tests mode/category filter cycling, spot filtering by ModeCat,
// and interaction with band/time filters. Uses temp DB with controlled
// spot data. No real DX Cluster connection.

// =============================================================================
// Mode filter cycling — forward (Insert key)
// =============================================================================

func TestDXCModeFilter_CycleForward(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Start: modeFilter=""
	if m.dxc.modeFilter != "" {
		t.Fatalf("initial modeFilter = %q, want \"\"", m.dxc.modeFilter)
	}

	// Insert → CW.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "CW" {
		t.Errorf("1st Insert: modeFilter = %q, want CW", m.dxc.modeFilter)
	}

	// Insert → DIGI.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "DIGI" {
		t.Errorf("2nd Insert: modeFilter = %q, want DIGI", m.dxc.modeFilter)
	}

	// Insert → PHONE.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "PHONE" {
		t.Errorf("3rd Insert: modeFilter = %q, want PHONE", m.dxc.modeFilter)
	}

	// Insert → "" (wraparound back to all).
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "" {
		t.Errorf("4th Insert (wrap): modeFilter = %q, want \"\"", m.dxc.modeFilter)
	}
}

// =============================================================================
// Mode filter cycling — backward (Delete key)
// =============================================================================

func TestDXCModeFilter_CycleBackward(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)

	// Delete from "" wraps to PHONE.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	if m.dxc.modeFilter != "PHONE" {
		t.Errorf("1st Delete (from \"\"): modeFilter = %q, want PHONE", m.dxc.modeFilter)
	}

	// Delete → DIGI.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	if m.dxc.modeFilter != "DIGI" {
		t.Errorf("2nd Delete: modeFilter = %q, want DIGI", m.dxc.modeFilter)
	}

	// Delete → CW.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	if m.dxc.modeFilter != "CW" {
		t.Errorf("3rd Delete: modeFilter = %q, want CW", m.dxc.modeFilter)
	}

	// Delete → "".
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	if m.dxc.modeFilter != "" {
		t.Errorf("4th Delete: modeFilter = %q, want \"\"", m.dxc.modeFilter)
	}
}

// =============================================================================
// Backspace clears mode filter
// =============================================================================

func TestDXCModeFilter_ClearWithBackspace(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.modeFilter = "CW"
	m.dxc.modeIdx = 2
	m.dxc.bandFilter = "20m"

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)

	if m.dxc.modeFilter != "" {
		t.Errorf("modeFilter = %q, want \"\" after backspace", m.dxc.modeFilter)
	}
	if m.dxc.modeIdx != 0 {
		t.Errorf("modeIdx = %d, want 0 after backspace", m.dxc.modeIdx)
	}
	if m.dxc.bandFilter != "" {
		t.Errorf("bandFilter = %q, want \"\" after backspace", m.dxc.bandFilter)
	}
}

// =============================================================================
// Mode filter forces table rebuild
// =============================================================================

func TestDXCModeFilter_ForcesTableRebuild(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)
	m.dxc.tableReady = true

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.tableReady {
		t.Error("mode filter change should set tableReady=false")
	}
}

// =============================================================================
// Mode filter actually filters spots by ModeCat
// =============================================================================

func TestDXCModeFilter_FiltersByCW(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9CW1", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9SSB", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
		{DXCall: "SP9CW2", Frequency: 14250000, Band: "20m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "CW"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2 (CW spots)", len(filtered))
	}
	for _, s := range filtered {
		if s.ModeCat != "CW" {
			t.Errorf("spot %q has ModeCat=%q, want CW", s.DXCall, s.ModeCat)
		}
	}
}

func TestDXCModeFilter_FiltersByDIGI(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9FT8", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: nowUnix()},
		{DXCall: "SP9CW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9RTTY", Frequency: 14250000, Band: "20m", Mode: "RTTY", ModeCat: "DIGI", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "DIGI"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2 (DIGI spots)", len(filtered))
	}
	for _, s := range filtered {
		if s.ModeCat != "DIGI" {
			t.Errorf("spot %q has ModeCat=%q, want DIGI", s.DXCall, s.ModeCat)
		}
	}
}

func TestDXCModeFilter_FiltersByPHONE(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9SSB", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
		{DXCall: "SP9FM", Frequency: 145500000, Band: "2m", Mode: "FM", ModeCat: "PHONE", ReceivedAt: nowUnix()},
		{DXCall: "SP9CW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "PHONE"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2 (PHONE spots)", len(filtered))
	}
	for _, s := range filtered {
		if s.ModeCat != "PHONE" {
			t.Errorf("spot %q has ModeCat=%q, want PHONE", s.DXCall, s.ModeCat)
		}
	}
}

func TestDXCModeFilter_AllModePassesAll(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9CW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9FT8", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: nowUnix()},
		{DXCall: "SP9SSB", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// modeFilter="" means all modes.
	m.dxc.modeFilter = ""
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 3 {
		t.Fatalf("filtered count = %d, want 3 (all modes)", len(filtered))
	}
}

// =============================================================================
// Mode filter combines with band filter
// =============================================================================

func TestDXCModeFilter_CombinedWithBandFilter(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9CW20", Frequency: 14250000, Band: "20m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9CW40", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9SSB20", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "CW"
	m.dxc.bandFilter = "20m"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (CW on 20m)", len(filtered))
	}
	if filtered[0].DXCall != "SP9CW20" {
		t.Errorf("filtered spot = %q, want SP9CW20", filtered[0].DXCall)
	}
}

// =============================================================================
// Mode filter combines with time filter
// =============================================================================

func TestDXCModeFilter_CombinedWithTimeFilter(t *testing.T) {
	now := time.Now().UTC().Unix()
	old := time.Now().UTC().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "SP9CW_NOW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: now},
		{DXCall: "SP9CW_OLD", Frequency: 14250000, Band: "20m", Mode: "CW", ModeCat: "CW", ReceivedAt: old},
		{DXCall: "SP9FT8_NOW", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "CW"
	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (recent CW)", len(filtered))
	}
	if filtered[0].DXCall != "SP9CW_NOW" {
		t.Errorf("filtered spot = %q, want SP9CW_NOW", filtered[0].DXCall)
	}
}

// =============================================================================
// Mode + band + time — all three filters combined
// =============================================================================

func TestDXCModeFilter_AllThreeFiltersCombined(t *testing.T) {
	now := time.Now().UTC().Unix()
	old := time.Now().UTC().Add(-2 * time.Hour).Unix()
	spots := []store.DXCSpot{
		{DXCall: "TARGET", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
		{DXCall: "WRONG_BAND", Frequency: 7200000, Band: "40m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: now},
		{DXCall: "WRONG_MODE", Frequency: 14195000, Band: "20m", Mode: "CW", ModeCat: "CW", ReceivedAt: now},
		{DXCall: "TOO_OLD", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: old},
	}
	m := newDXCBandFilterModel(t, spots)

	m.dxc.modeFilter = "DIGI"
	m.dxc.bandFilter = "20m"
	m.dxc.timeFilter = 5
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (only TARGET matches DIGI+20m+recent)", len(filtered))
	}
	if filtered[0].DXCall != "TARGET" {
		t.Errorf("filtered spot = %q, want TARGET", filtered[0].DXCall)
	}
}

// =============================================================================
// Unknown/empty ModeCat behavior
// =============================================================================

func TestDXCModeFilter_UnknownModeCatNotFiltered(t *testing.T) {
	// Spots with ModeCat="" (unknown category) are NOT matched by any
	// specific mode filter. They are only visible when modeFilter="".
	spots := []store.DXCSpot{
		{DXCall: "SP9UNK", Frequency: 14250000, Band: "20m", Mode: "BOGUS", ModeCat: "", ReceivedAt: nowUnix()},
		{DXCall: "SP9CW", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// With CW filter, only the CW spot should pass.
	m.dxc.modeFilter = "CW"
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 1 || filtered[0].DXCall != "SP9CW" {
		t.Errorf("CW filter: got %d spots, call=%q; want 1 spot SP9CW", len(filtered), filtered[0].DXCall)
	}

	// With all-mode filter, both spots pass.
	m.dxc.modeFilter = ""
	filtered = m.dxcFilteredSpots()
	if len(filtered) != 2 {
		t.Errorf("all-mode filter: got %d spots, want 2", len(filtered))
	}
}

// =============================================================================
// Edge cases: empty spots, cycling with empty DB
// =============================================================================

func TestDXCModeFilter_EmptySpotsNoPanic(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.modeFilter = "CW"
	filtered := m.dxcFilteredSpots()
	if len(filtered) > 0 {
		t.Errorf("filtered spots should be empty, got %d", len(filtered))
	}
}

func TestDXCModeFilter_CycleForwardEmptyDB(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	for i := 0; i < 10; i++ {
		_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	}
	// After 10 cycles: Insert cycles through 4 choices → index 2 (DIGI).
	if m.dxc.modeFilter != "DIGI" {
		t.Errorf("after 10 Insert cycles: modeFilter = %q, want DIGI", m.dxc.modeFilter)
	}
}

func TestDXCModeFilter_CycleBackwardEmptyDB(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	for i := 0; i < 10; i++ {
		_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	}
	// After 10 Delete cycles through 4 choices → index 2 (DIGI).
	if m.dxc.modeFilter != "DIGI" {
		t.Errorf("after 10 Delete cycles: modeFilter = %q, want DIGI", m.dxc.modeFilter)
	}
}
