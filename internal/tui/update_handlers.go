package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
)

// =============================================================================
// Core update pipeline handlers
// =============================================================================
//
// These are called in sequence from the main Update() in model.go:
//   1. handleTick        — periodic tick messages
//   2. handleAsyncMessages — async result messages (internet, Wavelog, rig)
//   3. handlePendingRequests — deferred actions (QSO refresh, QRZ/WL lookups)

// handleTick processes periodic tick messages: ADIF ingestion, WSJT-X status,
// toast expiry, date/time auto-update, and scheduled health checks.
//
// Concurrency: pendingADIFs and pendingStatus are written by WSJT-X UDP callbacks
// from a background goroutine. We snapshot both fields under a single adifMu lock
// so the read-and-clear is atomic, then release the lock before doing any
// downstream work (logging, tea.Batch, form updates) to keep the critical
// section minimal.
func (m *Model) handleTick(cmd tea.Cmd) tea.Cmd {
	m.adifQ.mu.Lock()
	adifs := m.adifQ.adifs
	m.adifQ.adifs = nil
	sp := m.adifQ.status
	m.adifQ.status = statusPending{}
	m.adifQ.mu.Unlock()

	for _, adif := range adifs {
		if adif == "" {
			continue
		}
		applog.Info("WSJT-X: processing pending ADIF")
		subCmd, retry := m.logQSOFromADIF(adif)
		if subCmd != nil {
			cmd = tea.Batch(cmd, subCmd)
		}
		if retry {
			// DB insert failed — re-queue for next tick.
			m.adifQ.mu.Lock()
			m.adifQ.adifs = append(m.adifQ.adifs, adif)
			m.adifQ.mu.Unlock()
		}
	}

	// Persist any remaining (unprocessed or retry) ADIFs.
	m.adifQ.mu.Lock()
	m.savePendingADIFsLocked()
	m.adifQ.mu.Unlock()

	if sp.hasData {
		m.applyWSJTXStatus(sp.call, sp.grid, sp.freq, sp.mode, sp.submode, sp.report, sp.txMessage, sp.transmitting)
	}
	// WSJT-X watchdog: if no status received in 15 seconds, mark offline.
	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
		m.wsjtx.tx = false
		m.wsjtx.txMsg = ""
		m.rc.status = ""
	}
	// WSJT-X auto-reconnect: if enabled but never online, retry start every 30s.
	// MaybeRestartWSJTX is a no-op when the listener is already running; it only
	// acts when the previous start failed (lastWSJTX wasn't updated on error).
	if !m.wsjtx.online && m.tickCount%30 == 0 {
		if rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok && rp.WsjtxEnabled {
			m.App.MaybeRestartWSJTX(rp.WsjtxEnabled, rp.WsjtxUDPHost, rp.WsjtxUDPPort)
		}
	}
	m.toasts.Expire()
	// Only update the QSO form clock when the form is visible.
	if m.screen == screenQSO {
		m.autoUpdateDateTime()
	}
	m.tickCount++
	return tea.Batch(tickCmd(), m.maybeCheckInet(), m.maybeRefreshDataFiles(), m.pollRig(), m.pollRotor(), m.maybeCheckWavelog(), m.maybeCheckQRZ(), m.maybeFetchSolar(), m.maybeDXC(), cmd)
}

// handleAsyncMessages processes async result messages (internet check, Wavelog status,
// Wavelog upload results, rig poll results). Returns true if the message was consumed
// and an optional command to batch.
func (m *Model) handleAsyncMessages(msg tea.Msg) (bool, tea.Cmd) {
	switch r := msg.(type) {
	case inetResultMsg:
		if !m.inetOnline && bool(r) {
			// Internet just came up — force Wavelog and QRZ checks.
			m.lookup.wlForceCheck = true
			m.lookup.qrzForceCheck = true
			m.toasts.Success("Internet connected")
		} else if m.inetOnline && !bool(r) {
			m.toasts.Warn("Internet not available — working in offline mode")
		}
		m.inetOnline = bool(r)
		return true, nil
	case versionCheckMsg:
		if r.latest != "" {
			current := version.Resolved()
			if versionNewer(r.latest, current) {
				m.toasts.Warn(fmt.Sprintf("CQOps %s available — visit github.com/szporwolik/cqops/releases", r.latest))
			}
		}
		return true, nil
	case wlStatusMsg:
		m.lookup.wlOnline = r.online
		if r.stationName != "" {
			m.lookup.wlStationName = r.stationName
		}
		if r.stationLabel != "" {
			m.lookup.wlStationLabel = r.stationLabel
		}
		m.rc.status = ""
		return true, nil
	case wlUploadResultMsg:
		n := m.App.Config.General.Notifications
		if r.ok {
			if r.isDup {
				m.toasts.Success(fmt.Sprintf("Wavelog: %s already present", r.call))
			} else {
				m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", r.call))
				if n.Enabled && n.Wavelog {
					applog.Info("Sending Wavelog success notification", "call", r.call)
					if err := beeep.Notify("CQOps — Wavelog", fmt.Sprintf("QSO %s sent to Wavelog", r.call), ""); err != nil {
						applog.Info("Wavelog notification failed", "error", err.Error())
					}
				}
			}
		} else {
			if r.err != nil {
				m.toasts.Error(fmt.Sprintf("Wavelog: %s — %s", r.call, r.err.Error()))
			} else {
				m.toasts.Error(fmt.Sprintf("Wavelog: %s failed", r.call))
			}
			if n.Enabled && n.WavelogErrors {
				msg := fmt.Sprintf("QSO %s upload failed", r.call)
				if r.err != nil {
					msg = fmt.Sprintf("QSO %s: %s", r.call, r.err.Error())
				}
				applog.Info("Sending Wavelog error notification", "call", r.call)
				if err := beeep.Notify("CQOps — Wavelog Error", msg, ""); err != nil {
					applog.Info("Wavelog error notification failed", "error", err.Error())
				}
			}
		}
		m.needRefresh = true
		return true, nil
	case wsjtxEnrichDoneMsg:
		m.needRefresh = true
		return true, nil
	case qrzStatusMsg:
		m.lookup.qrzOnline = r.online
		return true, nil
	case rigPollMsg:
		return true, m.applyRigPoll(r)
	case rigPowerMsg:
		m.applyRigPower(r)
		return true, nil
	case rotorPollMsg:
		return true, m.applyRotorPoll(r)
	case rotorNameMsg:
		if r.name != "" {
			m.rotor.name = r.name
		}
		return true, nil
	case rigModesMsg:
		if len(r.modes) > 0 {
			m.rig.modes = r.modes
		}
		return true, nil
	case rigNameMsg:
		if r.name != "" {
			m.rig.name = r.name
		}
		return true, nil
	case pskFetchMsg:
		m.psk.fetching = false
		if r.err != nil {
			applog.Error("PSK Reporter: fetch failed", "error", r.err)
			m.toasts.Error("PSK Reporter: " + r.err.Error())
		} else {
			// Store in SQLite.
			call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
			var spots []store.PSKSpot
			now := time.Now().UTC().Unix()
			for _, rpt := range r.reports {
				spots = append(spots, store.PSKSpot{
					ReceiverCall: rpt.ReceiverCallsign, ReceiverLoc: rpt.ReceiverLocator,
					Frequency: rpt.Frequency, SNR: rpt.SNR,
					Mode: rpt.Mode, FlowStart: rpt.FlowStartSeconds,
					FetchTime: now, StationCall: call,
				})
			}
			if n, err := store.InsertPSKSpots(m.App.DB, spots); err != nil {
				applog.Warn("PSK Reporter: DB insert failed", "error", err)
			} else if n > 0 {
				applog.Info("PSK Reporter: new spots stored", "count", n)
			}
			_ = store.PurgeOldPSKSpots(m.App.DB)
			m.psk.lastFetchByCall[call] = r.fetchTime
			m.psk.lastCall = call
			m.psk.fetched = true
			m.psk.spotKey = ""
			m.psk.viewKey = ""
			m.psk.spots = nil
			m.toasts.Info(fmt.Sprintf("PSK Reporter: %d spots updated", len(r.reports)))
		}
		return true, nil
	case solarFetchMsg:
		m.handleSolarResult(r)
		return true, nil
	case dxcStatusMsg:
		m.handleDXCStatus(r)
		return true, nil
	}
	return false, nil
}

// handlePendingRequests processes deferred actions (QSO refresh, QRZ lookup, WL lookup)
// that were flagged during normal message handling.
func (m *Model) handlePendingRequests(cmd tea.Cmd) (tea.Cmd, bool) {
	if m.needRefresh {
		m.needRefresh = false
		cmd = tea.Batch(cmd, m.refreshQSOS())
		// Fall through — do NOT short-circuit.  The refresh runs
		// asynchronously; dropping the current message (e.g. a
		// navigation key right after a screen transition) would
		// require the user to press the key twice.
	}
	if m.lookup.qrzNeed {
		call := m.lookup.qrzCall
		applog.Debug("DXC: handlePendingRequests qrzNeed",
			"call", call,
			"qrzEnabled", m.App.Config.Integrations.QRZ.Enabled,
			"qrzUser", m.App.Config.Integrations.QRZ.User != "",
		)
		if call == "" {
			m.lookup.qrzNeed = false
			return cmd, false
		}
		// Always fire DXC spot lookup when call changes, even if QRZ is disabled.
		if !m.App.Config.Integrations.QRZ.Enabled || m.App.Config.Integrations.QRZ.User == "" {
			m.lookup.qrzNeed = false
			return tea.Batch(cmd, m.dxcSpotLookupCmd(call)), true
		}
		if c := m.qrzLookup(call); c != nil {
			m.lookup.qrzNeed = false
			return tea.Batch(cmd, c, m.dxcSpotLookupCmd(call)), true
		}
		// qrzLookup rate-limited — leave qrzNeed to retry next tick.
		// Fall through so wlNeed below can still fire on this tick.
	}
	if m.lookup.wlNeed {
		call := m.lookup.wlCall
		if call != "" {
			if c := m.wlLookup(call); c != nil {
				m.lookup.wlNeed = false
				return tea.Batch(cmd, c), true
			}
			// wlLookup returned nil (rate-limited, offline, or disabled);
			// leave wlNeed=true so the next tick retries the lookup.
		} else {
			m.lookup.wlNeed = false
		}
	}
	if m.dxc.need {
		m.dxc.need = false
		call := m.dxc.call
		if call != "" {
			return tea.Batch(cmd, m.dxcSpotLookupCmd(call)), true
		}
	}
	// Auto-trigger REF database rebuild when enabled and empty.
	if m.App != nil && m.App.Config.General.UseRef &&
		m.App.RefDB != nil && !m.ref.building && !m.ref.ready {
		if n, err := m.App.RefDB.Count(); err == nil && n == 0 {
			if c := m.startRefRebuildCmd(); c != nil {
				return tea.Batch(cmd, c), true
			}
		} else if n > 0 {
			m.ref.ready = true
		}
	}
	return cmd, false
}
