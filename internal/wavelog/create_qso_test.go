package wavelog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// TestDeleteQSO verifies the DELETE verb: 204 on success, not_found on a
// missing QSO, and the token sent in the Authorization header.
func TestDeleteQSO(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "Bearer wl2_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/api/v2/qso/42":
			w.WriteHeader(http.StatusNoContent)
		case "/api/v2/qso/404":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "not_found", "message": "QSO not found"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if err := DeleteQSO(srv.URL, "wl2_test", 42); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}

	err := DeleteQSO(srv.URL, "wl2_test", 404)
	if err == nil {
		t.Fatal("expected not_found error")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "not_found" {
		t.Errorf("error = %v, want APIError not_found", err)
	}

	if err := DeleteQSO(srv.URL, "wl2_test", 0); err == nil {
		t.Error("expected error for invalid remote id")
	}
}

// TestCreateQSO_Success verifies the single JSON create payload and the
// remote id returned in the created object.
func TestCreateQSO_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "bad request", 400)
			return
		}
		if r.Header.Get("Authorization") != "Bearer wl2_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body["station_profile_id"] != float64(1) {
			t.Errorf("station_profile_id = %v, want 1", body["station_profile_id"])
		}
		if body["call"] != "SP9MOA" || body["band"] != "20m" || body["mode"] != "FT8" {
			t.Errorf("fields = %v/%v/%v", body["call"], body["band"], body["mode"])
		}
		if body["qso_date"] != "2026-06-18" || body["time_on"] != "120000" {
			t.Errorf("date/time = %v/%v", body["qso_date"], body["time_on"])
		}
		if body["freq"] != float64(14074550) {
			t.Errorf("freq = %v, want 14074550 (Hz)", body["freq"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"id": 42, "call": "SP9MOA"},
			"meta": map[string]string{"resource": "qso", "method": "POST"},
		})
	}))
	defer srv.Close()

	id, dup, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{
		StationProfileID: 1,
		Call:             "SP9MOA",
		Band:             "20m",
		Mode:             "FT8",
		QSODate:          "2026-06-18",
		TimeOn:           "120000",
		FreqHz:           14074550,
	})
	if err != nil {
		t.Fatalf("CreateQSO: %v", err)
	}
	if dup {
		t.Error("unexpected duplicate")
	}
	if id != 42 {
		t.Errorf("remote id = %d, want 42", id)
	}
}

// TestCreateQSO_Duplicate verifies conflict and bulk-summary duplicate shapes.
func TestCreateQSO_Duplicate(t *testing.T) {
	for _, mode := range []string{"conflict", "summary", "validation_duplicate"} {
		t.Run(mode, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if mode == "conflict" {
					w.WriteHeader(http.StatusConflict)
					json.NewEncoder(w).Encode(map[string]any{
						"error": map[string]string{"code": "conflict", "message": "Duplicate QSO"},
					})
					return
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]any{
					"data": map[string]any{"parsed": 1, "imported": 0, "skipped": 1, "messages": []string{}},
					"meta": map[string]string{"resource": "qso", "method": "POST"},
				})
			}))
			defer srv.Close()

			id, dup, err := CreateQSO(srv.URL, "wl2_test", CreateQSOInput{
				StationProfileID: 1, Call: "SP9MOA", Band: "20m", Mode: "SSB",
				QSODate: "2026-06-18", TimeOn: "120000",
			})
			if err != nil {
				t.Fatalf("CreateQSO: %v", err)
			}
			if !dup {
				t.Error("expected duplicate")
			}
			if id != 0 {
				t.Errorf("id = %d, want 0 for duplicate", id)
			}
		})
	}
}

func TestGetQSO(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/qso/42" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer wl2_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"id": 42, "station_id": 1, "qso_date": "2026-05-01 12:30:00",
				"mode": "SSB", "submode": nil, "freq": "14250000", "freq_rx": nil,
				"call": "SP9MOA", "band": "20m", "rst_sent": "55", "rst_rcvd": "57",
				"gridsquare": "JO90", "name": "Darek", "comment": "c", "qth": "Krakow",
				"tx_pwr": "25", "cqz": 15, "ituz": 28,
			},
		})
	}))
	defer srv.Close()

	qs, err := GetQSO(srv.URL, "wl2_test", 42)
	if err != nil {
		t.Fatalf("GetQSO: %v", err)
	}
	if qs.ID != 42 || qs.Call != "SP9MOA" || qs.QSODate != "2026-05-01 12:30:00" {
		t.Errorf("unexpected QSO data: %+v", qs)
	}
	if qs.Freq != "14250000" || qs.TXPower != "25" || qs.CQZ != 15 || qs.ITUZ != 28 {
		t.Errorf("field decoding wrong: %+v", qs)
	}
}

func TestGetQSOMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "not_found", "message": "QSO not found"},
		})
	}))
	defer srv.Close()

	_, err := GetQSO(srv.URL, "wl2_test", 42)
	if err == nil {
		t.Fatal("GetQSO on missing QSO should fail")
	}
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "not_found" {
		t.Errorf("error = %v, want APIError not_found", err)
	}
}

func TestUpdateQSO(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/v2/qso/42" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer wl2_test" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
	}))
	defer srv.Close()

	err := UpdateQSO(srv.URL, "wl2_test", 42, UpdateQSOInput{
		Call: "SP9MOA", Band: "20m", Mode: "SSB",
		QSODate: "2026-05-01", TimeOn: "12:30:00",
		RSTSent: "55", RSTRcvd: "57", Comment: "edited",
		FreqHz: 14250000, TXPower: "25",
	})
	if err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if body == nil {
		t.Fatal("no PATCH body received")
	}
	if body["qso_date"] != "2026-05-01" || body["time_on"] != "12:30:00" {
		t.Errorf("date/time = %v/%v", body["qso_date"], body["time_on"])
	}
	if body["freq"] != "14250000" {
		t.Errorf("freq = %v, want string Hz", body["freq"])
	}
	if body["comment"] != "edited" || body["rst_sent"] != "55" {
		t.Errorf("fields wrong: %v", body)
	}
	if _, ok := body["freq_rx"]; ok {
		t.Error("freq_rx must be omitted when 0")
	}
}

func TestUpdateQSODateRequiresTime(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		_, hasDate := body["qso_date"]
		_, hasTime := body["time_on"]
		if hasDate || hasTime {
			t.Errorf("date/time must be omitted when not sent together: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": 42}})
	}))
	defer srv.Close()

	if err := UpdateQSO(srv.URL, "wl2_test", 42, UpdateQSOInput{Call: "SP9MOA", QSODate: "2026-05-01"}); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
}

func TestFetchAllQSOIDs_Paginates(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v2/qso" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		page++
		if q.Get("station_id") != "1" || q.Get("per_page") != "5000" || q.Get("page") != strconv.Itoa(page) {
			t.Errorf("query = %v, want station_id=1 per_page=5000 page=%d", q, page)
		}
		var rows []map[string]any
		switch page {
		case 1:
			rows = []map[string]any{
				{"id": 10, "call": "SP9AAA", "band": "20m", "mode": "SSB", "qso_date": "2026-05-01 12:00:00"},
				{"id": 11, "call": "SP9BBB", "band": "40m", "mode": "CW", "qso_date": "2026-05-02 12:00:00"},
			}
		case 2:
			rows = []map[string]any{
				{"id": 12, "call": "SP9AAA", "band": "20m", "mode": "SSB", "qso_date": "2026-05-01 12:00:00"}, // duplicate key — first (newest) wins
				{"id": 13, "call": "SP9CCC", "band": "15m", "mode": "FT8", "mode_sub": "", "qso_date": "2026-05-03 12:00:00"},
			}
		case 3:
			rows = []map[string]any{
				{"id": 14, "call": "SP9DDD", "band": "10m", "mode": "MFSK", "submode": "FT4", "qso_date": "2026-05-04 12:00:00"},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": rows,
			"meta": map[string]any{"page": page, "per_page": 5000, "has_more": page < 3},
		})
	}))
	defer srv.Close()

	ids, err := FetchAllQSOIDs(srv.URL, "wl2_test", "1")
	if err != nil {
		t.Fatalf("FetchAllQSOIDs: %v", err)
	}
	if page != 3 {
		t.Errorf("pages fetched = %d, want 3", page)
	}
	keyAAA := QSOIDKey{Call: "SP9AAA", Band: "20M", Mode: "SSB", QSODate: "2026-05-01", TimeOn: "120000"}
	if ids[keyAAA] != 10 {
		t.Errorf("SP9AAA id = %d, want 10 (newest first wins)", ids[keyAAA])
	}
	keyFT4 := QSOIDKey{Call: "SP9DDD", Band: "10M", Mode: "FT4", QSODate: "2026-05-04", TimeOn: "120000"}
	if ids[keyFT4] != 14 {
		t.Errorf("FT4-canonicalized key id = %d, want 14", ids[keyFT4])
	}
	if len(ids) != 4 {
		t.Errorf("map size = %d, want 4", len(ids))
	}
}
