package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Screen routing tests
// =============================================================================

func TestScreenRouting_F2PartnerWithCall(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("SP9XXX")
	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF2})
	if !handled {
		t.Error("F2 should be handled when call is present")
	}
	if m.screen != screenPartner && m.screen != screenImage {
		t.Errorf("F2 with call should navigate to partner/image screen, got %v", m.screen)
	}
}

func TestScreenRouting_F2NoCall(t *testing.T) {
	m := newTestModel()
	prev := m.screen
	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF2})
	if !handled {
		t.Error("F2 should be handled even with no call (shows warning)")
	}
	if m.screen != prev {
		t.Errorf("F2 with no call should not change screen, got %v", m.screen)
	}
}

func TestScreenRouting_F1ReturnsToQSO(t *testing.T) {
	m := newTestModel()
	m.screen = screenPartner
	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF1})
	if !handled {
		t.Error("F1 should be handled")
	}
	if m.screen != screenQSO {
		t.Errorf("F1 should return to QSO form, got %v", m.screen)
	}
}

func TestScreenRouting_F4TogglesDXC(t *testing.T) {
	m := newTestModel()
	m.App.Config.DXC.Enabled = true
	m.dxc.online = true
	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF4})
	if !handled {
		t.Error("F4 should be handled")
	}
	if m.screen != screenDXC {
		t.Errorf("F4 should navigate to DXC screen, got %v", m.screen)
	}
}

func TestScreenRouting_EscapeFromDXC(t *testing.T) {
	m := newTestModel()
	m.screen = screenDXC
	_, _ = m.handleDXCUpdate(tea.KeyPressMsg{Code: tea.KeyEscape}, nil)
	if m.screen != screenQSO {
		t.Errorf("Esc from DXC should return to QSO, got %v", m.screen)
	}
}

func TestScreenRouting_EscapeFromPartner(t *testing.T) {
	m := newTestModel()
	m.screen = screenPartner
	_, _ = m.handlePartnerUpdate(tea.KeyPressMsg{Code: tea.KeyEscape}, nil)
	if m.screen != screenQSO {
		t.Errorf("Esc/F1 from partner should return to QSO, got %v", m.screen)
	}
}

func TestScreenRouting_F10QuitDialog(t *testing.T) {
	m := newTestModel()
	_, handled := m.handleGlobalKeys(tea.KeyPressMsg{Code: tea.KeyF10})
	if !handled {
		t.Error("F10 should be handled")
	}
	if m.confirm == nil {
		t.Error("F10 should show quit confirmation dialog")
	}
}

// =============================================================================
// Async result-message tests — flrig
// =============================================================================

func TestFlrigResult_Connected(t *testing.T) {
	m := newTestModel()
	msg := flrigResultMsg{
		connected: true,
		freq:      14.250,
		mode:      "USB",
		power:     50,
	}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("flrigResultMsg should be consumed")
	}
	if !m.rig.connected {
		t.Error("rig should be connected")
	}
	if m.rig.freq != 14.250 {
		t.Errorf("rig freq = %f, want 14.250", m.rig.freq)
	}
}

func TestFlrigResult_Disconnected(t *testing.T) {
	m := newTestModel()
	m.rig.connected = true
	msg := flrigResultMsg{
		connected: false,
		err:       "timeout",
	}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("flrigResultMsg with error should be consumed")
	}
	if m.rig.connected {
		t.Error("rig should be disconnected after error")
	}
}

func TestFlrigResult_ZeroValuesNoPanic(t *testing.T) {
	m := newTestModel()
	msg := flrigResultMsg{}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("zero flrigResultMsg should be consumed")
	}
}

// =============================================================================
// Async result-message tests — QRZ
// =============================================================================

func TestQRZResult_Success(t *testing.T) {
	m := newTestModel()
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	m.fields[fieldCall].SetValue("SP9XXX")
	msg := qrzResultMsg{
		Call: "SP9XXX",
		Data: &qrz.CallData{Callsign: "SP9XXX", Name: "Test", Grid: "JO90"},
	}
	m.fillQRZData(msg)
	if m.lookup.partnerData == nil {
		t.Fatal("partnerData should be set")
	}
	if m.lookup.partnerData.Callsign != "SP9XXX" {
		t.Errorf("partner callsign = %q", m.lookup.partnerData.Callsign)
	}
}

func TestQRZResult_NilDataNoPanic(t *testing.T) {
	m := newTestModel()
	msg := qrzResultMsg{Call: "SP9XXX"}
	m.fillQRZData(msg)
}

// =============================================================================
// Async result-message tests — Wavelog
// =============================================================================

func TestWLResult_SetsLookupDone(t *testing.T) {
	m := newTestModel()
	msg := wlResultMsg{
		Call: "SP9XXX",
		Data: &wavelog.PrivateLookupResult{},
	}
	m.fillWLData(msg)
	if !m.lookup.wlLookupDone {
		t.Error("wlLookupDone should be true after result")
	}
}

// =============================================================================
// Internet check tests
// =============================================================================

func TestInetResult_Online(t *testing.T) {
	m := newTestModel()
	m.inetOnline = false
	consumed := m.handleAsyncMessages(inetResultMsg(true))
	if !consumed {
		t.Error("inetResultMsg should be consumed")
	}
	if !m.inetOnline {
		t.Error("inetOnline should be true")
	}
}

func TestInetResult_Offline(t *testing.T) {
	m := newTestModel()
	m.inetOnline = true
	consumed := m.handleAsyncMessages(inetResultMsg(false))
	if !consumed {
		t.Error("inetResultMsg should be consumed")
	}
	if m.inetOnline {
		t.Error("inetOnline should be false")
	}
}

// =============================================================================
// Wavelog status/upload tests
// =============================================================================

func TestWLStatus_Success(t *testing.T) {
	m := newTestModel()
	msg := wlStatusMsg{online: true, stationName: "TestStation", stationLabel: "Test"}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("wlStatusMsg should be consumed")
	}
	if !m.lookup.wlOnline {
		t.Error("wlOnline should be true")
	}
	if m.lookup.wlStationName != "TestStation" {
		t.Errorf("wlStationName = %q", m.lookup.wlStationName)
	}
}

func TestWLUploadResult_Success(t *testing.T) {
	m := newTestModel()
	msg := wlUploadResultMsg{ok: true, call: "SP9XXX"}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("wlUploadResultMsg should be consumed")
	}
}

func TestWLUploadResult_Error(t *testing.T) {
	m := newTestModel()
	msg := wlUploadResultMsg{ok: false, call: "SP9XXX", err: nil}
	consumed := m.handleAsyncMessages(msg)
	if !consumed {
		t.Error("wlUploadResultMsg (error) should be consumed")
	}
}

// =============================================================================
// DXC spots stored tests
// =============================================================================

func TestDXCSpotsStored_ForcesRebuild(t *testing.T) {
	m := newTestModel()
	m.dxc.tableReady = true
	msg := dxcSpotsStoredMsg{calls: []string{"SP9XXX"}}
	// dxcSpotsStoredMsg is handled in main Update(), not handleAsyncMessages.
	_, _ = m.Update(msg)
	if m.dxc.tableReady {
		t.Error("dxcSpotsStored should set tableReady=false")
	}
}

// =============================================================================
// Tick message tests
// =============================================================================

func TestTickMsg_DoesNotPanic(t *testing.T) {
	m := newTestModel()
	msg := tickMsg(time.Now())
	_, _ = m.Update(msg)
}

func TestTickMsg_WSJTXWatchdog(t *testing.T) {
	m := newTestModel()
	m.wsjtx.online = true
	m.wsjtx.lastSeen = time.Now().Add(-20 * time.Second)
	msg := tickMsg(time.Now())
	_, _ = m.Update(msg)
	if m.wsjtx.online {
		t.Error("WSJT-X should be marked offline after 15s without status")
	}
}

// =============================================================================
// WSJT-X status application tests
// =============================================================================

func TestApplyWSJTXStatus_NewCall(t *testing.T) {
	m := newTestModel()
	m.applyWSJTXStatus("SP9XXX", "JO90", 14250000, "FT8", "", "-10", "", false)
	if m.fields[fieldCall].Value() != "SP9XXX" {
		t.Errorf("call field = %q, want SP9XXX", m.fields[fieldCall].Value())
	}
	if m.fields[fieldGrid].Value() != "JO90" {
		t.Errorf("grid field = %q, want JO90", m.fields[fieldGrid].Value())
	}
	if !m.wsjtx.online {
		t.Error("wsjtxOnline should be true")
	}
}

func TestApplyWSJTXStatus_SameCallPreservesPartner(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("SP9XXX")
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9XXX", Name: "Test"}
	m.applyWSJTXStatus("SP9XXX", "", 0, "", "", "", "", false)
	if m.lookup.partnerData == nil {
		t.Error("partnerData should be preserved when same call")
	}
}

func TestApplyWSJTXStatus_EmptyCallPreservesState(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("SP9XXX")
	m.lookup.partnerData = &qrz.CallData{Callsign: "SP9XXX", Name: "Test"}
	m.applyWSJTXStatus("", "", 0, "", "", "", "", false)
	if m.fields[fieldCall].Value() != "SP9XXX" {
		t.Errorf("call field should be preserved, got %q", m.fields[fieldCall].Value())
	}
	if m.lookup.partnerData == nil {
		t.Error("partnerData should be preserved when empty call")
	}
}

// =============================================================================
// handlePendingRequests tests
// =============================================================================

func TestHandlePendingRequests_NoPending(t *testing.T) {
	m := newTestModel()
	cmd, handled := m.handlePendingRequests(nil)
	if handled {
		t.Error("handlePendingRequests with no pending should not be handled")
	}
	if cmd != nil {
		t.Error("should return nil cmd when no pending")
	}
}

func TestHandlePendingRequests_QRZNeedWithCall(t *testing.T) {
	m := newTestModel()
	m.lookup.qrzNeed = true
	m.lookup.qrzCall = "SP9XXX"
	m.App.Config.QRZ.Enabled = true
	m.App.Config.QRZ.User = "testuser"
	cmd, handled := m.handlePendingRequests(nil)
	if !handled {
		t.Error("handlePendingRequests with qrzNeed should be handled")
	}
	if cmd == nil {
		t.Error("should return a lookup command")
	}
	if m.lookup.qrzNeed {
		t.Error("qrzNeed should be cleared")
	}
}

func TestHandlePendingRequests_QRZNeedDisabled(t *testing.T) {
	m := newTestModel()
	m.lookup.qrzNeed = true
	m.lookup.qrzCall = "SP9XXX"
	m.App.Config.QRZ.Enabled = false
	cmd, handled := m.handlePendingRequests(nil)
	if !handled {
		t.Error("handlePendingRequests should handle disabled QRZ (triggers DXC lookup)")
	}
	if m.lookup.qrzNeed {
		t.Error("qrzNeed should be cleared")
	}
	_ = cmd
}
