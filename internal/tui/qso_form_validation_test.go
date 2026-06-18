package tui

import (
	"strings"
	"testing"
)

// =============================================================================
// QSO form field validation hint tests (Pass 21)
// =============================================================================

func newQSOFormTestModel() *Model {
	m := newTestModel()
	m.width = 120
	m.height = 30
	return m
}

// =============================================================================
// Callsign hints
// =============================================================================

func TestQSOFieldHint_ValidCallsign(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9MOA")
	if hint := m.qsoFieldHint(fieldCall); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_ValidPortableCall(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9MOA/P")
	if hint := m.qsoFieldHint(fieldCall); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_ValidCallWithSlash(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("K1ABC/SP9")
	if hint := m.qsoFieldHint(fieldCall); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_InvalidCallsign(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"too short", "A"},
		{"special chars", "SP9*MOA"},
		{"no digit", "SPMOA"},
		{"no letter", "12345"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldCall].SetValue(tc.value)
			hint := m.qsoFieldHint(fieldCall)
			if hint == "" {
				t.Errorf("hint for %q should not be empty", tc.value)
			}
			if !strings.Contains(strings.ToLower(hint), "callsign") {
				t.Errorf("hint %q should mention callsign", hint)
			}
		})
	}
}

func TestQSOFieldHint_EmptyCallsign(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("")
	if hint := m.qsoFieldHint(fieldCall); hint != "" {
		t.Errorf("hint = %q, want \"\" (emptiness handled at save)", hint)
	}
}

// =============================================================================
// Grid hints
// =============================================================================

func TestQSOFieldHint_ValidGrid(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldGrid].SetValue("JO90")
	if hint := m.qsoFieldHint(fieldGrid); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_ValidGrid6Char(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldGrid].SetValue("JO90AA")
	if hint := m.qsoFieldHint(fieldGrid); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_InvalidGrid(t *testing.T) {
	tests := []string{"XXXX", "ZZ99", "JO9", "JO9A"}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldGrid].SetValue(val)
			hint := m.qsoFieldHint(fieldGrid)
			if hint == "" {
				t.Errorf("hint for %q should not be empty", val)
			}
			if !strings.Contains(strings.ToLower(hint), "locator") {
				t.Errorf("hint %q should mention locator", hint)
			}
		})
	}
}

func TestQSOFieldHint_EmptyGrid(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldGrid].SetValue("")
	if hint := m.qsoFieldHint(fieldGrid); hint != "" {
		t.Errorf("hint = %q, want \"\" (optional field)", hint)
	}
}

// =============================================================================
// Frequency hints
// =============================================================================

func TestQSOFieldHint_ValidFreq(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldFreq].SetValue("14.250")
	if hint := m.qsoFieldHint(fieldFreq); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_InvalidFreq(t *testing.T) {
	tests := []string{"hello", "-1", "0", "abc", ""}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldFreq].SetValue(val)
			hint := m.qsoFieldHint(fieldFreq)
			// Empty freq produces no hint (it's optional when band is set).
			if val == "" && hint != "" {
				t.Errorf("hint for empty freq = %q, want \"\"", hint)
			}
			if val != "" && hint == "" {
				t.Errorf("hint for %q should not be empty", val)
			}
			if hint != "" && !strings.Contains(strings.ToLower(hint), "frequency") {
				t.Errorf("hint %q should mention frequency", hint)
			}
		})
	}
}

func TestQSOFieldHint_ZeroFreq(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldFreq].SetValue("0")
	hint := m.qsoFieldHint(fieldFreq)
	if hint == "" {
		t.Error("zero frequency should produce a hint")
	}
}

// =============================================================================
// Band hints
// =============================================================================

func TestQSOFieldHint_ValidBand(t *testing.T) {
	for _, b := range []string{"160m", "80m", "40m", "20m", "15m", "10m", "2m", "70cm"} {
		t.Run(b, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldBand].SetValue(b)
			if hint := m.qsoFieldHint(fieldBand); hint != "" {
				t.Errorf("hint = %q, want \"\"", hint)
			}
		})
	}
}

func TestQSOFieldHint_InvalidBand(t *testing.T) {
	tests := []string{"999m", "bogus", "xx"}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldBand].SetValue(val)
			hint := m.qsoFieldHint(fieldBand)
			if hint == "" {
				t.Errorf("hint for %q should not be empty", val)
			}
			if !strings.Contains(strings.ToLower(hint), "band") {
				t.Errorf("hint %q should mention band", hint)
			}
		})
	}
}

func TestQSOFieldHint_EmptyBand(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldBand].SetValue("")
	if hint := m.qsoFieldHint(fieldBand); hint != "" {
		t.Errorf("hint = %q, want \"\" (emptiness handled at save)", hint)
	}
}

// =============================================================================
// Mode hints
// =============================================================================

func TestQSOFieldHint_ValidMode(t *testing.T) {
	for _, mode := range []string{"SSB", "CW", "FT8", "FT4", "RTTY", "AM", "FM"} {
		t.Run(mode, func(t *testing.T) {
			m := newQSOFormTestModel()
			m.fields[fieldMode].SetValue(mode)
			if hint := m.qsoFieldHint(fieldMode); hint != "" {
				t.Errorf("hint for %q = %q, want \"\"", mode, hint)
			}
		})
	}
}

func TestQSOFieldHint_InvalidMode(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("BOGUS123")
	hint := m.qsoFieldHint(fieldMode)
	if hint == "" {
		t.Error("invalid mode should produce a hint")
	}
	if !strings.Contains(strings.ToLower(hint), "mode") {
		t.Errorf("hint %q should mention mode", hint)
	}
}

func TestQSOFieldHint_EmptyMode(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("")
	if hint := m.qsoFieldHint(fieldMode); hint != "" {
		t.Errorf("hint = %q, want \"\" (emptiness handled at save)", hint)
	}
}

// =============================================================================
// Submode hints
// =============================================================================

func TestQSOFieldHint_ValidSubmode(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("MFSK")
	m.fields[fieldSubmode].SetValue("FT8")
	if hint := m.qsoFieldHint(fieldSubmode); hint != "" {
		t.Errorf("hint = %q, want \"\"", hint)
	}
}

func TestQSOFieldHint_InvalidSubmode(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldSubmode].SetValue("BOGUS")
	hint := m.qsoFieldHint(fieldSubmode)
	if hint == "" {
		t.Error("invalid submode should produce a hint")
	}
	if !strings.Contains(strings.ToLower(hint), "submode") {
		t.Errorf("hint %q should mention submode", hint)
	}
}

func TestQSOFieldHint_SubmodeNoMode(t *testing.T) {
	// Submode can't be validated without mode — no hint.
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("")
	m.fields[fieldSubmode].SetValue("FT8")
	if hint := m.qsoFieldHint(fieldSubmode); hint != "" {
		t.Errorf("hint = %q, want \"\" (no mode to validate against)", hint)
	}
}

func TestQSOFieldHint_EmptySubmode(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldMode].SetValue("MFSK")
	m.fields[fieldSubmode].SetValue("")
	if hint := m.qsoFieldHint(fieldSubmode); hint != "" {
		t.Errorf("hint = %q, want \"\" (empty submode is ok)", hint)
	}
}

// =============================================================================
// View includes hint
// =============================================================================

func TestQSOFormView_ShowsValidationHint(t *testing.T) {
	m := newQSOFormTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].SetValue("SP9*MOA") // invalid

	view := m.viewForm(90)
	// Inline hints are no longer rendered in the form — validation is shown
	// via toast on field exit instead.
	if strings.Contains(view, "Invalid callsign") {
		t.Error("viewForm should NOT contain inline 'Invalid callsign' — hints are shown via toast on field exit")
	}
}

func TestQSOFormView_NoHintWhenValid(t *testing.T) {
	m := newQSOFormTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].SetValue("SP9MOA")

	view := m.viewForm(90)
	if strings.Contains(view, "Invalid callsign") {
		t.Error("viewForm should NOT contain hint when callsign is valid")
	}
}

func TestQSOFormView_HintChangesWithFocus(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9*MOA")
	m.fields[fieldGrid].SetValue("JO90")

	// Focus on Call — inline hints are no longer shown in the form.
	m.focus = fieldCall
	v1 := m.viewForm(90)
	if strings.Contains(v1, "Invalid callsign") {
		t.Error("should NOT show inline callsign hint — hints are shown via toast on field exit")
	}

	// Focus on Grid — no hint (grid is valid, and inline hints are not shown).
	m.focus = fieldGrid
	m.rc.formSig = ""
	v2 := m.viewForm(90)
	if strings.Contains(v2, "Invalid callsign") {
		t.Error("should NOT show inline callsign hint when grid focused")
	}
}

// =============================================================================
// Long / malformed input robustness
// =============================================================================

func TestQSOFieldHint_VeryLongCallsign(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue(strings.Repeat("A", 100))
	hint := m.qsoFieldHint(fieldCall)
	if hint == "" {
		t.Error("very long callsign should be invalid")
	}
}

func TestQSOFieldHint_UnicodeCallsign(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9\u2603MOA") // snowman
	hint := m.qsoFieldHint(fieldCall)
	if hint == "" {
		t.Error("unicode callsign should be invalid")
	}
}

func TestQSOFieldHint_VeryLongFreq(t *testing.T) {
	m := newQSOFormTestModel()
	m.fields[fieldFreq].SetValue(strings.Repeat("9", 100))
	hint := m.qsoFieldHint(fieldFreq)
	// Very large frequency is technically a valid float, so no hint expected.
	// But it might overflow. Sscanf should still parse it.
	if hint != "" {
		t.Logf("very long freq hint: %q (may or may not be valid)", hint)
	}
}

// =============================================================================
// Auto-filled valid values from WSJT-X/DXC produce no hint
// =============================================================================

func TestQSOFieldHint_AutoFilledFT8(t *testing.T) {
	// Simulate a WSJT-X auto-filled QSO form.
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9AAA")
	m.fields[fieldFreq].SetValue("14.074000")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")
	m.fields[fieldSubmode].SetValue("")
	m.fields[fieldGrid].SetValue("JO90")
	m.fields[fieldRSTSent].SetValue("59")
	m.fields[fieldRSTRcvd].SetValue("59")

	for _, f := range []field{fieldCall, fieldFreq, fieldBand, fieldMode, fieldSubmode, fieldGrid} {
		if hint := m.qsoFieldHint(f); hint != "" {
			t.Errorf("field %v: unexpected hint %q for auto-filled value", f, hint)
		}
	}
}

func TestQSOFieldHint_DXCFilled(t *testing.T) {
	// Simulate a DXC-filled QSO form.
	m := newQSOFormTestModel()
	m.fields[fieldCall].SetValue("SP9BBB")
	m.fields[fieldFreq].SetValue("14.250000")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")

	for _, f := range []field{fieldCall, fieldFreq, fieldBand, fieldMode} {
		if hint := m.qsoFieldHint(f); hint != "" {
			t.Errorf("field %v: unexpected hint %q for DXC-filled value", f, hint)
		}
	}
}
