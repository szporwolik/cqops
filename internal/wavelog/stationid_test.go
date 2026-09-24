package wavelog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestParseStationID covers the display-label tolerance: the station picker
// renders "1 — SP9SPM (Home) JO90" into the read-only field, and that value
// historically leaked into the saved config.
func TestParseStationID(t *testing.T) {
	cases := []struct {
		in      string
		wantID  int
		wantErr bool
	}{
		{"1", 1, false},
		{"  7  ", 7, false},
		{"1 — SP9SPM (Niepołomice) KO00CA", 1, false},
		{"12 — SP9SPM (Niepołomice) KO00CA", 12, false},
		{"abc", 0, true},
		{"— 1", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		id, err := ParseStationID(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseStationID(%q) = %d, want error", c.in, id)
			}
			continue
		}
		if err != nil || id != c.wantID {
			t.Errorf("ParseStationID(%q) = %d, %v; want %d", c.in, id, err, c.wantID)
		}
	}
}

func TestSanitizeStationID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"1", "1"},
		{"1 — SP9SPM (Niepołomice) KO00CA", "1"},
		{"  42 — label", "42"},
		{"abc", "abc"},
		{"", ""},
	}
	for _, c := range cases {
		if got := SanitizeStationID(c.in); got != c.want {
			t.Errorf("SanitizeStationID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestFetchContacts_LabelStationID verifies the paginated download tolerates
// a station id carrying a display label and sends the numeric id upstream.
func TestFetchContacts_LabelStationID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("station_id"); got != "1" {
			t.Errorf("station_id = %q, want 1", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 0,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false},
		})
	}))
	defer srv.Close()

	res, err := FetchContacts(srv.URL, "wl2_test", "1 — SP9SPM (Niepołomice) KO00CA", 0)
	if err != nil {
		t.Fatalf("FetchContacts with label station id: %v", err)
	}
	defer os.Remove(res.ADIFPath)
	if res.ExportedQSOs != 0 {
		t.Errorf("ExportedQSOs = %d, want 0", res.ExportedQSOs)
	}
}
