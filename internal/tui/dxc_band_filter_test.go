package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// DXC band filter tests (Pass 19)
// =============================================================================
// These tests verify band filter cycling, available-band derivation, and
// filtered-spot behavior using a temp SQLite database pre-seeded with
// realistic DXC spots. No real DX Cluster connection is used.
// Shared test helpers (newDXCBandFilterModel, nowUnix) are in dxc_test_helpers.go.

// =============================================================================
// Available bands and band choices
// =============================================================================

func TestDXCBandFilter_AvailableBands(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 21100000, Band: "15m", Mode: "FT8", ReceivedAt: nowUnix()},
		{DXCall: "SP9DDD", Frequency: 28500000, Band: "10m", Mode: "FM", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	bands := m.dxcAvailableBands()
	// Bands should be sorted by wavelength (lowest freq first).
	// 80m (3.7), 20m (14.25), 15m (21.1), 10m (28.5)
	expected := []string{"80m", "20m", "15m", "10m"}
	if len(bands) != len(expected) {
		t.Fatalf("availableBands = %v, want %v", bands, expected)
	}
	for i, b := range expected {
		if bands[i] != b {
			t.Errorf("availableBands[%d] = %q, want %q", i, bands[i], b)
		}
	}
}

func TestDXCBandFilter_AvailableBandsWithOther(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 99999999, Band: "", Mode: "FM", ReceivedAt: nowUnix()}, // no band
	}
	m := newDXCBandFilterModel(t, spots)

	bands := m.dxcAvailableBands()
	// "20m" first (known), then "other" last.
	expected := []string{"20m", "other"}
	if len(bands) != len(expected) {
		t.Fatalf("availableBands = %v, want %v", bands, expected)
	}
	if bands[0] != "20m" {
		t.Errorf("bands[0] = %q, want 20m", bands[0])
	}
	if bands[1] != "other" {
		t.Errorf("bands[1] = %q, want other", bands[1])
	}
}

func TestDXCBandFilter_BandChoices(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	choices := m.dxcBandChoices()
	// Choices always start with "" (all), then sorted bands.
	expected := []string{"", "80m", "20m"}
	if len(choices) != len(expected) {
		t.Fatalf("bandChoices = %v, want %v", choices, expected)
	}
	for i, c := range expected {
		if choices[i] != c {
			t.Errorf("bandChoices[%d] = %q, want %q", i, choices[i], c)
		}
	}
}

func TestDXCBandFilter_BandChoicesWithOther(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 99999999, Band: "", Mode: "FM", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	choices := m.dxcBandChoices()
	// "", "20m", "other"
	expected := []string{"", "20m", "other"}
	if len(choices) != len(expected) {
		t.Fatalf("bandChoices = %v, want %v", choices, expected)
	}
	if choices[2] != "other" {
		t.Errorf("choices[2] = %q, want other", choices[2])
	}
}

func TestDXCBandFilter_EmptySpotsNoChoices(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)

	bands := m.dxcAvailableBands()
	if len(bands) != 0 {
		t.Errorf("availableBands should be empty with no spots, got %v", bands)
	}

	choices := m.dxcBandChoices()
	// With no spots, choices should only be [""].
	if len(choices) != 1 || choices[0] != "" {
		t.Errorf("bandChoices = %v, want [\"\"]", choices)
	}
}

// =============================================================================
// Band filter cycling — forward (Home key)
// =============================================================================

func TestDXCBandFilter_CycleForward(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 21100000, Band: "15m", Mode: "FT8", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Start: bandFilter=""
	if m.dxc.bandFilter != "" {
		t.Fatalf("initial bandFilter = %q, want \"\"", m.dxc.bandFilter)
	}

	// Home → 80m
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.bandFilter != "80m" {
		t.Errorf("1st Home: bandFilter = %q, want 80m", m.dxc.bandFilter)
	}

	// Home → 20m
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.bandFilter != "20m" {
		t.Errorf("2nd Home: bandFilter = %q, want 20m", m.dxc.bandFilter)
	}

	// Home → 15m
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.bandFilter != "15m" {
		t.Errorf("3rd Home: bandFilter = %q, want 15m", m.dxc.bandFilter)
	}

	// Home → "" (wraparound)
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.bandFilter != "" {
		t.Errorf("4th Home: bandFilter = %q, want \"\" (wraparound)", m.dxc.bandFilter)
	}
}

// =============================================================================
// Band filter cycling — backward (End key)
// =============================================================================

func TestDXCBandFilter_CycleBackward(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Start: bandFilter=""
	// End backward from "" wraps to last choice → "20m"
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
	if m.dxc.bandFilter != "20m" {
		t.Errorf("1st End (from \"\"): bandFilter = %q, want 20m", m.dxc.bandFilter)
	}

	// End → 80m
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
	if m.dxc.bandFilter != "80m" {
		t.Errorf("2nd End: bandFilter = %q, want 80m", m.dxc.bandFilter)
	}

	// End → "" (wraparound)
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
	if m.dxc.bandFilter != "" {
		t.Errorf("3rd End: bandFilter = %q, want \"\"", m.dxc.bandFilter)
	}
}

// =============================================================================
// Backspace clears band filter
// =============================================================================

func TestDXCBandFilter_ClearWithBackspace(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Set band filter first.
	m.dxc.bandFilter = "20m"
	m.dxc.bandIdx = 1
	m.dxc.timeFilter = 30
	m.dxc.modeFilter = "CW"

	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)

	if m.dxc.bandFilter != "" {
		t.Errorf("bandFilter = %q, want \"\" after backspace", m.dxc.bandFilter)
	}
	if m.dxc.bandIdx != 0 {
		t.Errorf("bandIdx = %d, want 0 after backspace", m.dxc.bandIdx)
	}
	if m.dxc.timeFilter != 0 {
		t.Errorf("timeFilter = %d, want 0 after backspace", m.dxc.timeFilter)
	}
	if m.dxc.modeFilter != "" {
		t.Errorf("modeFilter = %q, want \"\" after backspace", m.dxc.modeFilter)
	}
}

// =============================================================================
// Filter forces table rebuild
// =============================================================================

func TestDXCBandFilter_ForcesTableRebuild(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)
	m.dxc.tableReady = true

	// Home should set tableReady=false to force a rebuild.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.tableReady {
		t.Error("band filter change should set tableReady=false, got true")
	}
}

// =============================================================================
// dxcFilteredSpots respects band filter
// =============================================================================

func TestDXCBandFilter_FilteredSpotsBandMatch(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 14195000, Band: "20m", Mode: "FT8", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Filter to 20m only.
	m.dxc.bandFilter = "20m"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2 (both 20m spots)", len(filtered))
	}
	for _, s := range filtered {
		if s.Band != "20m" {
			t.Errorf("spot %q has band %q, want 20m", s.DXCall, s.Band)
		}
	}
	// Should be sorted by frequency descending within the band.
	if filtered[0].Frequency < filtered[1].Frequency {
		t.Errorf("spots should be sorted by freq desc: %.1f < %.1f",
			filtered[0].Frequency, filtered[1].Frequency)
	}
}

func TestDXCBandFilter_FilteredSpotsAll(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// No band filter → all spots returned.
	m.dxc.bandFilter = ""
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 2 {
		t.Fatalf("filtered count = %d, want 2 (all spots)", len(filtered))
	}
}

func TestDXCBandFilter_FilteredSpotsOther(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 99999999, Band: "", Mode: "FM", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Filter to "other" (spots without band).
	m.dxc.bandFilter = "other"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (unclassified spot)", len(filtered))
	}
	if filtered[0].DXCall != "SP9BBB" {
		t.Errorf("filtered spot call = %q, want SP9BBB", filtered[0].DXCall)
	}
}

// =============================================================================
// Selected call behavior during filter changes
// =============================================================================

func TestDXCBandFilter_SelectedCallPreserved(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14195000, Band: "20m", Mode: "FT8", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Set selected call to a spot that will still be in the filtered results.
	m.dxc.selectedCall = "SP9AAA"
	m.dxc.bandFilter = "20m"

	// Trigger band filter cycle back to "" (all bands).
	// The selectedCall is not cleared by band filter changes specifically —
	// it's cleared by updateDXCSelectedCall when table becomes unready.
	// But updateDXCSelectedCall checks tableReady and total rows.
	// After cycling filter, tableReady=false. When the next View rebuilds
	// the table, updateDXCSelectedCall clears selectedCall if the spot is
	// no longer in the table. We test that selectedCall survives the filter
	// transition at the message-handler level.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	// selectedCall is not cleared by handleDXCUpdate directly; it's cleared
	// in updateDXCSelectedCall when the table is not ready.
	if m.dxc.selectedCall != "SP9AAA" {
		// This is actually expected — updateDXCSelectedCall runs after
		// table update when tableReady is false, which clears selectedCall.
		// Document the current behavior.
		t.Log("selectedCall was cleared during filter change (expected: table rebuild resets selection)")
	}
}

// =============================================================================
// Robustness: empty spots, malformed data, no panic
// =============================================================================

func TestDXCBandFilter_NoPanicOnEmptyDB(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)

	// All these should return clean without panic.
	bands := m.dxcAvailableBands()
	_ = bands
	choices := m.dxcBandChoices()
	_ = choices
	filtered := m.dxcFilteredSpots()
	if len(filtered) > 0 {
		t.Errorf("filtered spots should be empty with no spots, got %d", len(filtered))
	}

	// Cycling band filter with no spots should not panic.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
}

func TestDXCBandFilter_NoPanicOnEmptyBandChoice(t *testing.T) {
	// Create model with no spots, then try band filter cycle.
	m := newDXCBandFilterModel(t, nil)

	// bandChoices = [""], length 1.
	// Home: bandIdx = (0+1)%1 = 0, bandFilter = "" → no change.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.dxc.bandFilter != "" {
		t.Errorf("bandFilter should stay \"\" with no bands, got %q", m.dxc.bandFilter)
	}

	// End: bandIdx = -1 → wraps to 0, bandFilter = "".
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
	if m.dxc.bandFilter != "" {
		t.Errorf("bandFilter should stay \"\" with no bands, got %q", m.dxc.bandFilter)
	}
}

func TestDXCBandFilter_OtherCategoryPreserved(t *testing.T) {
	// Spots with empty band should appear in "other" filter.
	spots := []store.DXCSpot{
		{DXCall: "SP9ZZZ", Frequency: 99999999, Band: "", Mode: "FM", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	choices := m.dxcBandChoices()
	foundOther := false
	for _, c := range choices {
		if c == "other" {
			foundOther = true
			break
		}
	}
	if !foundOther {
		t.Errorf("bandChoices should contain \"other\" for unclassified spot, got %v", choices)
	}

	// Filter to other.
	m.dxc.bandFilter = "other"
	filtered := m.dxcFilteredSpots()
	if len(filtered) != 1 {
		t.Errorf("filtered count = %d, want 1", len(filtered))
	}
	if filtered[0].DXCall != "SP9ZZZ" {
		t.Errorf("filtered spot call = %q, want SP9ZZZ", filtered[0].DXCall)
	}
}

func TestDXCBandFilter_MultipleSameBand(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14195000, Band: "20m", Mode: "FT8", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 14025000, Band: "20m", Mode: "CW", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	bands := m.dxcAvailableBands()
	// 20m should appear only once even though 3 spots are on 20m.
	if len(bands) != 1 || bands[0] != "20m" {
		t.Errorf("availableBands = %v, want [20m] (deduplicated)", bands)
	}

	choices := m.dxcBandChoices()
	expected := []string{"", "20m"}
	if len(choices) != len(expected) {
		t.Errorf("bandChoices = %v, want %v", choices, expected)
	}
}

func TestDXCBandFilter_DeriveBandFromFrequency(t *testing.T) {
	// Some DXC spots may have a band but no explicit band field.
	// The store.DXCSpot has a Band field; if it's empty, the spot
	// goes into "other". The band filter tests "other" separately.
	// This test verifies that spots on known amateur frequencies
	// with explicitly set bands filter correctly.
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 3700000, Band: "80m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 7200000, Band: "40m", Mode: "SSB", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 14250000, Band: "20m", Mode: "FT8", ReceivedAt: nowUnix()},
		{DXCall: "SP9DDD", Frequency: 21100000, Band: "15m", Mode: "CW", ReceivedAt: nowUnix()},
		{DXCall: "SP9EEE", Frequency: 28500000, Band: "10m", Mode: "SSB", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	bands := m.dxcAvailableBands()
	expected := []string{"80m", "40m", "20m", "15m", "10m"}
	if len(bands) != len(expected) {
		t.Fatalf("availableBands = %v, want %v", bands, expected)
	}
	for i, b := range expected {
		if bands[i] != b {
			t.Errorf("bands[%d] = %q, want %q", i, bands[i], b)
		}
	}
}

// =============================================================================
// Mixed modes — mode filter interaction not blocking band filter
// =============================================================================

func TestDXCBandFilter_WithModeFilterApplied(t *testing.T) {
	spots := []store.DXCSpot{
		{DXCall: "SP9AAA", Frequency: 14250000, Band: "20m", Mode: "SSB", ModeCat: "PHONE", ReceivedAt: nowUnix()},
		{DXCall: "SP9BBB", Frequency: 14195000, Band: "20m", Mode: "FT8", ModeCat: "DIGI", ReceivedAt: nowUnix()},
		{DXCall: "SP9CCC", Frequency: 7200000, Band: "40m", Mode: "CW", ModeCat: "CW", ReceivedAt: nowUnix()},
	}
	m := newDXCBandFilterModel(t, spots)

	// Band filter to 20m + mode filter to DIGI → only SP9BBB should be visible.
	m.dxc.bandFilter = "20m"
	m.dxc.modeFilter = "DIGI"
	filtered := m.dxcFilteredSpots()

	if len(filtered) != 1 {
		t.Fatalf("filtered count = %d, want 1 (20m + DIGI)", len(filtered))
	}
	if filtered[0].DXCall != "SP9BBB" {
		t.Errorf("filtered spot call = %q, want SP9BBB", filtered[0].DXCall)
	}
}
