package store

import (
	"database/sql"
	"fmt"

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
