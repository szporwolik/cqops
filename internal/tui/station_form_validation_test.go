package tui

import (
	"fmt"
	"strings"
	"testing"
)

// =============================================================================
// StationForm UI validation tests (Pass 20)
// =============================================================================

func newStationFormForTest() *StationForm {
	return NewStationForm("CALLSIGN", "OP", "GRID")
}

// =============================================================================
// ValidateField tests
// =============================================================================

func TestStationForm_ValidateField_ValidCallsign(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9MOA")
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Errorf("ValidateField(Callsign:) = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidateField_ValidPortableCall(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9MOA/P")
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Errorf("ValidateField(Callsign: portable) = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidateField_ValidWithSlash(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("K1ABC/SP9")
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Errorf("ValidateField(Callsign: with /) = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidateField_EmptyCallsign(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("")
	// Empty callsign passes ValidateField (non-empty check is in Validate()).
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Errorf("ValidateField(Callsign: empty) = %q, want \"\" (Validate checks emptiness)", hint)
	}
}

func TestStationForm_ValidateField_InvalidCallsign(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{"too short", "A"},
		{"special chars", "SP9*MOA"},
		{"double slash", "SP9//MOA"},
		{"no digit", "SPMOA"},
		{"no letter", "12345"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newStationFormForTest()
			f.Callsign.SetValue(tc.value)
			hint := f.ValidateField("Callsign:")
			if hint == "" {
				t.Errorf("ValidateField(Callsign: %q) = \"\", want non-empty hint", tc.value)
			}
			if !strings.Contains(strings.ToLower(hint), "callsign") {
				t.Errorf("hint %q should mention callsign", hint)
			}
		})
	}
}

func TestStationForm_ValidateField_ValidLocator4Char(t *testing.T) {
	f := newStationFormForTest()
	f.Locator.SetValue("JO90")
	if hint := f.ValidateField("Grid locator:"); hint != "" {
		t.Errorf("ValidateField(Grid locator: JO90) = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidateField_ValidLocator6Char(t *testing.T) {
	f := newStationFormForTest()
	f.Locator.SetValue("JO90AA")
	if hint := f.ValidateField("Grid locator:"); hint != "" {
		t.Errorf("ValidateField(Grid locator: JO90AA) = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidateField_EmptyLocator(t *testing.T) {
	f := newStationFormForTest()
	f.Locator.SetValue("")
	if hint := f.ValidateField("Grid locator:"); hint != "" {
		t.Errorf("ValidateField(Grid locator: empty) = %q, want \"\" (Validate checks emptiness)", hint)
	}
}

func TestStationForm_ValidateField_InvalidLocator(t *testing.T) {
	tests := []string{"XXXX", "ZZ99", "JO9", "JO9000", "JO9A"}
	for _, val := range tests {
		t.Run(val, func(t *testing.T) {
			f := newStationFormForTest()
			f.Locator.SetValue(val)
			hint := f.ValidateField("Grid locator:")
			if hint == "" {
				t.Errorf("ValidateField(Grid locator: %q) = \"\", want non-empty hint", val)
			}
			if !strings.Contains(strings.ToLower(hint), "locator") {
				t.Errorf("hint %q should mention locator", hint)
			}
		})
	}
}

func TestStationForm_ValidateField_UnknownLabel(t *testing.T) {
	f := newStationFormForTest()
	if hint := f.ValidateField("Bogus:"); hint != "" {
		t.Errorf("ValidateField(unknown label) = %q, want \"\"", hint)
	}
}

// =============================================================================
// Validate tests
// =============================================================================

func TestStationForm_Validate_Valid(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9MOA")
	f.Locator.SetValue("JO90")
	if err := f.Validate(); err != nil {
		t.Errorf("Validate() should pass for valid data: %v", err)
	}
}

func TestStationForm_Validate_EmptyCallsign(t *testing.T) {
	f := newStationFormForTest()
	f.Locator.SetValue("JO90")
	if err := f.Validate(); err == nil {
		t.Error("Validate() should fail when callsign is empty")
	}
}

func TestStationForm_Validate_InvalidCallsign(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9*MOA")
	f.Locator.SetValue("JO90")
	if err := f.Validate(); err == nil {
		t.Error("Validate() should fail when callsign is invalid")
	}
}

func TestStationForm_Validate_EmptyLocator(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9MOA")
	if err := f.Validate(); err == nil {
		t.Error("Validate() should fail when locator is empty")
	}
}

func TestStationForm_Validate_InvalidLocator(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9MOA")
	f.Locator.SetValue("XXXX")
	if err := f.Validate(); err == nil {
		t.Error("Validate() should fail when locator is invalid")
	}
}

// =============================================================================
// View includes validation hints
// =============================================================================

func TestStationForm_View_IncludesValidationHint(t *testing.T) {
	f := newStationFormForTest()
	f.width = 80
	f.Callsign.SetValue("SP9*MOA") // invalid
	f.Locator.SetValue("JO90")

	view := fmt.Sprint(f.View())
	// Inline hints are no longer rendered — validation is shown via toast on
	// field exit / save instead.
	if strings.Contains(view, "Invalid callsign") {
		t.Error("View should NOT contain inline 'Invalid callsign' — hints are shown via toast on field exit")
	}
}

func TestStationForm_View_NoValidationHintWhenValid(t *testing.T) {
	f := newStationFormForTest()
	f.width = 80
	f.Callsign.SetValue("SP9MOA")
	f.Locator.SetValue("JO90")

	view := fmt.Sprint(f.View())
	if strings.Contains(view, "Invalid callsign") {
		t.Error("View should NOT contain validation hint when data is valid")
	}
	if strings.Contains(view, "Invalid locator") {
		t.Error("View should NOT contain locator hint when valid")
	}
}

func TestStationForm_View_LocatorHint(t *testing.T) {
	f := newStationFormForTest()
	f.width = 80
	f.Callsign.SetValue("SP9MOA")
	f.Locator.SetValue("XXXX") // invalid

	view := fmt.Sprint(f.View())
	// Inline hints are no longer rendered — validation is shown via toast on
	// field exit / save instead.
	if strings.Contains(view, "Invalid locator") {
		t.Error("View should NOT contain inline 'Invalid locator' — hints are shown via toast on field exit")
	}
}

// =============================================================================
// Validation updates when field value changes
// =============================================================================

func TestStationForm_ValidationUpdatesFromInvalidToValid(t *testing.T) {
	f := newStationFormForTest()
	f.width = 80

	// Start invalid.
	f.Callsign.SetValue("SP9*MOA")
	if hint := f.ValidateField("Callsign:"); hint == "" {
		t.Fatal("should start invalid")
	}

	// Update to valid via HandleKey with backspace/deletes and new value.
	// Simulate clearing and retyping.
	f.Callsign.Focus()
	f.Callsign.SetValue("SP9MOA")
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Errorf("after update to valid: hint = %q, want \"\"", hint)
	}
}

func TestStationForm_ValidationUpdatesFromValidToInvalid(t *testing.T) {
	f := newStationFormForTest()

	f.Callsign.SetValue("SP9MOA")
	if hint := f.ValidateField("Callsign:"); hint != "" {
		t.Fatal("should start valid")
	}

	f.Callsign.SetValue("SP9*MOA")
	if hint := f.ValidateField("Callsign:"); hint == "" {
		t.Error("should become invalid after change")
	}
}

// =============================================================================
// Long/malformed input robustness
// =============================================================================

func TestStationForm_ValidateField_VeryLongInput(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue(strings.Repeat("A", 100))
	hint := f.ValidateField("Callsign:")
	if hint == "" {
		t.Error("very long callsign should be invalid")
	}
}

func TestStationForm_ValidateField_UnicodeInput(t *testing.T) {
	f := newStationFormForTest()
	f.Callsign.SetValue("SP9\u2603MOA") // snowman
	hint := f.ValidateField("Callsign:")
	if hint == "" {
		t.Error("unicode callsign should be invalid")
	}
}

func TestStationForm_ValidateField_LongLocator(t *testing.T) {
	f := newStationFormForTest()
	f.Locator.SetValue("XXXXXXXXXX") // 10 X's — X not in [A-R] for first two chars
	hint := f.ValidateField("Grid locator:")
	if hint == "" {
		t.Error("10-char invalid locator should return validation hint")
	}
}
