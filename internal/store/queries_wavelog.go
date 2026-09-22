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

// SetWavelogIDChecked stores the remote id while checking whether the row
// was edited since the snapshot that was uploaded (uploadedRev). The id is
// persisted either way — the remote copy exists — but when the current
// revision is NEWER, the row is durably marked dirty so the newer edit is
// pushed by a follow-up PATCH and can no longer be overwritten by a remote
// refresh. Returns changed=true in that case. uploadedRev < 0 disables the
// check entirely (legacy callers) — captured revisions are always >= 0, so
// a legitimate revision of zero is still validated.
func SetWavelogIDChecked(db *sql.DB, id, remoteID, uploadedRev int64) (bool, error) {
	var err error
	if uploadedRev >= 0 {
		_, err = db.Exec(`UPDATE qsos SET wavelog_id=?, wavelog_dirty=CASE WHEN wavelog_dirty_rev > ? THEN 1 ELSE wavelog_dirty END WHERE id=?`,
			remoteID, uploadedRev, id)
	} else {
		_, err = db.Exec(`UPDATE qsos SET wavelog_id=? WHERE id=?`, remoteID, id)
	}
	if err != nil {
		return false, fmt.Errorf("set wavelog id checked: %w", err)
	}
	if uploadedRev < 0 {
		return false, nil
	}
	var rev int64
	if err := db.QueryRow(`SELECT wavelog_dirty_rev FROM qsos WHERE id=?`, id).Scan(&rev); err != nil {
		return false, fmt.Errorf("read revision after id attach: %w", err)
	}
	return rev > uploadedRev, nil
}

// SetWavelogDirty marks (or clears) the durable pending-sync flag on a local
// QSO. It is set when the local row diverges from the Wavelog copy (PATCH
// deferred in offline mode or failed) and cleared when the remote copy has
// been brought in sync. A dirty row must never be overwritten by a remote
// refresh, because the local copy holds changes the server does not have.
func SetWavelogDirty(db *sql.DB, id int64, dirty bool) error {
	v := 0
	if dirty {
		v = 1
	}
	_, err := db.Exec(`UPDATE qsos SET wavelog_dirty=? WHERE id=?`, v, id)
	if err != nil {
		return fmt.Errorf("set wavelog dirty: %w", err)
	}
	return nil
}

// QSOHasPendingSync reports whether the QSO has local changes that were not
// yet pushed to Wavelog.
func QSOHasPendingSync(db *sql.DB, id int64) (bool, error) {
	var dirty int
	if err := db.QueryRow(`SELECT wavelog_dirty FROM qsos WHERE id=?`, id).Scan(&dirty); err != nil {
		return false, fmt.Errorf("qso pending sync: %w", err)
	}
	return dirty != 0, nil
}

// ClearWavelogDirtyIfRevision clears the pending-sync flag only when the
// row's pending-sync revision still equals rev — the revision of the exact
// edit the PATCH pushed. An acknowledgement for an older revision must not
// clear a newer pending edit; the newer edit's own acknowledgement will.
func ClearWavelogDirtyIfRevision(db *sql.DB, id, rev int64) error {
	_, err := db.Exec(`UPDATE qsos SET wavelog_dirty=0 WHERE id=? AND wavelog_dirty_rev=?`, id, rev)
	if err != nil {
		return fmt.Errorf("clear wavelog dirty: %w", err)
	}
	return nil
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
