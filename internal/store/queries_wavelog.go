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
// for a set of QSOs. Uses a single UPDATE with WHERE id IN (...) instead of
// per-row prepared statements, reducing DB round-trips for batch operations.
func NormalizeStationFields(db *sql.DB, ids []int64, stationCall, operator, grid string) error {
	if len(ids) == 0 {
		return nil
	}

	// Build WHERE id IN (?,?,...) clause.
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+4)
	args = append(args, stationCall, operator, grid, time.Now().UTC().Format(time.RFC3339))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := `UPDATE qsos SET station_callsign=?, operator=?, my_gridsquare=?, updated_at=? WHERE id IN (` +
		strings.Join(placeholders, ",") + `)`

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("normalize station fields: %w", err)
	}
	return nil
}
