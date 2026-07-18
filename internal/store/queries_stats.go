package store

import (
	"database/sql"
	"fmt"
	"strings"
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

// ── Worked panel statistics ────────────────────────────────────────────────

// CountItem pairs a label with an integer count for band/mode/grid lists.
type CountItem struct {
	Value string
	Count int
}

// QSOBrief holds minimal identifying fields for a single QSO, used for
// first/last QSO display.
type QSOBrief struct {
	Date string // YYYY-MM-DD
	Time string // HHMMSS
	Band string
	Mode string
}

// ScopeHistory holds aggregate statistics for a particular scope
// (callsign, 4-char grid, or DXCC entity).
type ScopeHistory struct {
	QSOCount    int
	UniqueCalls int
	UniqueBands int
	UniqueModes int
	FirstQSO    *QSOBrief
	LastQSO     *QSOBrief
	BandCounts  []CountItem
	ModeCounts  []CountItem
	GridCounts  []CountItem
}

// WorkedSummary collects call, grid, and DXCC scoped histories.
type WorkedSummary struct {
	CallHistory ScopeHistory
	GridHistory ScopeHistory
	DXCCHistory ScopeHistory
}

// GetWorkedSummary computes per-call, per-grid, and per-DXCC statistics
// for the given logbook. Callers should pass a non-empty callsign to
// resolve base_call. When grid4 is empty or <4 chars, GridHistory is left
// zero. When dxcc is empty, DXCCHistory is left zero.
// countryName is used as a fallback when the dxcc entity number column
// is empty (existing QSOs imported before the dxcc column was populated).
func GetWorkedSummary(db *sql.DB, call, grid4, dxcc, countryName string) (WorkedSummary, error) {
	var ws WorkedSummary
	var err error

	baseCall := qso.DeriveBaseCall(call)
	if baseCall != "" {
		ws.CallHistory, err = scopeStats(db, "base_call = ?", baseCall)
		if err != nil {
			return ws, fmt.Errorf("call history: %w", err)
		}
	}

	if len(grid4) >= 4 {
		grid4 = strings.ToUpper(grid4[:4])
		ws.GridHistory, err = scopeStats(db, "gridsquare LIKE ?", grid4+"%")
		if err != nil {
			return ws, fmt.Errorf("grid history: %w", err)
		}
	}

	if dxcc != "" {
		// Match by DXCC entity number when populated. Fall back to
		// case-insensitive country name for QSOs without the dxcc
		// column — covers "United States" / "UNITED STATES" / "united states"
		// and prefix variants like "United States of America".
		ws.DXCCHistory, err = scopeStats(db,
			"dxcc = ? OR LOWER(country) = LOWER(?) OR LOWER(country) LIKE LOWER(?)",
			dxcc, countryName, countryName+"%")
		if err != nil {
			return ws, fmt.Errorf("dxcc history: %w", err)
		}
	}

	return ws, nil
}

// scopeStats computes aggregate history for a single scope (call, grid,
// or DXCC). The whereClause is a SQL fragment and args provides bind values.
func scopeStats(db *sql.DB, whereClause string, args ...any) (ScopeHistory, error) {
	var sh ScopeHistory

	// ── Core aggregations ──────────────────────────────────────────────
	err := db.QueryRow(
		`SELECT
			COUNT(*),
			COUNT(DISTINCT base_call),
			COUNT(DISTINCT band),
			COUNT(DISTINCT CASE WHEN submode != '' THEN submode ELSE mode END)
		FROM qsos WHERE `+whereClause,
		args...,
	).Scan(
		&sh.QSOCount,
		&sh.UniqueCalls,
		&sh.UniqueBands,
		&sh.UniqueModes,
	)
	if err != nil {
		return sh, err
	}

	// ── First and last QSO ──────────────────────────────────────────
	sh.FirstQSO = queryQSOBrief(db, whereClause, "ASC", args...)
	sh.LastQSO = queryQSOBrief(db, whereClause, "DESC", args...)

	// ── Band, mode, grid counts ────────────────────────────────────────
	sh.BandCounts, err = countGroup(db, "band", whereClause, args, 6)
	if err != nil {
		return sh, fmt.Errorf("band counts: %w", err)
	}

	sh.ModeCounts, err = countGroup(db,
		"CASE WHEN submode != '' THEN submode ELSE mode END",
		whereClause, args, 4)
	if err != nil {
		return sh, fmt.Errorf("mode counts: %w", err)
	}

	sh.GridCounts, err = countGroup(db,
		"UPPER(SUBSTR(gridsquare, 1, 4))",
		whereClause+" AND gridsquare != ''", args, 4)
	if err != nil {
		return sh, fmt.Errorf("grid counts: %w", err)
	}

	return sh, nil
}

// queryQSOBrief returns the first or last QSO matching the given clause.
func queryQSOBrief(db *sql.DB, whereClause string, order string, args ...any) *QSOBrief {
	var date, time, band, mode string
	err := db.QueryRow(
		`SELECT qso_date, time_on, band,
			CASE WHEN submode != '' THEN submode ELSE mode END
		FROM qsos WHERE `+whereClause+`
		ORDER BY qso_date `+order+`, time_on `+order+`
		LIMIT 1`,
		args...,
	).Scan(&date, &time, &band, &mode)
	if err != nil || date == "" {
		return nil
	}
	return &QSOBrief{
		Date: dateCompact(date),
		Time: time,
		Band: band,
		Mode: mode,
	}
}

// countGroup returns a deduplicated, count-ordered list of (value, count)
// pairs for the given expression (e.g. "band", "mode", or a CASE).
func countGroup(db *sql.DB, expr, whereClause string, args []any, limit int) ([]CountItem, error) {
	query := `SELECT ` + expr + `, COUNT(*) AS cnt
		FROM qsos
		WHERE ` + whereClause + `
		GROUP BY ` + expr + `
		ORDER BY cnt DESC
		LIMIT ?`
	allArgs := append(args, limit)
	rows, err := db.Query(query, allArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []CountItem
	for rows.Next() {
		var val string
		var cnt int
		if err := rows.Scan(&val, &cnt); err != nil {
			return items, err
		}
		if val == "" {
			continue
		}
		items = append(items, CountItem{Value: val, Count: cnt})
	}
	return items, rows.Err()
}

// dateCompact turns YYYYMMDD into YYYY-MM-DD.
func dateCompact(d string) string {
	if len(d) == 8 {
		return d[0:4] + "-" + d[4:6] + "-" + d[6:8]
	}
	return d
}

// ── Dashboard ──────────────────────────────────────────────────────────────
type DashboardStats struct {
	QSOsToday   int
	Operators   int
	UniqueCalls int
	DXCC        int
	Grids       int
	Bands       int
	Modes       int
	LastQSOAgoS int
	Rate5m      int
	Rate15m     int
	Rate60m     int
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

	// Rate: QSOs in the last 5, 15, and 60 minutes.
	// Use printf to normalise time_on to 6 chars (HHMMSS) for reliable comparison.
	for _, w := range []struct {
		mins int
		dest *int
	}{
		{5, &s.Rate5m},
		{15, &s.Rate15m},
		{60, &s.Rate60m},
	} {
		cutoff := time.Now().UTC().Add(-time.Duration(w.mins) * time.Minute).Format("20060102150405")
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM qsos WHERE printf('%s%06s', qso_date, COALESCE(time_on,'000000')) >= ?`,
			cutoff,
		).Scan(&n); err != nil {
			n = 0
		}
		*w.dest = n
	}

	return s, nil
}
