package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// DXC status suffix visibility tests
// =============================================================================
// The "Spot X/Y  Page Z/W" status used to vanish on the entry frame and on
// every spot batch because the help bar was computed before the lazy table
// rebuild. These tests pin both frames.

func TestDXCStatusSuffixRendersOnSpotArrivalFrame(t *testing.T) {
	spots := []store.DXCSpot{
		{ReceivedAt: nowUnix(), Frequency: 14250000, Band: "20m", Mode: "SSB",
			DXCall: "SP9AAA", Spotter: "SP9XYZ", DXCC: "269"},
	}
	m := newDXCBandFilterModel(t, spots)
	m.width = 100
	m.height = 24
	// State right after a spot batch arrives: the table cache is
	// invalidated and awaits the lazy rebuild in View.
	m.dxc.tableReady = false
	m.dxc.cachedSpots = nil

	v := m.View()
	if !strings.Contains(v.Content, "Spot ") {
		t.Fatalf("spot/page status missing on spot-arrival frame:\n%s", v.Content)
	}
}

// TestDXCStatusSuffixRendersOnEntryFrames: the status must be visible on
// the very first frame after entering the DXC screen (before any cursor
// movement) and remain visible on subsequent idle frames.
func TestDXCStatusSuffixRendersOnEntryFrames(t *testing.T) {
	spots := []store.DXCSpot{
		{ReceivedAt: nowUnix(), Frequency: 14250000, Band: "20m", Mode: "SSB",
			DXCall: "SP9AAA", Spotter: "SP9XYZ", DXCC: "269"},
	}
	m := newDXCBandFilterModel(t, spots)
	m.App.Config.Integrations.DXC.Enabled = true
	m.dxc.online = true
	m.screen = screenQSO
	m.width = 100
	m.height = 24
	m.dxc.tableReady = false

	// Enter the DXC screen via F4.
	_, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyF4})
	v := m.View()
	if !strings.Contains(v.Content, "Spot ") {
		t.Fatalf("spot/page status missing on entry frame:\n%s", v.Content)
	}

	// Idle frame with no input — must not blink away.
	v2 := m.View()
	if !strings.Contains(v2.Content, "Spot ") {
		t.Fatalf("spot/page status missing on idle frame:\n%s", v2.Content)
	}
}
