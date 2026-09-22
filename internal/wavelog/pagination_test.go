package wavelog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
)

const adifHeader = "Wavelog ADIF export\n<ADIF_VER:5>3.1.7\n<PROGRAMID:7>Wavelog\n<EOH>\n"

func adifRecord(call string) string {
	return "<CALL:6>" + call + "<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260920<TIME_ON:4>1200<EOR>\n"
}

// adifRecordAt builds an SSB/20m record with a specific 4-digit time.
func adifRecordAt(call, timeOn string) string {
	return "<CALL:6>" + call + "<BAND:3>20m<MODE:3>SSB<QSO_DATE:8>20260920<TIME_ON:4>" + timeOn + "<EOR>\n"
}

// TestFetchContacts_MultiPage verifies the paginated ADIF export: pages are
// requested with since_id=lastfetchedid, headers of later pages are stripped,
// rows from all pages land in one file, and the progress callback reports
// cumulative exported counts against meta.total.
func TestFetchContacts_MultiPage(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/qso" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
			http.Error(w, "bad request", 400)
			return
		}

		// The JSON list (no format param) is the remote-id sidecar request:
		// one complete-list request, ordered newest first like real Wavelog.
		if r.URL.Query().Get("format") == "" {
			if r.URL.Query().Get("page") != "1" || r.URL.Query().Get("per_page") != "5000" {
				t.Errorf("list query = %s, want page=1 per_page=5000", r.URL.RawQuery)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 5, "call": "SP9CCC", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
					{"id": 2, "call": "SP9BBB", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
					{"id": 1, "call": "SP9AAA", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
				},
				"meta": map[string]any{"has_more": false, "total": 3},
			})
			return
		}

		page++
		since, _ := strconv.ParseInt(r.URL.Query().Get("since_id"), 10, 64)
		if page == 1 && since != 0 {
			t.Errorf("page 1 since_id = %d, want 0", since)
		}
		if page == 2 && since != 2 {
			t.Errorf("page 2 since_id = %d, want 2 (lastfetchedid of page 1)", since)
		}
		if r.URL.Query().Get("per_page") != strconv.Itoa(v2ADIFPageSize) {
			t.Errorf("per_page = %q, want %d", r.URL.Query().Get("per_page"), v2ADIFPageSize)
		}

		switch page {
		case 1:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"exported":      2,
					"lastfetchedid": 2,
					"adif":          adifHeader + adifRecord("SP9AAA") + adifRecord("SP9BBB"),
				},
				"meta": map[string]any{"has_more": true, "total": 3},
			})
		case 2:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"exported":      1,
					"lastfetchedid": 5,
					"adif":          adifHeader + adifRecord("SP9CCC"),
				},
				"meta": map[string]any{"has_more": false, "total": 3},
			})
		default:
			t.Error("unexpected extra page request")
			http.Error(w, "too many", 500)
		}
	}))
	defer srv.Close()

	var calls [][2]int
	result, err := FetchContactsProgress(srv.URL, "wl2_test", "1", 0,
		func(exported, total int) {
			calls = append(calls, [2]int{exported, total})
		})
	if err != nil {
		t.Fatalf("FetchContactsProgress: %v", err)
	}
	defer os.Remove(result.ADIFPath)

	if page != 2 {
		t.Errorf("server saw %d page requests, want 2", page)
	}
	if result.ExportedQSOs != 3 {
		t.Errorf("ExportedQSOs = %d, want 3", result.ExportedQSOs)
	}
	if result.TotalRows != 3 {
		t.Errorf("TotalRows = %d, want 3", result.TotalRows)
	}
	if result.LastFetchedID() != 5 {
		t.Errorf("LastFetchedID = %d, want 5", result.LastFetchedID())
	}

	data, err := os.ReadFile(result.ADIFPath)
	if err != nil {
		t.Fatalf("read temp file: %v", err)
	}
	content := string(data)
	if got := strings.Count(content, "<EOH>"); got != 1 {
		t.Errorf("file has %d <EOH> markers, want 1 (headers stripped)", got)
	}
	for _, call := range []string{"SP9AAA", "SP9BBB", "SP9CCC"} {
		if !strings.Contains(content, call) {
			t.Errorf("file missing record for %s", call)
		}
	}

	if len(calls) != 2 {
		t.Fatalf("onPage called %d times, want 2", len(calls))
	}
	if calls[0] != [2]int{2, 3} || calls[1] != [2]int{3, 3} {
		t.Errorf("progress calls = %v, want [(2,3) (3,3)]", calls)
	}

	// Remote ids are matched by verified identity — one entry per exported
	// row, keyed by call/band/mode/date/time.
	if result.WavelogIDsByKey == nil {
		t.Fatal("WavelogIDsByKey should not be nil when all id pages verified")
	}
	want := map[QSOIDKey]int64{
		MakeQSOIDKey("SP9AAA", "20m", "SSB", "2026-09-20", "120000"): 1,
		MakeQSOIDKey("SP9BBB", "20m", "SSB", "2026-09-20", "120000"): 2,
		MakeQSOIDKey("SP9CCC", "20m", "SSB", "2026-09-20", "120000"): 5,
	}
	if len(result.WavelogIDsByKey) != len(want) {
		t.Fatalf("WavelogIDsByKey has %d entries, want %d: %v",
			len(result.WavelogIDsByKey), len(want), result.WavelogIDsByKey)
	}
	for k, id := range want {
		if got := result.WavelogIDsByKey[k]; got != id {
			t.Errorf("WavelogIDsByKey[%+v] = %d, want %d", k, got, id)
		}
	}
}

// TestFetchContacts_FullListIdentityMatching is the regression test for the
// ordering mismatch: the JSON id list is ordered newest-first while the ADIF
// export is ascending by primary key. Per-page sidecar windows can therefore
// never align with ADIF pages; ids must come from the complete list and be
// matched by verified identity only.
func TestFetchContacts_FullListIdentityMatching(t *testing.T) {
	page := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("format") == "" {
			// Complete list, newest first (real Wavelog order).
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 12, "call": "SP9DDD", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 16:00:00"},
					{"id": 11, "call": "SP9CCC", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 15:00:00"},
					{"id": 2, "call": "SP9BBB", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
					{"id": 1, "call": "SP9AAA", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
				},
				"meta": map[string]any{"has_more": false, "total": 4},
			})
			return
		}

		page++
		adif := adifHeader + adifRecord("SP9AAA") + adifRecord("SP9BBB")
		last := 2
		hasMore := true
		if page == 2 {
			adif = adifHeader + adifRecordAt("SP9CCC", "1500") + adifRecordAt("SP9DDD", "1600")
			last = 12
			hasMore = false
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": last,
				"adif":          adif,
			},
			"meta": map[string]any{"has_more": hasMore, "total": 4},
		})
	}))
	defer srv.Close()

	result, err := FetchContacts(srv.URL, "wl2_test", "1", 0)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	defer os.Remove(result.ADIFPath)

	if result.ExportedQSOs != 4 {
		t.Errorf("ExportedQSOs = %d, want 4", result.ExportedQSOs)
	}

	want := map[string]struct {
		id     int64
		timeOn string
	}{
		"SP9AAA": {1, "120000"},
		"SP9BBB": {2, "120000"},
		"SP9CCC": {11, "150000"},
		"SP9DDD": {12, "160000"},
	}
	if result.WavelogIDsByKey == nil {
		t.Fatal("WavelogIDsByKey should not be nil when the list was fetched")
	}
	if len(result.WavelogIDsByKey) != len(want) {
		t.Fatalf("WavelogIDsByKey has %d entries, want %d: %v",
			len(result.WavelogIDsByKey), len(want), result.WavelogIDsByKey)
	}
	for call, w := range want {
		key := MakeQSOIDKey(call, "20m", "SSB", "2026-09-20", w.timeOn)
		if got := result.WavelogIDsByKey[key]; got != w.id {
			t.Errorf("WavelogIDsByKey[%s] = %d, want %d", call, got, w.id)
		}
	}
}

// TestFetchContacts_IDCaptureFailureDoesNotAbort verifies that a failing id
// list must never abort the download: rows arrive without ids (nil map) and
// the caller freezes its cursor so the next download retries.
func TestFetchContacts_IDCaptureFailureDoesNotAbort(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("format") == "" {
			http.Error(w, "boom", 500)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 2,
				"adif":          adifHeader + adifRecord("SP9AAA") + adifRecord("SP9BBB"),
			},
			"meta": map[string]any{"has_more": false, "total": 2},
		})
	}))
	defer srv.Close()

	result, err := FetchContacts(srv.URL, "wl2_test", "1", 0)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	defer os.Remove(result.ADIFPath)

	if result.ExportedQSOs != 2 {
		t.Errorf("ExportedQSOs = %d, want 2", result.ExportedQSOs)
	}
	if result.WavelogIDsByKey != nil {
		t.Errorf("WavelogIDsByKey = %v, want nil when the id list fails", result.WavelogIDsByKey)
	}
}

// TestFetchContacts_ShortIDListMatchesByIdentity: a list shorter than the
// export is simply an incomplete identity source — rows it covers get ids,
// the rest stay unlinked. No positional matching anywhere.
func TestFetchContacts_ShortIDListMatchesByIdentity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("format") == "" {
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": 1, "call": "SP9AAA", "band": "20m", "mode": "SSB",
						"qso_date": "2026-09-20 12:00:00"},
				},
				"meta": map[string]any{"has_more": false, "total": 1},
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      2,
				"lastfetchedid": 2,
				"adif":          adifHeader + adifRecord("SP9AAA") + adifRecord("SP9BBB"),
			},
			"meta": map[string]any{"has_more": false, "total": 2},
		})
	}))
	defer srv.Close()

	result, err := FetchContacts(srv.URL, "wl2_test", "1", 0)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	defer os.Remove(result.ADIFPath)

	keyA := MakeQSOIDKey("SP9AAA", "20m", "SSB", "2026-09-20", "120000")
	keyB := MakeQSOIDKey("SP9BBB", "20m", "SSB", "2026-09-20", "120000")
	if result.WavelogIDsByKey == nil || len(result.WavelogIDsByKey) != 1 {
		t.Fatalf("WavelogIDsByKey = %v, want exactly the covered row", result.WavelogIDsByKey)
	}
	if result.WavelogIDsByKey[keyA] != 1 {
		t.Errorf("SP9AAA id = %d, want 1", result.WavelogIDsByKey[keyA])
	}
	if _, ok := result.WavelogIDsByKey[keyB]; ok {
		t.Errorf("SP9BBB should not have an id: %d", result.WavelogIDsByKey[keyB])
	}
}

// TestFetchContacts_NoServerProgress verifies the loop-safety guard: a server
// that keeps returning the same lastfetchedid with has_more=true must not
// cause an infinite loop.
func TestFetchContacts_NoServerProgress(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      1,
				"lastfetchedid": 2, // never advances
				"adif":          adifHeader + adifRecord("SP9AAA"),
			},
			"meta": map[string]any{"has_more": true, "total": 100},
		})
	}))
	defer srv.Close()

	_, err := FetchContacts(srv.URL, "wl2_test", "1", 0)
	if err == nil {
		t.Fatal("expected an error when the server makes no progress")
	}
	if !strings.Contains(err.Error(), "did not advance") {
		t.Errorf("error = %q, want lastfetchedid progress error", err)
	}
}

// TestFetchContacts_EmptyFirstPage verifies the zero-rows short circuit:
// nothing new since since_id returns an empty file and keeps the input ID.
func TestFetchContacts_EmptyFirstPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"exported":      0,
				"lastfetchedid": 42,
				"adif":          nil,
			},
			"meta": map[string]any{"has_more": false, "total": 0},
		})
	}))
	defer srv.Close()

	result, err := FetchContacts(srv.URL, "wl2_test", "1", 42)
	if err != nil {
		t.Fatalf("FetchContacts: %v", err)
	}
	defer os.Remove(result.ADIFPath)

	if result.ExportedQSOs != 0 {
		t.Errorf("ExportedQSOs = %d, want 0", result.ExportedQSOs)
	}
	if result.LastFetchedID() != 42 {
		t.Errorf("LastFetchedID = %d, want 42 (unchanged)", result.LastFetchedID())
	}
	if result.TotalRows != 0 {
		t.Errorf("TotalRows = %d, want 0", result.TotalRows)
	}
}
