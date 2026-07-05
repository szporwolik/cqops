package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// spotModeCategory tests
// =============================================================================

func TestSpotModeCategory_CW(t *testing.T) {
	for _, m := range []string{"CW", "CW-L", "CW-U", "CWL", "CWU", "CW-R", "cw", "Cw"} {
		if got := spotModeCategory(m); got != "CW" {
			t.Errorf("spotModeCategory(%q) = %q, want CW", m, got)
		}
	}
}

func TestSpotModeCategory_DIGI(t *testing.T) {
	for _, m := range []string{"FT8", "FT4", "RTTY", "PSK", "JT65", "JT9", "MSK144", "FSK", "DATA", "DATA-U", "DATA-L", "DATA-FM"} {
		if got := spotModeCategory(m); got != "DIGI" {
			t.Errorf("spotModeCategory(%q) = %q, want DIGI", m, got)
		}
	}
}

func TestSpotModeCategory_PHONE(t *testing.T) {
	for _, m := range []string{"USB", "LSB", "AM", "FM"} {
		if got := spotModeCategory(m); got != "PHONE" {
			t.Errorf("spotModeCategory(%q) = %q, want PHONE", m, got)
		}
	}
}

func TestSpotModeCategory_Unknown(t *testing.T) {
	for _, m := range []string{"", "BOGUS", "SSB", "DATA-FM-DEEP"} {
		if got := spotModeCategory(m); got != "" {
			t.Errorf("spotModeCategory(%q) = %q, want empty", m, got)
		}
	}
}

// =============================================================================
// spotModeToFlrigMode tests
// =============================================================================

func TestSpotModeToFlrigMode_WSJTX(t *testing.T) {
	for _, m := range []string{"FT8", "FT4", "JT65", "JT9", "MSK144", "Q65", "FST4", "FST4W", "JS8", "WSPR"} {
		if got := spotModeToFlrigMode(m); got != "DATA-U" {
			t.Errorf("spotModeToFlrigMode(%q) = %q, want DATA-U", m, got)
		}
	}
}

func TestSpotModeToFlrigMode_CW(t *testing.T) {
	for _, m := range []string{"CW", "CW-L", "CWL", "CW-R"} {
		if got := spotModeToFlrigMode(m); got != "CW" {
			t.Errorf("spotModeToFlrigMode(%q) = %q, want CW", m, got)
		}
	}
}

func TestSpotModeToFlrigMode_Phone(t *testing.T) {
	tests := map[string]string{
		"USB":   "USB",
		"LSB":   "LSB",
		"AM":    "AM",
		"FM":    "FM",
		"C4FM":  "FM",
		"DMR":   "FM",
		"DSTAR": "FM",
	}
	for in, want := range tests {
		if got := spotModeToFlrigMode(in); got != want {
			t.Errorf("spotModeToFlrigMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSpotModeToFlrigMode_Digital(t *testing.T) {
	for _, m := range []string{"RTTY", "PSK31", "MFSK", "OLIVIA", "CONTESTI", "FSK"} {
		if got := spotModeToFlrigMode(m); got != "DATA-U" {
			t.Errorf("spotModeToFlrigMode(%q) = %q, want DATA-U", m, got)
		}
	}
}

func TestSpotModeToFlrigMode_Unknown(t *testing.T) {
	// Default fallback is DATA-U for unknown modes — safe default for ham radio.
	if got := spotModeToFlrigMode("BOGUS"); got != "DATA-U" {
		t.Errorf("spotModeToFlrigMode(BOGUS) = %q, want DATA-U (safe default)", got)
	}
	if got := spotModeToFlrigMode(""); got != "DATA-U" {
		t.Errorf("spotModeToFlrigMode(\"\") = %q, want DATA-U (safe default)", got)
	}
}

func TestSpotModeToFlrigMode_CaseInsensitive(t *testing.T) {
	if got := spotModeToFlrigMode("ft8"); got != "DATA-U" {
		t.Errorf("spotModeToFlrigMode(ft8) = %q, want DATA-U", got)
	}
	if got := spotModeToFlrigMode("cw"); got != "CW" {
		t.Errorf("spotModeToFlrigMode(cw) = %q, want CW", got)
	}
}

// =============================================================================
// dxcColValue tests
// =============================================================================

func TestDXCColValue_AllColumns(t *testing.T) {
	now := time.Now().UTC().Unix()
	s := &store.DXCSpot{
		DXCall: "SP9XXX", Frequency: 14250, Band: "20m", Mode: "FT8",
		Comment: "TNX QSO", Spotter: "K1ABC", ReceivedAt: now,
	}
	if v := dxcColValue("DX Call", s); v != "SP9XXX" {
		t.Errorf("DX Call = %q", v)
	}
	if v := dxcColValue("Freq", s); !strings.HasPrefix(v, "14") {
		t.Errorf("Freq = %q, want 14.2...", v)
	}
	if v := dxcColValue("Band", s); v != "20m" {
		t.Errorf("Band = %q", v)
	}
	if v := dxcColValue("Mode", s); v != "FT8" {
		t.Errorf("Mode = %q", v)
	}
	if v := dxcColValue("Spotter", s); v != "K1ABC" {
		t.Errorf("Spotter = %q", v)
	}
	if v := dxcColValue("Comment", s); v != "TNX QSO" {
		t.Errorf("Comment = %q", v)
	}
}

func TestDXCColValue_EmptyFields(t *testing.T) {
	s := &store.DXCSpot{}
	if v := dxcColValue("DX Call", s); v != "" {
		t.Errorf("empty DX Call = %q, want empty", v)
	}
	if v := dxcColValue("Freq", s); v != "\u2014" {
		t.Errorf("empty Freq = %q, want em dash", v)
	}
}

func TestDXCColValue_UnknownColumn(t *testing.T) {
	s := &store.DXCSpot{DXCall: "SP9XXX"}
	if v := dxcColValue("Bogus", s); v != "" {
		t.Errorf("unknown column = %q, want empty", v)
	}
}

func TestDXCColValue_TimeFormat(t *testing.T) {
	// 2026-06-18 12:34:56 UTC
	ts := time.Date(2026, 6, 18, 12, 34, 56, 0, time.UTC).Unix()
	s := &store.DXCSpot{ReceivedAt: ts}
	v := dxcColValue("Time", s)
	if v != "12:34:56" {
		t.Errorf("Time = %q, want 12:34:56", v)
	}
}

// =============================================================================
// dxcSpotAtCursor safety tests
// =============================================================================

func TestDXCSpotAtCursor_NoTable(t *testing.T) {
	m := &Model{}
	_, ok := m.dxcSpotAtCursor()
	if ok {
		t.Error("dxcSpotAtCursor with no table should return false")
	}
}

func TestDXCSpotAtCursor_TableNotReady(t *testing.T) {
	m := &Model{}
	m.dxc.tableReady = false
	_, ok := m.dxcSpotAtCursor()
	if ok {
		t.Error("dxcSpotAtCursor with table not ready should return false")
	}
}

func TestDXCSpotAtCursor_NoSelectedCall(t *testing.T) {
	m := &Model{}
	m.dxc.tableReady = true
	m.dxc.selectedCall = ""
	_, ok := m.dxcSpotAtCursor()
	if ok {
		t.Error("dxcSpotAtCursor with empty selectedCall should return false")
	}
}

// =============================================================================
// parseSpotCommentForRefs tests
// =============================================================================

// newModelForRefParse creates a minimal Model with form fields for
// parseSpotCommentForRefs testing. No DB, no integrations.
func newModelForRefParse() *Model {
	cfg := &config.Config{
		General: config.GeneralConfig{Units: "metric"},
		Logbooks: map[string]config.Logbook{
			"test": {Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
		},
	}
	a := &app.App{
		Config:      cfg,
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"}},
	}
	return New(a, nil)
}

func TestParseSpotCommentForRefs_POTA(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("POTA US-1091")
	if got := m.fields[fieldPOTA].Value(); got != "US-1091" {
		t.Errorf("POTA = %q, want US-1091", got)
	}
	// IOTA must NOT be filled with a POTA reference.
	if got := m.fields[fieldIOTA].Value(); got != "" {
		t.Errorf("IOTA = %q, want empty (POTA ref should not fill IOTA)", got)
	}
	if got := m.fields[fieldSOTA].Value(); got != "" {
		t.Errorf("SOTA = %q, want empty", got)
	}
	if got := m.fields[fieldWWFF].Value(); got != "" {
		t.Errorf("WWFF = %q, want empty", got)
	}
}

func TestParseSpotCommentForRefs_IOTA(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("IOTA EU-005")
	if got := m.fields[fieldIOTA].Value(); got != "EU-005" {
		t.Errorf("IOTA = %q, want EU-005", got)
	}
	// POTA must NOT be filled with an IOTA reference.
	if got := m.fields[fieldPOTA].Value(); got != "" {
		t.Errorf("POTA = %q, want empty", got)
	}
}

func TestParseSpotCommentForRefs_SOTA(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("SOTA SP/BZ-001")
	if got := m.fields[fieldSOTA].Value(); got != "SP/BZ-001" {
		t.Errorf("SOTA = %q, want SP/BZ-001", got)
	}
}

func TestParseSpotCommentForRefs_WWFF(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("WWFF DLFF-0123")
	if got := m.fields[fieldWWFF].Value(); got != "DLFF-0123" {
		t.Errorf("WWFF = %q, want DLFF-0123", got)
	}
	// POTA must NOT be filled with a WWFF reference.
	if got := m.fields[fieldPOTA].Value(); got != "" {
		t.Errorf("POTA = %q, want empty (WWFF ref should not fill POTA)", got)
	}
}

func TestParseSpotCommentForRefs_NoFalseIOTA(t *testing.T) {
	// US, PL, SP etc are NOT valid IOTA continent codes.
	tests := []string{
		"US-1091", // POTA reference
		"PL-1234", // POTA reference
		"K-0001",  // POTA reference
		"SP-0001", // POTA reference
		"VE-1234", // POTA reference
		"G-0001",  // POTA reference
	}
	for _, tc := range tests {
		m := newModelForRefParse()
		m.parseSpotCommentForRefs(tc)
		if got := m.fields[fieldIOTA].Value(); got != "" {
			t.Errorf("comment %q: IOTA = %q, want empty", tc, got)
		}
		if got := m.fields[fieldPOTA].Value(); got != tc {
			t.Errorf("comment %q: POTA = %q, want %q", tc, got, tc)
		}
	}
}

func TestParseSpotCommentForRefs_ValidIOTA(t *testing.T) {
	// All valid IOTA continent codes must be recognized.
	tests := []string{
		"AF-001", "AN-016", "AS-007", "EU-005",
		"NA-001", "OC-001", "SA-002",
	}
	for _, tc := range tests {
		m := newModelForRefParse()
		m.parseSpotCommentForRefs(tc)
		if got := m.fields[fieldIOTA].Value(); got != tc {
			t.Errorf("comment %q: IOTA = %q, want %q", tc, got, tc)
		}
	}
}

func TestParseSpotCommentForRefs_MultipleRefs(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("EU-005 and SP/BZ-001 with SP-0030")
	if got := m.fields[fieldIOTA].Value(); got != "EU-005" {
		t.Errorf("IOTA = %q, want EU-005", got)
	}
	if got := m.fields[fieldSOTA].Value(); got != "SP/BZ-001" {
		t.Errorf("SOTA = %q, want SP/BZ-001", got)
	}
	if got := m.fields[fieldPOTA].Value(); got != "SP-0030" {
		t.Errorf("POTA = %q, want SP-0030", got)
	}
}

func TestParseSpotCommentForRefs_Empty(t *testing.T) {
	m := newModelForRefParse()
	m.parseSpotCommentForRefs("")
	if got := m.fields[fieldIOTA].Value(); got != "" {
		t.Errorf("IOTA = %q, want empty", got)
	}
	if got := m.fields[fieldPOTA].Value(); got != "" {
		t.Errorf("POTA = %q, want empty", got)
	}
	if got := m.fields[fieldSOTA].Value(); got != "" {
		t.Errorf("SOTA = %q, want empty", got)
	}
}

func TestDXCSpotAtCursor_NoSelectedSpot(t *testing.T) {
	m := &Model{}
	m.dxc.tableReady = true
	m.dxc.selectedCall = "SP9XXX"
	m.dxc.selectedSpot = store.DXCSpot{}
	_, ok := m.dxcSpotAtCursor()
	if ok {
		t.Error("dxcSpotAtCursor with empty selectedSpot should return false")
	}
}

func TestDXCSpotAtCursor_Valid(t *testing.T) {
	m := &Model{}
	m.dxc.tableReady = true
	m.dxc.selectedCall = "SP9XXX"
	m.dxc.selectedSpot = store.DXCSpot{DXCall: "SP9XXX", Frequency: 14250}
	spot, ok := m.dxcSpotAtCursor()
	if !ok {
		t.Fatal("dxcSpotAtCursor should return true")
	}
	if spot.DXCall != "SP9XXX" {
		t.Errorf("DXCall = %q", spot.DXCall)
	}
	if spot.Frequency != 14250 {
		t.Errorf("Frequency = %.0f", spot.Frequency)
	}
}

// =============================================================================
// dxcFillFromSelected safety tests
// =============================================================================

func TestDXCFillFromSelected_NoSpot(t *testing.T) {
	m := &Model{}
	// Calling with no selected spot should not panic — just return silently.
	m.dxcFillFromSelected()
	// No panic = pass.
}

// =============================================================================
// dxcTuneCmd safety tests
// =============================================================================

func TestDXCTuneCmd_NoRig(t *testing.T) {
	m := &Model{}
	m.rig.connected = false
	m.wsjtx.online = false
	m.rig.client = nil
	cmd := m.dxcTuneCmd()
	if cmd != nil {
		t.Error("dxcTuneCmd with no rig should return nil")
	}
}

func TestDXCTuneCmd_WSJTXOnline(t *testing.T) {
	m := &Model{}
	m.rig.connected = true
	m.wsjtx.online = true
	cmd := m.dxcTuneCmd()
	if cmd != nil {
		t.Error("dxcTuneCmd with WSJT-X online should return nil")
	}
}

func TestDXCTuneCmd_NoSelectedSpot(t *testing.T) {
	m := &Model{}
	m.rig.connected = true
	m.wsjtx.online = false
	m.rig.client = &fakeRigClient{}
	m.dxc.tableReady = false
	cmd := m.dxcTuneCmd()
	if cmd != nil {
		t.Error("dxcTuneCmd with no selected spot should return nil")
	}
}

// =============================================================================
// updateDXCSelectedCall tests
// =============================================================================

func TestUpdateDXCSelectedCall_NotReady(t *testing.T) {
	m := &Model{}
	m.dxc.tableReady = false
	m.dxc.selectedCall = "SP9XXX"
	m.updateDXCSelectedCall()
	if m.dxc.selectedCall != "" {
		t.Error("updateDXCSelectedCall should clear selectedCall when table not ready")
	}
}

// =============================================================================
// DXC key/state-transition tests
// =============================================================================
// These tests verify handleDXCUpdate behaviour with fake key messages.
// No real DB, network, or terminal rendering is involved.

// newDXCTestModel returns a Model with DXC screen active and empty DB.
func newDXCTestModel() *Model {
	m := newTestModel()
	m.screen = screenDXC
	m.dxc.tableReady = false // force rebuild on first View
	return m
}

func TestDXCKeys_EscapeReturnsToQSO(t *testing.T) {
	m := newDXCTestModel()
	m.screen = screenDXC
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEscape}, nil)
	if m.screen != screenQSO {
		t.Errorf("escape on DXC screen should return to QSO, got screen=%v", m.screen)
	}
}

func TestDXCKeys_TimeCycleForward(t *testing.T) {
	m := newDXCTestModel()
	// dxcTimeWindows = {0, 60, 30, 15, 10, 5}
	// Start at index 0 (all, timeFilter=0).
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil) // → 60m
	if m.dxc.timeFilter != 60 {
		t.Errorf("PgUp should set timeFilter=60, got %d", m.dxc.timeFilter)
	}
	// Cycle through all windows back to start.
	for i := 0; i < len(dxcTimeWindows)+1; i++ {
		_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	}
	// 1 initial + 7 more = 8 PgUp presses → index 2 (30m) on 6-element array.
	if m.dxc.timeFilter != 30 {
		t.Errorf("after full cycle+1 PgUp, timeFilter=%d", m.dxc.timeFilter)
	}
}

func TestDXCKeys_TimeCycleBackward(t *testing.T) {
	m := newDXCTestModel()
	// PgDown from start wraps to last element.
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgDown}, nil)
	if m.dxc.timeFilter != dxcTimeWindows[len(dxcTimeWindows)-1] {
		t.Errorf("PgDown from start should wrap to %d, got %d",
			dxcTimeWindows[len(dxcTimeWindows)-1], m.dxc.timeFilter)
	}
}

func TestDXCKeys_ModeCycleForward(t *testing.T) {
	m := newDXCTestModel()
	// Mode choices: "", "CW", "DIGI", "PHONE"
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "CW" {
		t.Errorf("Insert should set modeFilter=CW, got %q", m.dxc.modeFilter)
	}
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil)
	if m.dxc.modeFilter != "DIGI" {
		t.Errorf("2nd Insert should set modeFilter=DIGI, got %q", m.dxc.modeFilter)
	}
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil) // PHONE
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyInsert}, nil) // back to ""
	if m.dxc.modeFilter != "" {
		t.Errorf("4th Insert should set modeFilter=\"\", got %q", m.dxc.modeFilter)
	}
}

func TestDXCKeys_ModeCycleBackward(t *testing.T) {
	m := newDXCTestModel()
	// Delete from "" wraps to "PHONE".
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyDelete}, nil)
	if m.dxc.modeFilter != "PHONE" {
		t.Errorf("Delete from start should wrap to PHONE, got %q", m.dxc.modeFilter)
	}
}

func TestDXCKeys_ClearFilters(t *testing.T) {
	m := newDXCTestModel()
	// Set some filters first.
	m.dxc.timeFilter = 30
	m.dxc.bandFilter = "20m"
	m.dxc.modeFilter = "CW"
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)
	if m.dxc.timeFilter != 0 {
		t.Errorf("Backspace should clear timeFilter, got %d", m.dxc.timeFilter)
	}
	if m.dxc.bandFilter != "" {
		t.Errorf("Backspace should clear bandFilter, got %q", m.dxc.bandFilter)
	}
	if m.dxc.modeFilter != "" {
		t.Errorf("Backspace should clear modeFilter, got %q", m.dxc.modeFilter)
	}
}

func TestDXCKeys_FilterForcesTableRebuild(t *testing.T) {
	m := newDXCTestModel()
	m.dxc.tableReady = true
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyPgUp}, nil)
	if m.dxc.tableReady {
		t.Error("filter change should set tableReady=false to force rebuild")
	}
}

func TestDXCKeys_EnterFillsAndGoesToQSO(t *testing.T) {
	m := newDXCTestModel()
	m.dxc.selectedCall = "SP9XXX"
	m.dxc.selectedSpot = store.DXCSpot{DXCall: "SP9XXX", Frequency: 14250}
	m.dxc.tableReady = true
	m.wsjtx.online = false
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnter}, nil)
	if m.screen != screenQSO {
		t.Errorf("Enter should switch to screenQSO, got %v", m.screen)
	}
	if m.fields[fieldCall].Value() != "SP9XXX" {
		t.Errorf("Enter should fill call field, got %q", m.fields[fieldCall].Value())
	}
}

func TestDXCKeys_EnterFillsTunesAndJumps(t *testing.T) {
	m := newDXCTestModel()
	m.dxc.selectedCall = "SP9XXX"
	m.dxc.selectedSpot = store.DXCSpot{DXCall: "SP9XXX", Frequency: 14250}
	m.dxc.tableReady = true
	m.wsjtx.online = false
	m.rig.connected = true
	m.rig.client = &fakeRigClient{}
	_, cmd := m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEnter}, nil)
	if cmd == nil {
		t.Error("Enter should return a tune command when rig is connected")
	}
	if m.fields[fieldCall].Value() != "SP9XXX" {
		t.Errorf("Enter should fill call field, got %q", m.fields[fieldCall].Value())
	}
	if m.screen != screenQSO {
		t.Errorf("Enter should switch to QSO screen, got %v", m.screen)
	}
}

func TestDXCKeys_SpaceTunesOnly(t *testing.T) {
	m := newDXCTestModel()
	m.dxc.selectedCall = "SP9XXX"
	m.dxc.selectedSpot = store.DXCSpot{DXCall: "SP9XXX", Frequency: 14250}
	m.dxc.tableReady = true
	m.wsjtx.online = false
	m.rig.connected = true
	m.rig.client = &fakeRigClient{}
	_, cmd := m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeySpace}, nil)
	if cmd == nil {
		t.Error("Space should return a tune command when rig is connected")
	}
	if m.fields[fieldCall].Value() != "" {
		t.Errorf("Space should NOT fill call field, got %q", m.fields[fieldCall].Value())
	}
}

func TestDXCKeys_UnknownKeyNoOp(t *testing.T) {
	m := newDXCTestModel()
	m.screen = screenDXC
	prevScreen := m.screen
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyF1}, nil) // F1 not handled in DXC
	if m.screen != prevScreen {
		t.Errorf("unhandled key should not change screen, got %v", m.screen)
	}
}

// =============================================================================
// dxcAvailableModes tests
// =============================================================================

func TestDXCAvailableModes(t *testing.T) {
	m := &Model{}
	modes := m.dxcAvailableModes()
	if len(modes) != 3 {
		t.Errorf("dxcAvailableModes should return 3 modes, got %d", len(modes))
	}
	expected := []string{"CW", "DIGI", "PHONE"}
	for i, want := range expected {
		if modes[i] != want {
			t.Errorf("modes[%d] = %q, want %q", i, modes[i], want)
		}
	}
}
