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

func TestRegionBands(t *testing.T) {
	// Region 0 (unknown) should match Region 2 (widest defaults).
	r0 := regionBands(0)
	r2 := regionBands(2)
	if len(r0) != len(r2) {
		t.Fatalf("regionBands(0) len = %d, regionBands(2) len = %d — should match", len(r0), len(r2))
	}

	// Region 1: 40m is 7.0–7.2 (narrower)
	r1 := regionBands(1)
	var r1_40m bandRange
	for _, r := range r1 {
		if r.name == "40m" {
			r1_40m = r
			break
		}
	}
	if r1_40m.low != 7.0 || r1_40m.high != 7.2 {
		t.Errorf("Region 1 40m = %.1f–%.1f, want 7.0–7.2", r1_40m.low, r1_40m.high)
	}

	// Region 3: 40m is 7.0–7.2, 80m is 3.5–3.9
	r3 := regionBands(3)
	for _, r := range r3 {
		switch r.name {
		case "40m":
			if r.low != 7.0 || r.high != 7.2 {
				t.Errorf("Region 3 40m = %.1f–%.1f, want 7.0–7.2", r.low, r.high)
			}
		case "80m":
			if r.low != 3.5 || r.high != 3.9 {
				t.Errorf("Region 3 80m = %.1f–%.1f, want 3.5–3.9", r.low, r.high)
			}
		}
	}

	// Default (Region 2): 40m is 7.0–7.3, 80m is 3.5–4.0
	for _, r := range r2 {
		switch r.name {
		case "40m":
			if r.low != 7.0 || r.high != 7.3 {
				t.Errorf("Region 2 40m = %.1f–%.1f, want 7.0–7.3", r.low, r.high)
			}
		case "80m":
			if r.low != 3.5 || r.high != 4.0 {
				t.Errorf("Region 2 80m = %.1f–%.1f, want 3.5–4.0", r.low, r.high)
			}
		}
	}
}

func TestIsInHamBand(t *testing.T) {
	tests := []struct {
		freqMHz float64
		region  int
		want    bool
	}{
		// 40m band — Region 1 (EU): 7.0–7.2
		{freqMHz: 7.150, region: 1, want: true},
		{freqMHz: 7.200, region: 1, want: true},
		{freqMHz: 7.210, region: 1, want: false}, // just outside R1 40m
		{freqMHz: 7.250, region: 1, want: false},
		{freqMHz: 7.300, region: 1, want: false},

		// 40m band — Region 2 (Americas): 7.0–7.3
		{freqMHz: 7.150, region: 2, want: true},
		{freqMHz: 7.250, region: 2, want: true},
		{freqMHz: 7.300, region: 2, want: true},
		{freqMHz: 7.310, region: 2, want: false},

		// 40m band — Region 3 (Asia/Pacific): 7.0–7.2
		{freqMHz: 7.150, region: 3, want: true},
		{freqMHz: 7.250, region: 3, want: false},

		// 40m band — Region 0 (unknown, should use Region 2 widest): 7.0–7.3
		{freqMHz: 7.150, region: 0, want: true},
		{freqMHz: 7.250, region: 0, want: true},
		{freqMHz: 7.300, region: 0, want: true},
		{freqMHz: 7.310, region: 0, want: false},

		// 80m band — Region 1 (EU): 3.5–3.8
		{freqMHz: 3.600, region: 1, want: true},
		{freqMHz: 3.800, region: 1, want: true},
		{freqMHz: 3.810, region: 1, want: false},

		// 80m band — Region 2 (Americas): 3.5–4.0
		{freqMHz: 3.900, region: 2, want: true},
		{freqMHz: 4.000, region: 2, want: true},

		// 80m band — Region 3 (Asia/Pacific): 3.5–3.9
		{freqMHz: 3.850, region: 3, want: true},
		{freqMHz: 3.900, region: 3, want: true},
		{freqMHz: 3.910, region: 3, want: false},

		// 160m band — Region 1 (EU): 1.81–2.0
		{freqMHz: 1.810, region: 1, want: true},
		{freqMHz: 1.800, region: 1, want: false},

		// 160m band — Region 2: 1.8–2.0
		{freqMHz: 1.800, region: 2, want: true},

		// Common bands (same across all regions)
		{freqMHz: 14.200, region: 1, want: true}, // 20m
		{freqMHz: 14.200, region: 2, want: true},
		{freqMHz: 14.200, region: 3, want: true},
		{freqMHz: 14.350, region: 1, want: true},  // 20m upper edge
		{freqMHz: 14.360, region: 1, want: false}, // just outside 20m

		// Out of all amateur bands
		{freqMHz: 9.500, region: 1, want: false},  // broadcast AM
		{freqMHz: 27.500, region: 1, want: false}, // CB
		{freqMHz: 0.0, region: 1, want: false},    // zero
	}

	for _, tt := range tests {
		got := IsInHamBand(tt.freqMHz, tt.region)
		if got != tt.want {
			t.Errorf("IsInHamBand(%.3f, region %d) = %v, want %v", tt.freqMHz, tt.region, got, tt.want)
		}
	}
}
