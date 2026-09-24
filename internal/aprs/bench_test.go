package aprs

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

const benchPacket = `SP9MOA-9>APRS,TCPIP*:!5003.50N/01959.20E>088/017/A=000312 test comment`

// ParsePositionPacket runs once per received APRS packet.
func BenchmarkParsePositionPacket(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := ParsePositionPacket(benchPacket); !ok {
			b.Fatal("packet did not parse")
		}
	}
}

// UpsertStation runs once per received APRS packet and is the cache's
// only write path, so its cost sets the ceiling on sustainable packet rate.
func BenchmarkUpsertStation(b *testing.B) {
	path := filepath.Join(b.TempDir(), "aprs_bench.db")
	db, err := OpenCacheDB(path)
	if err != nil {
		b.Fatalf("OpenCacheDB: %v", err)
	}
	defer db.Close()

	now := time.Now()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := StationRecord{
			// Rotate callsigns and nudge position so the trail-history
			// path is exercised the way a live feed would.
			Callsign:  fmt.Sprintf("SP9M%02d", i%50),
			Lat:       50.0 + float64(i%100)*0.001,
			Lon:       20.0 + float64(i%100)*0.001,
			Symbol:    "/>",
			Comment:   "bench",
			LastHeard: now.Add(time.Duration(i) * time.Second),
			Source:    "aprs_is",
		}
		if err := db.UpsertStation(rec); err != nil {
			b.Fatalf("UpsertStation: %v", err)
		}
	}
}
