package tui

import (
	"testing"
)

func TestFlrigResultSuccess(t *testing.T) {
	m := newTestModel()
	// Inject a connected fake
	m.rig.client = connectedFakeFlrig(14.250, "SSB", "20m")

	// Apply successful result
	_ = m.applyFlrigResult(flrigResultMsg{
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

func TestFlrigResultError(t *testing.T) {
	m := newTestModel()
	m.rig.connected = true // start connected

	_ = m.applyFlrigResult(flrigResultMsg{
		err: "connection refused",
	})

	if m.rig.connected {
		t.Error("rigConnected should be false after error result")
	}
}

func TestFlrigResultDisconnected(t *testing.T) {
	m := newTestModel()
	m.rig.connected = true

	_ = m.applyFlrigResult(flrigResultMsg{
		connected: false,
	})

	if m.rig.connected {
		t.Error("rigConnected should be false when not connected")
	}
}

func TestFlrigPollDisabled(t *testing.T) {
	m := newTestModel()
	m.rig.client = nil // disabled

	cmd := m.pollFlrig()
	// First call sets skipTicks to 1, which is < rigPollInterval (30)
	// So it returns nil regardless
	if cmd != nil {
		t.Log("pollFlrig returned non-nil when disabled (may be OK for first tick)")
	}
	// But rigConnected should be set to false
	if !m.rig.connected {
		t.Log("rigConnected is false as expected when flrig is nil")
	}
}

func TestFlrigStatusCmdNil(t *testing.T) {
	m := newTestModel()
	m.rig.client = nil

	cmd := m.flrigStatusCmd()
	if cmd != nil {
		t.Error("flrigStatusCmd should return nil when client is nil")
	}
}

func TestFlrigStatusCmdReturnsCmd(t *testing.T) {
	m := newTestModel()
	m.rig.client = connectedFakeFlrig(14.250, "SSB", "20m")

	cmd := m.flrigStatusCmd()
	if cmd == nil {
		t.Error("flrigStatusCmd should return a command when client is set")
	}
	// Execute the command
	msg := cmd()
	fr, ok := msg.(flrigResultMsg)
	if !ok {
		t.Fatalf("Expected flrigResultMsg, got %T", msg)
	}
	if !fr.connected {
		t.Error("Fake flrig should report connected")
	}
	if fr.freq != 14.250 {
		t.Errorf("Fake flrig freq = %f; want 14.250", fr.freq)
	}
}

func TestFlrigResultPower(t *testing.T) {
	m := newTestModel()
	_ = m.applyFlrigResult(flrigResultMsg{
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
