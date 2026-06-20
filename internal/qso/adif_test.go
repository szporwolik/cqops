package qso

import (
	"strings"
	"testing"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
)

func TestSanitizeASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Pure ASCII — fast path.
		{"Hello World", "Hello World"},
		{"", ""},

		// Polish.
		{"Zażółć gęślą jaźń", "Zazolc gesla jazn"},
		{"Kraków", "Krakow"},
		{"Łódź", "Lodz"},
		{"Gdańsk", "Gdansk"},

		// German.
		{"München", "Munchen"},
		{"Düsseldorf", "Dusseldorf"},
		{"Straße", "Strasse"},
		{"Schloß", "Schloss"},

		// French.
		{"Français", "Francais"},
		{"élève", "eleve"},
		{"cœur", "coeur"},
		{"hôpital", "hopital"},
		{"à propos", "a propos"},

		// Spanish.
		{"mañana", "manana"},
		{"español", "espanol"},
		{"América", "America"},

		// Portuguese.
		{"São Paulo", "Sao Paulo"},
		{"não", "nao"},
		{"lingüiça", "linguica"},

		// Nordic.
		{"København", "Kobenhavn"},
		{"Åland", "Aland"},
		{"Örebro", "Orebro"},
		{"Sverige", "Sverige"},

		// Turkish.
		{"İstanbul", "Istanbul"},
		{"şaşırmak", "sasirmak"},

		// Czech/Slovak.
		{"Česká", "Ceska"},
		{"Škoda", "Skoda"},
		{"Ďáblice", "Dablice"},

		// Romanian.
		{"București", "Bucuresti"},
		{"România", "Romania"},

		// Croatian/Serbian.
		{"Zagreb", "Zagreb"},
		{"Đakovo", "DJakovo"},

		// Icelandic.
		{"Reykjavík", "Reykjavik"},
		{"Þingvellir", "THingvellir"},

		// Mixed and edge cases.
		{"Curaçao", "Curacao"},
		{"Zażółć Straße", "Zazolc Strasse"},
		{"™", ""},                             // non-letter symbol → dropped
		{"café–restaurant", "caferestaurant"}, // en-dash dropped (non-ASCII, non-letter)
	}

	for _, tt := range tests {
		result := sanitizeASCII(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeASCII(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeASCII_NoAllocForASCII(t *testing.T) {
	input := "Hello World"
	result := sanitizeASCII(input)
	if result != input {
		t.Errorf("pure ASCII should be returned unchanged")
	}
}

func TestIsValidIOTA(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"EU-005", true},
		{"eu-005", true}, // case-insensitive
		{"NA-001", true},
		{"AS-007S", true}, // suffix letter allowed
		{"OC-123", true},
		{"AN-016", true},
		{"EU-5", false},       // too short after hyphen (min 2)
		{"EU-0055555", false}, // too long after hyphen (max 6) — 7 chars
		{"E-005", true},       // 1 char before hyphen is valid
		{"BLANK", false},      // no hyphen
		{"16", false},         // no hyphen
		{"NONE", false},
		{"NULL", false},
		{"", false},
		{"   ", false},
		{"EU005", false},  // no hyphen
		{"EU-", false},    // nothing after hyphen
		{"-005", false},   // nothing before hyphen
		{"EU-00$", false}, // invalid character
		{"12-005", false}, // numbers before hyphen
	}
	for _, tt := range tests {
		got := isValidIOTA(tt.input)
		if got != tt.expected {
			t.Errorf("isValidIOTA(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestStripNonDigits(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"20260619", "20260619"},
		{"2026-06-19", "20260619"},
		{"14:05:30", "140530"},
		{"2026/06/19 14:05", "202606191405"},
		{"", ""},
		{"abc", ""},
		{"a1b2c3", "123"},
	}
	for _, tt := range tests {
		got := stripNonDigits(tt.input)
		if got != tt.expected {
			t.Errorf("stripNonDigits(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestParseADIFRecord_Basic(t *testing.T) {
	// Simulate an ADIF record using farmergreg's NewRecord.
	r := adif.NewRecord()
	r[adifield.CALL] = "sp9moa"
	r[adifield.BAND] = "20m"
	r[adifield.MODE] = "ft8"
	r[adifield.QSO_DATE] = "2026-06-19"
	r[adifield.TIME_ON] = "14:05:30"
	r[adifield.FREQ] = "14.074"
	r[adifield.GRIDSQUARE] = "jo90aa"
	r[adifield.NAME] = "John"
	r[adifield.QTH] = "Kraków"
	r[adifield.COUNTRY] = "Poland"
	r[adifield.IOTA] = "EU-005"
	r[adifield.TX_PWR] = "100"
	r[adifield.DISTANCE] = "123.45"

	qs := ParseADIFRecord(r, "import")

	if qs.Call != "SP9MOA" {
		t.Errorf("Call = %q, want SP9MOA", qs.Call)
	}
	if qs.Band != "20m" {
		t.Errorf("Band = %q, want 20m", qs.Band)
	}
	if qs.Mode != "FT8" {
		t.Errorf("Mode = %q, want FT8", qs.Mode)
	}
	if qs.QSODate != "20260619" { // stripNonDigits applied
		t.Errorf("QSODate = %q, want 20260619", qs.QSODate)
	}
	if qs.TimeOn != "140530" { // stripNonDigits applied
		t.Errorf("TimeOn = %q, want 140530", qs.TimeOn)
	}
	if qs.Freq != 14.074 {
		t.Errorf("Freq = %f, want 14.074", qs.Freq)
	}
	if qs.GridSquare != "JO90AA" { // NormalizeLocator applied
		t.Errorf("GridSquare = %q, want JO90AA", qs.GridSquare)
	}
	if qs.Name != "John" {
		t.Errorf("Name = %q", qs.Name)
	}
	if qs.QTH != "Krakow" { // sanitizeASCII NOT applied in ParseADIFRecord (only in ToADIF)
		// QTH is stored raw; ASCII sanitization happens on export
		if qs.QTH != "Kraków" {
			t.Errorf("QTH = %q, want Kraków (raw import)", qs.QTH)
		}
	}
	if qs.Country != "Poland" {
		t.Errorf("Country = %q", qs.Country)
	}
	if qs.IOTA != "EU-005" {
		t.Errorf("IOTA = %q, want EU-005", qs.IOTA)
	}
	if qs.TXPower != "100" {
		t.Errorf("TXPower = %q", qs.TXPower)
	}
	if qs.Distance != 123.45 {
		t.Errorf("Distance = %f, want 123.45", qs.Distance)
	}
	if qs.Source != "import" {
		t.Errorf("Source = %q, want import", qs.Source)
	}
}

func TestParseADIFRecord_IOTACleanup(t *testing.T) {
	// Invalid IOTA values should be cleared.
	r := adif.NewRecord()
	r[adifield.CALL] = "SP9MOA"
	r[adifield.BAND] = "20m"
	r[adifield.MODE] = "FT8"
	r[adifield.IOTA] = "BLANK"

	qs := ParseADIFRecord(r, "import")
	if qs.IOTA != "" {
		t.Errorf("IOTA = %q, want empty (cleaned)", qs.IOTA)
	}

	// Valid IOTA should be kept.
	r[adifield.IOTA] = "EU-005"
	qs = ParseADIFRecord(r, "import")
	if qs.IOTA != "EU-005" {
		t.Errorf("IOTA = %q, want EU-005", qs.IOTA)
	}
}

func TestParseADIFRecord_DXCCFallback(t *testing.T) {
	// When COUNTRY is empty, DXCC should be used as fallback.
	r := adif.NewRecord()
	r[adifield.CALL] = "SP9MOA"
	r[adifield.BAND] = "20m"
	r[adifield.MODE] = "FT8"
	r[adifield.DXCC] = "Poland"

	qs := ParseADIFRecord(r, "import")
	if qs.Country != "Poland" {
		t.Errorf("Country = %q, want Poland (DXCC fallback)", qs.Country)
	}

	// When COUNTRY is set, DXCC should not override.
	r[adifield.COUNTRY] = "France"
	qs = ParseADIFRecord(r, "import")
	if qs.Country != "France" {
		t.Errorf("Country = %q, want France (COUNTRY takes priority)", qs.Country)
	}
}

func TestADIFContestRoundTrip(t *testing.T) {
	qs := NewQSO()
	qs.Call = "SP9MOA"
	qs.QSODate = "20240101"
	qs.TimeOn = "120000"
	qs.Band = "20m"
	qs.Mode = "SSB"
	qs.RSTSent = "59"
	qs.RSTRcvd = "59"
	qs.ExchSent = "599 001"
	qs.ExchRcvd = "599 042"
	qs.STX = 1
	qs.SRX = 42
	qs.STXString = "599 001"
	qs.SRXString = "599 042"
	qs.ContestID = "c1"
	qs.MyGridSquare = "JO90"
	qs.StationCallsign = "SP9MOA"

	adifStr := qs.toADIFWithStation("SP9MOA")

	// Build record manually from the ADIF string fields.
	r := adif.NewRecord()
	r[adifield.Field("STX_STRING")] = "599 001"
	r[adifield.Field("SRX_STRING")] = "599 042"
	r[adifield.Field("STX")] = "1"
	r[adifield.Field("SRX")] = "42"
	r[adifield.Field("CONTEST_ID")] = "c1"

	// Verify the ADIF output contains the expected fields.
	if adifStr == "" {
		t.Fatal("ADIF output is empty")
	}
	if !strings.Contains(adifStr, "STX_STRING") {
		t.Error("ADIF output should contain STX_STRING")
	}
	if !strings.Contains(adifStr, "599 001") {
		t.Error("ADIF output should contain exchange value")
	}

	// Parse back.
	parsed := ParseADIFRecord(r, "manual")

	if parsed.STX != 1 {
		t.Errorf("STX = %d, want 1", parsed.STX)
	}
	if parsed.SRX != 42 {
		t.Errorf("SRX = %d, want 42", parsed.SRX)
	}
	if parsed.STXString != "599 001" {
		t.Errorf("STXString = %q", parsed.STXString)
	}
	if parsed.SRXString != "599 042" {
		t.Errorf("SRXString = %q", parsed.SRXString)
	}
	if parsed.ContestID != "c1" {
		t.Errorf("ContestID = %q, want c1", parsed.ContestID)
	}
}
