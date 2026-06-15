package tui

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

func init() {
	// Initialize applog with a discard logger for tests.
	// Prevents nil-pointer panic when ToastQueue calls applog.Info().
	applog.Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

// =============================================================================
// Lifecycle test helpers — temporary SQLite DB + minimal Model setup
// =============================================================================

// newLifecycleTestModel creates a Model with a temporary SQLite database,
// a minimal station config, and Wavelog/WSJT-X/flrig disabled.
func newLifecycleTestModel(t *testing.T) *Model {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cfg := &config.Config{
		General: config.GeneralConfig{DistanceUnit: "km", RenderMap: true},
		Logbooks: map[string]config.Logbook{
			"test": {
				Station: config.Station{
					Callsign: "SP9MOA",
					Grid:     "JO90",
					Operator: "OP",
					RigName:  "default",
				},
			},
		},
		Rigs: map[string]config.RigPreset{
			"default": {Model: "FT-891", Antenna: "Dipole", Power: "100"},
		},
	}
	// Disable all integrations for tests
	cfg.WSJTX.Enabled = false

	a := &app.App{
		Config:      cfg,
		ConfigPath:  "", // no config file
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90", Operator: "OP", RigName: "default"}, Wavelog: &config.WavelogConfig{}},
		DB:          db,
		DBPath:      dbPath,
	}

	m := New(a, nil)
	return m
}

// fillMinimalValidQSO fills the QSO form with the minimum valid values for save.
func fillMinimalValidQSO(m *Model) {
	m.fields[fieldCall].SetValue("DJ7NT")
	m.fields[fieldDate].SetValue("20260614")
	m.fields[fieldTime].SetValue("120000")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldRSTSent].SetValue("59")
	m.fields[fieldRSTRcvd].SetValue("59")
}

// countQSOS returns the number of QSOs in the test database.
func countQSOS(t *testing.T, m *Model) int {
	t.Helper()
	qsos, err := store.ListAllQSOs(m.App.DB)
	if err != nil {
		t.Fatalf("ListAllQSOs: %v", err)
	}
	return len(qsos)
}

// latestQSO returns the most recently inserted QSO from the test database.
func latestQSO(t *testing.T, m *Model) qso.QSO {
	t.Helper()
	qsos, err := store.ListQSOs(m.App.DB, 1)
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) == 0 {
		t.Fatal("No QSOs in database")
	}
	return qsos[0]
}

// =============================================================================
// refreshQSOS tests
// =============================================================================

func TestRefreshQSOsEmptyDB(t *testing.T) {
	m := newLifecycleTestModel(t)

	// refreshQSOS returns a tea.Cmd — execute it
	cmd := m.refreshQSOS()
	if cmd == nil {
		t.Fatal("refreshQSOS returned nil command")
	}
	cmd() // execute the function

	// RecentQSOs should be empty
	if len(m.qsos) != 0 {
		t.Errorf("Expected 0 QSOs after refresh on empty DB, got %d", len(m.qsos))
	}
}

func TestRefreshQSOsOneQSO(t *testing.T) {
	m := newLifecycleTestModel(t)

	// Insert a QSO directly into the DB
	qs := qso.NewQSO()
	qs.Call = "DJ7NT"
	qs.QSODate = "20260614"
	qs.TimeOn = "120000"
	qs.Band = "20m"
	qs.Mode = "SSB"
	qs.RSTSent = "59"
	qs.RSTRcvd = "59"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Operator,
		MyGridSquare:    m.App.Logbook.Station.Grid,
	})
	if _, err := store.InsertQSO(m.App.DB, qs); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	m.refreshQSOS()()
	if len(m.qsos) != 1 {
		t.Errorf("Expected 1 QSO after refresh, got %d", len(m.qsos))
	}
	if m.qsos[0].Call != "DJ7NT" {
		t.Errorf("Expected DJ7NT, got %s", m.qsos[0].Call)
	}
}

func TestRefreshQSOsMultipleQSOs(t *testing.T) {
	m := newLifecycleTestModel(t)

	// Insert 3 QSOs
	for _, call := range []string{"AA1AA", "BB2BB", "CC3CC"} {
		qs := qso.NewQSO()
		qs.Call = call
		qs.QSODate = "20260614"
		qs.TimeOn = "120000"
		qs.Band = "20m"
		qs.Mode = "SSB"
		qs.RSTSent = "59"
		qs.RSTRcvd = "59"
		qso.ApplyStationDefaults(qs, qso.StationInfo{
			StationCallsign: m.App.Logbook.Station.Callsign,
			Operator:        m.App.Logbook.Station.Operator,
			MyGridSquare:    m.App.Logbook.Station.Grid,
		})
		store.InsertQSO(m.App.DB, qs)
	}

	m.refreshQSOS()()
	if len(m.qsos) != 3 {
		t.Errorf("Expected 3 QSOs after refresh, got %d", len(m.qsos))
	}
	// Most recent first (ID DESC)
	if m.qsos[0].Call != "CC3CC" {
		t.Errorf("Expected CC3CC first, got %s", m.qsos[0].Call)
	}
}

// =============================================================================
// saveQSO tests
// =============================================================================

func TestSaveQSOValid(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldComment].SetValue("Test QSO")

	cmd := m.saveQSO()
	if cmd == nil {
		t.Fatal("saveQSO returned nil")
	}
	// Execute the tea.Cmd (Batch of refreshQSOS + Wavelog upload)
	cmd()

	// Verify QSO in DB
	if countQSOS(t, m) != 1 {
		t.Fatal("Expected 1 QSO in DB after save")
	}
	saved := latestQSO(t, m)
	if saved.Call != "DJ7NT" {
		t.Errorf("Call = %q; want DJ7NT", saved.Call)
	}
	if saved.Band != "20m" {
		t.Errorf("Band = %q; want 20m", saved.Band)
	}
	if saved.Mode != "SSB" {
		t.Errorf("Mode = %q; want SSB", saved.Mode)
	}
	if saved.RSTSent != "59" {
		t.Errorf("RSTSent = %q; want 59", saved.RSTSent)
	}
	if saved.RSTRcvd != "59" {
		t.Errorf("RSTRcvd = %q; want 59", saved.RSTRcvd)
	}
	if saved.QSODate != "20260614" {
		t.Errorf("QSODate = %q; want 20260614", saved.QSODate)
	}
	if saved.TimeOn != "120000" {
		t.Errorf("TimeOn = %q; want 120000", saved.TimeOn)
	}
	if saved.Comment != "Test QSO" {
		t.Errorf("Comment = %q; want 'Test QSO'", saved.Comment)
	}
	if saved.Source != "manual" {
		t.Errorf("Source = %q; want 'manual'", saved.Source)
	}
}

func TestSaveQSOInvalidNoCall(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldCall].SetValue("") // clear call

	cmd := m.saveQSO()
	if cmd != nil {
		// Should not return a valid command for invalid save
		// But saveQSO may still return nil after showing error toast
		// Check that no QSO was inserted
	}
	if countQSOS(t, m) != 0 {
		t.Error("No QSO should be saved with empty call")
	}
}

func TestSaveQSOInvalidNoMode(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldMode].SetValue("")

	cmd := m.saveQSO()
	if cmd != nil {
		cmd()
	}
	if countQSOS(t, m) != 0 {
		t.Error("No QSO should be saved with empty mode")
	}
}

func TestSaveQSOInvalidNoRST(t *testing.T) {
	// Note: autoFillRST() runs before validation, so empty RST gets auto-filled.
	// This test verifies that autoFillRST does its job — the QSO should save.
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldRSTSent].SetValue("")
	m.fields[fieldRSTRcvd].SetValue("")

	cmd := m.saveQSO()
	if cmd == nil {
		t.Fatal("saveQSO should succeed — autoFillRST fills empty RST")
	}
	cmd()

	if countQSOS(t, m) != 1 {
		t.Error("QSO should be saved — autoFillRST fills empty RST fields")
	}
	saved := latestQSO(t, m)
	if saved.RSTSent != "59" {
		t.Errorf("RSTSent should be auto-filled to 59, got %q", saved.RSTSent)
	}
}

func TestSaveQSORefreshesRecentQSOs(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)

	m.saveQSO()()
	// After save, refreshQSOS should have updated m.qsos
	if len(m.qsos) != 1 {
		t.Errorf("RecentQSOs should have 1 QSO after save, got %d", len(m.qsos))
	}
}

func TestSaveQSORetainDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldComment].SetValue("Will be cleared")
	m.retainComment = false

	m.saveQSO()()

	// After save with retain disabled, comment should be cleared
	if m.fields[fieldComment].Value() != "" {
		t.Errorf("Comment should be cleared when retain is off, got %q", m.fields[fieldComment].Value())
	}
	// Call should be cleared
	if m.fields[fieldCall].Value() != "" {
		t.Errorf("Call should be cleared after save, got %q", m.fields[fieldCall].Value())
	}
}

func TestSaveQSORetainEnabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldComment].SetValue("Retain this comment")
	m.retainComment = true

	m.saveQSO()()

	// After save with retain enabled, comment should be preserved
	if m.fields[fieldComment].Value() != "Retain this comment" {
		t.Errorf("Comment should be preserved when retain is on, got %q", m.fields[fieldComment].Value())
	}
}

func TestSaveQSOStationDefaults(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)

	m.saveQSO()()
	saved := latestQSO(t, m)

	if saved.StationCallsign != "SP9MOA" {
		t.Errorf("StationCallsign = %q; want SP9MOA", saved.StationCallsign)
	}
	if saved.Operator != "OP" {
		t.Errorf("Operator = %q; want OP", saved.Operator)
	}
	if saved.MyGridSquare != "JO90" {
		t.Errorf("MyGridSquare = %q; want JO90", saved.MyGridSquare)
	}
	if saved.MyRig != "FT-891" {
		t.Errorf("MyRig = %q; want FT-891", saved.MyRig)
	}
	if saved.MyAntenna != "Dipole" {
		t.Errorf("MyAntenna = %q; want Dipole", saved.MyAntenna)
	}
}

func TestSaveQSOSuccessToast(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)

	m.saveQSO()()

	// Verify a success toast was created
	toasts := m.toasts.Active()
	found := false
	for _, toast := range toasts {
		if toast.Level == ToastSuccess {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected a success toast after save")
	}
}

// =============================================================================
// WSJT-X logQSOFromADIF tests
// =============================================================================

func TestLogQSOFromADIFValid(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = false

	adif := "<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <FREQ:7>14.2500 " +
		"<QSO_DATE:8>20260614 <TIME_ON:6>120000 " +
		"<RST_SENT:2>59 <RST_RCVD:2>59 <GRIDSQUARE:4>JO90 <EOR>"

	cmd := m.logQSOFromADIF(adif)
	// cmd now always includes refreshQSOS via tea.Batch
	if cmd != nil {
		cmd()
	}

	if countQSOS(t, m) != 1 {
		t.Fatal("Expected 1 QSO in DB after WSJT-X ADIF log")
	}
	saved := latestQSO(t, m)
	if saved.Call != "SP9MOA" {
		t.Errorf("Call = %q; want SP9MOA", saved.Call)
	}
}

func TestLogQSOFromADIFNoCall(t *testing.T) {
	m := newLifecycleTestModel(t)

	adif := "<BAND:3>20m <MODE:3>SSB <EOR>"
	cmd := m.logQSOFromADIF(adif)
	if cmd != nil {
		cmd()
	}
	if countQSOS(t, m) != 0 {
		t.Error("No QSO should be saved when ADIF has no call")
	}
}

func TestLogQSOFromADIFInvalidMode(t *testing.T) {
	m := newLifecycleTestModel(t)

	adif := "<CALL:6>SP9MOA <MODE:10>INVALIDMOD <EOR>"
	cmd := m.logQSOFromADIF(adif)
	if cmd != nil {
		cmd()
	}
	// Should not save — mode validation fails
	if countQSOS(t, m) != 0 {
		t.Error("No QSO should be saved with invalid mode")
	}
}

func TestLogQSOFromADIFUpdatesRecentQSOs(t *testing.T) {
	m := newLifecycleTestModel(t)

	adif := "<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB " +
		"<QSO_DATE:8>20260614 <TIME_ON:6>120000 " +
		"<RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	cmd := m.logQSOFromADIF(adif)
	// cmd always includes refreshQSOS via tea.Batch
	if cmd != nil {
		cmd()
	}

	if len(m.qsos) != 1 {
		t.Errorf("RecentQSOs should have 1 QSO after ADIF log, got %d", len(m.qsos))
	}
}

func TestLogQSOFromADIFWavelogDisabled(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.App.Logbook.Wavelog.Enabled = false

	adif := "<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB " +
		"<QSO_DATE:8>20260614 <TIME_ON:6>120000 " +
		"<RST_SENT:2>59 <RST_RCVD:2>59 <EOR>"

	cmd := m.logQSOFromADIF(adif)
	// cmd includes refreshQSOS via tea.Batch even when Wavelog is disabled
	if cmd == nil {
		t.Fatal("logQSOFromADIF should return cmd even when Wavelog disabled (refreshQSOS)")
	}
	cmd()

	if countQSOS(t, m) != 1 {
		t.Error("QSO should be saved even when Wavelog is disabled")
	}
	// RecentQSOs should be refreshed immediately
	if len(m.qsos) != 1 {
		t.Error("RecentQSOs should be refreshed immediately even when Wavelog is disabled")
	}
}
