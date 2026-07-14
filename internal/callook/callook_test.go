package callook

import (
	"encoding/json"
	"testing"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient()
	if c.Name() != "Callook.info" {
		t.Errorf("Name() = %q, want Callook.info", c.Name())
	}
	if c.Priority() != 40 {
		t.Errorf("Priority() = %d, want 40", c.Priority())
	}
}

func TestNewClientWithPriority(t *testing.T) {
	c := NewClientWithPriority(30)
	if c.Priority() != 30 {
		t.Errorf("Priority() = %d, want 30", c.Priority())
	}
}

func TestLookupEmptyCall(t *testing.T) {
	c := NewClient()
	res, err := c.Lookup("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if res != nil {
		t.Error("expected nil result for empty callsign")
	}
}

func TestLookupSuccess(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		resp := callookResponse{
			Status: "VALID",
			Type:   "PERSON",
			Name:   "JONES, ARTHUR E",
			Current: struct {
				Callsign  string `json:"callsign"`
				OperClass string `json:"operClass"`
			}{Callsign: "W1AW", OperClass: "EXTRA"},
			Address: struct {
				Line1 string `json:"line1"`
				Line2 string `json:"line2"`
			}{Line1: "225 MAIN ST", Line2: "NEWINGTON, CT 06111"},
			Location: struct {
				Latitude   string `json:"latitude"`
				Longitude  string `json:"longitude"`
				Gridsquare string `json:"gridsquare"`
			}{Latitude: "41.714776", Longitude: "-72.726744", Gridsquare: "FN31pr"},
		}
		b, _ := json.Marshal(resp)
		return b, nil
	})
	defer restore()

	c := NewClient()
	res, err := c.Lookup("W1AW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Provider != "callook" {
		t.Errorf("Provider = %q, want callook", res.Provider)
	}
	if res.Callsign != "W1AW" {
		t.Errorf("Callsign = %q", res.Callsign)
	}
	if res.Name != "JONES, ARTHUR E" {
		t.Errorf("Name = %q", res.Name)
	}
	if res.Grid != "FN31pr" {
		t.Errorf("Grid = %q", res.Grid)
	}
	if res.QTH != "NEWINGTON, CT 06111" {
		t.Errorf("QTH = %q", res.QTH)
	}
	if res.Class != "EXTRA" {
		t.Errorf("Class = %q", res.Class)
	}
}

func TestLookupInvalid(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return []byte(`{"status":"INVALID"}`), nil
	})
	defer restore()

	c := NewClient()
	res, err := c.Lookup("ZZ0ZZZ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res != nil {
		t.Error("expected nil result for INVALID status")
	}
}

func TestLookupNotFound(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return nil, nil // simulate 404
	})
	defer restore()

	// 404 will cause httpFn to return error
	// Use a different approach: return HTTP 404 error
	restore2 := SetHTTPFn(func(rawURL string) ([]byte, error) {
		return nil, nil // this will cause nil data, then JSON parse error
	})
	defer restore2()

	c := NewClient()
	_, err := c.Lookup("ZZ0ZZZ")
	if err == nil {
		t.Error("expected error for bad response")
	}
}

func TestTestConnection(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		resp := callookResponse{Status: "VALID", Current: struct {
			Callsign  string `json:"callsign"`
			OperClass string `json:"operClass"`
		}{Callsign: "W1AW"}}
		b, _ := json.Marshal(resp)
		return b, nil
	})
	defer restore()

	c := NewClient()
	if err := c.TestConnection(); err != nil {
		t.Errorf("TestConnection failed: %v", err)
	}
}

func TestLookupClub(t *testing.T) {
	restore := SetHTTPFn(func(rawURL string) ([]byte, error) {
		resp := callookResponse{
			Status: "VALID",
			Type:   "CLUB",
			Name:   "ARRL HQ OPERATORS CLUB",
			Current: struct {
				Callsign  string `json:"callsign"`
				OperClass string `json:"operClass"`
			}{Callsign: "W1AW"},
		}
		b, _ := json.Marshal(resp)
		return b, nil
	})
	defer restore()

	c := NewClient()
	res, err := c.Lookup("W1AW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if res.Callsign != "W1AW" {
		t.Errorf("Callsign = %q", res.Callsign)
	}
}
