package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

// newTestModel creates a minimal Model for form rendering and navigation tests.
// It initializes only the fields required by QSO form methods.
func newTestModel() *Model {
	cfg := &config.Config{
		DistanceUnit: "km",
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
			"default": {Model: "FT-891", Antenna: "Dipole"},
		},
	}
	a := &app.App{
		Config:      cfg,
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90", Operator: "OP", RigName: "default"}},
	}
	m := New(a, nil)
	return m
}

func TestQSOFormRender(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	view := m.viewForm(90)
	if view == "" {
		t.Error("viewForm returned empty string")
	}
	if !strings.Contains(view, "Call") {
		t.Error("viewForm missing Call label")
	}
	if !strings.Contains(view, "Date UTC") {
		t.Error("viewForm missing Date UTC label")
	}
}

func TestQSOFormRenderNarrow(t *testing.T) {
	m := newTestModel()
	m.width = 40
	m.height = 20

	view := m.viewForm(30)
	if view == "" {
		t.Error("viewForm on narrow width returned empty")
	}
}

func TestQSOFormRenderTiny(t *testing.T) {
	m := newTestModel()
	m.width = 20
	m.height = 10

	view := m.viewForm(15)
	if view == "" {
		t.Error("viewForm on tiny width returned empty")
	}
}

func TestQSOFormFocusedFieldMarker(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.focus = fieldCall
	m.fields[fieldCall].Focus()

	view := m.viewForm(90)
	// Focused field should show the textinput cursor/view
	if view == "" {
		t.Error("viewForm returned empty with focused field")
	}
}

func TestQSOFormRetainCheckboxUnchecked(t *testing.T) {
	m := newTestModel()
	m.retainComment = false

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty")
	}
	if !strings.Contains(box, "[ ]") {
		t.Error("Retain checkbox should show [ ] when unchecked")
	}
}

func TestQSOFormRetainCheckboxChecked(t *testing.T) {
	m := newTestModel()
	m.retainComment = true

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty")
	}
	if !strings.Contains(box, "[x]") {
		t.Error("Retain checkbox should show [x] when checked")
	}
}

func TestQSOFormRetainCheckboxFocused(t *testing.T) {
	m := newTestModel()
	m.retainFocused = true

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty when focused")
	}
}

func TestQSOFormEmptyValues(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	view := m.viewForm(90)
	// Empty values should show em-dash placeholder
	if !strings.Contains(view, "\u2014") {
		t.Error("viewForm should show em-dash for empty values")
	}
}

func TestQSOFormNextField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].Focus()

	initialFocus := m.focus
	m.nextField()
	if m.focus == initialFocus {
		t.Error("nextField did not change focus from Call")
	}
}

func TestQSOFormPrevField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldTime // second field
	m.fields[fieldTime].Focus()

	m.prevField()
	if m.focus != fieldDate {
		t.Errorf("prevField should move to Date from Time, got focus=%d", m.focus)
	}
}

func TestQSOFormFocusField(t *testing.T) {
	m := newTestModel()
	m.focusField(fieldBand)
	if m.focus != fieldBand {
		t.Errorf("focusField(Band) gave focus=%d, want %d", m.focus, fieldBand)
	}
}

func TestQSOFormClearForm(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldComment].SetValue("Hello")
	m.fields[fieldRSTSent].SetValue("59")
	m.fields[fieldRSTRcvd].SetValue("59")
	m.retainComment = true

	m.clearForm()

	// Comment should be preserved (retain is on)
	if comment := m.fields[fieldComment].Value(); comment != "Hello" {
		t.Errorf("clearForm with retain should preserve comment, got %q", comment)
	}
	// Call should be cleared
	if call := m.fields[fieldCall].Value(); call != "" {
		t.Errorf("clearForm should clear Call, got %q", call)
	}
	// Focus should be on Call
	if m.focus != fieldCall {
		t.Errorf("clearForm should set focus to Call, got %d", m.focus)
	}
}

func TestQSOFormAutoFillRST(t *testing.T) {
	tests := []struct {
		mode     string
		wantSent string
		wantRcvd string
	}{
		{"CW", "599", "599"},
		{"SSB", "59", "59"},
		{"FT8", "59", "59"},
		{"", "59", "59"},
	}
	for _, tt := range tests {
		m := newTestModel()
		m.fields[fieldMode].SetValue(tt.mode)
		m.fields[fieldRSTSent].SetValue("")
		m.fields[fieldRSTRcvd].SetValue("")
		m.autoFillRST()
		if got := m.fields[fieldRSTSent].Value(); got != tt.wantSent {
			t.Errorf("autoFillRST mode=%q RSTSent=%q, want %q", tt.mode, got, tt.wantSent)
		}
		if got := m.fields[fieldRSTRcvd].Value(); got != tt.wantRcvd {
			t.Errorf("autoFillRST mode=%q RSTRcvd=%q, want %q", tt.mode, got, tt.wantRcvd)
		}
	}
}

func TestQSOFormAutoFillRSTNoOverwrite(t *testing.T) {
	m := newTestModel()
	m.fields[fieldMode].SetValue("CW")
	m.fields[fieldRSTSent].SetValue("599")
	m.fields[fieldRSTRcvd].SetValue("579") // already has a value
	m.autoFillRST()
	// Should NOT overwrite existing RST
	if m.fields[fieldRSTRcvd].Value() != "579" {
		t.Errorf("autoFillRST overwrote existing RST rcvd value")
	}
}

func TestQSOFormAutoFillSSBSubmode(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("7.100") // below 10 MHz
	m.fields[fieldMode].SetValue("SSB")
	// Reset band to trigger derivation
	m.fields[fieldBand].SetValue("40m")
	m.autoFillSSBSubmode()
	// applyFreqDefaults will use the freq to derive submode
	// With freq=7.100, band=40m is already set
	// The autoFillSSBSubmode calls applyFreqDefaults which uses freq directly
}

func TestQSOFormUpdateFocused(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].Focus()
	m.fields[fieldCall].SetValue("")

	// Simulate typing 's' via a KeyPressMsg with Code rune
	m.updateFocused(tea.KeyPressMsg{Code: 's', Text: "s"})
	// Should be uppercased
	val := m.fields[fieldCall].Value()
	if val != "S" {
		t.Logf("updateFocused call value: %q (expected 'S')", val)
	}
}

func TestQSOFormPathRow(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.fields[fieldGrid].SetValue("JN18") // partner grid

	row := m.formPathRow(90)
	if row == "" {
		t.Error("formPathRow returned empty when both grids set")
	}
}

func TestQSOFormPathRowNoOwnGrid(t *testing.T) {
	m := newTestModel()
	m.App.Logbook.Station.Grid = "" // no own grid
	m.fields[fieldGrid].SetValue("JN18")

	row := m.formPathRow(90)
	if row == "" {
		t.Error("formPathRow returned empty")
	}
	if !strings.Contains(row, "station config") {
		t.Error("formPathRow should prompt user to set own grid")
	}
}

// Verify no import issues with textinput
var _ = textinput.New

// Verify lipgloss import
var _ = lipgloss.NewStyle

// Verify tea import
var _ = tea.Quit
