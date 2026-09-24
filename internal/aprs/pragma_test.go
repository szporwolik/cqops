package aprs

import (
	"path/filepath"
	"testing"
)

// The cache is written once per received packet, so it relies on WAL and a
// relaxed sync mode to avoid an fsync per packet. A DSN typo would silently
// fall back to the rollback journal, so assert the pragmas took effect.
func TestOpenCacheDB_UsesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pragma.db")
	c, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("OpenCacheDB: %v", err)
	}
	defer c.Close()

	var journal string
	if err := c.db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want wal", journal)
	}

	var sync int
	if err := c.db.QueryRow("PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatalf("read synchronous: %v", err)
	}
	if sync != 1 { // 1 = NORMAL
		t.Errorf("synchronous = %d, want 1 (NORMAL)", sync)
	}

	var busy int
	if err := c.db.QueryRow("PRAGMA busy_timeout").Scan(&busy); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if busy != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busy)
	}
}
