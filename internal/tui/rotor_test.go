package tui

import (
	"context"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/rotor"
)

// fakeRotorClient implements RotorClient for tests.
type fakeRotorClient struct {
	status  rotor.Status
	err     error
	lastAz  float64
	lastEl  float64
	stopped bool
	name    string
}

func (f *fakeRotorClient) Status(ctx context.Context) (rotor.Status, error) {
	if f.err != nil {
		return rotor.Status{}, f.err
	}
	return f.status, nil
}

func (f *fakeRotorClient) SetPosition(ctx context.Context, az, el float64) error {
	f.lastAz = az
	f.lastEl = el
	return f.err
}

func (f *fakeRotorClient) Stop(ctx context.Context) error {
	f.stopped = true
	return f.err
}

func (f *fakeRotorClient) GetName(ctx context.Context) (string, error) {
	return f.name, f.err
}

func TestClampAz(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0, 0}, {180, 180}, {360, 0}, {361, 1}, {-1, 359}, {720, 0}, {450, 90},
	}
	for _, tt := range tests {
		got := clampAz(tt.in)
		if got != tt.want {
			t.Errorf("clampAz(%f) = %f, want %f", tt.in, got, tt.want)
		}
	}
}

func TestClampEl(t *testing.T) {
	tests := []struct {
		in, want float64
	}{
		{0, 0}, {45, 45}, {90, 90}, {91, 90}, {-5, -5}, {-90, -90}, {-91, -90},
	}
	for _, tt := range tests {
		got := clampEl(tt.in)
		if got != tt.want {
			t.Errorf("clampEl(%f) = %f, want %f", tt.in, got, tt.want)
		}
	}
}

func TestAbsDiff(t *testing.T) {
	tests := []struct{ a, b, want float64 }{
		{10, 5, 5}, {5, 10, 5}, {0, 0, 0}, {-5, 5, 10},
	}
	for _, tt := range tests {
		got := absDiff(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("absDiff(%f,%f) = %f, want %f", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestApplyRotorPoll_Connected(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{status: rotor.Status{Connected: true, Azimuth: 180, Elevation: 5}}
	m.rotor.client = client

	msg := rotorPollMsg{client: client, connected: true, azimuth: 180, elevation: 5}
	cmd := m.applyRotorPoll(msg)
	// First connect triggers name fetch — non-nil cmd is fine.
	_ = cmd
	if !m.rotor.connected {
		t.Error("expected connected")
	}
	if m.rotor.azimuth != 180 {
		t.Errorf("azimuth = %f, want 180", m.rotor.azimuth)
	}
	if m.rotor.elevation != 5 {
		t.Errorf("elevation = %f, want 5", m.rotor.elevation)
	}
}

func TestApplyRotorPoll_Disconnected(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true

	msg := rotorPollMsg{client: client, err: "timeout"}
	cmd := m.applyRotorPoll(msg)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.rotor.connected {
		t.Error("expected disconnected")
	}
}

func TestApplyRotorPoll_StaleClient(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	oldClient := &fakeRotorClient{}
	newClient := &fakeRotorClient{}
	m.rotor.client = newClient

	// Message from old client should be ignored.
	msg := rotorPollMsg{client: oldClient, connected: true, azimuth: 90, elevation: 0}
	cmd := m.applyRotorPoll(msg)
	if cmd != nil {
		t.Error("expected nil cmd for stale message")
	}
	if m.rotor.connected {
		t.Error("stale message should not set connected")
	}
}

func TestApplyRotorPoll_TargetCleared(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{status: rotor.Status{Connected: true, Azimuth: 180, Elevation: 5}}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.targetAz = 180
	m.rotor.targetEl = 5

	msg := rotorPollMsg{client: client, connected: true, azimuth: 180, elevation: 5}
	cmd := m.applyRotorPoll(msg)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.rotor.targetAz != 0 {
		t.Error("target should be cleared when arrived")
	}
	if m.rotor.targetEl != 0 {
		t.Error("target should be cleared when arrived")
	}
}

func TestApplyRotorPoll_TargetNotCleared(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{status: rotor.Status{Connected: true, Azimuth: 175, Elevation: 3}}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.targetAz = 180
	m.rotor.targetEl = 5

	msg := rotorPollMsg{client: client, connected: true, azimuth: 175, elevation: 3}
	cmd := m.applyRotorPoll(msg)
	if cmd != nil {
		t.Error("expected nil cmd")
	}
	if m.rotor.targetAz != 180 {
		t.Error("target should NOT be cleared — not arrived yet")
	}
	if m.rotor.targetEl != 5 {
		t.Error("target should NOT be cleared — not arrived yet")
	}
}

func TestRotorSetPositionCmd(t *testing.T) {
	m := &Model{}
	client := &fakeRotorClient{}
	m.rotor.client = client

	cmd := m.rotorSetPositionCmd(45, 10)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	cmd() // execute the command

	if m.rotor.targetAz != 45 {
		t.Errorf("targetAz = %f, want 45", m.rotor.targetAz)
	}
	if m.rotor.targetEl != 10 {
		t.Errorf("targetEl = %f, want 10", m.rotor.targetEl)
	}
	if client.lastAz != 45 || client.lastEl != 10 {
		t.Errorf("SetPosition not called with correct values: az=%f el=%f", client.lastAz, client.lastEl)
	}
}

func TestHandleRotorKey_CtrlLeft(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.azimuth = 180
	m.screen = screenQSO

	cmd, handled := m.handleRotorKey(ctrlKey(tea.KeyLeft))
	if !handled {
		t.Fatal("expected handled")
	}
	cmd() // execute

	if client.lastAz != 175 {
		t.Errorf("az = %f, want 175", client.lastAz)
	}
}

func TestHandleRotorKey_CtrlRight(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.azimuth = 180
	m.screen = screenQSO

	cmd, handled := m.handleRotorKey(ctrlKey(tea.KeyRight))
	if !handled {
		t.Fatal("expected handled")
	}
	cmd()
	if client.lastAz != 185 {
		t.Errorf("az = %f, want 185", client.lastAz)
	}
}

func TestHandleRotorKey_CtrlUp(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.elevation = 10
	m.screen = screenQSO

	cmd, handled := m.handleRotorKey(ctrlKey(tea.KeyUp))
	if !handled {
		t.Fatal("expected handled")
	}
	cmd()
	if client.lastEl != 15 {
		t.Errorf("el = %f, want 15", client.lastEl)
	}
}

func TestHandleRotorKey_CtrlDown(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true
	m.rotor.elevation = 10
	m.screen = screenQSO

	cmd, handled := m.handleRotorKey(ctrlKey(tea.KeyDown))
	if !handled {
		t.Fatal("expected handled")
	}
	cmd()
	if client.lastEl != 5 {
		t.Errorf("el = %f, want 5", client.lastEl)
	}
}

func TestHandleRotorKey_CtrlEscape(t *testing.T) {
	t.Skip("Ctrl+Escape may not be supported as a Bubble Tea string key")
}

func TestHandleRotorKey_NotConnected(t *testing.T) {
	m := &Model{}
	m.rotor.connected = false
	m.screen = screenQSO

	// handleRotorKey does NOT check connected itself — the guard is in
	// handleFormKey.  This test verifies keys still work if called directly.
	// (The actual guard at the call site prevents this scenario.)
	_, handled := m.handleRotorKey(ctrlKey(tea.KeyLeft))
	// Ctrl+Left matches, so it's handled even though "disconnected".
	if !handled {
		t.Error("key should match — connected check is caller's responsibility")
	}
}

func TestHandleRotorKey_CtrlA_NoGrid(t *testing.T) {
	m := &Model{toasts: NewToastQueue()}
	client := &fakeRotorClient{}
	m.rotor.client = client
	m.rotor.connected = true
	m.screen = screenQSO
	m.App = &app.App{
		Logbook: &config.Logbook{
			Station: config.Station{Grid: ""},
		},
	}
	for i := range m.fields {
		m.fields[i] = textinput.New()
	}

	cmd, handled := m.handleRotorKey(ctrlKey('a'))
	if !handled {
		t.Fatal("expected handled even when no grids")
	}
	if cmd != nil {
		cmd() // won't panic — nil cmd means nothing to execute
	}
}

// ctrlKey creates a tea.KeyPressMsg with Ctrl modifier for testing.
func ctrlKey(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: r, Mod: tea.ModCtrl}
}
