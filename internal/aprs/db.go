package aprs

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	_ "modernc.org/sqlite"
)

// StationRecord holds a decoded APRS position report stored in the cache.
type StationRecord struct {
	Callsign  string
	Lat       float64
	Lon       float64
	Symbol    string
	Comment   string
	Course    int
	SpeedKmH  int
	AltitudeM int
	LastHeard time.Time
	RawPacket string
	Source    string // "aprs_is" or "kiss"
}

// TrailPoint is a single historic position for a station's movement trail.
type TrailPoint struct {
	Lat       float64
	Lon       float64
	LastHeard time.Time
}

// CacheDB wraps a SQLite database for caching received APRS stations.
type CacheDB struct {
	db *sql.DB
}

// OpenCacheDB opens (or creates) the APRS cache database at the given path.
// Creates the parent directory if it doesn't exist (e.g. cache folder was cleared).
func OpenCacheDB(path string) (*CacheDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("aprs cache dir: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("aprs cache open: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite works best single-writer.
	db.SetConnMaxLifetime(0)

	if err := migrateCache(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("aprs cache migrate: %w", err)
	}

	applog.Debug("APRS: cache database opened", "path", path)
	return &CacheDB{db: db}, nil
}

// Close closes the cache database.
func (c *CacheDB) Close() error {
	applog.Debug("APRS: cache database closing")
	return c.db.Close()
}

// UpsertStation inserts or updates a station record in the cache.
// Before updating an existing station, the old position is saved to the
// history table if the station has moved significantly (>~50 m).
func (c *CacheDB) UpsertStation(s StationRecord) error {
	// Save old position to history before overwriting.
	c.saveHistoryBeforeUpsert(s)

	lastHeardStr := s.LastHeard.UTC().Format(time.RFC3339)
	src := s.Source
	if src == "" {
		src = "aprs_is"
	}
	_, err := c.db.Exec(`
		INSERT INTO aprs_stations (callsign, lat, lon, symbol, comment, course, speed_kmh, altitude_m, last_heard, raw_packet, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(callsign) DO UPDATE SET
			lat=excluded.lat, lon=excluded.lon, symbol=excluded.symbol,
			comment=excluded.comment, course=excluded.course, speed_kmh=excluded.speed_kmh,
			altitude_m=excluded.altitude_m, last_heard=excluded.last_heard,
			raw_packet=excluded.raw_packet,
			source=excluded.source
	`, s.Callsign, s.Lat, s.Lon, s.Symbol, s.Comment, s.Course, s.SpeedKmH, s.AltitudeM, lastHeardStr, s.RawPacket, src)
	if err != nil {
		return fmt.Errorf("aprs upsert: %w", err)
	}
	return nil
}

// minTrailDelta is the minimum distance (in degrees, ~50 m) between
// positions before we record a trail point. Avoids jitter trails.
const minTrailDelta = 0.0005

// saveHistoryBeforeUpsert reads the current position for the callsign
// and, if it differs from the new position by a meaningful amount,
// inserts the old position into aprs_position_history.
func (c *CacheDB) saveHistoryBeforeUpsert(s StationRecord) {
	var oldLat, oldLon float64
	err := c.db.QueryRow(
		"SELECT lat, lon FROM aprs_stations WHERE callsign=?", s.Callsign,
	).Scan(&oldLat, &oldLon)
	if err != nil {
		return // new station — no old position to save
	}
	// Skip if position hasn't changed meaningfully.
	dLat := oldLat - s.Lat
	dLon := oldLon - s.Lon
	if dLat < 0 {
		dLat = -dLat
	}
	if dLon < 0 {
		dLon = -dLon
	}
	if dLat < minTrailDelta && dLon < minTrailDelta {
		return
	}
	now := s.LastHeard.UTC().Format(time.RFC3339)
	var execErr error
	_, execErr = c.db.Exec(
		"INSERT OR IGNORE INTO aprs_position_history (callsign, lat, lon, last_heard) VALUES (?, ?, ?, ?)",
		s.Callsign, oldLat, oldLon, now,
	)
	if execErr == nil {
		applog.Debug("APRS: trail point saved", "callsign", s.Callsign, "oldLat", fmt.Sprintf("%.5f", oldLat), "oldLon", fmt.Sprintf("%.5f", oldLon))
	} else {
		applog.Debug("APRS: trail point skipped (dup key)", "callsign", s.Callsign, "error", execErr)
	}
}

// StationCount returns the number of cached stations heard.
func (c *CacheDB) StationCount() (int, error) {
	var n int
	err := c.db.QueryRow("SELECT COALESCE(COUNT(*),0) FROM aprs_stations").Scan(&n)
	return n, err
}

// RecentStations returns the N most recently heard stations, optionally
// filtered by source. Pass empty source to return all.
func (c *CacheDB) RecentStations(limit int, source ...string) ([]StationRecord, error) {
	src := ""
	if len(source) > 0 {
		src = source[0]
	}
	var rows *sql.Rows
	var err error
	if src != "" {
		rows, err = c.db.Query(`
			SELECT callsign, lat, lon, COALESCE(symbol,''), COALESCE(comment,''), COALESCE(course,0), COALESCE(speed_kmh,0), COALESCE(altitude_m,0), last_heard, COALESCE(raw_packet,''), COALESCE(source,'')
			FROM aprs_stations WHERE source=? ORDER BY last_heard DESC LIMIT ?
		`, src, limit)
	} else {
		rows, err = c.db.Query(`
			SELECT callsign, lat, lon, COALESCE(symbol,''), COALESCE(comment,''), COALESCE(course,0), COALESCE(speed_kmh,0), COALESCE(altitude_m,0), last_heard, COALESCE(raw_packet,''), COALESCE(source,'')
			FROM aprs_stations ORDER BY last_heard DESC LIMIT ?
		`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StationRecord
	for rows.Next() {
		var s StationRecord
		var lastHeardStr string
		if err := rows.Scan(&s.Callsign, &s.Lat, &s.Lon, &s.Symbol, &s.Comment, &s.Course, &s.SpeedKmH, &s.AltitudeM, &lastHeardStr, &s.RawPacket, &s.Source); err != nil {
			return result, err
		}
		s.LastHeard, _ = time.Parse(time.RFC3339, lastHeardStr)
		result = append(result, s)
	}
	return result, rows.Err()
}

// PruneOlderThan removes stations and trail history not heard since the
// given cutoff time.
func (c *CacheDB) PruneOlderThan(cutoff time.Time) (int64, error) {
	cs := cutoff.UTC().Format(time.RFC3339)
	// Prune history first (station delete cascades, but explicit is safer).
	c.db.Exec("DELETE FROM aprs_position_history WHERE last_heard < ?", cs)
	res, err := c.db.Exec("DELETE FROM aprs_stations WHERE last_heard < ?", cs)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// StationTrail returns the last N historic positions for a station,
// oldest first. Current position is NOT included.
func (c *CacheDB) StationTrail(callsign string, limit int) ([]TrailPoint, error) {
	rows, err := c.db.Query(`
		SELECT lat, lon, last_heard FROM aprs_position_history
		WHERE callsign=? ORDER BY last_heard DESC LIMIT ?
	`, callsign, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Collect and reverse to get oldest first.
	var reversed []TrailPoint
	for rows.Next() {
		var p TrailPoint
		var ts string
		if err := rows.Scan(&p.Lat, &p.Lon, &ts); err != nil {
			continue
		}
		p.LastHeard, _ = time.Parse(time.RFC3339, ts)
		reversed = append(reversed, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Reverse to chronological order.
	result := make([]TrailPoint, len(reversed))
	for i, p := range reversed {
		result[len(reversed)-1-i] = p
	}
	return result, nil
}

// StationTrails returns trail positions for multiple callsigns at once.
// Returns a map from callsign to trail (oldest first, max 5 per station).
func (c *CacheDB) StationTrails(callsigns []string) (map[string][]TrailPoint, error) {
	if len(callsigns) == 0 {
		return nil, nil
	}
	result := make(map[string][]TrailPoint)
	for _, cs := range callsigns {
		trail, err := c.StationTrail(cs, 5)
		if err != nil {
			continue
		}
		if len(trail) > 0 {
			result[cs] = trail
		}
	}
	return result, nil
}

func migrateCache(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS aprs_stations (
			callsign   TEXT PRIMARY KEY,
			lat        REAL NOT NULL,
			lon        REAL NOT NULL,
			symbol     TEXT DEFAULT '',
			comment    TEXT DEFAULT '',
			course     INTEGER DEFAULT 0,
			speed_kmh  INTEGER DEFAULT 0,
			altitude_m INTEGER DEFAULT 0,
			last_heard TEXT NOT NULL,
			raw_packet TEXT DEFAULT '',
			source     TEXT NOT NULL DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	// Migration: add source column if upgrading from older schema.
	db.Exec("ALTER TABLE aprs_stations ADD COLUMN source TEXT NOT NULL DEFAULT ''")
	// Index for time-based pruning.
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_aprs_last_heard ON aprs_stations(last_heard)")
	if err != nil {
		return err
	}
	// Position history table — stores up to N previous positions per station.
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS aprs_position_history (
			callsign   TEXT NOT NULL,
			lat        REAL NOT NULL,
			lon        REAL NOT NULL,
			last_heard TEXT NOT NULL,
			PRIMARY KEY (callsign, last_heard)
		)
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_aph_callsign ON aprs_position_history(callsign, last_heard DESC)")
	return err
}
