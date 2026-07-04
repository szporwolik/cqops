package aprs

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestConnection_NoServer(t *testing.T) {
	err := TestConnection("", "N0CALL", "-1")
	if err == nil {
		t.Fatal("expected error for empty server")
	}
	if !strings.Contains(err.Error(), "server") {
		t.Errorf("error should mention server, got: %v", err)
	}
}

func TestConnection_NoCallsign(t *testing.T) {
	err := TestConnection("localhost:9999", "", "-1")
	if err == nil {
		t.Fatal("expected error for empty callsign")
	}
	if !strings.Contains(err.Error(), "callsign") {
		t.Errorf("error should mention callsign, got: %v", err)
	}
}

func TestConnection_ServerUnreachable(t *testing.T) {
	// Use a port that nothing listens on.
	err := TestConnection("127.0.0.1:19999", "N0CALL", "-1")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if !strings.Contains(err.Error(), "cannot reach") && !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("unexpected error: %v", err)
	}
}

// fakeAPRSServer starts a TCP listener that mimics a minimal APRS-IS server.
// It sends the banner and logresp, then keeps the connection open until the
// test signals done (so the client can detect the disconnect).
func fakeAPRSServer(t *testing.T, verified bool) (addr string, done chan struct{}) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done = make(chan struct{})
	go func() {
		defer close(done)
		defer ln.Close()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			return
		}
		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil || n == 0 {
			return
		}

		fmt.Fprintln(conn, "# aprsc 2.1.14-g91e674b")
		if verified {
			fmt.Fprintln(conn, "# logresp N0CALL verified, server TEST")
		} else {
			fmt.Fprintln(conn, "# logresp N0CALL unverified, server TEST")
		}
		// Wait for client to disconnect (read blocks until close).
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		conn.Read(make([]byte, 1))
	}()
	return ln.Addr().String(), done
}

func TestConnection_Verified(t *testing.T) {
	addr, done := fakeAPRSServer(t, true)
	err := TestConnection(addr, "N0CALL", "-1")
	if err != nil {
		t.Errorf("expected success, got: %v", err)
	}
	<-done
}

func TestConnection_Unverified(t *testing.T) {
	addr, done := fakeAPRSServer(t, false)
	err := TestConnection(addr, "N0CALL", "badpass")
	if err == nil {
		t.Fatal("expected error for unverified login")
	}
	if !strings.Contains(err.Error(), "rejected") {
		t.Errorf("error should mention rejected, got: %v", err)
	}
	<-done
}

// =============================================================================
// Client lifecycle tests
// =============================================================================

func TestClient_StartStop(t *testing.T) {
	addr, done := fakeAPRSServer(t, true)
	c := NewClient(addr, "N0CALL", "-1", "")
	connected := make(chan bool, 2)
	c.OnStatus = func(ok bool, _ error) { connected <- ok }

	c.Start()
	select {
	case ok := <-connected:
		if !ok {
			t.Error("expected connected=true")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for connect")
	}

	if !c.IsRunning() {
		t.Error("expected running after start")
	}
	if !c.IsConnected() {
		t.Error("expected connected after start")
	}

	c.Stop()
	if c.IsRunning() {
		t.Error("expected not running after stop")
	}
	<-done
}

func TestClient_StopDuringConnect(t *testing.T) {
	c := NewClient("192.0.2.1:14580", "N0CALL", "-1", "")
	c.Start()
	time.Sleep(100 * time.Millisecond)
	c.Stop()
	if c.IsRunning() {
		t.Error("should not be running after stop during connect")
	}
}

// =============================================================================
// Position packet parser tests
// =============================================================================

func TestParsePosition_ValidUncompressed(t *testing.T) {
	raw := "N0CALL>APRS,TCPIP*:!4903.50N/07201.75Wr/.../A=000000 test comment"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("expected successful parse")
	}
	if sr.Callsign != "N0CALL" {
		t.Errorf("callsign = %q, want N0CALL", sr.Callsign)
	}
	if sr.Lat < 49.0 || sr.Lat > 49.1 {
		t.Errorf("lat = %f, want ~49.058", sr.Lat)
	}
	if sr.Lon > -72.0 || sr.Lon < -72.1 {
		t.Errorf("lon = %f, want ~-72.029", sr.Lon)
	}
	if sr.Symbol != "/r" {
		t.Errorf("symbol = %q, want /r", sr.Symbol)
	}
}

func TestParsePosition_WithTimestamp(t *testing.T) {
	// APRS timestamp format: @DDHHMMz then standard position.
	raw := "SP9MOA>APRS,TCPIP*:@120350z4903.50N/07201.75Wr/A=000000 hello"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("expected successful parse with timestamp")
	}
	if sr.Callsign != "SP9MOA" {
		t.Errorf("callsign = %q", sr.Callsign)
	}
}

func TestParsePosition_SouthWest(t *testing.T) {
	raw := "VK2ABC>APRS,TCPIP*:!3348.50S/15112.25E&/A=000000 Sydney"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("expected successful parse")
	}
	if sr.Lat > 0 {
		t.Error("south latitude should be negative")
	}
	if sr.Lon < 0 {
		t.Error("east longitude should be positive")
	}
}

func TestParsePosition_StatusPacket(t *testing.T) {
	raw := "N0CALL>APRS,TCPIP*:>status text here"
	_, ok := ParsePositionPacket(raw)
	if ok {
		t.Error("status packet should not parse as position")
	}
}

func TestParsePosition_TelemetryPacket(t *testing.T) {
	raw := "N0CALL>APRS,TCPIP*:T#000,100,200,300,400,500,00000000"
	_, ok := ParsePositionPacket(raw)
	if ok {
		t.Error("telemetry packet should not parse as position")
	}
}

func TestParsePosition_EmptyBody(t *testing.T) {
	_, ok := ParsePositionPacket("N0CALL>APRS,TCPIP*:")
	if ok {
		t.Error("empty body should not parse")
	}
}

func TestParsePosition_InvalidLatLon(t *testing.T) {
	raw := "N0CALL>APRS,TCPIP*:!99ZZ.ZZN/18000.00Wr/"
	_, ok := ParsePositionPacket(raw)
	if ok {
		t.Error("invalid lat/lon should not parse")
	}
}

// =============================================================================
// Mic-E compressed position parser tests
// =============================================================================

func TestParsePosition_MicEStandard(t *testing.T) {
	// Standard Mic-E packet — now properly decodes Yaesu destinations.
	raw := "SP9SPM>UPPQLL,SR9WXP*,WIDE1*,WIDE2-1,qAR,SR9NIS-1:`0(\x1cl \x1c-/\x145.500MHz QRV_4"
	sr, ok := ParsePositionPacket(raw)
	if ok {
		t.Logf("Yaesu Mic-E decoded: lat=%.4f lon=%.1f", sr.Lat, sr.Lon)
	}
}

func TestParsePosition_MicEStandardEncoded(t *testing.T) {
	// Mic-E with valid destination encoding.
	// Destination "!!!!!!" = lat 00°00.00'N.
	// Info field: `0 followed by longitude (0x1c = 28-28=0°), spd/course, symbol /-.
	raw := "N0CALL>!!!!!!,WIDE1*:`0\x1c\x1c\x1c/-test"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("standard Mic-E packet should parse")
	}
	if sr.Callsign != "N0CALL" {
		t.Errorf("callsign = %q, want N0CALL", sr.Callsign)
	}
	// Destination "!!!!!!" + SSID 0 → lat 0°N, lon 0°.
	t.Logf("Mic-E decoded: lat=%.4f lon=%.1f sym=%q course=%d speed=%d",
		sr.Lat, sr.Lon, sr.Symbol, sr.Course, sr.SpeedKmH)
}

func TestParsePosition_MicENonPosition(t *testing.T) {
	// Mic-E data type with no valid destination — should fail.
	raw := "N0CALL>APRS,WIDE1*:`0(T#000,100,200"
	_, ok := ParsePositionPacket(raw)
	// Destination "APRS" is not a valid Mic-E destination — should not parse.
	if ok {
		t.Error("Mic-E with non-MicE destination should not parse")
	}
}

// =============================================================================
// Base-91 compressed position parser tests
// =============================================================================

func TestParsePosition_Base91RoundTrip(t *testing.T) {
	// Encode 50°N 20°E using the standard APRS base-91 formula (ASCII-33).
	// lat = 90 - val/380926, lon = -180 + val/190463
	latVal := int((90.0 - 50.0) * 380926)
	lonVal := int((20.0 + 180.0) * 190463)
	latEnc := base91Encode4Ascii(latVal)
	lonEnc := base91Encode4Ascii(lonVal)
	// Format: !<sym_table><lat4><lon4><sym_code>
	raw := "N0CALL>APRS,TCPIP*:!/" + latEnc + lonEnc + "-"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatalf("round-trip Base-91 should parse (encoded %q %q): %q", latEnc, lonEnc, raw)
	}
	t.Logf("Base-91 lat=%.4f lon=%.4f (encoded: %q %q sym=%q)", sr.Lat, sr.Lon, latEnc, lonEnc, sr.Symbol)
	if sr.Lat < 49.0 || sr.Lat > 51.0 {
		t.Errorf("lat = %.4f, want ~50.0", sr.Lat)
	}
	if sr.Lon < 19.0 || sr.Lon > 21.0 {
		t.Errorf("lon = %.4f, want ~20.0", sr.Lon)
	}
}

// base91Encode4Ascii encodes an integer to a 4-character ASCII-offset-33 string.
func base91Encode4Ascii(v int) string {
	b := make([]byte, 4)
	for i := 3; i >= 0; i-- {
		b[i] = byte(v%91) + 33
		v /= 91
	}
	return string(b)
}

func TestParsePosition_Base91LoRa(t *testing.T) {
	// Real LoRa packet from APLRG1 device (SP9SVH-2).
	raw := "SP9SVH-2>APLRG1,WIDE1-1,WIDE2-1,QA6L9,qAR,SQ9LDR-2:!L59z@Sapd#  GLoRa Digi/Igate/WX Lewniowa 434.855MHz 1k2|*j%Y|"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("LoRa base-91 packet should parse")
	}
	t.Logf("LoRa Base-91 lat=%.4f lon=%.4f sym=%q", sr.Lat, sr.Lon, sr.Symbol)
	if sr.Lat < 49.0 || sr.Lat > 51.0 {
		t.Errorf("lat = %.4f, expected near Kraków (~49.9)", sr.Lat)
	}
	if sr.Lon < 20.0 || sr.Lon > 21.0 {
		t.Errorf("lon = %.4f, expected near Kraków (~20.7)", sr.Lon)
	}
}

func TestParsePosition_MicEYaesu(t *testing.T) {
	// Yaesu FTM-500D Mic-E packet (destination UPPQLL).
	raw := "SP9SPM>UPPQLL,WIDE1-1,WIDE2-1,qAR,SR9EMP-3:`0(\x1cl \x1c-/`145.500MHz QRV_4"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("Yaesu Mic-E packet should parse")
	}
	t.Logf("Yaesu Mic-E lat=%.4f lon=%.4f sym=%q course=%d speed=%d",
		sr.Lat, sr.Lon, sr.Symbol, sr.Course, sr.SpeedKmH)
	if sr.Symbol != "/-" {
		t.Errorf("symbol = %q, want '/-' (table '/' + code '-')", sr.Symbol)
	}
	if sr.Lat < 49.0 || sr.Lat > 51.0 {
		t.Errorf("lat = %.4f, expected near Kraków (~50.0)", sr.Lat)
	}
	if sr.Lon < 19.0 || sr.Lon > 21.0 {
		t.Errorf("lon = %.4f, expected near Kraków (~20.0)", sr.Lon)
	}
}

func TestParsePosition_Base91UncompressedFallsThrough(t *testing.T) {
	// Uncompressed '!' format should NOT be parsed as Base-91.
	raw := "N0CALL>APRS,TCPIP*:!4903.50N/07201.75Wr/ test"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("uncompressed '!' should parse as uncompressed, not Base-91")
	}
	if sr.Symbol != "/r" {
		t.Errorf("symbol = %q, want /r", sr.Symbol)
	}
}

func TestParsePosition_Base91Invalid(t *testing.T) {
	// Not a valid position packet at all.
	raw := "N0CALL>APRS,TCPIP*:!99ZZ.ZZN/18000.00Wr/"
	_, ok := ParsePositionPacket(raw)
	if ok {
		t.Error("invalid packet should not parse")
	}
}

// =============================================================================
// Object / Item parser tests
// =============================================================================

func TestParsePosition_ObjectUncompressed(t *testing.T) {
	// Standard APRS object with uncompressed position.
	raw := "N0CALL-1>APRS:;LEADER   *092345z4903.50N/07201.75W>088/036"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("object packet should parse")
	}
	if sr.Callsign != "LEADER" {
		t.Errorf("callsign = %q, want LEADER", sr.Callsign)
	}
	if sr.Lat < 49.0 || sr.Lat > 49.1 {
		t.Errorf("lat = %.4f, want ~49.058", sr.Lat)
	}
	if sr.Lon > -72.0 || sr.Lon < -72.1 {
		t.Errorf("lon = %.4f, want ~-72.029", sr.Lon)
	}
}

func TestParsePosition_ObjectCompressed(t *testing.T) {
	// APRS object with compressed position (base-91).
	// Symbol table '/', lat = 60.2305N, lon = 24.8790E, symbol code '/'
	raw := "N0CALL-1>APRS,TCPIP*,qAC,FIRST:;SRAL HQ  *100927z/0%E/Th4_/  A"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("compressed object packet should parse")
	}
	if sr.Callsign != "SRAL HQ" {
		t.Errorf("callsign = %q, want 'SRAL HQ'", sr.Callsign)
	}
	if sr.Lat < 60.0 || sr.Lat > 60.5 {
		t.Errorf("lat = %.4f, want ~60.23", sr.Lat)
	}
	if sr.Lon < 24.0 || sr.Lon > 25.0 {
		t.Errorf("lon = %.4f, want ~24.88", sr.Lon)
	}
}

func TestParsePosition_ObjectKilled(t *testing.T) {
	// Killed object (underscore instead of asterisk).
	raw := "N0CALL-1>APRS:;LEADER   _092345z4903.50N/07201.75W>088/036"
	_, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("killed object packet should still parse position")
	}
}

func TestParsePosition_ItemUncompressed(t *testing.T) {
	// APRS item with uncompressed position.
	raw := "N0CALL>APRS:)TEST!4903.50N/07201.75Wr/ test"
	sr, ok := ParsePositionPacket(raw)
	if !ok {
		t.Fatal("item packet should parse")
	}
	if sr.Callsign != "TEST" {
		t.Errorf("callsign = %q, want TEST", sr.Callsign)
	}
	if sr.Lat < 49.0 || sr.Lat > 49.1 {
		t.Errorf("lat = %.4f, want ~49.058", sr.Lat)
	}
}

func TestParsePosition_ObjectTooShort(t *testing.T) {
	raw := "N0CALL-1>APRS:;SHORT"
	_, ok := ParsePositionPacket(raw)
	if ok {
		t.Error("too-short object should not parse")
	}
}
