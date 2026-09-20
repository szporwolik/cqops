package tui

import (
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/ref"
)

// newRefPrecedenceModel creates a model with a REF database containing one
// SOTA summit with a known grid.
func newRefPrecedenceModel(t *testing.T) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	db, err := ref.Open(filepath.Join(t.TempDir(), "ref.db"))
	if err != nil {
		t.Fatalf("ref.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.UnderlyingDB().Exec(
		`INSERT INTO refs (ref_type, ref, name, grid, search) VALUES ('SOTA','SP/TA-001','Tatra Test','JO90ab','sota sp/ta-001 tatra test')`)
	if err != nil {
		t.Fatalf("seed refs: %v", err)
	}
	m.App.RefDB = db
	return m
}

func callbookMsg(grid, qth string) callbookResultMsg {
	return callbookResultMsg{
		Call: "SP9MOA",
		Data: &callbook.Result{
			Callsign: "SP9MOA",
			Name:     "John",
			Grid:     grid,
			QTH:      qth,
			Country:  "Poland",
		},
	}
}

// Manual grid must survive a REF field exit.
func TestManualGridBeatsRefGrid(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldGrid].SetValue("JN99aa")
	m.gridSource = gridSourceManual // state produced by typing in the form

	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	if got := m.fields[fieldGrid].Value(); got != "JN99aa" {
		t.Errorf("manual grid overwritten by REF, got %q", got)
	}
	if m.gridSource != gridSourceManual {
		t.Errorf("grid source = %q, want manual", m.gridSource)
	}
	// QTH is not manual — REF may fill it.
	if got := m.fields[fieldQTH].Value(); got != "Tatra Test" {
		t.Errorf("QTH = %q, want REF-derived value", got)
	}
}

// Manual QTH must survive a REF field exit.
func TestManualQTHBeatsRefQTH(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldQTH].SetValue("My shack")
	m.qthSource = gridSourceManual

	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	if got := m.fields[fieldQTH].Value(); got != "My shack" {
		t.Errorf("manual QTH overwritten by REF, got %q", got)
	}
	// Grid is not manual — REF may fill it.
	if got := m.fields[fieldGrid].Value(); got != "JO90ab" {
		t.Errorf("grid = %q, want REF-derived value", got)
	}
}

// Manual grid and QTH must survive an async callbook result.
func TestManualValuesBeatCallbook(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("JN99aa")
	m.gridSource = gridSourceManual
	m.fields[fieldQTH].SetValue("My shack")
	m.qthSource = gridSourceManual

	m.fillCallbookData(callbookMsg("JO90", "Krakow"))

	if got := m.fields[fieldGrid].Value(); got != "JN99aa" {
		t.Errorf("manual grid overwritten by callbook, got %q", got)
	}
	if got := m.fields[fieldQTH].Value(); got != "My shack" {
		t.Errorf("manual QTH overwritten by callbook, got %q", got)
	}
}

// REF data filled first must survive a later callbook result.
func TestRefBeatsLateCallbook(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	m.fillCallbookData(callbookMsg("JO90", "Krakow"))

	if got := m.fields[fieldGrid].Value(); got != "JO90ab" {
		t.Errorf("REF grid overwritten by callbook, got %q", got)
	}
	if got := m.fields[fieldQTH].Value(); got != "Tatra Test" {
		t.Errorf("REF QTH overwritten by callbook, got %q", got)
	}
}

// Callbook data filled first is replaced when a REF is added.
func TestRefBeatsEarlierCallbook(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fillCallbookData(callbookMsg("JO90", "Krakow"))

	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	if got := m.fields[fieldGrid].Value(); got != "JO90ab" {
		t.Errorf("grid = %q, want REF grid", got)
	}
	if got := m.fields[fieldQTH].Value(); got != "Tatra Test" {
		t.Errorf("QTH = %q, want REF QTH", got)
	}
}

// Typing in the grid field marks the source as manual.
func TestTypingMarksManualSource(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.focus = fieldGrid
	m.fields[fieldGrid].Focus()
	m.fields[fieldGrid].SetValue("")

	for _, k := range "jn99aa" {
		m.updateFocused(tea.KeyPressMsg{Code: k, Text: string(k)})
	}
	if m.gridSource != gridSourceManual {
		t.Errorf("gridSource = %q after typing, want manual", m.gridSource)
	}

	m.focus = fieldQTH
	m.fields[fieldQTH].Focus()
	m.fields[fieldQTH].SetValue("")
	m.updateFocused(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if m.qthSource != gridSourceManual {
		t.Errorf("qthSource = %q after typing, want manual", m.qthSource)
	}
}

// Tabbing through a REF-filled grid must not fake a manual entry.
func TestTabThroughGridKeepsRefSource(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	m.focus = fieldGrid
	m.onFieldExit()

	if m.gridSource != gridSourceSOTA {
		t.Errorf("gridSource = %q after tab-through, want SOTA", m.gridSource)
	}
	if got := m.fields[fieldGrid].Value(); got != "JO90ab" {
		t.Errorf("grid = %q after tab-through, want JO90ab", got)
	}
}

// Removing the last reference clears the ref-derived grid and QTH.
func TestClearingRefsClearsRefDerivedValues(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	m.fields[fieldSOTA].SetValue("")
	m.applyRefGridAndQTH()

	if got := m.fields[fieldGrid].Value(); got != "" {
		t.Errorf("grid = %q after clearing refs, want empty", got)
	}
	if got := m.fields[fieldQTH].Value(); got != "" {
		t.Errorf("QTH = %q after clearing refs, want empty", got)
	}
	if m.gridSource != gridSourceNone || m.qthSource != gridSourceNone {
		t.Errorf("sources not reset: grid=%q qth=%q", m.gridSource, m.qthSource)
	}
}

// A live WSJT-X beacon grid must not be overwritten by REF data.
func TestWSJTXGridProtectedFromRef(t *testing.T) {
	m := newRefPrecedenceModel(t)
	m.fields[fieldGrid].SetValue("JN54ks")
	m.gridSource = gridSourceWSJTX

	m.fields[fieldSOTA].SetValue("SP/TA-001")
	m.applyRefGridAndQTH()

	if got := m.fields[fieldGrid].Value(); got != "JN54ks" {
		t.Errorf("WSJT-X grid overwritten by REF, got %q", got)
	}
	if m.gridSource != gridSourceWSJTX {
		t.Errorf("gridSource = %q, want WSJT-X", m.gridSource)
	}
}
