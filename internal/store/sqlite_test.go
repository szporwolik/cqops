package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

// TestOpenAppliesPragmasToEveryConnection verifies WAL, busy timeout and
// foreign-key enforcement are actually in effect — and on EVERY connection,
// not just the first one the pool opens. All four pool connections are held
// open simultaneously while each is probed, so a per-connection pragma that
// only ran on the first connection would be caught here.
func TestOpenAppliesPragmasToEveryConnection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	const conns = 4 // db.SetMaxOpenConns(4)
	held := make([]*sql.Conn, 0, conns)
	for i := 0; i < conns; i++ {
		conn, err := db.Conn(context.Background())
		if err != nil {
			t.Fatalf("acquire connection %d: %v", i, err)
		}
		held = append(held, conn) // keep it open while probing the next one

		var mode string
		if err := conn.QueryRowContext(context.Background(), `PRAGMA journal_mode`).Scan(&mode); err != nil {
			t.Fatalf("conn %d journal_mode: %v", i, err)
		}
		if mode != "wal" {
			t.Errorf("conn %d journal_mode = %q, want wal", i, mode)
		}

		var timeout int
		if err := conn.QueryRowContext(context.Background(), `PRAGMA busy_timeout`).Scan(&timeout); err != nil {
			t.Fatalf("conn %d busy_timeout: %v", i, err)
		}
		if timeout < 5000 {
			t.Errorf("conn %d busy_timeout = %d, want >= 5000", i, timeout)
		}

		var fk int
		if err := conn.QueryRowContext(context.Background(), `PRAGMA foreign_keys`).Scan(&fk); err != nil {
			t.Fatalf("conn %d foreign_keys: %v", i, err)
		}
		if fk != 1 {
			t.Errorf("conn %d foreign_keys = %d, want 1", i, fk)
		}
	}
	for _, c := range held {
		c.Close()
	}
}
