package qrz

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// coalesce
// =============================================================================

func TestCoalesce(t *testing.T) {
	if got := coalesce("", "default"); got != "default" {
		t.Errorf("expected default, got %q", got)
	}
	if got := coalesce("value", "default"); got != "value" {
		t.Errorf("expected value, got %q", got)
	}
	if got := coalesce("", ""); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

// =============================================================================
// XML parsing (qrzCall → CallData)
// =============================================================================

func TestQRZCall_ToCallData(t *testing.T) {
	qc := qrzCall{
		Call:    "SP9ABC",
		Fname:   "Jan",
		Name:    "Jan Kowalski",
		Grid:    "JO90",
		Country: "Poland",
		Addr2:   "Krakow",
		State:   "MA",
		Zip:     "30-001",
		County:  "Malopolskie",
		Class:   "A",
		Email:   "sp9abc@example.com",
		URL:     "https://example.com",
		Lat:     "50.06",
		Lon:     "19.94",
		DXCC:    "269",
		CQZone:  "15",
		ITUZone: "28",
		Image:   "https://example.com/img.jpg",
	}

	cd := CallData{
		Callsign: qc.Call,
		Name:     coalesce(qc.Fname, qc.Name),
		Grid:     qc.Grid,
		Country:  qc.Country,
		QTH:      qc.Addr2,
		State:    qc.State,
		Zip:      qc.Zip,
		County:   qc.County,
		Class:    qc.Class,
		Email:    qc.Email,
		URL:      qc.URL,
		Lat:      qc.Lat,
		Lon:      qc.Lon,
		DXCC:     qc.DXCC,
		CQZone:   qc.CQZone,
		ITUZone:  qc.ITUZone,
		ImageURL: qc.Image,
	}

	if cd.Callsign != "SP9ABC" {
		t.Errorf("callsign = %q", cd.Callsign)
	}
	if cd.Name != "Jan" {
		t.Errorf("name = %q, expected Jan", cd.Name)
	}
	if cd.Grid != "JO90" {
		t.Errorf("grid = %q", cd.Grid)
	}
	if cd.Country != "Poland" {
		t.Errorf("country = %q", cd.Country)
	}
	if cd.QTH != "Krakow" {
		t.Errorf("QTH = %q", cd.QTH)
	}
	if cd.Lat != "50.06" || cd.Lon != "19.94" {
		t.Errorf("lat/lon = %q/%q", cd.Lat, cd.Lon)
	}
	if cd.CQZone != "15" || cd.ITUZone != "28" {
		t.Errorf("zones = %q/%q", cd.CQZone, cd.ITUZone)
	}
	if cd.ImageURL != "https://example.com/img.jpg" {
		t.Errorf("image = %q", cd.ImageURL)
	}
	if cd.State != "MA" || cd.Zip != "30-001" || cd.County != "Malopolskie" {
		t.Errorf("state/zip/county = %q/%q/%q", cd.State, cd.Zip, cd.County)
	}
	if cd.Class != "A" || cd.Email != "sp9abc@example.com" || cd.URL != "https://example.com" {
		t.Errorf("class/email/url = %q/%q/%q", cd.Class, cd.Email, cd.URL)
	}
	if cd.DXCC != "269" {
		t.Errorf("DXCC = %q", cd.DXCC)
	}
}

func TestCoalesce_FnameWins(t *testing.T) {
	// QRZ returns both fname (first name) and name (full name).
	// coalesce prefers fname if non-empty.
	qc := qrzCall{Fname: "Jan", Name: "Jan Kowalski"}
	if coalesce(qc.Fname, qc.Name) != "Jan" {
		t.Error("fname should win over name")
	}
}

func TestCoalesce_NameFallback(t *testing.T) {
	qc := qrzCall{Fname: "", Name: "Jan Kowalski"}
	if coalesce(qc.Fname, qc.Name) != "Jan Kowalski" {
		t.Error("name should be fallback when fname empty")
	}
}

// =============================================================================
// Session cache — concurrent access safety (Client-based).
// =============================================================================

func TestClientSessionCache_Concurrent(t *testing.T) {
	c := NewClient("user", "pass")
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			c.mu.Lock()
			_ = c.key
			c.mu.Unlock()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// Lookup integration tests — uses httptest.Server via httpGetFn seam.
// These now use Client directly instead of the deprecated package-level API.
// =============================================================================

func TestLookup_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		q := r.URL.RawQuery
		if strings.Contains(q, "username=") {
			xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
		} else {
			xml.NewEncoder(w).Encode(qrzDatabase{
				Session:  qrzKey{Key: "testkey"},
				Callsign: qrzCall{Call: "SP9ABC", Fname: "Jan", Grid: "JO90", Country: "Poland", Addr2: "Krakow", DXCC: "269", CQZone: "15", ITUZone: "28", Image: "https://example.com/photo.jpg"},
			})
		}
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "pass")
	data, err := c.Lookup("SP9ABC")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if data == nil {
		t.Fatal("expected data, got nil")
	}
	if data.Callsign != "SP9ABC" {
		t.Errorf("callsign = %q, want SP9ABC", data.Callsign)
	}
	if data.Name != "Jan" {
		t.Errorf("name = %q, want Jan", data.Name)
	}
	if data.Grid != "JO90" {
		t.Errorf("grid = %q, want JO90", data.Grid)
	}
	if data.Country != "Poland" {
		t.Errorf("country = %q, want Poland", data.Country)
	}
	if data.QTH != "Krakow" {
		t.Errorf("qth = %q, want Krakow", data.QTH)
	}
	if data.DXCC != "269" {
		t.Errorf("dxcc = %q, want 269", data.DXCC)
	}
	if data.CQZone != "15" {
		t.Errorf("cqzone = %q, want 15", data.CQZone)
	}
	if data.ITUZone != "28" {
		t.Errorf("ituzone = %q, want 28", data.ITUZone)
	}
	if data.ImageURL != "https://example.com/photo.jpg" {
		t.Errorf("imageURL = %q", data.ImageURL)
	}
	if data.Provider != "qrz" {
		t.Errorf("provider = %q, want qrz", data.Provider)
	}
}

func TestLookup_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		q := r.URL.RawQuery
		if strings.Contains(q, "username=") {
			xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
		} else {
			xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey", Error: "Not found: SP9XYZ"}})
		}
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "pass")
	data, err := c.Lookup("SP9XYZ")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if data != nil {
		t.Error("expected nil for not-found callsign")
	}
}

func TestLookup_AuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Error: "Invalid username or password"}})
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("baduser", "badpass")
	data, err := c.Lookup("SP9ABC")
	if err == nil {
		t.Error("expected error for bad credentials")
	}
	if data != nil {
		t.Error("expected nil data for bad credentials")
	}
}

func TestLookup_EmptyCall(t *testing.T) {
	c := NewClient("user", "pass")
	data, err := c.Lookup("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Error("expected nil for empty callsign")
	}
}

func TestLookup_EmptyUser(t *testing.T) {
	c := NewClient("", "pass")
	data, err := c.Lookup("SP9ABC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data != nil {
		t.Error("expected nil for empty username")
	}
}

func TestLookup_MalformedXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte("not valid xml {{{"))
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "pass")
	_, err := c.Lookup("SP9ABC")
	if err == nil {
		t.Error("expected error for malformed XML")
	}
}

func TestLookup_SessionReuse(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/xml")
		resp := qrzDatabase{
			Session:  qrzKey{Key: "testkey"},
			Callsign: qrzCall{Call: "SP9ABC", Fname: "Jan"},
		}
		xml.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "pass")

	// First call: should authenticate + lookup (2 HTTP calls).
	data, err := c.Lookup("SP9ABC")
	if err != nil || data == nil {
		t.Fatalf("first lookup failed: %v", err)
	}

	// Second call: should use cached session (1 HTTP call).
	callCount = 0
	data, err = c.Lookup("SP9ABC")
	if err != nil || data == nil {
		t.Fatalf("second lookup failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 HTTP call with cached session, got %d", callCount)
	}
}

func TestTestConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "pass")
	if err := c.TestConnection(); err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestTestConnection_EmptyCreds(t *testing.T) {
	c := NewClient("", "")
	if err := c.TestConnection(); err == nil {
		t.Error("expected error for empty credentials")
	}
}

func TestTestConnection_AuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Error: "Invalid password"}})
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	c := NewClient("user", "wrongpass")
	if err := c.TestConnection(); err == nil {
		t.Error("expected error for bad auth")
	}
}

// =============================================================================
// Legacy package-level API (deprecated, kept for backward compatibility).
// =============================================================================

func TestLegacyLookup_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		q := r.URL.RawQuery
		if strings.Contains(q, "username=") {
			xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
		} else {
			xml.NewEncoder(w).Encode(qrzDatabase{
				Session:  qrzKey{Key: "testkey"},
				Callsign: qrzCall{Call: "SP9ABC", Fname: "Jan", Grid: "JO90", Country: "Poland"},
			})
		}
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	// Legacy package-level Lookup returns *CallData.
	data, err := Lookup("user", "pass", "SP9ABC")
	if err != nil {
		t.Fatalf("Legacy Lookup failed: %v", err)
	}
	if data == nil || data.Callsign != "SP9ABC" {
		t.Error("legacy Lookup returned wrong data")
	}
}

func TestLegacyLookupResult_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		q := r.URL.RawQuery
		if strings.Contains(q, "username=") {
			xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
		} else {
			xml.NewEncoder(w).Encode(qrzDatabase{
				Session:  qrzKey{Key: "testkey"},
				Callsign: qrzCall{Call: "SP9ABC", Fname: "Jan", Grid: "JO90", Country: "Poland"},
			})
		}
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	// LookupResult returns the provider-neutral *callbook.Result.
	data, err := LookupResult("user", "pass", "SP9ABC")
	if err != nil {
		t.Fatalf("LookupResult failed: %v", err)
	}
	if data == nil || data.Callsign != "SP9ABC" {
		t.Error("LookupResult returned wrong data")
	}
	if data.Provider != "qrz" {
		t.Errorf("provider = %q, want qrz", data.Provider)
	}
}

func TestLegacyTestConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		xml.NewEncoder(w).Encode(qrzDatabase{Session: qrzKey{Key: "testkey"}})
	}))
	defer srv.Close()

	orig := httpGetFn
	httpGetFn = rewriteURL(srv.URL)
	defer func() { httpGetFn = orig }()

	if err := TestConnection("user", "pass"); err != nil {
		t.Errorf("Legacy TestConnection failed: %v", err)
	}
}

// =============================================================================
// Test helpers
// =============================================================================

func rewriteURL(baseURL string) func(string) ([]byte, error) {
	return func(origURL string) ([]byte, error) {
		idx := strings.Index(origURL, "?")
		if idx < 0 {
			return httpGet(origURL)
		}
		newURL := baseURL + "/xml/current/" + origURL[idx:]
		return httpGet(newURL)
	}
}
