package wavelog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListRadios(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/radio" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer wl2_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": 3, "radio": "FT-950", "frequency": 14075000, "mode": "SSB", "power": 100},
				{"id": 7, "radio": "CQOps", "frequency": 7100000, "mode": "CW", "power": 50},
			},
		})
	}))
	defer srv.Close()

	radios, err := ListRadios(srv.URL, "wl2_test")
	if err != nil {
		t.Fatalf("ListRadios: %v", err)
	}
	if len(radios) != 2 || radios[1].Name != "CQOps" || radios[1].ID != 7 {
		t.Errorf("radios = %+v", radios)
	}
}

func TestEnsureRadio_Existing(t *testing.T) {
	postCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": 7, "radio": "CQOps", "frequency": 7100000, "mode": "CW"}},
			})
		case http.MethodPost:
			postCount++
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	r, err := EnsureRadio(srv.URL, "wl2_test", "CQOps")
	if err != nil {
		t.Fatalf("EnsureRadio: %v", err)
	}
	if r == nil || r.ID != 7 {
		t.Fatalf("radio = %+v, want id 7", r)
	}
	if postCount != 0 {
		t.Errorf("POST called %d times — existing radio must not be recreated", postCount)
	}
}

func TestEnsureRadio_Creates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case http.MethodPost:
			var body map[string]any
			json.NewDecoder(r.Body).Decode(&body)
			if body["radio"] != "CQOps" {
				t.Errorf("create body radio = %v", body["radio"])
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{"id": 9, "radio": "CQOps", "frequency": 0},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	r, err := EnsureRadio(srv.URL, "wl2_test", "CQOps")
	if err != nil {
		t.Fatalf("EnsureRadio: %v", err)
	}
	if r == nil || r.ID != 9 {
		t.Fatalf("radio = %+v, want id 9", r)
	}
}

func TestPushRadio_Payload(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v2/radio" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 9, "radio": "CQOps", "frequency": 14250000, "mode": "SSB", "power": 25},
		})
	}))
	defer srv.Close()

	r, err := PushRadio(srv.URL, "wl2_test", "CQOps", RadioState{
		FrequencyHz: 14250000, FrequencyRxHz: 145550000, Mode: "SSB", PowerWatts: 25,
	})
	if err != nil {
		t.Fatalf("PushRadio: %v", err)
	}
	if r == nil || r.ID != 9 {
		t.Fatalf("radio = %+v, want id 9", r)
	}
	if body["radio"] != "CQOps" {
		t.Errorf("radio = %v", body["radio"])
	}
	if body["frequency"] != float64(14250000) || body["frequency_rx"] != float64(145550000) {
		t.Errorf("frequencies = %v/%v", body["frequency"], body["frequency_rx"])
	}
	if body["mode"] != "SSB" || body["power"] != float64(25) {
		t.Errorf("mode/power = %v/%v", body["mode"], body["power"])
	}
}

func TestPushRadio_ModeTrimmedAndOmitted(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 1, "radio": "CQOps"}})
	}))
	defer srv.Close()

	if _, err := PushRadio(srv.URL, "wl2_test", "CQOps", RadioState{
		Mode: "VERYLONGMODENAME",
	}); err != nil {
		t.Fatalf("PushRadio: %v", err)
	}
	if body["mode"] != "VERYLONGMO" {
		t.Errorf("mode = %v, want trimmed to 10 chars", body["mode"])
	}
	if _, ok := body["frequency"]; ok {
		t.Error("frequency must be omitted when 0")
	}
	if _, ok := body["power"]; ok {
		t.Error("power must be omitted when 0")
	}
}

func TestPushRadio_NotFoundError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "not_found", "message": "Radio not found"},
		})
	}))
	defer srv.Close()

	_, err := PushRadio(srv.URL, "wl2_test", "CQOps", RadioState{FrequencyHz: 7100000, Mode: "CW"})
	if err == nil {
		t.Fatal("PushRadio should fail on 404")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "not_found" {
		t.Errorf("error = %v, want APIError not_found", err)
	}
}

func TestGetStation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Station ids with display-label suffixes are sanitized to the numeric id.
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/station/1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 1, "callsign": "SP9SPM", "gridsquare": "KO00CA",
				"dxcc": 269, "cq": 15, "itu": 28, "sota": "SP/TA-001", "active": true,
			},
		})
	}))
	defer srv.Close()

	st, err := GetStation(srv.URL, "wl2_test", "1 — Home (SP9SPM) KO00CA")
	if err != nil {
		t.Fatalf("GetStation: %v", err)
	}
	if st.Gridsquare != "KO00CA" || st.DXCC != 269 || st.CQ != 15 || st.ITU != 28 || st.SOTA != "SP/TA-001" {
		t.Errorf("station = %+v", st)
	}
}
