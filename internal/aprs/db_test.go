package aprs

import (
	"path/filepath"
	"testing"
	"time"
)

func TestCacheDB_OpenClose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	n, err := db.StationCount()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 stations, got %d", n)
	}

	if err := db.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
}

func TestCacheDB_UpsertAndQuery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	s1 := StationRecord{
		Callsign: "N0CALL", Lat: 50.0, Lon: 20.0,
		Symbol: "/r", Comment: "test1",
		Course: 90, SpeedKmH: 50,
		LastHeard: now,
		RawPacket: "raw1",
	}
	s2 := StationRecord{
		Callsign: "SP9MOA", Lat: 49.5, Lon: 19.3,
		Symbol: "/[", Comment: "test2",
		Course: 180, SpeedKmH: 30,
		LastHeard: now.Add(-10 * time.Minute),
		RawPacket: "raw2",
	}

	if err := db.UpsertStation(s1); err != nil {
		t.Fatalf("upsert1: %v", err)
	}
	if err := db.UpsertStation(s2); err != nil {
		t.Fatalf("upsert2: %v", err)
	}

	n, err := db.StationCount()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 stations, got %d", n)
	}

	// Upsert again — should update, not duplicate.
	s1.Comment = "updated"
	s1.Course = 270
	if err := db.UpsertStation(s1); err != nil {
		t.Fatalf("upsert3: %v", err)
	}

	n, err = db.StationCount()
	if err != nil {
		t.Fatalf("count2: %v", err)
	}
	if n != 2 {
		t.Errorf("upsert should not duplicate, got %d", n)
	}

	// Query recent.
	recent, err := db.RecentStations(10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 2 {
		t.Errorf("expected 2 recent, got %d", len(recent))
	}

	// Most recent should be first (N0CALL updated).
	if recent[0].Callsign != "N0CALL" {
		t.Errorf("most recent should be N0CALL, got %s", recent[0].Callsign)
	}
	if recent[0].Comment != "updated" {
		t.Errorf("upsert should update comment, got %s", recent[0].Comment)
	}
}

func TestCacheDB_Prune(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	old := now.Add(-2 * time.Hour)
	fresh := now.Add(-5 * time.Minute)

	s1 := StationRecord{Callsign: "OLD1", Lat: 50, Lon: 20, LastHeard: old}
	s2 := StationRecord{Callsign: "FRESH", Lat: 51, Lon: 21, LastHeard: fresh}
	db.UpsertStation(s1)
	db.UpsertStation(s2)

	cutoff := now.Add(-1 * time.Hour)
	removed, err := db.PruneOlderThan(cutoff)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	n, _ := db.StationCount()
	if n != 1 {
		t.Errorf("expected 1 remaining, got %d", n)
	}
}
