package qso

import "testing"

// FillSSBSubmode is the store-side safety net for SSB contacts that arrive
// without a sideband (ADIF imports with only MODE=SSB, spot fills, rows whose
// submode was blanked by an edit).
func TestFillSSBSubmode(t *testing.T) {
	cases := []struct {
		name string
		qso  QSO
		want string
	}{
		{"40m SSB gets LSB", QSO{Mode: "SSB", Freq: 7.1968}, "LSB"},
		{"80m SSB gets LSB", QSO{Mode: "SSB", Freq: 3.7}, "LSB"},
		{"20m SSB gets USB", QSO{Mode: "SSB", Freq: 14.25}, "USB"},
		{"10m SSB gets USB", QSO{Mode: "SSB", Freq: 28.5}, "USB"},
		{"lowercase mode", QSO{Mode: "ssb", Freq: 7.1}, "LSB"},
		{"explicit LSB is left alone", QSO{Mode: "SSB", Submode: "LSB", Freq: 14.25}, "LSB"},
		{"explicit USB is left alone", QSO{Mode: "SSB", Submode: "USB", Freq: 7.1}, "USB"},
		{"CW keeps its empty submode", QSO{Mode: "CW", Freq: 7.02}, ""},
		{"FT8 keeps its empty submode", QSO{Mode: "FT8", Freq: 14.074}, ""},
		{"no frequency means no guess", QSO{Mode: "SSB"}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := tc.qso
			FillSSBSubmode(&q)
			if q.Submode != tc.want {
				t.Errorf("Submode = %q, want %q", q.Submode, tc.want)
			}
		})
	}
	// nil must not panic — the store calls this on every write.
	FillSSBSubmode(nil)
}
