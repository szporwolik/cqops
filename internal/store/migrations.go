package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/szporwolik/cqops/internal/qso"
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
		wavelog_uploaded TEXT DEFAULT '',

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
		base_call TEXT DEFAULT '',
		source TEXT NOT NULL DEFAULT 'manual',

		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,

	// ── qsos indexes ─────────────────────────────────────────────────────────
	`CREATE INDEX IF NOT EXISTS idx_qsos_call ON qsos(call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_qso_date ON qsos(qso_date)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_band ON qsos(band)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_mode ON qsos(mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_gridsquare ON qsos(gridsquare)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_source ON qsos(source)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_wavelog_uploaded ON qsos(wavelog_uploaded)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_id ON qsos(contest_id)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_time ON qsos(qso_date DESC, time_on DESC)`,

	`CREATE INDEX IF NOT EXISTS idx_qsos_country ON qsos(country)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_country_base ON qsos(country, base_call)`,

	`CREATE INDEX IF NOT EXISTS idx_qsos_base_call ON qsos(base_call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_time_call ON qsos(qso_date, time_on, base_call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_operator ON qsos(qso_date, operator)`,

	`CREATE INDEX IF NOT EXISTS idx_qsos_date_call_band_mode ON qsos(qso_date, call, band, mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_call_band_mode ON qsos(contest_id, call, band, mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_date_time ON qsos(contest_id, qso_date DESC, time_on DESC)`,

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
}

// schemaVersion is the current database schema version. Bump this when
// the schema changes in a way that requires data migration (new columns,
// index rebuilds, backfills). DDL-only changes that use IF NOT EXISTS
// don't require a bump — they're idempotent.
//
// Version history:
//
//	1 — v0.9.0  consolidated schema, base_call backfill, dxcc column
const schemaVersion = 1

// Migrate runs all migrations. Safe to call multiple times — every
// statement uses IF NOT EXISTS guards, and PRAGMA user_version prevents
// re-running migrations that have already been applied.
//
// The one-time base_call backfill runs on the first startup where any
// row still has an empty base_call (covers direct upgrades from
// pre-v0.8.7 databases).
func Migrate(db *sql.DB) error {
	var current int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	// Already at or above the current schema — nothing to do.
	if current >= schemaVersion {
		return nil
	}

	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			if strings.Contains(m, "DROP INDEX") {
				continue
			}
			return fmt.Errorf("migration %d: %w", i, err)
		}
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

	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("write schema version: %w", err)
	}
	return nil
}
