package qso

import (
	"testing"
)

func TestValidateImportRecord_ValidSSB(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Freq = 14.250
	q.Band = "20m"
	q.Mode = "SSB"
	q.QSODate = "20260618"
	q.TimeOn = "120000"
	q.RSTSent = "59"
	q.RSTRcvd = "59"
	q.GridSquare = "JO90"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("should accept valid SSB QSO: %v", err)
	}
	if q.Band != "20m" {
		t.Errorf("band should be 20m, got %q", q.Band)
	}
	if q.GridSquare != "JO90" {
		t.Errorf("grid should be JO90, got %q", q.GridSquare)
	}
}

func TestValidateImportRecord_ValidCW(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9BBB"
	q.Freq = 7.015
	q.Band = "40m"
	q.Mode = "CW"
	q.QSODate = "20260618"
	q.TimeOn = "130000"
	q.RSTSent = "599"
	q.RSTRcvd = "579"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("should accept valid CW QSO: %v", err)
	}
}

func TestValidateImportRecord_ValidFT8(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9CCC"
	q.Freq = 21.074
	q.Band = "15m"
	q.Mode = "FT8"
	q.Submode = ""
	q.QSODate = "20260618"
	q.TimeOn = "140000"
	q.RSTSent = "-05"
	q.RSTRcvd = "+02"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("should accept valid FT8 QSO: %v", err)
	}
	if q.Mode != "MFSK" {
		t.Errorf("FT8 should be normalized to MFSK, got %q", q.Mode)
	}
	if q.Submode != "FT8" {
		t.Errorf("submode should be FT8, got %q", q.Submode)
	}
}

func TestValidateImportRecord_DeriveBandFromFreq(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Freq = 14.250
	q.Band = "" // empty, should derive from freq
	q.Mode = "SSB"
	q.QSODate = "20260618"
	q.TimeOn = "120000"
	q.RSTSent = "59"
	q.RSTRcvd = "59"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("should derive band from frequency: %v", err)
	}
	if q.Band != "20m" {
		t.Errorf("should derive 20m from 14.250 MHz, got %q", q.Band)
	}
}

func TestValidateImportRecord_EmptyCallsign(t *testing.T) {
	q := NewQSO()
	q.Call = ""
	q.Band = "20m"
	q.Mode = "SSB"

	err := ValidateImportRecord(q)
	if err == nil {
		t.Error("should reject empty callsign")
	}
}

func TestValidateImportRecord_InvalidCallsign(t *testing.T) {
	tests := []struct {
		name string
		call string
	}{
		{"too short", "A"},
		{"too long", "SP9MOASP9MOASP9MOASP9MOA"},
		{"special chars", "SP9*MOA"},
		{"leading slash", "/SP9MOA"},
		{"double slash", "SP9//MOA"},
		{"no digit", "SPMOA"},
		{"no letter", "12345"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQSO()
			q.Call = tc.call
			q.Band = "20m"
			q.Mode = "SSB"
			if err := ValidateImportRecord(q); err == nil {
				t.Errorf("should reject invalid call %q", tc.call)
			}
		})
	}
}

func TestValidateImportRecord_InvalidGridCleared(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "SSB"
	q.GridSquare = "XXXX" // invalid

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("invalid grid should be non-fatal, but got error: %v", err)
	}
	if q.GridSquare != "" {
		t.Errorf("invalid grid should be cleared, got %q", q.GridSquare)
	}
}

func TestValidateImportRecord_EmptyGridPreserved(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "SSB"
	q.GridSquare = "" // empty is fine

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("empty grid should be fine: %v", err)
	}
	if q.GridSquare != "" {
		t.Errorf("grid should remain empty, got %q", q.GridSquare)
	}
}

func TestValidateImportRecord_ValidGridPreserved(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "SSB"
	q.GridSquare = "JO90"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("valid grid JO90 should pass: %v", err)
	}
	if q.GridSquare != "JO90" {
		t.Errorf("grid should remain JO90, got %q", q.GridSquare)
	}
}

func TestValidateImportRecord_2CharGridAccepted(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "SSB"
	q.GridSquare = "JO" // 2-char grid is valid for WSJT-X compatibility

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("2-char grid JO should pass: %v", err)
	}
	if q.GridSquare != "JO" {
		t.Errorf("grid should remain JO, got %q", q.GridSquare)
	}
}

func TestValidateImportRecord_EmptyMode(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = ""

	if err := ValidateImportRecord(q); err == nil {
		t.Error("should reject empty mode")
	}
}

func TestValidateImportRecord_UnknownMode(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "XYZ123" // not a known mode

	if err := ValidateImportRecord(q); err == nil {
		t.Error("should reject unknown mode")
	}
}

func TestValidateImportRecord_InvalidSubmodeCleared(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "20m"
	q.Mode = "SSB"
	q.Submode = "BOGUS" // invalid submode for SSB

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("invalid submode should be non-fatal: %v", err)
	}
	if q.Submode != "" {
		t.Errorf("invalid submode should be cleared, got %q", q.Submode)
	}
}

func TestValidateImportRecord_EmptyBandAndFreq(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = ""
	q.Freq = 0
	q.Mode = "SSB"

	if err := ValidateImportRecord(q); err == nil {
		t.Error("should reject empty band and zero freq")
	}
}

func TestValidateImportRecord_EmptyBandFreqOutOfRange(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = ""
	q.Freq = 999.0 // not in any amateur band
	q.Mode = "SSB"

	if err := ValidateImportRecord(q); err == nil {
		t.Error("should reject when frequency does not match any band")
	}
}

func TestValidateImportRecord_UnknownBand(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "999m" // not a valid band
	q.Mode = "SSB"

	if err := ValidateImportRecord(q); err == nil {
		t.Error("should reject unknown band")
	}
}

func TestValidateImportRecord_NormalizeModeWithSubmode(t *testing.T) {
	q := NewQSO()
	q.Call = "SP9MOA"
	q.Band = "2m"
	q.Freq = 145.500
	q.Mode = "FT4"
	q.Submode = ""
	q.QSODate = "20260618"
	q.TimeOn = "120000"
	q.RSTSent = "59"
	q.RSTRcvd = "59"

	if err := ValidateImportRecord(q); err != nil {
		t.Errorf("FT4 should normalize to MFSK/FT4: %v", err)
	}
	if q.Mode != "MFSK" {
		t.Errorf("FT4 mode should become MFSK, got %q", q.Mode)
	}
	if q.Submode != "FT4" {
		t.Errorf("submode should be FT4, got %q", q.Submode)
	}
}
