package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/qso"
)

// Worked-status index DDL. Declared once so the base migration list and
// migrateBuildWorkedIndex cannot drift apart.
const (
	// confirmed_count counts contacts with any confirmation received (paper
	// QSL, LoTW or eQSL); source_mask records where the aggregated contacts
	// came from (see SourceMask* bits).
	workedCallDDL = `CREATE TABLE IF NOT EXISTS worked_call (
		base_call TEXT PRIMARY KEY,
		qso_count INTEGER NOT NULL DEFAULT 0,
		confirmed_count INTEGER NOT NULL DEFAULT 0,
		first_utc  TEXT NOT NULL DEFAULT '',
		last_utc   TEXT NOT NULL DEFAULT '',
		updated_utc TEXT NOT NULL DEFAULT ''
	)`
	workedGridDDL = `CREATE TABLE IF NOT EXISTS worked_grid (
		grid4     TEXT PRIMARY KEY,
		qso_count INTEGER NOT NULL DEFAULT 0,
		confirmed_count INTEGER NOT NULL DEFAULT 0,
		first_utc TEXT NOT NULL DEFAULT '',
		last_utc  TEXT NOT NULL DEFAULT '',
		updated_utc TEXT NOT NULL DEFAULT ''
	)`
	workedDxccDDL = `CREATE TABLE IF NOT EXISTS worked_dxcc (
		dxcc      TEXT NOT NULL,
		band      TEXT NOT NULL DEFAULT '',
		mode      TEXT NOT NULL DEFAULT '',
		qso_count INTEGER NOT NULL DEFAULT 0,
		confirmed_count INTEGER NOT NULL DEFAULT 0,
		source_mask INTEGER NOT NULL DEFAULT 0,
		first_utc TEXT NOT NULL DEFAULT '',
		last_utc  TEXT NOT NULL DEFAULT '',
		updated_utc TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (dxcc, band, mode)
	)`
	// worked_index_meta carries the index rules version (a rules change marks
	// every aggregate for rebuild without bumping the schema version) and the
	// timestamp of the last clean remote synchronization, which decides
	// whether a missing aggregate means NEW or UNKNOWN.
	workedMetaDDL = `CREATE TABLE IF NOT EXISTS worked_index_meta (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	)`
)

// migrations holds the ordered DDL statements that bring a new or existing
// SQLite database to the current schema. Every statement uses IF NOT EXISTS
// so migrations are safe to re-run (idempotent). v0.9.0 consolidated all
// historical ALTER TABLE additions into the base CREATE TABLE.
var migrations = []string{
	// ── qsos — main QSO table ────────────────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS qsos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,

		call TEXT NOT NULL,
		qso_date TEXT NOT NULL,
		time_on TEXT NOT NULL,
		time_off TEXT,

		band TEXT,
		freq REAL,
		mode TEXT NOT NULL,
		submode TEXT,

		rst_sent TEXT,
		rst_rcvd TEXT,

		gridsquare TEXT,
		name TEXT,
		qth TEXT,
		country TEXT,
		comment TEXT,
		notes TEXT,

		tx_pwr TEXT,

		distance REAL,
		bearing REAL,

		freq_rx REAL DEFAULT 0,
		sota_ref TEXT DEFAULT '',
		pota_ref TEXT DEFAULT '',
		wwff_ref TEXT DEFAULT '',
		my_sota_ref TEXT DEFAULT '',
		my_pota_ref TEXT DEFAULT '',
		my_wwff_ref TEXT DEFAULT '',
		iota TEXT DEFAULT '',
		sig TEXT DEFAULT '',
		sig_info TEXT DEFAULT '',

		wavelog_id INTEGER DEFAULT 0,
		wavelog_dirty INTEGER DEFAULT 0,
		wavelog_dirty_rev INTEGER DEFAULT 0,
		station_callsign TEXT,
		operator TEXT,
		my_gridsquare TEXT,
		my_rig TEXT,
		my_antenna TEXT,

		cq_zone TEXT DEFAULT '',
		itu_zone TEXT DEFAULT '',
		contest_id TEXT DEFAULT '',
		exch_sent TEXT DEFAULT '',
		exch_rcvd TEXT DEFAULT '',
		stx INTEGER DEFAULT 0,
		srx INTEGER DEFAULT 0,
		stx_string TEXT DEFAULT '',
		srx_string TEXT DEFAULT '',
		contest_adif_id TEXT DEFAULT '',
		my_cq_zone TEXT DEFAULT '',
		my_itu_zone TEXT DEFAULT '',
		my_dxcc TEXT DEFAULT '',
		my_sig TEXT DEFAULT '',
		my_sig_info TEXT DEFAULT '',

		dxcc TEXT DEFAULT '',
		dxcc_source TEXT NOT NULL DEFAULT '',
		base_call TEXT NOT NULL DEFAULT '',
		source TEXT NOT NULL DEFAULT 'manual',

		-- ADIF confirmation status: 'Y' received, 'N' not received,
		-- '' not provided. Confirmations are aggregated into the worked
		-- index (confirmed_count).
		qsl_rcvd TEXT NOT NULL DEFAULT '',
		lotw_qsl_rcvd TEXT NOT NULL DEFAULT '',
		eqsl_qsl_rcvd TEXT NOT NULL DEFAULT '',

		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,

	// ── qsos indexes ─────────────────────────────────────────────────────────
	`CREATE INDEX IF NOT EXISTS idx_qsos_qso_date ON qsos(qso_date)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_gridsquare ON qsos(gridsquare)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_source ON qsos(source)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_wavelog_id ON qsos(wavelog_id)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_id ON qsos(contest_id)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_adif_id ON qsos(contest_adif_id)`,
	// Ordered contest arms: the contest list queries run two index-ordered
	// scans (contest_id and contest_adif_id) merged in Go — each index serves
	// the ORDER BY directly, so no per-refresh sort of the contest subset.
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_date_time ON qsos(contest_id, qso_date DESC, time_on DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_adif_date ON qsos(contest_adif_id, qso_date DESC, time_on DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_time ON qsos(qso_date DESC, time_on DESC)`,

	`CREATE INDEX IF NOT EXISTS idx_qsos_country_base ON qsos(country, base_call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_dxcc ON qsos(dxcc)`,

	`CREATE INDEX IF NOT EXISTS idx_qsos_base_call ON qsos(base_call)`,

	// Partial index for the pending-sync queue: rows with a remote id and a
	// pending edit. The predicate matches CountDirtyQSOs/ListDirtyQSOs
	// exactly, so those queries can serve the backlog without scanning the
	// synced logbook.
	`CREATE INDEX IF NOT EXISTS idx_qsos_dirty ON qsos(wavelog_dirty, id DESC) WHERE wavelog_id > 0 AND wavelog_dirty = 1`,

	// ── worked-status index — fast worked DXCC/call/grid answers ────────────
	// Maintained incrementally by every QSO write path and rebuilt by
	// RebuildWorkedIndex. The normalized qsos table stays the source of
	// truth; these tables are disposable summaries.
	//
	// The keys are global on purpose: CQOps keeps one SQLite file per logbook
	// (app.SwitchLogbook), so logbook/station identity is the file itself.
	// Do NOT add logbook_id/station_profile_id columns without first moving
	// every logbook into a shared database — the primary keys below (and the
	// four lookup shapes the worked panel issues) assume a single logbook.
	//
	// Similarly, the '' band/mode values are aggregation sentinels: they mean
	// "all bands" / "all modes", not "unknown".
	workedCallDDL,
	workedGridDDL,
	workedDxccDDL,
	workedMetaDDL,

	// ── schema v2: composite indexes for dupe-check and dedup queries ─────────
	`CREATE INDEX IF NOT EXISTS idx_qsos_call_band_mode_date ON qsos(call, band, mode, qso_date)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_base_call_band_mode_date ON qsos(base_call, band, mode, qso_date)`,

	// ── dxc_spots — DX Cluster spot cache ────────────────────────────────────
	`CREATE TABLE IF NOT EXISTS dxc_spots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dx_call TEXT NOT NULL,
		frequency REAL NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		spotter TEXT NOT NULL DEFAULT '',
		band TEXT NOT NULL DEFAULT '',
		mode TEXT NOT NULL DEFAULT '',
		mode_cat TEXT NOT NULL DEFAULT '',
		dx_cont TEXT NOT NULL DEFAULT '',
		spot_cont TEXT NOT NULL DEFAULT '',
		dxcc TEXT NOT NULL DEFAULT '',
		received_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_dxc_spots_received ON dxc_spots(received_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_dxc_spots_call ON dxc_spots(dx_call)`,
	// Composite index for band + time queries — used by the DXC path line
	// above the QSO form to efficiently fetch recent spots on the current band.
	`CREATE INDEX IF NOT EXISTS idx_dxc_spots_band_time ON dxc_spots(band, received_at)`,

	// ── psk_spots — PSK Reporter spot cache ──────────────────────────────────
	`CREATE TABLE IF NOT EXISTS psk_spots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		receiver_call TEXT NOT NULL,
		receiver_loc TEXT NOT NULL DEFAULT '',
		frequency REAL NOT NULL,
		snr INTEGER DEFAULT 0,
		mode TEXT NOT NULL DEFAULT '',
		flow_start INTEGER NOT NULL,
		fetch_time INTEGER NOT NULL,
		station_call TEXT NOT NULL DEFAULT ''
	)`,
	`CREATE INDEX IF NOT EXISTS idx_psk_spots_station_flow ON psk_spots(station_call, flow_start)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_psk_spots_uniq ON psk_spots(receiver_call, frequency, mode, flow_start)`,
	// PurgeOldPSKSpots scans this — without it every PSK fetch batch
	// full-scans a table that can hold a week of spots.
	`CREATE INDEX IF NOT EXISTS idx_psk_spots_flow_start ON psk_spots(flow_start)`,
}

// schemaVersion is the current database schema version. Bump this when
// the schema changes in a way that requires data migration (new columns,
// index rebuilds, backfills). DDL-only changes that use IF NOT EXISTS
// don't require a bump — they're idempotent.
//
// Version history:
//
//	1 — v0.9.0  consolidated schema, base_call backfill, dxcc column
//	2 — v0.10.0 worked-status index tables and the one-time build
//	3 — v0.11.2 rebuild the worked index: edits indexed the caller's object
//	           (losing dxcc on comment-only edits) and enrichment / DXCC
//	           backfill wrote qsos without touching the index
//	4 — v0.11.3 DXCC provenance + confirmation status on qsos, confirmed_count
//	           and source_mask in the worked index, and worked_index_meta
//	           (index rules version + last clean remote sync)
const schemaVersion = 4

// Migrate runs all migrations. Safe to call multiple times — every
// statement uses IF NOT EXISTS guards, and PRAGMA user_version prevents
// re-running migrations that have already been applied.
//
// Column additions for schema version 1 (dxcc) are applied unconditionally
// because CREATE TABLE IF NOT EXISTS won't add columns to tables that
// already existed before the column was introduced.
//
// The one-time base_call backfill runs on the first startup where any
// row still has an empty base_call (covers direct upgrades from
// pre-v0.8.7 databases).
func Migrate(db *sql.DB) error {
	var current int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	// Unconditional column additions — these must run even when
	// user_version is already at the current schema because
	// CREATE TABLE IF NOT EXISTS won't add columns to existing tables.
	// ALTER TABLE ADD COLUMN is idempotent — SQLite ignores duplicates.
	//
	// Must run AFTER DDL migrations (the tables must exist first).

	// Already at or above the current schema — nothing to do for DDL.
	if current >= schemaVersion {
		// Still run column additions for existing databases.
		if err := migrateAddColumn(db, "qsos", "dxcc", "TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("add column dxcc: %w", err)
		}
		if err := migrateAddColumn(db, "qsos", "wavelog_id", "INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("add column wavelog_id: %w", err)
		}
		if err := migrateAddColumn(db, "qsos", "wavelog_dirty", "INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("add column wavelog_dirty: %w", err)
		}
		if err := migrateAddColumn(db, "qsos", "wavelog_dirty_rev", "INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("add column wavelog_dirty_rev: %w", err)
		}
		if err := migrateAddColumn(db, "qsos", "base_call", "TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("add column base_call: %w", err)
		}
		if err := migrateAddColumn(db, "qsos", "submode", "TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("add column submode: %w", err)
		}
		if err := ensureColumnIndexes(db); err != nil {
			return err
		}
		if err := migrateDropWavelogFlag(db); err != nil {
			return err
		}
		if err := migrateDropRedundantIndexes(db); err != nil {
			return err
		}
		if err := migrateBuildWorkedIndex(db); err != nil {
			return err
		}
		// A normalization/aggregation rule change bumps workedIndexRulesVersion;
		// the stored version tells us whether the aggregates must be rebuilt.
		return ensureWorkedIndexRules(db, false)
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			if strings.Contains(m, "DROP INDEX") {
				continue
			}
			// An index on a column this old database does not have yet.
			// The column is added below and ensureColumnIndexes creates
			// the index afterwards.
			if strings.Contains(m, "CREATE INDEX") &&
				(strings.Contains(err.Error(), "no such column") || strings.Contains(err.Error(), "no such table")) {
				continue
			}
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	// Add columns that may be missing from upgraded databases.
	// Safe after CREATE TABLE — the table exists by now.
	if err := migrateAddColumn(db, "qsos", "dxcc", "TEXT DEFAULT ''"); err != nil {
		return fmt.Errorf("add column dxcc: %w", err)
	}
	if err := migrateAddColumn(db, "qsos", "wavelog_id", "INTEGER DEFAULT 0"); err != nil {
		return fmt.Errorf("add column wavelog_id: %w", err)
	}
	if err := migrateAddColumn(db, "qsos", "wavelog_dirty", "INTEGER DEFAULT 0"); err != nil {
		return fmt.Errorf("add column wavelog_dirty: %w", err)
	}
	if err := migrateAddColumn(db, "qsos", "wavelog_dirty_rev", "INTEGER DEFAULT 0"); err != nil {
		return fmt.Errorf("add column wavelog_dirty_rev: %w", err)
	}
	if err := migrateAddColumn(db, "qsos", "base_call", "TEXT DEFAULT ''"); err != nil {
		return fmt.Errorf("add column base_call: %w", err)
	}
	if err := migrateAddColumn(db, "qsos", "submode", "TEXT DEFAULT ''"); err != nil {
		return fmt.Errorf("add column submode: %w", err)
	}
	if err := ensureColumnIndexes(db); err != nil {
		return err
	}
	if err := migrateDropWavelogFlag(db); err != nil {
		return err
	}
	if err := migrateDropRedundantIndexes(db); err != nil {
		return err
	}
	if err := migrateBuildWorkedIndex(db); err != nil {
		return err
	}
	// One-time backfill: if any QSO row still has an empty base_call
	// (direct upgrade from pre-v0.8.7 or first migration after schema
	// consolidation), populate it now. Subsequent calls are skipped
	// by the user_version guard above.
	var pending int
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos WHERE base_call = '' OR base_call IS NULL LIMIT 1`).Scan(&pending); err == nil && pending > 0 {
		rows, err := db.Query(`SELECT id, call FROM qsos WHERE base_call = '' OR base_call IS NULL`)
		if err != nil {
			return fmt.Errorf("backfill query: %w", err)
		}
		defer rows.Close()

		type update struct {
			id int64
			bc string
		}
		var updates []update
		for rows.Next() {
			var id int64
			var call string
			if err := rows.Scan(&id, &call); err != nil {
				rows.Close()
				return fmt.Errorf("backfill scan: %w", err)
			}
			if bc := qso.DeriveBaseCall(call); bc != "" {
				updates = append(updates, update{id, bc})
			}
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("backfill rows: %w", err)
		}

		for _, u := range updates {
			if _, err := db.Exec(`UPDATE qsos SET base_call = ? WHERE id = ?`, u.bc, u.id); err != nil {
				return fmt.Errorf("backfill update: %w", err)
			}
		}
	}

	// Existing indexes can have drifted from qsos (v3: edits indexed the
	// caller's object; enrichment/backfill wrote qsos without touching the
	// index), and any later change to the aggregation rules must also rebuild.
	// Rebuild once, only *after* every backfill above — the rebuild reads
	// qsos, so a row that gains its base_call below would otherwise be missing
	// from worked_call until the next rebuild. Later opens skip this through
	// the stored rules version.
	if err := ensureWorkedIndexRules(db, current < schemaVersion); err != nil {
		return err
	}

	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}
	return nil
}

// ensureColumnIndexes creates the indexes that depend on columns possibly
// added by migrateAddColumn. Idempotent via IF NOT EXISTS. Runs on both
// migration paths, so existing databases pick up newly added indexes.
//
// A column referenced by an index may still be absent when migrating a very
// old database whose original CREATE TABLE predates it (e.g. `country` in the
// v0.10 schema). Such indexes are skipped rather than failing the migration —
// the column is never added by this code path, so the index is meaningless
// for that database anyway.
func ensureColumnIndexes(db *sql.DB) error {
	for _, idx := range []string{
		`CREATE INDEX IF NOT EXISTS idx_qsos_dxcc ON qsos(dxcc)`,
		`CREATE INDEX IF NOT EXISTS idx_qsos_wavelog_id ON qsos(wavelog_id)`,
		// NOCASE lets the DXCC-scope lookup match country case-insensitively
		// through an index; the plain idx_qsos_country cannot serve it.
		`CREATE INDEX IF NOT EXISTS idx_qsos_country_nocase ON qsos(country COLLATE NOCASE)`,
		// Partial index for the "never uploaded" Wavelog queue.
		`CREATE INDEX IF NOT EXISTS idx_qsos_unsent ON qsos(id DESC) WHERE COALESCE(wavelog_id, 0) = 0`,
	} {
		if _, err := db.Exec(idx); err != nil {
			if strings.Contains(err.Error(), "no such column") || strings.Contains(err.Error(), "no such table") {
				continue
			}
			return fmt.Errorf("create index: %w", err)
		}
	}
	return nil
}

// migrateDropRedundantIndexes removes indexes that have no query consumer or
// whose coverage is subsumed by compound indexes. Every maintained index
// costs a b-tree write on each QSO save, so unused indexes are pure write
// amplification. The dropped set was verified against every SQL statement in
// the repository (EXPLAIN QUERY PLAN over a seeded logbook): the single-
// column call/band/mode lookups are served by idx_qsos_call_band_mode_date or
// idx_qsos_base_call_band_mode_date, contest lookups by contest_id +
// contest_adif_id, and ordering by idx_qsos_date_time.
func migrateDropRedundantIndexes(db *sql.DB) error {
	for _, idx := range []string{
		"idx_qsos_country", "idx_qsos_submode",
		"idx_qsos_call", "idx_qsos_band", "idx_qsos_mode",
		"idx_qsos_date_time_call", "idx_qsos_date_operator",
		"idx_qsos_date_call_band_mode", "idx_qsos_contest_call_band_mode",
	} {
		if _, err := db.Exec(`DROP INDEX IF EXISTS ` + idx); err != nil {
			return fmt.Errorf("drop redundant index %s: %w", idx, err)
		}
	}
	return nil
}

// migrateBuildWorkedIndex creates the worked-status index tables (they live
// in the base DDL for fresh databases but must also exist on databases that
// are already at the current schema and skip the base list) and builds the
// index once for databases that already contain QSOs but predate it. Later
// opens skip the rebuild: the tables are non-empty (the cheap count answers),
// and the index is maintained incrementally from then on.
func migrateBuildWorkedIndex(db *sql.DB) error {
	for _, ddl := range []string{workedCallDDL, workedGridDDL, workedDxccDDL, workedMetaDDL} {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("worked index table: %w", err)
		}
	}
	if err := migrateAddWorkedColumns(db); err != nil {
		return err
	}
	var worked int
	if err := db.QueryRow(`SELECT COUNT(*) FROM worked_call`).Scan(&worked); err != nil {
		return fmt.Errorf("worked index probe: %w", err)
	}
	if worked > 0 {
		return nil
	}
	var qsos int
	if err := db.QueryRow(`SELECT COUNT(*) FROM qsos`).Scan(&qsos); err != nil {
		return fmt.Errorf("worked index qsos probe: %w", err)
	}
	if qsos == 0 {
		return nil
	}
	return RebuildWorkedIndex(db)
}

// migrateAddWorkedColumns adds the columns introduced after a table was first
// created. CREATE TABLE IF NOT EXISTS never alters an existing table, so both
// migration paths call this unconditionally; ALTER TABLE ADD COLUMN is an
// idempotent no-op here (duplicate columns are ignored by migrateAddColumn).
func migrateAddWorkedColumns(db *sql.DB) error {
	adds := []struct{ table, column, colType string }{
		{"qsos", "dxcc_source", "TEXT NOT NULL DEFAULT ''"},
		{"qsos", "qsl_rcvd", "TEXT NOT NULL DEFAULT ''"},
		{"qsos", "lotw_qsl_rcvd", "TEXT NOT NULL DEFAULT ''"},
		{"qsos", "eqsl_qsl_rcvd", "TEXT NOT NULL DEFAULT ''"},
		{"worked_call", "confirmed_count", "INTEGER NOT NULL DEFAULT 0"},
		{"worked_call", "updated_utc", "TEXT NOT NULL DEFAULT ''"},
		{"worked_grid", "confirmed_count", "INTEGER NOT NULL DEFAULT 0"},
		{"worked_grid", "updated_utc", "TEXT NOT NULL DEFAULT ''"},
		{"worked_dxcc", "confirmed_count", "INTEGER NOT NULL DEFAULT 0"},
		{"worked_dxcc", "source_mask", "INTEGER NOT NULL DEFAULT 0"},
		{"worked_dxcc", "updated_utc", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, a := range adds {
		if err := migrateAddColumn(db, a.table, a.column, a.colType); err != nil {
			return fmt.Errorf("add column %s.%s: %w", a.table, a.column, err)
		}
	}
	return nil
}

// ensureWorkedIndexRules rebuilds the worked index when the aggregation rules
// stored in the database differ from the running binary's, or when the schema
// version was just upgraded (older databases may carry aggregates written by
// a version with different indexing behaviour). A rules change bumps
// workedIndexRulesVersion in worked_index.go, and every installation repairs
// itself on the next start — no schema bump required.
func ensureWorkedIndexRules(db *sql.DB, upgrade bool) error {
	var stored string
	err := db.QueryRow(`SELECT value FROM worked_index_meta WHERE key = 'rules_version'`).Scan(&stored)
	if err != nil && err != sql.ErrNoRows &&
		!strings.Contains(err.Error(), "no such table") && !strings.Contains(err.Error(), "no such column") {
		return fmt.Errorf("read worked index rules version: %w", err)
	}
	if !upgrade && err == nil && stored == workedIndexRulesVersion {
		return nil
	}
	if err := RebuildWorkedIndex(db); err != nil {
		return fmt.Errorf("rebuild worked index: %w", err)
	}
	if _, err := db.Exec(
		`INSERT INTO worked_index_meta (key, value) VALUES ('rules_version', ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, workedIndexRulesVersion); err != nil {
		return fmt.Errorf("write worked index rules version: %w", err)
	}
	return nil
}

// migrateDropWavelogFlag removes the legacy wavelog_uploaded status column.
// Since v0.11.0 the remote id (wavelog_id > 0) is the single source of truth
// for "uploaded". Idempotent: a missing column is a no-op.
func migrateDropWavelogFlag(db *sql.DB) error {
	// The released v0.11.0 schema indexed wavelog_uploaded — the index must
	// go first, otherwise SQLite refuses to drop the column.
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_qsos_wavelog_uploaded`); err != nil {
		return fmt.Errorf("drop index wavelog_uploaded: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE qsos DROP COLUMN wavelog_uploaded`); err != nil {
		if strings.Contains(err.Error(), "no such column") {
			return nil // already dropped
		}
		return fmt.Errorf("drop column wavelog_uploaded: %w", err)
	}
	return nil
}

// migrateAddColumn adds a column to a table if it doesn't already exist. Uses ALTER TABLE ADD COLUMN with error suppression for the "duplicate
// column name" case — SQLite's ALTER TABLE is idempotent this way.
func migrateAddColumn(db *sql.DB, table, column, colType string) error {
	_, err := db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, colType))
	if err != nil && strings.Contains(err.Error(), "duplicate column name") {
		return nil // already exists — no-op
	}
	return err
}
