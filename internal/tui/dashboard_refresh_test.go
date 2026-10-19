package tui

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// resetDashboardFingerprints clears the package-level panel caches (they are
// globals shared by the whole model) so a test starts from "nothing pushed".
func resetDashboardFingerprints(t *testing.T) {
	t.Helper()
	lastTodayFP = panelFingerprint{}
	lastRecentFP = panelFingerprint{}
	t.Cleanup(func() {
		lastTodayFP = panelFingerprint{}
		lastRecentFP = panelFingerprint{}
	})
}

// TestDashboardTodayRepaintsAfterFieldOnlyChange is the regression test for
// the "same QSO has a country in Recent but none in Top QSOs" dashboard bug.
//
// WSJT-X auto-logs a QSO without a country, the dashboard pushes the Today
// list (Top QSOs + map + stats are built from it), and only then does the
// callbook/CTY enrichment fill the country via store.UpdateQSOEnrichment —
// which changes no row ids. The Today panel used to fingerprint rows by id
// alone, skip the push, and keep the countryless row forever, so a browser
// reload could not help: the server snapshot itself was stale. Recent was
// force-pushed after enrichment, hence the mismatch the operator saw.
func TestDashboardTodayRepaintsAfterFieldOnlyChange(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)
	// Include every QSO date in the today query.
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-01-01"

	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "DJ7NT", QSODate: "20260315", TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "JO31",
		RSTSent: "+05", RSTRcvd: "-11",
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardToday(ds)
	snap := ds.Snapshot()
	if len(snap.Today) != 1 {
		t.Fatalf("today panel has %d rows, want 1", len(snap.Today))
	}
	if got := snap.Today[0].Country; got != "" {
		t.Fatalf("precondition: today country = %q, want empty", got)
	}

	// Enrichment writes the country; the row id is unchanged.
	if err := store.UpdateQSOEnrichment(m.App.DB, id, store.EnrichmentData{
		Country: "Fed. Rep. of Germany",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}

	m.pushDashboardToday(ds)
	snap = ds.Snapshot()
	if got := snap.Today[0].Country; got != "Fed. Rep. of Germany" {
		t.Errorf("today panel kept stale country %q after enrichment — Top QSOs would show no country while Recent has it", got)
	}
}

// TestDashboardTodaySkipsUnchangedContent guards the other direction: the
// content fingerprint must still suppress redundant pushes when nothing
// changed (this is what keeps the periodic push cheap).
func TestDashboardTodaySkipsUnchangedContent(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)
	m.App.Config.Integrations.HTTPServer.EventStart = "2026-01-01"

	if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260315", TimeOn: "120000",
		Band: "40m", Mode: "SSB", GridSquare: "JO90",
	}); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardToday(ds)

	// Replace the panel with a marker; an unchanged re-push must not repaint.
	ds.SetToday([]dashboard.QSOView{{ID: "marker", Call: "MARKER"}})
	m.pushDashboardToday(ds)

	got := ds.Snapshot().Today
	if len(got) != 1 || got[0].Call != "MARKER" {
		t.Errorf("unchanged data still pushed (panel = %+v), fingerprint is not suppressing redundant work", got)
	}
}

// TestDashboardRecentRepaintsAfterFieldOnlyChange covers the Recent panel's
// half of the same bug, so both panels stay consistent.
func TestDashboardRecentRepaintsAfterFieldOnlyChange(t *testing.T) {
	resetDashboardFingerprints(t)
	m := newLifecycleTestModel(t)

	today := time.Now().UTC().Format("20060102")
	id, err := store.InsertQSO(m.App.DB, &qso.QSO{
		Call: "K1ABC", QSODate: today, TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "FN42",
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	ds := dashboard.NewState(dashboard.NewHub())
	m.pushDashboardRecent(ds)
	if len(ds.Snapshot().Recent) == 0 {
		t.Fatal("recent panel is empty, want the inserted QSO")
	}

	if err := store.UpdateQSOEnrichment(m.App.DB, id, store.EnrichmentData{
		Country: "United States",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}

	m.pushDashboardRecent(ds)
	recent := ds.Snapshot().Recent
	if recent[0].Country != "United States" {
		t.Errorf("recent panel country = %q after enrichment, want United States", recent[0].Country)
	}
}

// TestPanelFingerprintTracksRenderedFields documents which changes must
// invalidate a panel and which must not.
func TestPanelFingerprintTracksRenderedFields(t *testing.T) {
	base := []qso.QSO{{ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: ""}}
	fp := newPanelFingerprint(base)
	if fp.equal(panelFingerprint{}) {
		t.Error("an unset fingerprint must never match (forced push spelling)")
	}
	if !fp.equal(newPanelFingerprint([]qso.QSO{{ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: ""}})) {
		t.Error("identical data must produce an equal fingerprint")
	}

	changes := map[string]qso.QSO{
		"country": {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", Country: "Poland"},
		"grid":    {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8", GridSquare: "JO90"},
		"mode":    {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "SSB"},
		"operator": {ID: 7, Call: "DJ7NT", Band: "20m", Mode: "FT8",
			Operator: "SP9MOA"},
	}
	for name, changed := range changes {
		if fp.equal(newPanelFingerprint([]qso.QSO{changed})) {
			t.Errorf("%s change did not invalidate the fingerprint", name)
		}
	}

	if fp.equal(newPanelFingerprint([]qso.QSO{{ID: 8, Call: "DJ7NT", Band: "20m", Mode: "FT8"}})) {
		t.Error("a different row id did not invalidate the fingerprint")
	}
}
