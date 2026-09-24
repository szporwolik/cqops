package tui

import (
	"testing"
)

// =============================================================================
// View() must not touch the database
// =============================================================================
//
// The QSO form renders on every keystroke, so a synchronous query here stalls
// the UI on slow storage. On a cache miss these functions must only record
// what they need; the query itself belongs in a tea.Cmd. These tests use a
// real (empty) SQLite database so a reintroduced query would actually run.

func TestFormPathRowDefersStatsQueryOutOfView(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.width = 120
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.rc.pathCall = "VK3A"

	_ = m.formPathRow(100)

	if !m.rc.logStatsNeedFetch {
		t.Error("formPathRow should flag an async stats fetch on a cache miss")
	}
	if m.rc.logStatsSig != "" {
		t.Errorf("formPathRow must not populate stats during View(); got sig %q "+
			"(a synchronous DB query was likely reintroduced)", m.rc.logStatsSig)
	}
	if m.rc.logStatsFetchCall != "VK3A" {
		t.Errorf("deferred fetch call = %q, want VK3A", m.rc.logStatsFetchCall)
	}
}

func TestDXCPathLineDefersSpotQueryOutOfView(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.width = 120
	m.fields[fieldFreq].SetValue("14.074") // 20m

	_ = m.dxcPathLine(120)

	if !m.rc.dxcSpotsNeedFetch {
		t.Error("dxcPathLine should flag an async spot fetch when the in-memory cache is empty")
	}
	if m.rc.dxcSpotsBand != "" {
		t.Errorf("dxcPathLine must not populate spots during View(); got band %q "+
			"(a synchronous DB query was likely reintroduced)", m.rc.dxcSpotsBand)
	}
	if m.rc.dxcSpotsFetchBand != "20m" {
		t.Errorf("deferred fetch band = %q, want 20m", m.rc.dxcSpotsFetchBand)
	}
}

// The dispatcher must consume each flag exactly once, otherwise a cache miss
// would re-queue the same query on every update.
func TestDispatchViewFetchesClearsFlags(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.rc.logStatsNeedFetch = true
	m.rc.logStatsFetchCall = "VK3A"
	m.rc.dxcSpotsNeedFetch = true
	m.rc.dxcSpotsFetchBand = "20m"
	m.rc.dxcDupeNeedFetch = true
	m.rc.dxcDupeFetchDate = "20260921"

	if cmd := m.dispatchViewFetches(nil); cmd == nil {
		t.Fatal("dispatchViewFetches returned no command for three pending fetches")
	}

	if m.rc.logStatsNeedFetch || m.rc.dxcSpotsNeedFetch || m.rc.dxcDupeNeedFetch {
		t.Error("dispatchViewFetches must clear every flag it services")
	}

	if cmd := m.dispatchViewFetches(nil); cmd != nil {
		t.Error("dispatchViewFetches should return nil when nothing is pending")
	}
}
