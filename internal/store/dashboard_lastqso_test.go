package store

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// TestDashboardStatsLastQSOIsOneContact pins the aggregate that feeds
// LastQSOAgoS across an event spanning several days. It used to be
// MAX(qso_date) || MAX(time_on) — two independent maxima — so a 23:00 contact
// on the first day plus a 01:00 contact on the second reported "23:00 on the
// second day", a moment no contact has and 22 hours into the future.
func TestDashboardStatsLastQSOIsOneContact(t *testing.T) {
	db := newTempDB(t)
	older := time.Date(2026, 5, 1, 23, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 5, 2, 1, 0, 0, 0, time.UTC)
	for _, at := range []time.Time{older, newer} {
		mustInsertQSO(t, db, &qso.QSO{
			Call: "SP9MOA", QSODate: at.Format("20060102"), TimeOn: at.Format("150405"),
			Band: "20m", Mode: "CW",
		})
	}

	s, err := GetDashboardStats(db, "20260501")
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	if s.QSOsToday != 2 {
		t.Fatalf("QSOsToday = %d, want 2", s.QSOsToday)
	}
	want := int(time.Since(newer).Seconds())
	if diff := s.LastQSOAgoS - want; diff < -5 || diff > 5 {
		t.Errorf("LastQSOAgoS = %d, want %d (the 01:00 contact); the combined-maximum answer would be %d, %d seconds off",
			s.LastQSOAgoS, want, want-22*3600, s.LastQSOAgoS-want)
	}

	// The event-window filter still applies: asking only about the second day
	// must report the contact from that day.
	s, err = GetDashboardStats(db, "20260502")
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	if s.QSOsToday != 1 {
		t.Fatalf("QSOsToday (second day) = %d, want 1", s.QSOsToday)
	}
	if diff := s.LastQSOAgoS - want; diff < -5 || diff > 5 {
		t.Errorf("filtered LastQSOAgoS = %d, want %d", s.LastQSOAgoS, want)
	}
}

// TestDashboardStatsLastQSOWithShortTime: a contact whose TIME_ON is HHMM (or
// missing) must still produce an elapsed time; the old 14-character parse
// silently left LastQSOAgoS at zero.
func TestDashboardStatsLastQSOWithShortTime(t *testing.T) {
	db := newTempDB(t)
	at := time.Date(2026, 5, 3, 14, 30, 0, 0, time.UTC)
	mustInsertQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: at.Format("20060102"), TimeOn: "1430",
		Band: "20m", Mode: "CW",
	})
	s, err := GetDashboardStats(db, "20260501")
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	want := int(time.Since(at).Seconds())
	if diff := s.LastQSOAgoS - want; diff < -5 || diff > 5 {
		t.Errorf("LastQSOAgoS = %d, want %d for an HHMM time", s.LastQSOAgoS, want)
	}
}
