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
