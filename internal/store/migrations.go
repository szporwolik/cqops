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
