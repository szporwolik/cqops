package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// QSOCounts holds aggregate QSO statistics.
type QSOCounts struct {
	Total     int
	FromWSJTX int
	ToWavelog int
}

// CountQSOs returns aggregate statistics for the current logbook.
func CountQSOs(db *sql.DB) (QSOCounts, error) {
	return CountQSOsForContest(db, "")
}

// CountQSOsForContest returns aggregate statistics filtered by contest ID.
// Pass empty string for no contest filter.
func CountQSOsForContest(db *sql.DB, contestID string) (QSOCounts, error) {
	var c QSOCounts
	filter := ""
	args := []any{}
	if contestID != "" {
		filter = " WHERE contest_id = ?"
		args = append(args, contestID)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos`+filter, args...).Scan(&c.Total); err != nil {
		return c, fmt.Errorf("count qsos: %w", err)
	}
	fromWSJTX := `SELECT COUNT(*) FROM qsos WHERE source='wsjtx'`
	toWavelog := `SELECT COUNT(*) FROM qsos WHERE wavelog_uploaded='yes'`
	if contestID != "" {
		fromWSJTX += ` AND contest_id = ?`
		toWavelog += ` AND contest_id = ?`
	}
	if err := db.QueryRow(fromWSJTX, args...).Scan(&c.FromWSJTX); err != nil {
		return c, fmt.Errorf("count wsjtx qsos: %w", err)
	}
	if err := db.QueryRow(toWavelog, args...).Scan(&c.ToWavelog); err != nil {
		return c, fmt.Errorf("count wavelog qsos: %w", err)
	}
	return c, nil
}

// LogbookStats holds per-call aggregate statistics from the local logbook.
type LogbookStats struct {
	CallWorked  bool
	CallOnBand  bool
	CallOnMode  bool
	QSOCount    int
	LastQSODate string // YYYY-MM-DD or empty
}

// GetLogbookStats computes per-call statistics with a single query.
// Uses the indexed base_call column for fast exact-match lookups,
// avoiding the old unindexable LIKE '%/call' table scans.
func GetLogbookStats(db *sql.DB, call, band, mode string) (LogbookStats, error) {
	var s LogbookStats
	baseCall := qso.DeriveBaseCall(call)

	// Single query: COUNT total, SUM for band/mode matches, MAX for last date.
	// All four aggregations run in one table scan over the matching rows.
	// When band/mode is empty, the SUM returns 0 (no rows match ''=band).
	var onBandCount, onModeCount int
	var lastDate string
	err := db.QueryRow(
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN ? = '' THEN 0 WHEN band = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN ? = '' THEN 0 WHEN mode = ? THEN 1 ELSE 0 END), 0),
			COALESCE(MAX(qso_date), '')
		FROM qsos
		WHERE base_call = ?`,
		band, band,
		mode, mode,
		baseCall,
	).Scan(&s.QSOCount, &onBandCount, &onModeCount, &lastDate)
	if err != nil {
		return s, fmt.Errorf("logbook stats: %w", err)
	}

	s.CallWorked = s.QSOCount > 0
	s.CallOnBand = onBandCount > 0
	s.CallOnMode = onModeCount > 0
	if lastDate != "" && len(lastDate) == 8 {
		s.LastQSODate = lastDate[0:4] + "-" + lastDate[4:6] + "-" + lastDate[6:8]
	}

	return s, nil
}

// DashboardStats holds aggregate statistics for the CQOps Live dashboard.
type DashboardStats struct {
	QSOsToday   int
	Operators   int
	UniqueCalls int
	DXCC        int
	Grids       int
	Bands       int
	Modes       int
	LastQSOAgoS int
	RatePerHour float64
}

// GetDashboardStats computes dashboard aggregate statistics for all QSOs
// from the given start date (inclusive, YYYYMMDD format). For a typical
// today-only view, pass time.Now().UTC().Format("20060102").
// When an event start date is configured, pass that instead.
func GetDashboardStats(db *sql.DB, startDate string) (DashboardStats, error) {
	var s DashboardStats
	var lastQSOStr string
	// Limit scan to the event window for large logs.
	cutoff := startDate

	err := db.QueryRow(`
		SELECT
			COALESCE(COUNT(*), 0),
			COALESCE(COUNT(DISTINCT CASE WHEN operator != '' THEN operator END), 0),
			COALESCE(COUNT(DISTINCT base_call), 0),
			COALESCE(COUNT(DISTINCT country), 0),
			COALESCE(COUNT(DISTINCT CASE WHEN gridsquare != '' THEN UPPER(SUBSTR(gridsquare,1,4)) END), 0),
			COALESCE(COUNT(DISTINCT band), 0),
			COALESCE(COUNT(DISTINCT mode), 0),
			COALESCE(MAX(qso_date) || MAX(time_on), '')
		FROM qsos WHERE qso_date >= ?
	`, cutoff).Scan(
		&s.QSOsToday,
		&s.Operators,
		&s.UniqueCalls,
		&s.DXCC,
		&s.Grids,
		&s.Bands,
		&s.Modes,
		&lastQSOStr,
	)
	if err != nil {
		return s, fmt.Errorf("dashboard stats: %w", err)
	}

	// Parse last QSO time for elapsed-seconds computation.
	if len(lastQSOStr) >= 14 {
		if t, err := time.Parse("20060102150405", lastQSOStr[:14]); err == nil {
			s.LastQSOAgoS = int(time.Since(t).Seconds())
		}
	}

	// Rate: QSOs in the last hour. Compare only the HHMM portion of time_on
	// because it may be stored as HHMM (4 chars) or HHMMSS (6 chars).
	var lastHour int
	oneHourAgo := time.Now().UTC().Add(-1 * time.Hour)
	oneHourDate := oneHourAgo.Format("20060102")
	oneHourTime := oneHourAgo.Format("1504") // HHMM
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM qsos
		WHERE qso_date > ? OR (qso_date = ? AND SUBSTR(time_on,1,4) >= ?)
	`, oneHourDate, oneHourDate, oneHourTime).Scan(&lastHour); err != nil {
		lastHour = 0
	}
	s.RatePerHour = float64(lastHour)

	return s, nil
}
