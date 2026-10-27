package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
// the same expression the rebuild SQL uses. Keep canonicalMode and the SQL in
// sync; changing either bumps workedIndexRulesVersion.
//
// Besides counts, each aggregate carries:
//   - confirmed_count — contacts with any confirmation (QSL/LoTW/eQSL)
//   - source_mask     — which write paths contributed (worked_dxcc only)
//   - first/last/updated timestamps
//
// Provenance of the entity number itself lives on the QSO (dxcc_source).
// ─────────────────────────────────────────────────────────────────────────────

// workedIndexRulesVersion is stored in worked_index_meta. Bump it whenever the
// meaning of the aggregates changes (canonical mode mapping, grid precision,
// what counts as confirmed, source classification). Every installation
// rebuilds its index on the next start; no schema version bump needed.
//
// "3" added the maintained completeness counters, which a rebuild seeds.
const workedIndexRulesVersion = "3"

// Source provenance bits stored in worked_dxcc.source_mask.
const (
	SourceMaskManual  = 1 << 0 // typed in the form / unknown legacy source
	SourceMaskWSJTX   = 1 << 1 // WSJT-X auto-log
	SourceMaskADIF    = 1 << 2 // ADIF import
	SourceMaskRemote  = 1 << 3 // remote logbook download (Wavelog)
	SourceMaskOther   = 1 << 4 // any other integration
	sourceMaskAllBits = SourceMaskManual | SourceMaskWSJTX | SourceMaskADIF | SourceMaskRemote | SourceMaskOther
)

// sourceMaskBit classifies a qsos.source value. Precedence matters: the SQL
// rebuild uses the same order, so indexQSO and a rebuild always agree.
func sourceMaskBit(source string) int {
	s := strings.ToLower(strings.TrimSpace(source))
	switch {
	case s == "" || s == "manual" || s == "form":
		return SourceMaskManual
	case strings.Contains(s, "wsjt"):
		return SourceMaskWSJTX
	case strings.Contains(s, "adif") || s == "import":
		return SourceMaskADIF
	case strings.Contains(s, "wavelog"):
		return SourceMaskRemote
	}
	return SourceMaskOther
}

// sourceMaskSQL returns the SQL expression computing the source mask of a
// group. It mirrors sourceMaskBit exactly.
func sourceMaskSQL() string {
	return `MAX(CASE WHEN LOWER(COALESCE(source,'')) IN ('', 'manual', 'form') THEN 1 ELSE 0 END) * 1
		+ MAX(CASE WHEN LOWER(COALESCE(source,'')) LIKE '%wsjt%' THEN 1 ELSE 0 END) * 2
		+ MAX(CASE WHEN LOWER(COALESCE(source,'')) LIKE '%adif%' OR LOWER(COALESCE(source,'')) = 'import' THEN 1 ELSE 0 END) * 4
		+ MAX(CASE WHEN LOWER(COALESCE(source,'')) LIKE '%wavelog%' THEN 1 ELSE 0 END) * 8
		+ MAX(CASE WHEN LOWER(COALESCE(source,'')) NOT IN ('', 'manual', 'form')
		           AND LOWER(COALESCE(source,'')) NOT LIKE '%wsjt%'
		           AND LOWER(COALESCE(source,'')) NOT LIKE '%adif%'
		           AND LOWER(COALESCE(source,'')) <> 'import'
		           AND LOWER(COALESCE(source,'')) NOT LIKE '%wavelog%' THEN 1 ELSE 0 END) * 16`
}

// confirmedSQL is the SQL counterpart of qso.QSO.Confirmed().
const confirmedSQL = `CASE WHEN UPPER(COALESCE(qsl_rcvd,'')) IN ('Y','V')
	OR UPPER(COALESCE(lotw_qsl_rcvd,'')) IN ('Y','V')
	OR UPPER(COALESCE(eqsl_qsl_rcvd,'')) IN ('Y','V') THEN 1 ELSE 0 END`

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

// indexDelta is one QSO's contribution to an aggregate.
type indexDelta struct {
	stamp     string
	count     int // +1 on insert, -1 on removal
	confirmed int // +1 / -1 accordingly
	mask      int // source bit; OR-ed in, recomputed on removal
}

// applyCountDelta upserts one aggregate row and removes it when the count
// reaches zero. For decrements the first/last edge timestamps (and the source
// mask) are recomputed from qsos when the row survives — the removed QSO may
// have been the only representative of an edge or of its source.
func applyCountDelta(e sqlExecutor, table, keyCols, keyVals string, keyArgs []any, d indexDelta) error {
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

	// Only worked_dxcc carries the provenance mask.
	maskCol, maskSet, maskVal := "", "", ""
	if table == "worked_dxcc" {
		maskCol = ", source_mask"
		maskVal = ", ?"
		maskSet = ", source_mask = " + table + ".source_mask | excluded.source_mask"
	}

	upsert := fmt.Sprintf(
		`INSERT INTO %s (%s, qso_count, confirmed_count, first_utc, last_utc, updated_utc%s) VALUES (%s, ?, ?, ?, ?, ?%s)
		 ON CONFLICT(%s) DO UPDATE SET
		   qso_count = %s.qso_count + excluded.qso_count,
		   confirmed_count = MAX(0, %s.confirmed_count + excluded.confirmed_count),
		   first_utc = MIN(%s.first_utc, excluded.first_utc),
		   last_utc  = MAX(%s.last_utc, excluded.last_utc),
		   updated_utc = excluded.updated_utc%s`,
		table, keyCols, maskCol, keyVals, maskVal,
		keyCols, table, table, table, table, maskSet,
	)
	args := append(append([]any{}, keyArgs...), d.count, d.confirmed, d.stamp, d.stamp, d.stamp)
	if maskCol != "" {
		args = append(args, d.mask)
	}
	if _, err := e.Exec(upsert, args...); err != nil {
		return fmt.Errorf("worked index upsert %s: %w", table, err)
	}
	if d.count > 0 {
		return nil
	}
	// Remove zeroed rows.
	if _, err := e.Exec(fmt.Sprintf(`DELETE FROM %s WHERE %s AND qso_count <= 0`, table, where), keyArgs...); err != nil {
		return fmt.Errorf("worked index cleanup %s: %w", table, err)
	}
	// Recompute derived values only when the row survived a decrement.
	var remaining int
	if err := e.QueryRow(fmt.Sprintf(`SELECT qso_count FROM %s WHERE %s`, table, where), keyArgs...).Scan(&remaining); err != nil {
		return nil // row gone (deleted above) — nothing to fix
	}
	if remaining <= 0 {
		return nil
	}
	stampSQL := `printf('%s%06s', qso_date, COALESCE(time_on,'000000'))`
	switch table {
	case "worked_call":
		_, err := e.Exec(`UPDATE worked_call SET first_utc = (SELECT MIN(`+stampSQL+`) FROM qsos WHERE base_call = ?), last_utc = (SELECT MAX(`+stampSQL+`) FROM qsos WHERE base_call = ?) WHERE base_call = ?`, keyArgs[0], keyArgs[0], keyArgs[0])
		if err != nil {
			return fmt.Errorf("worked call edge recompute: %w", err)
		}
	case "worked_grid":
		_, err := e.Exec(`UPDATE worked_grid SET first_utc = (SELECT MIN(`+stampSQL+`) FROM qsos WHERE UPPER(SUBSTR(gridsquare,1,4)) = ?), last_utc = (SELECT MAX(`+stampSQL+`) FROM qsos WHERE UPPER(SUBSTR(gridsquare,1,4)) = ?) WHERE grid4 = ?`, keyArgs[0], keyArgs[0], keyArgs[0])
		if err != nil {
			return fmt.Errorf("worked grid edge recompute: %w", err)
		}
	case "worked_dxcc":
		dxcc, band, mode := keyArgs[0], keyArgs[1], keyArgs[2]
		level := `dxcc = ? AND (? = '' OR band = ?) AND (? = '' OR CASE WHEN submode != '' THEN submode ELSE mode END = ?)`
		_, err := e.Exec(`UPDATE worked_dxcc SET
			first_utc = (SELECT MIN(`+stampSQL+`) FROM qsos WHERE `+level+`),
			last_utc  = (SELECT MAX(`+stampSQL+`) FROM qsos WHERE `+level+`),
			source_mask = (SELECT `+sourceMaskSQL()+` FROM qsos WHERE `+level+`)
			WHERE dxcc = ? AND band = ? AND mode = ?`,
			dxcc, band, band, mode, mode,
			dxcc, band, band, mode, mode,
			dxcc, band, band, mode, mode,
			dxcc, band, mode)
		if err != nil {
			return fmt.Errorf("worked dxcc recompute: %w", err)
		}
	}
	return nil
}

// indexQSO applies one QSO (delta +1) or removes it (delta -1) from every
// aggregate it participates in. Incomplete data contributes conservatively:
// missing base call / grid / DXCC / band / mode simply skips the affected
// aggregates.
func indexQSO(e sqlExecutor, q *qso.QSO, delta int) error {
	contrib := contributionOf(q)
	d := indexDelta{
		stamp: contrib.stamp,
		count: delta,
		mask:  contrib.source,
	}
	if contrib.confirmed {
		d.confirmed = delta
	}

	if contrib.baseCall != "" {
		if err := applyCountDelta(e, "worked_call", "base_call", "?", []any{contrib.baseCall}, d); err != nil {
			return err
		}
	}

	if contrib.grid4 != "" {
		if err := applyCountDelta(e, "worked_grid", "grid4", "?", []any{contrib.grid4}, d); err != nil {
			return err
		}
	}

	if contrib.dxcc != "" {
		levels := [][2]string{{"", ""}}
		if contrib.band != "" {
			levels = append(levels, [2]string{contrib.band, ""})
		}
		if contrib.mode != "" {
			levels = append(levels, [2]string{"", contrib.mode})
		}
		if contrib.band != "" && contrib.mode != "" {
			levels = append(levels, [2]string{contrib.band, contrib.mode})
		}
		for _, lv := range levels {
			args := []any{contrib.dxcc, lv[0], lv[1]}
			if err := applyCountDelta(e, "worked_dxcc", "dxcc, band, mode", "?, ?, ?", args, d); err != nil {
				return err
			}
		}
	}

	// The completeness counters follow the same delta, so the panel never has
	// to scan the log to answer "is this log complete?" for a new callsign.
	return applyCoverageDelta(e, contrib, delta)
}

// indexContribution is everything a QSO adds to the index: the aggregate keys
// it joins, the timestamp that feeds first/last, its confirmation and its
// source. Two rows with an equal contribution add and remove exactly the same
// deltas, so an edit that leaves it alone — a comment, a name, a QTH, a
// no-op enrichment — does not have to touch the aggregates at all.
type indexContribution struct {
	baseCall  string
	grid4     string
	dxcc      string
	band      string
	mode      string
	stamp     string
	confirmed bool
	source    int

	// Completeness inputs, taken from the raw columns the coverage counters
	// count. They are part of the contribution so that skipping an index
	// update can never leave those counters behind.
	noDXCC bool
	noBand bool
	noMode bool
}

// contributionOf mirrors indexQSO field for field: if these ever disagree, an
// edit is either indexed when it should not be, or skipped when it must not.
func contributionOf(q *qso.QSO) indexContribution {
	c := indexContribution{
		baseCall:  qso.DeriveBaseCall(q.Call),
		dxcc:      strings.TrimSpace(q.DXCC),
		band:      q.Band,
		mode:      canonicalMode(q.Mode, q.Submode),
		stamp:     qsoStamp(q.QSODate, q.TimeOn),
		confirmed: q.Confirmed(),
		source:    sourceMaskBit(q.Source),
		noDXCC:    q.DXCC == "",
		noBand:    q.Band == "",
		noMode:    q.Mode == "",
	}
	if len(q.GridSquare) >= 4 {
		c.grid4 = strings.ToUpper(q.GridSquare[:4])
	}
	return c
}

// reindexQSO applies one edit's index delta: the stored row's previous
// contribution stops counting, its new one starts. Callers run it AFTER the row
// update, because the removal path re-derives edge timestamps and source masks
// from the whole qsos table and must see the new values.
//
// When the contribution did not change nothing is written. The -1/+1 pair would
// cancel out, but the removal half re-derives first/last/source_mask for every
// aggregate the contact belongs to — across a whole log that made a
// comment-only save the most expensive write there is (hundreds of ms at 100k
// QSOs, and the enrichment loop calls this from the UI path).
func reindexQSO(e sqlExecutor, id int64, before *qso.QSO) error {
	after, err := loadQSOForIndex(e, id)
	if err != nil {
		return fmt.Errorf("reload qso for worked index: %w", err)
	}
	if contributionOf(before) == contributionOf(after) {
		return nil
	}
	if err := indexQSO(e, before, -1); err != nil {
		return err
	}
	return indexQSO(e, after, 1)
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
	// Stamp the rules version with the same transaction, so the next start
	// does not rebuild again.
	if _, err := tx.Exec(
		`INSERT INTO worked_index_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		metaKeyRulesVersion, workedIndexRulesVersion); err != nil {
		return fmt.Errorf("worked index rules version: %w", err)
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

	now := time.Now().UTC().Format(time.RFC3339)
	stampSQL := `printf('%s%06s', qso_date, COALESCE(time_on,'000000'))`

	if _, err := e.Exec(`
		INSERT INTO worked_call(base_call, qso_count, confirmed_count, first_utc, last_utc, updated_utc)
		SELECT base_call, COUNT(*), SUM(`+confirmedSQL+`), MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE base_call != '' GROUP BY base_call`, now); err != nil {
		return fmt.Errorf("worked index rebuild call: %w", err)
	}
	if _, err := e.Exec(`
		INSERT INTO worked_grid(grid4, qso_count, confirmed_count, first_utc, last_utc, updated_utc)
		SELECT UPPER(SUBSTR(gridsquare,1,4)), COUNT(*), SUM(`+confirmedSQL+`), MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE LENGTH(gridsquare) >= 4 GROUP BY 1`, now); err != nil {
		return fmt.Errorf("worked index rebuild grid: %w", err)
	}
	if _, err := e.Exec(`
		INSERT INTO worked_dxcc(dxcc, band, mode, qso_count, confirmed_count, source_mask, first_utc, last_utc, updated_utc)
		SELECT dxcc, '', '', COUNT(*), SUM(`+confirmedSQL+`), `+sourceMaskSQL()+`, MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE dxcc != '' GROUP BY dxcc
		UNION ALL
		SELECT dxcc, band, '', COUNT(*), SUM(`+confirmedSQL+`), `+sourceMaskSQL()+`, MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE dxcc != '' AND band != '' GROUP BY dxcc, band
		UNION ALL
		SELECT dxcc, '', CASE WHEN submode != '' THEN submode ELSE mode END, COUNT(*), SUM(`+confirmedSQL+`), `+sourceMaskSQL()+`, MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE dxcc != '' AND mode != '' GROUP BY dxcc, 3
		UNION ALL
		SELECT dxcc, band, CASE WHEN submode != '' THEN submode ELSE mode END, COUNT(*), SUM(`+confirmedSQL+`), `+sourceMaskSQL()+`, MIN(`+stampSQL+`), MAX(`+stampSQL+`), ?
		FROM qsos WHERE dxcc != '' AND band != '' AND mode != '' GROUP BY dxcc, band, 3`,
		now, now, now, now); err != nil {
		return fmt.Errorf("worked index rebuild dxcc: %w", err)
	}

	// Seed the completeness counters from the same rebuild: Coverage answers
	// "is this log complete?" from them instead of scanning qsos.
	coverage, err := scanCoverage(e)
	if err != nil {
		return err
	}
	return writeCoverageCounters(e, coverage)
}

// qsoIndexColumns is the SELECT list of the qsos columns index maintenance
// needs from a row (unindex before edit/delete).
const qsoIndexColumns = `call, band, mode, submode, gridsquare, dxcc, qso_date, time_on, source,
	qsl_rcvd, lotw_qsl_rcvd, eqsl_qsl_rcvd`

// loadQSOForIndex reads the index-relevant fields of one row.
func loadQSOForIndex(q sqlExecutor, id int64) (*qso.QSO, error) {
	row := q.QueryRow(`SELECT `+qsoIndexColumns+` FROM qsos WHERE id = ?`, id)
	var out qso.QSO
	out.ID = id
	if err := row.Scan(&out.Call, &out.Band, &out.Mode, &out.Submode, &out.GridSquare, &out.DXCC,
		&out.QSODate, &out.TimeOn, &out.Source,
		&out.QSLRcvd, &out.LOTWQSLRcvd, &out.EQSLQSLRcvd); err != nil {
		return nil, err
	}
	return &out, nil
}

// ── Point lookups ──────────────────────────────────────────────────────────

// WorkedInfo is one index answer. An absent aggregate is the zero value
// (Count 0) and means "not worked in this log" — whether that can be reported
// as NEW depends on LogCoverage.
type WorkedInfo struct {
	Count          int
	ConfirmedCount int
	SourceMask     int
	FirstUTC       string
	LastUTC        string
	UpdatedUTC     string
}

// Worked reports whether the aggregate exists.
func (w WorkedInfo) Worked() bool { return w.Count > 0 }

// WorkedCall looks up a callsign (derived base call) in worked_call.
func WorkedCall(db *sql.DB, call string) (WorkedInfo, error) {
	var w WorkedInfo
	base := qso.DeriveBaseCall(call)
	if base == "" {
		return w, nil
	}
	err := db.QueryRow(
		`SELECT qso_count, confirmed_count, first_utc, last_utc, updated_utc
		 FROM worked_call WHERE base_call = ?`, base,
	).Scan(&w.Count, &w.ConfirmedCount, &w.FirstUTC, &w.LastUTC, &w.UpdatedUTC)
	if errors.Is(err, sql.ErrNoRows) {
		return w, nil
	}
	return w, err
}

// WorkedGrid looks up a 4-character Maidenhead grid (longer locators are
// reduced to their first four characters).
func WorkedGrid(db *sql.DB, grid string) (WorkedInfo, error) {
	var w WorkedInfo
	g := strings.ToUpper(strings.TrimSpace(grid))
	if len(g) < 4 {
		return w, nil
	}
	err := db.QueryRow(
		`SELECT qso_count, confirmed_count, first_utc, last_utc, updated_utc
		 FROM worked_grid WHERE grid4 = ?`, g[:4],
	).Scan(&w.Count, &w.ConfirmedCount, &w.FirstUTC, &w.LastUTC, &w.UpdatedUTC)
	if errors.Is(err, sql.ErrNoRows) {
		return w, nil
	}
	return w, err
}

// WorkedDXCC looks up a DXCC entity at one of the four aggregation levels.
// Empty band and mode mean "all bands" / "all modes". mode is the canonical
// mode (FT8, FT4, SSB …) — the same value stored in worked_dxcc.mode.
func WorkedDXCC(db *sql.DB, dxcc, band, mode string) (WorkedInfo, error) {
	var w WorkedInfo
	entity := strings.TrimSpace(dxcc)
	if entity == "" {
		return w, nil
	}
	err := db.QueryRow(
		`SELECT qso_count, confirmed_count, source_mask, first_utc, last_utc, updated_utc
		 FROM worked_dxcc WHERE dxcc = ? AND band = ? AND mode = ?`,
		entity, strings.TrimSpace(band), strings.TrimSpace(mode),
	).Scan(&w.Count, &w.ConfirmedCount, &w.SourceMask, &w.FirstUTC, &w.LastUTC, &w.UpdatedUTC)
	if errors.Is(err, sql.ErrNoRows) {
		return w, nil
	}
	return w, err
}

// ── Coverage and verdicts ──────────────────────────────────────────────────

// LogCoverage describes how complete the local log is. A missing aggregate
// only proves "new" when the local data covers everything that could contain
// a contact; otherwise the honest answer is UNKNOWN.
type LogCoverage struct {
	QSOCount       int
	UnresolvedDXCC int
	MissingBand    int
	MissingMode    int
	LastRemoteSync string // RFC3339 of the last clean remote synchronization; "" = never
}

// CallComplete reports whether an absent call aggregate can be trusted as NEW.
// A log is always complete for calls: whether a remote logbook still has to be
// synchronized is a caller concern (see LogCoverage.LastRemoteSync).
func (c LogCoverage) CallComplete() bool { return true }

// DXCCComplete reports whether an absent DXCC aggregate can be trusted as NEW.
func (c LogCoverage) DXCCComplete() bool {
	return c.UnresolvedDXCC == 0
}

// BandComplete reports whether an absent band aggregate can be trusted as NEW.
func (c LogCoverage) BandComplete() bool {
	return c.MissingBand == 0
}

// ModeComplete reports whether an absent mode aggregate can be trusted as NEW.
func (c LogCoverage) ModeComplete() bool {
	return c.MissingMode == 0
}

// Complete reports whether every worked-status answer can be trusted.
func (c LogCoverage) Complete() bool {
	return c.DXCCComplete() && c.MissingBand == 0 && c.MissingMode == 0
}

// Coverage returns the completeness of the local log (plus the stored
// remote-sync marker), which decides how much confidence a "not worked" answer
// deserves.
//
// The numbers are read from the counters kept in worked_index_meta, not by
// scanning qsos: this runs for every callsign the operator types, and the
// answer describes the whole logbook rather than the partner, so the scan was
// pure waste (41 ms at 100k contacts, on the panel's async lookup). When the
// counters are absent — an installation that has not rebuilt its index since
// they were introduced — the scan is used instead, so the answer stays correct.
func Coverage(db *sql.DB) (LogCoverage, error) {
	c, ok, err := readCoverageCounters(db)
	if err != nil {
		return c, err
	}
	if !ok {
		if c, err = scanCoverage(db); err != nil {
			return c, err
		}
		c.LastRemoteSync = metaValue(db, metaKeyLastRemoteSync)
	}
	return c, nil
}

// scanCoverage counts the log the slow way: one pass over qsos.
func scanCoverage(e sqlExecutor) (LogCoverage, error) {
	var c LogCoverage
	err := e.QueryRow(
		`SELECT COUNT(*),
			COALESCE(SUM(CASE WHEN COALESCE(dxcc,'') = '' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN COALESCE(band,'') = '' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN COALESCE(mode,'') = '' THEN 1 ELSE 0 END), 0)
		 FROM qsos`,
	).Scan(&c.QSOCount, &c.UnresolvedDXCC, &c.MissingBand, &c.MissingMode)
	if err != nil {
		return c, fmt.Errorf("log coverage: %w", err)
	}
	return c, nil
}

// readCoverageCounters loads the maintained numbers and the remote-sync marker
// in one query. ok is false when any counter is missing, which means "fall back
// to the scan".
func readCoverageCounters(db *sql.DB) (LogCoverage, bool, error) {
	var c LogCoverage
	rows, err := db.Query(
		`SELECT key, value FROM worked_index_meta WHERE key IN (?, ?, ?, ?, ?)`,
		metaKeyCoverageQSOs, metaKeyCoverageNoDXCC, metaKeyCoverageNoBand, metaKeyCoverageNoMode,
		metaKeyLastRemoteSync)
	if err != nil {
		return c, false, nil // table missing on a database mid-migration
	}
	defer rows.Close()
	seen := 0
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return c, false, fmt.Errorf("read coverage counters: %w", err)
		}
		n, _ := strconv.Atoi(value)
		switch key {
		case metaKeyCoverageQSOs:
			c.QSOCount, seen = n, seen|1
		case metaKeyCoverageNoDXCC:
			c.UnresolvedDXCC, seen = n, seen|2
		case metaKeyCoverageNoBand:
			c.MissingBand, seen = n, seen|4
		case metaKeyCoverageNoMode:
			c.MissingMode, seen = n, seen|8
		case metaKeyLastRemoteSync:
			c.LastRemoteSync = value
		}
	}
	if err := rows.Err(); err != nil {
		return c, false, fmt.Errorf("read coverage counters: %w", err)
	}
	return c, seen == 15, nil
}

// writeCoverageCounters stores the absolute numbers (rebuild path).
func writeCoverageCounters(e sqlExecutor, c LogCoverage) error {
	for _, kv := range []struct {
		key   string
		value int
	}{
		{metaKeyCoverageQSOs, c.QSOCount},
		{metaKeyCoverageNoDXCC, c.UnresolvedDXCC},
		{metaKeyCoverageNoBand, c.MissingBand},
		{metaKeyCoverageNoMode, c.MissingMode},
	} {
		if _, err := e.Exec(
			`INSERT INTO worked_index_meta (key, value) VALUES (?, ?)
			 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			kv.key, strconv.Itoa(kv.value)); err != nil {
			return fmt.Errorf("write coverage counter %s: %w", kv.key, err)
		}
	}
	return nil
}

// applyCoverageDelta keeps the counters in step with one QSO insert (+1) or
// removal (-1). A contact that carries DXCC, band and mode only moves the
// total, so the usual write is a single UPDATE.
//
// The statements update existing rows only: on a database whose counters have
// not been seeded yet nothing happens and Coverage scans instead, so the
// numbers can never start from a wrong base.
func applyCoverageDelta(e sqlExecutor, d indexContribution, delta int) error {
	deltas := make([]struct {
		key string
		by  int
	}, 0, 4)
	deltas = append(deltas, struct {
		key string
		by  int
	}{metaKeyCoverageQSOs, delta})
	if d.noDXCC {
		deltas = append(deltas, struct {
			key string
			by  int
		}{metaKeyCoverageNoDXCC, delta})
	}
	if d.noBand {
		deltas = append(deltas, struct {
			key string
			by  int
		}{metaKeyCoverageNoBand, delta})
	}
	if d.noMode {
		deltas = append(deltas, struct {
			key string
			by  int
		}{metaKeyCoverageNoMode, delta})
	}
	for _, item := range deltas {
		if _, err := e.Exec(
			`UPDATE worked_index_meta SET value = CAST(MAX(0, CAST(value AS INTEGER) + ?) AS TEXT)
			 WHERE key = ?`, item.by, item.key); err != nil {
			return fmt.Errorf("update coverage counter %s: %w", item.key, err)
		}
	}
	return nil
}

// WorkedState is the panel-facing verdict for one worked-status row.
type WorkedState int

const (
	StateUnknown WorkedState = iota // local data cannot prove either way
	StateNew
	StateWorked
)

// Verdict turns an index answer plus coverage certainty into a state. A
// positive answer is always WORKED (even partial data is proof); an absence is
// only NEW when the relevant dimension is known complete.
func Verdict(worked, complete bool) WorkedState {
	if worked {
		return StateWorked
	}
	if complete {
		return StateNew
	}
	return StateUnknown
}

// ── Index metadata ─────────────────────────────────────────────────────────

const (
	metaKeyRulesVersion   = "rules_version"
	metaKeyLastRemoteSync = "last_remote_sync"

	// Completeness counters for Coverage: the whole-log numbers behind every
	// NEW/UNKNOWN verdict. Maintained by indexQSO (same delta as the worked
	// aggregates) and seeded by RebuildWorkedIndex.
	metaKeyCoverageQSOs   = "coverage_qsos"
	metaKeyCoverageNoDXCC = "coverage_no_dxcc"
	metaKeyCoverageNoBand = "coverage_no_band"
	metaKeyCoverageNoMode = "coverage_no_mode"
)

// metaValue reads one worked_index_meta entry; a missing table/key is "".
func metaValue(db *sql.DB, key string) string {
	var v string
	if err := db.QueryRow(`SELECT value FROM worked_index_meta WHERE key = ?`, key).Scan(&v); err != nil {
		return ""
	}
	return v
}

// SetMetaValue writes one worked_index_meta entry (internal use + tests).
func SetMetaValue(db *sql.DB, key, value string) error {
	_, err := db.Exec(
		`INSERT INTO worked_index_meta (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	if err != nil {
		return fmt.Errorf("write worked index meta %s: %w", key, err)
	}
	return nil
}

// MarkRemoteSync records that the local log has been synchronized with the
// remote logbook. Coverage treats an absent marker as "remote history may be
// incomplete", so a never-synced installation reports UNKNOWN instead of NEW.
func MarkRemoteSync(db *sql.DB, at time.Time) error {
	return SetMetaValue(db, metaKeyLastRemoteSync, at.UTC().Format(time.RFC3339))
}
