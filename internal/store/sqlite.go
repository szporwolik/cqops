package store

import (
	"database/sql"
	"fmt"
)

func Open(path string) (*sql.DB, error) {
	// WAL mode for concurrent reads + writes. 5s busy timeout so SQLite
	// retries internally instead of returning SQLITE_BUSY immediately.
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
