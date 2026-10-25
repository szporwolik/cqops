package tui

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
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
}

func (cs *contestStats) computeIfStale(db *sql.DB, contestID string, contestName string) bool {
	now := time.Now().UTC()
	if cs.contestID == contestID && now.Sub(cs.computedAt) < 5*time.Second {
		return false
	}
	cs.contestID = contestID
	cs.Name = contestName
	cs.computedAt = now
	cs.cachedPanel = ""

	// Reset every derived value before recomputing. The struct is reused
	// across ticks and contest switches, and the window counters used to be
	// incremented in place: "Count 60m" and "hr" grew by the current window
	// on every 5-second refresh (4 QSOs logged for 9 minutes read as 354),
	// and an empty contest kept the previous contest's numbers.
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

	qsos, err := store.ListQSOs(db, 1000, contestID)
	if err != nil || len(qsos) == 0 {
		applog.Debug("contest stats: no QSOs", "err", err)
		return true
	}

	// TotalQSOs from COUNT — ListQSOs is capped at 1000 rows and
	// undercounts active contests. The window counters below and the peaks
	// see at most those 1000 newest rows.
	if counts, err := store.CountQSOsForContest(db, contestID); err == nil {
		cs.TotalQSOs = counts.Total
	} else {
		cs.TotalQSOs = len(qsos)
	}
	cs.LastQSO = qsos[0].CreatedAt.UTC()
	cs.FirstQSO = qsos[len(qsos)-1].CreatedAt.UTC()
	cs.Session = cs.LastQSO.Sub(cs.FirstQSO)

	hourStart := now.Truncate(time.Hour)
	cutoff60 := now.Add(-60 * time.Minute)
	for _, q := range qsos {
		created := q.CreatedAt.UTC()
		if created.After(hourStart) {
			cs.ThisHour++
		}
		if created.After(cutoff60) {
			cs.Last60Min++
		}
	}

	cs.Rate10 = cs.qsoRate(qsos, 10)
	cs.Rate100 = cs.qsoRate(qsos, 100)
	cs.Best1Min = cs.bestRate(qsos, 1*time.Minute)
	cs.Best10Min = cs.bestRate(qsos, 10*time.Minute)
	cs.Best60Min = cs.bestRate(qsos, 60*time.Minute)
	cs.OnAir = cs.computeOnAir(qsos)

	if cs.Session.Seconds() > 0 {
		// The session span covers the QSOs in the listing window (≤1000), so
		// the average has to use the same set — dividing the all-time total
		// by a shorter span overstated the rate on long contests.
		cs.AvgSession = int(float64(len(qsos)) / cs.Session.Hours())
	}

	cs.MinuteBuckets = bucketQSOsByMinute(qsos, 60, now)
	return true
}

func (cs *contestStats) qsoRate(qsos []qso.QSO, n int) int {
	if len(qsos) < 2 || n < 2 {
		return 0
	}
	count := n
	if count > len(qsos) {
		count = len(qsos)
	}
	first := qsos[count-1].CreatedAt.UTC()
	last := qsos[0].CreatedAt.UTC()
	span := last.Sub(first)
	if span <= 0 {
		return 0
	}
	return int(float64(count-1) / span.Hours())
}

func (cs *contestStats) bestRate(qsos []qso.QSO, window time.Duration) int {
	if len(qsos) < 2 {
		return 0
	}
	best := 0
	for end := 0; end < len(qsos); end++ {
		endTime := qsos[end].CreatedAt.UTC()
		startTime := endTime.Add(-window)
		count := 0
		for i := end; i < len(qsos); i++ {
			if qsos[i].CreatedAt.UTC().After(startTime) {
				count++
			} else {
				break
			}
		}
		rate := int(float64(count) / window.Hours())
		if rate > best {
			best = rate
		}
	}
	return best
}

func (cs *contestStats) computeOnAir(qsos []qso.QSO) time.Duration {
	if len(qsos) < 2 {
		return 0
	}
	const offThreshold = 30 * time.Minute
	var total time.Duration
	for i := 0; i < len(qsos)-1; i++ {
		gap := qsos[i].CreatedAt.UTC().Sub(qsos[i+1].CreatedAt.UTC())
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

func bucketQSOsByMinute(qsos []qso.QSO, numBuckets int, now time.Time) []float64 {
	buckets := make([]float64, numBuckets)
	bucketStart := now.Add(-time.Duration(numBuckets) * time.Minute)
	for _, q := range qsos {
		t := q.CreatedAt.UTC()
		if t.Before(bucketStart) {
			continue
		}
		idx := int(t.Sub(bucketStart).Minutes())
		if idx >= 0 && idx < numBuckets {
			buckets[idx]++
		}
	}
	return buckets
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

	m.contest.computeIfStale(m.App.DB, contestID, ct.Name)

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
	rows = append(rows, d(pad("Count"))+v(fmt.Sprintf("60m %3d  hr %3d", m.contest.Last60Min, m.contest.ThisHour)))

	// --- Peak row ---
	// bestRate extrapolates each window to QSOs per hour (one QSO inside the
	// last minute is 60/h), so the label carries the unit.
	rows = append(rows, d(pad("Peak/h"))+v(fmt.Sprintf("1m%3d 10m%3d 60m%3d",
		m.contest.Best1Min, m.contest.Best10Min, m.contest.Best60Min)))

	// --- Avg row ---
	rows = append(rows, d(pad("Avg"))+v(fmt.Sprintf("%4d/h  Sess %s",
		m.contest.AvgSession, formatDurationShort(m.contest.Session))))

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
		rows = append(rows, d(fmt.Sprintf("QSO/%dm  last 60m  max %.0f", bucketMin, maxVal)))
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
