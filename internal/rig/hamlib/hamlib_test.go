package hamlib

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// otherVFO tests
// =============================================================================

func TestOtherVFO(t *testing.T) {
	tests := []struct{ in, want string }{
		{"VFOA", "VFOB"}, {"vfoa", "VFOB"}, {" VFOA ", "VFOB"},
		{"VFOB", "VFOA"}, {"vfob", "VFOA"},
		{"Main", ""}, {"", ""},
	}
	for _, tt := range tests {
		if got := otherVFO(tt.in); got != tt.want {
			t.Errorf("otherVFO(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// =============================================================================
// hamlibMode tests
// =============================================================================

func TestHamlibMode_Passthrough(t *testing.T) {
	for _, m := range []string{"USB", "LSB", "AM", "FM", "CW", "RTTY"} {
		if got := hamlibMode(m); got != m {
			t.Errorf("hamlibMode(%q) = %q, want %q", m, got, m)
		}
	}
}

func TestHamlibMode_CW(t *testing.T) {
	for _, m := range []string{"CW-L", "CWL", "CW-U", "CWU"} {
		if got := hamlibMode(m); got != "CW" {
			t.Errorf("hamlibMode(%q) = %q, want CW", m, got)
		}
	}
}

func TestHamlibMode_Data(t *testing.T) {
	tests := map[string]string{
		"DATA-U": "PKTUSB", "DATAU": "PKTUSB",
		"DATA-L": "PKTLSB", "DATAL": "PKTLSB",
		"FT8": "PKTUSB", "FT4": "PKTUSB",
	}
	for in, want := range tests {
		if got := hamlibMode(in); got != want {
			t.Errorf("hamlibMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHamlibMode_Unknown(t *testing.T) {
	if got := hamlibMode("BOGUS"); got != "BOGUS" {
		t.Errorf("hamlibMode(BOGUS) = %q, want BOGUS", got)
	}
}

// =============================================================================
// parseFloat tests
// =============================================================================

func TestParseFloat(t *testing.T) {
	tests := []struct {
		s    string
		want float64
	}{
		{"14.074", 14.074}, {"14123000", 14123000}, {"0", 0},
		{"-1", -1}, {"1.5e2", 150}, {"", 0}, {"abc", 0},
	}
	for _, tt := range tests {
		if got := parseFloat(tt.s); got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.s, got, tt.want)
		}
	}
}

// =============================================================================
// Client integration tests (fake TCP server)
// =============================================================================

func TestClient_Status(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		// v → VFO name
		n, _ := conn.Read(buf)
		if cmd := strings.TrimSpace(string(buf[:n])); cmd != "v" {
			conn.Write([]byte("RPRT 1\n"))
			return
		}
		conn.Write([]byte("VFOA\n"))
		// s VFOA (VFO-prefixed — with --vfo, plain 's' returns nothing)
		n, _ = conn.Read(buf)
		if cmd := strings.TrimSpace(string(buf[:n])); cmd != "s VFOA" {
			return
		}
		conn.Write([]byte("0\n"))
		// f VFOA (VFO-prefixed frequency)
		n, _ = conn.Read(buf)
		if cmd := strings.TrimSpace(string(buf[:n])); cmd != "f VFOA" {
			return
		}
		conn.Write([]byte("14123000\n"))
		// m VFOA (VFO-prefixed — with --vfo, plain 'm' returns nothing)
		n, _ = conn.Read(buf)
		if cmd := strings.TrimSpace(string(buf[:n])); cmd != "m VFOA" {
			return
		}
		conn.Write([]byte("USB\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	c.SetVfoMode(true) // skip VFO probe in tests
	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Connected {
		t.Error("expected connected")
	}
	if status.Provider != "hamlib" {
		t.Errorf("provider = %q, want hamlib", status.Provider)
	}
	if status.Band != "20m" {
		t.Errorf("band = %q, want 20m", status.Band)
	}
	if status.Mode != "USB" {
		t.Errorf("mode = %q, want USB", status.Mode)
	}
	if status.Split {
		t.Error("expected simplex (split=false)")
	}
}

func TestClient_Status_Split(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		// v
		conn.Read(buf)
		conn.Write([]byte("VFOA\n"))
		// s VFOA (VFO-prefixed)
		conn.Read(buf)
		conn.Write([]byte("1\n"))
		// f VFOA
		conn.Read(buf)
		conn.Write([]byte("14123000\n"))
		// m VFOA (VFO-prefixed)
		conn.Read(buf)
		conn.Write([]byte("USB\n"))
		// f VFOB (split VFO B query)
		conn.Read(buf)
		conn.Write([]byte("14125000\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	c.SetVfoMode(true)
	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Split {
		t.Error("expected split=true")
	}
	if status.FrequencyHz != 14123000 {
		t.Errorf("FrequencyHz = %d, want 14123000", status.FrequencyHz)
	}
}

func TestClient_Status_SplitUnsupported(t *testing.T) {
	// Verifies that an unsupported split command (RPRT -11) is non-fatal:
	// the status should succeed with split=false and report frequency/mode.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		scanner := bufio.NewScanner(conn)
		// v — RPRT -11 (unsupported, non-fatal)
		scanner.Scan()
		conn.Write([]byte("RPRT -11\n"))
		// s — RPRT -11 (unsupported, now non-fatal — split stays false)
		scanner.Scan()
		conn.Write([]byte("RPRT -11\n"))
		// f
		scanner.Scan()
		conn.Write([]byte("14123000\n"))
		// m
		scanner.Scan()
		conn.Write([]byte("USB\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	c.SetVfoMode(false)
	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status should succeed despite split failure, got: %v", err)
	}
	if !status.Connected {
		t.Error("expected connected")
	}
	if status.Split {
		t.Error("expected split=false when split query is unsupported")
	}
	if status.FrequencyHz != 14123000 {
		t.Errorf("FrequencyHz = %d, want 14123000", status.FrequencyHz)
	}
	if status.Mode != "USB" {
		t.Errorf("Mode = %q, want USB", status.Mode)
	}
}

func TestClient_Status_Error(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		cmd := strings.TrimSpace(string(buf[:n]))
		if cmd == "v" {
			conn.Write([]byte("RPRT 1\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	_, err = c.Status(context.Background())
	if err == nil {
		t.Error("expected error for RPRT 1")
	}
	if !strings.Contains(err.Error(), "hamlib") {
		t.Errorf("error should mention hamlib: %v", err)
	}
}

func TestClient_Power(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		cmd := strings.TrimSpace(string(buf[:n]))
		if cmd == "l RFPOWER" {
			conn.Write([]byte("0.5\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	pwr, err := c.Power(context.Background())
	if err != nil {
		t.Fatalf("Power: %v", err)
	}
	if pwr != 50 {
		t.Errorf("power = %f, want 50", pwr)
	}
}

func TestClient_SetFrequency(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var received string
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		received = strings.TrimSpace(string(buf[:n]))
		conn.Write([]byte("RPRT 0\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	err = c.SetFrequency(context.Background(), 14250000)
	if err != nil {
		t.Fatalf("SetFrequency: %v", err)
	}
	if !strings.Contains(received, "14250000") {
		t.Errorf("received = %q, want 14250000 Hz", received)
	}
}

func TestClient_SetMode(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	var received string
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		received = strings.TrimSpace(string(buf[:n]))
		conn.Write([]byte("RPRT 0\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	err = c.SetMode(context.Background(), "USB")
	if err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if !strings.Contains(received, "USB") || !strings.Contains(received, "2400") {
		t.Errorf("received = %q, want \\set_mode USB 2400", received)
	}
}

func TestClient_GetModes(t *testing.T) {
	c := New("localhost", "0", 5*time.Second)
	modes, err := c.GetModes(context.Background())
	if err != nil {
		t.Fatalf("GetModes: %v", err)
	}
	if len(modes) < 4 {
		t.Errorf("expected at least 4 modes, got %d: %v", len(modes), modes)
	}
	found := map[string]bool{}
	for _, m := range modes {
		found[m] = true
	}
	for _, want := range []string{"USB", "LSB", "CW", "AM"} {
		if !found[want] {
			t.Errorf("GetModes missing %q", want)
		}
	}
}

func TestClient_GetName(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		cmd := strings.TrimSpace(string(buf[:n]))
		if cmd == "_" {
			conn.Write([]byte("FT-DX10\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 5*time.Second)
	name, err := c.GetName(context.Background())
	if err != nil {
		t.Fatalf("GetName: %v", err)
	}
	if name != "FT-DX10" {
		t.Errorf("name = %q, want FT-DX10", name)
	}
}

func TestNew(t *testing.T) {
	c := New("127.0.0.1", "4533", 5*time.Second)
	if c == nil {
		t.Fatal("New returned nil")
	}
	if c.addr != "127.0.0.1:4533" {
		t.Errorf("addr = %q, want 127.0.0.1:4533", c.addr)
	}
}
