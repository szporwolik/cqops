package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/geo"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
)

// =============================================================================
// HTTP Server — built-in lightweight HTTP server for CQOps
// =============================================================================

// httpStatusMsg is sent when the HTTP server connection state changes.
type httpStatusMsg struct {
	online bool
	err    error
	client *dashboard.Server
}

// httpReconnectDelay is the pause before retrying after a bind failure.
const httpReconnectDelay = 10 * time.Second

// maybeHTTP manages the built-in HTTP server lifecycle.
// Called from the tick handler. Returns a command when an action is pending.
func (m *Model) maybeHTTP() tea.Cmd {
	cfg := m.App.Config.Integrations.HTTPServer

	// Not enabled — ensure stopped.
	if !cfg.Enabled {
		if m.http.client != nil {
			m.http.client.Stop()
			m.http.client = nil
		}
		m.http.online = false
		m.http.err = nil
		m.http.lastAttempt = time.Time{}
		m.http.restart = false
		return nil
	}

	// Restart requested — stop the old server first.
	if m.http.restart {
		if m.http.client != nil {
			m.http.client.Stop()
			m.http.client = nil
		}
		m.http.online = false
		m.http.err = nil
		m.http.restart = false
		// Fall through to start a new one below.
	}

	// Already running — drain status channel for state changes.
	if m.http.client != nil && m.http.online {
		select {
		case online, ok := <-m.http.client.Status():
			if ok && !online {
				applog.Warn("HTTP server: stopped unexpectedly")
				m.http.online = false
				m.http.err = m.http.client.Error()
				if m.http.err != nil {
					m.toasts.Error("HTTP server: " + m.http.err.Error())
				}
				m.http.client.Stop()
				m.http.client = nil
				return nil
			}
		default:
		}
		return nil
	}

	// Not running but should be — start the server.
	// Backoff: wait httpReconnectDelay between attempts when the previous one failed.
	if !m.http.lastAttempt.IsZero() && time.Since(m.http.lastAttempt) < httpReconnectDelay {
		return nil
	}
	m.http.lastAttempt = time.Now()

	addr := cfg.Address
	if addr == "" {
		addr = "0.0.0.0"
	}
	port := cfg.Port
	if port == "" {
		port = "8073"
	}

	return func() tea.Msg {
		applog.Info("HTTP server: starting", "addr", addr, "port", port)
		client := dashboard.New(addr, port)
		client.Start()

		// Wait for initial status.
		select {
		case online := <-client.Status():
			if online {
				applog.Info("HTTP server: started", "addr", client.Addr())
				return httpStatusMsg{online: true, client: client}
			}
			err := client.Error()
			if err == nil {
				err = &httpBindError{addr: client.Addr()}
			}
			applog.Error("HTTP server: failed to start", "addr", client.Addr(), "error", err)
			return httpStatusMsg{online: false, err: err, client: client}
		case <-time.After(3 * time.Second):
			return httpStatusMsg{online: false, err: &httpBindError{addr: addr + ":" + port}}
		}
	}
}

// httpBindError is a sentinel error for HTTP server bind failures.
type httpBindError struct {
	addr string
}

func (e *httpBindError) Error() string {
	return "cannot bind " + e.addr + " — port in use or address unavailable"
}

// restartHTTPServer schedules a server restart on the next tick.
// Called when the integration config is saved. Does NOT stop the old
// server here — maybeHTTP handles the full stop→start cycle with proper
// synchronisation so we never try to bind before the old socket is freed.
func (m *Model) restartHTTPServer() {
	m.http.online = false
	m.http.err = nil
	m.http.lastAttempt = time.Time{}
	m.http.restart = true
	applog.Info("HTTP server: restart scheduled")
}

// pushLoggedQSOToDashboard converts a saved QSO into a QSOView and pushes it
// to the dashboard state. Called inline from saveQSO() after persist succeeds.
func (m *Model) pushLoggedQSOToDashboard(qs *qso.QSO) {
	if m.http.client == nil || !m.http.online {
		return
	}
	ds := m.http.client.State()

	t, err := time.Parse("20060102 150405", qs.QSODate+" "+qs.TimeOn)
	if err != nil {
		t = time.Now().UTC()
	}

	view := dashboard.QSOView{
		ID:        fmt.Sprintf("%d", qs.ID),
		TimeUTC:   t,
		Call:      qs.Call,
		Band:      qs.Band,
		Mode:      qs.Mode,
		Submode:   qs.Submode,
		Frequency: fmt.Sprintf("%.3f MHz", qs.Freq),
		RSTSent:   qs.RSTSent,
		RSTRcvd:   qs.RSTRcvd,
		Grid:      qs.GridSquare,
		Country:   qs.Country,
		Operator:  qs.Operator,
	}
	if qs.Freq > 0 {
		view.FrequencyHz = int64(qs.Freq * 1_000_000)
	}

	ds.AddLoggedQSO(view)
	// AddLoggedQSO already publishes EventQSOLogged + EventRecentQSOs
	// so the browser sees the new QSO instantly. The next tick will push
	// a fresh DB-sourced list via pushDashboardRecent.
	applog.Debug("dashboard: pushed logged QSO", "call", qs.Call)
}

// stripANSI removes ANSI escape sequences (CSI codes) from a string.
// Used to clean Lip Gloss-styled text before sending it to the browser.
var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return reANSI.ReplaceAllString(s, "")
}

func clubLogoURL(cfgURL string) string {
	if cfgURL != "" {
		return cfgURL
	}
	return "https://raw.githubusercontent.com/szporwolik/cqops/main/assets/other/gh-logo.png"
}

// unitForDashboard returns "imperial" or "metric" for the dashboard.
func unitForDashboard(unit string) string {
	if unit == "imperial" {
		return "imperial"
	}
	return "metric"
}

// eventStartDate returns the event start date as YYYYMMDD, or empty if not set.
func (m *Model) eventStartDate() string {
	es := m.App.Config.Integrations.HTTPServer.EventStart
	if es == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", es)
	if err != nil {
		return ""
	}
	return t.Format("20060102")
}

// pushDashboardPartner pushes QRZ/Wavelog lookup data. Only publishes when
// actual lookup data (from QRZ) is available, not partial form fields.
func (m *Model) pushDashboardPartner(ds *dashboard.State, call string) {
	if call == "" {
		if partnerEmpty {
			return // already reported empty, skip log
		}
		partnerEmpty = true
		lastPushedPartner = cachedPartner{}
		ds.SetPartner(nil)
		applog.Debug("dashboard: partner cleared (empty call)")
		return
	}
	partnerEmpty = false
	// Clear stale partner data when the callsign changed and the old
	// QRZ lookup no longer matches the current form call.
	if m.lookup.partnerData != nil && !strings.EqualFold(m.lookup.partnerData.Callsign, call) {
		if lastPushedPartner.Call != "" {
			lastPushedPartner = cachedPartner{}
			ds.SetPartner(nil)
			applog.Debug("dashboard: partner cleared (call changed)",
				"old", m.lookup.partnerData.Callsign, "new", call)
		}
		return
	}
	// Only push when QRZ/Wavelog lookup data is actually available
	// and matches the current callsign.
	if m.lookup.partnerData == nil {
		return
	}

	pi := &dashboard.PartnerInfo{
		Call:   call,
		Source: "qrz",
	}
	d := m.lookup.partnerData
	pi.Name = d.Name
	pi.QTH = d.QTH
	pi.Country = d.Country
	pi.Grid = d.Grid
	fmt.Sscanf(d.Lat, "%f", &pi.Lat)
	fmt.Sscanf(d.Lon, "%f", &pi.Lon)
	pi.ImageURL = d.ImageURL

	// Build provider badges with URLs for external providers.
	// CTY.DAT is an always-on offline fallback — do not show a badge.
	if len(d.Providers) > 0 {
		pi.CallbookProviders = make([]dashboard.ProviderBadge, 0, len(d.Providers))
		seen := map[string]bool{}
		for _, name := range d.Providers {
			if name == "CTY.DAT" || seen[name] {
				continue // skip always-on fallback and duplicates (base-call merge)
			}
			seen[name] = true
			badge := dashboard.ProviderBadge{Name: name}
			switch name {
			case "QRZ.com":
				badge.URL = "https://www.qrz.com/db/" + call
			case "Wavelog":
				if wl := m.App.Logbook.Wavelog; wl != nil && wl.URL != "" {
					badge.URL = strings.TrimRight(wl.URL, "/")
				}
			}
			pi.CallbookProviders = append(pi.CallbookProviders, badge)
		}
	}

	// Skip push + debug log when nothing changed since last tick.
	cur := cachedPartner{
		Call: pi.Call, Name: pi.Name, QTH: pi.QTH,
		Country: pi.Country, Grid: pi.Grid, ImageURL: pi.ImageURL,
		Providers: strings.Join(d.Providers, ","),
	}
	if cur != lastPushedPartner {
		lastPushedPartner = cur
		ds.SetPartner(pi)
		applog.Debug("dashboard: partner pushed",
			"call", pi.Call,
			"name", pi.Name,
			"country", pi.Country,
			"grid", pi.Grid,
			"hasPhoto", pi.ImageURL != "")
	}
}

// forcePushDashboardPartner pushes partner data to the dashboard immediately,
// bypassing the normal tick-based throttling. Called from fillCallbookData
// and fillWLData after lookup results arrive, so provider badges appear
// on the dashboard without a 2-second delay.
func (m *Model) forcePushDashboardPartner() {
	if m.http.client == nil || !m.http.online {
		return
	}
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call == "" {
		return
	}
	ds := m.http.client.State()
	if ds == nil {
		return
	}
	// No explicit throttle — pushDashboardPartner has its own change
	// detection via cachedPartner. If the provider list grew (e.g.
	// base-call fallback merged richer data), the push fires. If
	// nothing changed, it's a no-op.
	applog.Debug("dashboard: force-pushing partner", "call", call)
	m.pushDashboardPartner(ds, call)
}

// internetCallbook returns the name and URL template of the highest-priority
// online callbook provider (QRZ.com only — Wavelog is intentionally excluded
// because its search URLs are instance-specific and not a public callsign
// lookup). Returns empty strings if QRZ is not enabled or configured.
func (m *Model) internetCallbook() (name, urlTemplate string) {
	qrz := m.App.Config.Integrations.QRZ

	if qrz.Enabled && qrz.User != "" {
		return "QRZ.com", "https://www.qrz.com/db/{CALL}"
	}
	return "", ""
}

// lastDashboardPushTick throttles pushDashboardState to every 2 ticks (~2 s)
// instead of every tick (~1 s). On low-end hardware this halves the per-tick
// overhead of building rig/WSJT-X/solar/partner structs. The 2 s interval is
// still fast enough for SSE push to feel real-time in the browser.
var lastDashboardPushTick int

// Called from the tick handler every second. Most Set* calls early-exit because
// nothing changed (same rig freq, same WSJT-X state, etc.).
func (m *Model) pushDashboardState() {
	if m.http.client == nil || !m.http.online {
		return
	}

	// Throttle to every 2 ticks (~2 s) to reduce CPU on low-end hardware.
	// The dashboard SSE push is change-detected, so a slightly slower poll
	// rate is imperceptible while significantly reducing per-tick work.
	if m.tickCount-lastDashboardPushTick < 2 {
		return
	}
	lastDashboardPushTick = m.tickCount

	// Fast: rig, operator, logbook, station, active QSO — cheap, change-detected.
	m.pushDashboardFast()

	ds := m.http.client.State()

	// --- Refresh today QSOs + stats for the map (rate-limited) ---
	now := time.Now()
	if now.Sub(lastTodayPush) > 5*time.Second {
		lastTodayPush = now
		m.pushDashboardToday(ds)
		m.pushDashboardStats(ds)
		m.pushDashboardRecent(ds)
	}

	// --- APRS stations for the local map (rate-limited, 30 s, or on-demand) ---
	if now.Sub(lastAPRSPush) > 30*time.Second || m.App.ConsumeAPRSRefresh() {
		lastAPRSPush = now
		m.pushDashboardAPRS(ds)
	}
}

// pushDashboardFast pushes only the light-weight, change-detected parts:
// app, display, station, operator, logbook, rig, WSJT-X, active QSO, partner.
// No DB queries for today/recent QSOs. Guarded by lastFastTick to avoid
// duplicate work when called from both handleTick and applyRigPoll.
func (m *Model) pushDashboardFast() {
	if m.http.client == nil || !m.http.online {
		return
	}
	if m.tickCount == lastFastTick {
		return
	}
	lastFastTick = m.tickCount
	ds := m.http.client.State()

	// --- App ---
	ds.SetApp("CQOps", version.Resolved())

	// --- Display config ---
	cfg := m.App.Config.Integrations.HTTPServer
	icName, icURL := m.internetCallbook()
	ds.SetDisplay(dashboard.DisplayConfig{
		Header1:              cfg.Header1,
		Header2:              cfg.Header2,
		ClubLogo:             clubLogoURL(cfg.ClubLogo),
		QRLink:               cfg.QRLink,
		MapTileURL:           cfg.MapTileURL,
		MapAttrib:            cfg.MapAttrib,
		DrawLines:            true,
		MaxLines:             250,
		HighlightLastQSO:     true,
		Units:                unitForDashboard(m.App.Config.General.Units),
		Theme:                cfg.Theme,
		InternetCallbookName: icName,
		InternetCallbookURL:  icURL,
	})

	// --- Station ---
	st := m.App.Logbook.Station
	rp, hasRig := m.App.Config.Rigs[st.RigName]
	grid := m.effectiveGrid()
	stationInfo := dashboard.StationInfo{
		Callsign: st.Callsign,
		Locator:  grid,
	}
	if grid != "" {
		stationInfo.Lat, stationInfo.Lon = gridToLatLon(grid)
	}
	if hasRig {
		stationInfo.Radio = rp.Name
		if rp.Model != "" {
			stationInfo.Radio = rp.Model
		}
		stationInfo.Antenna = rp.Antenna
	}
	// Power: prefer the form value (updated by rig polling), fall back to config.
	if pw := m.fields[fieldTXPower].Value(); pw != "" {
		var p int
		fmt.Sscanf(pw, "%d", &p)
		stationInfo.PowerW = p
	} else if hasRig {
		if pw := st.RigPower(m.App.Config.Rigs); pw != "" {
			var p int
			fmt.Sscanf(pw, "%d", &p)
			stationInfo.PowerW = p
		}
	}
	// APRS radius for local map circle.
	if aprsCfg := m.App.Logbook.APRS; aprsCfg != nil && aprsCfg.Enabled && aprsCfg.RadiusKm > 0 {
		stationInfo.AprsRadiusKm = float64(aprsCfg.RadiusKm)
	}
	ds.SetStation(stationInfo)

	// --- Operator ---
	opInfo := dashboard.OperatorInfo{}
	if op := m.App.Logbook.ActiveOperator; op != "" {
		if opCfg, ok := m.App.Config.Operators[op]; ok {
			opInfo.Callsign = opCfg.Callsign
			opInfo.Name = opCfg.Name
		}
	}
	ds.SetOperator(opInfo)

	// --- Logbook ---
	ds.SetLogbook(dashboard.LogbookInfo{
		Name: config.LogbookDisplayName(m.App.Logbook),
	})

	// --- Rig ---
	rigInfo := dashboard.RigInfo{
		Enabled:   hasRig && rp.RadioBackend != "",
		Connected: m.rig.connected,
		Name:      m.rig.name,
	}
	if m.rig.connected {
		rigInfo.FrequencyHz = int64(m.rig.freq * 1_000_000)
		rigInfo.Frequency = fmt.Sprintf("%.3f MHz", m.rig.freq)
		rigInfo.Band = qso.DeriveBand(m.rig.freq)
		rigInfo.Mode = m.fields[fieldMode].Value()
		rigInfo.Submode = m.fields[fieldSubmode].Value()
		rigInfo.UpdatedAtUTC = time.Now().UTC()
	}
	ds.SetRig(rigInfo)

	// --- WSJT-X ---
	wsjtxE := hasRig && rp.WsjtxEnabled
	wsjtxInfo := dashboard.WSJTXInfo{
		Enabled:   wsjtxE,
		Connected: m.wsjtx.online,
	}
	if m.wsjtx.online {
		wsjtxInfo.LastMessage = m.wsjtx.txMsg
		wsjtxInfo.UpdatedAtUTC = time.Now().UTC()
	}
	ds.SetWSJTX(wsjtxInfo)

	// --- Solar ---
	if m.solar.data != nil {
		ds.SetSolar(dashboard.SolarInfo{
			SolarFlux:      m.solar.data.SolarFlux,
			AIndex:         m.solar.data.AIndex,
			KIndex:         m.solar.data.KIndex,
			Sunspots:       m.solar.data.Sunspots,
			BandConditions: m.solar.data.Bands,
			UpdatedAt:      "now",
		})
	}

	// --- Active QSO ---
	call := strings.TrimSpace(m.fields[fieldCall].Value())
	if call != "" {
		// Always build the active QSO and push. SetActiveQSO has
		// internal change detection — it only publishes when the
		// call or its dupe/new flags actually changed. This lets
		// late-arriving flags (from DB queries) reach the dashboard
		// even after the initial call push.
		aq := &dashboard.ActiveQSO{
			State:  "editing",
			Source: "form",
			Call:   call,
		}
		if b := m.fields[fieldBand].Value(); b != "" {
			aq.Band = b
		}
		if mo := m.fields[fieldMode].Value(); mo != "" {
			aq.Mode = mo
		}
		if sm := m.fields[fieldSubmode].Value(); sm != "" {
			aq.Submode = sm
		}
		if f := m.fields[fieldFreq].Value(); f != "" {
			aq.Frequency = f
		}
		if g := m.fields[fieldGrid].Value(); g != "" {
			aq.Grid = g
		}
		if n := m.fields[fieldName].Value(); n != "" {
			aq.Name = n
		}
		if qth := m.fields[fieldQTH].Value(); qth != "" {
			aq.QTH = qth
		}
		if c := m.fields[fieldCountry].Value(); c != "" {
			aq.Country = c
		}
		if rs := m.fields[fieldRSTSent].Value(); rs != "" {
			aq.RSTSent = rs
		}
		if rr := m.fields[fieldRSTRcvd].Value(); rr != "" {
			aq.RSTRcvd = rr
		}
		// Resolved reference names line (SOTA/POTA/WWFF/IOTA).
		// Strip ANSI escape codes — the TUI form uses Lip Gloss styling
		// (e.g. red for unresolved refs), but the browser renders HTML.
		if rn := stripANSI(m.buildRefNamesLine()); rn != "" {
			aq.RefNames = rn
		}
		aq.UpdatedAtUTC = time.Now().UTC()
		// Dupe / new call / new DXCC flags.
		// Only recompute when the callsign changed (avoid DB queries every tick).
		if call != "" && call != pushDashboardLastCall && m.App.DB != nil {
			pushDashboardLastCall = call
			stats, err := store.GetLogbookStats(m.App.DB, call, aq.Band, aq.Mode)
			if err == nil {
				aq.IsNewCall = stats.QSOCount == 0
				aq.IsDupe = m.dupe
				if aq.IsNewCall && aq.Country != "" {
					aq.IsNewDXCC = !m.countryWorkedBefore(aq.Country)
				}
			}
			// Cache for reuse on later ticks when call hasn't changed.
			lastActiveDupe = aq.IsDupe
			lastActiveNewCall = aq.IsNewCall
			lastActiveNewDXCC = aq.IsNewDXCC
		} else if call != "" {
			// Call unchanged — reuse cached flags so SetActiveQSO
			// doesn't see a spurious reset to false and re-publish.
			// Dupe is an exception: checkDupe may have updated m.dupe
			// after the initial push, and it's a cheap in-memory read.
			aq.IsDupe = m.dupe
			aq.IsNewCall = lastActiveNewCall
			aq.IsNewDXCC = lastActiveNewDXCC
			// Country may arrive late (manual entry or QRZ lookup).
			// When it does, recompute the NEW DXCC flag if still unknown.
			if aq.IsNewCall && aq.Country != "" && !lastActiveNewDXCC && m.App.DB != nil {
				if !m.countryWorkedBefore(aq.Country) {
					aq.IsNewDXCC = true
					lastActiveNewDXCC = true
				}
			}
		}
		// Skip the push + debug log when nothing changed since last tick.
		cur := cachedAQSO{
			Call: aq.Call, Band: aq.Band, Mode: aq.Mode, Submode: aq.Submode,
			Frequency: aq.Frequency, Grid: aq.Grid, Name: aq.Name, QTH: aq.QTH,
			Country: aq.Country, RSTSent: aq.RSTSent, RSTRcvd: aq.RSTRcvd,
			RefNames: aq.RefNames, IsDupe: aq.IsDupe, IsNewCall: aq.IsNewCall,
			IsNewDXCC: aq.IsNewDXCC,
		}
		if cur != lastPushedAQSO {
			lastPushedAQSO = cur
			ds.SetActiveQSO(aq)
			applog.Debug("dashboard: pushed active QSO",
				"call", call,
				"band", aq.Band,
				"mode", aq.Mode,
				"dupe", aq.IsDupe,
				"newCall", aq.IsNewCall,
				"newDxcc", aq.IsNewDXCC)
		}
	} else if call == "" && ds.LastActiveCall() != "" {
		ds.ClearActiveQSO()
		lastPushedAQSO = cachedAQSO{}
	}

	// --- Partner info (QRZ/Wavelog lookup results) ---
	m.pushDashboardPartner(ds, call)
}

// lastTodayPush rate-limits the DB query for today QSOs (map data).
var lastTodayPush time.Time

// lastAPRSPush rate-limits the APRS station push to the dashboard local map.
var lastAPRSPush time.Time

// pushDashboardLastCall tracks the last call for which we computed dupe/new flags.
var pushDashboardLastCall string

// lastActiveDupe/newCall/newDXCC cache computed flags for reuse on subsequent
// ticks when the call hasn't changed (avoids DB queries and spurious resets).
var lastActiveDupe, lastActiveNewCall, lastActiveNewDXCC bool

// lastPushedAQSO caches the last active-QSO fields pushed to the dashboard
// so we skip ds.SetActiveQSO and the debug log when nothing changed.
// UpdatedAtUTC is excluded — it always changes.
type cachedAQSO struct {
	Call, Band, Mode, Submode, Frequency, Grid, Name, QTH, Country string
	RSTSent, RSTRcvd, RefNames                                     string
	IsDupe, IsNewCall, IsNewDXCC                                   bool
}

var lastPushedAQSO cachedAQSO

// lastPushedPartner caches the last partner fields pushed to the dashboard.
type cachedPartner struct {
	Call, Name, QTH, Country, Grid, ImageURL string
	Providers                                string // comma-separated provider names
}

var lastPushedPartner cachedPartner

// partnerEmpty tracks whether we already reported partner as empty.
var partnerEmpty bool

// lastRecentIDs holds the last pushed QSO ID list for change detection.
var lastRecentIDs []int64

// forcePushDashboardRecent clears the change-detection cache and pushes.
// Use when QSO fields (country, grid, distance) change without ID changes,
// e.g. after WSJT-X enrichment.
func (m *Model) forcePushDashboardRecent(ds *dashboard.State) {
	lastRecentIDs = nil
	m.pushDashboardRecent(ds)
}

// lastFastTick prevents pushDashboardFast from being called
// redundantly from both handleTick and applyRigPoll within the same tick.
var lastFastTick int

// lastTodayIDs holds the last pushed today QSO ID list for change detection.
var lastTodayIDs []int64

// invalidateDashboardFlags clears the cached dupe/new-call/new-DXCC state
// so the next pushDashboardFast recomputes them from the DB. Called after
// a QSO is saved — the QSO we just logged may change dupe status for the
// current call.
func (m *Model) invalidateDashboardFlags() {
	pushDashboardLastCall = ""
	lastActiveDupe = false
	lastActiveNewCall = false
	lastActiveNewDXCC = false
	lastPushedAQSO = cachedAQSO{}
}

func idsEqual(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func cloneIDs(a []int64) []int64 {
	if a == nil {
		return nil
	}
	b := make([]int64, len(a))
	copy(b, a)
	return b
}

// countryWorkedBefore checks whether any QSO exists for the given country
// by a different base_call than the current active call.
func (m *Model) countryWorkedBefore(country string) bool {
	if m.App.DB == nil || country == "" {
		return false
	}
	var count int
	err := m.App.DB.QueryRow(`SELECT COUNT(*) FROM qsos WHERE country = ? AND base_call != ? LIMIT 1`,
		country, qso.DeriveBaseCall(pushDashboardLastCall)).Scan(&count)
	return err == nil && count > 0
}

// pushDashboardRecent queries the DB directly for the 20 most recent QSOs
// (newest-first). Pushes only when the list of QSO IDs changed.
func (m *Model) pushDashboardRecent(ds *dashboard.State) {
	if m.App.DB == nil {
		return
	}
	eventStart := m.eventStartDate()
	limit := 20

	var qsos []qso.QSO
	var err error

	if eventStart != "" {
		qsos, err = m.loadRecentFromDate(eventStart, limit)
	} else {
		// No event start: show today's QSOs (matches map/stats logic).
		today := time.Now().UTC().Format("20060102")
		qsos, err = m.loadRecentFromDate(today, limit)
	}
	if err != nil {
		applog.Debug("dashboard: cannot load recent QSOs", "error", err)
		return
	}

	// Build a fingerprint from QSO IDs to detect changes cheaply.
	fp := make([]int64, len(qsos))
	for i, q := range qsos {
		fp[i] = q.ID
	}
	if idsEqual(fp, lastRecentIDs) {
		applog.Debug("dashboard: recent_qsos unchanged, skipping push", "count", len(fp))
		return
	}
	applog.Debug("dashboard: recent_qsos changed, pushing", "count", len(fp), "prevCount", len(lastRecentIDs))
	lastRecentIDs = cloneIDs(fp)

	views := make([]dashboard.QSOView, 0, len(qsos))
	for _, q := range qsos {
		t, err := time.Parse("20060102 150405", q.QSODate+" "+q.TimeOn)
		if err != nil {
			t = time.Now().UTC()
		}
		view := dashboard.QSOView{
			ID:        fmt.Sprintf("%d", q.ID),
			TimeUTC:   t,
			Call:      q.Call,
			Band:      q.Band,
			Mode:      q.Mode,
			Submode:   q.Submode,
			Frequency: fmt.Sprintf("%.3f MHz", q.Freq),
			RSTSent:   q.RSTSent,
			RSTRcvd:   q.RSTRcvd,
			Grid:      q.GridSquare,
			Country:   q.Country,
			Operator:  q.Operator,
		}
		if q.Freq > 0 {
			view.FrequencyHz = int64(q.Freq * 1_000_000)
		}
		views = append(views, view)
	}
	ds.SetRecent(views)
}

// loadRecentFromDate loads up to <limit> QSOs with qso_date >= date,
// ordered newest-first.
func (m *Model) loadRecentFromDate(date string, limit int) ([]qso.QSO, error) {
	return store.ListQSOsFromDate(m.App.DB, date, limit)
}

func (m *Model) pushDashboardToday(ds *dashboard.State) {
	now := time.Now().UTC()
	today := now.Format("20060102")

	// If event start is configured, use it as the minimum date.
	eventStart := m.App.Config.Integrations.HTTPServer.EventStart
	minDate := today
	yesterday := now.Add(-24 * time.Hour).Format("20060102")
	if eventStart != "" {
		if t, err := time.Parse("2006-01-02", eventStart); err == nil {
			minDate = t.Format("20060102")
		}
	}

	// Load QSOs from minDate through today (capped at 5000).
	qsos, err := store.ListQSOsFromDate(m.App.DB, minDate, 5000)
	if err != nil {
		applog.Debug("dashboard: cannot load QSOs from date", "minDate", minDate, "error", err)
		return
	}
	// If today has very few QSOs (midnight crossing), also include yesterday.
	if len(qsos) < 5 && minDate == today && yesterday > minDate {
		yQsos, yErr := store.ListQSOsFromDate(m.App.DB, yesterday, 5000)
		if yErr == nil {
			qsos = append(yQsos, qsos...)
		}
	}
	// Build fingerprint to skip redundant pushes.
	fp := make([]int64, len(qsos))
	for i, qs := range qsos {
		fp[i] = qs.ID
	}
	if idsEqual(fp, lastTodayIDs) {
		return
	}
	lastTodayIDs = cloneIDs(fp)

	views := make([]dashboard.QSOView, 0, len(qsos))
	for _, qs := range qsos {
		t, err := time.Parse("20060102 150405", qs.QSODate+" "+qs.TimeOn)
		if err != nil {
			t = time.Now().UTC()
		}
		view := dashboard.QSOView{
			ID:        fmt.Sprintf("%d", qs.ID),
			TimeUTC:   t,
			Call:      qs.Call,
			Band:      qs.Band,
			Mode:      qs.Mode,
			Submode:   qs.Submode,
			Frequency: fmt.Sprintf("%.3f MHz", qs.Freq),
			RSTSent:   qs.RSTSent,
			RSTRcvd:   qs.RSTRcvd,
			Grid:      qs.GridSquare,
			Country:   qs.Country,
			Operator:  qs.Operator,
		}
		if qs.Freq > 0 {
			view.FrequencyHz = int64(qs.Freq * 1_000_000)
		}
		views = append(views, view)
	}
	// DB returns newest-first; JS expects newest-first (matching appendTodayQSO).
	ds.SetToday(views)
	applog.Debug("dashboard: pushed today QSOs", "count", len(views))
}

// pushDashboardStats computes and pushes aggregate stats from the event
// start date (or today if not configured) to the dashboard.
func (m *Model) pushDashboardStats(ds *dashboard.State) {
	if m.App.DB == nil {
		return
	}
	startDate := m.eventStartDate()
	if startDate == "" {
		startDate = time.Now().UTC().Format("20060102")
	}
	s, err := store.GetDashboardStats(m.App.DB, startDate)
	if err != nil {
		applog.Debug("dashboard: cannot compute stats", "error", err)
		return
	}
	ds.SetStats(dashboard.Stats{
		QSOsToday:   s.QSOsToday,
		Operators:   s.Operators,
		UniqueCalls: s.UniqueCalls,
		DXCC:        s.DXCC,
		Grids:       s.Grids,
		Bands:       s.Bands,
		Modes:       s.Modes,
		LastQSOAgoS: s.LastQSOAgoS,
		Rate5m:      s.Rate5m,
		Rate15m:     s.Rate15m,
		Rate60m:     s.Rate60m,
	})
}

// pushDashboardAPRS reads recent APRS stations from the cache and pushes them
// to the dashboard for the local map display. Stations outside the configured
// radius are filtered out.
func (m *Model) pushDashboardAPRS(ds *dashboard.State) {
	if m.App.APRSCache == nil {
		return
	}
	// Determine which source(s) to show based on the active APRS service.
	src := ""
	if svc := m.App.Config.Integrations.APRS.Service; svc == "kiss" || svc == "kiss_server" {
		src = "kiss"
	} else {
		src = "aprs_is"
	}
	stations, err := m.App.APRSCache.RecentStations(200, src)
	if err != nil {
		applog.Debug("dashboard: cannot read APRS cache", "error", err)
		return
	}
	applog.Debug("dashboard: APRS cache read", "source", src, "count", fmt.Sprintf("%d", len(stations)))
	// Get station position and APRS config for distance filtering.
	var stLat, stLon, radiusKm float64
	if g := m.effectiveGrid(); g != "" {
		stLat, stLon = gridToLatLon(g)
	}
	if aprsCfg := m.App.Logbook.APRS; aprsCfg != nil && aprsCfg.Enabled && aprsCfg.RadiusKm > 0 {
		radiusKm = float64(aprsCfg.RadiusKm)
	}
	cutoff := time.Now().Add(-60 * time.Minute)
	var view []dashboard.APRSStation
	var trailCalls []string
	for _, s := range stations {
		if s.LastHeard.Before(cutoff) {
			continue
		}
		// Distance filter: only include stations within the configured radius.
		if radiusKm > 0 && stLat != 0 && stLon != 0 {
			d := geo.HaversineKm(stLat, stLon, s.Lat, s.Lon)
			if d > radiusKm {
				continue
			}
		}
		ds := dashboard.APRSStation{
			Callsign:  s.Callsign,
			Lat:       s.Lat,
			Lon:       s.Lon,
			Symbol:    s.Symbol,
			Comment:   s.Comment,
			Course:    s.Course,
			SpeedKmH:  s.SpeedKmH,
			LastHeard: s.LastHeard,
			Source:    s.Source,
		}
		trailCalls = append(trailCalls, s.Callsign)
		view = append(view, ds)
	}
	// Fetch trails for all visible stations — trail query returns
	// data only for callsigns that have moved ≥50 m between updates,
	// regardless of whether the current packet includes speed.
	trailCount := 0
	if len(trailCalls) > 0 {
		trails, err := m.App.APRSCache.StationTrails(trailCalls)
		if err == nil {
			for i := range view {
				if t, ok := trails[view[i].Callsign]; ok && len(t) >= 1 {
					dt := make([]dashboard.TrailPoint, len(t))
					for j, p := range t {
						dt[j] = dashboard.TrailPoint{Lat: p.Lat, Lon: p.Lon, LastHeard: p.LastHeard}
					}
					view[i].Trail = dt
					trailCount++
				}
				// Diagnostic: log trail status for SP9KSK-9 specifically.
				if view[i].Callsign == "SP9KSK-9" {
					t, has := trails[view[i].Callsign]
					applog.Debug("dashboard: SP9KSK-9 trail", "hasTrail", has, "trailLen", len(t), "currentLat", fmt.Sprintf("%.5f", view[i].Lat), "currentLon", fmt.Sprintf("%.5f", view[i].Lon))
				}
			}
		}
		applog.Debug("dashboard: APRS trails attached", "stationsWithTrail", fmt.Sprintf("%d", trailCount), "totalStations", fmt.Sprintf("%d", len(view)))
	}
	ds.SetAPRS(view)
}
