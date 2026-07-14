package store

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/szporwolik/cqops/internal/qso"
)

var migrations = []string{
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
		dxcc TEXT DEFAULT '',
		my_sig TEXT DEFAULT '',
		my_sig_info TEXT DEFAULT '',

		source TEXT NOT NULL DEFAULT 'manual',

		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_call ON qsos(call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_qso_date ON qsos(qso_date)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_band ON qsos(band)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_mode ON qsos(mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_gridsquare ON qsos(gridsquare)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_source ON qsos(source)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_wavelog_uploaded ON qsos(wavelog_uploaded)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_id ON qsos(contest_id)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_time ON qsos(qso_date DESC, time_on DESC)`,

	// sig_info column added in v0.8.1 — ALTER for existing databases.
	`ALTER TABLE qsos ADD COLUMN sig_info TEXT DEFAULT ''`,

	// base_call column added in v0.8.7 — enables index-friendly callsign lookups
	// by extracting the core callsign from prefixed/suffixed variants
	// (e.g. "DL/SP9MOA/P" → "SP9MOA"). Avoids LIKE '%/call' table scans.
	`ALTER TABLE qsos ADD COLUMN base_call TEXT DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_base_call ON qsos(base_call)`,

	// Backfill base_call for existing rows (v0.8.14).
	// Must run as a separate statement since SQLite doesn't support
	// subqueries referencing the same table in UPDATE.
	// We iterate in Go below with the post-migration hook.
	// Marker: base_call_backfill is handled in backfillBaseCall().

	// Dashboard performance indexes (v0.8.9):
	// - country lookup for New DXCC / dupe detection
	// - date+time+base_call for filtered recent QSO scanning
	`CREATE INDEX IF NOT EXISTS idx_qsos_country ON qsos(country)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_country_base ON qsos(country, base_call)`,

	// dxcc column added in v0.8.14 — remote station DXCC entity number (from QRZ).
	`ALTER TABLE qsos ADD COLUMN dxcc TEXT DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_time_call ON qsos(qso_date, time_on, base_call)`,

	// Dashboard stats operator count index (v0.9.x):
	// - speeds up COUNT(DISTINCT operator) in GetDashboardStats
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_operator ON qsos(qso_date, operator)`,

	// Covering indexes for dupe-set queries (v0.10.x):
	// - DXCDupeSet / DXC path line dupe markers query by date or contest_id
	//   selecting DISTINCT call, band, mode. These indexes let SQLite
	//   answer the query from the index alone — no table scan needed.
	`CREATE INDEX IF NOT EXISTS idx_qsos_date_call_band_mode ON qsos(qso_date, call, band, mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_call_band_mode ON qsos(contest_id, call, band, mode)`,

	// Composite index for contest-filtered ListQSOs — covers
	//   WHERE contest_id = ? ORDER BY qso_date DESC, time_on DESC
	// so SQLite can satisfy both the filter and sort from one index.
	`CREATE INDEX IF NOT EXISTS idx_qsos_contest_date_time ON qsos(contest_id, qso_date DESC, time_on DESC)`,

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
	// The UNIQUE index may fail on databases that previously had
	// a botched migration (duplicate rows were inserted while the
	// index was temporarily missing). Purge conflicting rows first.
	`DELETE FROM dxc_spots WHERE rowid NOT IN (SELECT MIN(rowid) FROM dxc_spots GROUP BY dx_call)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_dxc_spots_call ON dxc_spots(dx_call)`,
}

// Migrate runs all migrations against the given database.
// Existing columns/indexes are skipped (IF NOT EXISTS or error ignored).
func Migrate(db *sql.DB) error {
	fmt.Fprintln(os.Stderr, "CQOps: running database migrations...")
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			// ALTER TABLE ADD COLUMN fails if the column already exists.
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			// DROP INDEX may fail if the index doesn't exist or
			// the database engine doesn't support the syntax.
			if strings.Contains(m, "DROP INDEX") {
				continue
			}
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	// Backfill base_call for rows that have it empty (pre-v0.8.7 QSOs).
	fmt.Fprintln(os.Stderr, "CQOps: backfilling base_call...")
	if err := backfillBaseCall(db); err != nil {
		return fmt.Errorf("backfill base_call: %w", err)
	}
	fmt.Fprintln(os.Stderr, "CQOps: migrations complete.")
	return nil
}

// backfillBaseCall fills empty base_call columns for existing QSOs.
func backfillBaseCall(db *sql.DB) error {
	rows, err := db.Query(`SELECT id, call FROM qsos WHERE base_call = '' OR base_call IS NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var updates []struct {
		id       int64
		baseCall string
	}
	for rows.Next() {
		var id int64
		var call string
		if err := rows.Scan(&id, &call); err != nil {
			return err
		}
		bc := qso.DeriveBaseCall(call)
		if bc != "" {
			updates = append(updates, struct {
				id       int64
				baseCall string
			}{id, bc})
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for _, u := range updates {
		if _, err := db.Exec(`UPDATE qsos SET base_call = ? WHERE id = ?`, u.baseCall, u.id); err != nil {
			return err
		}
	}
	return nil
}
