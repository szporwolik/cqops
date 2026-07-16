package tui

import (
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/dxc"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// DX Cluster — telnet connection to dxspider.co.uk:7300
// =============================================================================

// dxcStatusMsg is sent when the DX Cluster connection state changes.
type dxcStatusMsg struct {
	online bool
	err    error
}

// dxcSpotsStoredMsg is sent after a batch of spots has been stored.
type dxcSpotsStoredMsg struct {
	calls     []string
	spottedMe string
	newSpots  []store.DXCSpot // newly inserted spots for in-memory append
}

// sendSpotCmd sends a DX spot to the connected cluster and stores it locally
// so it appears immediately in the DXC table even if the cluster doesn't echo.
func (m *Model) sendSpotCmd(call string, freqKhz float64, comment string) tea.Cmd {
	db := m.App.DB
	return func() tea.Msg {
		if m.dxc.client == nil || !m.dxc.online {
			m.toasts.Warn("DXC: not connected — cannot send spot")
			return nil
		}

		// Toast immediately so the user gets instant feedback that the spot
		// is being sent — don't wait for the cluster round-trip (~1.5 s).
		toastMsg := fmt.Sprintf("Spotted %s @ %.1f kHz", call, freqKhz)
		if comment != "" {
			toastMsg += " — " + comment
		}
		m.toasts.Info(toastMsg)

		rsp, err := m.dxc.client.SendSpot(freqKhz, call, comment)
		if err != nil {
			m.toasts.Warn("DXC: spot failed — " + err.Error())
			return nil
		}
		applog.Info("DXC: spot sent", "cmd", fmt.Sprintf("DX %.1f %s %s", freqKhz, call, comment))

		// If cluster responded with an error-like message, warn as a follow-up.
		if rsp != "" {
			applog.Warn("DXC: cluster response", "response", rsp)
			m.toasts.Warn("DXC: " + rsp)
		}

		// If DB was closed (logbook cycle), silently drop local store.
		if db == nil {
			applog.Debug("DXC: skip local spot store — db nil")
			return dxcSpotsStoredMsg{calls: []string{call}, newSpots: nil}
		}

		// Also store locally so it shows up in the DXC table immediately.
		now := time.Now().UTC().Unix()
		mode := deriveSpotMode(comment, freqKhz/1000)
		spot := store.DXCSpot{
			DXCall:     strings.ToUpper(call),
			Frequency:  freqKhz,
			Band:       qso.DeriveBand(freqKhz / 1000),
			Mode:       mode,
			ModeCat:    spotModeCategory(mode),
			Comment:    comment,
			Spotter:    m.App.Logbook.Station.Callsign,
			ReceivedAt: now,
		}
		if cty := m.App.BigCTY; cty != nil {
			if e := cty.Find(call); e != nil {
				spot.DXCont = e.Continent
				spot.DXCC = e.Name
			}
			if e := cty.Find(spot.Spotter); e != nil {
				spot.SpotCont = e.Continent
			}
		}
		if _, err := store.InsertDXCSpots(db, []store.DXCSpot{spot}); err != nil {
			applog.Warn("DXC: local spot store failed", "error", err)
		}

		return dxcSpotsStoredMsg{calls: []string{call}, newSpots: []store.DXCSpot{spot}}
	}
}

// dxcConnectCmd returns a tea.Cmd that attempts to connect to the DX cluster.
func (m *Model) dxcConnectCmd() tea.Cmd {
	return func() tea.Msg {
		cfg := m.App.Config.Integrations.DXC
		host := cfg.Host
		if host == "" {
			host = "dxspots.com"
		}
		port := cfg.Port
		if port == "" {
			port = "7300"
		}
		login := cfg.Login
		if login == "" {
			login = m.App.Logbook.Station.Callsign
		}

		client := dxc.NewClient(host, port, login)
		m.dxc.client = client

		if err := client.Start(); err != nil {
			return dxcStatusMsg{online: false, err: err}
		}
		return dxcStatusMsg{online: true}
	}
}

// dxcReconnectDelay is the pause between reconnection attempts.
var dxcReconnectDelays = []time.Duration{
	5 * time.Second,
	10 * time.Second,
	30 * time.Second,
	60 * time.Second,
}

// maybeDXC returns a tea.Cmd to start, stop, or maintain the DX Cluster connection.
func (m *Model) maybeDXC() tea.Cmd {
	cfg := m.App.Config.Integrations.DXC

	// Not enabled — ensure disconnected and reset all state.
	if !cfg.Enabled {
		if m.dxc.client != nil {
			m.dxc.client.Stop()
			m.dxc.client = nil
		}
		m.dxc.online = false
		m.dxc.connecting = false
		m.dxc.lastAttempt = time.Time{}
		m.dxc.reconnectIdx = 0
		m.rc.status = ""
		if m.screen == screenDXC {
			m.screen = screenQSO
		}
		return nil
	}

	// Need internet.
	if !m.inetOnline {
		if m.dxc.client != nil {
			m.dxc.client.Stop()
			m.dxc.client = nil
		}
		m.dxc.online = false
		m.dxc.connecting = false
		m.dxc.lastAttempt = time.Time{}
		m.dxc.reconnectIdx = 0
		m.rc.status = ""
		if m.screen == screenDXC {
			m.screen = screenQSO
		}
		return nil
	}

	// Already connected — check for disconnect, then drain spots (throttled to 4s).
	if m.dxc.client != nil && m.dxc.online {
		// Check if the connection dropped.
		select {
		case status, ok := <-m.dxc.client.Status():
			if ok && !status {
				applog.Warn("DXC: connection lost, will reconnect")
				m.dxc.online = false
				m.dxc.connecting = false
				m.dxc.client = nil
				m.rc.status = ""
				return nil
			}
		default:
		}
		// Drain spots at most once every 4 seconds to reduce DB write pressure.
		// The client buffers them; drainDXCSpots empties the entire channel at once.
		if time.Since(m.dxc.lastDrain) >= 4*time.Second {
			m.dxc.lastDrain = time.Now()
			return m.drainDXCSpots()
		}
		return nil
	}

	// Connecting in progress — don't double-connect.
	if m.dxc.connecting {
		return nil
	}

	// Reconnect delay.
	if !m.dxc.online && !m.dxc.lastAttempt.IsZero() {
		delay := dxcReconnectDelays[m.dxc.reconnectIdx]
		if time.Since(m.dxc.lastAttempt) < delay {
			return nil
		}
	}

	m.dxc.connecting = true
	m.dxc.lastAttempt = time.Now()
	applog.Info("DXC: connecting")
	return m.dxcConnectCmd()
}

// drainDXCSpots reads any available spots from the cluster client and returns
// a command to store them. Non-blocking — returns nil when no spots are queued.
func (m *Model) drainDXCSpots() tea.Cmd {
	if m.dxc.client == nil {
		return nil
	}
	ch := m.dxc.client.Spots()
	var spots []dxc.Spot
	for {
		select {
		case s := <-ch:
			spots = append(spots, s)
		default:
			goto done
		}
	}
done:
	if len(spots) == 0 {
		return nil
	}
	return m.storeDXCSpotsCmd(spots)
}

func (m *Model) storeDXCSpotsCmd(spots []dxc.Spot) tea.Cmd {
	db := m.App.DB
	cty := m.App.BigCTY // may be nil if Big CTY failed to load
	return func() tea.Msg {
		// Silently drop spots when DB has been closed (logbook cycle).
		if db == nil {
			return nil
		}
		var dbSpots []store.DXCSpot
		now := time.Now().UTC().Unix()
		for _, s := range spots {
			// Drop spots with no callsign or no frequency.
			if s.DXCall == "" || s.Frequency <= 0 {
				continue
			}
			mode := deriveSpotMode(s.Comment, s.Frequency/1000)
			spot := store.DXCSpot{
				DXCall:     s.DXCall,
				Frequency:  s.Frequency,
				Band:       qso.DeriveBand(s.Frequency / 1000),
				Mode:       mode,
				ModeCat:    spotModeCategory(mode),
				Comment:    s.Comment,
				Spotter:    s.Spotter,
				ReceivedAt: now,
			}
			// Compute continents from DXCC prefix lookup.
			if cty != nil {
				if e := cty.Find(s.DXCall); e != nil {
					spot.DXCont = e.Continent
					spot.DXCC = e.Name
				}
				if e := cty.Find(s.Spotter); e != nil {
					spot.SpotCont = e.Continent
				}
			}
			dbSpots = append(dbSpots, spot)
		}

		// Retry on SQLITE_BUSY — other operations (Wavelog download, QSO save)
		// may hold brief write locks.
		var n int
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			n, err = store.InsertDXCSpots(db, dbSpots)
			if err == nil {
				break
			}
			if strings.Contains(err.Error(), "database is closed") {
				return nil
			}
			if !strings.Contains(err.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			applog.Warn("DXC: spot insert failed", "error", err)
		} else if n > 0 {
			applog.Info("DXC: spots stored", "count", n, "sample_spotter", spots[0].Spotter, "sample_dx", spots[0].DXCall)
		}

		// Purge old spots at most once per minute to reduce DB contention.
		if time.Since(m.dxc.lastPurge) > 60*time.Second {
			m.dxc.lastPurge = time.Now()
			for attempt := 0; attempt < 3; attempt++ {
				if err := store.PurgeOldDXCSpots(db); err == nil {
					break
				} else if !strings.Contains(err.Error(), "database is locked") {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Return the unique callsigns so the QSO form can auto-fill frequency
		// when a spot for the current call arrives.
		seen := map[string]bool{}
		for _, s := range spots {
			seen[s.DXCall] = true
		}
		var calls []string
		for c := range seen {
			calls = append(calls, c)
		}

		// Check if own callsign was spotted.
		var spottedMe string
		myCall := strings.ToUpper(m.App.Logbook.Station.Callsign)
		if myCall != "" {
			for _, s := range spots {
				if strings.EqualFold(s.DXCall, myCall) {
					msg := s.Comment
					if msg == "" {
						msg = fmt.Sprintf("%.1f kHz", s.Frequency)
					}
					spottedMe = s.Spotter + ": " + msg
					break
				}
			}
		}

		return dxcSpotsStoredMsg{calls: calls, spottedMe: spottedMe, newSpots: dbSpots}
	}
}

// handleDXCStatus processes a connection state change.
// Returns a tea.Cmd to fire an immediate health check when the failure
// looks like a network-layer issue (DNS), so we detect internet outages
// in seconds instead of waiting for the 60 s poll cycle.
func (m *Model) handleDXCStatus(msg dxcStatusMsg) tea.Cmd {
	m.dxc.connecting = false
	if msg.online {
		m.dxc.online = true
		m.dxc.reconnectIdx = 0
		m.rc.status = ""
		applog.Info("DXC: connected OK")
		m.toasts.Success("DXC: connected")
		// Push DXC status to dashboard for footer attribution.
		if m.http.client != nil && m.http.online {
			cfg := m.App.Config.Integrations.DXC
			m.http.client.State().SetDXC(dashboard.DXCInfo{
				Connected: true,
				Host:      cfg.Host + ":" + cfg.Port,
			})
		}
		return nil
	}
	m.dxc.online = false
	m.dxc.client = nil
	m.rc.status = ""
	// Redirect to QSO form if the user is viewing the DXC tab.
	if m.screen == screenDXC {
		m.screen = screenQSO
	}
	if m.dxc.reconnectIdx < len(dxcReconnectDelays)-1 {
		m.dxc.reconnectIdx++
	}
	applog.Warn("DXC: connection failed",
		"attempt", m.dxc.reconnectIdx+1,
		"error", msg.err,
	)
	m.toasts.Warn("DXC: connection failed — retrying")
	// When the failure looks like DNS or routing is dead, fire an
	// immediate health check instead of waiting for the poll timer.
	if isNetworkError(msg.err) {
		m.noteNetworkError()
		return checkInetCmd()
	}
	return nil
}

// resetDXC stops any active connection and resets reconnect state.
// Called when the user saves new DXC settings so they take effect immediately.
func (m *Model) resetDXC() {
	if m.dxc.client != nil {
		m.dxc.client.Stop()
		m.dxc.client = nil
	}
	m.dxc.online = false
	m.dxc.connecting = false
	m.dxc.reconnectIdx = 0
	m.dxc.lastAttempt = time.Time{}
	m.rc.status = ""
}

// dxcSpotLookupCmd searches the DXC spot database for the given callsign
// and returns the most recent known frequency, if any.
func (m *Model) dxcSpotLookupCmd(call string) tea.Cmd {
	db := m.App.DB
	return func() tea.Msg {
		spot, err := store.QueryDXCSpotByCall(db, call)
		if err != nil || spot == nil {
			applog.Debug("DXC: dxcSpotLookupCmd — not found", "call", call, "err", err)
			return dxcSpotLookupMsg{call: call}
		}
		applog.Debug("DXC: dxcSpotLookupCmd — found", "call", call, "freq_khz", spot.Frequency)
		return dxcSpotLookupMsg{call: call, freq: spot.Frequency}
	}
}

// fillDXCFreq fills the QSO form frequency field from a DXC spot when
// neither flrig nor WSJT-X is actively providing frequency data.
func (m *Model) fillDXCFreq(msg dxcSpotLookupMsg) {
	applog.Debug("DXC: fillDXCFreq called",
		"call", msg.call,
		"freq", msg.freq,
		"rigConnected", m.rig.connected,
		"wsjtxOnline", m.wsjtx.online,
		"fieldFreq", strings.TrimSpace(m.fields[fieldFreq].Value()),
	)
	if msg.freq <= 0 {
		applog.Debug("DXC: fillDXCFreq bail — freq <= 0")
		return
	}
	// Only use DXC spot freq when NEITHER WSJT-X NOR flrig is connected.
	// The rig provides authoritative frequency; don't let stale spot data
	// overwrite the actual VFO frequency the operator is tuned to.
	if m.wsjtx.online {
		applog.Debug("DXC: fillDXCFreq bail — wsjtx online")
		return
	}
	if m.rig.connected {
		applog.Debug("DXC: fillDXCFreq bail — rig connected")
		return
	}
	freqMHz := msg.freq / 1000 // DXC spots store kHz
	// Only skip if the field already has the exact same frequency.
	if strings.TrimSpace(m.fields[fieldFreq].Value()) == fmt.Sprintf("%.5f", freqMHz) {
		return
	}
	m.fields[fieldFreq].SetValue(fmt.Sprintf("%.5f", freqMHz))
	m.applyFreqDefaults()
	applog.Debug("DXC: filled freq from spot",
		"call", msg.call,
		"freq_khz", msg.freq,
	)
}

// handleDXCSpotsStored checks newly stored spots against the QSO form call
// and invalidates the DXC table cache to keep the view in sync with the DB.
func (m *Model) handleDXCSpotsStored(msg dxcSpotsStoredMsg) {
	// Invalidate table cache so it rebuilds with fresh data.
	m.dxc.tableReady = false
	m.dxc.cachedBands = nil
	m.dxc.cachedConts = nil

	// Append new spots to the raw cache and re-filter in-memory
	// instead of re-querying the full DB.
	if len(msg.newSpots) > 0 {
		m.dxc.cachedRaw = append(m.dxc.cachedRaw, msg.newSpots...)
		// Cap raw cache to prevent unbounded growth (old spots are purged from DB too).
		if len(m.dxc.cachedRaw) > 600 {
			m.dxc.cachedRaw = m.dxc.cachedRaw[len(m.dxc.cachedRaw)-500:]
		}
		// Re-filter from updated raw cache.
		m.dxc.cachedSpots = nil
		m.dxc.cachedSortBand = "" // force re-sort on band-filtered views
		m.dxcFilteredSpots()
	} else {
		m.dxc.cachedSpots = nil
	}

	// Notify when own callsign was spotted.
	if msg.spottedMe != "" {
		m.toasts.Info("DXC: spotted by " + msg.spottedMe)
		// Push to dashboard for the DXC info module.
		if m.http.client != nil && m.http.online {
			parts := strings.SplitN(msg.spottedMe, ": ", 2)
			spotter := parts[0]
			comment := ""
			if len(parts) > 1 {
				comment = parts[1]
			}
			m.http.client.State().SetDXC(dashboard.DXCInfo{
				SpottedBy: spotter,
				Comment:   comment,
			})
		}
	}

	formCall := qso.NormalizeCall(m.fields[fieldCall].Value())
	if formCall == "" {
		return
	}
	// Only override from live spots when WSJT-X is not connected.
	if m.wsjtx.online {
		return
	}
	for _, c := range msg.calls {
		if strings.EqualFold(c, formCall) {
			m.dxc.need = true
			m.dxc.call = formCall
			return
		}
	}
}

// deriveSpotMode determines the operating mode from spot comment and frequency.
// Looks for known mode keywords as whole words in the comment (case-insensitive).
// Falls back to USB (>10MHz) or LSB (<10MHz) when no mode is found.
func deriveSpotMode(comment string, freqMHz float64) string {
	c := strings.ToUpper(comment)
	// Check for mode keywords using word-boundary matching to avoid false
	// positives like "AM" inside "I AM QRV" or callsign fragments.
	for _, kw := range []string{"FT8", "FT4", "FT2", "CW", "RTTY", "FM", "PSK", "JT65", "JT9", "MSK144", "FSK", "DATA"} {
		if wordContains(c, kw) {
			return kw
		}
	}
	// When no mode keyword found in the comment, try to derive FT8 or FT4
	// from the frequency — typical calling frequencies (± 3 kHz tolerance).
	if mode := ft8ft4FromFrequency(freqMHz); mode != "" {
		return mode
	}
	// "AM" is checked separately only at word boundaries to avoid matching
	// the common word "am" in comments.
	if wordContains(c, "AM") {
		return "AM"
	}
	// Detect CW from frequency: lower edge of each HF band.
	if isCWFrequency(freqMHz) {
		return "CW"
	}
	// Default: SSB with sideband based on frequency.
	if freqMHz < 10 {
		return "LSB"
	}
	return "USB"
}

// ft8ft4FromFrequency returns "FT8" or "FT4" if freqMHz is within ±3 kHz of
// a known FT8/FT4 calling frequency on any band. FT4 is checked first because
// some FT4 frequencies are close to FT8. Returns "" if no match.
func ft8ft4FromFrequency(freqMHz float64) string {
	// FT4 calling frequencies (IARU Region 1, common worldwide).
	for _, f := range []float64{
		3.575, 7.047, 10.140, 14.080, 18.104, 21.080, 24.919, 28.080, 50.318,
	} {
		if math.Abs(freqMHz-f) <= 0.003 {
			return "FT4"
		}
	}
	// FT8 calling frequencies.
	for _, f := range []float64{
		1.840, 3.573, 7.074, 10.136, 14.074, 18.100, 21.074, 24.915, 28.074, 50.313, 144.174,
	} {
		if math.Abs(freqMHz-f) <= 0.003 {
			return "FT8"
		}
	}
	return ""
}

// isCWFrequency returns true if the frequency (MHz) falls in the typical
// CW sub-band of any amateur band.
func isCWFrequency(freqMHz float64) bool {
	// CW sub-band edges (lower portion of each band).
	cwRanges := [][2]float64{
		{1.800, 1.840},
		{3.500, 3.580},
		{5.060, 5.080},
		{7.000, 7.050},
		{10.100, 10.130},
		{14.000, 14.070},
		{18.068, 18.100},
		{21.000, 21.080},
		{24.890, 24.930},
		{28.000, 28.150},
		{50.000, 50.100},
	}
	for _, r := range cwRanges {
		if freqMHz >= r[0] && freqMHz <= r[1] {
			return true
		}
	}
	return false
}

// wordContains checks if substr appears as a whole word in s.
// A word boundary is defined as start-of-string, end-of-string, or a space.
func wordContains(s, substr string) bool {
	idx := strings.Index(s, substr)
	if idx < 0 {
		return false
	}
	before := idx == 0 || s[idx-1] == ' '
	after := idx+len(substr) == len(s) || s[idx+len(substr)] == ' '
	return before && after
}
