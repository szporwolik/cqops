package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/store"
)

// TestDXCPathLineRespectsContinent verifies the QSO-form cluster line never
// shows spots outside the selected continent — the continent filter is strict
// and is never silently relaxed.
func TestDXCPathLineRespectsContinent(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Station.Continent = "EU"
	now := time.Now().Unix()

	euSpot := store.DXCSpot{DXCall: "SP9MOA", Frequency: 145540, SpotCont: "EU", ModeCat: "PHONE", ReceivedAt: now}
	naSpot := store.DXCSpot{DXCall: "KG5DCP", Frequency: 145520, SpotCont: "NA", ModeCat: "PHONE", ReceivedAt: now}

	m.fields[fieldFreq].SetValue("145.550")

	// Both continents present: only the EU spot may appear.
	m.dxc.cachedRaw = []store.DXCSpot{naSpot, euSpot}
	m.rc.dxcPathSig = ""
	line := m.dxcPathLine(120)
	if !strings.Contains(line, "SP9MOA") {
		t.Errorf("EU spot missing from path line: %q", line)
	}
	if strings.Contains(line, "KG5DCP") {
		t.Errorf("NA spot leaked into the path line: %q", line)
	}

	// Only non-EU spots: the line must stay empty instead of leaking them.
	m.dxc.cachedRaw = []store.DXCSpot{naSpot}
	m.rc.dxcPathSig = ""
	if line := m.dxcPathLine(120); line != "" {
		t.Errorf("path line should be empty with only NA spots, got %q", line)
	}

	// The DXC pane's explicit continent filter overrides the station default.
	m.dxc.contFilter = "NA"
	m.rc.dxcPathSig = ""
	line = m.dxcPathLine(120)
	if !strings.Contains(line, "KG5DCP") {
		t.Errorf("pane NA filter should show the NA spot, got %q", line)
	}
	m.dxc.contFilter = ""

	// No continent configured: no continent filter (historic behaviour).
	m.App.Logbook.Station.Continent = ""
	m.rc.dxcPathSig = ""
	line = m.dxcPathLine(120)
	if !strings.Contains(line, "KG5DCP") {
		t.Errorf("without continent config the spot should appear, got %q", line)
	}
}
