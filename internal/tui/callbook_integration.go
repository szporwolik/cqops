package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/locator"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/callook"
	"github.com/szporwolik/cqops/internal/hamqth"
	"github.com/szporwolik/cqops/internal/qrzcom"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Callbook multi-provider lookup integration.
// =============================================================================

// callbookRegLookup is the test seam for the callbook registry.
var callbookRegLookup = func(m *Model, call string) (*callbook.Result, error) {
	if m.callbookRegistry == nil {
		applog.Debug("Callbook: no registry, skipping lookup", "call", call)
		return nil, nil
	}
	applog.Debug("Callbook: lookup start", "call", call, "providers", m.callbookRegistry.Len())
	data, err := m.callbookRegistry.Lookup(call)
	if data != nil {
		applog.Debug("Callbook: lookup ok", "call", call, "provider", data.Provider,
			"name", data.Name, "grid", data.Grid, "country", data.Country, "qth", data.QTH)
		// Base-call fallback: if the looked-up call is a suffix call
		// (e.g. SP9MOA/P) and the result has no name (CTY-only backfill),
		// also try the base call and merge richer data into the result.
		if m.App.Config.Integrations.Callbook.BaseCallFallback {
			base := qso.DeriveBaseCall(call)
			if base != "" && !strings.EqualFold(base, call) && data.Name == "" {
				applog.Debug("Callbook: suffix call has no name, trying base", "call", call, "base", base)
				baseData, baseErr := m.callbookRegistry.Lookup(base)
				if baseData != nil {
					callbook.MergeInto(data, baseData)
					applog.Debug("Callbook: base-call fallback merged", "base", base, "provider", baseData.Provider)
					_ = baseErr
				}
			}
		}
		return data, nil
	}
	if err != nil {
		applog.Debug("Callbook: lookup error", "call", call, "error", err)
		return nil, err
	}
	// Base-call fallback: if SP9MOA/P was not found, try SP9MOA.
	if m.App.Config.Integrations.Callbook.BaseCallFallback {
		base := qso.DeriveBaseCall(call)
		if base != "" && !strings.EqualFold(base, call) {
			applog.Debug("Callbook: no result for suffix call, trying base", "call", call, "base", base)
			data, err := m.callbookRegistry.Lookup(base)
			if data != nil {
				applog.Debug("Callbook: base-call fallback ok", "base", base, "provider", data.Provider)
			} else {
				applog.Debug("Callbook: base-call fallback no result", "base", base, "err", err)
			}
			return data, err
		}
	}
	applog.Debug("Callbook: no result", "call", call)
	return nil, nil
}

// buildCallbookRegistry creates the provider registry from configuration.
func buildCallbookRegistry(a *app.App) *callbook.Registry {
	var providers []callbook.Provider

	// Logbook provider — searches past local QSOs.
	// Default priority 100 (tried before QRZ). Enabled by default.
	lc := a.Config.Integrations.Callbook.Logbook
	if lc.Enabled || lc.Priority == 0 {
		p := lc.Priority
		if p == 0 {
			p = 100 // default
		}
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		if a.DB != nil {
			providers = append(providers, NewLogbookCallbookProvider(a.DB, p))
		}
	}

	// QRZ provider.
	cfg := a.Config.Integrations.Callbook.QRZ
	if cfg.Enabled && cfg.User != "" {
		p := cfg.Priority
		if p == 0 {
			p = 50
		}
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		providers = append(providers, qrzcom.NewClientWithPriority(cfg.User, cfg.Pass, p))
	}

	// HamQTH provider — free callsign database.
	hqCfg := a.Config.Integrations.Callbook.HamQTH
	if hqCfg.Enabled && hqCfg.User != "" {
		p := hqCfg.Priority
		if p == 0 {
			p = 45
		}
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		providers = append(providers, hamqth.NewClientWithPriority(hqCfg.User, hqCfg.Pass, p))
	}

	// Callook.info provider — free US callsign database, no auth required.
	coCfg := a.Config.Integrations.Callbook.Callook
	if coCfg.Enabled {
		p := coCfg.Priority
		if p == 0 {
			p = 40
		}
		if p < 0 {
			p = 0
		}
		if p > 100 {
			p = 100
		}
		providers = append(providers, callook.NewClientWithPriority(p))
	}

	// Wavelog provider — only when explicitly enabled and configured.
	wc := a.Config.Integrations.Callbook.Wavelog
	if wc.Enabled {
		// Find any logbook with Wavelog configured.
		var wlURL, wlAPIKey string
		for _, lb := range a.Config.Logbooks {
			if lb.Wavelog != nil && lb.Wavelog.Enabled && lb.Wavelog.URL != "" && lb.Wavelog.APIKey != "" {
				wlURL = lb.Wavelog.URL
				wlAPIKey = lb.Wavelog.APIKey
				break
			}
		}
		if wlURL != "" && wlAPIKey != "" {
			p := wc.Priority
			if p == 0 {
				p = 10
			}
			if p < 0 {
				p = 0
			}
			if p > 100 {
				p = 100
			}
			providers = append(providers, NewWavelogCallbookProvider(wlURL, wlAPIKey, p))
		}
	}

	// CTY.DAT prefix lookup — always-on fallback, runs last at priority 1.
	// Fills country, grid, and CQ/ITU zone from the DXCC prefix database.
	if a.DXCC != nil {
		providers = append(providers, NewCTYProvider(a.DXCC, 1))
	}

	if len(providers) == 0 {
		applog.Debug("Callbook: no providers configured")
		return nil
	}
	applog.Debug("Callbook: registry built", "count", len(providers))
	return callbook.NewRegistry(providers)
}

// maybeCheckCallbook returns a tea.Cmd to check callbook provider
// connectivity at startup (first tick).
func (m *Model) maybeCheckCallbook() tea.Cmd {
	if m.Offline || !m.inetOnline {
		m.lookup.qrzOnline = false
		return nil
	}
	if m.callbookRegistry == nil {
		m.lookup.qrzOnline = false
		return nil
	}
	if m.tickCount != 1 && !m.lookup.qrzForceCheck {
		return nil
	}
	m.lookup.qrzForceCheck = false
	return m.checkCallbookCmd()
}

// callbookLookupCmd returns a tea.Cmd that performs a cascading callbook
// lookup through all registered providers.
func (m *Model) callbookLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := callbookRegLookup(m, call)
		return callbookResultMsg{Call: call, Data: data, Err: err}
	}
}

// checkCallbookCmd returns a tea.Cmd that tests connectivity for all
// registered callbook providers.
func (m *Model) checkCallbookCmd() tea.Cmd {
	return func() tea.Msg {
		if m.callbookRegistry == nil {
			return qrzStatusMsg{online: false}
		}
		// Report online if any provider is available.
		// Logbook and CTY providers work locally; QRZ requires network.
		online := true
		if m.App.Config.Integrations.Callbook.QRZ.Enabled {
			err := qrzcom.TestConnection(m.App.Config.Integrations.Callbook.QRZ.User, m.App.Config.Integrations.Callbook.QRZ.Pass)
			online = err == nil
		}
		return qrzStatusMsg{online: online}
	}
}

// callbookLookup returns a tea.Cmd to look up a callsign via all registered
// callbook providers in priority order, with rate-limiting.
func (m *Model) callbookLookup(call string) tea.Cmd {
	if call == "" {
		return nil
	}
	if m.callbookRegistry == nil {
		return nil
	}
	// Already completed for this call — no need to re-query.
	if m.lookup.qrzLookupDone && strings.EqualFold(call, m.lookup.qrzLookupCall) {
		return nil
	}
	if time.Since(m.lookup.qrzLast) < 3*time.Second && strings.EqualFold(call, m.lookup.qrzLastCall) {
		return nil
	}
	m.lookup.qrzLast = time.Now()
	m.lookup.qrzLastCall = call
	applog.Info("Callbook: looking up", "call", call)
	return m.callbookLookupCmd(call)
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
	if m.Offline || !m.inetOnline {
		return nil
	}
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.URL == "" || wl.APIKey == "" {
		return nil
	}
	// Already completed for this call — no need to re-query.
	if m.lookup.wlLookupDone && strings.EqualFold(call, m.lookup.wlLookupCall) {
		return nil
	}
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := qso.NormalizeRigMode(m.fields[fieldMode].Value())
	if time.Since(m.lookup.wlLast) < 5*time.Second &&
		strings.EqualFold(call, m.lookup.wlLastCall) &&
		band == m.lookup.wlLastBand && mode == m.lookup.wlLastMode {
		return nil
	}
	m.lookup.wlLast = time.Now()
	m.lookup.wlLastCall = call
	m.lookup.wlLastBand = band
	m.lookup.wlLastMode = mode
	m.lookup.wlDispatchTime = time.Now() // for timeout detection
	applog.Info("Wavelog: looking up", "call", call)
	return m.wlLookupCmd(call, band, mode)
}

// fillCallbookData fills the QSO form from callbook lookup result data.
func (m *Model) fillCallbookData(msg callbookResultMsg) {
	if msg.Call == "" {
		return
	}
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		return
	}
	m.lookup.qrzLookupDone = true
	m.lookup.qrzLookupCall = strings.ToUpper(msg.Call)
	if msg.Err != nil {
		m.toasts.Error("Callbook: " + msg.Err.Error())
		m.clearQRZFields()
		m.prefillContestExchange()
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" {
		m.showCallbookToast(msg.Call) // show "no data" toast if everything is done
		m.clearQRZFields()
		m.prefillContestExchange()
		return
	}
	m.lookup.partnerData = d
	m.invalidatePartnerMapCache()
	m.rc.helpSig = "" // force help bar refresh (may now show F2 Photo)
	if d.ImageURL != "" && d.ImageURL != m.photo.partnerPicURL {
		m.photo.partnerPicURL = d.ImageURL
		m.photo.partnerPicNeedLoad = true
	}
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" {
		// Reject fake/default grids from QRZ (e.g. AA00aa, JJ00aa).
		grid := strings.ToUpper(strings.TrimSpace(d.Grid))
		if strings.HasPrefix(grid, "AA") || strings.HasPrefix(grid, "JJ") || len(grid) < 4 {
			applog.Debug("QRZ: grid rejected as fake/default", "grid", d.Grid)
			grid = ""
		}
		// Only fill grid from callbook if no higher-priority source has set it
		// (manual entry, SOTA, POTA, WWFF, or IOTA take precedence).
		if grid != "" && (m.gridSource == gridSourceNone || m.gridSource == gridSourceCallbook) {
			m.fields[fieldGrid].SetValue(formatLocator(grid))
			m.rc.pathGrid = strings.ToUpper(formatLocator(grid))
			m.gridSource = gridSourceCallbook
			m.invalidatePartnerMapCache()
			applog.Debug("Callbook: filled partner grid", "grid", grid)
		}
	}
	if d.QTH != "" {
		m.fields[fieldQTH].SetValue(d.QTH)
	}
	if d.Country != "" {
		m.fields[fieldCountry].SetValue(d.Country)
	}
	m.autoFillRST()

	// Show consolidated toast after all providers have reported in.
	m.showCallbookToast(d.Callsign)

	// Force immediate dashboard partner push so provider badges appear
	// without waiting for the next throttled tick cycle.
	m.forcePushDashboardPartner()

	// Recalculate exchange fields — async lookup may have filled grid/zone
	// data that contest exchange markers depend on.
	m.prefillContestExchange()
}

// showCallbookToast shows a single consolidated toast listing all providers
// that contributed data for the given callsign. It waits until both the
// callbook registry lookup and the Wavelog private lookup have completed
// (or are not applicable), then shows one toast. The toast is only shown
// once per callsign to avoid duplicate notifications.
func (m *Model) showCallbookToast(call string) {
	if call == "" {
		return
	}
	// Already shown for this call — suppress duplicate.
	if m.lookup.callbookToastCall == call {
		return
	}
	// Wait for both lookups to complete (or become not applicable).
	if !m.lookupsCompleteForCall(call) {
		return
	}

	m.lookup.callbookToastCall = call

	// Collect provider names that contributed data.
	var parts []string

	// Providers from the callbook registry.
	if m.lookup.partnerData != nil && strings.EqualFold(m.lookup.partnerData.Callsign, call) {
		for _, p := range m.lookup.partnerData.Providers {
			// Deduplicate (different providers may have the same name).
			found := false
			for _, existing := range parts {
				if existing == p {
					found = true
					break
				}
			}
			if !found {
				parts = append(parts, p)
			}
		}
	}

	// Wavelog private lookup — only counts if the Wavelog callbook
	// provider is enabled, returned data, and not already listed
	// from the registry (avoid "Wavelog, Wavelog" duplicate).
	if m.App.Config.Integrations.Callbook.Wavelog.Enabled && m.lookup.wlPrivateData != nil {
		found := false
		for _, p := range parts {
			if p == "Wavelog" {
				found = true
				break
			}
		}
		if !found {
			parts = append(parts, "Wavelog")
		}
	}

	if len(parts) > 0 {
		// Suppress toast when CTY.DAT is the only provider — it's an
		// always-on fallback that silently fills country/grid.
		if len(parts) == 1 && parts[0] == "CTY.DAT" {
			return
		}
		// Build the list: "QRZ, Logbook, Wavelog"
		list := ""
		for i, p := range parts {
			if i > 0 {
				list += ", "
			}
			list += p
		}
		m.toasts.Info("Callbook: " + call + " — " + list)
	} else if m.callbookRegistry == nil && !m.App.Config.Integrations.Callbook.Wavelog.Enabled {
		// No providers available — show a one-time hint.
		if !m.lookup.noProviderWarned {
			m.lookup.noProviderWarned = true
			ctyAvailable := m.App != nil && m.App.DXCC != nil && m.App.Config.General.UseCTY
			if ctyAvailable {
				m.toasts.Warn("Callbook: no providers configured — using CTY.DAT backfill (country + grid only)")
			} else {
				m.toasts.Warn("Callbook: no providers configured — enable QRZ, Wavelog, or Logbook in Callbook settings")
			}
		}
	} else {
		// Registry exists but no provider returned data for this call.
		m.toasts.Warn("Callbook: " + call + " — no data")
	}
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
//
// Chain behaviour:
//   - Normal (IsFallback=false): if the result is stale (user moved on),
//     re-trigger lookup for the current form call. On success, fill the
//     form. If the suffix result is sparse (no name or grid) and
//     BaseCallFallback is enabled, immediately dispatch a second lookup
//     for the base call (without suffix) with IsFallback=true.
//   - Fallback (IsFallback=true): the result is from a base-call lookup
//     triggered by a sparse suffix. Skip the stale check (the form still
//     shows the suffix), verify the base matches, and fill the form.
func (m *Model) fillWLData(msg wlResultMsg) tea.Cmd {
	if msg.Call == "" {
		return nil
	}
	formCall := qso.NormalizeCall(m.fields[fieldCall].Value())

	// --- Fallback result (base-call lookup triggered by sparse suffix) ---
	if msg.IsFallback {
		if formCall == "" {
			return nil
		}
		formBase := qso.DeriveBaseCall(formCall)
		if !strings.EqualFold(formBase, msg.Call) {
			// User moved to a different base call — drop.
			return nil
		}
		if msg.Err != nil {
			applog.Warn("Wavelog: base fallback error", "base", msg.Call, "error", msg.Err)
			return nil
		}
		if msg.Data == nil {
			return nil
		}
		applog.InfoDetail("Wavelog: base fallback OK", fmt.Sprintf("base=%s worked=%v confirmed=%v", msg.Call, msg.Data.Worked(), msg.Data.DXCCConfirmed()))
		if m.App.Config.Integrations.Callbook.Wavelog.Enabled {
			wlFillForm(m, msg.Data)
		}
		m.forcePushDashboardPartner()
		return nil
	}

	// --- Normal (direct) result ---
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		// Stale result — the user cycled away. Re-trigger lookup for the
		// current form call so the pending state eventually resolves.
		applog.Debug("Wavelog: stale result, re-triggering",
			"result_call", msg.Call, "form_call", formCall)
		return m.wlLookup(formCall)
	}
	if msg.Err != nil {
		m.lookup.wlLookupDone = true
		m.lookup.wlLookupCall = strings.ToUpper(msg.Call)
		applog.Warn("Wavelog: lookup error", "call", msg.Call, "error", msg.Err)
		m.showCallbookToast(msg.Call)
		return nil
	}
	m.lookup.wlLookupDone = true
	m.lookup.wlLookupCall = strings.ToUpper(msg.Call)
	if msg.Data == nil {
		m.showCallbookToast(msg.Call)
		return nil
	}
	applog.InfoDetail("Wavelog: lookup OK", fmt.Sprintf("call=%s worked=%v confirmed=%v", msg.Call, msg.Data.Worked(), msg.Data.DXCCConfirmed()))
	m.lookup.wlPrivateData = msg.Data

	if m.App.Config.Integrations.Callbook.Wavelog.Enabled {
		wlFillForm(m, msg.Data)
	}

	// When the suffix result is sparse and BaseCallFallback is enabled,
	// chain a second lookup for the base call (without suffix).
	if m.App.Config.Integrations.Callbook.BaseCallFallback {
		base := qso.DeriveBaseCall(msg.Call)
		if base != "" && !strings.EqualFold(base, msg.Call) && msg.Data.Name() == "" && msg.Data.Grid() == "" {
			applog.Debug("Wavelog: suffix sparse, chaining base lookup", "suffix", msg.Call, "base", base)
			return m.wlFallbackLookup(base)
		}
	}

	// Force immediate dashboard partner push so badges reflect Wavelog data.
	m.forcePushDashboardPartner()

	// Show consolidated toast listing all contributing providers.
	m.showCallbookToast(msg.Call)
	return nil
}

// wlFallbackLookup returns a tea.Cmd that performs a Wavelog private lookup
// for the base call (without suffix/slash). The result carries IsFallback=true
// so fillWLData can distinguish it from the original suffix lookup.
func (m *Model) wlFallbackLookup(call string) tea.Cmd {
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.URL == "" || wl.APIKey == "" {
		return nil
	}
	if m.Offline || !m.inetOnline {
		return nil
	}
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := qso.NormalizeRigMode(m.fields[fieldMode].Value())
	return func() tea.Msg {
		data, err := wavelog.PrivateLookup(wl.URL, wl.APIKey, call, band, mode)
		return wlResultMsg{Call: call, Data: data, Err: err, IsFallback: true}
	}
}

// wlFillForm fills empty QSO form fields from Wavelog private lookup data.
// Only writes fields that are currently blank — higher-priority providers
// (QRZ, Logbook) in the callbook registry already ran first.
// Grid is an exception: Wavelog's grid (actual station location) replaces
// the rough center-point estimate from CTY/DXCC when available.
func wlFillForm(m *Model, data *wavelog.PrivateLookupResult) {
	if data == nil {
		return
	}

	if strings.TrimSpace(m.fields[fieldName].Value()) == "" {
		if n := data.Name(); n != "" {
			m.fields[fieldName].SetValue(n)
		}
	}
	if strings.TrimSpace(m.fields[fieldQTH].Value()) == "" {
		if q := data.QTH(); q != "" {
			m.fields[fieldQTH].SetValue(q)
		}
	}
	// Grid: overwrite if currently blank OR if it came from the callbook
	// (CTY center-point estimate) — Wavelog has the real station location.
	if m.gridSource == gridSourceNone || m.gridSource == gridSourceCallbook {
		if g := data.Grid(); len(g) >= 4 {
			// Preserve full precision — Wavelog returns 6-char grids (e.g. JO91mn).
			grid := strings.ToUpper(g)
			if len(grid) > 6 {
				grid = grid[:6]
			}
			m.fields[fieldGrid].SetValue(grid)
			m.rc.pathGrid = grid
			m.gridSource = gridSourceCallbook
			m.invalidatePartnerMapCache()
		}
	}
	if strings.TrimSpace(m.fields[fieldCountry].Value()) == "" {
		if c := data.Country(); c != "" {
			m.fields[fieldCountry].SetValue(c)
		}
	}
	m.rc.formSig = ""
}

// lookupsCompleteForCall returns true when both QRZ and Wavelog lookups
// for the given callsign have completed (or are not applicable).
func (m *Model) lookupsCompleteForCall(call string) bool {
	if call == "" {
		return true
	}

	// QRZ: complete if disabled, offline, or lookup done for this exact call.
	qrzEnabled := m.App.Config.Integrations.Callbook.QRZ.Enabled && m.App.Config.Integrations.Callbook.QRZ.User != ""
	qrzDone := !qrzEnabled || m.Offline || !m.inetOnline || (m.lookup.qrzLookupDone && m.lookup.qrzLookupCall == call)

	// Wavelog: complete if disabled, offline, or a lookup attempt returned.
	wl := m.App.Logbook.Wavelog
	wlEnabled := wl != nil && wl.Enabled
	wlDone := !wlEnabled || m.Offline || !m.inetOnline || (m.lookup.wlLookupDone && m.lookup.wlLookupCall == call)

	return qrzDone && wlDone
}
