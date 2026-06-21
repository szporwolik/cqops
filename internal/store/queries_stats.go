package store

import (
	"database/sql"
	"fmt"
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

// GetLogbookStats computes per-call statistics efficiently with a single
// multi-count query. band and mode use the same prefix-match logic as
// SearchQSOsByCall for consistency.
func GetLogbookStats(db *sql.DB, call, band, mode string) (LogbookStats, error) {
	var s LogbookStats

	// Match: exact, suffix (/P etc.), prefix (DL/ etc.), both (DL/.../P).
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM qsos WHERE call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?`,
		call, call+"/%", "%/"+call, "%/"+call+"/%",
	).Scan(&s.QSOCount); err != nil {
		return s, fmt.Errorf("count call qsos: %w", err)
	}
	s.CallWorked = s.QSOCount > 0

	// On specific band.
	if band != "" {
		var n int
		db.QueryRow(
			`SELECT COUNT(*) FROM qsos WHERE (call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?) AND band = ?`,
			call, call+"/%", "%/"+call, "%/"+call+"/%", band,
		).Scan(&n)
		s.CallOnBand = n > 0
	}

	// On specific mode.
	if mode != "" {
		var n int
		db.QueryRow(
			`SELECT COUNT(*) FROM qsos WHERE (call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ?) AND mode = ?`,
			call, call+"/%", "%/"+call, "%/"+call+"/%", mode,
		).Scan(&n)
		s.CallOnMode = n > 0
	}

	// Last QSO date.
	var lastDate string
	db.QueryRow(
		`SELECT qso_date FROM qsos WHERE call = ? OR call LIKE ? OR call LIKE ? OR call LIKE ? ORDER BY id DESC LIMIT 1`,
		call, call+"/%", "%/"+call, "%/"+call+"/%",
	).Scan(&lastDate)
	if lastDate != "" && len(lastDate) == 8 {
		s.LastQSODate = lastDate[0:4] + "-" + lastDate[4:6] + "-" + lastDate[6:8]
	}

	return s, nil
}
