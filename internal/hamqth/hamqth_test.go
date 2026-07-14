package hamqth

import (
	"encoding/xml"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("user", "pass")
	if c.Name() != "HamQTH" {
		t.Errorf("Name() = %q, want HamQTH", c.Name())
	}
	if c.Priority() != 45 {
		t.Errorf("Priority() = %d, want 45", c.Priority())
	}
}

func TestNewClientWithPriority(t *testing.T) {
	c := NewClientWithPriority("u", "p", 30)
	if c.Priority() != 30 {
		t.Errorf("Priority() = %d, want 30", c.Priority())
	}
}

func TestLookupEmptyCall(t *testing.T) {
	c := NewClient("user", "pass")
	res, err := c.Lookup("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != nil {
		t.Error("expected nil result for empty callsign")
	}
}

func TestLookupEmptyUser(t *testing.T) {
	c := NewClient("", "pass")
	res, err := c.Lookup("SP9MOA")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != nil {
		t.Error("expected nil result when user is empty")
	}
}

func TestSearchSuccess(t *testing.T) {
	// Fake HTTP transport that returns a valid search XML.
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		xmlResp := `<HamQTH><search>
			<callsign>SP9MOA</callsign>
			<nick>John</nick>
			<qth>Krakow</qth>
			<grid>JO90</grid>
			<country>Poland</country>
			<itu>28</itu>
			<cq>15</cq>
			<latitude>50.0</latitude>
			<longitude>19.9</longitude>
			<adif>269</adif>
		</search></HamQTH>`
		return []byte(xmlResp), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	// Bypass auth by injecting a fake session.
	c.sID = "fake-session"
	c.sUser = "user"
	c.sPass = "pass"
	c.sAt = time.Now()

	sd, err := c.lookup("SP9MOA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sd == nil {
		t.Fatal("expected non-nil result")
	}
	if sd.Callsign != "SP9MOA" {
		t.Errorf("Callsign = %q, want SP9MOA", sd.Callsign)
	}
	if sd.Name != "John" {
		t.Errorf("Name = %q, want John", sd.Name)
	}
	if sd.QTH != "Krakow" {
		t.Errorf("QTH = %q, want Krakow", sd.QTH)
	}
	if sd.Grid != "JO90" {
		t.Errorf("Grid = %q, want JO90", sd.Grid)
	}
	if sd.Country != "Poland" {
		t.Errorf("Country = %q, want Poland", sd.Country)
	}
	if sd.DXCC != "269" {
		t.Errorf("DXCC = %q, want 269", sd.DXCC)
	}
}

func TestSearchNotFound(t *testing.T) {
	// HamQTH returns empty search XML when not found.
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`<HamQTH><search></search></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	c.sID = "fake-session"
	c.sUser = "user"
	c.sPass = "pass"
	c.sAt = time.Now()

	sd, err := c.lookup("ZZ0ZZZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sd != nil {
		t.Error("expected nil result for not-found callsign")
	}
}

func TestSearchSessionError(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`<HamQTH><session><error>Session does not exist or expired</error></session></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	c.sID = "expired-session"
	c.sUser = "user"
	c.sPass = "pass"
	c.sAt = time.Now()

	sd, err := c.lookup("SP9MOA")
	if err == nil {
		t.Fatal("expected error for expired session")
	}
	if sd != nil {
		t.Error("expected nil result for session error")
	}
}

func TestAuthSuccess(t *testing.T) {
	var capturedURL string
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		capturedURL = rawURL
		// First call: auth; second call: search (both use same transport).
		if rawURL == "https://www.hamqth.com/xml.php?u=user&p=pass" {
			return []byte(`<HamQTH><session><session_id>abc123</session_id></session></HamQTH>`), nil
		}
		return []byte(`<HamQTH><search><callsign>SP9MOA</callsign><nick>Test</nick></search></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	sd, err := c.lookup("SP9MOA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sd == nil {
		t.Fatal("expected non-nil result")
	}
	if capturedURL == "" {
		t.Error("expected HTTP call to be made")
	}

	// Verify session was cached.
	if c.sID != "abc123" {
		t.Errorf("session ID = %q, want abc123", c.sID)
	}
}

func TestAuthError(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`<HamQTH><session><error>Wrong user name or password</error></session></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "wrongpass")
	sd, err := c.lookup("SP9MOA")
	if err == nil {
		t.Fatal("expected error for bad credentials")
	}
	if sd != nil {
		t.Error("expected nil result for auth error")
	}
}

func TestLookupResult(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`<HamQTH><search>
			<callsign>SP9MOA</callsign>
			<nick>John</nick>
			<grid>JO90wa</grid>
			<country>Poland</country>
			<qth>Krakow</qth>
		</search></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	c.sID = "s"
	c.sUser = "user"
	c.sPass = "pass"
	c.sAt = time.Now()

	res, err := c.Lookup("SP9MOA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Provider != "hamqth" {
		t.Errorf("Provider = %q, want hamqth", res.Provider)
	}
	if res.Callsign != "SP9MOA" {
		t.Errorf("Callsign = %q, want SP9MOA", res.Callsign)
	}
	if res.Name != "John" {
		t.Errorf("Name = %q, want John", res.Name)
	}
}

func TestAdrNameFallback(t *testing.T) {
	// When <nick> is empty, use <adr_name> instead.
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`<HamQTH><search>
			<callsign>SP9MOA</callsign>
			<nick></nick>
			<adr_name>John Doe</adr_name>
			<qth></qth>
			<adr_city>Krakow</adr_city>
		</search></HamQTH>`), nil
	})
	defer restore()

	c := NewClient("user", "pass")
	c.sID = "s"
	c.sUser = "user"
	c.sPass = "pass"
	c.sAt = time.Now()

	res, err := c.Lookup("SP9MOA")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Name != "John Doe" {
		t.Errorf("Name = %q, want John Doe (adr_name fallback)", res.Name)
	}
	if res.QTH != "Krakow" {
		t.Errorf("QTH = %q, want Krakow (adr_city fallback)", res.QTH)
	}
}

func TestXMLUnmarshalFull(t *testing.T) {
	xmlData := `<search>
		<callsign>SP9MOA</callsign>
		<nick>John</nick>
		<qth>Krakow</qth>
		<grid>JO90wa</grid>
		<country>Poland</country>
		<adr_name>John Doe</adr_name>
		<adr_street1>Main St 1</adr_street1>
		<adr_city>Krakow</adr_city>
		<adr_zip>30-001</adr_zip>
		<itu>28</itu>
		<cq>15</cq>
		<latitude>50.0619</latitude>
		<longitude>19.9369</longitude>
		<adif>269</adif>
		<picture>https://example.com/img.jpg</picture>
	</search>`

	var s hamqthSearch
	if err := xml.Unmarshal([]byte(xmlData), &s); err != nil {
		t.Fatalf("xml unmarshal: %v", err)
	}

	sd := toSearchData(&s)
	if sd.Callsign != "SP9MOA" {
		t.Errorf("Callsign = %q", sd.Callsign)
	}
	if sd.Name != "John" {
		t.Errorf("Name = %q, want John (nick takes priority over adr_name)", sd.Name)
	}
	if sd.Grid != "JO90wa" {
		t.Errorf("Grid = %q", sd.Grid)
	}
	if sd.Country != "Poland" {
		t.Errorf("Country = %q", sd.Country)
	}
	if sd.QTH != "Krakow" {
		t.Errorf("QTH = %q", sd.QTH)
	}
	if sd.ITUZone != "28" {
		t.Errorf("ITUZone = %q", sd.ITUZone)
	}
	if sd.CQZone != "15" {
		t.Errorf("CQZone = %q", sd.CQZone)
	}
	if sd.Lat != "50.06190" {
		t.Errorf("Lat = %q", sd.Lat)
	}
	if sd.Lon != "19.93690" {
		t.Errorf("Lon = %q", sd.Lon)
	}
}
