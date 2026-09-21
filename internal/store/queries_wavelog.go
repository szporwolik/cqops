package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// SetWavelogID stores the remote Wavelog QSO id for a local QSO. The remote
// id is the single source of truth for "uploaded": wavelog_id > 0 means the
// QSO exists in Wavelog. Same SQLITE_BUSY retry policy as the other writers.
func SetWavelogID(db *sql.DB, id, remoteID int64) error {
	var err error
	sleep := 100 * time.Millisecond
	for attempt := 0; attempt < 5; attempt++ {
		_, err = db.Exec(`UPDATE qsos SET wavelog_id=? WHERE id=?`, remoteID, id)
		if err == nil {
			return nil
		}
		if !strings.Contains(err.Error(), "database is locked") {
			break
		}
		time.Sleep(sleep)
		sleep *= 2
	}
	return fmt.Errorf("set wavelog id: %w", err)
}

// NormalizeStationFields updates station_callsign, operator and my_gridsquare
// for a set of QSOs. Only the fields with a non-empty replacement value are
// updated — empty means "keep what is stored" (e.g. preserving original
// station callsigns when only operator/grid mismatches were confirmed). Uses
// a single UPDATE with WHERE id IN (...) instead of per-row prepared
// statements, reducing DB round-trips for batch operations.
func NormalizeStationFields(db *sql.DB, ids []int64, stationCall, operator, grid string) error {
	if len(ids) == 0 {
		return nil
	}

	sets := []string{}
	args := []any{}
	add := func(col, val string) {
		if val == "" {
			return
		}
		sets = append(sets, col+"=?")
		args = append(args, val)
	}
	add("station_callsign", stationCall)
	add("operator", operator)
	add("my_gridsquare", grid)
	if len(sets) == 0 {
		return nil
	}
	args = append(args, time.Now().UTC().Format(time.RFC3339))

	// Build WHERE id IN (?,?,...) clause.
	placeholders := make([]string, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := `UPDATE qsos SET ` + strings.Join(sets, ", ") + `, updated_at=? WHERE id IN (` +
		strings.Join(placeholders, ",") + `)`

	_, err := db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("normalize station fields: %w", err)
	}
	return nil
}
