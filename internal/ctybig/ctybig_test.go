package ctybig

import (
	"strings"
	"testing"
)

func TestParseCSV_Basic(t *testing.T) {
	input := `K,United States,291,NA,5,8,37.60,91.87,5.0,AA AB AC AD K N W;
SP,Poland,269,EU,15,28,52.00,-20.00,-1.0,HF SN SO SP SQ SR 3Z;
`

	db, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	tests := []struct {
		call     string
		wantDXCC int
		wantName string
		wantCZ   int
		wantIZ   int
		wantCont string
		wantTZ   float64
	}{
		{"K1ABC", 291, "United States", 5, 8, "NA", 5.0},
		{"W1AW", 291, "United States", 5, 8, "NA", 5.0},
		{"AA5B", 291, "United States", 5, 8, "NA", 5.0},
		{"SP9MOA", 269, "Poland", 15, 28, "EU", -1.0},
		{"SQ8LSB", 269, "Poland", 15, 28, "EU", -1.0},
		{"HF0HQ", 269, "Poland", 15, 28, "EU", -1.0},
		{"3Z0X", 269, "Poland", 15, 28, "EU", -1.0},
	}

	for _, tt := range tests {
		e := db.Find(tt.call)
		if e == nil {
			t.Errorf("Find(%q) = nil", tt.call)
			continue
		}
		if e.DXCC != tt.wantDXCC {
			t.Errorf("Find(%q).DXCC = %d, want %d", tt.call, e.DXCC, tt.wantDXCC)
		}
		if e.Name != tt.wantName {
			t.Errorf("Find(%q).Name = %q, want %q", tt.call, e.Name, tt.wantName)
		}
		if e.CQZone != tt.wantCZ {
			t.Errorf("Find(%q).CQZone = %d, want %d", tt.call, e.CQZone, tt.wantCZ)
		}
		if e.ITUZone != tt.wantIZ {
			t.Errorf("Find(%q).ITUZone = %d, want %d", tt.call, e.ITUZone, tt.wantIZ)
		}
		if e.Continent != tt.wantCont {
			t.Errorf("Find(%q).Continent = %q, want %q", tt.call, e.Continent, tt.wantCont)
		}
		if e.TZOffset != tt.wantTZ {
			t.Errorf("Find(%q).TZOffset = %g, want %g", tt.call, e.TZOffset, tt.wantTZ)
		}
	}
}

func TestParseCSV_LongestPrefixMatch(t *testing.T) {
	input := `K,United States,291,NA,5,8,37.60,91.87,5.0,K W N;
KH6,Hawaii,110,OC,31,61,20.57,157.37,10.0,KH6 WH6 NH6 AH6;
KL,Alaska,6,NA,1,1,64.43,146.93,9.0,AL7 KL7 NL7 WL7;
`

	db, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	// KH6 (Hawaii) should match before K (United States).
	e := db.Find("KH6XYZ")
	if e == nil {
		t.Fatal("Find(KH6XYZ) = nil")
	}
	if e.DXCC != 110 {
		t.Errorf("KH6XYZ DXCC = %d, want 110 (Hawaii)", e.DXCC)
	}

	// KL7 (Alaska) should match before K.
	e = db.Find("KL7AA")
	if e == nil {
		t.Fatal("Find(KL7AA) = nil")
	}
	if e.DXCC != 6 {
		t.Errorf("KL7AA DXCC = %d, want 6 (Alaska)", e.DXCC)
	}

	// K5 (continental US) should match K.
	e = db.Find("K5ABC")
	if e == nil {
		t.Fatal("Find(K5ABC) = nil")
	}
	if e.DXCC != 291 {
		t.Errorf("K5ABC DXCC = %d, want 291 (USA)", e.DXCC)
	}
}

func TestParseCSV_ExactMatch(t *testing.T) {
	input := `3D2,Fiji,176,OC,32,56,-17.78,-177.92,-12.0,3D2;
3D2/c,Conway Reef,489,OC,32,56,-22.00,-175.00,-12.0,=3D20CR =3D2C;
`

	db, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	// 3D2C is exact match for Conway Reef.
	e := db.Find("3D2C")
	if e == nil {
		t.Fatal("Find(3D2C) = nil")
	}
	if e.DXCC != 489 {
		t.Errorf("3D2C DXCC = %d, want 489 (Conway Reef)", e.DXCC)
	}

	// 3D2XYZ is a Fiji prefix.
	e = db.Find("3D2XYZ")
	if e == nil {
		t.Fatal("Find(3D2XYZ) = nil")
	}
	if e.DXCC != 176 {
		t.Errorf("3D2XYZ DXCC = %d, want 176 (Fiji)", e.DXCC)
	}
}

func TestParseCSV_ZoneOverridesStripped(t *testing.T) {
	input := `K,United States,291,NA,5,8,37.60,91.87,5.0,AA0(4)[7] K;
SP,Poland,269,EU,15,28,52.00,-20.00,-1.0,SP;
`

	db, err := ParseCSV(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	// AA0 with zone overrides should still match as US.
	e := db.Find("AA0ZZZ")
	if e == nil {
		t.Fatal("Find(AA0ZZZ) = nil")
	}
	if e.DXCC != 291 {
		t.Errorf("AA0ZZZ DXCC = %d, want 291", e.DXCC)
	}
}

func TestParseCSV_EmptyInput(t *testing.T) {
	db, err := ParseCSV(strings.NewReader(""))
	if err != nil {
		t.Fatalf("ParseCSV empty: %v", err)
	}
	if e := db.Find("K1ABC"); e != nil {
		t.Errorf("expected nil for empty DB, got %v", e)
	}
}

func TestStripZoneOverrides(t *testing.T) {
	tests := []struct{ in, want string }{
		{"AA0(4)[7]", "AA0"},
		{"=N2NL/MM(7)", "=N2NL/MM"},
		{"SP9MOA", "SP9MOA"},
		{"K", "K"},
		{"=3D20CR", "=3D20CR"},
	}
	for _, tt := range tests {
		got := stripZoneOverrides(tt.in)
		if got != tt.want {
			t.Errorf("stripZoneOverrides(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
