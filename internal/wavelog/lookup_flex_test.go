package wavelog

import (
	"encoding/json"
	"testing"
)

// A real Wavelog answers the lookup block with `"dxcc_id": 287` even though
// the API documents it as a string. Decoding that into a plain string failed
// the whole response — the provider returned nothing and the callbook chain
// still paid its timeout. Numeric and quoted values must both decode.
func TestLookupDataAcceptsNumericFields(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"numbers", `{"callsign":"HB9H","dxcc":"SWITZERLAND","dxcc_id":287,"dxcc_cqz":28,"dxcc_lat":46.1,"dxcc_long":8.86,"bearing":120}`},
		{"quoted", `{"callsign":"HB9H","dxcc":"SWITZERLAND","dxcc_id":"287","dxcc_cqz":"28","bearing":"120"}`},
		{"nulls", `{"callsign":"HB9H","dxcc_id":null,"dxcc_cqz":null}`},
		{"floats", `{"callsign":"HB9H","dxcc_lat":46.104408,"dxcc_long":-8.5}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d lookupData
			if err := json.Unmarshal([]byte(tc.body), &d); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			r := newLookupResult(&d)
			if r.Callsign() != "HB9H" {
				t.Errorf("callsign = %q", r.Callsign())
			}
			if tc.name == "numbers" || tc.name == "quoted" {
				if r.DXCCID() != "287" {
					t.Errorf("dxcc id = %q, want 287", r.DXCCID())
				}
				if r.CQZone() != "28" {
					t.Errorf("cq zone = %q, want 28", r.CQZone())
				}
			}
		})
	}
}

func TestFlexStringKeepsText(t *testing.T) {
	var d lookupData
	if err := json.Unmarshal([]byte(`{"dxcc_id":269,"dxcc":"POLAND","bearing":"045"}`), &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.DXCCID.String() != "269" || d.DXCC.String() != "POLAND" || d.Bearing.String() != "045" {
		t.Errorf("got %q/%q/%q", d.DXCCID, d.DXCC, d.Bearing)
	}
}
