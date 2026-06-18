package store

import (
	"database/sql"
	"fmt"
	"strings"
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

		station_callsign TEXT,
		operator TEXT,
		my_gridsquare TEXT,
		my_rig TEXT,
		my_antenna TEXT,

		source TEXT NOT NULL DEFAULT 'manual',

		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`,
	`ALTER TABLE qsos ADD COLUMN notes TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN tx_pwr TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN distance REAL DEFAULT 0`,
	`ALTER TABLE qsos ADD COLUMN bearing REAL DEFAULT 0`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_call ON qsos(call)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_qso_date ON qsos(qso_date)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_band ON qsos(band)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_mode ON qsos(mode)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_gridsquare ON qsos(gridsquare)`,
	`ALTER TABLE qsos ADD COLUMN freq_rx REAL DEFAULT 0`,
	`ALTER TABLE qsos ADD COLUMN sota_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN pota_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN wwff_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN my_sota_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN my_pota_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN my_wwff_ref TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN iota TEXT DEFAULT ''`,
	`ALTER TABLE qsos ADD COLUMN wavelog_uploaded TEXT DEFAULT ''`,
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
	`CREATE INDEX IF NOT EXISTS idx_psk_spots_station ON psk_spots(station_call)`,
	`CREATE INDEX IF NOT EXISTS idx_psk_spots_flow_start ON psk_spots(flow_start)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_psk_spots_uniq ON psk_spots(receiver_call, frequency, mode, flow_start)`,
	`CREATE TABLE IF NOT EXISTS dxc_spots (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		dx_call TEXT NOT NULL,
		frequency REAL NOT NULL,
		comment TEXT NOT NULL DEFAULT '',
		spotter TEXT NOT NULL DEFAULT '',
		received_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_dxc_spots_received ON dxc_spots(received_at)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_dxc_spots_uniq ON dxc_spots(dx_call, frequency, received_at)`,
	`ALTER TABLE dxc_spots ADD COLUMN band TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE dxc_spots ADD COLUMN mode TEXT NOT NULL DEFAULT ''`,
	`DELETE FROM dxc_spots WHERE id NOT IN (SELECT id FROM (SELECT MAX(id) AS id FROM dxc_spots GROUP BY dx_call))`,
	`DROP INDEX IF EXISTS idx_dxc_spots_uniq`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_dxc_spots_call ON dxc_spots(dx_call)`,
	`ALTER TABLE dxc_spots ADD COLUMN mode_cat TEXT NOT NULL DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_source ON qsos(source)`,
	`CREATE INDEX IF NOT EXISTS idx_qsos_wavelog_uploaded ON qsos(wavelog_uploaded)`,
	`DROP INDEX IF EXISTS idx_psk_spots_station`,
	`CREATE INDEX IF NOT EXISTS idx_psk_spots_station_flow ON psk_spots(station_call, flow_start)`,
}

func Migrate(db *sql.DB) error {
	for i, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "duplicate column name") ||
				strings.Contains(errStr, "already exists") {
				continue
			}
			return fmt.Errorf("migration %d: %w", i, err)
		}
	}
	return nil
}
