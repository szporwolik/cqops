package dashboard

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestStationAPRSPositionZeroCoordinates guards the JSON contract for the
// APRS position on the station card.
//
// The first version of the payload used float fields with `omitempty`, which
// dropped a perfectly valid 0,0 position (Gulf of Guinea, the equator/prime
// meridian crossing). The browser then fell back to the logging grid and the
// marker disagreed with the transmitted position again. AprsPos is a pointer
// so "no position" and "position 0,0" stay distinguishable.
func TestStationAPRSPositionZeroCoordinates(t *testing.T) {
	s := NewState(NewHub())
	s.SetStation(StationInfo{
		Callsign:    "SP9SPM",
		Locator:     "KO00CA",
		AprsLocator: "JJ00AA00AA",
		AprsPos:     &GeoPoint{Lat: 0, Lon: 0},
	})

	raw, err := json.Marshal(s.Snapshot().Station)
	if err != nil {
		t.Fatalf("marshal station: %v", err)
	}
	if !strings.Contains(string(raw), `"aprsPos":{"lat":0,"lon":0}`) {
		t.Errorf("zero-coordinate APRS position was dropped from the payload: %s", raw)
	}
}

// TestStationAPRSPositionOmittedWhenUnset keeps "no APRS position" out of the
// payload instead of publishing a fake 0,0 point at the map's origin.
func TestStationAPRSPositionOmittedWhenUnset(t *testing.T) {
	s := NewState(NewHub())
	s.SetStation(StationInfo{Callsign: "SP9SPM", Locator: "KO00CA"})

	raw, err := json.Marshal(s.Snapshot().Station)
	if err != nil {
		t.Fatalf("marshal station: %v", err)
	}
	if strings.Contains(string(raw), "aprsPos") {
		t.Errorf("unset APRS position must be omitted, got: %s", raw)
	}
}
