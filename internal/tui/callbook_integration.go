package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/locator"
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
	if m.Offline || !m.inetOnline {
		m.lookup.qrzOnline = false
		return nil
	}
	if !m.App.Config.Integrations.QRZ.Enabled {
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
	user := m.App.Config.Integrations.QRZ.User
	pass := m.App.Config.Integrations.QRZ.Pass
	return func() tea.Msg {
		err := qrz.TestConnection(user, pass)
		return qrzStatusMsg{online: err == nil}
	}
}

// qrzLookupCmd returns a tea.Cmd that performs a QRZ lookup.
func (m *Model) qrzLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := qrzLookupFunc(m.App.Config.Integrations.QRZ.User, m.App.Config.Integrations.QRZ.Pass, call)
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
	m.lookup.qrzLookupDone = true
	m.lookup.qrzLookupCall = strings.ToUpper(msg.Call)
	if !m.App.Config.Integrations.QRZ.Enabled || m.App.Config.Integrations.QRZ.User == "" {
		// QRZ not configured — silently skip. All callers guard before firing,
		// this is a belt-and-suspenders check.
		return
	}
	if msg.Err != nil {
		m.toasts.Error(msg.Err.Error())
		m.clearQRZFields()
		m.dxccAutoFill()
		m.prefillContestExchange()
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" {
		m.toasts.Warn("QRZ.com: no data for " + msg.Call)
		m.clearQRZFields()
		m.dxccAutoFill()
		m.prefillContestExchange()
		return
	}
	m.lookup.partnerData = d
	m.invalidatePartnerMapCache()
	if d.ImageURL != "" && d.ImageURL != m.photo.partnerPicURL {
		m.photo.partnerPicURL = d.ImageURL
		m.photo.partnerPicNeedLoad = true
	}
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" {
		// Only fill grid from QRZ if no higher-priority source has set it
		// (manual entry, SOTA, POTA, WWFF, or IOTA take precedence).
		if m.gridSource == gridSourceNone || m.gridSource == gridSourceQRZ {
			m.fields[fieldGrid].SetValue(formatLocator(d.Grid))
			m.rc.pathGrid = strings.ToUpper(formatLocator(d.Grid))
			m.gridSource = gridSourceQRZ
			m.invalidatePartnerMapCache()
			applog.Debug("QRZ: filled partner grid", "grid", d.Grid)
		}
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

	// Recalculate exchange fields — async lookup may have filled grid/zone
	// data that contest exchange markers depend on.
	m.prefillContestExchange()
}

// clearQRZFields clears the form fields that QRZ normally populates
// (name, QTH, grid, country). Called when a QRZ lookup returns no data
// so that stale data from a previous lookup does not persist.
func (m *Model) clearQRZFields() {
	m.fields[fieldName].SetValue("")
	m.fields[fieldQTH].SetValue("")
	m.fields[fieldGrid].SetValue("")
	m.fields[fieldCountry].SetValue("")
	m.rc.formSig = ""
}

// dxccLookup returns the DXCC prefix entry for a callsign, or nil if not found.
func (m *Model) dxccLookup(call string) *dxcc.Prefix {
	if m.App == nil || m.App.DXCC == nil || call == "" {
		return nil
	}
	if !m.App.Config.General.UseCTY {
		return nil
	}
	matches, ok := m.App.DXCC.Find(call)
	if !ok || len(matches) == 0 {
		return nil
	}
	return &matches[0]
}

// dxccAutoFill fills empty QSO form country and grid locator from DXCC.
func (m *Model) dxccAutoFill() {
	if m.App == nil || m.App.DXCC == nil || !m.App.Config.General.UseCTY {
		return
	}
	call := strings.TrimSpace(m.fields[fieldCall].Value())
	if call == "" {
		return
	}
	p := m.dxccLookup(call)
	if p == nil {
		applog.Debug("DXCC: dxccAutoFill no match", "call", call, "dxccLoaded", m.App != nil && m.App.DXCC != nil)
		return
	}
	if p.Name != "" && strings.TrimSpace(m.fields[fieldCountry].Value()) == "" {
		m.fields[fieldCountry].SetValue(p.Name)
		applog.Debug("DXCC: dxccAutoFill country", "call", call, "country", p.Name)
	}
	// Derive approximate grid locator from the DXCC entity's center coordinates.
	if strings.TrimSpace(m.fields[fieldGrid].Value()) == "" {
		grid := locator.LatLonToLocator(p.LatLon, 4)
		gridStr := strings.TrimRight(string(grid[:]), "\x00")
		if len(gridStr) >= 4 {
			m.fields[fieldGrid].SetValue(strings.ToUpper(gridStr[:4]))
			applog.Debug("DXCC: dxccAutoFill grid", "call", call, "grid", strings.ToUpper(gridStr[:4]))
		}
	}
	// Invalidate form render cache — the field values changed but the
	// signature-based cache may not detect it synchronously in all paths.
	m.rc.formSig = ""
}

// updateSCP queries the SCP database for callsigns matching the current
// call field prefix. Only runs when SCP is enabled and >= 3 chars typed.
func (m *Model) updateSCP() {
	m.scpMatches = nil
	m.scpCacheKey = ""

	if m.App == nil || m.App.SCP == nil || !m.App.Config.General.UseSCP {
		return
	}
	prefix := strings.TrimSpace(m.fields[fieldCall].Value())
	if len(prefix) < 3 {
		return
	}
	if prefix == m.scpCacheKey {
		return
	}
	m.scpCacheKey = prefix

	matches, err := m.App.SCP.FindStrings(prefix)
	if err != nil || len(matches) == 0 {
		return
	}
	// Cap at 12 results.
	if len(matches) > 12 {
		matches = matches[:12]
	}
	m.scpMatches = matches
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
		m.lookup.wlLookupCall = strings.ToUpper(msg.Call)
		applog.Warn("Wavelog: lookup error", "call", msg.Call, "error", msg.Err)
		m.toasts.Warn(msg.Err.Error())
		return
	}
	m.lookup.wlLookupDone = true
	m.lookup.wlLookupCall = strings.ToUpper(msg.Call)
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

// lookupsCompleteForCall returns true when both QRZ and Wavelog lookups
// for the given callsign have completed (or are not applicable).
func (m *Model) lookupsCompleteForCall(call string) bool {
	if call == "" {
		return true
	}

	// QRZ: complete if disabled, offline, or lookup done for this exact call.
	qrzEnabled := m.App.Config.Integrations.QRZ.Enabled && m.App.Config.Integrations.QRZ.User != ""
	qrzDone := !qrzEnabled || m.Offline || !m.inetOnline || (m.lookup.qrzLookupDone && m.lookup.qrzLookupCall == call)

	// Wavelog: complete if disabled, offline, or a lookup attempt returned.
	wl := m.App.Logbook.Wavelog
	wlEnabled := wl != nil && wl.Enabled
	wlDone := !wlEnabled || m.Offline || !m.inetOnline || (m.lookup.wlLookupDone && m.lookup.wlLookupCall == call)

	return qrzDone && wlDone
}
