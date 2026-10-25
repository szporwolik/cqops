package wavelog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A contest contact used to reach Wavelog without its exchange: the JSON
// create sent only the core columns, so the remote QSO had no STX/SRX/
// STX_STRING/SRX_STRING and no CONTEST_ID — it landed as a plain contact with
// no contest session attached, and the ADIF path was the only way to get it
// right.
func TestCreateQSOCarriesContestExchange(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9},"meta":{}}`))
	}))
	defer srv.Close()

	id, dup, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{
		StationProfileID: 1,
		Call:             "DL1ABC",
		Band:             "40m",
		Mode:             "SSB",
		Submode:          "LSB",
		QSODate:          "2026-10-25",
		TimeOn:           "1932",
		RSTSent:          "59",
		RSTRcvd:          "59",
		ContestID:        "CQ-WW-SSB",
		STX:              15,
		SRX:              16,
		STXString:        "15",
		SRXString:        "16",
	})
	if err != nil || dup || id != 9 {
		t.Fatalf("CreateQSO = %d/%v/%v", id, dup, err)
	}

	want := map[string]any{
		"contest_id": "CQ-WW-SSB",
		"stx":        float64(15),
		"srx":        float64(16),
		"stx_string": "15",
		"srx_string": "16",
		"submode":    "LSB",
		"mode":       "SSB",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("payload[%q] = %v, want %v", k, got[k], v)
		}
	}
}

// A canonical digital mode is sent as the mode itself — no duplicate submode.
func TestCreateQSOOmitsRedundantSubmode(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":9},"meta":{}}`))
	}))
	defer srv.Close()

	if _, _, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{
		StationProfileID: 1, Call: "DL1ABC", Band: "20m",
		Mode: "FT8", Submode: "FT8", QSODate: "2026-10-25", TimeOn: "1932",
	}); err != nil {
		t.Fatalf("CreateQSO: %v", err)
	}
	if _, ok := got["submode"]; ok {
		t.Errorf("submode sent for FT8: %v", got["submode"])
	}
	if got["mode"] != "FT8" {
		t.Errorf("mode = %v", got["mode"])
	}
}

// Editing a contest contact must carry the exchange too: stx/srx/
// stx_string/srx_string are in the API's editable-field list.
func TestUpdateQSOCarriesContestExchange(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":77},"meta":{}}`))
	}))
	defer srv.Close()

	stx, srx := 21, 22
	sts, srs := "21", "22"
	err := UpdateQSO(srv.URL, "wl2_test", 77, UpdateQSOInput{
		Call: "DL1ABC", Band: "40m", Mode: "SSB",
		QSODate: "2026-10-25", TimeOn: "19:32:00",
		STX: &stx, SRX: &srx, STXString: &sts, SRXString: &srs,
	})
	if err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	want := map[string]any{
		"stx":        float64(21),
		"srx":        float64(22),
		"stx_string": "21",
		"srx_string": "22",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("payload[%q] = %v, want %v", k, got[k], v)
		}
	}
	// contest_id is not editable through PATCH; sending it would be rejected.
	if _, ok := got["contest_id"]; ok {
		t.Error("contest_id must not be sent on PATCH")
	}
}

// A contact without a serial exchange clears the remote serials instead of
// leaving stale numbers behind.
func TestUpdateQSOClearsEmptyExchange(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":77},"meta":{}}`))
	}))
	defer srv.Close()

	zero := 0
	empty := ""
	if err := UpdateQSO(srv.URL, "wl2_test", 77, UpdateQSOInput{
		Call: "DL1ABC", STX: &zero, SRX: &zero, STXString: &empty, SRXString: &empty,
	}); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	for _, k := range []string{"stx", "srx", "stx_string", "srx_string"} {
		v, ok := got[k]
		if !ok {
			t.Errorf("%q missing from the PATCH body", k)
			continue
		}
		if v != nil {
			t.Errorf("%q = %v, want null", k, v)
		}
	}
}
