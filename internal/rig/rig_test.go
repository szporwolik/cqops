package rig

import (
	"context"
	"testing"
)

// mockRig implements the Rig interface for testing.
type mockRig struct {
	status RigStatus
	err    error
}

func (m *mockRig) Status(ctx context.Context) (RigStatus, error) {
	return m.status, m.err
}

func TestRigStatus_Fields(t *testing.T) {
	rs := RigStatus{
		Provider:     "flrig",
		Connected:    true,
		FrequencyHz:  14074000,
		FrequencyMHz: 14.074,
		Band:         "20m",
		Mode:         "USB",
		RawMode:      "USB-D",
		Power:        50.0,
	}
	if rs.Provider != "flrig" {
		t.Errorf("Provider = %q, want flrig", rs.Provider)
	}
	if !rs.Connected {
		t.Error("Connected = false, want true")
	}
	if rs.FrequencyHz != 14074000 {
		t.Errorf("FrequencyHz = %d, want 14074000", rs.FrequencyHz)
	}
	if rs.FrequencyMHz != 14.074 {
		t.Errorf("FrequencyMHz = %f, want 14.074", rs.FrequencyMHz)
	}
	if rs.Band != "20m" {
		t.Errorf("Band = %q, want 20m", rs.Band)
	}
	if rs.Mode != "USB" {
		t.Errorf("Mode = %q, want USB", rs.Mode)
	}
	if rs.RawMode != "USB-D" {
		t.Errorf("RawMode = %q, want USB-D", rs.RawMode)
	}
	if rs.Power != 50.0 {
		t.Errorf("Power = %f, want 50.0", rs.Power)
	}
}

func TestRigStatus_ZeroValue(t *testing.T) {
	var rs RigStatus
	if rs.Provider != "" {
		t.Errorf("zero Provider = %q, want empty", rs.Provider)
	}
	if rs.Connected {
		t.Error("zero Connected = true, want false")
	}
	if rs.FrequencyHz != 0 {
		t.Errorf("zero FrequencyHz = %d, want 0", rs.FrequencyHz)
	}
}

func TestRigInterface_Mock(t *testing.T) {
	expected := RigStatus{
		Provider:  "test",
		Connected: true,
		Band:      "40m",
		Mode:      "CW",
		Power:     100.0,
	}
	m := &mockRig{status: expected}
	ctx := context.Background()
	got, err := m.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if got.Provider != expected.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, expected.Provider)
	}
	if got.Band != expected.Band {
		t.Errorf("Band = %q, want %q", got.Band, expected.Band)
	}
	if got.Mode != expected.Mode {
		t.Errorf("Mode = %q, want %q", got.Mode, expected.Mode)
	}
}

func TestRigInterface_MockError(t *testing.T) {
	m := &mockRig{err: context.DeadlineExceeded}
	ctx := context.Background()
	_, err := m.Status(ctx)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRigStatus_Disconnected(t *testing.T) {
	rs := RigStatus{
		Provider:  "rigctld",
		Connected: false,
	}
	if rs.Connected {
		t.Error("expected disconnected")
	}
	if rs.Provider != "rigctld" {
		t.Errorf("Provider = %q, want rigctld", rs.Provider)
	}
}
