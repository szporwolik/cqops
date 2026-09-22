package tui

import (
	"errors"
	"net"
	"testing"
	"time"

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

// TestDXCConnectCmdReturnsTaggedClientForOwnerInstall verifies the connect
// command no longer touches model state from the worker: settings and the
// existing client are captured at creation, and the result carries the
// client plus the connection generation so the owner loop can install it.
func TestDXCConnectCmdReturnsTaggedClientForOwnerInstall(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	m := newDXCBandFilterModel(t, nil)
	m.App.Config.Integrations.DXC.Enabled = true
	m.App.Config.Integrations.DXC.Host = host
	m.App.Config.Integrations.DXC.Port = port

	cmd := m.dxcConnectCmd()
	msg := execCmd(cmd).(dxcStatusMsg)
	if !msg.online || msg.client == nil {
		t.Fatalf("expected an online result with a client, got %#v", msg)
	}
	if msg.gen != m.dxc.clientGen {
		t.Errorf("result gen = %d, want %d", msg.gen, m.dxc.clientGen)
	}
	if m.dxc.client != nil {
		t.Error("the worker must not publish the client into m.dxc.client")
	}

	// The owner loop installs the returned client.
	m.handleDXCStatus(msg)
	if m.dxc.client != msg.client {
		t.Error("handleDXCStatus should install the client returned by the connect")
	}
	m.dxc.client.Stop()
}

// TestHandleDXCStatusStaleGenerationStopsReturnedClient verifies a connect
// result from before a reset is discarded and its client stopped — the
// client may have been created just for that attempt and must not leak.
func TestHandleDXCStatusStaleGenerationStopsReturnedClient(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.resetDXC() // bumps the generation — any earlier connect is stale

	stale := dxc.NewClient("127.0.0.1", "1", "SP9MOA")
	m.handleDXCStatus(dxcStatusMsg{client: stale, gen: 0, online: true})

	if m.dxc.client != nil {
		t.Error("a stale connect result must not install its client")
	}
	if m.dxc.online {
		t.Error("a stale connect result must not mark the UI online")
	}
	// The stale client must have been stopped by the discard.
	if err := stale.Start(); err == nil {
		t.Error("stale client was not stopped — it can still connect")
	}
}

// TestHandleDXCStatusMatchingGenerationInstallsClient verifies the matching
// generation installs the returned client (even on failure, so retries reuse
// it) and clears the connecting flag.
func TestHandleDXCStatusMatchingGenerationInstallsClient(t *testing.T) {
	m := newDXCBandFilterModel(t, nil)
	m.dxc.connecting = true

	c := dxc.NewClient("127.0.0.1", "1", "SP9MOA")
	defer c.Stop()
	m.handleDXCStatus(dxcStatusMsg{
		client: c, gen: m.dxc.clientGen,
		online: false, err: errors.New("connection refused"),
	})

	if m.dxc.client != c {
		t.Error("matching generation should install the client for retries")
	}
	if m.dxc.connecting {
		t.Error("connecting should clear after the result is processed")
	}
	if m.dxc.online {
		t.Error("online should stay false after a connection failure")
	}
}

// TestMaybeDXCRestoresOnlineAfterClientReconnect reproduces the reported
// stuck-offline bug with a real local TCP server: the client reconnects on
// its own after a drop, but the UI only drained status events while online,
// so the reconnect event was never consumed. The tick must restore the
// online flag from the client's status channel.
func TestMaybeDXCRestoresOnlineAfterClientReconnect(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	m := newDXCBandFilterModel(t, nil)
	m.inetOnline = true
	m.App.Config.Integrations.DXC.Enabled = true
	m.dxc.client = dxc.NewClient(host, port, "SP9MOA")
	defer m.dxc.client.Stop()

	// First connection via the normal tick path.
	cmd := m.maybeDXC()
	if cmd == nil {
		t.Fatal("expected a connect command")
	}
	st, ok := execCmd(cmd).(dxcStatusMsg)
	if !ok || !st.online {
		t.Fatalf("expected an online status, got %#v", execCmd(cmd))
	}
	m.handleDXCStatus(st)
	if !m.dxc.online {
		t.Fatal("online should be true after the first connect")
	}

	// Accept the client's connection, then drop it from the server side.
	accepted := make(chan net.Conn, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			accepted <- c
		}
	}()
	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case <-time.After(10 * time.Second):
		t.Fatal("client never connected")
	}

	// The drop makes the client's readLoop post a disconnect event.
	serverConn.Close()
	disconnectDeadline := time.Now().Add(10 * time.Second)
	for m.dxc.online {
		m.maybeDXC()
		if time.Now().After(disconnectDeadline) {
			t.Fatal("disconnect event was never consumed")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// The client owns reconnection (2s initial backoff) and dials the still
	// listening server again. The tick must consume the reconnect event and
	// restore the online flag — without the fix the UI stays offline forever.
	reconnected := false
	for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
		m.maybeDXC()
		if m.dxc.online {
			reconnected = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !reconnected {
		t.Fatal("client reconnected but the UI never restored the online state")
	}
}
