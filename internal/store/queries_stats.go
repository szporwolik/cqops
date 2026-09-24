package store

import (
	"database/sql"
	"errors"
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
	toWavelog := `SELECT COUNT(*) FROM qsos WHERE wavelog_id > 0`
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

// LogbookCounts returns total QSOs and today's QSO count for the entire
// logbook. today is the UTC date in YYYYMMDD format.
func LogbookCounts(db *sql.DB, today string) (total, todayCount int, err error) {
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos`).Scan(&total); err != nil {
		return 0, 0, fmt.Errorf("count qsos: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE qso_date = ?`, today).Scan(&todayCount); err != nil {
		return 0, 0, fmt.Errorf("count today qsos: %w", err)
	}
	return total, todayCount, nil
}

// LogbookStats holds per-call aggregate statistics from the local logbook.
type LogbookStats struct {
	CallWorked  bool
	CallOnBand  bool
	CallOnMode  bool
	QSOCount    int
	TodayCount  int
	LastQSODate string // YYYY-MM-DD or empty
}

// GetLogbookStats computes per-call statistics with a single query.
// Uses the indexed base_call column for fast exact-match lookups,
// avoiding the old unindexable LIKE '%/call' table scans.
func GetLogbookStats(db *sql.DB, call, band, mode string) (LogbookStats, error) {
	var s LogbookStats
	baseCall := qso.DeriveBaseCall(call)
	today := time.Now().UTC().Format("20060102")

	// Single query: COUNT total, SUM for band/mode/today matches, MAX for last date.
	// All five aggregations run in one table scan over the matching rows.
	var onBandCount, onModeCount int
	var lastDate string
	err := db.QueryRow(
		`SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN ? = '' THEN 0 WHEN band = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN ? = '' THEN 0 WHEN mode = ? THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN qso_date = ? THEN 1 ELSE 0 END), 0),
			COALESCE(MAX(qso_date), '')
		FROM qsos
		WHERE base_call = ?`,
		band, band,
		mode, mode,
		today,
		baseCall,
	).Scan(&s.QSOCount, &onBandCount, &onModeCount, &s.TodayCount, &lastDate)
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
		// Range predicate instead of LIKE 'DM03%': a mixed-case pattern
		// cannot use the LIKE index optimization, which made every grid
		// scope query a full table scan. The half-open range seeks the
		// idx_qsos_gridsquare b-tree directly. The +1 upper bound is an
		// exclusive sentinel — 'R'→'S', '9'→':' still sort after every
		// possible continuation.
		upper := grid4[:3] + string(grid4[3]+1)
		ws.GridHistory, err = scopeStats(db, "gridsquare >= ? AND gridsquare < ?", grid4, upper)
		if err != nil {
			return ws, fmt.Errorf("grid history: %w", err)
		}
	}

	if dxcc != "" {
		ws.DXCCHistory, err = scopeStatsDXCCFast(db, dxcc, countryName)
		if err != nil {
			return ws, fmt.Errorf("dxcc history: %w", err)
		}
	}

	return ws, nil
}

// scopeStatsDXCCFast computes the DXCC-entity scope from the materialized
// worked index. Rows whose entity number matches come straight from
// worked_dxcc (the four aggregation levels); the few remaining aggregate
// dimensions (distinct calls, grid spread, edge QSOs) are index-served
// queries on the dxcc column.
//
// Databases with legacy country-fallback rows (no matching entity number,
// matched by country name) fall back to the union-source path so the results
// stay exactly what the historical query produced.
func scopeStatsDXCCFast(db *sql.DB, dxcc, countryName string) (ScopeHistory, error) {
	var fallback int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM qsos
		 WHERE (dxcc IS NULL OR dxcc = '' OR dxcc != ?) AND country LIKE ? COLLATE NOCASE LIMIT 1`,
		dxcc, strings.ToLower(countryName)+"%",
	).Scan(&fallback); err != nil {
		return ScopeHistory{}, err
	}
	if fallback > 0 {
		return scopeStatsDXCC(db, dxcc, countryName)
	}

	var sh ScopeHistory
	err := db.QueryRow(
		`SELECT qso_count, first_utc, last_utc FROM worked_dxcc
		 WHERE dxcc = ? AND band = '' AND mode = ''`,
		dxcc,
	).Scan(&sh.QSOCount, new(string), new(string))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sh, nil // not worked — honest zero history
		}
		return sh, err
	}

	if err := db.QueryRow(`SELECT COUNT(DISTINCT base_call) FROM qsos WHERE dxcc = ?`, dxcc).Scan(&sh.UniqueCalls); err != nil {
		return sh, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM worked_dxcc WHERE dxcc = ? AND band != '' AND mode = ''`, dxcc).Scan(&sh.UniqueBands); err != nil {
		return sh, err
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM worked_dxcc WHERE dxcc = ? AND band = '' AND mode != ''`, dxcc).Scan(&sh.UniqueModes); err != nil {
		return sh, err
	}

	sh.FirstQSO = queryQSOBrief(db, "dxcc = ?", "ASC", dxcc)
	sh.LastQSO = queryQSOBrief(db, "dxcc = ?", "DESC", dxcc)

	sh.BandCounts, err = sumGroupFromWhere(db,
		"band", "worked_dxcc", `dxcc = ? AND band != '' AND mode = ''`, []any{dxcc}, 6)
	if err != nil {
		return sh, fmt.Errorf("band counts: %w", err)
	}
	sh.ModeCounts, err = sumGroupFromWhere(db,
		"mode", "worked_dxcc", `dxcc = ? AND band = '' AND mode != ''`, []any{dxcc}, 4)
	if err != nil {
		return sh, fmt.Errorf("mode counts: %w", err)
	}
	sh.GridCounts, err = countGroup(db,
		"UPPER(SUBSTR(gridsquare, 1, 4))", "dxcc = ? AND gridsquare != ''", []any{dxcc}, 4)
	if err != nil {
		return sh, fmt.Errorf("grid counts: %w", err)
	}
	return sh, nil
}

// scopeStatsDXCC computes the DXCC-entity scope as two index-driven arms
// united in one source: rows with the stored entity number (idx_qsos_dxcc)
// and rows without one whose country name matches case-insensitively
// (idx_qsos_country_nocase, prefix included for "United States of America"-
// style variants). The previous single three-way OR predicate made SQLite
// abandon every index and full-scan the table in all six queries; each arm
// here seeks its own index and only the matched rows are materialized.
func scopeStatsDXCC(db *sql.DB, dxcc, countryName string) (ScopeHistory, error) {
	var sh ScopeHistory

	src := `(
		SELECT base_call, band, submode, mode, qso_date, time_on, gridsquare
		FROM qsos WHERE dxcc = ?
		UNION ALL
		SELECT base_call, band, submode, mode, qso_date, time_on, gridsquare
		FROM qsos WHERE (dxcc IS NULL OR dxcc = '' OR dxcc != ?) AND country LIKE ? COLLATE NOCASE
	)`
	pattern := strings.ToLower(countryName) + "%"
	args := []any{dxcc, dxcc, pattern}

	err := db.QueryRow(
		`SELECT
			COUNT(*),
			COUNT(DISTINCT base_call),
			COUNT(DISTINCT band),
			COUNT(DISTINCT CASE WHEN submode != '' THEN submode ELSE mode END)
		FROM `+src,
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

	for _, order := range []string{"ASC", "DESC"} {
		brief := &QSOBrief{}
		var date, time, band, mode string
		err := db.QueryRow(
			`SELECT qso_date, time_on, band,
				CASE WHEN submode != '' THEN submode ELSE mode END
			FROM `+src+`
			ORDER BY qso_date `+order+`, time_on `+order+`
			LIMIT 1`,
			args...,
		).Scan(&date, &time, &band, &mode)
		if err != nil || date == "" {
			brief = nil
		} else {
			brief.Date = dateCompact(date)
			brief.Time = time
			brief.Band = band
			brief.Mode = mode
		}
		if order == "ASC" {
			sh.FirstQSO = brief
		} else {
			sh.LastQSO = brief
		}
	}

	sh.BandCounts, err = countGroupFrom(db, "band", src, args, 6)
	if err != nil {
		return sh, fmt.Errorf("band counts: %w", err)
	}

	sh.ModeCounts, err = countGroupFrom(db,
		"CASE WHEN submode != '' THEN submode ELSE mode END", src, args, 4)
	if err != nil {
		return sh, fmt.Errorf("mode counts: %w", err)
	}

	sh.GridCounts, err = countGroupFromWhere(db,
		"UPPER(SUBSTR(gridsquare, 1, 4))", src, "gridsquare != ''", args, 4)
	if err != nil {
		return sh, fmt.Errorf("grid counts: %w", err)
	}

	return sh, nil
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
	return countGroupFromWhere(db, expr, `qsos`, whereClause, args, limit)
}

// countGroupFrom is countGroup over an arbitrary source expression — a
// table name or a UNION ALL subquery with its own bind arguments.
func countGroupFrom(db *sql.DB, expr, source string, args []any, limit int) ([]CountItem, error) {
	return countGroupFromWhere(db, expr, source, "", args, limit)
}

// countGroupFromWhere is countGroupFrom with an extra row predicate (e.g.
// gridsquare != ” for grid counts over a subquery source).
func countGroupFromWhere(db *sql.DB, expr, source, extraWhere string, args []any, limit int) ([]CountItem, error) {
	return groupFromWhere(db, expr, source, extraWhere, "COUNT(*)", args, limit)
}

// sumGroupFromWhere aggregates the qso_count column of a worked-index table
// instead of counting rows.
func sumGroupFromWhere(db *sql.DB, expr, source, extraWhere string, args []any, limit int) ([]CountItem, error) {
	return groupFromWhere(db, expr, source, extraWhere, "SUM(qso_count)", args, limit)
}

func groupFromWhere(db *sql.DB, expr, source, extraWhere, agg string, args []any, limit int) ([]CountItem, error) {
	query := `SELECT ` + expr + `, ` + agg + ` AS cnt
		FROM ` + source
	if extraWhere != "" {
		query += `
		WHERE ` + extraWhere
	}
	query += `
		GROUP BY ` + expr + `
		ORDER BY cnt DESC
		LIMIT ?`
	allArgs := append(append([]any{}, args...), limit)
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
	// Two-column range so idx_qsos_date_time serves the predicate — the old
	// printf('%s%06s', ...) form was a function-on-column that forced a full
	// table scan three times per dashboard refresh.
	for _, w := range []struct {
		mins int
		dest *int
	}{
		{5, &s.Rate5m},
		{15, &s.Rate15m},
		{60, &s.Rate60m},
	} {
		cutoff := time.Now().UTC().Add(-time.Duration(w.mins) * time.Minute)
		var n int
		if err := db.QueryRow(
			`SELECT COUNT(*) FROM qsos
			WHERE qso_date > ? OR (qso_date = ? AND COALESCE(time_on,'000000') >= ?)`,
			cutoff.Format("20060102"), cutoff.Format("20060102"), cutoff.Format("150405"),
		).Scan(&n); err != nil {
			n = 0
		}
		*w.dest = n
	}

	return s, nil
}
