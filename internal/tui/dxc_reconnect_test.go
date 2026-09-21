package tui

import (
	"errors"
	"testing"

	"github.com/szporwolik/cqops/internal/dxc"
)

// TestHandleDXCStatusFailureKeepsClient is the regression test for abandoned
// reconnecting clients: a connection failure must not drop the client
// pointer. Only resetDXC (config change) and the disabled/offline paths stop
// and replace the client — nil-ing it on failure orphans its reconnect loop
// and leads to duplicate cluster sessions.
func TestHandleDXCStatusFailureKeepsClient(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.client = dxc.NewClient("127.0.0.1", "1", "SP9MOA")
	defer m.dxc.client.Stop()

	m.handleDXCStatus(dxcStatusMsg{online: false, err: errors.New("connection refused")})

	if m.dxc.client == nil {
		t.Error("connection failure orphaned the client — its reconnect loop would keep running unreachable")
	}
	if m.dxc.online {
		t.Error("online should be false after a connection failure")
	}
}

// TestResetDXCStopsAndReplacesClient verifies the config-change path stops
// and joins the old client before dropping it.
func TestResetDXCStopsAndReplacesClient(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.client = dxc.NewClient("127.0.0.1", "1", "SP9MOA")
	m.dxc.online = true

	m.resetDXC()

	if m.dxc.client != nil {
		t.Error("resetDXC should clear the client")
	}
	if m.dxc.online {
		t.Error("resetDXC should clear the online flag")
	}
	if m.dxc.connecting {
		t.Error("resetDXC should clear the connecting flag")
	}
}
