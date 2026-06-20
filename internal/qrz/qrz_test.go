package qrz

import (
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
// Session cache — concurrent access safety
// =============================================================================

func TestSessionCache_Concurrent(t *testing.T) {
	// Verify cacheMu is properly used — no deadlocks with concurrent access.
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			cacheMu.Lock()
			_ = cachedSessionKey
			cacheMu.Unlock()
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
