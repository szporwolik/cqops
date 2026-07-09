package aprs

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// encodeAX25Addr tests
// =============================================================================

func TestEncodeAX25Addr_SimpleCallsign(t *testing.T) {
	b := encodeAX25Addr("SP9MOA", true)
	if len(b) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(b))
	}
	// Verify "SP9MOA" encoding (each char shifted left 1).
	expected := []byte{'S', 'P', '9', 'M', 'O', 'A'}
	for i := 0; i < 6; i++ {
		if b[i] != expected[i]<<1 {
			t.Errorf("byte %d: got 0x%02X, want 0x%02X", i, b[i], expected[i]<<1)
		}
	}
	// Last byte: SSID=0, end bit set, bits 5-6 set.
	if b[6] != 0x61 { // 0110_0001
		t.Errorf("SSID byte: got 0x%02X, want 0x61", b[6])
	}
}

func TestEncodeAX25Addr_WithSSID(t *testing.T) {
	b := encodeAX25Addr("SP9MOA-9", true)
	if len(b) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(b))
	}
	// SSID 9 in bits 1-4 = 9<<1 = 18 = 0x12, plus end bit (1) + reserved bits (0x60) = 0x73
	if b[6] != 0x73 {
		t.Errorf("SSID byte: got 0x%02X, want 0x73", b[6])
	}
}

func TestEncodeAX25Addr_NotLast(t *testing.T) {
	b := encodeAX25Addr("APRS", false)
	if len(b) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(b))
	}
	// End bit should NOT be set.
	if b[6]&1 != 0 {
		t.Error("end bit should not be set for non-last address")
	}
	// Reserved bits should still be set.
	if b[6]&0x60 != 0x60 {
		t.Error("reserved bits should be set")
	}
}

func TestEncodeAX25Addr_PaddedCallsign(t *testing.T) {
	// Short callsign should be space-padded.
	b := encodeAX25Addr("K1AB", true)
	if len(b) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(b))
	}
	// First 4 bytes: K1AB, next 2: spaces.
	if b[0] != 'K'<<1 {
		t.Errorf("byte 0: got 0x%02X", b[0])
	}
	if b[3] != 'B'<<1 {
		t.Errorf("byte 3: got 0x%02X", b[3])
	}
	if b[4] != ' '<<1 {
		t.Errorf("byte 4: got 0x%02X, want space<<1", b[4])
	}
	if b[5] != ' '<<1 {
		t.Errorf("byte 5: got 0x%02X, want space<<1", b[5])
	}
}

func TestEncodeAX25Addr_LowercaseUppercased(t *testing.T) {
	b := encodeAX25Addr("sp9moa", true)
	// First char should be 'S' not 's'.
	if b[0] != 'S'<<1 {
		t.Errorf("lowercase should be uppercased: got 0x%02X, want %c", b[0], 'S'<<1)
	}
}

func TestEncodeAX25Addr_DigipeaterPath(t *testing.T) {
	b := encodeAX25Addr("WIDE1-1", true)
	if len(b) != 7 {
		t.Fatalf("expected 7 bytes, got %d", len(b))
	}
	// SSID=1 in bits 1-4 = 2 (0x02) + end + reserved = 0x63
	if b[6] != 0x63 {
		t.Errorf("SSID byte: got 0x%02X, want 0x63", b[6])
	}
}

func TestEncodeAX25Addr_InvalidSSID(t *testing.T) {
	// Invalid SSID should default to 0.
	b := encodeAX25Addr("CALL-xyz", true)
	if b[6]&0x0E != 0 { // bits 1-4 should be 0 for SSID=0
		t.Errorf("invalid SSID should default to 0, got byte 6=0x%02X", b[6])
	}
}

// =============================================================================
// decodeAX25Addr tests
// =============================================================================

func TestDecodeAX25Addr_SimpleCallsign(t *testing.T) {
	b := encodeAX25Addr("SP9MOA", true)
	cs := decodeAX25Addr(b)
	if cs != "SP9MOA" {
		t.Errorf("round-trip failed: got %q, want %q", cs, "SP9MOA")
	}
}

func TestDecodeAX25Addr_WithSSID(t *testing.T) {
	b := encodeAX25Addr("SP9MOA-9", true)
	cs := decodeAX25Addr(b)
	if cs != "SP9MOA-9" {
		t.Errorf("round-trip failed: got %q, want %q", cs, "SP9MOA-9")
	}
}

func TestDecodeAX25Addr_ShortBuffer(t *testing.T) {
	if cs := decodeAX25Addr([]byte{0x00, 0x00}); cs != "" {
		t.Errorf("short buffer should return empty, got %q", cs)
	}
}

func TestDecodeAX25Addr_EmptyBuffer(t *testing.T) {
	if cs := decodeAX25Addr(nil); cs != "" {
		t.Errorf("nil buffer should return empty, got %q", cs)
	}
}

func TestDecodeAX25Addr_Padded(t *testing.T) {
	b := encodeAX25Addr("K1AB", true)
	cs := decodeAX25Addr(b)
	if cs != "K1AB" {
		t.Errorf("got %q, want K1AB", cs)
	}
}

func TestDecodeAX25Addr_ZeroSSID(t *testing.T) {
	b := encodeAX25Addr("N0CALL", true)
	cs := decodeAX25Addr(b)
	if cs != "N0CALL" {
		t.Errorf("got %q, want N0CALL", cs)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []string{
		"SP9MOA",
		"SP9MOA-9",
		"K1AB",
		"WIDE1-1",
		"WIDE2-2",
		"APRS",
		"N0CALL-15",
		"G1ABC",
		"ZL1ABC-5",
	}
	for _, tc := range tests {
		t.Run(tc, func(t *testing.T) {
			b := encodeAX25Addr(tc, true)
			cs := decodeAX25Addr(b)
			if cs != tc {
				t.Errorf("round-trip: got %q, want %q", cs, tc)
			}
		})
	}
}

// SSID-0 is equivalent to no SSID in AX.25 — the suffix is dropped on decode.
func TestEncodeDecode_SSIDZeroDropped(t *testing.T) {
	b := encodeAX25Addr("SP9SPM-0", true)
	cs := decodeAX25Addr(b)
	if cs != "SP9SPM" {
		t.Errorf("SSID-0 should be dropped: got %q, want %q", cs, "SP9SPM")
	}
}

// =============================================================================
// sendKISSFrame tests
// =============================================================================

func TestSendKISSFrame_BasicPosition(t *testing.T) {
	tnc2 := "SP9MOA-9>APRS,WIDE1-1,WIDE2-1:!5003.50N/02008.50E-/...test"
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, tnc2)
	if err != nil {
		t.Fatalf("sendKISSFrame failed: %v", err)
	}
	data := buf.Bytes()
	if len(data) == 0 {
		t.Fatal("expected non-empty output")
	}
	// Check KISS framing: starts with FEND + command byte (0x00 for data).
	if data[0] != 0xC0 {
		t.Errorf("expected FEND (0xC0) at start, got 0x%02X", data[0])
	}
	if data[1] != 0x00 {
		t.Errorf("expected command byte 0x00, got 0x%02X", data[1])
	}
	// Check ends with FEND.
	if data[len(data)-1] != 0xC0 {
		t.Errorf("expected FEND (0xC0) at end, got 0x%02X", data[len(data)-1])
	}
	// Verify the frame contains the info payload.
	if !bytes.Contains(data, []byte("test")) {
		t.Error("frame should contain info payload")
	}
}

func TestSendKISSFrame_MissingColon(t *testing.T) {
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, "SP9MOA>APRS") // no colon
	if err == nil {
		t.Fatal("expected error for missing ':'")
	}
	if !strings.Contains(err.Error(), "missing ':'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKISSFrame_MissingGreaterThan(t *testing.T) {
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, "SP9MOAAPRS:body") // no '>'
	if err == nil {
		t.Fatal("expected error for missing '>'")
	}
	if !strings.Contains(err.Error(), "missing '>'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKISSFrame_NoDigipeaters(t *testing.T) {
	tnc2 := "SP9MOA>APRS:!5003.50N/02008.50E-/...no digis"
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, tnc2)
	if err != nil {
		t.Fatalf("sendKISSFrame failed: %v", err)
	}
	data := buf.Bytes()
	if len(data) < 5 {
		t.Fatal("output too short")
	}
	if !bytes.Contains(data, []byte("no digis")) {
		t.Error("should contain info payload even without digis")
	}
}

func TestSendKISSFrame_EscapeHandling(t *testing.T) {
	// Build a TNC2 where the info field contains 0xC0 (FEND) — should be escaped.
	tnc2 := "SP9MOA>APRS:test\xC0escaped"
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, tnc2)
	if err != nil {
		t.Fatalf("sendKISSFrame failed: %v", err)
	}
	data := buf.Bytes()
	// After FEND+0x00, the frame data should contain escape sequence for 0xC0.
	// FESC is 0xDB, TFEND is 0xDC.
	inner := data[2 : len(data)-1] // strip FEND CMD ... FEND
	if !bytes.Contains(inner, []byte{0xDB, 0xDC}) {
		t.Error("0xC0 in payload should be escaped as FESC+TFEND")
	}
	// Original FEND (0xC0) should NOT appear unescaped in the frame body.
	if bytes.Contains(inner, []byte{0xC0}) {
		t.Error("unescaped FEND found in frame body")
	}
}

func TestSendKISSFrame_FESCEscape(t *testing.T) {
	// Build a TNC2 where the info field contains 0xDB (FESC) — should be escaped.
	tnc2 := "SP9MOA>APRS:test\xDBescaped"
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, tnc2)
	if err != nil {
		t.Fatalf("sendKISSFrame failed: %v", err)
	}
	data := buf.Bytes()
	inner := data[2 : len(data)-1]
	if !bytes.Contains(inner, []byte{0xDB, 0xDD}) {
		t.Error("0xDB in payload should be escaped as FESC+TFESC")
	}
}

// =============================================================================
// extractTNC2 tests
// =============================================================================

func TestExtractTNC2_PlainTNC2Passthrough(t *testing.T) {
	// Raw text that already looks like TNC2 should pass through.
	raw := []byte("SP9MOA-9>APRS,WIDE1-1:!5003.50N/02008.50E-/...")
	result := extractTNC2(raw)
	if result != string(raw) {
		t.Errorf("plain TNC2 should pass through unchanged: got %q", result)
	}
}

func TestExtractTNC2_PlainTNC2WithDigitStart(t *testing.T) {
	// Some packets start with a digit (e.g. object reports).
	raw := []byte("092345z>APRS:!5003.50N/02008.50E-/...")
	result := extractTNC2(raw)
	if result != string(raw) {
		t.Errorf("TNC2 with digit start should pass through: got %q", result)
	}
}

func TestExtractTNC2_ShortBuffer(t *testing.T) {
	raw := []byte("short")
	result := extractTNC2(raw)
	if result != "short" {
		t.Errorf("short buffer should be returned as-is: got %q", result)
	}
}

func TestExtractTNC2_EmptyBuffer(t *testing.T) {
	result := extractTNC2(nil)
	if result != "" {
		t.Errorf("nil buffer: got %q", result)
	}
}

func TestExtractTNC2_RoundTripFromEncodedFrame(t *testing.T) {
	// Build a KISS frame from TNC2, then extract it back.
	tnc2 := "SP9MOA-9>APRS,WIDE1-1,WIDE2-1:!5003.50N/02008.50E-/...hello"
	var buf bytes.Buffer
	err := sendKISSFrame(&buf, tnc2)
	if err != nil {
		t.Fatalf("sendKISSFrame failed: %v", err)
	}
	kissFrame := buf.Bytes()

	// Strip KISS wrapping (FEND CMD ... FEND) and unescape.
	frame := kissKISSUnframe(kissFrame)
	if len(frame) == 0 {
		t.Fatal("failed to unframe KISS frame")
	}

	// Extract TNC2 from raw AX.25 frame.
	result := extractTNC2(frame)
	if !strings.Contains(result, "SP9MOA-9") {
		t.Errorf("extracted TNC2 should contain source: got %q", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("extracted TNC2 should contain info: got %q", result)
	}
}

// kissKISSUnframe strips KISS framing (FEND, command byte) and undoes escaping.
func kissKISSUnframe(data []byte) []byte {
	if len(data) < 3 {
		return nil
	}
	if data[0] != 0xC0 || data[len(data)-1] != 0xC0 {
		return nil
	}
	// Skip FEND and command byte, strip trailing FEND.
	inner := data[2 : len(data)-1]
	out := make([]byte, 0, len(inner))
	for i := 0; i < len(inner); i++ {
		b := inner[i]
		if b == 0xDB && i+1 < len(inner) { // FESC
			switch inner[i+1] {
			case 0xDC: // TFEND → FEND
				out = append(out, 0xC0)
				i++
			case 0xDD: // TFESC → FESC
				out = append(out, 0xDB)
				i++
			default:
				out = append(out, b)
			}
		} else {
			out = append(out, b)
		}
	}
	return out
}

// =============================================================================
// KISSServerClient tests (using local TCP server)
// =============================================================================

func TestKISSServerClient_StartStop(t *testing.T) {
	// Start a fake KISS server that just accepts and idles.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	addr := ln.Addr().String()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		// Keep connection open for a bit.
		time.Sleep(200 * time.Millisecond)
		conn.Close()
	}()

	kc := NewKISSServerClient(addr)
	if kc.IsRunning() {
		t.Error("should not be running before Start")
	}
	if kc.IsConnected() {
		t.Error("should not be connected before Start")
	}

	kc.Start()
	time.Sleep(50 * time.Millisecond)

	if !kc.IsRunning() {
		t.Error("should be running after Start")
	}

	// Wait for connection or timeout.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if kc.IsConnected() {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	kc.Stop()
	time.Sleep(100 * time.Millisecond)

	if kc.IsRunning() {
		t.Error("should not be running after Stop")
	}
	wg.Wait()
}

func TestKISSServerClient_DoubleStart(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, _ := ln.Accept()
		if conn != nil {
			time.Sleep(100 * time.Millisecond)
			conn.Close()
		}
	}()

	kc := NewKISSServerClient(ln.Addr().String())
	kc.Start()
	kc.Start() // second Start should be a no-op.
	time.Sleep(50 * time.Millisecond)
	if !kc.IsRunning() {
		t.Error("should still be running")
	}
	kc.Stop()
	wg.Wait()
}

func TestKISSServerClient_SendPositionWhileDisconnected(t *testing.T) {
	kc := NewKISSServerClient("127.0.0.1:19999")
	err := kc.SendPosition("N0CALL", 50.0, 20.0, "/-", "test")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKISSServerClient_StatusCallback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, _ := ln.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	statusCh := make(chan bool, 2)
	kc := NewKISSServerClient(ln.Addr().String())
	kc.OnStatus = func(connected bool, err error) {
		statusCh <- connected
	}
	kc.Start()

	// Should eventually get a connected callback.
	select {
	case connected := <-statusCh:
		if !connected {
			t.Error("expected connected=true callback")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for status callback")
	}

	kc.Stop()
	wg.Wait()
}

// =============================================================================
// TestKISSServerConnection tests
// =============================================================================

func TestTestKISSServerConnection_Unreachable(t *testing.T) {
	err := TestKISSServerConnection("127.0.0.1:19999")
	if err == nil {
		t.Fatal("expected error for unreachable server")
	}
	if !strings.Contains(err.Error(), "connect") && !strings.Contains(err.Error(), "refused") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTestKISSServerConnection_Reachable(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		conn, _ := ln.Accept()
		if conn != nil {
			// Read the probe frame (FEND 0xFF FEND) to verify it was sent.
			buf := make([]byte, 64)
			n, _ := conn.Read(buf)
			if n > 0 && buf[0] != 0xC0 {
				t.Logf("unexpected first byte from KISS test: 0x%02X", buf[0])
			}
			conn.Close()
		}
	}()

	err = TestKISSServerConnection(ln.Addr().String())
	if err != nil {
		t.Errorf("expected no error for reachable server, got: %v", err)
	}
	wg.Wait()
}

// =============================================================================
// KISSClient basic lifecycle tests (no real serial port)
// =============================================================================

func TestKISSClient_StartStopNoPort(t *testing.T) {
	// KISSClient without a real serial port — Start should fail gracefully.
	kc := NewKISSClient("COM_NONEXISTENT_99999", 9600, 8, 0, 0, false, false)
	if kc.IsRunning() {
		t.Error("should not be running before Start")
	}
	if kc.IsConnected() {
		t.Error("should not be connected before Start")
	}
	// Start will attempt to open the port and fail. runLoop should exit.
	kc.Start()
	time.Sleep(300 * time.Millisecond)
	// Should not be stuck — runLoop should exit after failing to open port.
	kc.Stop()
	time.Sleep(50 * time.Millisecond)
	if kc.IsRunning() {
		t.Error("should not be running after Stop")
	}
}

func TestKISSClient_DoubleStop(t *testing.T) {
	kc := NewKISSClient("COM_NONEXISTENT", 9600, 8, 0, 0, false, false)
	kc.Stop() // stop when not running should be safe
	kc.Stop() // double stop should be safe
}

func TestKISSClient_DoubleStart(t *testing.T) {
	kc := NewKISSClient("COM_NONEXISTENT", 9600, 8, 0, 0, false, false)
	kc.Start()
	kc.Start() // second start should be no-op
	time.Sleep(100 * time.Millisecond)
	kc.Stop()
}

func TestKISSClient_SendPositionDisconnected(t *testing.T) {
	kc := NewKISSClient("COM_NONEXISTENT", 9600, 8, 0, 0, false, false)
	err := kc.SendPosition("N0CALL", 50.0, 20.0, "/-", "test")
	if err == nil {
		t.Fatal("expected error when not connected")
	}
	if !strings.Contains(err.Error(), "not connected") {
		t.Errorf("unexpected error: %v", err)
	}
}
