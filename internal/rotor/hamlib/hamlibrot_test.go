package hamlib

import (
	"context"
	"net"
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
