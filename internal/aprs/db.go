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
func (c *CacheDB) UpsertStation(s StationRecord) error {
	lastHeardStr := s.LastHeard.UTC().Format(time.RFC3339)
	_, err := c.db.Exec(`
		INSERT INTO aprs_stations (callsign, lat, lon, symbol, comment, course, speed_kmh, altitude_m, last_heard, raw_packet)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(callsign) DO UPDATE SET
			lat=excluded.lat, lon=excluded.lon, symbol=excluded.symbol,
			comment=excluded.comment, course=excluded.course, speed_kmh=excluded.speed_kmh,
			altitude_m=excluded.altitude_m, last_heard=excluded.last_heard,
			raw_packet=excluded.raw_packet
	`, s.Callsign, s.Lat, s.Lon, s.Symbol, s.Comment, s.Course, s.SpeedKmH, s.AltitudeM, lastHeardStr, s.RawPacket)
	if err != nil {
		return fmt.Errorf("aprs upsert: %w", err)
	}
	return nil
}

// StationCount returns the number of cached stations heard.
func (c *CacheDB) StationCount() (int, error) {
	var n int
	err := c.db.QueryRow("SELECT COALESCE(COUNT(*),0) FROM aprs_stations").Scan(&n)
	return n, err
}

// RecentStations returns the N most recently heard stations.
func (c *CacheDB) RecentStations(limit int) ([]StationRecord, error) {
	rows, err := c.db.Query(`
		SELECT callsign, lat, lon, COALESCE(symbol,''), COALESCE(comment,''), COALESCE(course,0), COALESCE(speed_kmh,0), COALESCE(altitude_m,0), last_heard, COALESCE(raw_packet,'')
		FROM aprs_stations ORDER BY last_heard DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []StationRecord
	for rows.Next() {
		var s StationRecord
		var lastHeardStr string
		if err := rows.Scan(&s.Callsign, &s.Lat, &s.Lon, &s.Symbol, &s.Comment, &s.Course, &s.SpeedKmH, &s.AltitudeM, &lastHeardStr, &s.RawPacket); err != nil {
			return result, err
		}
		s.LastHeard, _ = time.Parse(time.RFC3339, lastHeardStr)
		result = append(result, s)
	}
	return result, rows.Err()
}

// PruneOlderThan removes stations not heard since the given time.
func (c *CacheDB) PruneOlderThan(cutoff time.Time) (int64, error) {
	res, err := c.db.Exec("DELETE FROM aprs_stations WHERE last_heard < ?", cutoff.UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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
			raw_packet TEXT DEFAULT ''
		)
	`)
	if err != nil {
		return err
	}
	// Index for time-based pruning.
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_aprs_last_heard ON aprs_stations(last_heard)")
	return err
}
