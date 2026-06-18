package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// UpdateWavelogStatus sets the wavelog_uploaded status for a QSO.
// Retries on SQLITE_BUSY — the main thread may briefly hold a write lock
// during concurrent operations (e.g. WSJT-X insert + immediate upload).
func UpdateWavelogStatus(db *sql.DB, id int64, status string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		_, err = db.Exec(`UPDATE qsos SET wavelog_uploaded=? WHERE id=?`, status, id)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("update wavelog status: %w", err)
}

// NormalizeStationFields updates station_callsign, operator and my_gridsquare
// for a set of QSOs. Used before batch upload to Wavelog to align QSOs with
// the target station profile.
func NormalizeStationFields(db *sql.DB, ids []int64, stationCall, operator, grid string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE qsos SET station_callsign=?, operator=?, my_gridsquare=?, updated_at=? WHERE id=?`)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, id := range ids {
		if _, err := stmt.Exec(stationCall, operator, grid, now, id); err != nil {
			return fmt.Errorf("update qso %d: %w", id, err)
		}
	}
	return tx.Commit()
}
