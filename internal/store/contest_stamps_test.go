package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// TestListContestQSOStampsReturnsSortedTimes covers the contract the contest
// panel depends on: bare times (no QSO struct per contact), newest first, with
// created_at used only for a row whose ADIF date/time is unusable.
func TestListContestQSOStampsReturnsSortedTimes(t *testing.T) {
	db := newTempDB(t)
	for _, at := range []string{"20260501120000", "20260501123000", "20260501121500"} {
		mustInsertQSO(t, db, &qso.QSO{
			Call: "SP9MOA", QSODate: at[:8], TimeOn: at[8:],
			Band: "20m", Mode: "CW", ContestID: "c1",
		})
	}
	// A different contest must not leak in.
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260501", TimeOn: "140000",
		Band: "40m", Mode: "CW", ContestID: "c2",
	})

	times, err := ListContestQSOStamps(db, "c1")
	if err != nil {
		t.Fatalf("ListContestQSOStamps: %v", err)
	}
	if len(times) != 3 {
		t.Fatalf("got %d stamps, want 3", len(times))
	}
	want := []string{"2026-05-01T12:30:00Z", "2026-05-01T12:15:00Z", "2026-05-01T12:00:00Z"}
	for i, w := range want {
		if got := times[i].Format(time.RFC3339); got != w {
			t.Errorf("stamp %d = %s, want %s (newest first)", i, got, w)
		}
	}

	if times, err := ListContestQSOStamps(db, ""); err != nil || times != nil {
		t.Errorf("empty contest id: got %v, %v; want nil, nil", times, err)
	}
}

// TestListContestQSOStampsFallsBackToCreatedAt: a row with a broken ADIF date
// still contributes its created_at, which is what the old []qso.QSO hand-off
// did through qsoTimeOf.
//
// The malformed row has a valid QSO_DATE and a broken TIME_ON: that is the case
// time.Date silently normalises (12:60:00 → 13:00:00), so it has to land on the
// created_at fallback instead of inventing a moment that would skew rate, peak
// and session numbers.
func TestListContestQSOStampsFallsBackToCreatedAt(t *testing.T) {
	db := newTempDB(t)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20260501", TimeOn: "120000",
		Band: "20m", Mode: "CW", ContestID: "c1",
	})
	id := insertRawContestRow(t, db, "20260501", "126000")
	if _, err := db.Exec(`UPDATE qsos SET created_at = ? WHERE id = ?`,
		"2026-05-01T18:00:00Z", id); err != nil {
		t.Fatalf("set created_at: %v", err)
	}
	// A row with no usable date at all falls back the same way.
	id = insertRawContestRow(t, db, "", "")
	if _, err := db.Exec(`UPDATE qsos SET created_at = ? WHERE id = ?`,
		"2026-05-01T20:00:00Z", id); err != nil {
		t.Fatalf("set created_at: %v", err)
	}

	times, err := ListContestQSOStamps(db, "c1")
	if err != nil {
		t.Fatalf("ListContestQSOStamps: %v", err)
	}
	if len(times) != 3 {
		t.Fatalf("got %d stamps, want 3", len(times))
	}
	want := []string{"2026-05-01T20:00:00Z", "2026-05-01T18:00:00Z", "2026-05-01T12:00:00Z"}
	for i, w := range want {
		if got := times[i].Format(time.RFC3339); got != w {
			t.Errorf("stamp %d = %s, want %s", i, got, w)
		}
	}
	for _, got := range times {
		if got.Format("15:04") == "13:00" {
			t.Fatalf("a broken 12:60:00 TIME_ON produced 13:00 instead of the created_at fallback")
		}
	}
}

// insertRawContestRow writes a contest row through raw SQL so a test can store
// a malformed date the normal insert paths would reject.
func insertRawContestRow(t *testing.T, db *sql.DB, date, timeOn string) int64 {
	t.Helper()
	res, err := db.Exec(
		`INSERT INTO qsos (call, base_call, qso_date, time_on, band, mode, contest_id, created_at, updated_at)
		 VALUES ('SP9MOA', 'SP9MOA', ?, ?, '20m', 'CW', 'c1', datetime('now'), datetime('now'))`,
		date, timeOn)
	if err != nil {
		t.Fatalf("insert raw row: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}
