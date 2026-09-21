package hamlib

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestClient_Status(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ch := make(chan struct{})
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		// 'p' command returns azimuth + elevation.
		var buf [64]byte
		n, _ := conn.Read(buf[:])
		if string(buf[:n]) == "p\r\n" {
			conn.Write([]byte("180.500000\n45.250000\nRPRT 0\n"))
		}
		ch <- struct{}{}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	ctx := context.Background()
	s, err := c.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Connected {
		t.Error("expected connected")
	}
	if s.Azimuth != 180.5 {
		t.Errorf("azimuth = %f, want 180.5", s.Azimuth)
	}
	if s.Elevation != 45.25 {
		t.Errorf("elevation = %f, want 45.25", s.Elevation)
	}

	<-ch
}

func TestClient_Status_SingleLine(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		var buf [64]byte
		n, _ := conn.Read(buf[:])
		if string(buf[:n]) == "p\r\n" {
			conn.Write([]byte("90.000000\nRPRT 0\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	s, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Azimuth != 90.0 {
		t.Errorf("azimuth = %f, want 90", s.Azimuth)
	}
	if s.Elevation != 0.0 {
		t.Errorf("elevation = %f, want 0", s.Elevation)
	}
}

func TestClient_Status_Error(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		var buf [64]byte
		n, _ := conn.Read(buf[:])
		if string(buf[:n]) == "p\r\n" {
			conn.Write([]byte("RPRT -1\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)

	_, err = c.Status(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClient_SetPosition(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ch := make(chan string, 1)
	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		var buf [64]byte
		n, _ := conn.Read(buf[:])
		ch <- string(buf[:n])
		conn.Write([]byte("RPRT 0\n"))
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	err = c.SetPosition(context.Background(), 45.0, 10.0)
	if err != nil {
		t.Fatal(err)
	}
	got := <-ch
	want := "P 45.000000 10.000000\r\n"
	if got != want {
		t.Errorf("command = %q, want %q", got, want)
	}
}

func TestClient_Status_Disconnected(t *testing.T) {
	c := New("127.0.0.1", "19999", 50*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := c.Status(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew(t *testing.T) {
	c := New("localhost", "4533", 2*time.Second)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.addr != "localhost:4533" {
		t.Errorf("addr = %q, want localhost:4533", c.addr)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		in   string
		want float64
	}{
		{"180.5", 180.5},
		{"  45.25  ", 45.25},
		{"0", 0},
		{"-5.0", -5.0},
		{"", 0},
		{"invalid", 0},
	}
	for _, tt := range tests {
		got := parseFloat(tt.in)
		if got != tt.want {
			t.Errorf("parseFloat(%q) = %f, want %f", tt.in, got, tt.want)
		}
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want float64
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{200, 0, 100, 100},
		{0, 0, 360, 0},
		{360, 0, 360, 360},
	}
	for _, tt := range tests {
		got := clamp(tt.v, tt.lo, tt.hi)
		if got != tt.want {
			t.Errorf("clamp(%f,%f,%f) = %f, want %f", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}

// TestClient_Status_SplitResponse verifies a response fragmented across
// several TCP reads is reassembled into complete lines — a single Read is
// not assumed to carry the whole reply.
func TestClient_Status_SplitResponse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		br := bufio.NewReader(conn)
		cmd, _ := br.ReadString('\n')
		if cmd != "p\r\n" {
			return
		}
		for _, part := range []string{"18", "0.50", "0000\n4", "5.250000\nRP", "RT 0\n"} {
			conn.Write([]byte(part))
			time.Sleep(10 * time.Millisecond)
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	s, err := c.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if s.Azimuth != 180.5 {
		t.Errorf("azimuth = %f, want 180.5", s.Azimuth)
	}
	if s.Elevation != 45.25 {
		t.Errorf("elevation = %f, want 45.25", s.Elevation)
	}
}

// TestClient_NoLeftoverBetweenCommands verifies buffered bytes from a split
// response cannot contaminate the next command's reply.
func TestClient_NoLeftoverBetweenCommands(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		br := bufio.NewReader(conn)

		cmd, _ := br.ReadString('\n')
		if cmd == "p\r\n" {
			conn.Write([]byte("180.5\n"))
			time.Sleep(20 * time.Millisecond)
			conn.Write([]byte("45.25\nRPRT 0\n"))
		}

		cmd, _ = br.ReadString('\n')
		if strings.HasPrefix(cmd, "P ") {
			conn.Write([]byte("RPRT 0\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	s, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if s.Azimuth != 180.5 || s.Elevation != 45.25 {
		t.Errorf("az/el = %f/%f, want 180.5/45.25 (leftover elevation leaked?)", s.Azimuth, s.Elevation)
	}

	if err := c.SetPosition(context.Background(), 10, 5); err != nil {
		t.Fatalf("set position: %v", err)
	}
}

// TestClient_IncompleteAckFails verifies an acknowledgement cut short (no
// terminator) is an error, not an apparent success.
func TestClient_IncompleteAckFails(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		br := bufio.NewReader(conn)
		cmd, _ := br.ReadString('\n')
		if cmd == "S\r\n" {
			conn.Write([]byte("RPRT 0")) // missing newline, then close
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	if err := c.Stop(context.Background()); err == nil {
		t.Fatal("expected error for an acknowledgement cut short")
	}
}

// TestClient_GetName_MultiLine verifies multi-line payloads are returned
// without the RPRT terminator line.
func TestClient_GetName_MultiLine(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	go func() {
		conn, _ := ln.Accept()
		defer conn.Close()
		br := bufio.NewReader(conn)
		cmd, _ := br.ReadString('\n')
		if cmd == "_\r\n" {
			conn.Write([]byte("Rig: Dummy\nModel: Rotor X\nRPRT 0\n"))
		}
	}()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c := New(host, port, 2*time.Second)
	defer c.Close()

	name, err := c.GetName(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(name, "Model: Rotor X") {
		t.Errorf("name = %q, want model line", name)
	}
	if strings.Contains(name, "RPRT") {
		t.Errorf("name = %q, must not include the RPRT terminator", name)
	}
}
