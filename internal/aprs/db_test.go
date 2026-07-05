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

func TestCacheDB_TrailHistory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	// Insert initial position.
	s1 := StationRecord{Callsign: "MOBILE", Lat: 50.0, Lon: 20.0, LastHeard: now}
	db.UpsertStation(s1)

	// First update — should NOT create trail (same position).
	s1.Lat = 50.0
	s1.Lon = 20.0
	s1.LastHeard = now.Add(1 * time.Minute)
	db.UpsertStation(s1)

	// Second update — moved significantly, should create trail point.
	s1.Lat = 50.1
	s1.Lon = 20.1
	s1.LastHeard = now.Add(2 * time.Minute)
	db.UpsertStation(s1)

	// Third update — moved again, should create second trail point.
	s1.Lat = 50.2
	s1.Lon = 20.2
	s1.LastHeard = now.Add(3 * time.Minute)
	db.UpsertStation(s1)

	// Fourth update — moved again, should create third trail point.
	s1.Lat = 50.3
	s1.Lon = 20.3
	s1.LastHeard = now.Add(4 * time.Minute)
	db.UpsertStation(s1)

	trail, err := db.StationTrail("MOBILE", 5)
	if err != nil {
		t.Fatalf("trail: %v", err)
	}
	// Should have 3 trail points (50.0/20.0, then 50.1/20.1, then 50.2/20.2).
	// The skip on first upsert (no old position) + the same-position skip = 3 trail points from 4 updates.
	if len(trail) != 3 {
		t.Errorf("expected 3 trail points, got %d", len(trail))
	}
	if len(trail) > 0 && (trail[0].Lat != 50.0 || trail[0].Lon != 20.0) {
		t.Errorf("first trail point should be 50.0/20.0, got %.1f/%.1f", trail[0].Lat, trail[0].Lon)
	}
	if len(trail) > 2 && (trail[2].Lat != 50.2 || trail[2].Lon != 20.2) {
		t.Errorf("third trail point should be 50.2/20.2, got %.1f/%.1f", trail[2].Lat, trail[2].Lon)
	}

	// Trail should be chronological (oldest first).
	for i := 1; i < len(trail); i++ {
		if !trail[i].LastHeard.After(trail[i-1].LastHeard) {
			t.Errorf("trail not chronological at index %d", i)
		}
	}

	// Current station position should be 50.3/20.3 (not in trail).
	stations, _ := db.RecentStations(1)
	if len(stations) > 0 && (stations[0].Lat != 50.3 || stations[0].Lon != 20.3) {
		t.Errorf("current position should be 50.3/20.3, got %.1f/%.1f", stations[0].Lat, stations[0].Lon)
	}
}

func TestCacheDB_TrailNoiseSuppression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	// Insert initial position.
	s := StationRecord{Callsign: "STATIC", Lat: 50.0, Lon: 20.0, LastHeard: now}
	db.UpsertStation(s)

	// Many updates with tiny GPS jitter — should NOT create trail points.
	for i := 1; i <= 10; i++ {
		s.Lat = 50.0 + float64(i)*0.00001 // ~1m jitter
		s.Lon = 20.0 - float64(i)*0.00001
		s.LastHeard = now.Add(time.Duration(i) * time.Minute)
		db.UpsertStation(s)
	}

	trail, _ := db.StationTrail("STATIC", 10)
	if len(trail) != 0 {
		t.Errorf("GPS jitter should not create trail points, got %d", len(trail))
	}
}

func TestCacheDB_StationTrails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	// Station A: moves twice.
	a := StationRecord{Callsign: "CARA", Lat: 52.0, Lon: 13.0, LastHeard: now}
	db.UpsertStation(a)
	a.Lat = 52.1
	a.Lon = 13.1
	a.LastHeard = now.Add(1 * time.Minute)
	db.UpsertStation(a)

	// Station B: moves once.
	b := StationRecord{Callsign: "CARB", Lat: 48.0, Lon: 11.0, LastHeard: now}
	db.UpsertStation(b)
	b.Lat = 48.1
	b.Lon = 11.1
	b.LastHeard = now.Add(1 * time.Minute)
	db.UpsertStation(b)

	// Station C: no movement.
	c := StationRecord{Callsign: "FIXED", Lat: 40.0, Lon: -3.0, LastHeard: now}
	db.UpsertStation(c)

	trails, err := db.StationTrails([]string{"CARA", "CARB", "FIXED"})
	if err != nil {
		t.Fatalf("stationTrails: %v", err)
	}
	if len(trails["CARA"]) != 1 {
		t.Errorf("CARA should have 1 trail, got %d", len(trails["CARA"]))
	}
	if len(trails["CARB"]) != 1 {
		t.Errorf("CARB should have 1 trail, got %d", len(trails["CARB"]))
	}
	if trails["FIXED"] != nil {
		t.Errorf("FIXED should have no trail")
	}
}

func TestCacheDB_TrailPrune(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "aprs.db")

	db, err := OpenCacheDB(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()

	s := StationRecord{Callsign: "PRUNE", Lat: 50.0, Lon: 20.0, LastHeard: now.Add(-2 * time.Hour)}
	db.UpsertStation(s)
	s.Lat = 51.0
	s.Lon = 21.0
	s.LastHeard = now.Add(-90 * time.Minute)
	db.UpsertStation(s)

	// Prune everything older than 1 hour.
	cutoff := now.Add(-1 * time.Hour)
	db.PruneOlderThan(cutoff)

	// Station and its trail should both be gone.
	trail, _ := db.StationTrail("PRUNE", 5)
	if len(trail) != 0 {
		t.Errorf("trail should be pruned, got %d points", len(trail))
	}
	stations, _ := db.RecentStations(10)
	for _, st := range stations {
		if st.Callsign == "PRUNE" {
			t.Error("station should be pruned")
		}
	}
}
