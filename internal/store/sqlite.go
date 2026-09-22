package store

import (
	"database/sql"
	"fmt"
)

func Open(path string) (*sql.DB, error) {
	// modernc.org/sqlite applies `_pragma` parameters to EVERY connection:
	// WAL journal mode (concurrent readers + one writer), a 5s busy timeout
	// so SQLite retries internally instead of returning SQLITE_BUSY, and
	// foreign-key enforcement.
	// The legacy mattn-style names (_journal_mode, _busy_timeout,
	// _foreign_keys) are silently ignored by this driver.
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite allows a single writer. An unbounded pool lets concurrent TUI
	// reads, WSJT-X logging and Wavelog sync each open a connection with its
	// own page cache, which is wasteful on Pi-class hardware.
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}
