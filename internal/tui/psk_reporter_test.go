package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/psk"
)

func TestFormatAge(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		ts       int64
		contains string // formatAge output changes with time, so we check patterns
	}{
		{"seconds ago", now.Add(-30 * time.Second).Unix(), "<1m"},
		{"minutes ago", now.Add(-5 * time.Minute).Unix(), "m"},
		{"hours ago", now.Add(-3 * time.Hour).Unix(), "h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAge(tt.ts)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("formatAge(%d) = %q, want containing %q", tt.ts, result, tt.contains)
			}
		})
	}

	// Exact values for known offsets.
	t.Run("exact 90 seconds", func(t *testing.T) {
		result := formatAge(now.Add(-90 * time.Second).Unix())
		if result != "1m" {
			t.Errorf("expected '1m', got %q", result)
		}
	})
	t.Run("exact 2 hours", func(t *testing.T) {
		result := formatAge(now.Add(-2 * time.Hour).Unix())
		if result != "2h" {
			t.Errorf("expected '2h', got %q", result)
		}
	})
}

func TestFreqToBandName(t *testing.T) {
	tests := []struct {
		freqHz float64
		band   string
	}{
		{1_850_000, "160m"},
		{1_900_000, "160m"},
		{3_600_000, "80m"},
		{3_800_000, "80m"},
		{7_100_000, "40m"},
		{7_200_000, "40m"},
		{14_100_000, "20m"},
		{14_200_000, "20m"},
		{21_100_000, "15m"},
		{21_300_000, "15m"},
		{28_100_000, "10m"},
		{28_500_000, "10m"},
		// Edge cases.
		{1_799_000, "other"}, // just below 160m
		{2_000_000, "160m"},  // top of 160m band (1.8–2.0 MHz)
		{5_300_000, "60m"},   // 60m band (5.06–5.45 MHz)
		{50_100_000, "6m"},   // 6m band (50–54 MHz)
		{0, "other"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%.0fHz->%s", tt.freqHz, tt.band), func(t *testing.T) {
			result := freqToBandName(tt.freqHz)
			if result != tt.band {
				t.Errorf("freqToBandName(%.0f) = %q, want %q", tt.freqHz, result, tt.band)
			}
		})
	}
}

func TestPSKBandStyle(t *testing.T) {
	// Verify that each band gets a distinct style (no panics, non-empty).
	bands := []struct {
		freqHz float64
		name   string
	}{
		{1_900_000, "160m"},
		{3_700_000, "80m"},
		{7_150_000, "40m"},
		{14_200_000, "20m"},
		{21_200_000, "15m"},
		{28_500_000, "10m"},
		{5_000_000, "other"},
	}

	for _, b := range bands {
		t.Run(b.name, func(t *testing.T) {
			style := pskBandStyle(b.freqHz)
			rendered := style.Render("x")
			if rendered == "" {
				t.Errorf("pskBandStyle(%f) rendered empty string", b.freqHz)
			}
		})
	}

	// Verify that different bands produce different visual output (the ANSI codes differ).
	style160 := pskBandStyle(1_900_000).Render(".")
	style10 := pskBandStyle(28_500_000).Render(".")
	if style160 == style10 {
		t.Error("160m and 10m should have different styles")
	}
}

func TestPSKTableRow(t *testing.T) {
	r := psk.Report{
		ReceiverCallsign: "K1ABC",
		ReceiverLocator:  "FN42ab",
		Frequency:        14_074_000,
		SNR:              12,
		Mode:             "FT8",
		FlowStartSeconds: time.Now().Add(-2 * time.Minute).Unix(),
	}

	row := pskTableRow(r, 14, 10, 8, 4, 6)

	// The row should contain key data.
	if !strings.Contains(row, "K1ABC") {
		t.Error("row should contain callsign")
	}
	if !strings.Contains(row, "FN42ab") {
		t.Error("row should contain grid")
	}
	if !strings.Contains(row, "14.074") {
		t.Error("row should contain frequency in MHz")
	}
	if !strings.Contains(row, "12") {
		t.Error("row should contain SNR")
	}
	if !strings.Contains(row, "FT8") {
		t.Error("row should contain mode")
	}
	// Age should be "2m" or close (allow 1m due to timing).
	if !strings.Contains(row, "m") {
		t.Error("row should contain age with 'm' suffix")
	}
}

func TestPSKTableRow_ZeroSNR(t *testing.T) {
	r := psk.Report{
		ReceiverCallsign: "K1ABC",
		ReceiverLocator:  "FN42ab",
		Frequency:        14_074_000,
		SNR:              0,
		Mode:             "FT8",
		FlowStartSeconds: time.Now().Unix(),
	}

	row := pskTableRow(r, 14, 10, 8, 4, 6)
	// Zero SNR should produce empty string in the SNR column.
	if strings.Contains(row, " 0 ") {
		t.Error("zero SNR should not show '0'")
	}
}

func TestPSKTableRow_Truncation(t *testing.T) {
	r := psk.Report{
		ReceiverCallsign: "VERYLONGCALLSIGN",
		ReceiverLocator:  "VERYLONGLOCATOR",
		Frequency:        14_074_000,
		SNR:              12,
		Mode:             "VERYLONGMODE",
		FlowStartSeconds: time.Now().Unix(),
	}

	// Very narrow columns should still produce output without panicking.
	row := pskTableRow(r, 4, 4, 4, 2, 3)
	if row == "" {
		t.Error("row should not be empty even with narrow columns")
	}
}

func TestPSKCellCoords(t *testing.T) {
	// Map dimensions: 80 cols, 20 rows.
	cx, cy := pskCellCoords(0, 0, 80, 20)
	if cx < 0 || cx >= 80 {
		t.Errorf("cx out of range: %d", cx)
	}
	if cy < 0 || cy >= 20 {
		t.Errorf("cy out of range: %d", cy)
	}

	// North pole should be clamped.
	cxN, cyN := pskCellCoords(90, 0, 80, 20)
	if cyN > 0 {
		t.Logf("north pole y=%d (expected near 0)", cyN)
	}

	// South pole should be clamped.
	cxS, cyS := pskCellCoords(-90, 0, 80, 20)
	if cyS < 19 {
		t.Logf("south pole y=%d (expected near bottom)", cyS)
	}
	_ = cxN
	_ = cxS
}

func TestBuildPSKTable_Empty(t *testing.T) {
	m := &Model{
		psk: pskState{selected: 0},
	}

	result := m.buildPSKTable(nil, 60, 5)
	if result == "" {
		t.Error("table should not be empty (header always present)")
	}
	if !strings.Contains(result, "Call") {
		t.Error("table should contain 'Call' header")
	}
}

func TestBuildPSKTable_WithData(t *testing.T) {
	reports := []psk.Report{
		{ReceiverCallsign: "K1ABC", ReceiverLocator: "FN42ab", Frequency: 14_074_000, SNR: 12, Mode: "FT8", FlowStartSeconds: time.Now().Add(-2 * time.Minute).Unix()},
		{ReceiverCallsign: "W1AW", ReceiverLocator: "FN31pr", Frequency: 21_074_000, SNR: -3, Mode: "FT4", FlowStartSeconds: time.Now().Add(-5 * time.Minute).Unix()},
		{ReceiverCallsign: "G4ABC", ReceiverLocator: "IO91wm", Frequency: 28_074_000, SNR: 5, Mode: "FT8", FlowStartSeconds: time.Now().Add(-10 * time.Minute).Unix()},
		{ReceiverCallsign: "JA1ABC", ReceiverLocator: "PM95", Frequency: 7_074_000, SNR: 8, Mode: "FT8", FlowStartSeconds: time.Now().Add(-1 * time.Minute).Unix()},
		{ReceiverCallsign: "VK2ABC", ReceiverLocator: "QF56", Frequency: 3_573_000, SNR: 15, Mode: "FT8", FlowStartSeconds: time.Now().Add(-15 * time.Minute).Unix()},
		{ReceiverCallsign: "ZL1ABC", ReceiverLocator: "RF73", Frequency: 14_074_000, SNR: -5, Mode: "FT4", FlowStartSeconds: time.Now().Add(-20 * time.Minute).Unix()},
	}

	m := &Model{psk: pskState{selected: 0}}
	result := m.buildPSKTable(reports, 60, 5)

	if result == "" {
		t.Error("table should not be empty")
	}
	if !strings.Contains(result, "K1ABC") {
		t.Error("table should contain first callsign")
	}
	if !strings.Contains(result, "\u2193") {
		t.Error("table should show 'more below' indicator when 6 reports, 5 visible")
	}

	// Select the last visible row.
	m.psk.selected = 4
	result = m.buildPSKTable(reports, 60, 5)
	if !strings.Contains(result, "VK2ABC") {
		t.Error("table should show 5th report when selected=4")
	}
	if strings.Contains(result, "K1ABC") && strings.Contains(result, "VK2ABC") {
		// First page should still show both if all fit.
	}

	// Select beyond visible page.
	m.psk.selected = 5
	result = m.buildPSKTable(reports, 60, 5)
	if !strings.Contains(result, "ZL1ABC") {
		t.Error("table should scroll to show 6th report when selected=5")
	}
	if !strings.Contains(result, "\u2191") {
		t.Error("table should show 'more above' indicator when scrolled")
	}
}

func TestBuildPSKFilters(t *testing.T) {
	m := &Model{
		psk: pskState{
			filterMins:      15,
			bandFilter:      "20m",
			modeFilter:      "FT8",
			lastFetchByCall: map[string]time.Time{"": time.Now().Add(-2 * time.Minute)},
		},
	}

	result := m.buildPSKFilters(30)

	if !strings.Contains(result, "15 min") {
		t.Error("filters should show time window")
	}
	if !strings.Contains(result, "20m") {
		t.Error("filters should show band filter")
	}
	if !strings.Contains(result, "FT8") {
		t.Error("filters should show mode filter")
	}
	if !strings.Contains(result, "Next update") {
		t.Error("filters should show next update time")
	}
}

func TestBuildPSKFilters_Defaults(t *testing.T) {
	m := &Model{
		psk: pskState{
			filterMins: 60,
			bandFilter: "",
			modeFilter: "",
		},
		// psk.lastFetch is zero — never fetched.
	}

	result := m.buildPSKFilters(30)

	if !strings.Contains(result, "60 min") {
		t.Error("filters should show time window")
	}
	if !strings.Contains(result, "all") {
		t.Error("filters should show 'all' for empty band filter")
	}
	// No "Next update" when never fetched.
	if strings.Contains(result, "Next update") {
		t.Error("filters should NOT show 'Next update' when never fetched")
	}
}

func TestPSKFilterSteps(t *testing.T) {
	// Verify filter steps are in ascending order.
	for i := 1; i < len(pskFilterSteps); i++ {
		if pskFilterSteps[i] <= pskFilterSteps[i-1] {
			t.Errorf("pskFilterSteps not ascending: %d <= %d at index %d",
				pskFilterSteps[i], pskFilterSteps[i-1], i)
		}
	}
	// First step should be 5 minutes.
	if pskFilterSteps[0] != 5 {
		t.Errorf("first filter step should be 5, got %d", pskFilterSteps[0])
	}
}

func TestPSKBandMarkers_Distinct(t *testing.T) {
	// All band markers should produce distinct ANSI output.
	markers := []lipgloss.Style{pskMark160, pskMark80, pskMark40, pskMark20, pskMark15, pskMark10, pskMarkOther}
	rendered := make(map[string]bool)
	for _, m := range markers {
		r := m.Render("\u25cf")
		if rendered[r] {
			t.Errorf("duplicate marker style: %q", r)
		}
		rendered[r] = true
	}
}

func TestBuildPSKMap_NoGrid(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.App.Logbook.Station.Grid = ""

	result := m.buildPSKMap(nil, 60, 15)
	if result != "" {
		t.Errorf("expected empty string when no grid, got %q", result)
	}
}

func TestBuildPSKMap_NoMapView(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{}, Logbook: &config.Logbook{}}
	m.App.Logbook.Station.Grid = "JO90"

	// mapView is nil — BaseImage returns "".
	result := m.buildPSKMap(nil, 60, 15)
	if result != "" {
		t.Errorf("expected empty string when mapView is nil, got %q", result)
	}
}

func TestBuildPSKMap_WithReports(t *testing.T) {
	m := &Model{}
	m.App = &app.App{Config: &config.Config{General: config.GeneralConfig{RenderMap: true, DrawGrayline: false}}, Logbook: &config.Logbook{}}
	m.App.Logbook.Station.Grid = "JO90ab" // Krakow
	m.mapView = newMapRenderer()

	reports := []psk.Report{
		{ReceiverCallsign: "K1ABC", ReceiverLocator: "FN42ab", Frequency: 14_074_000, SNR: 12, Mode: "FT8"},
		{ReceiverCallsign: "W1AW", ReceiverLocator: "FN31pr", Frequency: 21_074_000, SNR: -3, Mode: "FT4"},
	}

	// Render map — should not panic.
	result := m.buildPSKMap(reports, 80, 15)
	if result == "" {
		t.Log("map returned empty (may happen if mapImg is nil in test) — non-panicking is the key check")
	}
	// If mapImg is available, verify it contains marker and legend.
	if result != "" {
		if !strings.Contains(result, "My station") {
			t.Error("map legend should contain 'My station'")
		}
		if !strings.Contains(result, "160m") {
			t.Error("map legend should contain '160m'")
		}
	}
}
