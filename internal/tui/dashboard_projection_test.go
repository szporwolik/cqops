package tui

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Dashboard payload — projection vs full record
//
// The Today and Recent panels used to build their payload from complete QSO
// records and now read a fourteen-column projection of the same rows. These
// tests render both and compare the JSON the browser receives, so a field that
// the panels show but the projection forgot cannot slip through silently.
// =============================================================================

// projectionOf maps a full record the way the old loader fed the panels.
func projectionOf(q qso.QSO) store.DashboardQSO {
	return store.DashboardQSO{
		ID:           q.ID,
		Call:         q.Call,
		QSODate:      q.QSODate,
		TimeOn:       q.TimeOn,
		Band:         q.Band,
		Freq:         q.Freq,
		Mode:         q.Mode,
		Submode:      q.Submode,
		RSTSent:      q.RSTSent,
		RSTRcvd:      q.RSTRcvd,
		GridSquare:   q.GridSquare,
		Country:      q.Country,
		Operator:     q.Operator,
		MyGridSquare: q.MyGridSquare,
	}
}

// payloadJSON renders views exactly as the SSE event carries them.
func payloadJSON(t *testing.T, views []dashboard.QSOView) string {
	t.Helper()
	raw, err := json.Marshal(views)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return string(raw)
}

// seedPanelContacts inserts contacts that exercise every rendered field: ties
// inside one minute (ordering), a contact with no frequency, an FT8 submode, a
// six-character grid with a separate "my grid", and one row that is enriched
// after the save.
func seedPanelContacts(t *testing.T, m *Model) []int64 {
	t.Helper()
	today := time.Now().UTC().Format("20060102")
	rows := []*qso.QSO{
		{Call: "SP9MOA", QSODate: today, TimeOn: "101500", Band: "20m", Freq: 14.0745,
			Mode: "MFSK", Submode: "FT8", RSTSent: "-10", RSTRcvd: "-12",
			GridSquare: "JO90WE", MyGridSquare: "KO00AA", Country: "Poland", Operator: "SP9MOA"},
		{Call: "DJ7NT", QSODate: today, TimeOn: "120000", Band: "40m", Freq: 7.047,
			Mode: "MFSK", Submode: "FT4", RSTSent: "-05", RSTRcvd: "-07",
			GridSquare: "JO31", MyGridSquare: "KO00AA", Country: "Fed. Rep. of Germany"},
		{Call: "K1ABC", QSODate: today, TimeOn: "130000", Band: "15m", Freq: 21.074,
			Mode: "CW", RSTSent: "599", RSTRcvd: "579", GridSquare: "FN42"},
		// Same minute as K1ABC: newest-first must fall through to the id.
		{Call: "K2DEF", QSODate: today, TimeOn: "130000", Band: "15m",
			Mode: "SSB", Submode: "USB", RSTSent: "59", RSTRcvd: "57", GridSquare: "FN30"},
		// No frequency at all.
		{Call: "SP9SPM", QSODate: today, TimeOn: "140000", Band: "80m",
			Mode: "SSB", RSTSent: "59", RSTRcvd: "59", GridSquare: "JO90"},
	}
	ids := make([]int64, 0, len(rows))
	for _, q := range rows {
		id, err := store.InsertQSO(m.App.DB, q)
		if err != nil {
			t.Fatalf("InsertQSO(%s): %v", q.Call, err)
		}
		ids = append(ids, id)
	}
	// Enrichment after the save: the panels repaint on it, so the projection
	// has to see the filled-in country. The grid is already set and enrichment
	// is fill-only, so it stays as stored.
	if err := store.UpdateQSOEnrichment(m.App.DB, ids[2], store.EnrichmentData{
		Country: "United States", GridSquare: "FN42AA",
	}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}
	return ids
}

// oldPanelViews is the payload the panels produced before the projection: the
// general-purpose loader plus the same rendering.
func oldPanelViews(t *testing.T, m *Model, date string, limit int) []dashboard.QSOView {
	t.Helper()
	full, err := store.ListQSOsFromDate(m.App.DB, date, limit)
	if err != nil {
		t.Fatalf("ListQSOsFromDate: %v", err)
	}
	proj := make([]store.DashboardQSO, 0, len(full))
	for _, q := range full {
		proj = append(proj, projectionOf(q))
	}
	return dashboardViews(proj)
}

// TestDashboardTodayPayloadIsUnchangedByTheProjection is the before/after gate
// for the Today panel: same contacts, same order, same every field.
func TestDashboardTodayPayloadIsUnchangedByTheProjection(t *testing.T) {
	m := dashboardStatsTestModel(t)
	seedPanelContacts(t, m)

	m.forcePushDashboardToday(m.http.client.State())
	got := m.http.client.State().Snapshot().Today
	if len(got) != 5 {
		t.Fatalf("Today holds %d contacts, want 5", len(got))
	}

	today := time.Now().UTC().Format("20060102")
	want := oldPanelViews(t, m, today, 5000)
	if payloadJSON(t, got) != payloadJSON(t, want) {
		t.Errorf("Today payload changed with the projection:\n got %s\nwant %s",
			payloadJSON(t, got), payloadJSON(t, want))
	}

	// The specific fields the ticket called out, spelled out so a failure says
	// which one drifted.
	byCall := map[string]dashboard.QSOView{}
	for _, v := range got {
		byCall[v.Call] = v
	}
	if v := byCall["SP9MOA"]; v.Submode != "FT8" || v.Frequency != "14.075 MHz" ||
		v.FrequencyHz != 14074500 || v.MyGrid != "KO00AA" || !v.TimeUTC.Equal(
		mustParseADIF(t, today+"101500")) {
		t.Errorf("FT8 contact wrong: %+v", v)
	}
	if v := byCall["SP9SPM"]; v.Frequency != "0.000 MHz" || v.FrequencyHz != 0 {
		t.Errorf("contact without a frequency: %+v", v)
	}
	// Enrichment fills the country; the grid it already had is left alone.
	if v := byCall["K1ABC"]; v.Country != "United States" || v.Grid != "FN42" {
		t.Errorf("enriched contact wrong: %+v", v)
	}
	// Newest first, ties broken by id descending.
	wantOrder := []string{"SP9SPM", "K2DEF", "K1ABC", "DJ7NT", "SP9MOA"}
	for i, call := range wantOrder {
		if got[i].Call != call {
			t.Errorf("row %d is %s, want %s", i, got[i].Call, call)
		}
	}
}

// TestDashboardRecentPayloadIsUnchangedByTheProjection does the same for the
// Recent panel, which loads its own rows (limit 20, plus the midnight guard).
func TestDashboardRecentPayloadIsUnchangedByTheProjection(t *testing.T) {
	m := dashboardStatsTestModel(t)
	seedPanelContacts(t, m)

	m.forcePushDashboardRecent(m.http.client.State())
	got := m.http.client.State().Snapshot().Recent
	if len(got) != 5 {
		t.Fatalf("Recent holds %d contacts, want 5", len(got))
	}
	today := time.Now().UTC().Format("20060102")
	if want := oldPanelViews(t, m, today, 5); payloadJSON(t, got) != payloadJSON(t, want) {
		t.Errorf("Recent payload changed with the projection:\n got %s\nwant %s",
			payloadJSON(t, got), payloadJSON(t, want))
	}
}

// TestDashboardPanelsAreEmptyWithoutContacts keeps the empty state intact: no
// rows, no error, the panel cleared.
func TestDashboardPanelsAreEmptyWithoutContacts(t *testing.T) {
	m := dashboardStatsTestModel(t)

	m.forcePushDashboardToday(m.http.client.State())
	m.forcePushDashboardRecent(m.http.client.State())

	if got := m.http.client.State().Snapshot().Today; len(got) != 0 {
		t.Errorf("Today holds %d contacts on an empty logbook, want 0", len(got))
	}
	if got := m.http.client.State().Snapshot().Recent; len(got) != 0 {
		t.Errorf("Recent holds %d contacts on an empty logbook, want 0", len(got))
	}
}

// TestDashboardFingerprintSeesTheProjection pins the change detection to the
// same data the panels render: enrichment that fills the country on an
// unchanged id must repaint, and a second push with identical data must not.
func TestDashboardFingerprintSeesTheProjection(t *testing.T) {
	m := dashboardStatsTestModel(t)
	ids := seedPanelContacts(t, m)
	ds := m.http.client.State()

	m.forcePushDashboardRecent(ds)
	first := payloadJSON(t, ds.Snapshot().Recent)

	// Nothing changed: the fingerprint suppresses the push entirely.
	pushDashboardRecentRaw(m)
	if got := payloadJSON(t, ds.Snapshot().Recent); got != first {
		t.Errorf("a repeated push repainted Recent:\n got %s\nwant %s", got, first)
	}

	// Enrichment on an unchanged id has to repaint.
	if err := store.UpdateQSOEnrichment(m.App.DB, ids[4], store.EnrichmentData{Country: "Poland"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}
	pushDashboardRecentRaw(m)
	if got := ds.Snapshot().Recent; got[0].Country != "Poland" {
		t.Errorf("enrichment did not repaint Recent: %+v", got[0])
	}
}

// pushDashboardRecentRaw bypasses the force helper, keeping the fingerprint
// cache so the change detection itself is exercised.
func pushDashboardRecentRaw(m *Model) {
	m.pushDashboardRecent(m.http.client.State())
}

// mustParseADIF parses the ADIF date+time pair the panels render from.
func mustParseADIF(t *testing.T, stamp string) time.Time {
	t.Helper()
	parsed, err := time.Parse("20060102 150405", stamp[:8]+" "+stamp[8:])
	if err != nil {
		t.Fatalf("parse %q: %v", stamp, err)
	}
	return parsed
}
