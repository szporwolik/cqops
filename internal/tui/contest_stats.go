package tui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/store"
)

type contestStats struct {
	Name       string
	LastQSO    time.Time
	FirstQSO   time.Time
	TotalQSOs  int
	ThisHour   int
	Last60Min  int
	Rate10     int
	Rate100    int
	Best1Min   int
	Best10Min  int
	Best60Min  int
	AvgSession int
	OnAir      time.Duration
	Session    time.Duration

	MinuteBuckets []float64

	computedAt  time.Time
	contestID   string
	cachedSig   string
	cachedPanel string

	// History cache. The whole contest is read once per change, not once per
	// tick: times holds the on-air timestamps newest-first (24 bytes each, no
	// QSO structs), timesRev is the model revision it was read at, and the
	// clock-dependent numbers (last 60 minutes, current hour, the minute
	// chart) are recomputed from it on every tick without touching the
	// database.
	times     []time.Time
	timesRev  uint64
	fetchedAt time.Time
	loaded    bool

	// now is the clock seam for tests (nil means time.Now). The window
	// counters compare against the wall clock, so a test that pins the QSO
	// times must pin the clock too — otherwise it flips whenever the run
	// happens to start within the first minutes of an hour.
	now func() time.Time
}

const (
	// contestStatsRefresh is how often the clock-dependent numbers are
	// refreshed: the last-60-minutes window, the current hour and the chart
	// all slide with the wall clock even when the log does not change.
	contestStatsRefresh = 5 * time.Second
	// contestStatsMaxAge bounds how long the cached history may be trusted
	// when nothing reported a change. Every UI write path bumps the model's
	// qsoRev, which refreshes the panel immediately; this only covers a write
	// that bypassed the model (a remote refresh merging a corrected time, an
	// external tool editing the database) and does it without walking the
	// contest every five seconds.
	contestStatsMaxAge = 60 * time.Second
)

// computeIfStale refreshes the panel numbers if the 5-second window has passed.
// rev is the model's QSO revision: the expensive part — reading the contest's
// whole history and deriving session, peaks and averages — runs only when it
// changed (or the cache expired), never once per tick.
func (cs *contestStats) computeIfStale(db *sql.DB, contestID string, contestName string, rev uint64) bool {
	now := time.Now().UTC()
	if cs.now != nil {
		now = cs.now().UTC()
	}
	if cs.contestID == contestID && now.Sub(cs.computedAt) < contestStatsRefresh {
		return false
	}
	switching := cs.contestID != contestID
	cs.contestID = contestID
	cs.Name = contestName
	cs.computedAt = now
	cs.cachedPanel = ""

	if switching || !cs.loaded || rev != cs.timesRev || now.Sub(cs.fetchedAt) >= contestStatsMaxAge {
		cs.reload(db, contestID, rev, now)
	}
	// Cheap: a prefix of the cached slice, nothing older than an hour can
	// enter these numbers.
	cs.computeWindows(now)
	return true
}

// reload reads the contest history and re-derives every value that depends on
// it alone. Clock-dependent numbers are left to computeWindows.
func (cs *contestStats) reload(db *sql.DB, contestID string, rev uint64, now time.Time) {
	cs.loaded = true
	cs.fetchedAt = now
	cs.timesRev = rev

	// Reset every derived value before recomputing. The struct is reused
	// across ticks and contest switches, and the window counters used to be
	// incremented in place: "Count 60m" and "hr" grew by the current window
	// on every 5-second refresh (4 QSOs logged for 9 minutes read as 354),
	// and an empty contest kept the previous contest's numbers.
	cs.times = nil
	cs.TotalQSOs = 0
	cs.ThisHour = 0
	cs.Last60Min = 0
	cs.Rate10 = 0
	cs.Rate100 = 0
	cs.Best1Min = 0
	cs.Best10Min = 0
	cs.Best60Min = 0
	cs.AvgSession = 0
	cs.OnAir = 0
	cs.Session = 0
	cs.LastQSO = time.Time{}
	cs.FirstQSO = time.Time{}
	cs.MinuteBuckets = nil

	times, err := store.ListContestQSOStamps(db, contestID)
	if err != nil || len(times) == 0 {
		applog.Debug("contest stats: no QSOs", "err", err)
		return
	}
	cs.times = times

	// Rates are measured on the QSO's on-air time (qso_date + time_on), not on
	// created_at: after an ADIF import created_at is the import time, which
	// made "last 60m" report the whole import as a burst.
	cs.TotalQSOs = len(times)
	cs.LastQSO = times[0]
	cs.FirstQSO = times[len(times)-1]
	cs.Session = cs.LastQSO.Sub(cs.FirstQSO)

	cs.Rate10 = cs.qsoRate(times, 10)
	cs.Rate100 = cs.qsoRate(times, 100)
	cs.Best1Min = cs.bestRate(times, 1*time.Minute)
	cs.Best10Min = cs.bestRate(times, 10*time.Minute)
	cs.Best60Min = cs.bestRate(times, 60*time.Minute)
	cs.OnAir = cs.computeOnAir(times)

	// The average is taken over on-air time when the operator took breaks
	// (gaps > 30 min), which is the rate an operator actually sustained;
	// the wall-clock span is used only when there are too few contacts to
	// measure a gap.
	base := cs.Session
	if cs.OnAir > 0 {
		base = cs.OnAir
	}
	if base.Seconds() > 0 {
		cs.AvgSession = int(float64(len(times)) / base.Hours())
	}
}

// computeWindows recalculates the numbers that slide with the clock. times is
// newest-first, so the loops stop at the first contact older than the window.
func (cs *contestStats) computeWindows(now time.Time) {
	hourStart := now.Truncate(time.Hour)
	cutoff60 := now.Add(-60 * time.Minute)
	cs.ThisHour = 0
	cs.Last60Min = 0
	for _, t := range cs.times {
		if !t.After(cutoff60) {
			break
		}
		cs.Last60Min++
		if t.After(hourStart) {
			cs.ThisHour++
		}
	}
	cs.MinuteBuckets = nil
	if len(cs.times) == 0 {
		return
	}
	// The chart is drawn whenever the contest has contacts, even when none of
	// them is recent enough to land in a bar: an empty hour is information.
	const minutes = 60
	buckets := make([]float64, minutes)
	bucketStart := now.Add(-time.Duration(minutes) * time.Minute)
	for _, t := range cs.times {
		if t.Before(bucketStart) {
			break
		}
		idx := int(t.Sub(bucketStart).Minutes())
		if idx >= 0 && idx < minutes {
			buckets[idx]++
		}
	}
	cs.MinuteBuckets = buckets
}

func (cs *contestStats) qsoRate(times []time.Time, n int) int {
	if len(times) < 2 || n < 2 {
		return 0
	}
	count := n
	if count > len(times) {
		count = len(times)
	}
	first := times[count-1]
	last := times[0]
	span := last.Sub(first)
	if span <= 0 {
		return 0
	}
	return int(float64(count-1) / span.Hours())
}

// bestRate returns the highest hourly rate seen in any window-long stretch of
// contacts. times is newest-first, so the window ending at each contact grows
// monotonically: one sweep with a single moving end index instead of walking
// every contact's window again (that nested scan was quadratic on a dense
// contest and dominated the panel's cost).
func (cs *contestStats) bestRate(times []time.Time, window time.Duration) int {
	if len(times) < 2 || window <= 0 {
		return 0
	}
	best, end := 0, 0
	for start := 0; start < len(times); start++ {
		if end < start+1 {
			end = start + 1
		}
		limit := times[start].Add(-window)
		for end < len(times) && times[end].After(limit) {
			end++
		}
		if n := end - start; n > best {
			best = n
		}
	}
	return int(float64(best) / window.Hours())
}

func (cs *contestStats) computeOnAir(times []time.Time) time.Duration {
	if len(times) < 2 {
		return 0
	}
	const offThreshold = 30 * time.Minute
	var total time.Duration
	for i := 0; i < len(times)-1; i++ {
		gap := times[i].Sub(times[i+1])
		if gap < offThreshold {
			total += gap
		}
	}
	return total
}

func formatDurationShort(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMin := int(d.Minutes())
	if totalMin < 1 {
		return "0"
	}
	if totalMin < 60 {
		return strconv.Itoa(totalMin)
	}
	totalH := int(d.Hours())
	m := totalMin % 60
	return fmt.Sprintf("%d:%02d", totalH, m)
}

func aggregateBuckets(buckets []float64, target int) []float64 {
	if target <= 0 || len(buckets) == 0 {
		return nil
	}
	if target >= len(buckets) {
		return buckets
	}
	perBucket := (len(buckets) + target - 1) / target
	out := make([]float64, target)
	for i := range out {
		start := i * perBucket
		end := start + perBucket
		if end > len(buckets) {
			end = len(buckets)
		}
		for j := start; j < end; j++ {
			out[i] += buckets[j]
		}
	}
	return out
}

func (m *Model) renderContestPanel() string {
	contestID := m.App.Logbook.ActiveContest
	if contestID == "" {
		return ""
	}
	ct, ok := m.App.Config.Contests[contestID]
	if !ok {
		return ""
	}

	m.contest.computeIfStale(m.App.DB, contestID, ct.Name, m.qsoRev)

	// Cache key — avoids rebuilding the panel string at 60fps when
	// the underlying data only changes every 5 seconds.
	var sigB strings.Builder
	fmt.Fprintf(&sigB, "%s|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|%d|",
		contestID,
		m.contest.TotalQSOs, m.contest.ThisHour, m.contest.Last60Min,
		m.contest.Rate10, m.contest.Rate100,
		m.contest.Best1Min, m.contest.Best10Min, m.contest.Best60Min,
		m.contest.AvgSession,
		int(m.contest.Session.Seconds()), int(m.contest.OnAir.Seconds()),
	)
	if len(m.contest.MinuteBuckets) > 0 {
		// Hash the shape as well as the sum: as time passes the same QSOs
		// slide towards the left of the chart while the sum stays the same, so
		// a sum-only signature froze the bars until another QSO arrived.
		sum := 0.0
		var h uint64 = 1469598103934665603
		for _, b := range m.contest.MinuteBuckets {
			sum += b
			h ^= uint64(b)
			h *= 1099511628211
		}
		fmt.Fprintf(&sigB, "%d:%.0f:%x", len(m.contest.MinuteBuckets), sum, h)
	}
	sig := sigB.String()
	if m.contest.cachedPanel != "" && m.contest.cachedSig == sig {
		return m.contest.cachedPanel
	}
	m.contest.cachedSig = sig

	const chartW = 30

	v := ValueStyle.Render
	d := DimStyle.Render
	rows := make([]string, 0, 12) // 4 data + caption + 4 chart + footer

	pad := func(s string) string {
		n := len(s)
		if n < 6 {
			return s + strings.Repeat(" ", 6-n)
		}
		return s
	}

	// --- Rate row ---
	rate100 := fmt.Sprintf("%4d/h", m.contest.Rate100)
	if m.contest.TotalQSOs < 100 {
		rate100 = "--/h"
	}
	rows = append(rows, d(pad("Rate"))+v(fmt.Sprintf("%4d/h  %5s", m.contest.Rate10, rate100)))

	// --- Count row ---
	// Both windows are counted on on-air time, so they are equal whenever the
	// current hour is younger than 60 minutes — that is expected, not a bug.
	rows = append(rows, d(pad("Count"))+v(fmt.Sprintf("Last60m %3d  Hour %3d",
		m.contest.Last60Min, m.contest.ThisHour)))

	// --- Peak row ---
	// bestRate extrapolates each window to QSOs per hour (one QSO inside the
	// last minute is 60/h), so the label carries the unit.
	rows = append(rows, d(pad("Peak/h"))+v(fmt.Sprintf("1m%3d 10m%3d 60m%3d",
		m.contest.Best1Min, m.contest.Best10Min, m.contest.Best60Min)))

	// --- Avg row ---
	// The average is measured over on-air time (gaps over 30 minutes excluded),
	// so the row shows that time instead of the wall-clock span: the panel is a
	// fixed 30 columns wide and both durations do not fit. The fixed-width
	// duration keeps every line the same length as time passes, so the panel
	// never jitters.
	rows = append(rows, d(pad("Avg"))+v(fmt.Sprintf("%4d/h  On %5s",
		m.contest.AvgSession, formatDurationShort(m.contest.OnAir))))

	// --- Chart ---
	if len(m.contest.MinuteBuckets) > 0 {
		data := aggregateBuckets(m.contest.MinuteBuckets, chartW)
		maxVal := 0.0
		for _, val := range data {
			if val > maxVal {
				maxVal = val
			}
		}
		if maxVal < 1 {
			maxVal = 1
		}
		chartH := 4
		// aggregateBuckets folds the 60 minute buckets into chartW bars, so
		// each bar covers several minutes — say so instead of "QSO/min".
		bucketMin := 1
		if len(data) > 0 && len(m.contest.MinuteBuckets) > len(data) {
			bucketMin = (len(m.contest.MinuteBuckets) + len(data) - 1) / len(data)
		}
		rows = append(rows, d(fmt.Sprintf("QSO/%dm  last 60m  max %3.0f", bucketMin, maxVal)))
		for row := chartH - 1; row >= 0; row-- {
			line := make([]rune, 0, len(data))
			for _, val := range data {
				h := int(val * float64(chartH) / maxVal)
				if h > chartH {
					h = chartH
				}
				if h > row {
					line = append(line, '\u2588')
				} else {
					line = append(line, ' ')
				}
			}
			rows = append(rows, string(line))
		}
		fpad := len(data) - 7 // len("-60m") + len("now") = 4 + 3
		if fpad < 0 {
			fpad = 0
		}
		rows = append(rows, fmt.Sprintf("-60m%snow", strings.Repeat(" ", fpad)))
	}

	m.contest.cachedPanel = contestBorderBoxStyle.Width(solarBoxW).Padding(0, 2).Render(strings.Join(rows, "\n"))
	return m.contest.cachedPanel
}
