package tui

import (
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

func makeQSO(t time.Time) qso.QSO {
	return qso.QSO{CreatedAt: t}
}

func TestQsoRate_Basic(t *testing.T) {
	now := time.Now().UTC()
	// 10 QSOs over 30 minutes = 20 QSOs/h
	qsos := make([]qso.QSO, 10)
	for i := range qsos {
		qsos[i] = makeQSO(now.Add(-time.Duration(i) * 3 * time.Minute))
	}
	cs := contestStats{}
	rate := cs.qsoRate(qsos, 10)
	// 9 gaps × 3 min = 27 min span → 9 QSOs / 0.45h ≈ 20/h
	if rate < 18 || rate > 22 {
		t.Errorf("qsoRate(10) = %d, want ~20", rate)
	}
}

func TestQsoRate_LessThanN(t *testing.T) {
	now := time.Now().UTC()
	qsos := []qso.QSO{makeQSO(now), makeQSO(now.Add(-1 * time.Minute))}
	cs := contestStats{}
	rate := cs.qsoRate(qsos, 10) // only 2 QSOs, n=10
	// 1 gap × 1 min = 1 QSO / 0.0167h ≈ 60/h
	if rate < 50 || rate > 70 {
		t.Errorf("qsoRate with < n QSOs = %d, want ~60", rate)
	}
}

func TestQsoRate_SingleQSO(t *testing.T) {
	qsos := []qso.QSO{makeQSO(time.Now().UTC())}
	cs := contestStats{}
	if rate := cs.qsoRate(qsos, 10); rate != 0 {
		t.Errorf("qsoRate with 1 QSO = %d, want 0", rate)
	}
}

func TestBestRate_SlidingWindow(t *testing.T) {
	now := time.Now().UTC()
	// 5 QSOs in 1 minute (burst of 300/h), then slow.
	qsos := []qso.QSO{
		makeQSO(now),
		makeQSO(now.Add(-12 * time.Second)),
		makeQSO(now.Add(-24 * time.Second)),
		makeQSO(now.Add(-36 * time.Second)),
		makeQSO(now.Add(-48 * time.Second)),
		makeQSO(now.Add(-10 * time.Minute)), // gap
	}
	cs := contestStats{}
	best := cs.bestRate(qsos, 1*time.Minute)
	if best < 200 {
		t.Errorf("bestRate 1m = %d, want >= 200 (5 QSOs in 48s)", best)
	}
}

func TestComputeOnAir_Basic(t *testing.T) {
	now := time.Now().UTC()
	// 3 QSOs, each 5 min apart → 10 min on-air.
	qsos := []qso.QSO{
		makeQSO(now),
		makeQSO(now.Add(-5 * time.Minute)),
		makeQSO(now.Add(-10 * time.Minute)),
	}
	cs := contestStats{}
	onAir := cs.computeOnAir(qsos)
	expected := 10 * time.Minute
	if onAir < expected-time.Second || onAir > expected+time.Second {
		t.Errorf("computeOnAir = %v, want %v", onAir, expected)
	}
}

func TestComputeOnAir_OffTimeGap(t *testing.T) {
	now := time.Now().UTC()
	// 2 QSOs close, then 31 min gap, then 1 QSO → gap > 30 min = off-time.
	qsos := []qso.QSO{
		makeQSO(now),
		makeQSO(now.Add(-2 * time.Minute)),  // 2 min gap = on-air
		makeQSO(now.Add(-33 * time.Minute)), // 31 min gap → off-time
		makeQSO(now.Add(-35 * time.Minute)), // 2 min gap = on-air
	}
	cs := contestStats{}
	onAir := cs.computeOnAir(qsos)
	expected := 4 * time.Minute // two 2-min gaps
	if onAir < expected-time.Second || onAir > expected+time.Second {
		t.Errorf("computeOnAir with off-time gap = %v, want %v", onAir, expected)
	}
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0"},
		{30 * time.Second, "0"},
		{5 * time.Minute, "5"},
		{90 * time.Minute, "1:30"},
		{25 * time.Hour, "25:00"},
	}
	for _, tt := range tests {
		got := formatDurationShort(tt.d)
		if got != tt.want {
			t.Errorf("formatDurationShort(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestApplyCap(t *testing.T) {
	caps := map[string]int{"Call": 12, "Pwr": 5}
	// Within cap.
	if s := applyCap(8, 3, "Call", caps); s != 3 {
		t.Errorf("applyCap(8,3,Call) = %d, want 3", s)
	}
	// Hits cap.
	if s := applyCap(10, 5, "Call", caps); s != 2 {
		t.Errorf("applyCap(10,5,Call) = %d, want 2", s)
	}
	// Already at cap.
	if s := applyCap(12, 3, "Call", caps); s != 0 {
		t.Errorf("applyCap(12,3,Call) = %d, want 0", s)
	}
	// Not in caps map.
	if s := applyCap(8, 10, "Name", caps); s != 10 {
		t.Errorf("applyCap(8,10,Name) = %d, want 10 (no cap)", s)
	}
}

func TestBucketQSOsByMinute(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Minute)
	qsos := []qso.QSO{
		{CreatedAt: now.Add(-5 * time.Minute)},
		{CreatedAt: now.Add(-5 * time.Minute)},
		{CreatedAt: now.Add(-1 * time.Minute)},
	}
	b := bucketQSOsByMinute(qsos, 60, now)
	if len(b) != 60 {
		t.Fatalf("bucketQSOsByMinute len = %d, want 60", len(b))
	}
	if b[55] != 2 {
		t.Errorf("bucket at 5 min ago = %.0f, want 2", b[55])
	}
	if b[59] != 1 {
		t.Errorf("bucket at 1 min ago = %.0f, want 1", b[59])
	}
	// Past/future should be zero.
	if b[0] != 0 {
		t.Errorf("oldest bucket = %.0f, want 0", b[0])
	}
}

func TestAggregateBuckets(t *testing.T) {
	in := []float64{1, 0, 2, 3, 1, 0}
	// Aggregate 6 → 3 buckets.
	out := aggregateBuckets(in, 3)
	if len(out) != 3 {
		t.Fatalf("aggregateBuckets len = %d, want 3", len(out))
	}
	if out[0] != 1 {
		t.Errorf("bucket 0 = %.0f", out[0])
	}
	if out[1] != 5 {
		t.Errorf("bucket 1 = %.0f", out[1])
	}
	if out[2] != 1 {
		t.Errorf("bucket 2 = %.0f", out[2])
	}

	// Target >= len returns original.
	out2 := aggregateBuckets(in, 10)
	if len(out2) != 6 {
		t.Errorf("aggregateBuckets(6, 10) len = %d, want 6", len(out2))
	}
}
