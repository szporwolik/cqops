package tui

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

// insertStampQSO writes a contest row with an explicit on-air time and
// logging time, bypassing the worked index (the panel only reads timings).
func insertStampQSO(t *testing.T, db *sql.DB, contestID string, onAir, logged time.Time) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO qsos (call, qso_date, time_on, band, mode, contest_id, source, created_at, updated_at)
		 VALUES ('SP9MOA', ?, ?, '20m', 'CW', ?, 'manual', ?, ?)`,
		onAir.UTC().Format("20060102"), onAir.UTC().Format("150405"), contestID,
		logged.UTC().Format(time.RFC3339), logged.UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatalf("insert stamp: %v", err)
	}
}

// TestContestStatsUsesQSOtimeNotCreatedAt: the rates must follow when the
// contacts happened, not when the rows were written. An ADIF import logs a
// whole contest at once; keyed on created_at that read as one giant burst in
// the last minute.
func TestContestStatsUsesQSOtimeNotCreatedAt(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	day := 48 * time.Hour
	// Four contacts from two days ago, imported just now.
	for i := 0; i < 4; i++ {
		insertStampQSO(t, db, "c1", now.Add(-day-time.Duration(i)*3*time.Minute), now)
	}

	cs := &contestStats{}
	cs.computeIfStale(db, "c1", "Test", 0)
	if cs.TotalQSOs != 4 {
		t.Fatalf("TotalQSOs = %d, want 4", cs.TotalQSOs)
	}
	if cs.Last60Min != 0 || cs.ThisHour != 0 {
		t.Errorf("windows = %d/%d, want 0/0 (contacts are two days old, the import is not a rate)",
			cs.Last60Min, cs.ThisHour)
	}
	if got := formatDurationShort(cs.Session); got != "9" {
		t.Errorf("session = %s, want 9 minutes from the QSO times", got)
	}
	if len(cs.MinuteBuckets) != 60 {
		t.Fatalf("bucket count = %d", len(cs.MinuteBuckets))
	}
	for i, b := range cs.MinuteBuckets {
		if b != 0 {
			t.Errorf("bucket %d = %.0f, want 0 for contacts outside the last hour", i, b)
		}
	}
}

// TestContestStatsBeyondTheThousandRowWindow: session, peaks and windows used
// to be computed from at most the newest 1000 rows; they now cover the whole
// contest.
func TestContestStatsBeyondTheThousandRowWindow(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	const n = 1005
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	for i := 0; i < n; i++ {
		// One QSO every 10 seconds, newest first in time.
		at := now.Add(-time.Duration(i) * 10 * time.Second)
		if _, err := tx.Exec(
			`INSERT INTO qsos (call, qso_date, time_on, band, mode, contest_id, source, created_at, updated_at)
			 VALUES ('SP9MOA', ?, ?, '20m', 'CW', 'c1', 'manual', ?, ?)`,
			at.UTC().Format("20060102"), at.UTC().Format("150405"),
			at.UTC().Format(time.RFC3339), at.UTC().Format(time.RFC3339),
		); err != nil {
			tx.Rollback()
			t.Fatalf("insert %d: %v", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	cs := &contestStats{}
	cs.computeIfStale(db, "c1", "Test", 0)
	if cs.TotalQSOs != n {
		t.Fatalf("TotalQSOs = %d, want %d", cs.TotalQSOs, n)
	}
	wantFirst := now.Add(-time.Duration(n-1) * 10 * time.Second)
	if d := cs.FirstQSO.Sub(wantFirst); d > time.Second || d < -time.Second {
		t.Errorf("FirstQSO = %s, want %s (the row beyond the old 1000-row cap)",
			cs.FirstQSO.Format(time.RFC3339), wantFirst.Format(time.RFC3339))
	}
	// 1005 contacts 10 s apart ≈ 360 QSO/h sustained.
	if cs.Best60Min < 330 || cs.Best60Min > 390 {
		t.Errorf("Best60Min = %d/h, want ~360 for one QSO every 10 s", cs.Best60Min)
	}
}

// TestContestPanelNoLayoutShift: the panel keeps its size whatever the values
// are — same number of lines, same width on every line, so the form next to
// it never reflows when a counter changes.
func TestContestPanelNoLayoutShift(t *testing.T) {
	render := func(data func(db *sql.DB, now time.Time)) []string {
		db := newContestStatsDB(t)
		data(db, time.Now().UTC())
		a := &app.App{
			DB:      db,
			Logbook: &config.Logbook{ActiveContest: "c1"},
			Config: &config.Config{Contests: map[string]config.Contest{
				"c1": {Name: "CQ WPX CW", ContestID: "CQ-WPX-CW"},
			}},
		}
		m := &Model{App: a}
		return strings.Split(m.renderContestPanel(), "\n")
	}

	// A quiet four-contact session.
	small := render(func(db *sql.DB, now time.Time) {
		for i := 0; i < 4; i++ {
			insertStampQSO(t, db, "c1", now.Add(-time.Duration(i)*3*time.Minute), now)
		}
	})
	// A long contest with four-digit counters and hour-long durations.
	large := render(func(db *sql.DB, now time.Time) {
		for i := 0; i < 1500; i++ {
			insertStampQSO(t, db, "c1", now.Add(-time.Duration(i)*30*time.Second), now)
		}
	})

	if len(small) != len(large) {
		t.Fatalf("panel height changed with the data: %d vs %d lines", len(small), len(large))
	}
	// Two borders + four data rows + chart title + four bars + axis. A row
	// that outgrows the panel width wraps and adds a line, which is exactly
	// the layout shift this guards against.
	const wantLines = 12
	if len(small) != wantLines {
		t.Fatalf("panel height = %d lines, want %d:\n%s", len(small), wantLines, strings.Join(small, "\n"))
	}
	borderW := lipgloss.Width(small[0])
	for i := range small {
		w1, w2 := lipgloss.Width(small[i]), lipgloss.Width(large[i])
		if w1 != w2 {
			t.Errorf("line %d width changed with the data: %d vs %d\n%q\n%q",
				i, w1, w2, small[i], large[i])
		}
		if w1 != borderW {
			t.Errorf("line %d is %d wide, border is %d — the panel is not uniform:\n%q",
				i, w1, borderW, small[i])
		}
	}
}

// TestContestPanelShowsOnAir: the on-air time is part of the Avg row instead
// of being computed and thrown away, and it does not add a line.
func TestContestPanelShowsOnAir(t *testing.T) {
	db := newContestStatsDB(t)
	now := time.Now().UTC()
	// Three contacts 2 min apart, then a 40-minute break, then two more:
	// on-air (4+2 min) is well below the wall-clock span.
	for i := 0; i < 3; i++ {
		insertStampQSO(t, db, "c1", now.Add(-time.Duration(i)*2*time.Minute), now)
	}
	for i := 0; i < 2; i++ {
		insertStampQSO(t, db, "c1", now.Add(-40*time.Minute-time.Duration(i)*2*time.Minute), now)
	}
	a := &app.App{
		DB:      db,
		Logbook: &config.Logbook{ActiveContest: "c1"},
		Config: &config.Config{Contests: map[string]config.Contest{
			"c1": {Name: "CQ WPX CW"},
		}},
	}
	m := &Model{App: a}
	panel := m.renderContestPanel()

	if !strings.Contains(panel, "On ") {
		t.Fatalf("on-air time not rendered:\n%s", panel)
	}
	if m.contest.OnAir <= 0 || m.contest.OnAir >= m.contest.Session {
		t.Fatalf("OnAir = %v, Session = %v — expected a break to cut on-air below the span",
			m.contest.OnAir, m.contest.Session)
	}
	// The average follows the on-air time: 5 QSOs over 6 minutes of operating
	// (two 2-minute gaps either side of a 40-minute break).
	if m.contest.AvgSession < 45 || m.contest.AvgSession > 55 {
		t.Errorf("AvgSession = %d/h, want ~50/h over the on-air time", m.contest.AvgSession)
	}
}
