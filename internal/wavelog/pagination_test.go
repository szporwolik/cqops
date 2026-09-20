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

		// The JSON list (no format param) is the remote-id sidecar request.
		// Real Wavelog orders it newest first; encode descending to exercise
		// the client-side ascending sort.
		if r.URL.Query().Get("format") == "" {
			w.Header().Set("Content-Type", "application/json")
			if page == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{{"id": 2}, {"id": 1}},
				})
			} else {
				json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{{"id": 5}},
				})
			}
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

	// Remote ids must align 1:1 with the exported rows, ascending.
	if len(result.WavelogIDs) != 3 {
		t.Fatalf("WavelogIDs = %v, want 3 entries", result.WavelogIDs)
	}
	want := []int64{1, 2, 5}
	for i, id := range result.WavelogIDs {
		if id != want[i] {
			t.Errorf("WavelogIDs[%d] = %d, want %d", i, id, want[i])
		}
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
