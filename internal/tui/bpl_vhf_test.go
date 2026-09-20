package tui

import (
	"strings"
	"testing"
)

// TestBPLVHF_Formatting verifies the VHF tab renders aligned columns,
// full "CALL" tags, repeater wording, and no duplicated markers.
func TestBPLVHF_Formatting(t *testing.T) {
	lines := (&Model{}).viewBPLVHF(1)
	joined := strings.Join(lines, "\n")

	// Band headers: padded band name + range, with repeater wording.
	if !containsLine(lines, "2m   144.000\u2013146.000") {
		t.Errorf("2m band header missing or misaligned:\n%s", joined)
	}
	if !strings.Contains(joined, "repeater shift \u2212600 kHz") {
		t.Errorf("2m header should use full 'repeater' wording:\n%s", joined)
	}
	if !strings.Contains(joined, "repeater shifts 1.6/2.0/7.6 MHz") {
		t.Errorf("70cm header should use full 'repeater' wording:\n%s", joined)
	}

	// Calling tags are never truncated to "CAL".
	if strings.Contains(joined, " CAL ") {
		t.Errorf("calling tag truncated to CAL:\n%s", joined)
	}
	if !strings.Contains(joined, "70.200 MHz") || !strings.Contains(joined, "CALL") {
		t.Errorf("70.200 calling row missing:\n%s", joined)
	}

	// Range entries show the calling frequency once, as "CoA".
	if !strings.Contains(joined, "CW/weak signal; CW calling \u00b7 CoA 144.050") {
		t.Errorf("2m SSB range should carry CoA 144.050:\n%s", joined)
	}
	if !strings.Contains(joined, "CW/SSB/MGM \u00b7 CoA 432.200") {
		t.Errorf("70cm SSB range should carry CoA 432.200:\n%s", joined)
	}

	// No duplicated "country-specific" or doubled CoA markers.
	if strings.Count(joined, "country-specific) (country-specific)") != 0 {
		t.Errorf("duplicated country-specific marker:\n%s", joined)
	}
	if strings.Count(joined, "CoA \u00b7 CoA") != 0 {
		t.Errorf("duplicated CoA marker:\n%s", joined)
	}

	// Sections are separated by blank lines (4m, 6m, 2m, 70cm).
	wantBlanks := 3
	got := 0
	for _, l := range lines {
		if l == "" {
			got++
		}
	}
	if got != wantBlanks {
		t.Errorf("section separators = %d, want %d:\n%s", got, wantBlanks, joined)
	}
}

// TestBPLVHF_AllRegions renders every region without panics and checks
// each has its band headers.
func TestBPLVHF_AllRegions(t *testing.T) {
	for _, r := range []int{1, 2, 3} {
		lines := (&Model{}).viewBPLVHF(r)
		if len(lines) == 0 {
			t.Fatalf("region %d rendered no lines", r)
		}
		joined := strings.Join(lines, "\n")
		if !strings.Contains(joined, "2m") || !strings.Contains(joined, "70cm") {
			t.Errorf("region %d missing band headers:\n%s", r, joined)
		}
	}
}

func containsLine(lines []string, want string) bool {
	for _, l := range lines {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}
