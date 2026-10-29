package tui

import (
	"testing"
	"time"
)

// TestContestStatsCacheAvoidsRereadingTheHistory covers the contract the panel
// now relies on: reading the whole contest and deriving session/peaks/averages
// happens once per QSO change, not once per tick. The clock-dependent numbers
// still move every tick, from the cached timestamps.
func TestContestStatsCacheAvoidsRereadingTheHistory(t *testing.T) {
	db := newContestStatsDB(t)
	base := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)
	insertContestQSO(t, db, "c1", base.Add(-30*time.Second))
	insertContestQSO(t, db, "c1", base.Add(-(59*time.Minute + 50*time.Second))) // leaves the hour window during the test

	cs := &contestStats{now: func() time.Time { return base }}
	cs.computeIfStale(db, "c1", "Test", 1)
	if cs.TotalQSOs != 2 || cs.Last60Min != 2 {
		t.Fatalf("first compute: total=%d 60m=%d, want 2 and 2", cs.TotalQSOs, cs.Last60Min)
	}
	fetched := cs.fetchedAt

	// Five ticks inside the age limit, nothing changed in the database. The
	// older contact slides out of the 60-minute window as the clock moves, so
	// the numbers change while the history stays cached.
	for i := 0; i < 5; i++ {
		base = base.Add(5 * time.Second)
		cs.computeIfStale(db, "c1", "Test", 1)
	}
	if !cs.fetchedAt.Equal(fetched) {
		t.Errorf("the contest history was re-read after %v without a change", cs.fetchedAt.Sub(fetched))
	}
	if cs.TotalQSOs != 2 {
		t.Errorf("total = %d, want 2 (the history must survive the sliding windows)", cs.TotalQSOs)
	}
	if cs.Last60Min != 1 {
		t.Errorf("60m = %d after the older contact aged out, want 1", cs.Last60Min)
	}

	// Past the cache age limit the history is re-read — that is the safety net
	// for writes that bypassed the model.
	base = base.Add(70 * time.Minute)
	cs.computeIfStale(db, "c1", "Test", 1)
	if !cs.fetchedAt.After(fetched) {
		t.Error("the cache did not expire")
	}
	if cs.Last60Min != 0 || cs.ThisHour != 0 {
		t.Errorf("after 70 minutes: 60m=%d hr=%d, want 0 and 0", cs.Last60Min, cs.ThisHour)
	}
	if cs.TotalQSOs != 2 {
		t.Errorf("total = %d, want 2", cs.TotalQSOs)
	}
	// The chart is drawn whenever the contest has contacts, even when the last
	// hour is empty: sixty zero buckets, not a hidden chart.
	if len(cs.MinuteBuckets) != 60 {
		t.Errorf("bucket count = %d, want 60 (an empty hour still renders the chart)", len(cs.MinuteBuckets))
	}
	for i, v := range cs.MinuteBuckets {
		if v != 0 {
			t.Errorf("bucket %d = %v, want 0 — no contact is inside the last hour", i, v)
			break
		}
	}

	// A new contact with a bumped revision is picked up on the next tick.
	insertContestQSO(t, db, "c1", base.Add(-10*time.Second))
	base = base.Add(5 * time.Second)
	cs.computeIfStale(db, "c1", "Test", 2)
	if cs.TotalQSOs != 3 || cs.Last60Min != 1 {
		t.Errorf("after a new QSO: total=%d 60m=%d, want 3 and 1", cs.TotalQSOs, cs.Last60Min)
	}
}

// TestContestStatsCacheExpires: a write that bypassed the model (an external
// tool, a remote refresh) must still be seen, so the cache has an age limit.
func TestContestStatsCacheExpires(t *testing.T) {
	db := newContestStatsDB(t)
	base := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)
	insertContestQSO(t, db, "c1", base.Add(-time.Minute))

	cs := &contestStats{now: func() time.Time { return base }}
	cs.computeIfStale(db, "c1", "Test", 1)

	insertContestQSO(t, db, "c1", base.Add(-2*time.Minute)) // no revision bump
	base = base.Add(contestStatsMaxAge)
	cs.computeIfStale(db, "c1", "Test", 1)
	if cs.TotalQSOs != 2 {
		t.Errorf("total = %d after the cache expired, want 2", cs.TotalQSOs)
	}
}

// TestContestStatsEmptyContestDoesNotRepoll covers the case the cache flag
// exists for: an empty contest has no timestamps at all, and a nil slice must
// not be mistaken for "not loaded yet" (that would query on every tick).
func TestContestStatsEmptyContestDoesNotRepoll(t *testing.T) {
	db := newContestStatsDB(t)
	base := time.Date(2026, 5, 1, 12, 30, 0, 0, time.UTC)

	cs := &contestStats{now: func() time.Time { return base }}
	cs.computeIfStale(db, "c1", "Test", 1)
	fetched := cs.fetchedAt
	for i := 0; i < 3; i++ {
		base = base.Add(5 * time.Second)
		cs.computeIfStale(db, "c1", "Test", 1)
	}
	if !cs.fetchedAt.Equal(fetched) {
		t.Error("an empty contest re-read the database on every tick")
	}
	if cs.TotalQSOs != 0 || len(cs.MinuteBuckets) != 0 {
		t.Errorf("empty contest has numbers: total=%d buckets=%d", cs.TotalQSOs, len(cs.MinuteBuckets))
	}
}

// TestBestRateMatchesTheNaiveSweep is the equivalence guard for the linear peak
// scan: the panel must report exactly the rates the per-contact sweep did.
func TestBestRateMatchesTheNaiveSweep(t *testing.T) {
	naive := func(times []time.Time, window time.Duration) int {
		if len(times) < 2 {
			return 0
		}
		best := 0
		for end := 0; end < len(times); end++ {
			startTime := times[end].Add(-window)
			count := 0
			for i := end; i < len(times); i++ {
				if times[i].After(startTime) {
					count++
				} else {
					break
				}
			}
			if rate := int(float64(count) / window.Hours()); rate > best {
				best = rate
			}
		}
		return best
	}

	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	offsets := []time.Duration{
		0, 10 * time.Second, 20 * time.Second, 45 * time.Second, // burst
		30 * time.Minute, 31 * time.Minute, 31*time.Minute + 20*time.Second, // gap, then a pair
		90 * time.Minute, 95 * time.Minute, 96 * time.Minute, 97 * time.Minute,
		200 * time.Minute, 201 * time.Minute, 202 * time.Minute, 203 * time.Minute, 204 * time.Minute,
	}
	times := make([]time.Time, len(offsets))
	for i, off := range offsets { // newest first
		times[i] = base.Add(-off)
	}
	cs := &contestStats{}
	for _, window := range []time.Duration{time.Minute, 5 * time.Minute, 10 * time.Minute, 60 * time.Minute, 4 * time.Hour} {
		got, want := cs.bestRate(times, window), naive(times, window)
		if got != want {
			t.Errorf("bestRate(%v) = %d, naive sweep = %d", window, got, want)
		}
	}
}

// TestBestRateDenseWindow checks the sweep on a dense log, where the old
// nested loop walked the whole window for every contact.
func TestBestRateDenseWindow(t *testing.T) {
	base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	const n = 2000
	times := make([]time.Time, n)
	for i := 0; i < n; i++ {
		times[i] = base.Add(-time.Duration(i) * 2 * time.Second) // one every 2 s
	}
	cs := &contestStats{}
	// 2000 contacts, 2 s apart: 30 contacts in a minute → 1800/h.
	if got := cs.bestRate(times, time.Minute); got != 1800 {
		t.Errorf("bestRate(1m) = %d, want 1800", got)
	}
	if got := cs.bestRate(times, 10*time.Minute); got != 1800 {
		t.Errorf("bestRate(10m) = %d, want 1800", got)
	}
}
