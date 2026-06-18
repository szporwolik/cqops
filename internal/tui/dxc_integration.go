package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
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
// It carries the unique callsigns that were in the batch, so the
// QSO form can auto-fill frequency when a spot arrives for the current call.
type dxcSpotsStoredMsg struct {
	calls []string
}

// dxcConnectCmd returns a tea.Cmd that attempts to connect to the DX cluster.
func (m *Model) dxcConnectCmd() tea.Cmd {
	return func() tea.Msg {
		cfg := m.App.Config.DXC
		host := cfg.Host
		if host == "" {
			host = "dxspider.co.uk"
		}
		port := cfg.Port
		if port == "" {
			port = "7300"
		}
		login := cfg.Login
		if login == "" {
			login = m.App.Logbook.Station.Operator
		}
		if login == "" {
			login = m.App.Logbook.Station.Callsign
		}

		client := dxc.NewClient(host, port, login)
		m.dxcClient = client

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
	cfg := m.App.Config.DXC

	// Not enabled — ensure disconnected.
	if !cfg.Enabled {
		if m.dxcClient != nil {
			m.dxcClient.Stop()
			m.dxcClient = nil
		}
		if m.dxcOnline {
			m.dxcOnline = false
			m.cachedStatus = ""
			if m.screen == screenDXC {
				m.screen = screenQSO
			}
		}
		return nil
	}

	// Need internet.
	if !m.inetOnline {
		if m.dxcClient != nil {
			m.dxcClient.Stop()
			m.dxcClient = nil
		}
		if m.dxcOnline {
			m.dxcOnline = false
			m.cachedStatus = ""
			if m.screen == screenDXC {
				m.screen = screenQSO
			}
		}
		return nil
	}

	// Already connected — drain spots.
	if m.dxcClient != nil && m.dxcOnline {
		return m.drainDXCSpots()
	}

	// Connecting in progress — don't double-connect.
	if m.dxcConnecting {
		return nil
	}

	// Reconnect delay.
	if !m.dxcOnline && !m.dxcLastAttempt.IsZero() {
		delay := dxcReconnectDelays[m.dxcReconnectIdx]
		if time.Since(m.dxcLastAttempt) < delay {
			return nil
		}
	}

	m.dxcConnecting = true
	m.dxcLastAttempt = time.Now()
	applog.Info("DXC: connecting")
	return m.dxcConnectCmd()
}

// drainDXCSpots reads any available spots from the cluster client and returns
// a command to store them. Non-blocking — returns nil when no spots are queued.
func (m *Model) drainDXCSpots() tea.Cmd {
	if m.dxcClient == nil {
		return nil
	}
	ch := m.dxcClient.Spots()
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
	return func() tea.Msg {
		var dbSpots []store.DXCSpot
		now := time.Now().UTC().Unix()
		for _, s := range spots {
			dbSpots = append(dbSpots, store.DXCSpot{
				DXCall:     s.DXCall,
				Frequency:  s.Frequency,
				Band:       qso.DeriveBand(s.Frequency / 1000), // convert kHz to MHz
				Comment:    s.Comment,
				Spotter:    s.Spotter,
				ReceivedAt: now,
			})
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
		if time.Since(m.dxcLastPurge) > 60*time.Second {
			m.dxcLastPurge = time.Now()
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
		return dxcSpotsStoredMsg{calls: calls}
	}
}

// handleDXCStatus processes a connection state change.
func (m *Model) handleDXCStatus(msg dxcStatusMsg) {
	m.dxcConnecting = false
	if msg.online {
		m.dxcOnline = true
		m.dxcReconnectIdx = 0
		m.cachedStatus = ""
		applog.Info("DXC: connected OK")
		m.toasts.Success("DXC: connected")
	} else {
		m.dxcOnline = false
		m.dxcClient = nil
		m.cachedStatus = ""
		// Redirect to QSO form if the user is viewing the DXC tab.
		if m.screen == screenDXC {
			m.screen = screenQSO
		}
		if m.dxcReconnectIdx < len(dxcReconnectDelays)-1 {
			m.dxcReconnectIdx++
		}
		applog.Warn("DXC: connection failed",
			"attempt", m.dxcReconnectIdx+1,
			"error", msg.err,
		)
		m.toasts.Warn("DXC: connection failed — retrying")
	}
}

// resetDXC stops any active connection and resets reconnect state.
// Called when the user saves new DXC settings so they take effect immediately.
func (m *Model) resetDXC() {
	if m.dxcClient != nil {
		m.dxcClient.Stop()
		m.dxcClient = nil
	}
	m.dxcOnline = false
	m.dxcConnecting = false
	m.dxcReconnectIdx = 0
	m.dxcLastAttempt = time.Time{}
	m.cachedStatus = ""
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
		"rigConnected", m.rigConnected,
		"wsjtxOnline", m.wsjtxOnline,
		"fieldFreq", strings.TrimSpace(m.fields[fieldFreq].Value()),
	)
	if msg.freq <= 0 {
		applog.Debug("DXC: fillDXCFreq bail — freq <= 0")
		return
	}
	// Only use DXC spot freq when WSJT-X is NOT connected.
	// flrig being connected is fine — the spot frequency overrides the rig's.
	if m.wsjtxOnline {
		applog.Debug("DXC: fillDXCFreq bail — wsjtx online")
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
// and flags a deferred frequency lookup when a matching spot arrives.
func (m *Model) handleDXCSpotsStored(msg dxcSpotsStoredMsg) {
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall == "" {
		return
	}
	// Only override from live spots when WSJT-X is not connected.
	if m.wsjtxOnline {
		return
	}
	for _, c := range msg.calls {
		if strings.EqualFold(c, formCall) {
			m.dxcNeed = true
			m.dxcCall = formCall
			return
		}
	}
}
