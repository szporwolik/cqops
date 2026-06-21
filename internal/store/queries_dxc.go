package store

import (
	"database/sql"
	"fmt"
	"time"
)

// DXCSpot holds a single DX Cluster spot from the database.
type DXCSpot struct {
	ID         int64
	DXCall     string
	Frequency  float64
	Band       string
	Mode       string
	ModeCat    string
	Comment    string
	Spotter    string
	DXCont     string // DX call continent (from CTY.DAT)
	DXCC       string // DX call country/entity name (from CTY.DAT)
	SpotCont   string // Spotter continent (from CTY.DAT)
	ReceivedAt int64
}

// InsertDXCSpots bulk-inserts DX Cluster spots, replacing duplicates
// for the same DX call (INSERT OR REPLACE).
func InsertDXCSpots(db *sql.DB, spots []DXCSpot) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO dxc_spots
		(dx_call, frequency, band, mode, mode_cat, comment, spotter, dx_cont, dxcc, spot_cont, received_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for _, s := range spots {
		res, err := stmt.Exec(s.DXCall, s.Frequency, s.Band, s.Mode, s.ModeCat, s.Comment, s.Spotter, s.DXCont, s.DXCC, s.SpotCont, s.ReceivedAt)
		if err != nil {
			return inserted, fmt.Errorf("insert dxc_spot: %w", err)
		}
		n, _ := res.RowsAffected()
		if n > 0 {
			inserted++
		}
	}
	if err := tx.Commit(); err != nil {
		return inserted, fmt.Errorf("commit: %w", err)
	}
	return inserted, nil
}

// PurgeOldDXCSpots removes DXC spots older than 120 minutes.
func PurgeOldDXCSpots(db *sql.DB) error {
	cutoff := time.Now().UTC().Add(-120 * time.Minute).Unix()
	_, err := db.Exec(`DELETE FROM dxc_spots WHERE received_at < ?`, cutoff)
	if err != nil {
		return fmt.Errorf("purge dxc_spots: %w", err)
	}
	return nil
}

// QueryDXCSpots returns recent DXC spots ordered by time (newest first).
func QueryDXCSpots(db *sql.DB) ([]DXCSpot, error) {
	rows, err := db.Query(`SELECT id, dx_call, frequency, band, mode, mode_cat, comment, spotter, dx_cont, dxcc, spot_cont, received_at
		FROM dxc_spots ORDER BY received_at DESC LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("query dxc_spots: %w", err)
	}
	defer rows.Close()
	var spots []DXCSpot
	for rows.Next() {
		var s DXCSpot
		if err := rows.Scan(&s.ID, &s.DXCall, &s.Frequency, &s.Band, &s.Mode, &s.ModeCat, &s.Comment, &s.Spotter, &s.DXCont, &s.DXCC, &s.SpotCont, &s.ReceivedAt); err != nil {
			return spots, fmt.Errorf("scan dxc_spot: %w", err)
		}
		spots = append(spots, s)
	}
	return spots, rows.Err()
}

// QueryDXCSpotByCall returns the most recent DXC spot for a given callsign, if any.
func QueryDXCSpotByCall(db *sql.DB, call string) (*DXCSpot, error) {
	var s DXCSpot
	err := db.QueryRow(`SELECT id, dx_call, frequency, band, mode, mode_cat, comment, spotter, dx_cont, dxcc, spot_cont, received_at
		FROM dxc_spots WHERE dx_call = ? ORDER BY received_at DESC LIMIT 1`, call).Scan(
		&s.ID, &s.DXCall, &s.Frequency, &s.Band, &s.Mode, &s.ModeCat, &s.Comment, &s.Spotter, &s.DXCont, &s.DXCC, &s.SpotCont, &s.ReceivedAt)
	if err != nil {
		return nil, err // sql.ErrNoRows if not found
	}
	return &s, nil
}
