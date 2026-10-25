package tui

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

func newContestStatsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.InitDB(filepath.Join(t.TempDir(), "contest.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// insertContestQSO inserts one QSO of the given contest, logging it at the
// supplied time (created_at is what the rate panel measures).
func insertContestQSO(t *testing.T, db *sql.DB, contestID string, at time.Time) {
	t.Helper()
	id, err := store.InsertQSO(db, &qso.QSO{
		Call: "SP9MOA", QSODate: at.UTC().Format("20060102"), TimeOn: at.UTC().Format("150405"),
		Band: "20m", Mode: "CW", ContestID: contestID,
	})
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	if _, err := db.Exec(`UPDATE qsos SET created_at = ? WHERE id = ?`,
		at.UTC().Format(time.RFC3339), id); err != nil {
		t.Fatalf("set created_at: %v", err)
	}
}

// recompute forces the next computeIfStale to run again (the 5-second cache
// would otherwise keep the previous result).
func recompute(cs *contestStats, db *sql.DB, contestID string) {
	cs.computedAt = cs.computedAt.Add(-time.Minute)
	cs.computeIfStale(db, contestID, contestID)
}

// TestContestStatsCountersDoNotAccumulate is the regression test for the
// production reading "Count 60m 354  hr 354" on a contest holding 4 QSOs: the
// window counters were incremented in place on every 5-second refresh instead
// of being recalculated.
func TestContestStatsCountersDoNotAccumulate(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	for i := 0; i < 4; i++ {
		insertContestQSO(t, db, "c1", now.Add(-time.Duration(i)*2*time.Minute))
	}

	cs := &contestStats{}
	cs.computeIfStale(db, "c1", "Test")
	if cs.Last60Min != 4 || cs.ThisHour != 4 {
		t.Fatalf("first compute: 60m=%d hr=%d, want 4 and 4", cs.Last60Min, cs.ThisHour)
	}

	for i := 0; i < 5; i++ {
		recompute(cs, db, "c1")
		if cs.Last60Min != 4 || cs.ThisHour != 4 {
			t.Fatalf("after %d refreshes: 60m=%d hr=%d, want 4 and 4 (counters must be recalculated, not accumulated)",
				i+1, cs.Last60Min, cs.ThisHour)
		}
	}
}

// TestContestStatsClearsWhenContestIsEmpty: switching to a contest without
// contacts must not leave the previous contest's numbers on the panel.
func TestContestStatsClearsWhenContestIsEmpty(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		insertContestQSO(t, db, "c1", now.Add(-time.Duration(i)*time.Minute))
	}

	cs := &contestStats{}
	cs.computeIfStale(db, "c1", "One")
	if cs.TotalQSOs != 3 || cs.Last60Min != 3 {
		t.Fatalf("precondition: total=%d 60m=%d", cs.TotalQSOs, cs.Last60Min)
	}

	cs.computeIfStale(db, "c2", "Empty")
	if cs.TotalQSOs != 0 || cs.ThisHour != 0 || cs.Last60Min != 0 ||
		cs.Rate10 != 0 || cs.Best1Min != 0 || cs.AvgSession != 0 ||
		cs.Session != 0 || len(cs.MinuteBuckets) != 0 {
		t.Fatalf("stale values after switching to an empty contest: %+v", cs)
	}
}

// TestContestStatsAverageUsesTheWindow pins the panel reading "Avg 26/h,
// Sess 9" for four contacts spread over nine minutes.
func TestContestStatsAverageUsesTheWindow(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	for i := 0; i < 4; i++ {
		insertContestQSO(t, db, "c1", now.Add(-time.Duration(i)*3*time.Minute))
	}

	cs := &contestStats{}
	cs.computeIfStale(db, "c1", "Test")
	if got := formatDurationShort(cs.Session); got != "9" {
		t.Fatalf("session = %s, want 9 minutes", got)
	}
	if cs.AvgSession < 25 || cs.AvgSession > 27 {
		t.Errorf("AvgSession = %d/h, want ~26/h for 4 QSOs in 9 minutes", cs.AvgSession)
	}
	// Peak values are hourly rates: one QSO inside a minute is 60/h, four in
	// ten minutes is 24/h, four across the hour is 4/h.
	if cs.Best1Min != 60 || cs.Best10Min != 24 || cs.Best60Min != 4 {
		t.Errorf("peaks = %d/%d/%d, want 60/24/4 (QSO/h per window)",
			cs.Best1Min, cs.Best10Min, cs.Best60Min)
	}
}

// TestContestPanelLabelsUnits: the peak row is an hourly rate and the chart
// bars cover more than one minute, so both labels carry their unit.
func TestContestPanelLabelsUnits(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	for i := 0; i < 4; i++ {
		insertContestQSO(t, db, "c1", now.Add(-time.Duration(i)*3*time.Minute))
	}
	a := &app.App{
		DB:      db,
		Logbook: &config.Logbook{ActiveContest: "c1"},
		Config: &config.Config{Contests: map[string]config.Contest{
			"c1": {Name: "Test Contest", ContestID: "TEST-CW"},
		}},
	}
	m := &Model{App: a}
	panel := m.renderContestPanel()
	if panel == "" {
		t.Fatal("contest panel is empty")
	}
	if !strings.Contains(panel, "Peak/h") {
		t.Errorf("peak row must say it is a rate:\n%s", panel)
	}
	if strings.Contains(panel, "QSO/min") {
		t.Errorf("chart claims QSO/min while each bar covers two minutes:\n%s", panel)
	}
	if !strings.Contains(panel, "QSO/2m") {
		t.Errorf("chart bucket unit missing:\n%s", panel)
	}
	if !strings.Contains(panel, "Count") || !strings.Contains(panel, "60m") {
		t.Errorf("count row missing:\n%s", panel)
	}
}
