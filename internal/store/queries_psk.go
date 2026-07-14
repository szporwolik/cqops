package store

import (
	"database/sql"
	"fmt"
	"time"
)

// PSKSpot holds a single PSK Reporter reception record from the database.
type PSKSpot struct {
	ID           int64
	ReceiverCall string
	ReceiverLoc  string
	Frequency    float64
	SNR          int
	Mode         string
	FlowStart    int64
	FetchTime    int64
	StationCall  string
}

// InsertPSKSpots bulk-inserts PSK Reporter spots, skipping duplicates.
// Returns the number of newly inserted rows.
func InsertPSKSpots(db *sql.DB, spots []PSKSpot) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO psk_spots
		(receiver_call, receiver_loc, frequency, snr, mode, flow_start, fetch_time, station_call)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range spots {
		res, err := stmt.Exec(s.ReceiverCall, s.ReceiverLoc, s.Frequency, s.SNR, s.Mode,
			s.FlowStart, s.FetchTime, s.StationCall)
		if err != nil {
			return inserted, fmt.Errorf("insert psk_spot: %w", err)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// QueryPSKSpots returns PSK Reporter spots for the given station callsign
// within the specified time window (flow_start >= since).
func QueryPSKSpots(db *sql.DB, stationCall string, since int64) ([]PSKSpot, error) {
	rows, err := db.Query(`SELECT id, receiver_call, receiver_loc, frequency, snr, mode,
		flow_start, fetch_time, station_call
		FROM psk_spots WHERE station_call=? AND flow_start >= ?
		ORDER BY flow_start DESC LIMIT 500`, stationCall, since)
	if err != nil {
		return nil, fmt.Errorf("query psk_spots: %w", err)
	}
	defer rows.Close()

	var spots = make([]PSKSpot, 0, 500)
	for rows.Next() {
		var s PSKSpot
		if err := rows.Scan(&s.ID, &s.ReceiverCall, &s.ReceiverLoc, &s.Frequency,
			&s.SNR, &s.Mode, &s.FlowStart, &s.FetchTime, &s.StationCall); err != nil {
			return spots, fmt.Errorf("scan psk_spot: %w", err)
		}
		spots = append(spots, s)
	}
	return spots, rows.Err()
}

// PurgeOldPSKSpots removes spots older than 7 days.
func PurgeOldPSKSpots(db *sql.DB) error {
	cutoff := time.Now().UTC().Add(-7 * 24 * time.Hour).Unix()
	_, err := db.Exec(`DELETE FROM psk_spots WHERE flow_start < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("purge psk_spots: %w", err)
	}
	return nil
}
