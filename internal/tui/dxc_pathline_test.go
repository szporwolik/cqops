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

// TestDXCDupeSigIncludesLogbookAndRevision verifies the dupe cache key is
// distinct per logbook and per QSO revision, so results from another
// logbook — or from before the last mutation — can never be reused.
func TestDXCDupeSigIncludesLogbookAndRevision(t *testing.T) {
	base := dxcDupeSigFor("20260922", "contest", "A", 0)
	if got := dxcDupeSigFor("20260922", "contest", "A", 0); got != base {
		t.Errorf("identical inputs should produce identical sigs: %q vs %q", got, base)
	}
	if got := dxcDupeSigFor("20260922", "contest", "B", 0); got == base {
		t.Error("logbook identity is missing from the dupe sig")
	}
	if got := dxcDupeSigFor("20260922", "contest", "A", 1); got == base {
		t.Error("dupe revision is missing from the sig")
	}
	if got := dxcDupeSigFor("20260922", "other", "A", 0); got == base {
		t.Error("contest is missing from the sig")
	}
}

// TestRefreshQSOSInvalidatesDXCDupeCache verifies logging (and every other
// mutation flowing through refreshQSOS) bumps the dupe revision and clears
// the cached signature, so the path line re-fetches instead of showing
// stale dupe markers.
func TestRefreshQSOSInvalidatesDXCDupeCache(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.dxc.dupeSet = map[string]bool{"X": true}
	m.rc.dxcDupeSig = "20260922||test|0"

	m.refreshQSOS()

	if m.dxc.dupeGen == 0 {
		t.Error("dupeGen should be bumped by refreshQSOS")
	}
	if m.rc.dxcDupeSig != "" {
		t.Errorf("cached dupe sig not cleared: %q", m.rc.dxcDupeSig)
	}
}

// TestDXCPathSpotFallbackExpires verifies the DB spot fallback is
// time-dependent: an entry older than the TTL triggers a re-fetch instead of
// serving spots that have aged out.
func TestDXCPathSpotFallbackExpires(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.fields[fieldFreq].SetValue("145.550")
	m.dxc.cachedRaw = nil // empty in-memory cache → fallback path

	m.rc.dxcSpotsBand = "2m"
	m.rc.dxcSpots = []store.DXCSpot{{DXCall: "SP9MOA", Frequency: 145540, ReceivedAt: time.Now().Unix()}}

	// Fresh fallback: served without a re-fetch.
	m.rc.dxcSpotsAt = time.Now()
	m.rc.dxcSpotsNeedFetch = false
	m.rc.dxcPathSig = ""
	_ = m.dxcPathLine(120)
	if m.rc.dxcSpotsNeedFetch {
		t.Error("fresh fallback should not trigger a re-fetch")
	}

	// Expired fallback: a re-fetch is requested.
	m.rc.dxcSpotsAt = time.Now().Add(-3 * time.Minute)
	m.rc.dxcSpotsNeedFetch = false
	m.rc.dxcPathSig = ""
	_ = m.dxcPathLine(120)
	if !m.rc.dxcSpotsNeedFetch {
		t.Error("expired fallback should trigger a re-fetch")
	}
	if m.rc.dxcSpotsFetchBand != "2m" {
		t.Errorf("fetch band = %q, want 2m", m.rc.dxcSpotsFetchBand)
	}
}
