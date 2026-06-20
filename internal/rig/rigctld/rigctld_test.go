package rigctld

import (
	"context"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	c := New("127.0.0.1", 4532, 500)
	if c.Host != "127.0.0.1" {
		t.Errorf("Host = %q, want 127.0.0.1", c.Host)
	}
	if c.Port != 4532 {
		t.Errorf("Port = %d, want 4532", c.Port)
	}
	if c.TimeoutMS != 500 {
		t.Errorf("TimeoutMS = %d, want 500", c.TimeoutMS)
	}
}

func TestNew_DifferentValues(t *testing.T) {
	c := New("192.168.1.100", 1234, 1000)
	if c.Host != "192.168.1.100" {
		t.Errorf("Host = %q", c.Host)
	}
	if c.Port != 1234 {
		t.Errorf("Port = %d", c.Port)
	}
	if c.TimeoutMS != 1000 {
		t.Errorf("TimeoutMS = %d", c.TimeoutMS)
	}
}

func TestStatus_ReturnsDisconnected(t *testing.T) {
	c := New("localhost", 4532, 500)
	ctx := context.Background()
	status, err := c.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Provider != "rigctld" {
		t.Errorf("Provider = %q, want rigctld", status.Provider)
	}
	if status.Connected {
		t.Error("expected disconnected (Connected=false)")
	}
}

func TestStatus_NonNilContext(t *testing.T) {
	c := New("localhost", 4532, 500)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancelled context — but implementation ignores it currently
	status, err := c.Status(ctx)
	if err != nil {
		t.Fatalf("Status with cancelled ctx: %v", err)
	}
	if status.Provider != "rigctld" {
		t.Errorf("Provider = %q, want rigctld", status.Provider)
	}
}

func TestNew_ZeroValues(t *testing.T) {
	c := New("", 0, 0)
	if c.Host != "" {
		t.Errorf("Host = %q, want empty", c.Host)
	}
	if c.Port != 0 {
		t.Errorf("Port = %d, want 0", c.Port)
	}
	if c.TimeoutMS != 0 {
		t.Errorf("TimeoutMS = %d, want 0", c.TimeoutMS)
	}
	// Status should still work with zero values.
	status, err := c.Status(context.Background())
	if err != nil {
		t.Fatalf("Status with zero config: %v", err)
	}
	if status.Connected {
		t.Error("expected disconnected")
	}
}
