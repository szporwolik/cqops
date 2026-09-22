package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/qso"
)

// ─────────────────────────────────────────────────────────────────────────────
// Materialized worked-status index.
//
// The worked index answers "is this call/grid/DXCC/band/mode worked?" without
// scanning qsos. It is maintained incrementally by every QSO write path
// (single insert, bulk import, edit, delete) and rebuilt deterministically by
// RebuildWorkedIndex. The normalized qsos table remains the source of truth;
// these tables are disposable summaries.
//
// Aggregation levels stored in worked_dxcc (band/mode default ''):
//
//	DXCC                     (dxcc, '', '')
//	DXCC + band              (dxcc, band, '')
//	DXCC + canonical mode    (dxcc, '', mode)
//	DXCC + band + mode       (dxcc, band, mode)
//
// Canonical mode: submode wins when present (MFSK+FT8 → FT8, MFSK+FT4 → FT4) —
// the same expression the live aggregate queries use.
// ─────────────────────────────────────────────────────────────────────────────

// canonicalMode resolves the mode a QSO counts toward.
func canonicalMode(mode, submode string) string {
	if submode != "" {
		return submode
	}
	return mode
}

// qsoStamp is the sortable timestamp used for first/last aggregation.
// Consistent with the SQL printf('%s%06s', qso_date, COALESCE(time_on,'000000')).
func qsoStamp(qsoDate, timeOn string) string {
	if len(timeOn) < 6 {
		timeOn = strings.Repeat("0", 6-len(timeOn)) + timeOn
	}
	return qsoDate + timeOn
}

// sqlExecutor is the surface index maintenance needs from *sql.DB and *sql.Tx.
type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

// applyCountDelta upserts one aggregate row and removes it when the count
// reaches zero. For decrements the first/last edge timestamps are recomputed
// from qsos when the row survives — the removed QSO may have been the edge.
func applyCountDelta(e sqlExecutor, table, keyCols, keyVals string, keyArgs []any, stamp string, delta int) error {
	cols := strings.Split(keyCols, ",")
	vals := strings.Split(keyVals, ",")
	var pred strings.Builder
	for i := range cols {
		if i > 0 {
			pred.WriteString(" AND ")
		}
		pred.WriteString(strings.TrimSpace(cols[i]) + " = " + strings.TrimSpace(vals[i]))
	}
	where := pred.String()

	upsert := fmt.Sprintf(
		`INSERT INTO %s (%s, qso_count, first_utc, last_utc) VALUES (%s, ?, ?, ?)
		 ON CONFLICT(%s) DO UPDATE SET
		   qso_count = %s.qso_count + excluded.qso_count,
		   first_utc = MIN(%s.first_utc, excluded.first_utc),
		   last_utc  = MAX(%s.last_utc, excluded.last_utc)`,
		table, keyCols, keyVals, keyCols, table, table, table,
	)
	args := append(append([]any{}, keyArgs...), delta, stamp, stamp)
	if _, err := e.Exec(upsert, args...); err != nil {
		return fmt.Errorf("worked index upsert %s: %w", table, err)
	}
	if delta > 0 {
		return nil
	}
	// Remove zeroed rows.
	if _, err := e.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s AND qso_count <= 0`, table, where), keyArgs...); err != nil {
		return fmt.Errorf("worked index cleanup %s: %w", table, err)
	}
	// Recompute edges only when the row survived a decrement.
	var remaining int
	if err := e.QueryRow(fmt.Sprintf(`SELECT qso_count FROM %s WHERE %s`, table, where), keyArgs...).Scan(&remaining); err != nil {
		return nil // row gone (deleted above) — nothing to fix
	}
	if remaining <= 0 {
		return nil
	}
	switch table {
	case "worked_call":
		_, err := e.Exec(`UPDATE worked_call SET first_utc = (SELECT MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE base_call = ?), last_utc = (SELECT MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE base_call = ?) WHERE base_call = ?`, keyArgs[0], keyArgs[0], keyArgs[0])
		if err != nil {
			return fmt.Errorf("worked call edge recompute: %w", err)
		}
	case "worked_grid":
		_, err := e.Exec(`UPDATE worked_grid SET first_utc = (SELECT MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE UPPER(SUBSTR(gridsquare,1,4)) = ?), last_utc = (SELECT MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE UPPER(SUBSTR(gridsquare,1,4)) = ?) WHERE grid4 = ?`, keyArgs[0], keyArgs[0], keyArgs[0])
		if err != nil {
			return fmt.Errorf("worked grid edge recompute: %w", err)
		}
	case "worked_dxcc":
		_, err := e.Exec(`UPDATE worked_dxcc SET first_utc = (SELECT MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE dxcc = ? AND (? = '' OR band = ?) AND (? = '' OR CASE WHEN submode != '' THEN submode ELSE mode END = ?)), last_utc = (SELECT MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))) FROM qsos WHERE dxcc = ? AND (? = '' OR band = ?) AND (? = '' OR CASE WHEN submode != '' THEN submode ELSE mode END = ?)) WHERE dxcc = ? AND band = ? AND mode = ?`,
			keyArgs[0], keyArgs[1], keyArgs[1], keyArgs[2], keyArgs[2],
			keyArgs[0], keyArgs[1], keyArgs[1], keyArgs[2], keyArgs[2],
			keyArgs[0], keyArgs[1], keyArgs[2])
		if err != nil {
			return fmt.Errorf("worked dxcc edge recompute: %w", err)
		}
	}
	return nil
}

// indexQSO applies one QSO (delta +1) or removes it (delta -1) from every
// aggregate it participates in. Incomplete data contributes conservatively:
// missing base call / grid / DXCC / band / mode simply skips the affected
// aggregates.
func indexQSO(e sqlExecutor, q *qso.QSO, delta int) error {
	stamp := qsoStamp(q.QSODate, q.TimeOn)

	if bc := qso.DeriveBaseCall(q.Call); bc != "" {
		if err := applyCountDelta(e, "worked_call", "base_call", "?", []any{bc}, stamp, delta); err != nil {
			return err
		}
	}

	if len(q.GridSquare) >= 4 {
		grid4 := strings.ToUpper(q.GridSquare[:4])
		if err := applyCountDelta(e, "worked_grid", "grid4", "?", []any{grid4}, stamp, delta); err != nil {
			return err
		}
	}

	dxcc := strings.TrimSpace(q.DXCC)
	if dxcc == "" {
		return nil
	}
	mode := canonicalMode(q.Mode, q.Submode)
	levels := [][2]string{{"", ""}}
	if q.Band != "" {
		levels = append(levels, [2]string{q.Band, ""})
	}
	if mode != "" {
		levels = append(levels, [2]string{"", mode})
	}
	if q.Band != "" && mode != "" {
		levels = append(levels, [2]string{q.Band, mode})
	}
	for _, lv := range levels {
		args := []any{dxcc, lv[0], lv[1]}
		if err := applyCountDelta(e, "worked_dxcc", "dxcc, band, mode", "?, ?, ?", args, stamp, delta); err != nil {
			return err
		}
	}
	return nil
}

// RebuildWorkedIndex rebuilds the entire worked index from qsos in one
// transaction. Deterministic and safe to run repeatedly — used after
// migration, purges, or when an inconsistency is suspected.
func RebuildWorkedIndex(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("worked index rebuild begin: %w", err)
	}
	defer tx.Rollback()
	if err := rebuildWorkedIndex(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// rebuildWorkedIndex is the executor-level rebuild — callers may supply their
// own transaction so the index and the data change commit together.
func rebuildWorkedIndex(e sqlExecutor) error {
	for _, t := range []string{"worked_call", "worked_grid", "worked_dxcc"} {
		if _, err := e.Exec(`DELETE FROM ` + t); err != nil {
			return fmt.Errorf("worked index clear %s: %w", t, err)
		}
	}

	if _, err := e.Exec(`
		INSERT INTO worked_call(base_call, qso_count, first_utc, last_utc)
		SELECT base_call, COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE base_call != '' GROUP BY base_call`); err != nil {
		return fmt.Errorf("worked index rebuild call: %w", err)
	}
	if _, err := e.Exec(`
		INSERT INTO worked_grid(grid4, qso_count, first_utc, last_utc)
		SELECT UPPER(SUBSTR(gridsquare,1,4)), COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE LENGTH(gridsquare) >= 4 GROUP BY 1`); err != nil {
		return fmt.Errorf("worked index rebuild grid: %w", err)
	}
	if _, err := e.Exec(`
		INSERT INTO worked_dxcc(dxcc, band, mode, qso_count, first_utc, last_utc)
		SELECT dxcc, '', '', COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE dxcc != '' GROUP BY dxcc
		UNION ALL
		SELECT dxcc, band, '', COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE dxcc != '' AND band != '' GROUP BY dxcc, band
		UNION ALL
		SELECT dxcc, '', CASE WHEN submode != '' THEN submode ELSE mode END, COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE dxcc != '' AND mode != '' GROUP BY dxcc, 3
		UNION ALL
		SELECT dxcc, band, CASE WHEN submode != '' THEN submode ELSE mode END, COUNT(*),
			MIN(printf('%s%06s', qso_date, COALESCE(time_on,'000000'))),
			MAX(printf('%s%06s', qso_date, COALESCE(time_on,'000000')))
		FROM qsos WHERE dxcc != '' AND band != '' AND mode != '' GROUP BY dxcc, band, 3`); err != nil {
		return fmt.Errorf("worked index rebuild dxcc: %w", err)
	}

	return nil
}

// qsoIndexColumns is the SELECT list of the qsos columns index maintenance
// needs from a row (unindex before edit/delete).
const qsoIndexColumns = `call, band, mode, submode, gridsquare, dxcc, qso_date, time_on`

// loadQSOForIndex reads the index-relevant fields of one row.
func loadQSOForIndex(q sqlExecutor, id int64) (*qso.QSO, error) {
	row := q.QueryRow(`SELECT `+qsoIndexColumns+` FROM qsos WHERE id = ?`, id)
	var out qso.QSO
	out.ID = id
	if err := row.Scan(&out.Call, &out.Band, &out.Mode, &out.Submode, &out.GridSquare, &out.DXCC, &out.QSODate, &out.TimeOn); err != nil {
		return nil, err
	}
	return &out, nil
}
