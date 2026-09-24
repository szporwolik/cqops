package dxc

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestParseSpot(t *testing.T) {
	tests := []struct {
		line        string
		wantOK      bool
		wantDX      string
		wantFreq    float64
		wantSpotter string
	}{
		{"DX de SP9MOA:  14074.0  K1ABC  FT8 TNX", true, "K1ABC", 14074.0, "SP9MOA"},
		{"DX de N0NBH:     3800.0   W1AW     Hello World", true, "W1AW", 3800.0, "N0NBH"},
		{"DX de EA3XXX:  28001.5  JA1ABC", true, "JA1ABC", 28001.5, "EA3XXX"},
		{"DX de IZ1:  7000.0  DL1ABC  59 QSL", true, "DL1ABC", 7000.0, "IZ1"},
		// Non-spot lines
		{"Hello from cluster", false, "", 0, ""},
		{"Welcome to DXSpider", false, "", 0, ""},
		{"", false, "", 0, ""},
		// Edge cases
		{"DX de : 14000.0 CALL", false, "", 0, ""},
		{"DX de SP9:  x  CALL", false, "", 0, ""},
		{"DX de SP9:  14000.0", false, "", 0, ""},
	}

	for _, tt := range tests {
		s, ok := parseSpot(tt.line)
		if ok != tt.wantOK {
			t.Errorf("parseSpot(%q) ok=%v, want %v", tt.line, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if s.DXCall != tt.wantDX {
			t.Errorf("parseSpot(%q) DXCall=%q, want %q", tt.line, s.DXCall, tt.wantDX)
		}
		if s.Frequency != tt.wantFreq {
			t.Errorf("parseSpot(%q) Freq=%f, want %f", tt.line, s.Frequency, tt.wantFreq)
		}
		if s.Spotter != tt.wantSpotter {
			t.Errorf("parseSpot(%q) Spotter=%q, want %q", tt.line, s.Spotter, tt.wantSpotter)
		}
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("", "", "SP9MOA")
	if c.host != "dxspots.com" {
		t.Errorf("default host = %q, want dxspots.com", c.host)
	}
	if c.port != "7300" {
		t.Errorf("default port = %q, want 7300", c.port)
	}
	if c.login != "SP9MOA" {
		t.Errorf("login = %q, want SP9MOA", c.login)
	}
}

// TestClient_ReconnectsAfterDisconnectAndStopJoins verifies the client owns
// reconnection after the first successful connect and that Stop joins its
// goroutines (no abandoned reconnect loops).
func TestClient_ReconnectsAfterDisconnectAndStopJoins(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())

	var mu sync.Mutex
	var accepted []net.Conn
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			accepted = append(accepted, conn)
			mu.Unlock()
			go func(conn net.Conn) {
				buf := make([]byte, 1024)
				for {
					if _, err := conn.Read(buf); err != nil {
						conn.Close()
						return
					}
				}
			}(conn)
		}
	}()

	client := NewClient("127.0.0.1", portStr, "SP9MOA")
	if err := client.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	if !client.ConnectedOnce() {
		t.Fatal("ConnectedOnce should be true after a successful Start")
	}

	// All accepted-connection access goes through these lock-protected
	// helpers — the accept goroutine reassigns the slice under mu.
	connCount := func() int {
		mu.Lock()
		defer mu.Unlock()
		return len(accepted)
	}
	connAt := func(i int) net.Conn {
		mu.Lock()
		defer mu.Unlock()
		return accepted[i]
	}

	// Wait for the first connection, then sever it server-side.
	waitConns(t, connCount, 1, 3*time.Second)
	connAt(0).Close()

	// The client must reconnect on its own (first backoff: 2s).
	waitConns(t, connCount, 2, 8*time.Second)

	// Stop joins the goroutines — no further reconnects may occur.
	done := make(chan struct{})
	go func() { client.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Stop did not return — goroutines not joined")
	}
	time.Sleep(300 * time.Millisecond)
	if n := connCount(); n > 2 {
		t.Errorf("client reconnected after Stop (conns %d, want 2)", n)
	}
}

// waitConns waits until at least want connections were accepted.
func waitConns(t *testing.T, count func() int, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if n := count(); n >= want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d connections (got %d)", want, count())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
