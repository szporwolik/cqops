package store

import (
	"database/sql"
	"fmt"
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
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_dxc_spots_call ON dxc_spots(dx_call)`,
}

// Migrate runs all migrations against the given database.
// Existing columns/indexes are skipped (IF NOT EXISTS).
func Migrate(db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	return nil
}
