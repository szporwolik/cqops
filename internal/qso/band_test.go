package qso

import "testing"

func TestIsValidBand(t *testing.T) {
	valid := []string{"160m", "80m", "40m", "30m", "20m", "17m", "15m", "12m", "10m", "6m", "2m", "70cm", "23cm", "submm", "160M", "20M"}
	for _, b := range valid {
		if !IsValidBand(b) {
			t.Errorf("IsValidBand(%q) = false, want true", b)
		}
	}
	invalid := []string{"11m", "BOGUS", "", "m", "20 m"}
	for _, b := range invalid {
		if IsValidBand(b) {
			t.Errorf("IsValidBand(%q) = true, want false", b)
		}
	}
}

func TestNormalizeBand(t *testing.T) {
	tests := map[string]string{
		"20m":   "20m",
		"20M":   "20m",
		"2M":    "2m",
		"70CM":  "70cm",
		"6mm":   "6mm",
		"6MM":   "6mm",
		"BOGUS": "BOGUS",
		"":      "",
	}
	for in, want := range tests {
		if got := NormalizeBand(in); got != want {
			t.Errorf("NormalizeBand(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDeriveBand(t *testing.T) {
	tests := map[float64]string{
		0.136:  "2190m",
		1.9:    "160m",
		3.7:    "80m",
		7.15:   "40m",
		14.2:   "20m",
		21.2:   "15m",
		28.5:   "10m",
		50.1:   "6m",
		146.0:  "2m",
		432.0:  "70cm",
		1296.0: "23cm",
		0.0:    "",
		999.0:  "",
	}
	for freq, want := range tests {
		if got := DeriveBand(freq); got != want {
			t.Errorf("DeriveBand(%f) = %q, want %q", freq, got, want)
		}
	}
}

func TestBandRange(t *testing.T) {
	low, high, ok := BandRange("20m")
	if !ok || low != 14.0 || high != 14.35 {
		t.Errorf("BandRange(20m) = (%f, %f, %v), want (14.0, 14.35, true)", low, high, ok)
	}
	_, _, ok = BandRange("BOGUS")
	if ok {
		t.Error("BandRange(BOGUS) = true, want false")
	}
}

func TestAllBands(t *testing.T) {
	all := AllBands()
	if len(all) != len(bandRanges) {
		t.Errorf("AllBands() len = %d, want %d", len(all), len(bandRanges))
	}
}
