// Package ref provides a fast SQLite-backed lookup database for amateur radio
// reference programmes: SOTA, POTA, WWFF, and IOTA. The database is stored
// separately from the QSO logbook and is rebuilt from cached data files.
//
// Design goals:
//   - Sub-millisecond lookups via SQLite B-tree index
//   - Streaming parsers — never loads entire data files into memory
//   - Atomic rebuilds — transaction + prepared statements, all-or-nothing
//   - Offline-first — cached files preferred, downloads only when stale
//   - Potato-PC ready — minimal allocations, no goroutine leaks
package ref

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// RefType identifies a reference programme.
type RefType string

const (
	RefSOTA RefType = "SOTA"
	RefPOTA RefType = "POTA"
	RefWWFF RefType = "WWFF"
	RefIOTA RefType = "IOTA"
)

// Row is one reference row as stored in the database.
type Row struct {
	RefType RefType
	Ref     string
	Name    string
	Grid    string
	Height  int // metres, SOTA only; 0 for other programmes
}

// DB wraps a read-optimised SQLite database for reference lookups.
// The underlying connection is safe for concurrent reads; writes happen
// only during the controlled rebuild phase (single writer).
type DB struct {
	db *sql.DB
}

// Open opens or creates the reference database at path. The database uses
// WAL journal mode and a 5-second busy timeout so transient locks resolve
// without returning SQLITE_BUSY to the caller.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("ref: open db at %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ref: ping db at %s: %w", path, err)
	}
	// Performance tuning for bulk inserts during rebuild.
	if _, err := db.Exec(`PRAGMA synchronous=NORMAL; PRAGMA cache_size=-8000`); err != nil {
		db.Close()
		return nil, fmt.Errorf("ref: pragma: %w", err)
	}
	rdb := &DB{db: db}
	if err := rdb.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ref: migrate: %w", err)
	}
	return rdb, nil
}

// UnderlyingDB returns the raw *sql.DB for use by test helpers.
func (rdb *DB) UnderlyingDB() *sql.DB { return rdb.db }

// Close closes the reference database. Safe to call multiple times.
func (rdb *DB) Close() error {
	if rdb.db == nil {
		return nil
	}
	return rdb.db.Close()
}

// migrate creates the schema if it does not already exist.
func (rdb *DB) migrate() error {
	_, err := rdb.db.Exec(`
		CREATE TABLE IF NOT EXISTS refs (
			ref_type TEXT NOT NULL,
			ref      TEXT NOT NULL,
			name     TEXT NOT NULL,
			grid     TEXT NOT NULL DEFAULT '',
			height   INTEGER NOT NULL DEFAULT 0,
			is_group INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (ref_type, ref, name)
		);
		CREATE INDEX IF NOT EXISTS idx_refs_lookup ON refs(ref_type, ref);
	`)
	if err != nil {
		return err
	}
	// Migration: add is_group column for databases created before this field existed.
	rdb.db.Exec(`ALTER TABLE refs ADD COLUMN is_group INTEGER NOT NULL DEFAULT 0`)
	return nil
}
