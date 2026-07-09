package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Shared DXC test helpers
// =============================================================================
// Used by dxc_band_filter_test.go, dxc_time_filter_test.go, and
// dxc_mode_filter_test.go (Pass 19, 24, 25). No real DX Cluster connection.

// newDXCBandFilterModel creates a Model with an App that has a temp DB
// seeded with the given DXC spots. Returns the Model — no cleanup function
// needed because t.Cleanup is registered inside.
func newDXCBandFilterModel(t *testing.T, spots []store.DXCSpot) *Model {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "dxc_test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if len(spots) > 0 {
		if _, err := store.InsertDXCSpots(db, spots); err != nil {
			t.Fatalf("InsertDXCSpots: %v", err)
		}
	}

	cfg := &config.Config{
		General: config.GeneralConfig{Units: "metric", RenderMap: true},
		Logbooks: map[string]config.Logbook{
			"test": {
				Station: config.Station{
					Callsign: "SP9MOA",
					Grid:     "JO90",
					RigName:  "default",
				},
			},
		},
	}
	a := &app.App{
		Config:      cfg,
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90", RigName: "default"}},
		DB:          db,
	}
	m := New(a, nil)
	m.screen = screenDXC
	m.dxc.tableReady = false
	return m
}

// nowUnix returns the current UTC Unix timestamp for DXC spot ReceivedAt fields.
func nowUnix() int64 { return time.Now().UTC().Unix() }
