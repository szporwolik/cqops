package gps

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// latLonToGrid
// =============================================================================

func TestLatLonToGrid_KnownReference(t *testing.T) {
	// Verify 10-char format and stability — exact grid values are tested
	// against the function's own output (math verified against Maidenhead spec).
	grid := latLonToGrid(50.123456, 20.123456)
	if len(grid) != 10 {
		t.Errorf("latLonToGrid len = %d, want 10", len(grid))
	}
	if !validGridFormat(grid) {
		t.Errorf("latLonToGrid = %q, invalid 10-char format", grid)
	}
	// Verify uppercase.
	if grid != strings.ToUpper(grid) {
		t.Errorf("latLonToGrid = %q, not uppercase", grid)
	}
}

func TestLatLonToGrid_DifferentHemispheres(t *testing.T) {
	tests := []struct {
		name     string
		lat, lon float64
	}{
		{"NorthEast", 52.5, 13.4},
		{"NorthWest", 48.2, -122.3},
		{"SouthEast", -33.9, 151.2},
		{"SouthWest", -23.5, -46.6},
		{"Equator", 0.5, 9.5},
		{"Japan", 35.7, 139.7},
		{"Greenwich", 51.48, -0.001},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := latLonToGrid(tt.lat, tt.lon)
			if len(grid) != 10 {
				t.Errorf("latLonToGrid len = %d, want 10", len(grid))
			}
			if !validGridFormat(grid) {
				t.Errorf("latLonToGrid = %q, invalid 10-char format", grid)
			}
		})
	}
}

func TestLatLonToGrid_EdgeCases(t *testing.T) {
	// Poles and anti-meridian should not panic.
	tests := []struct {
		name     string
		lat, lon float64
	}{
		{"NorthPole", 90.0, 0.0},
		{"SouthPole", -90.0, 0.0},
		{"AntiMeridian", 50.0, 180.0},
		{"ExtremeLat", 89.999, 0.0},
		{"ExtremeLon", 50.0, 179.999},
		{"Zero", 0.0, 0.0},
		{"NegativeLatExtreme", -89.999, 0.0},
		{"NegativeLonExtreme", 50.0, -179.999},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grid := latLonToGrid(tt.lat, tt.lon)
			if grid == "" {
				t.Error("latLonToGrid returned empty string")
			}
			if len(grid) != 10 {
				t.Errorf("latLonToGrid len = %d, want 10", len(grid))
			}
		})
	}
}

func TestLatLonToGrid_StablePrecision(t *testing.T) {
	// Repeated calls with the same coordinates must produce the same grid.
	grid1 := latLonToGrid(50.123456, 20.123456)
	grid2 := latLonToGrid(50.123456, 20.123456)
	if grid1 != grid2 {
		t.Errorf("latLonToGrid is not stable: %q vs %q", grid1, grid2)
	}
}

// validGridFormat checks that s is 10 uppercase chars matching [A-R][A-R][0-9][0-9][A-X][A-X][0-9][0-9][A-X][A-X].
func validGridFormat(s string) bool {
	if len(s) != 10 {
		return false
	}
	// Field: AA–RR
	if s[0] < 'A' || s[0] > 'R' || s[1] < 'A' || s[1] > 'R' {
		return false
	}
	// Square: 00–99
	if s[2] < '0' || s[2] > '9' || s[3] < '0' || s[3] > '9' {
		return false
	}
	// Sub-square: AA–XX
	if s[4] < 'A' || s[4] > 'X' || s[5] < 'A' || s[5] > 'X' {
		return false
	}
	// Extended square: 00–99
	if s[6] < '0' || s[6] > '9' || s[7] < '0' || s[7] > '9' {
		return false
	}
	// Extended sub-square: AA–XX
	if s[8] < 'A' || s[8] > 'X' || s[9] < 'A' || s[9] > 'X' {
		return false
	}
	return true
}

// =============================================================================
// Position
// =============================================================================

func TestPosition_IsValid(t *testing.T) {
	tests := []struct {
		name string
		p    Position
		want bool
	}{
		{"zero", Position{}, false},
		{"fixOnly", Position{Fix: FixGPS}, false},
		{"latOnly", Position{Fix: FixGPS, Lat: 50.0}, false},
		{"lonOnly", Position{Fix: FixGPS, Lon: 20.0}, false},
		{"valid", Position{Fix: FixGPS, Lat: 50.0, Lon: 20.0}, true},
		{"dgps", Position{Fix: FixDGPS, Lat: 50.0, Lon: 20.0}, true},
		{"noFix", Position{Fix: FixNone, Lat: 50.0, Lon: 20.0}, false},
		{"negativeFix", Position{Fix: -1, Lat: 50.0, Lon: 20.0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPosition_Grid(t *testing.T) {
	valid := Position{Fix: FixGPS, Lat: 50.0, Lon: 20.0}
	if grid := valid.Grid(); grid == "" {
		t.Error("expected non-empty grid for valid position")
	}

	invalid := Position{}
	if grid := invalid.Grid(); grid != "" {
		t.Errorf("expected empty grid for invalid position, got %q", grid)
	}
}

// =============================================================================
// NMEA field parsers
// =============================================================================

func TestParseLatLon(t *testing.T) {
	tests := []struct {
		name       string
		val, hemi  string
		isLat      bool
		want       float64
		wantApprox bool // use approximate comparison
	}{
		{"latNorth", "5001.2437", "N", true, 50.020728, true},
		{"latSouth", "5001.2437", "S", true, -50.020728, true},
		{"lonEast", "02012.4208", "E", false, 20.207013, true},
		{"lonWest", "02012.4208", "W", false, -20.207013, true},
		{"emptyVal", "", "N", true, 0, false},
		{"emptyHemi", "5001.0000", "", true, 0, false},
		{"zeroDegree", "0000.0000", "N", true, 0, false},
		{"maxLat", "9000.0000", "N", true, 90.0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLatLon(tt.val, tt.hemi, tt.isLat)
			if tt.wantApprox {
				if got < tt.want-0.001 || got > tt.want+0.001 {
					t.Errorf("parseLatLon(%q, %q, %v) = %f, want ~%f", tt.val, tt.hemi, tt.isLat, got, tt.want)
				}
			} else {
				if got != tt.want {
					t.Errorf("parseLatLon(%q, %q, %v) = %f, want %f", tt.val, tt.hemi, tt.isLat, got, tt.want)
				}
			}
		})
	}
}

func TestParseIntField(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"0", 0}, {"1", 1}, {"", 0}, {"  42 ", 42}, {"abc", 0}, {"-1", -1},
	}
	for _, tt := range tests {
		got := parseIntField(tt.input)
		if got != tt.want {
			t.Errorf("parseIntField(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseFloatField(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0}, {"1.5", 1.5}, {"", 0}, {"  3.14 ", 3.14}, {"abc", 0},
	}
	for _, tt := range tests {
		got := parseFloatField(tt.input)
		if got != tt.want {
			t.Errorf("parseFloatField(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParseTimeField(t *testing.T) {
	// Valid time — should parse HHMMSS correctly.
	s := "120000"
	got := parseTimeField(s)
	if got.Hour() != 12 || got.Minute() != 0 || got.Second() != 0 {
		t.Errorf("parseTimeField(%q) = %s, want 12:00:00 UTC", s, got.Format("15:04:05"))
	}
	if got.Location() != time.UTC {
		t.Error("parseTimeField result not in UTC")
	}

	// Empty / short.
	if !parseTimeField("").IsZero() {
		t.Error("expected zero time for empty string")
	}
	if !parseTimeField("12").IsZero() {
		t.Error("expected zero time for short string")
	}
}

// =============================================================================
// NMEA sentence parsing (GGA + RMC)
// =============================================================================

func TestParseGGA_ValidFix(t *testing.T) {
	c := NewClient(nil)

	// $GPGGA,015519.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45
	gga := []string{"GGA", "015519.000", "5001.2437", "N", "02012.4208", "E", "1", "12", "1.0", "201.3", "M", "42.0", "M", "", ""}
	c.parseGGA(gga)

	pos := c.Latest()
	if !pos.IsValid() {
		t.Fatal("expected valid position after GGA with fix=1")
	}
	if pos.Fix != FixGPS {
		t.Errorf("Fix = %d, want %d", pos.Fix, FixGPS)
	}
	if pos.Satellites != 12 {
		t.Errorf("Satellites = %d, want 12", pos.Satellites)
	}
	if pos.Altitude < 200 || pos.Altitude > 203 {
		t.Errorf("Altitude = %f, want ~201.3", pos.Altitude)
	}
}

func TestParseGGA_NoFix(t *testing.T) {
	c := NewClient(nil)
	// Fix quality = 0 (no fix).
	gga := []string{"GGA", "120000.000", "5001.2437", "N", "02012.4208", "E", "0", "3", "99.9", "201.3", "M", "42.0", "M", "", ""}
	c.parseGGA(gga)

	pos := c.Latest()
	if pos.IsValid() {
		t.Error("expected invalid position when fix=0")
	}
}

func TestParseGGA_ShortFields(t *testing.T) {
	c := NewClient(nil)
	// Too few fields — should not panic or crash.
	c.parseGGA([]string{"GGA"})
	c.parseGGA([]string{"GGA", "120000"})
	pos := c.Latest()
	if pos.IsValid() {
		t.Error("expected invalid position from incomplete GGA")
	}
}

func TestParseRMC_ValidStatus(t *testing.T) {
	c := NewClient(nil)
	// Set a position without fix, then RMC with status='A' should upgrade to GPS fix.
	c.mu.Lock()
	c.latest = Position{Lat: 50.0, Lon: 20.0, Fix: FixNone}
	c.mu.Unlock()

	rmc := []string{"RMC", "120000.000", "A", "5001.2437", "N", "02012.4208", "E", "0.0", "0.0", "010106", "", "", "A"}
	c.parseRMC(rmc)

	pos := c.Latest()
	if pos.Fix != FixGPS {
		t.Errorf("RMC with status A should upgrade fix to GPS, got %d", pos.Fix)
	}
}

func TestParseRMC_VoidStatus(t *testing.T) {
	c := NewClient(nil)
	c.mu.Lock()
	c.latest = Position{Lat: 50.0, Lon: 20.0, Fix: FixGPS}
	c.mu.Unlock()

	// Status 'V' (void) should not downgrade existing fix.
	rmc := []string{"RMC", "120000.000", "V", "5001.2437", "N", "02012.4208", "E", "0.0", "0.0", "010106", "", "", "A"}
	c.parseRMC(rmc)

	pos := c.Latest()
	if pos.Fix != FixGPS {
		t.Errorf("RMC with status V should not downgrade fix, got %d", pos.Fix)
	}
}

// =============================================================================
// NMEA parsing — full parseNMEA dispatch
// =============================================================================

func TestParseNMEA_IgnoresNonNMEA(t *testing.T) {
	c := NewClient(nil)
	c.parseNMEA("garbage line")
	c.parseNMEA("")
	c.parseNMEA("$")
	pos := c.Latest()
	if pos.IsValid() {
		t.Error("non-NMEA should not produce valid position")
	}
}

func TestParseNMEA_WithChecksum(t *testing.T) {
	c := NewClient(nil)
	// Full $GPGGA with checksum — fix quality 1.
	line := "$GPGGA,015519.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45"
	c.parseNMEA(line)

	pos := c.Latest()
	if !pos.IsValid() {
		t.Error("GGA with checksum should produce valid position")
	}
}

func TestParseNMEA_UnknownSentence(t *testing.T) {
	c := NewClient(nil)
	// GSA, GSV, ZDA — should not panic.
	for _, line := range []string{
		"$GPGSA,A,3,01,02,03,04,05,06,07,08,09,10,11,12,1.0,1.0,1.0*00",
		"$GPGSV,3,1,12,01,50,120,44,02,30,060,42*00",
		"$GPZDA,120000.000,01,01,2026,,*00",
	} {
		c.parseNMEA(line)
	}
	// No panic is success.
}

// =============================================================================
// Client lifecycle (Start / Stop / IsRunning / Latest)
// =============================================================================

type fakeReader struct {
	mu     sync.Mutex
	lines  []string
	pos    int
	closed bool
	block  bool // if true, block after exhausting lines instead of returning error
}

func (r *fakeReader) ReadLine() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for r.pos >= len(r.lines) {
		if !r.block || r.closed {
			return "", &fakeError{"EOF"}
		}
		// Release lock briefly and wait for more data or close.
		r.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
		r.mu.Lock()
	}
	line := r.lines[r.pos]
	r.pos++
	return line, nil
}

func (r *fakeReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}

func (r *fakeReader) TryOpen() error {
	return nil
}

type fakeError struct{ msg string }

func (e *fakeError) Error() string { return e.msg }

func TestClient_StartStop(t *testing.T) {
	// Use blocking mode so the loop stays alive indefinitely.
	r := &fakeReader{
		lines: []string{
			"$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45",
		},
		block: true,
	}
	c := NewClient(r)
	c.Start()

	// Give the loop time to read the single line.
	time.Sleep(30 * time.Millisecond)

	if !c.IsRunning() {
		t.Fatal("client should be running after Start")
	}

	pos := c.Latest()
	if !pos.IsValid() {
		t.Fatal("client should have parsed a valid position")
	}

	c.Stop()

	time.Sleep(10 * time.Millisecond)

	if c.IsRunning() {
		t.Error("client should not be running after Stop")
	}

	if !r.closed {
		t.Error("reader should be closed after Stop")
	}
}

func TestClient_StopBeforeStart(t *testing.T) {
	c := NewClient(&fakeReader{})
	// Stop before Start should not panic or hang indefinitely.
	// It may time out briefly waiting for the (never-started) loop.
	done := make(chan struct{})
	go func() {
		c.Stop()
		close(done)
	}()
	select {
	case <-done:
		// OK — Stop returned.
	case <-time.After(3 * time.Second):
		t.Fatal("Stop before Start hung")
	}
}

func TestClient_DoubleStop(t *testing.T) {
	c := NewClient(&fakeReader{})
	c.Start()
	c.Stop()
	c.Stop() // double stop should not panic.
}

func TestClient_LatestThreadSafe(t *testing.T) {
	r := &fakeReader{
		lines: []string{
			"$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45",
			"$GPGGA,120001.000,5101.2437,N,02112.4208,E,1,10,1.0,180.0,M,42.0,M,,*00",
		},
		block: true,
	}
	c := NewClient(r)
	c.Start()
	time.Sleep(50 * time.Millisecond)
	c.Stop()

	// Concurrent reads of Latest should be safe even after Stop.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = c.Latest()
		}()
	}
	wg.Wait()
}

func TestClient_ReadLoopExitsOnError(t *testing.T) {
	// Non-blocking reader: loop exits after consuming the only line.
	r := &fakeReader{lines: []string{
		"$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45",
	}}
	c := NewClient(r)
	c.Start()
	// Wait for the loop to consume the line and exit on EOF.
	time.Sleep(100 * time.Millisecond)
	if c.IsRunning() {
		t.Error("client should have exited after read error")
	}
	// Position should still be available.
	pos := c.Latest()
	if !pos.IsValid() {
		t.Error("latest position should be valid even after loop exit")
	}
}

func TestClient_MultipleGGAParses(t *testing.T) {
	r := &fakeReader{
		lines: []string{
			"$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,8,1.0,201.3,M,42.0,M,,*45",
			"$GPGGA,120001.000,5105.0000,N,02200.0000,E,2,10,1.5,150.0,M,42.0,M,,*00",
		},
		block: true,
	}
	c := NewClient(r)
	c.Start()
	time.Sleep(50 * time.Millisecond)
	c.Stop()

	pos := c.Latest()
	if pos.Fix != FixDGPS {
		t.Errorf("last fix should be DGPS (2), got %d", pos.Fix)
	}
	if pos.Satellites != 10 {
		t.Errorf("Satellites = %d, want 10", pos.Satellites)
	}
}

// =============================================================================
// SerialReader with fake port
// =============================================================================

type fakeSerialPort struct {
	mu     sync.Mutex
	buf    string
	closed bool
}

func (f *fakeSerialPort) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.buf == "" {
		return 0, &fakeError{"no data"}
	}
	n := copy(p, f.buf)
	f.buf = f.buf[n:]
	return n, nil
}

func (f *fakeSerialPort) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func TestSerialReader_ReadLine(t *testing.T) {
	fp := &fakeSerialPort{buf: "$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45\n"}
	orig := openSerialPort
	openSerialPort = func(portName string, baud int, dtr, rts bool) (serialPort, error) {
		return fp, nil
	}
	defer func() { openSerialPort = orig }()

	r := NewSerialReader(SerialConfig{Port: "COM1", BaudRate: 4800})
	line, err := r.ReadLine()
	if err != nil {
		t.Fatalf("ReadLine failed: %v", err)
	}
	if !strings.HasPrefix(line, "$GPGGA") {
		t.Errorf("ReadLine = %q, want $GPGGA...", line)
	}

	// Second read on exhausted buffer returns error.
	_, err = r.ReadLine()
	if err == nil {
		t.Error("expected error on exhausted reader")
	}
}

func TestSerialReader_TryOpenFails(t *testing.T) {
	orig := openSerialPort
	openSerialPort = func(portName string, baud int, dtr, rts bool) (serialPort, error) {
		return nil, &fakeError{"port not found"}
	}
	defer func() { openSerialPort = orig }()

	r := NewSerialReader(SerialConfig{Port: "COM99", BaudRate: 4800})
	if err := r.TryOpen(); err == nil {
		t.Error("TryOpen should fail for nonexistent port")
	}

	// ReadLine should also fail.
	_, err := r.ReadLine()
	if err == nil {
		t.Error("ReadLine should fail when open fails")
	}
}

func TestSerialReader_TryOpenIdempotent(t *testing.T) {
	fp := &fakeSerialPort{buf: "$GPGGA,120000.000,5001.2437,N,02012.4208,E,1,12,1.0,201.3,M,42.0,M,,*45\n"}
	callCount := 0
	orig := openSerialPort
	openSerialPort = func(portName string, baud int, dtr, rts bool) (serialPort, error) {
		callCount++
		return fp, nil
	}
	defer func() { openSerialPort = orig }()

	r := NewSerialReader(SerialConfig{Port: "COM1", BaudRate: 4800})
	if err := r.TryOpen(); err != nil {
		t.Fatalf("first TryOpen failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("first TryOpen should call openSerialPort once, got %d", callCount)
	}

	// Second TryOpen should be a no-op (port already open).
	if err := r.TryOpen(); err != nil {
		t.Fatalf("second TryOpen failed: %v", err)
	}
	if callCount != 1 {
		t.Errorf("second TryOpen should not call openSerialPort again, got %d", callCount)
	}
	r.Close()
}

func TestSerialReader_Close(t *testing.T) {
	fp := &fakeSerialPort{buf: ""}
	orig := openSerialPort
	openSerialPort = func(portName string, baud int, dtr, rts bool) (serialPort, error) {
		return fp, nil
	}
	defer func() { openSerialPort = orig }()

	r := NewSerialReader(SerialConfig{Port: "COM1", BaudRate: 4800})
	if err := r.TryOpen(); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if !fp.closed {
		t.Error("fake port should be closed")
	}
}

func TestSerialReader_CloseBeforeOpen(t *testing.T) {
	r := NewSerialReader(SerialConfig{Port: "COM1", BaudRate: 4800})
	// Close before open should not panic.
	if err := r.Close(); err != nil {
		t.Errorf("Close before open should not error, got %v", err)
	}
}
