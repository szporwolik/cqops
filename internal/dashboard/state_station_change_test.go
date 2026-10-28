package dashboard

import (
	"testing"
	"time"
)

// =============================================================================
// Station change detection
//
// StationInfo carries AprsPos as a *GeoPoint (so "no APRS position" stays
// distinct from a valid 0,0), and the TUI builds a fresh GeoPoint on every
// push. `a != b` on the structs therefore compared the POINTER: while APRS was
// enabled and a position resolved, every two-second push published an unchanged
// Station event and the browser rebuilt its map layers on each one.
// =============================================================================

// stationEventWindow is how long a "nothing was published" assertion waits.
// The hub delivers synchronously, so the wait only pads for a false negative.
const stationEventWindow = 50 * time.Millisecond

// nextStationEvent returns the next published event when it is a station one.
// reported=false means nothing was published within the window.
func nextStationEvent(t *testing.T, ch chan Event) (reported bool) {
	t.Helper()
	select {
	case ev := <-ch:
		if ev.Type != string(EventStation) {
			t.Fatalf("published %s, want %s", ev.Type, EventStation)
		}
		if _, ok := ev.Payload.(StationInfo); !ok {
			t.Fatalf("station event payload is %T, want StationInfo", ev.Payload)
		}
		return true
	case <-time.After(stationEventWindow):
		return false
	}
}

// TestSetStationPublishesOnlyRealChanges is the gate for the broadcast storm:
// identical station data (built the way the TUI builds it, with a fresh
// GeoPoint) must stay quiet, while a moved position, an appeared/disappeared
// position and every other station field must still publish.
func TestSetStationPublishesOnlyRealChanges(t *testing.T) {
	hub := NewHub()
	state := NewState(hub)
	ch := hub.Subscribe()
	defer hub.Unsubscribe(ch)

	// A new GeoPoint every time — the same way pushDashboardState builds it.
	pos := func(lat, lon float64) *GeoPoint { return &GeoPoint{Lat: lat, Lon: lon} }
	station := func(aprsPos *GeoPoint) StationInfo {
		return StationInfo{
			Callsign:    "SP9SPM",
			Locator:     "KO00CA",
			Lat:         50.06,
			Lon:         19.94,
			Radio:       "FT-891",
			Antenna:     "Dipole",
			PowerW:      100,
			AprsLocator: "JO90WE",
			AprsPos:     aprsPos,
		}
	}

	publish := func(t *testing.T, name string, info StationInfo) {
		t.Helper()
		state.SetStation(info)
		if !nextStationEvent(t, ch) {
			t.Errorf("%s: no station event published", name)
		}
	}
	quiet := func(t *testing.T, name string, info StationInfo) {
		t.Helper()
		state.SetStation(info)
		if nextStationEvent(t, ch) {
			t.Errorf("%s: an unchanged station was published again", name)
		}
	}

	// The first value has to reach a browser that has seen nothing yet.
	publish(t, "first value", station(pos(50.06, 19.94)))

	// The reported case: the regular two-second push, same numbers.
	quiet(t, "same coordinates in a new GeoPoint", station(pos(50.06, 19.94)))
	quiet(t, "same coordinates again", station(pos(50.06, 19.94)))

	// A moved position is a change.
	publish(t, "moved latitude", station(pos(50.07, 19.94)))
	publish(t, "moved longitude", station(pos(50.07, 19.95)))
	quiet(t, "the new position repeated", station(pos(50.07, 19.95)))

	// "No position" is not the same as a position, in either direction.
	publish(t, "position disappeared", station(nil))
	quiet(t, "still no position", station(nil))

	// ...and a valid 0,0 is a real position, not an absent one.
	publish(t, "position reappeared at 0,0", station(pos(0, 0)))
	quiet(t, "0,0 repeated", station(pos(0, 0)))
	publish(t, "0,0 position disappeared", station(nil))
	quiet(t, "nil repeated", station(nil))

	// Everything else still counts, with the position out of the picture.
	changed := station(nil)
	changed.Callsign = "SP9MOA"
	publish(t, "callsign", changed)

	changed = station(nil)
	changed.Locator = "JO90WE"
	publish(t, "locator", changed)

	changed = station(nil)
	changed.PowerW = 50
	publish(t, "power", changed)

	changed = station(nil)
	changed.AprsBeacon = true
	changed.AprsCallsign = "SP9SPM-9"
	changed.AprsSymbol = "/>"
	publish(t, "APRS identity", changed)

	changed = station(nil)
	changed.AprsRadiusKm = 25
	publish(t, "APRS radius", changed)
}

// TestSetStationKeepsTheZeroCoordinateDistinct pins what the value comparison
// must preserve: 0,0 is a real position (the equator/prime meridian crossing)
// and compares equal to itself across pointers, but never to "no position".
func TestSetStationKeepsTheZeroCoordinateDistinct(t *testing.T) {
	state := NewState(NewHub())

	state.SetStation(StationInfo{Callsign: "SP9SPM", AprsPos: &GeoPoint{Lat: 0, Lon: 0}})
	got := state.Snapshot().Station.AprsPos
	if got == nil || got.Lat != 0 || got.Lon != 0 {
		t.Fatalf("stored APRS position = %v, want 0,0", got)
	}

	if stationInfoEqual(StationInfo{AprsPos: &GeoPoint{Lat: 0, Lon: 0}}, StationInfo{}) {
		t.Error("a 0,0 position compares equal to no position at all")
	}
	if !stationInfoEqual(
		StationInfo{AprsPos: &GeoPoint{Lat: 0, Lon: 0}},
		StationInfo{AprsPos: &GeoPoint{Lat: 0, Lon: 0}},
	) {
		t.Error("two 0,0 positions in different pointers compare unequal")
	}
}
