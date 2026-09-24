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
// without returning SQLITE_BUSY to the caller. All settings are passed as
// modernc `_pragma` parameters, which the driver applies to every
// connection — the legacy mattn-style DSN names are silently ignored.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path+
		"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"+
		"&_pragma=synchronous(NORMAL)&_pragma=cache_size(-8000)")
	if err != nil {
		return nil, fmt.Errorf("ref: open db at %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ref: ping db at %s: %w", path, err)
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
			search   TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (ref_type, ref, name)
		);
		CREATE INDEX IF NOT EXISTS idx_refs_lookup ON refs(ref_type, ref);
		CREATE VIRTUAL TABLE IF NOT EXISTS refs_fts USING fts5(
			search, grid,
			content='refs', content_rowid='rowid', tokenize='trigram');
		CREATE TABLE IF NOT EXISTS refs_meta (key TEXT PRIMARY KEY, value TEXT);
	`)
	if err != nil {
		return err
	}
	// Migration: add is_group column for databases created before this field existed.
	rdb.db.Exec(`ALTER TABLE refs ADD COLUMN is_group INTEGER NOT NULL DEFAULT 0`)
	// Migration: add search column for diacritic/case-insensitive search.
	rdb.db.Exec(`ALTER TABLE refs ADD COLUMN search TEXT NOT NULL DEFAULT ''`)

	// Populate the FTS index once for databases that already contain rows
	// (upgraded installations) — fresh databases rebuild it during Rebuild().
	var marker string
	rdb.db.QueryRow(`SELECT value FROM refs_meta WHERE key = 'fts_built'`).Scan(&marker)
	if marker != "1" {
		if err := rdb.rebuildFTS(); err != nil {
			return fmt.Errorf("ref: initial fts rebuild: %w", err)
		}
		rdb.db.Exec(`INSERT OR REPLACE INTO refs_meta(key, value) VALUES('fts_built','1')`)
	}
	return nil
}

// rebuildFTS resynchronizes the trigram index from the content table. Runs
// after every Rebuild() (inside its transaction) and once at migration for
// databases that predate the index.
func (rdb *DB) rebuildFTS() error {
	_, err := rdb.db.Exec(`INSERT INTO refs_fts(refs_fts) VALUES('rebuild')`)
	return err
}
