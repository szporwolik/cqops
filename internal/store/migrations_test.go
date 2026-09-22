package store

import (
	"database/sql"
	"testing"
)

// TestMigrationsApplyCleanly verifies all migrations run without error
// and the resulting schema has the expected tables and columns.
func TestMigrationsApplyCleanly(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	// Verify key tables exist.
	tables := []string{"qsos", "dxc_spots", "psk_spots"}
	for _, name := range tables {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", name).Scan(&count); err != nil {
			t.Errorf("check table %s: %v", name, err)
		} else if count == 0 {
			t.Errorf("table %s not created", name)
		}
	}

	// Verify qsos has base_call column (added by migration).
	rows, err := db.Query("PRAGMA table_info(qsos)")
	if err != nil {
		t.Fatalf("pragma table_info: %v", err)
	}
	defer rows.Close()
	foundBaseCall := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		if name == "base_call" {
			foundBaseCall = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows iteration: %v", err)
	}
	if !foundBaseCall {
		t.Error("base_call column not found in qsos table — migration may have been skipped")
	}

	// Verify idx_qsos_base_call index exists.
	var idxCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_qsos_base_call'").Scan(&idxCount); err != nil {
		t.Errorf("check index: %v", err)
	} else if idxCount == 0 {
		t.Error("idx_qsos_base_call index not created")
	}
}

// TestMigrationsIdempotent verifies Migrate is safe to call multiple times.
func TestMigrationsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("third Migrate: %v", err)
	}
}

// TestMigrateUpgradesV11Database simulates a released v0.11.0 database
// (user_version=2, wavelog_uploaded column and its index present, no
// wavelog_id). Migrate must succeed, add wavelog_id and drop the legacy
// column and index.
func TestMigrateUpgradesV11Database(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	stmts := []string{
		`CREATE TABLE qsos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			call TEXT NOT NULL,
			qso_date TEXT, time_on TEXT, time_off TEXT, band TEXT, freq REAL,
			mode TEXT, submode TEXT, gridsquare TEXT, name TEXT, country TEXT,
			wavelog_uploaded TEXT DEFAULT '',
			dxcc TEXT DEFAULT '', base_call TEXT DEFAULT '',
			source TEXT NOT NULL DEFAULT 'manual',
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX idx_qsos_wavelog_uploaded ON qsos(wavelog_uploaded)`,
		`PRAGMA user_version = 2`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("setup v0.11 db: %v", err)
		}
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate on v0.11 db: %v", err)
	}

	// Legacy column and index must be gone.
	for _, probe := range []struct{ sql, what string }{
		{`SELECT wavelog_uploaded FROM qsos LIMIT 1`, "legacy column"},
		{`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_qsos_wavelog_uploaded'`, "legacy index"},
	} {
		if probe.what == "legacy index" {
			var n int
			if err := db.QueryRow(probe.sql).Scan(&n); err != nil || n != 0 {
				t.Errorf("legacy index still present (n=%d err=%v)", n, err)
			}
			continue
		}
		if _, err := db.Query(probe.sql); err == nil {
			t.Error("legacy column wavelog_uploaded still present")
		}
	}

	// New column and index must exist.
	if _, err := db.Exec(`INSERT INTO qsos (call, wavelog_id, wavelog_dirty, created_at, updated_at) VALUES ('SP9MOA', 42, 1, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("insert with wavelog_id/wavelog_dirty: %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_qsos_wavelog_id'`).Scan(&n); err != nil || n != 1 {
		t.Errorf("idx_qsos_wavelog_id missing (n=%d err=%v)", n, err)
	}
}

// TestMigrateUpgradesV10Database simulates a v0.10.x database (user_version=1)
// whose qsos table has neither wavelog_id nor dxcc. Migrate must succeed.
func TestMigrateUpgradesV10Database(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	defer db.Close()

	// Minimal old qsos table: only the columns existing in v0.10.
	if _, err := db.Exec(`CREATE TABLE qsos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		call TEXT NOT NULL,
		qso_date TEXT, time_on TEXT, band TEXT, mode TEXT, gridsquare TEXT,
		source TEXT NOT NULL DEFAULT 'manual',
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("setup v0.10 db: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 1`); err != nil {
		t.Fatalf("set user_version: %v", err)
	}

	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate on v0.10 db: %v", err)
	}

	// dxcc, wavelog_id and wavelog_dirty must now exist.
	for _, col := range []string{"dxcc", "wavelog_id", "wavelog_dirty"} {
		if _, err := db.Exec(`INSERT INTO qsos (call, ` + col + `, created_at, updated_at) VALUES ('SP9MOA', 'x', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err != nil {
			t.Errorf("column %s missing after upgrade: %v", col, err)
		}
	}

	// Migrate must be idempotent.
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
}
