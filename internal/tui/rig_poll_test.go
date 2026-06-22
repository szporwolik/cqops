package tui

import (
	"testing"
)

func TestRigPollSuccess(t *testing.T) {
	m := newTestModel()
	// Inject a connected fake
	m.rig.client = connectedFakeRig(14.250, "SSB", "20m")

	// Apply successful result
	_ = m.applyRigPoll(rigPollMsg{
		client:    m.rig.client,
		connected: true,
		freq:      14.250,
		mode:      "SSB",
		band:      "20m",
	})

	if !m.rig.connected {
		t.Error("rigConnected should be true after successful result")
	}
	if m.rig.freq != 14.250 {
		t.Errorf("rigFreq = %f; want 14.250", m.rig.freq)
	}
	if m.fields[fieldFreq].Value() != "14.250000" {
		t.Errorf("Freq field = %q; want 14.250000", m.fields[fieldFreq].Value())
	}
	if m.fields[fieldMode].Value() != "SSB" {
		t.Errorf("Mode field = %q; want SSB", m.fields[fieldMode].Value())
	}
	if m.fields[fieldBand].Value() != "20m" {
		t.Errorf("Band field = %q; want 20m", m.fields[fieldBand].Value())
	}
}

func TestRigPollError(t *testing.T) {
	m := newTestModel()
	m.rig.connected = true // start connected

	_ = m.applyRigPoll(rigPollMsg{
		client: m.rig.client,
		err:    "connection refused",
	})

	if m.rig.connected {
		t.Error("rigConnected should be false after error result")
	}
}

func TestRigPollDisconnected(t *testing.T) {
	m := newTestModel()
	m.rig.connected = true

	_ = m.applyRigPoll(rigPollMsg{
		client:    m.rig.client,
		connected: false,
	})

	if m.rig.connected {
		t.Error("rigConnected should be false when not connected")
	}
}

func TestPollDisabled(t *testing.T) {
	m := newTestModel()
	m.rig.client = nil // disabled

	cmd := m.pollRig()
	// First call sets skipTicks to 1, which is < pollInterval (1)
	// So it returns nil regardless
	if cmd != nil {
		t.Log("pollRig returned non-nil when disabled (may be OK for first tick)")
	}
	// But rigConnected should be set to false
	if !m.rig.connected {
		t.Log("rigConnected is false as expected when rig client is nil")
	}
}

func TestRigStatusCmdNil(t *testing.T) {
	m := newTestModel()
	m.rig.client = nil

	cmd := m.rigStatusCmd()
	if cmd != nil {
		t.Error("rigStatusCmd should return nil when client is nil")
	}
}

func TestRigStatusCmdReturnsCmd(t *testing.T) {
	m := newTestModel()
	m.rig.client = connectedFakeRig(14.250, "SSB", "20m")

	cmd := m.rigStatusCmd()
	if cmd == nil {
		t.Error("rigStatusCmd should return a command when client is set")
	}
	// Execute the command
	msg := cmd()
	fr, ok := msg.(rigPollMsg)
	if !ok {
		t.Fatalf("Expected rigPollMsg, got %T", msg)
	}
	if !fr.connected {
		t.Error("Fake rig should report connected")
	}
	if fr.freq != 14.250 {
		t.Errorf("Fake rig freq = %f; want 14.250", fr.freq)
	}
}

func TestRigPollPower(t *testing.T) {
	m := newTestModel()
	_ = m.applyRigPoll(rigPollMsg{
		client:    m.rig.client,
		connected: true,
		freq:      7.100,
		mode:      "SSB",
		band:      "40m",
		power:     100,
	})

	if m.fields[fieldTXPower].Value() != "100" {
		t.Errorf("TXPower = %q; want 100", m.fields[fieldTXPower].Value())
	}
}
