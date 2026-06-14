package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// QRZ and Wavelog callbook lookup integration.
// =============================================================================

// qrzLookupFunc is the function used for QRZ lookups. It defaults to
// qrz.Lookup but can be replaced in tests for mock-based verification.
var qrzLookupFunc = qrz.Lookup

// maybeCheckQRZ returns a tea.Cmd to check QRZ connectivity on the first
// tick and periodically thereafter.
func (m *Model) maybeCheckQRZ() tea.Cmd {
	if !m.App.Config.QRZEnabled {
		m.qrzOnline = false
		return nil
	}
	if m.tickCount != 1 && m.tickCount%healthCheckTicks != 0 {
		return nil
	}
	return m.checkQRZCmd()
}

// checkQRZCmd returns a tea.Cmd that tests QRZ.com connectivity.
func (m *Model) checkQRZCmd() tea.Cmd {
	user := m.App.Config.QRZUser
	pass := m.App.Config.QRZPass
	return func() tea.Msg {
		err := qrz.TestConnection(user, pass)
		return qrzStatusMsg{online: err == nil}
	}
}

// qrzLookupCmd returns a tea.Cmd that performs a QRZ lookup.
func (m *Model) qrzLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := qrzLookupFunc(m.App.Config.QRZUser, m.App.Config.QRZPass, call)
		return qrzResultMsg{Call: call, Data: data, Err: err}
	}
}

// qrzLookup returns a tea.Cmd to look up a callsign via QRZ, with rate-limiting.
func (m *Model) qrzLookup(call string) tea.Cmd {
	if call == "" {
		return nil
	}
	if time.Since(m.qrzLastLook) < 3*time.Second && strings.EqualFold(call, m.qrzLastCall) {
		return nil
	}
	m.qrzLastLook = time.Now()
	m.qrzLastCall = call
	applog.Info("QRZ: looking up", "call", call)
	return m.qrzLookupCmd(call)
}

// wlLookupCmd returns a tea.Cmd that performs a Wavelog private lookup.
func (m *Model) wlLookupCmd(call, band, mode string) tea.Cmd {
	return func() tea.Msg {
		data, err := wavelog.PrivateLookup(
			m.App.Config.Wavelog.URL,
			m.App.Config.Wavelog.APIKey,
			call, band, mode,
		)
		return wlResultMsg{Call: call, Data: data, Err: err}
	}
}

// wlLookup returns a tea.Cmd to look up a callsign via Wavelog, with rate-limiting.
func (m *Model) wlLookup(call string) tea.Cmd {
	if call == "" {
		return nil
	}
	if !m.App.Config.Wavelog.Enabled || m.App.Config.Wavelog.URL == "" || m.App.Config.Wavelog.APIKey == "" {
		return nil
	}
	if !m.inetOnline {
		return nil
	}
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	if time.Since(m.wlLastLook) < 5*time.Second &&
		strings.EqualFold(call, m.wlLastCall) &&
		band == m.wlLastBand && mode == m.wlLastMode {
		return nil
	}
	m.wlLastLook = time.Now()
	m.wlLastCall = call
	m.wlLastBand = band
	m.wlLastMode = mode
	applog.Info("Wavelog: looking up", "call", call)
	return m.wlLookupCmd(call, band, mode)
}

// fillQRZData fills the QSO form from QRZ lookup result data.
func (m *Model) fillQRZData(msg qrzResultMsg) {
	if msg.Call == "" {
		return
	}
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		return
	}
	if !m.App.Config.QRZEnabled || m.App.Config.QRZUser == "" {
		m.toasts.Warn("QRZ not configured")
		return
	}
	if msg.Err != nil {
		m.toasts.Error("QRZ error: " + msg.Err.Error())
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" {
		m.toasts.Warn("QRZ: no data for " + msg.Call)
		return
	}
	m.partnerData = d
	m.invalidatePartnerMapCache()
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" {
		m.fields[fieldGrid].SetValue(formatLocator(d.Grid))
		applog.Debug("QRZ: filled partner grid", "grid", d.Grid)
	}
	if d.QTH != "" {
		m.fields[fieldQTH].SetValue(d.QTH)
	}
	if d.Country != "" && m.fields[fieldCountry].Value() == "" {
		m.fields[fieldCountry].SetValue(d.Country)
	}
	m.autoFillRST()
	m.toasts.Info("QRZ: " + d.Callsign + " " + d.Name)
}

// fillWLData stores Wavelog private lookup result data.
func (m *Model) fillWLData(msg wlResultMsg) {
	if msg.Call == "" {
		return
	}
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		return
	}
	if msg.Err != nil {
		m.wlLookupDone = true
		applog.Warn("Wavelog: lookup error", "call", msg.Call, "error", msg.Err)
		m.toasts.Warn("Wavelog: " + msg.Err.Error())
		return
	}
	m.wlLookupDone = true
	if msg.Data == nil {
		return
	}
	applog.InfoDetail("Wavelog: lookup OK", fmt.Sprintf("call=%s worked=%v confirmed=%v", msg.Call, msg.Data.Worked(), msg.Data.DXCCConfirmed()))
	m.wlPrivateData = msg.Data
	name := ""
	if msg.Data.Name() != "" {
		name = " " + msg.Data.Name()
	}
	m.toasts.Info("Wavelog: " + msg.Call + name)
}
