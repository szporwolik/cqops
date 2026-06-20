package tui

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
	qsos, err := store.ListQSOs(m.App.DB, 1, "")
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

	cmd, _ := m.logQSOFromADIF(adif)
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
	cmd, _ := m.logQSOFromADIF(adif)
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
	cmd, _ := m.logQSOFromADIF(adif)
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

	cmd, _ := m.logQSOFromADIF(adif)
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

	cmd, _ := m.logQSOFromADIF(adif)
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

// =============================================================================
// Contest mode QSO tests
// =============================================================================

// newContestTestModel creates a Model with contests configured.
func newContestTestModel(t *testing.T, activeContest string) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	m.App.Config.Contests = map[string]config.Contest{
		"c1": {ID: "c1", Name: "CQ WPX", ContestID: "CQ-WPX-CW", NextQSO: 1},
		"c2": {ID: "c2", Name: "ARRL DX", ContestID: "ARRL-DX-CW", NextQSO: 42},
	}
	m.App.Config.State.ActiveContest = activeContest
	return m
}

func TestSaveQSOWithActiveContest(t *testing.T) {
	m := newContestTestModel(t, "c1")
	fillMinimalValidQSO(m)
	m.saveQSO()()

	saved := latestQSO(t, m)
	if saved.ContestID != "c1" {
		t.Errorf("ContestID = %q; want c1", saved.ContestID)
	}
}

func TestSaveQSOWithoutActiveContest(t *testing.T) {
	m := newContestTestModel(t, "") // None active
	fillMinimalValidQSO(m)
	m.saveQSO()()

	saved := latestQSO(t, m)
	if saved.ContestID != "" {
		t.Errorf("ContestID = %q; want empty", saved.ContestID)
	}
}

func TestSaveQSOContestFilteredRecentQSOs(t *testing.T) {
	m := newContestTestModel(t, "c1")

	// Save a QSO with contest c1 active
	fillMinimalValidQSO(m)
	m.fields[fieldCall].SetValue("DJ7NT")
	m.saveQSO()()

	// Switch to contest c2 and save another
	m.App.Config.State.ActiveContest = "c2"
	fillMinimalValidQSO(m)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.saveQSO()()

	// Recent QSOs should only show c2's QSO (active contest)
	if len(m.qsos) != 1 {
		t.Fatalf("RecentQSOs = %d QSOs; want 1 (only c2's)", len(m.qsos))
	}
	if m.qsos[0].Call != "SP9MOA" {
		t.Errorf("RecentQSOs[0].Call = %q; want SP9MOA", m.qsos[0].Call)
	}
}

func TestSaveQSOContestFilterRespectsNone(t *testing.T) {
	m := newContestTestModel(t, "c1")

	// Save with contest c1 active
	fillMinimalValidQSO(m)
	m.fields[fieldCall].SetValue("CT1AAA")
	m.saveQSO()()

	// Switch to None and save another
	m.App.Config.State.ActiveContest = ""
	fillMinimalValidQSO(m)
	m.fields[fieldCall].SetValue("DJ7NT")
	m.saveQSO()()

	// With None active, all QSOs should be visible
	if len(m.qsos) != 2 {
		t.Fatalf("RecentQSOs = %d QSOs; want 2 (none active → no filter)", len(m.qsos))
	}
}

func TestListQSOsContestFiltering(t *testing.T) {
	m := newLifecycleTestModel(t)
	db := m.App.DB

	// Insert QSOs with different contest IDs
	mustInsert := func(call, contestID string) {
		qs := qso.NewQSO()
		qs.Call = call
		qs.QSODate = "20260614"
		qs.TimeOn = "120000"
		qs.Band = "20m"
		qs.Mode = "SSB"
		qs.RSTSent = "59"
		qs.RSTRcvd = "59"
		qs.ContestID = contestID
		qso.ApplyStationDefaults(qs, qso.StationInfo{
			StationCallsign: m.App.Logbook.Station.Callsign,
			Operator:        m.App.Logbook.Station.Operator,
			MyGridSquare:    m.App.Logbook.Station.Grid,
		})
		if _, err := store.InsertQSO(db, qs); err != nil {
			t.Fatalf("InsertQSO: %v", err)
		}
	}

	mustInsert("C1AAA", "c1")
	mustInsert("C1BBB", "c1")
	mustInsert("C2AAA", "c2")
	mustInsert("NOCONT", "")

	// Filter by c1
	qsos, err := store.ListQSOs(db, 10, "c1")
	if err != nil {
		t.Fatalf("ListQSOs c1: %v", err)
	}
	if len(qsos) != 2 {
		t.Errorf("c1 filter: got %d QSOs, want 2", len(qsos))
	}
	for _, q := range qsos {
		if q.ContestID != "c1" {
			t.Errorf("c1 filter returned QSO with ContestID=%q", q.ContestID)
		}
	}

	// Filter by c2
	qsos, err = store.ListQSOs(db, 10, "c2")
	if err != nil {
		t.Fatalf("ListQSOs c2: %v", err)
	}
	if len(qsos) != 1 {
		t.Errorf("c2 filter: got %d QSOs, want 1", len(qsos))
	}

	// No filter
	qsos, err = store.ListQSOs(db, 10, "")
	if err != nil {
		t.Fatalf("ListQSOs all: %v", err)
	}
	if len(qsos) != 4 {
		t.Errorf("no filter: got %d QSOs, want 4", len(qsos))
	}
}

func TestCycleActiveContestToasts(t *testing.T) {
	m := newContestTestModel(t, "")

	// None → first alphabetically (c2 = ARRL DX)
	m.cycleActiveContest()
	if m.App.Config.State.ActiveContest != "c2" {
		t.Errorf("ActiveContest = %q; want c2 (ARRL DX, first alphabetically)", m.App.Config.State.ActiveContest)
	}
	if !m.needRefresh {
		t.Error("needRefresh should be true after contest cycle")
	}

	// c2 → c1 (CQ WPX)
	m.cycleActiveContest()
	if m.App.Config.State.ActiveContest != "c1" {
		t.Errorf("ActiveContest = %q; want c1 (CQ WPX, second)", m.App.Config.State.ActiveContest)
	}

	// c1 → None (wrap)
	m.cycleActiveContest()
	if m.App.Config.State.ActiveContest != "" {
		t.Errorf("ActiveContest = %q; want empty (None)", m.App.Config.State.ActiveContest)
	}

	// None → c2 (wrap)
	m.cycleActiveContest()
	if m.App.Config.State.ActiveContest != "c2" {
		t.Errorf("ActiveContest = %q; want c2 (wrap)", m.App.Config.State.ActiveContest)
	}
}

func TestContestBoxVisible(t *testing.T) {
	m := newContestTestModel(t, "c1")

	line := m.buildContestLine()
	if line == "" {
		t.Fatal("buildContestLine should return content when contest active")
	}
	if !strings.Contains(line, "CQ WPX") {
		t.Errorf("Contest line should contain name, got: %s", line)
	}
	if !strings.Contains(line, "CQ-WPX-CW") {
		t.Errorf("Contest line should contain Contest ID, got: %s", line)
	}
}

func TestContestBoxHiddenWhenNone(t *testing.T) {
	m := newContestTestModel(t, "")

	line := m.buildContestLine()
	if line != "" {
		t.Errorf("buildContestLine should be empty when no contest active, got: %s", line)
	}
}

func TestContestBoxHiddenWhenUnknownID(t *testing.T) {
	m := newContestTestModel(t, "bogus")

	line := m.buildContestLine()
	if line != "" {
		t.Errorf("buildContestLine should be empty for unknown contest ID, got: %s", line)
	}
}

func TestSaveQSOExchangeFields(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldExchSent].SetValue("599 001")
	m.fields[fieldExchRcvd].SetValue("599 042")
	m.saveQSO()()

	saved := latestQSO(t, m)
	if saved.ExchSent != "599 001" {
		t.Errorf("ExchSent = %q, want 599 001", saved.ExchSent)
	}
	if saved.ExchRcvd != "599 042" {
		t.Errorf("ExchRcvd = %q, want 599 042", saved.ExchRcvd)
	}
	if saved.STX != 1 {
		t.Errorf("STX = %d, want 1 (last integer in '599 001')", saved.STX)
	}
	if saved.SRX != 42 {
		t.Errorf("SRX = %d, want 42 (last integer in '599 042')", saved.SRX)
	}
	if saved.STXString != "599 001" {
		t.Errorf("STXString = %q, want 599 001", saved.STXString)
	}
	if saved.SRXString != "599 042" {
		t.Errorf("SRXString = %q, want 599 042", saved.SRXString)
	}
}

func TestSaveQSOExchangeEmpty(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	// Don't set exchange fields — leave them empty.
	m.saveQSO()()

	saved := latestQSO(t, m)
	if saved.ExchSent != "" {
		t.Errorf("ExchSent = %q, want empty", saved.ExchSent)
	}
	if saved.ExchRcvd != "" {
		t.Errorf("ExchRcvd = %q, want empty", saved.ExchRcvd)
	}
	if saved.STX != 0 {
		t.Errorf("STX = %d, want 0", saved.STX)
	}
	if saved.SRX != 0 {
		t.Errorf("SRX = %d, want 0", saved.SRX)
	}
}

func TestContestPrefillFillsExchFields(t *testing.T) {
	m := newContestTestModel(t, "c1")
	// Set up prefill template with ### marker.
	ct := m.App.Config.Contests["c1"]
	ct.PrefillExchange = true
	ct.ExchangeSent = "599 ###"
	ct.PrefillExchangeRcvd = true
	ct.ExchangeRcvd = "599 ###"
	ct.NextQSO = 7
	m.App.Config.Contests["c1"] = ct
	m.App.Config.State.ActiveContest = "c1"

	fillMinimalValidQSO(m)
	// Simulate leaving the call field (triggers prefill).
	m.focus = fieldCall
	m.onFieldExit()

	if m.fields[fieldExchSent].Value() != "599 007" {
		t.Errorf("ExchSent = %q, want 599 007", m.fields[fieldExchSent].Value())
	}
	if m.fields[fieldExchRcvd].Value() != "599 007" {
		t.Errorf("ExchRcvd = %q, want 599 007", m.fields[fieldExchRcvd].Value())
	}
	// NextQSO should NOT have incremented during prefill (happens on save).
	ct = m.App.Config.Contests["c1"]
	if ct.NextQSO != 7 {
		t.Errorf("NextQSO = %d, want 7 (not incremented during prefill)", ct.NextQSO)
	}
}

func TestContestPrefillLargeSerial(t *testing.T) {
	m := newContestTestModel(t, "c1")
	ct := m.App.Config.Contests["c1"]
	ct.PrefillExchange = true
	ct.ExchangeSent = "599 ###"
	ct.NextQSO = 10212
	m.App.Config.Contests["c1"] = ct
	m.App.Config.State.ActiveContest = "c1"

	fillMinimalValidQSO(m)
	m.focus = fieldCall
	m.onFieldExit()

	if m.fields[fieldExchSent].Value() != "599 10212" {
		t.Errorf("ExchSent = %q, want 599 10212", m.fields[fieldExchSent].Value())
	}
}

func TestContestPrefillOnlyWhenEnabled(t *testing.T) {
	m := newContestTestModel(t, "c1")
	ct := m.App.Config.Contests["c1"]
	ct.PrefillExchange = false // disabled
	ct.ExchangeSent = "599 ###"
	ct.NextQSO = 5
	m.App.Config.Contests["c1"] = ct
	m.App.Config.State.ActiveContest = "c1"

	fillMinimalValidQSO(m)
	m.focus = fieldCall
	m.onFieldExit()

	// Should NOT prefill when disabled.
	if m.fields[fieldExchSent].Value() != "" {
		t.Errorf("ExchSent should be empty when prefill is off, got %q", m.fields[fieldExchSent].Value())
	}
}

func TestContestPrefillOnlyWhenContestActive(t *testing.T) {
	m := newContestTestModel(t, "") // None active
	fillMinimalValidQSO(m)
	m.focus = fieldCall
	m.onFieldExit()

	// Should not prefill when no contest is active.
	if m.fields[fieldExchSent].Value() != "" {
		t.Errorf("ExchSent should be empty when no contest active, got %q", m.fields[fieldExchSent].Value())
	}
}

func TestClearFormResetsExchangeFields(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldExchSent].SetValue("599 001")
	m.fields[fieldExchRcvd].SetValue("599 042")

	m.clearForm()

	if m.fields[fieldExchSent].Value() != "" {
		t.Errorf("ExchSent should be cleared, got %q", m.fields[fieldExchSent].Value())
	}
	if m.fields[fieldExchRcvd].Value() != "" {
		t.Errorf("ExchRcvd should be cleared, got %q", m.fields[fieldExchRcvd].Value())
	}
}

func TestNextQSOIncrementsOnSave(t *testing.T) {
	m := newContestTestModel(t, "c1")
	ct := m.App.Config.Contests["c1"]
	ct.NextQSO = 7
	m.App.Config.Contests["c1"] = ct

	fillMinimalValidQSO(m)
	m.saveQSO()()

	ct = m.App.Config.Contests["c1"]
	if ct.NextQSO != 8 {
		t.Errorf("NextQSO = %d, want 8 (incremented on save)", ct.NextQSO)
	}
}

func TestNextQSONoIncrementWithoutContest(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.saveQSO()()

	// No contest active — nothing to check except no panic.
}

// =============================================================================
// Dupe check tests
// =============================================================================

func TestCheckDupe_DetectsDuplicate(t *testing.T) {
	m := newLifecycleTestModel(t)

	// First, save a QSO to create a dupe candidate.
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldTime].SetValue("120000")
	m.saveQSO()()

	// Now fill the same call/band/mode/date in the form.
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldSOTA].SetValue("")
	m.fields[fieldPOTA].SetValue("")
	m.fields[fieldWWFF].SetValue("")
	m.fields[fieldIOTA].SetValue("")

	// Simulate tabbing away from Call to trigger onFieldExit.
	m.focus = fieldCall
	m.onFieldExit()

	if !m.dupe {
		t.Error("dupe should be true when same call/band/mode/date exists")
	}

	// Verify the path row contains DUPE!
	m.width = 100
	m.height = 30
	m.rc.pathCall = "SP9MOA" // needed for formPathRow to render badges
	row := m.formPathRow(90)
	if !strings.Contains(row, "DUPE!") {
		t.Error("formPathRow should contain DUPE! when dupe is detected")
	}
}

func TestCheckDupe_NoDupeDifferentBand(t *testing.T) {
	m := newLifecycleTestModel(t)

	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldTime].SetValue("120000")
	m.saveQSO()()

	// Different band — should NOT be a dupe.
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("40m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")

	m.focus = fieldCall
	m.onFieldExit()

	if m.dupe {
		t.Error("dupe should be false when band differs")
	}
}

func TestCheckDupe_DifferentReferenceNotDupe(t *testing.T) {
	m := newLifecycleTestModel(t)

	// Save QSO with SOTA ref.
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldTime].SetValue("120000")
	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.saveQSO()()

	// Same call/band/mode/date but DIFFERENT SOTA ref — NOT a dupe.
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldSOTA].SetValue("SP/TA-002")

	m.focus = fieldCall
	m.onFieldExit()

	if m.dupe {
		t.Error("dupe should be false when SOTA ref differs (different summit)")
	}
}

func TestCheckDupe_ClearedOnFormReset(t *testing.T) {
	m := newLifecycleTestModel(t)

	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldDate].SetValue("20240501")
	m.fields[fieldTime].SetValue("120000")
	m.saveQSO()()

	// Set dupe to true (simulate detection).
	m.dupe = true

	// Clear form should reset dupe.
	m.clearForm()

	if m.dupe {
		t.Error("dupe should be false after clearForm")
	}
}
