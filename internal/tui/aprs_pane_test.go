package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/aprs"
	"github.com/szporwolik/cqops/internal/config"
)

// newTestAPRSCache opens a temporary APRS cache database and seeds it with
// the given stations. Caller must close it.
func newTestAPRSCache(t *testing.T, stations []aprs.StationRecord) *aprs.CacheDB {
	t.Helper()
	cache, err := aprs.OpenCacheDB(filepath.Join(t.TempDir(), "aprs.db"))
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() { cache.Close() })
	for _, s := range stations {
		if err := cache.UpsertStation(s); err != nil {
			t.Fatalf("upsert %s: %v", s.Callsign, err)
		}
	}
	return cache
}

// testAPRSRecord builds a station record at the given offset from JO90 center.
func testAPRSRecord(call string, dLat, dLon float64) aprs.StationRecord {
	stLat, stLon := gridToLatLon("JO90")
	return aprs.StationRecord{
		Callsign:  call,
		Lat:       stLat + dLat,
		Lon:       stLon + dLon,
		Comment:   "test " + call,
		Symbol:    "/-", // house — operator station, passes the type filter
		Source:    "aprs_is",
		LastHeard: time.Now(),
	}
}

func TestAPRSPaneRefresh_FilterSort(t *testing.T) {
	records := []aprs.StationRecord{
		testAPRSRecord("NEAR1", 0.005, 0),
		testAPRSRecord("FAR01", 0.100, 0),
		testAPRSRecord("MID01", 0.030, 0),
		testAPRSRecord("OLD01", 0.001, 0),
	}
	// OLD01 is stale — must be dropped by the 60-minute cutoff.
	records[3].LastHeard = time.Now().Add(-2 * time.Hour)
	// FAR01 is beyond the radius — must be dropped by the filter.
	records[1].Lon += 5 // ~far away

	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 15}

	m.aprsPaneRefresh()

	if len(m.aprsPane.stations) != 2 {
		t.Fatalf("expected 2 stations after filtering, got %d", len(m.aprsPane.stations))
	}
	if m.aprsPane.stations[0].rec.Callsign != "NEAR1" {
		t.Errorf("expected NEAR1 first (closest), got %s", m.aprsPane.stations[0].rec.Callsign)
	}
	if m.aprsPane.stations[1].rec.Callsign != "MID01" {
		t.Errorf("expected MID01 second, got %s", m.aprsPane.stations[1].rec.Callsign)
	}
	if m.aprsPane.stations[0].distKm >= m.aprsPane.stations[1].distKm {
		t.Error("distance order not ascending")
	}
	if got := m.aprsPane.stations[0].grid; len(got) != 6 {
		t.Errorf("expected 6-character grid, got %q", got)
	}
}

func TestAPRSPaneRefresh_NoCache(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{{rec: aprs.StationRecord{Callsign: "X"}}}
	m.aprsPaneRefresh()
	if len(m.aprsPane.stations) != 0 {
		t.Errorf("expected empty list without cache, got %d", len(m.aprsPane.stations))
	}
}

func TestAPRSPaneRefresh_NoStationGrid(t *testing.T) {
	// Without a station grid no distance is computable — all stations
	// within the cutoff are kept, sorted stably.
	m := newTestModel()
	m.App.Logbook.Station.Grid = ""
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{
		testAPRSRecord("AAA01", 0, 0),
		testAPRSRecord("BBB01", 0.01, 0),
	})
	m.aprsPaneRefresh()
	if len(m.aprsPane.stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(m.aprsPane.stations))
	}
	if m.aprsPane.stations[0].distKm != 0 || m.aprsPane.stations[0].bearing != 0 {
		t.Error("distance/bearing must be zero without station grid")
	}
}

func TestAPRSPaneSelect_Clamps(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "A1"}},
		{rec: aprs.StationRecord{Callsign: "B1"}},
		{rec: aprs.StationRecord{Callsign: "C1"}},
	}
	m.aprsPaneSelect(-5)
	if m.aprsPane.sel != 0 {
		t.Errorf("clamp to 0 failed, sel=%d", m.aprsPane.sel)
	}
	m.aprsPaneSelect(99)
	if m.aprsPane.sel != 2 {
		t.Errorf("clamp to last failed, sel=%d", m.aprsPane.sel)
	}
	m.aprsPane.stations = nil
	m.aprsPaneSelect(0)
	if m.aprsPane.sel != 0 {
		t.Errorf("empty select failed, sel=%d", m.aprsPane.sel)
	}
}

func TestAPRSPaneFillFromSelected(t *testing.T) {
	m := newTestModel()
	call := "SP9XYZ"
	rec := testAPRSRecord(call+"-10", 0.005, 0) // APRS SSID must be stripped
	grid := latLonToGrid(rec.Lat, rec.Lon, 6)
	m.aprsPane.stations = []aprsStation{
		{rec: rec, distKm: 0.5, bearing: 90, grid: grid},
	}
	m.aprsPaneSelect(0)

	m.aprsFillFromSelected()
	if got := m.fields[fieldCall].Value(); got != call {
		t.Errorf("call field = %q, want %q", got, call)
	}
	if got := m.fields[fieldGrid].Value(); got != grid {
		t.Errorf("grid field = %q, want %q", got, grid)
	}
	if got := m.fields[fieldName].Value(); got != "" {
		t.Errorf("name field should be cleared, got %q", got)
	}
}

func TestAPRSPaneKeys_Navigation(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "A1"}},
		{rec: aprs.StationRecord{Callsign: "B1"}},
		{rec: aprs.StationRecord{Callsign: "C1"}},
	}
	m.buildAPRSTable(40, 8)
	m.aprsSyncSelection()

	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyDown}, nil)
	if m.aprsPane.sel != 1 {
		t.Errorf("down: sel=%d, want 1", m.aprsPane.sel)
	}
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyEnd}, nil)
	if m.aprsPane.sel != 2 {
		t.Errorf("end: sel=%d, want 2", m.aprsPane.sel)
	}
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyUp}, nil)
	if m.aprsPane.sel != 1 {
		t.Errorf("up: sel=%d, want 1", m.aprsPane.sel)
	}
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyHome}, nil)
	if m.aprsPane.sel != 0 {
		t.Errorf("home: sel=%d, want 0", m.aprsPane.sel)
	}
	// Enter fills the form and jumps back to the QSO screen.
	m.screen = screenAPRS
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyEnter}, nil)
	if m.screen != screenQSO {
		t.Errorf("enter should return to QSO screen, screen=%d", m.screen)
	}
	if m.fields[fieldCall].Value() != "A1" {
		t.Errorf("enter should fill call, got %q", m.fields[fieldCall].Value())
	}
	// Esc returns to QSO without filling.
	m.screen = screenAPRS
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyEscape}, nil)
	if m.screen != screenQSO {
		t.Errorf("esc should return to QSO screen, screen=%d", m.screen)
	}
}

func TestAPRSPaneView_States(t *testing.T) {
	lay := Layout{TerminalW: 100, ContentW: 98, ContentH: 20}

	// No grid — nothing but the hint is rendered.
	m := newTestModel()
	m.App.Logbook.Station.Grid = ""
	v := m.viewAPRS(lay)
	if !strings.Contains(v, "grid") {
		t.Errorf("no-grid view missing hint: %q", v)
	}
	if strings.Contains(v, "Distance all") {
		t.Errorf("no-grid view should not render the station list: %q", v)
	}

	// No cache — hint message, own-station panel still on the right.
	m = newTestModel()
	v = m.viewAPRS(lay)
	if !strings.Contains(v, "not receiving") {
		t.Errorf("no-cache view missing hint: %q", v)
	}
	if !strings.Contains(v, "APRS-RX") {
		t.Errorf("no-cache view should keep the own-station panel: %q", v)
	}

	// Empty list — empty-state message plus the own-station panel.
	m = newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, nil)
	m.aprsPaneRefresh()
	v = m.viewAPRS(lay)
	if !strings.Contains(v, "No nearby stations") {
		t.Errorf("empty view missing hint: %q", v)
	}
	if !strings.Contains(v, "APRS-RX") {
		t.Errorf("empty view should keep the own-station panel: %q", v)
	}

	// Empty list with TX configured — the beacon status stays visible.
	m = newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, nil)
	m.App.Logbook.APRS = &config.APRSConfig{
		Enabled:      true,
		SendLocation: true,
		Callsign:     "SP9MOA-7",
		Symbol:       "/-",
		IntervalMin:  15,
	}
	m.aprsPaneRefresh()
	v = m.viewAPRS(lay)
	if !strings.Contains(v, "No nearby stations") {
		t.Errorf("empty TX view missing hint: %q", v)
	}
	if !strings.Contains(v, "Transmitting") || !strings.Contains(v, "SP9MOA-7") {
		t.Errorf("empty TX view should keep the beacon panel: %q", v)
	}

	// Populated — list shows callsign and details show fields.
	m = newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{
		testAPRSRecord("SP9XYZ", 0.005, 0),
	})
	m.aprsPaneRefresh()
	v = m.viewAPRS(lay)
	if !strings.Contains(v, "SP9XYZ") {
		t.Errorf("populated view missing callsign: %q", v)
	}
	// Bearing is a table column; details carry no distance/bearing.
	if !strings.Contains(v, "000\u00b0") {
		t.Errorf("populated view missing bearing column: %q", v)
	}
}

// Emoji and decorative symbols in APRS comments must not reach the
// terminal — they are stripped before display.
func TestAPRSDetailRows_CommentEmojiStripped(t *testing.T) {
	m := newTestModel()
	sel := &aprsStation{
		rec: aprs.StationRecord{
			Callsign: "SP9ABC",
			Comment:  "73 \u2600\ufe0f GL! \U0001F600",
		},
		grid: "JO90AA",
	}
	rows := m.aprsDetailRows(sel, 46)
	joined := strings.Join(rows, "\n")
	for _, bad := range []string{"\u2600", "\U0001F600"} {
		if strings.Contains(joined, bad) {
			t.Errorf("emoji leaked into the comment row: %v", rows)
		}
	}
	if !strings.Contains(joined, "73 GL!") {
		t.Errorf("sanitized comment text missing: %v", rows)
	}
}

func TestAPRSDetailRows_WeatherComment(t *testing.T) {
	m := newTestModel() // metric units
	sel := &aprsStation{
		rec: aprs.StationRecord{
			Callsign: "SP9WSS-10",
			Symbol:   "/_",
			Comment:  "236/022t075h79b10172",
		},
		grid: "KO00AB",
	}
	rows := m.aprsDetailRows(sel, 46)
	joined := strings.Join(rows, "\n")
	for _, want := range []string{"Weather Station", "24\u00b0C", "79%", "1017 hPa"} {
		if !strings.Contains(joined, want) {
			t.Errorf("weather comment missing %q: %v", want, rows)
		}
	}
	if strings.Contains(joined, "t075") {
		t.Errorf("raw weather block should not appear: %v", rows)
	}
}

func TestAPRSDetailRows_CallLine(t *testing.T) {
	m := newTestModel()
	sel := &aprsStation{
		rec:  aprs.StationRecord{Callsign: "SP9ABC", Symbol: "/-"},
		grid: "JO90AA",
	}
	rows := m.aprsDetailRows(sel, 46)
	joined := strings.Join(rows, "\n")
	// One row: callsign first, then the symbol with its name.
	for _, want := range []string{"SP9ABC", "House"} {
		if !strings.Contains(joined, want) {
			t.Errorf("call line missing %q: %v", want, rows)
		}
	}
	// The symbol no longer has its own row.
	if strings.Contains(joined, "Symbol") {
		t.Errorf("unexpected Symbol row: %v", rows)
	}
}

func TestAPRSDetailRows_NavLine(t *testing.T) {
	m := newTestModel()
	sel := &aprsStation{
		rec: aprs.StationRecord{
			Callsign: "SP9ABC",
			Course:   161,
			SpeedKmH: 0,
		},
		distKm:  24.1,
		bearing: 280,
		grid:    "JO90AA",
	}
	rows := m.aprsDetailRows(sel, 46)
	joined := strings.Join(rows, "\n")
	// Course and speed ride on the grid line — no separate Nav row, so the
	// panel height never shifts when movement reporting appears.
	for _, want := range []string{"crs 161\u00b0", "0 km/h"} {
		if !strings.Contains(joined, want) {
			t.Errorf("grid line missing %q: %v", want, rows)
		}
	}
	for _, row := range rows {
		if strings.Contains(row, "crs 161\u00b0") && !strings.Contains(row, "JO90AA") {
			t.Errorf("course must be on the grid line: %v", rows)
		}
	}
	// Distance and bearing moved to the table — details must not carry
	// them, and there is never a trail row.
	for _, bad := range []string{"24.1 km", "280\u00b0", "Distance", "Bearing", "Course", "Speed", "Nav", "Trail"} {
		if strings.Contains(joined, bad) {
			t.Errorf("unexpected %q row in details: %v", bad, rows)
		}
	}
}

func TestAPRSDetailRows_LastLine(t *testing.T) {
	m := newTestModel()
	sel := &aprsStation{
		rec: aprs.StationRecord{
			Callsign:  "SP9ABC",
			Source:    "aprs_is",
			LastHeard: time.Date(2026, 9, 20, 13, 8, 0, 0, time.UTC),
		},
		grid: "JO90AA",
	}
	rows := m.aprsDetailRows(sel, 46)
	joined := strings.Join(rows, "\n")
	// One row: last heard first, then source.
	for _, want := range []string{"Last", "13:08Z", "aprs_is"} {
		if !strings.Contains(joined, want) {
			t.Errorf("last line missing %q: %v", want, rows)
		}
	}
	for _, bad := range []string{"Source", "Last heard"} {
		if strings.Contains(joined, bad) {
			t.Errorf("unexpected %q row: %v", bad, rows)
		}
	}
}

func TestAPRSDetailRows_NoNavLine(t *testing.T) {
	m := newTestModel()
	sel := &aprsStation{
		rec:     aprs.StationRecord{Callsign: "SP9ABC"},
		distKm:  24.1,
		bearing: 280,
		grid:    "JO90AA",
	}
	rows := m.aprsDetailRows(sel, 46)
	if strings.Contains(strings.Join(rows, "\n"), "Nav") {
		t.Errorf("no Nav row expected without course/speed: %v", rows)
	}
}

func TestAPRSTable_BearingColumn(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "A1"}, distKm: 12.3, bearing: 280},
	}
	m.buildAPRSTable(40, 8)
	v := m.aprsPane.table.View()
	if !strings.Contains(v, "Brg") {
		t.Errorf("table missing Brg column header: %q", v)
	}
	if !strings.Contains(v, "280\u00b0") {
		t.Errorf("table missing bearing value: %q", v)
	}
}

func TestAPRSStatusRows_RX(t *testing.T) {
	m := newTestModel()
	rows := m.aprsStatusRows(40)
	joined := strings.Join(rows, "\n")
	if !strings.Contains(joined, "APRS-RX") || !strings.Contains(joined, "not") {
		t.Errorf("RX status rows wrong: %v", rows)
	}
}

func TestAPRSStatusRows_TX(t *testing.T) {
	m := newTestModel()
	m.App.Config.Integrations.APRS.Enabled = true
	m.App.Logbook.APRS = &config.APRSConfig{
		Enabled:      true,
		SendLocation: true,
		Callsign:     "SP9MOA-10",
		Symbol:       "/-",
		Comment:      "CQOps test",
		IntervalMin:  10,
		LastBeaconAt: time.Now().UTC().Add(-5 * time.Minute).Format(time.RFC3339),
	}
	rows := m.aprsStatusRows(40)
	joined := strings.Join(rows, "\n")
	// Call and symbol share one line — symbol name included.
	for _, want := range []string{"Transmitting", "SP9MOA-10", "/-", "House", "CQOps test", "10 min", "Last TX", "5m ago"} {
		if !strings.Contains(joined, want) {
			t.Errorf("TX status rows missing %q: %v", want, rows)
		}
	}
}

func TestAPRSBeaconShortcut(t *testing.T) {
	m := newTestModel()
	m.screen = screenAPRS
	// Receive-only — the shortcut must warn, never panic.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 'b', Text: "b"}, nil)
	if len(m.toasts.Active()) == 0 {
		t.Error("expected warning toast in receive-only mode")
	}
}

func TestAPRSAgeAgo(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name string
		at   time.Time
		want string
	}{
		{"seconds", now.Add(-7 * time.Second), "less than a minute ago"},
		{"zero", now, "less than a minute ago"},
		{"minutes", now.Add(-5 * time.Minute), "5m ago"},
		{"hours", now.Add(-3 * time.Hour), "3.0h ago"},
	}
	for _, c := range cases {
		if got := aprsAgeAgo(c.at); got != c.want {
			t.Errorf("%s: aprsAgeAgo = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestAPRSRadarRows(t *testing.T) {
	m := newTestModel()
	// Four stations at the cardinal points, two stacked on top of each other.
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "N1"}, distKm: 8, bearing: 0},
		{rec: aprs.StationRecord{Callsign: "E1"}, distKm: 8, bearing: 90},
		{rec: aprs.StationRecord{Callsign: "W1"}, distKm: 8, bearing: 270},
		{rec: aprs.StationRecord{Callsign: "S1"}, distKm: 8, bearing: 180},
		{rec: aprs.StationRecord{Callsign: "S2"}, distKm: 8, bearing: 180},
	}
	m.aprsPane.sel = 1

	rows := m.aprsRadarRows(&m.aprsPane, 21, 10)
	if len(rows) != 10 {
		t.Fatalf("expected 10 radar rows, got %d: %v", len(rows), rows)
	}
	joined := strings.Join(rows, "\n")
	for _, want := range []string{"+", "~", "*"} {
		if !strings.Contains(joined, want) {
			t.Errorf("radar missing %q:\n%s", want, joined)
		}
	}
	// The two southern stations share a cell — count digit 2.
	if !strings.Contains(joined, "2") {
		t.Errorf("stacked stations should render a 2:\n%s", joined)
	}
	// Selected station (E1) is highlighted with the cursor style.
	if !strings.Contains(joined, CursorStyle.Render("*")) {
		t.Errorf("selected station not highlighted:\n%s", joined)
	}

	// Plain geometry: box is 21 wide (inner 19, cx=9) and 10 rows
	// (inner 7, cy=3). N sits at [0], center at [4][10], caption at [9].
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = stripANSI(r)
	}
	// Cardinal labels around the radar restore azimuth context. W / E hug
	// the ring (rh=6): one cell left of it at box 3, one cell right at 17.
	if plain[0][10] != 'N' || plain[8][10] != 'S' {
		t.Errorf("N/S labels misaligned:\n%s", strings.Join(plain, "\n"))
	}
	if plain[4][3] != 'W' || plain[4][17] != 'E' {
		t.Errorf("W/E labels misaligned:\n%s", strings.Join(plain, "\n"))
	}
	// No floating labels at the box edges.
	if plain[4][0] != ' ' || plain[4][20] != ' ' {
		t.Errorf("W/E labels should not float at the box edges:\n%s", strings.Join(plain, "\n"))
	}
	// Roundness: same distance renders 5 cells east/west of center but
	// only 2 rows north/south (terminal cells are ~2:1).
	if plain[4][15] != '*' || plain[4][5] != '*' {
		t.Errorf("E/W stations not on the center row:\n%s", strings.Join(plain, "\n"))
	}
	if plain[2][10] != '*' || plain[6][10] != '2' {
		t.Errorf("N/S stations not vertically compressed:\n%s", strings.Join(plain, "\n"))
	}
	// Single bottom caption with the own grid; at this narrow width the
	// right-aligned selected-station bearing/distance drops.
	if !strings.Contains(plain[9], "10 km") || !strings.Contains(plain[9], "JO90") {
		t.Errorf("caption wrong:\n%s", plain[9])
	}
	if strings.Contains(plain[9], "090\u00b0") {
		t.Errorf("bearing should drop at narrow width:\n%s", plain[9])
	}
}

func TestAPRSRadarRows_CaptionFullGrid(t *testing.T) {
	m := newTestModel()
	m.App.Logbook.Station.Grid = "JO90AB"
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "E1"}, distKm: 8, bearing: 90},
	}
	m.aprsPane.sel = 0
	rows := m.aprsRadarRows(&m.aprsPane, 30, 10)
	caption := stripANSI(rows[len(rows)-1])
	if !strings.Contains(caption, "JO90AB") {
		t.Errorf("caption should show the exact grid sent to APRS:\n%s", caption)
	}
}

func TestAPRSRadarRows_CaptionBearing(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "E1"}, distKm: 8, bearing: 90},
	}
	m.aprsPane.sel = 0
	rows := m.aprsRadarRows(&m.aprsPane, 34, 10)
	caption := stripANSI(rows[len(rows)-1])
	if !strings.Contains(caption, "JO90") {
		t.Errorf("caption should name the center grid:\n%s", caption)
	}
	if !strings.Contains(caption, "E1 090\u00b0") {
		t.Errorf("caption bearing missing when it fits:\n%s", caption)
	}
	if !strings.Contains(caption, "8.0 km") {
		t.Errorf("caption should include the selected station's distance:\n%s", caption)
	}
}

func TestAPRSRadarRows_Minimal(t *testing.T) {
	m := newTestModel()
	if rows := m.aprsRadarRows(&m.aprsPane, 10, 8); len(rows) != 8 {
		t.Errorf("minimal box should render a radar, got %v", rows)
	}
}

// In a wide box the ring does not reach the sides — W / E must hug the
// ring instead of floating at the box edges.
func TestAPRSRadarRows_WideLabelsHugRing(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "E1"}, distKm: 8, bearing: 90},
	}
	// 41x10: innerW=39, cx=19, innerH=7, cy=3, rh=6.
	rows := m.aprsRadarRows(&m.aprsPane, 41, 10)
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = stripANSI(r)
	}
	if plain[4][13] != 'W' || plain[4][27] != 'E' {
		t.Errorf("W/E labels should hug the ring (13/27), got:\n%s", strings.Join(plain, "\n"))
	}
	if plain[4][0] != ' ' || plain[4][40] != ' ' {
		t.Errorf("W/E labels should not float at the box edges:\n%s", strings.Join(plain, "\n"))
	}
}

func TestAPRSRadarRows_TypeMarkers(t *testing.T) {
	m := newTestModel()
	// E and W double as cardinal letters — station markers must keep the
	// value style while the cardinal labels stay dim.
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "E1"}, distKm: 8, bearing: 90, dtype: "E"},
		{rec: aprs.StationRecord{Callsign: "W1"}, distKm: 8, bearing: 270, dtype: "W"},
	}
	m.aprsPane.sel = 1
	rows := m.aprsRadarRows(&m.aprsPane, 21, 10)
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = stripANSI(r)
	}
	// Single stations render their Yaesu-style type markers.
	if plain[4][15] != 'E' || plain[4][5] != 'W' {
		t.Errorf("type markers not rendered:\n%s", strings.Join(plain, "\n"))
	}
	// Markers keep the value style; cardinal labels stay dim.
	if !strings.Contains(rows[4], ValueStyle.Render("E")) || !strings.Contains(rows[4], DimStyle.Render("W")) {
		t.Errorf("unselected marker should use value, cardinal should stay dim:\n%s", strings.Join(rows, "\n"))
	}
	if !strings.Contains(rows[4], DimStyle.Render("E")) {
		t.Errorf("cardinal E label should stay dim:\n%s", strings.Join(rows, "\n"))
	}
	// The selected marker is highlighted.
	if !strings.Contains(rows[4], CursorStyle.Render("W")) {
		t.Errorf("selected type marker not highlighted:\n%s", strings.Join(rows, "\n"))
	}
}

// An active distance filter zooms the radar: the outer ring and the
// caption reflect the filter radius instead of a fixed 10 km floor.
func TestAPRSRadarRows_DistanceFilterScales(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "NEAR"}, distKm: 0.9, bearing: 90},
		{rec: aprs.StationRecord{Callsign: "FAR"}, distKm: 9, bearing: 270},
	}
	caption := func() string {
		rows := m.aprsRadarRows(&m.aprsPane, 21, 10)
		return stripANSI(rows[len(rows)-1])
	}

	m.aprsPane.distFilter = 1
	if got := caption(); !strings.Contains(got, "~ 1 km") {
		t.Errorf("caption should reflect the 1 km filter: %q", got)
	}
	m.aprsPane.distFilter = 5
	if got := caption(); !strings.Contains(got, "~ 5 km") {
		t.Errorf("caption should reflect the 5 km filter: %q", got)
	}
	// No filter — scale covers the farthest station, at least 10 km.
	m.aprsPane.distFilter = 0
	if got := caption(); !strings.Contains(got, "~ 10 km") {
		t.Errorf("no filter should keep the 10 km floor: %q", got)
	}
	// A filter wider than the visible stations must not upscale the radar.
	m.aprsPane.distFilter = 100
	if got := caption(); !strings.Contains(got, "~ 10 km") {
		t.Errorf("wide filter should not upscale: %q", got)
	}
}

func TestAPRSRadarRows_TooSmall(t *testing.T) {
	m := newTestModel()
	if rows := m.aprsRadarRows(&m.aprsPane, 8, 4); len(rows) != 0 {
		t.Errorf("radar should be empty for tiny boxes, got %v", rows)
	}
}

func TestAPRSRadarRows_NearZero(t *testing.T) {
	m := newTestModel()
	m.aprsPane.stations = []aprsStation{
		{rec: aprs.StationRecord{Callsign: "ME1"}, distKm: 0.05, bearing: 90},
		{rec: aprs.StationRecord{Callsign: "FAR"}, distKm: 5, bearing: 90},
	}
	m.aprsPane.sel = 0
	rows := m.aprsRadarRows(&m.aprsPane, 21, 10)
	plain := make([]string, len(rows))
	for i, r := range rows {
		plain[i] = stripANSI(r)
	}
	// The near-zero station is marked at the center, highlighted when
	// selected, instead of being plotted at the ring edge.
	if plain[4][10] != '+' {
		t.Errorf("near-zero station not at center:\n%s", strings.Join(plain, "\n"))
	}
	if !strings.Contains(rows[4], CursorStyle.Render("+")) {
		t.Errorf("selected near-zero station not highlighted:\n%s", strings.Join(rows, "\n"))
	}
	// The far station still plots east of the center.
	if !strings.Contains(plain[4], "*") {
		t.Errorf("far station missing:\n%s", strings.Join(plain, "\n"))
	}

	// Unselected near-zero stations keep the regular center marker and
	// do not disturb the selection highlight of the far station.
	m2 := newTestModel()
	m2.aprsPane.stations = m.aprsPane.stations
	m2.aprsPane.sel = 1
	rows2 := m2.aprsRadarRows(&m2.aprsPane, 21, 10)
	if !strings.Contains(rows2[4], S.StatusValue.Render("+")) {
		t.Errorf("center marker missing for unselected near-zero:\n%s", strings.Join(rows2, "\n"))
	}
	if !strings.Contains(strings.Join(rows2, "\n"), CursorStyle.Render("*")) {
		t.Errorf("selected far station not highlighted:\n%s", strings.Join(rows2, "\n"))
	}
}

func TestAPRSClosestDistStep(t *testing.T) {
	cases := []struct {
		radius int
		want   int
	}{
		{0, 0},
		{1, 1},
		{3, 1},
		{8, 10},
		{30, 25},
		{40, 50},
		{60, 50},
		{100, 100},
		{200, 100},
	}
	for _, tc := range cases {
		if got := aprsClosestDistStep(tc.radius); got != tc.want {
			t.Errorf("aprsClosestDistStep(%d) = %d, want %d", tc.radius, got, tc.want)
		}
	}
}

func TestAPRSPaneEnter_RadiusAlignsFilter(t *testing.T) {
	records := []aprs.StationRecord{
		testAPRSRecord("NEAR", 0.005, 0), // ~0.55 km
		testAPRSRecord("MID", 0.30, 0),   // ~33 km
		testAPRSRecord("FAR", 0.70, 0),   // ~77 km
	}
	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, records)
	// Enabled=false so the refresh radius filter stays off — only the
	// distance filter alignment is exercised.
	m.App.Logbook.APRS = &config.APRSConfig{RadiusKm: 25}
	m.aprsEnterPane()
	if m.aprsPane.distFilter != 25 {
		t.Fatalf("dist filter = %d, want 25", m.aprsPane.distFilter)
	}
	if len(m.aprsPane.stations) != 1 || m.aprsPane.stations[0].rec.Callsign != "NEAR" {
		t.Errorf("stations = %v, want NEAR only", m.aprsPane.stations)
	}
}

func TestAPRSPaneEnter_NoRadiusKeepsAll(t *testing.T) {
	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{testAPRSRecord("NEAR", 0.005, 0)})
	m.aprsEnterPane()
	if m.aprsPane.distFilter != 0 {
		t.Errorf("dist filter = %d, want all (0)", m.aprsPane.distFilter)
	}
}

// Manual filter changes must survive leaving and re-entering the pane —
// only the first entry aligns the distance filter with the APRS radius.
func TestAPRSPaneEnter_KeepsManualFilters(t *testing.T) {
	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{testAPRSRecord("NEAR", 0.005, 0)})
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 50}

	m.aprsEnterPane()
	if m.aprsPane.distFilter != 50 {
		t.Fatalf("initial dist filter = %d, want 50", m.aprsPane.distFilter)
	}

	// Manual change while the pane is open.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 'd', Text: "d"}, nil)
	if m.aprsPane.distFilter == 50 {
		t.Fatalf("'d' should move the filter away from 50")
	}
	want := m.aprsPane.distFilter

	// Re-entering the pane keeps the manual setting.
	m.aprsEnterPane()
	if m.aprsPane.distFilter != want {
		t.Errorf("manual filter not preserved: got %d, want %d", m.aprsPane.distFilter, want)
	}
	// Time and type filters are untouched by re-entry as well.
	m.aprsPane.timeFilter = 15
	m.aprsEnterPane()
	if m.aprsPane.timeFilter != 15 {
		t.Errorf("time filter not preserved across re-entry: %d", m.aprsPane.timeFilter)
	}
}

func TestAPRSIsOperatorSymbol(t *testing.T) {
	for _, sym := range []string{"/>", "/-", "/R", "/k", "/v", "/j", "/<", "/b", "/[", "/s", "/Y", "/^", "/'", "/X", "/O", "/p",
		"\\k", "\\>", "\\-", "\\p", "\\v"} {
		if !aprsIsOperatorSymbol(sym) {
			t.Errorf("aprsIsOperatorSymbol(%q) = false, want true", sym)
		}
	}
	for _, sym := range []string{"", "/#", "/r", "/_", "/W", "/n", "/m", "\\#", "3#"} {
		if aprsIsOperatorSymbol(sym) {
			t.Errorf("aprsIsOperatorSymbol(%q) = true, want false", sym)
		}
	}
}

func TestAPRSPaneFilters(t *testing.T) {
	m := newTestModel()
	// 60 km digi and a 5 km operator heard 2 minutes ago.
	records := []aprs.StationRecord{
		testAPRSRecord("NEAROP", 0.05, 0), // operator
		testAPRSRecord("FARDIG", 0.60, 0), // digi — filtered by type by default
		testAPRSRecord("OLDCAR", 0.02, 0), // operator, stale
	}
	records[1].Symbol = "/#"
	records[2].Symbol = "/>"
	records[2].LastHeard = time.Now().Add(-45 * time.Minute)
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 100}

	m.aprsPaneRefresh()
	// Default: all filters off — the digi is visible.
	if len(m.aprsPane.stations) != 3 {
		t.Fatalf("default all filter: got %d stations, want 3", len(m.aprsPane.stations))
	}

	// s: cycle type to operators — the digi is hidden.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 's', Text: "s"}, nil)
	if m.aprsPane.typeFilter != "operators" {
		t.Fatalf("type filter = %q, want operators", m.aprsPane.typeFilter)
	}
	if len(m.aprsPane.stations) != 2 {
		t.Errorf("after operators filter: got %d stations, want 2", len(m.aprsPane.stations))
	}

	// t: cycle time filter to 15m (60m → 30m → 15m) — drops OLDCAR.
	for i := 0; i < 3; i++ {
		_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 't', Text: "t"}, nil)
	}
	if m.aprsPane.timeFilter != 15 {
		t.Fatalf("time filter = %d, want 15", m.aprsPane.timeFilter)
	}
	if len(m.aprsPane.stations) != 1 || m.aprsPane.stations[0].rec.Callsign != "NEAROP" {
		t.Errorf("after time filter: stations = %v", m.aprsPane.stations)
	}

	// s: cycle type back to All — the digi returns.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 's', Text: "s"}, nil)
	if m.aprsPane.typeFilter != "" {
		t.Fatalf("type filter = %q, want all", m.aprsPane.typeFilter)
	}
	if len(m.aprsPane.stations) != 2 {
		t.Errorf("after type All: got %d stations, want 2", len(m.aprsPane.stations))
	}

	// backspace: clear — type back to all, time back to all.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: tea.KeyBackspace}, nil)
	if m.aprsPane.timeFilter != 0 || m.aprsPane.distFilter != 0 || m.aprsPane.typeFilter != "" {
		t.Errorf("backspace clear failed: dist=%d time=%d type=%q",
			m.aprsPane.distFilter, m.aprsPane.timeFilter, m.aprsPane.typeFilter)
	}
	if len(m.aprsPane.stations) != 3 {
		t.Errorf("after clear: got %d stations, want 3", len(m.aprsPane.stations))
	}
}

func TestAPRSPaneRefresh_OwnEchoFiltered(t *testing.T) {
	records := []aprs.StationRecord{
		testAPRSRecord("SP9SPM-7", 0.0005, 0), // our beacon echo — hidden
		testAPRSRecord("SP9SPM-0", 0.0005, 0), // own callsign, other SSID — kept
		testAPRSRecord("SP9MOA-1", 0.01, 0),   // other station — kept
	}
	m := newTestModel()
	m.App.Logbook.Station.Callsign = "SP9SPM"
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, SendLocation: true, RadiusKm: 100, Callsign: "SP9SPM-7"}
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.aprsPaneRefresh()
	if len(m.aprsPane.stations) != 2 {
		t.Fatalf("only the exact transmitting SSID should be hidden, got %v", m.aprsPane.stations)
	}
	for _, s := range m.aprsPane.stations {
		if s.rec.Callsign == "SP9SPM-7" {
			t.Errorf("own echo SP9SPM-7 should be hidden: %v", m.aprsPane.stations)
		}
	}
}

func TestAPRSPaneRefresh_OwnEchoNoSSID(t *testing.T) {
	records := []aprs.StationRecord{
		testAPRSRecord("SP9SPM", 0.0005, 0),   // our beacon, no SSID
		testAPRSRecord("SP9SPM-0", 0.0005, 0), // same identity per APRS spec
		testAPRSRecord("SP9MOA-1", 0.01, 0),   // other station — kept
	}
	m := newTestModel()
	m.App.Logbook.Station.Callsign = "SP9SPM"
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, SendLocation: true, RadiusKm: 100, Callsign: "SP9SPM"}
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.aprsPaneRefresh()
	if len(m.aprsPane.stations) != 1 || m.aprsPane.stations[0].rec.Callsign != "SP9MOA-1" {
		t.Fatalf("own identity (with and without SSID) must be hidden, got %v", m.aprsPane.stations)
	}
}

func TestAPRSPaneRefresh_ReceiveOnlyHidesOwnStations(t *testing.T) {
	records := []aprs.StationRecord{
		testAPRSRecord("SP9MOA-9", 0.0005, 0), // own SSID — RX mode QTH clutter
		testAPRSRecord("SP9MOA-0", 0.0005, 0), // own SSID, different suffix
		testAPRSRecord("SP9XYZ", 0.01, 0),
	}
	m := newTestModel() // station SP9MOA, no logbook APRS config
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.aprsPaneRefresh()
	if len(m.aprsPane.stations) != 1 || m.aprsPane.stations[0].rec.Callsign != "SP9XYZ" {
		t.Fatalf("receive-only must hide all own SSIDs, got %v", m.aprsPane.stations)
	}
}

func TestAPRSPaneFilters_DistanceSteps(t *testing.T) {
	m := newTestModel()
	records := []aprs.StationRecord{
		testAPRSRecord("NEAR1", 0.005, 0), // ~0.55 km
		testAPRSRecord("MID5", 0.03, 0),   // ~3.3 km
		testAPRSRecord("FAR10", 0.10, 0),  // ~11 km
	}
	m.App.APRSCache = newTestAPRSCache(t, records)
	m.App.Logbook.APRS = &config.APRSConfig{Enabled: true, RadiusKm: 100}
	m.aprsPaneRefresh()

	// d: cycle to 1 km — only NEAR1 remains.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 'd', Text: "d"}, nil)
	if m.aprsPane.distFilter != 1 {
		t.Fatalf("first dist step = %d, want 1", m.aprsPane.distFilter)
	}
	if len(m.aprsPane.stations) != 1 || m.aprsPane.stations[0].rec.Callsign != "NEAR1" {
		t.Errorf("1 km filter: got %v", m.aprsPane.stations)
	}

	// d: cycle to 5 km — NEAR1 + MID5 remain.
	_, _ = m.handleAPRSUpdate(tea.KeyPressMsg{Code: 'd', Text: "d"}, nil)
	if m.aprsPane.distFilter != 5 {
		t.Fatalf("second dist step = %d, want 5", m.aprsPane.distFilter)
	}
	if len(m.aprsPane.stations) != 2 {
		t.Errorf("5 km filter: got %d stations, want 2", len(m.aprsPane.stations))
	}
}

func TestAPRSPaneView_Narrow(t *testing.T) {
	// Minimum supported terminal (75x24) — the pane must fit without
	// overflowing the content area (list + details side by side).
	lay := Layout{TerminalW: 75, ContentW: 73, ContentH: 18}

	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{
		testAPRSRecord("SP9XYZ", 0.005, 0),
		testAPRSRecord("SP8ABC", 0.030, 0.010),
	})
	m.aprsPaneRefresh()
	m.aprsPaneSelect(0)

	v := m.viewAPRS(lay)
	if strings.Contains(v, "SP8ABC") == false {
		t.Errorf("narrow view missing second station: %q", v)
	}
	lines := strings.Split(v, "\n")
	if len(lines) > lay.ContentH {
		t.Errorf("narrow view overflow: %d lines > ContentH %d", len(lines), lay.ContentH)
	}
}

func TestAPRSPaneView_Borders(t *testing.T) {
	lay := Layout{TerminalW: 100, ContentW: 98, ContentH: 20}

	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{
		testAPRSRecord("SP9XYZ", 0.005, 0),
	})
	m.aprsPaneRefresh()
	m.aprsPaneSelect(0)
	v := m.viewAPRS(lay)
	// Both columns are wrapped in rounded border boxes.
	for _, want := range []string{"\u256d", "\u256e", "\u2570", "\u256f", "\u2502"} {
		if !strings.Contains(v, want) {
			t.Errorf("bordered view missing %q", want)
		}
	}
}

// With a selection the right column holds two stacked boxes: the own
// status on top and the selected contact + radar below it.
func TestAPRSPaneView_TwoRightBoxes(t *testing.T) {
	lay := Layout{TerminalW: 100, ContentW: 98, ContentH: 24}

	m := newTestModel()
	m.App.APRSCache = newTestAPRSCache(t, []aprs.StationRecord{
		testAPRSRecord("SP9XYZ", 0.005, 0),
	})
	m.aprsPaneRefresh()
	m.aprsPaneSelect(0)
	plain := stripANSI(m.viewAPRS(lay))

	// Three boxes in total: station table, status, contact.
	if n := strings.Count(plain, "\u256d"); n != 3 {
		t.Errorf("box tops = %d, want 3 (table, status, contact)", n)
	}
	if n := strings.Count(plain, "\u2570"); n != 3 {
		t.Errorf("box bottoms = %d, want 3 (table, status, contact)", n)
	}

	// Status and contact content live in different boxes: the status box
	// must close before the contact box opens.
	lines := strings.Split(plain, "\n")
	statusDone, contactOpen := false, false
	seenTransmitting := false
	for _, l := range lines {
		if strings.Contains(l, "transmitting.") {
			seenTransmitting = true
		}
		if seenTransmitting && strings.Contains(l, "\u2570") {
			statusDone = true
		}
		if strings.Contains(l, "Call SP9XYZ") {
			if !statusDone {
				t.Error("contact box opens before the status box closed")
			}
			contactOpen = true
		}
	}
	if !contactOpen {
		t.Error("contact box content missing")
	}
}
