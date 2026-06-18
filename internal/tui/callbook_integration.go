package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/dxcc"
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

// maybeCheckQRZ returns a tea.Cmd to check QRZ connectivity once at
// startup (first tick). Periodic re-checking is unnecessary — the
// internet health check already monitors connectivity.
func (m *Model) maybeCheckQRZ() tea.Cmd {
	if !m.App.Config.QRZ.Enabled {
		m.lookup.qrzOnline = false
		return nil
	}
	if m.tickCount != 1 {
		return nil
	}
	return m.checkQRZCmd()
}

// checkQRZCmd returns a tea.Cmd that tests QRZ.com connectivity.
func (m *Model) checkQRZCmd() tea.Cmd {
	user := m.App.Config.QRZ.User
	pass := m.App.Config.QRZ.Pass
	return func() tea.Msg {
		err := qrz.TestConnection(user, pass)
		return qrzStatusMsg{online: err == nil}
	}
}

// qrzLookupCmd returns a tea.Cmd that performs a QRZ lookup.
func (m *Model) qrzLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := qrzLookupFunc(m.App.Config.QRZ.User, m.App.Config.QRZ.Pass, call)
		return qrzResultMsg{Call: call, Data: data, Err: err}
	}
}

// qrzLookup returns a tea.Cmd to look up a callsign via QRZ, with rate-limiting.
func (m *Model) qrzLookup(call string) tea.Cmd {
	if call == "" {
		return nil
	}
	if time.Since(m.lookup.qrzLast) < 3*time.Second && strings.EqualFold(call, m.lookup.qrzLastCall) {
		return nil
	}
	m.lookup.qrzLast = time.Now()
	m.lookup.qrzLastCall = call
	applog.Info("QRZ: looking up", "call", call)
	return m.qrzLookupCmd(call)
}

// wlLookupCmd returns a tea.Cmd that performs a Wavelog private lookup.
func (m *Model) wlLookupCmd(call, band, mode string) tea.Cmd {
	wl := m.App.Logbook.Wavelog
	return func() tea.Msg {
		data, err := wavelog.PrivateLookup(
			wl.URL,
			wl.APIKey,
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
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.URL == "" || wl.APIKey == "" {
		return nil
	}
	if !m.inetOnline {
		return nil
	}
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	if time.Since(m.lookup.wlLast) < 5*time.Second &&
		strings.EqualFold(call, m.lookup.wlLastCall) &&
		band == m.lookup.wlLastBand && mode == m.lookup.wlLastMode {
		return nil
	}
	m.lookup.wlLast = time.Now()
	m.lookup.wlLastCall = call
	m.lookup.wlLastBand = band
	m.lookup.wlLastMode = mode
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
	if !m.App.Config.QRZ.Enabled || m.App.Config.QRZ.User == "" {
		// QRZ not configured — silently skip. All callers guard before firing,
		// this is a belt-and-suspenders check.
		return
	}
	if msg.Err != nil {
		m.toasts.Error(msg.Err.Error())
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" {
		m.toasts.Warn("QRZ.com: no data for " + msg.Call)
		return
	}
	m.lookup.partnerData = d
	m.invalidatePartnerMapCache()
	if d.ImageURL != "" && d.ImageURL != m.photo.partnerPicURL {
		m.photo.partnerPicNeedLoad = true
	}
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" {
		m.fields[fieldGrid].SetValue(formatLocator(d.Grid))
		m.rc.pathGrid = strings.ToUpper(formatLocator(d.Grid))
		applog.Debug("QRZ: filled partner grid", "grid", d.Grid)
	}
	if d.QTH != "" {
		m.fields[fieldQTH].SetValue(d.QTH)
	}
	if d.Country != "" {
		m.fields[fieldCountry].SetValue(d.Country)
	}
	m.autoFillRST()
	m.toasts.Info("QRZ.com: " + d.Callsign + " " + d.Name)

	// After QRZ filled what it could, fill remaining empty country/continent
	// from DXCC prefix lookup if available.
	m.dxccAutoFill()
}

// dxccLookup returns the DXCC prefix entry for a callsign, or nil if not found.
func (m *Model) dxccLookup(call string) *dxcc.Prefix {
	if m.App == nil || m.App.DXCC == nil || call == "" {
		return nil
	}
	matches, ok := m.App.DXCC.Find(call)
	if !ok || len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

// dxccAutoFill fills empty QSO form country/continent fields from DXCC.
func (m *Model) dxccAutoFill() {
	call := strings.TrimSpace(m.fields[fieldCall].Value())
	if call == "" {
		return
	}
	p := m.dxccLookup(call)
	if p == nil {
		return
	}
	if p.Name != "" && strings.TrimSpace(m.fields[fieldCountry].Value()) == "" {
		m.fields[fieldCountry].SetValue(p.Name)
	}
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
		m.lookup.wlLookupDone = true
		applog.Warn("Wavelog: lookup error", "call", msg.Call, "error", msg.Err)
		m.toasts.Warn(msg.Err.Error())
		return
	}
	m.lookup.wlLookupDone = true
	if msg.Data == nil {
		return
	}
	applog.InfoDetail("Wavelog: lookup OK", fmt.Sprintf("call=%s worked=%v confirmed=%v", msg.Call, msg.Data.Worked(), msg.Data.DXCCConfirmed()))
	m.lookup.wlPrivateData = msg.Data
	name := ""
	if msg.Data.Name() != "" {
		name = " " + msg.Data.Name()
	}
	m.toasts.Info("Wavelog: " + msg.Call + name)
}
