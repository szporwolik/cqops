package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

func TestOperatorFormValidate_EmptyCall(t *testing.T) {
	f := NewOperatorForm()
	if msg := f.Validate(); msg == "" {
		t.Error("Validate should fail on empty callsign")
	}
}

func TestOperatorFormValidate_NonEmpty(t *testing.T) {
	f := NewOperatorForm()
	f.Callsign.SetValue("SP9XXX")
	if msg := f.Validate(); msg != "" {
		t.Errorf("Validate should pass on non-empty callsign: %s", msg)
	}
}

func TestOperatorFormValidateCall_StandardCall(t *testing.T) {
	f := NewOperatorForm()
	f.Callsign.SetValue("SP9XXX")
	if msg := f.ValidateCall(); msg != "" {
		t.Errorf("ValidateCall should pass on standard call: %s", msg)
	}
}

func TestOperatorFormValidateCall_NoDigit(t *testing.T) {
	f := NewOperatorForm()
	f.Callsign.SetValue("DSF")
	msg := f.ValidateCall()
	if msg == "" {
		t.Error("ValidateCall should warn on callsign without digit")
	}
	if !strings.Contains(msg, "doesn't look like") {
		t.Errorf("ValidateCall warning should mention non-standard callsign, got: %s", msg)
	}
}

func TestOperatorFormValidateCall_Empty(t *testing.T) {
	f := NewOperatorForm()
	if msg := f.ValidateCall(); msg != "" {
		t.Error("ValidateCall should pass on empty callsign (separate from Validate)")
	}
}

func TestOperatorFormSetOperator(t *testing.T) {
	f := NewOperatorForm()
	op := &config.Operator{Callsign: "SP9EGL", Name: "Egon"}
	f.SetOperator(op)
	call, name := f.Values()
	if call != "SP9EGL" {
		t.Errorf("callsign = %q; want SP9EGL", call)
	}
	if name != "Egon" {
		t.Errorf("name = %q; want Egon", name)
	}
}

func TestOperatorFormFocus(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	if !f.Callsign.Focused() {
		t.Error("Callsign should be focused after Focus()")
	}
	if f.Name.Focused() {
		t.Error("Name should not be focused after Focus()")
	}
}

func TestOperatorFormHandleKey_Tab(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	cmd := f.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd != nil {
		t.Error("Tab should not return a command")
	}
	if f.Callsign.Focused() {
		t.Error("Callsign should lose focus after Tab")
	}
	if !f.Name.Focused() {
		t.Error("Name should gain focus after Tab")
	}
}

func TestOperatorFormHandleKey_Down(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	cmd := f.HandleKey(tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		t.Error("Down should not return a command")
	}
	if f.Callsign.Focused() {
		t.Error("Callsign should lose focus after Down")
	}
	if !f.Name.Focused() {
		t.Error("Name should gain focus after Down")
	}
}

func TestOperatorFormHandleKey_ShiftTab(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	f.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab}) // move to Name
	f.HandleKey(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if !f.Callsign.Focused() {
		t.Error("Callsign should be focused after Shift+Tab from Name")
	}
}

func TestOperatorFormHandleKey_CtrlS(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	// Simulate Ctrl+S key: String() should match "ctrl+s" or "\x13".
	cmd := f.HandleKey(tea.KeyPressMsg{Text: "\x13"})
	if cmd == nil {
		// Try alternative construction.
		cmd = f.HandleKey(tea.KeyPressMsg{Code: 19, Mod: tea.ModCtrl})
	}
	if cmd == nil {
		t.Error("Ctrl+S should return a save command")
	}
}

func TestOperatorFormBlurAll(t *testing.T) {
	f := NewOperatorForm()
	f.Focus()
	f.BlurAll()
	if f.Callsign.Focused() || f.Name.Focused() {
		t.Error("No field should be focused after BlurAll()")
	}
}

func TestOperatorFormView(t *testing.T) {
	f := NewOperatorForm()
	f.Callsign.SetValue("SP9XXX")
	v := f.View()
	if v == "" {
		t.Error("View should return non-empty string")
	}
}
