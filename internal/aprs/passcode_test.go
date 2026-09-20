package aprs

import "testing"

// TestPasscode verifies the APRS-IS passcode against known reference
// vectors. The SP9SPM vector was verified against a live APRS-IS server.
func TestPasscode(t *testing.T) {
	cases := []struct {
		call string
		want string
	}{
		{"SP9SPM", "18860"}, // verified live against euro.aprs2.net (aprsc)
		{"N0CALL", "13023"},
		{"W1AW", "25988"},
		{"VK2BEN", "21157"},
	}
	for _, tc := range cases {
		if got := Passcode(tc.call); got != tc.want {
			t.Errorf("Passcode(%q) = %q, want %q", tc.call, got, tc.want)
		}
	}
}

// TestPasscode_Normalization verifies that SSIDs, portable suffixes,
// case, and whitespace do not change the result.
func TestPasscode_Normalization(t *testing.T) {
	base := Passcode("SP9SPM")
	cases := []string{
		"sp9spm",
		"SP9SPM-10",
		"SP9SPM-7",
		"SP9SPM/P",
		"SP9SPM/M",
		" SP9SPM ",
		"SP9SPM/2",
	}
	for _, c := range cases {
		if got := Passcode(c); got != base {
			t.Errorf("Passcode(%q) = %q, want %q", c, got, base)
		}
	}
}

// TestCanonicalCall verifies SSID-aware identity normalization: an omitted
// SSID defaults to -0 per the APRS 1.0.1 spec.
func TestCanonicalCall(t *testing.T) {
	cases := []struct{ in, want string }{
		{"SP9SPM", "SP9SPM-0"},
		{"SP9SPM-0", "SP9SPM-0"},
		{"SP9SPM-7", "SP9SPM-7"},
		{"sp9spm-7", "SP9SPM-7"},
		{"SP9SPM-7/P", "SP9SPM-7"},
		{"SP9SPM/P", "SP9SPM-0"},
		{" SP9SPM ", "SP9SPM-0"},
	}
	for _, tc := range cases {
		if got := CanonicalCall(tc.in); got != tc.want {
			t.Errorf("CanonicalCall(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
