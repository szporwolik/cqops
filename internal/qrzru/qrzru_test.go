package qrzru

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// =============================================================================
// XML parsing tests
// =============================================================================

func TestXMLParsing_Success(t *testing.T) {
	resp := `<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>abc123</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
</QRZDatabase>`

	var db qrzDB
	if err := xml.Unmarshal([]byte(resp), &db); err != nil {
		t.Fatalf("unmarshal login: %v", err)
	}
	if db.Session.SessionID != "abc123" {
		t.Errorf("session_id = %q, want abc123", db.Session.SessionID)
	}
	if db.Session.ErrorCode != 0 {
		t.Errorf("errorcode = %d, want 0", db.Session.ErrorCode)
	}
}

func TestXMLParsing_LoginError(t *testing.T) {
	resp := `<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id></session_id>
  <errorcode>1</errorcode>
  <error>Invalid username or password</error>
 </Session>
</QRZDatabase>`

	var db qrzDB
	if err := xml.Unmarshal([]byte(resp), &db); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if db.Session.ErrorCode != 1 {
		t.Errorf("errorcode = %d, want 1", db.Session.ErrorCode)
	}
}

func TestXMLParsing_CallsignLookup(t *testing.T) {
	resp := `<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>abc123</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
 <Callsign>
  <call>RA3ABC</call>
  <name>Ivan</name>
  <surname>Petrov</surname>
  <ename>Ivan</ename>
  <esurname>Petrov</esurname>
  <city>Moscow</city>
  <country>Russia</country>
  <zip>101000</zip>
  <url>https://www.qrz.ru/db/RA3ABC</url>
 </Callsign>
</QRZDatabase>`

	var db qrzDB
	if err := xml.Unmarshal([]byte(resp), &db); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if db.Callsign.Call != "RA3ABC" {
		t.Errorf("call = %q, want RA3ABC", db.Callsign.Call)
	}
	if db.Callsign.EName != "Ivan" {
		t.Errorf("ename = %q, want Ivan", db.Callsign.EName)
	}
	if db.Callsign.City != "Moscow" {
		t.Errorf("city = %q, want Moscow", db.Callsign.City)
	}
}

func TestXMLParsing_NotFound(t *testing.T) {
	resp := `<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>abc123</session_id>
  <errorcode>0</errorcode>
  <error>not found</error>
 </Session>
 <Callsign>
  <call></call>
 </Callsign>
</QRZDatabase>`

	var db qrzDB
	if err := xml.Unmarshal([]byte(resp), &db); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if db.Session.ErrorCode != 0 {
		t.Errorf("errorcode = %d, want 0", db.Session.ErrorCode)
	}
	if !strings.Contains(db.Session.Error, "not found") {
		t.Errorf("error = %q, should contain 'not found'", db.Session.Error)
	}
}

// =============================================================================
// Name formatting tests
// =============================================================================

func TestLookupResult_NameEN(t *testing.T) {
	call := qrzCallSign{
		Call:     "RA3ABC",
		EName:    "Ivan",
		ESurname: "Petrov",
		City:     "Moscow",
		Country:  "Russia",
	}

	// English name + English surname → "Ivan Petrov"
	name := buildCallName(call)
	if name != "Ivan Petrov" {
		t.Errorf("name = %q, want 'Ivan Petrov'", name)
	}
}

func TestLookupResult_NameNative(t *testing.T) {
	call := qrzCallSign{
		Call:    "RA3ABC",
		Name:    "Иван",
		Surname: "Петров",
		City:    "Москва",
	}

	// No English → fall back to native name
	name := buildCallName(call)
	if name != "Иван Петров" {
		t.Errorf("name = %q, want 'Иван Петров'", name)
	}
}

func TestLookupResult_NameOnly(t *testing.T) {
	call := qrzCallSign{
		Call: "RA3ABC",
		Name: "Иван",
		City: "Москва",
	}
	name := buildCallName(call)
	if name != "Иван" {
		t.Errorf("name = %q, want 'Иван'", name)
	}
}

func TestLookupResult_SurnameOnly(t *testing.T) {
	call := qrzCallSign{
		Call:     "RA3ABC",
		Surname:  "Петров",
		ESurname: "Petrov",
		City:     "Moscow",
	}
	// Only ESurname, no EName → "Petrov"
	name := buildCallName(call)
	if name != "Petrov" {
		t.Errorf("name = %q, want 'Petrov'", name)
	}
}

// =============================================================================
// QTH and image tests
// =============================================================================

func TestBuildQTH_CityOnly(t *testing.T) {
	call := qrzCallSign{City: "Moscow"}
	if got := buildQTH(call); got != "Moscow" {
		t.Errorf("qth = %q, want Moscow", got)
	}
}

func TestBuildQTH_CityAndStreet(t *testing.T) {
	call := qrzCallSign{City: "Moscow", Street: "Tverskaya 1"}
	if got := buildQTH(call); got != "Moscow, Tverskaya 1" {
		t.Errorf("qth = %q, want 'Moscow, Tverskaya 1'", got)
	}
}

func TestBuildQTH_StreetOnly(t *testing.T) {
	call := qrzCallSign{Street: "Tverskaya 1"}
	if got := buildQTH(call); got != "Tverskaya 1" {
		t.Errorf("qth = %q, want 'Tverskaya 1'", got)
	}
}

func TestBuildQTH_Empty(t *testing.T) {
	if got := buildQTH(qrzCallSign{}); got != "" {
		t.Errorf("qth = %q, want empty", got)
	}
}

func TestImageURL_PrefersFiles(t *testing.T) {
	call := qrzCallSign{Image: "https://example.com/img.jpg"}
	files := qrzFiles{Files: []string{"https://static.qrz.su/callbook/abc/def.jpg"}}
	if got := imageURL(call, files); got != "https://static.qrz.su/callbook/abc/def.jpg" {
		t.Errorf("imageURL = %q, want Files[0]", got)
	}
}

func TestImageURL_FallsBackToImage(t *testing.T) {
	call := qrzCallSign{Image: "https://example.com/img.jpg", Photo: "https://example.com/photo.jpg"}
	if got := imageURL(call, qrzFiles{}); got != "https://example.com/img.jpg" {
		t.Errorf("imageURL = %q, want Image field", got)
	}
}

func TestImageURL_FallsBackToPhoto(t *testing.T) {
	call := qrzCallSign{Photo: "https://example.com/photo.jpg"}
	if got := imageURL(call, qrzFiles{}); got != "https://example.com/photo.jpg" {
		t.Errorf("imageURL = %q, want Photo field", got)
	}
}

func TestImageURL_Empty(t *testing.T) {
	if got := imageURL(qrzCallSign{}, qrzFiles{}); got != "" {
		t.Errorf("imageURL = %q, want empty", got)
	}
}

// =============================================================================
// Client tests with httptest.Server
// =============================================================================

func TestClient_LookupSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/login") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess123</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
</QRZDatabase>`))
			return
		}
		if strings.Contains(r.URL.Path, "/callsign") {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess123</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
 <Callsign>
  <call>RA3ABC</call>
  <name>Иван</name>
  <surname>Петров</surname>
  <ename>Ivan</ename>
  <esurname>Petrov</esurname>
  <city>Moscow</city>
  <street>Tverskaya 1</street>
  <country>Россия</country>
  <zip>101000</zip>
  <url>https://www.qrz.ru/db/RA3ABC</url>
  <qthloc>KO85</qthloc>
  <latitude>55.75</latitude>
  <longitude>37.62</longitude>
  <class>1</class>
  <is_lotw>Y</is_lotw>
  <is_eqsl>N</is_eqsl>
 </Callsign>
 <Files>
  <file>https://static.qrz.su/callbook/abc/photo.jpg</file>
 </Files>
</QRZDatabase>`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	c := NewClientWithPriority("testuser", "testpass", 35)
	// Override HTTP with our test server.
	c.httpFn = func(rawURL string) ([]byte, error) {
		// Rewrite the URL to use our test server.
		u := strings.Replace(rawURL, "https://api.qrz.ru", srv.URL, 1)
		return defaultHTTPGet(u)
	}

	// Test connection — should cache the session.
	if err := c.TestConnection(); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}

	res, err := c.Lookup("RA3ABC")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Callsign != "RA3ABC" {
		t.Errorf("callsign = %q, want RA3ABC", res.Callsign)
	}
	if res.Name != "Ivan Petrov" {
		t.Errorf("name = %q, want 'Ivan Petrov'", res.Name)
	}
	if res.QTH != "Moscow, Tverskaya 1" {
		t.Errorf("qth = %q, want 'Moscow, Tverskaya 1'", res.QTH)
	}
	// Country intentionally left empty — QRZ.RU returns Cyrillic names;
	// CTY provider fills the English entity name.
	if res.Country != "" {
		t.Errorf("country = %q, want empty (CTY fills English name)", res.Country)
	}
	if res.ImageURL != "https://static.qrz.su/callbook/abc/photo.jpg" {
		t.Errorf("imageURL = %q, want 'https://static.qrz.su/callbook/abc/photo.jpg'", res.ImageURL)
	}
	if res.Grid != "KO85" {
		t.Errorf("grid = %q, want KO85", res.Grid)
	}
	if res.Lat != "55.75" {
		t.Errorf("lat = %q, want 55.75", res.Lat)
	}
	if res.Lon != "37.62" {
		t.Errorf("lon = %q, want 37.62", res.Lon)
	}
	if res.Class != "1" {
		t.Errorf("class = %q, want 1", res.Class)
	}
	if !res.LoTW {
		t.Error("LoTW should be true")
	}
	if res.EQSL {
		t.Error("eQSL should be false")
	}
	if res.Provider != "qrzru" {
		t.Errorf("provider = %q, want qrzru", res.Provider)
	}
}

func TestClient_LookupNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if strings.Contains(r.URL.Path, "/login") {
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess456</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
</QRZDatabase>`))
			return
		}
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess456</session_id>
  <errorcode>0</errorcode>
  <error>not found</error>
 </Session>
 <Callsign>
  <call></call>
 </Callsign>
</QRZDatabase>`))
	}))
	defer srv.Close()

	c := NewClientWithPriority("testuser", "testpass", 35)
	c.httpFn = func(rawURL string) ([]byte, error) {
		u := strings.Replace(rawURL, "https://api.qrz.ru", srv.URL, 1)
		return defaultHTTPGet(u)
	}
	if err := c.TestConnection(); err != nil {
		t.Fatalf("TestConnection: %v", err)
	}

	res, err := c.Lookup("ZZ9ZZZ")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil result for not-found callsign, got %+v", res)
	}
}

func TestClient_LookupEmptyCallsign(t *testing.T) {
	c := NewClientWithPriority("user", "pass", 35)
	res, err := c.Lookup("")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if res != nil {
		t.Error("expected nil result for empty callsign")
	}
}

func TestClient_LookupNoCredentials(t *testing.T) {
	c := NewClientWithPriority("", "", 35)
	res, err := c.Lookup("RA3ABC")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if res != nil {
		t.Error("expected nil result when user is empty")
	}
}

func TestClient_TestConnectionNoCredentials(t *testing.T) {
	c := NewClientWithPriority("", "", 35)
	err := c.TestConnection()
	if err == nil {
		t.Error("expected error for empty credentials")
	}
}

func TestClient_Priority(t *testing.T) {
	c := NewClientWithPriority("user", "pass", 35)
	if c.Priority() != 35 {
		t.Errorf("priority = %d, want 35", c.Priority())
	}
}

func TestClient_Name(t *testing.T) {
	c := NewClientWithPriority("user", "pass", 35)
	if c.Name() != "QRZ.RU" {
		t.Errorf("name = %q, want QRZ.RU", c.Name())
	}
}

func TestClient_SessionReuse(t *testing.T) {
	loginCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		if strings.Contains(r.URL.Path, "/login") {
			loginCount++
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess-reuse</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
</QRZDatabase>`))
			return
		}
		w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<QRZDatabase version="1.37" xmlns="http://api.qrz.ru/namespace">
 <Session>
  <session_id>sess-reuse</session_id>
  <errorcode>0</errorcode>
  <error>OK</error>
 </Session>
 <Callsign>
  <call>RA3ABC</call>
  <ename>Ivan</ename>
  <esurname>Petrov</esurname>
  <city>Moscow</city>
  <country>Russia</country>
 </Callsign>
</QRZDatabase>`))
	}))
	defer srv.Close()

	c := NewClientWithPriority("testuser", "testpass", 35)
	c.httpFn = func(rawURL string) ([]byte, error) {
		u := strings.Replace(rawURL, "https://api.qrz.ru", srv.URL, 1)
		return defaultHTTPGet(u)
	}

	// First lookup logs in.
	if _, err := c.Lookup("RA3ABC"); err != nil {
		t.Fatalf("first lookup: %v", err)
	}
	login1 := loginCount

	// Second lookup should reuse cached session.
	if _, err := c.Lookup("RA3ABC"); err != nil {
		t.Fatalf("second lookup: %v", err)
	}
	if loginCount != login1 {
		t.Errorf("expected %d logins, got %d (session should be reused)", login1, loginCount)
	}
}
