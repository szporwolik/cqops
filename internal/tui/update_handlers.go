package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
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
	// WL lookup timeout: if a lookup was dispatched >20s ago and hasn't
	// completed, force wlLookupDone and clear the dispatch time to prevent
	// repeated warnings when wlLookupDone gets cleared again independently.
	if !m.lookup.wlLookupDone && !m.lookup.wlDispatchTime.IsZero() &&
		time.Since(m.lookup.wlDispatchTime) > 20*time.Second {
		m.lookup.wlLookupDone = true
		m.lookup.wlLookupCall = m.lookup.wlLastCall
		m.lookup.wlDispatchTime = time.Time{}
		applog.Warn("Wavelog: lookup timed out", "call", m.lookup.wlLastCall)
	}
	// WSJT-X auto-recover: only retry when the rig preset has WSJT-X
	// enabled AND the listener was explicitly started (not user-disabled).
	// MaybeRestartWSJTX is a no-op when config hasn't changed, so we
	// never "fight" a user who intentionally turned WSJT-X off.
	if !m.wsjtx.online && m.tickCount%30 == 0 {
		if rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok {
			m.App.MaybeRestartWSJTX(rp.WsjtxEnabled, rp.WsjtxUDPHost, rp.WsjtxUDPPort)
		}
	}
	m.toasts.Expire()
	// Only update the QSO form clock when the form is visible.
	if m.screen == screenQSO {
		m.autoUpdateDateTime()
	}
	m.tickCount++
	// Dispatch async logbook stats fetch if a View() cache miss was recorded.
	if m.rc.logStatsNeedFetch && m.App.DB != nil {
		m.rc.logStatsNeedFetch = false
		cmd = tea.Batch(cmd, m.fetchLogbookStatsCmd(
			m.rc.logStatsFetchCall, m.rc.logStatsFetchBand, m.rc.logStatsFetchMode))
	}
	// Dispatch async PSK spot DB load if a View() cache miss was recorded.
	if m.psk.needDBLoad && m.App.DB != nil {
		m.psk.needDBLoad = false
		cmd = tea.Batch(cmd, m.loadPSKSpotsCmd(
			m.psk.pendingCall, m.psk.pendingCutoff, m.psk.pendingSpotKey))
	}
	// Consolidate periodic commands — only batch non-nil commands to reduce
	// closure allocation and tea.Batch overhead on low-end hardware.
	cmds := []tea.Cmd{tickCmd()}
	if c := m.maybeCheckInet(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeRefreshDataFiles(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.pollRig(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.pollRotor(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeCheckWavelog(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeCheckCallbook(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeFetchSolar(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeDXC(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeHTTP(); c != nil {
		cmds = append(cmds, c)
	}
	// Push current state to the dashboard (cheap — early-exits if unchanged).
	m.pushDashboardState()
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// handleAsyncMessages processes async result messages (internet check, Wavelog status,
// Wavelog upload results, rig poll results). Returns true if the message was consumed
// and an optional command to batch.
func (m *Model) handleAsyncMessages(msg tea.Msg) (bool, tea.Cmd) {
	switch r := msg.(type) {
	case inetResultMsg:
		if !m.inetOnline && bool(r) {
			// Internet just came up — set the flag FIRST so the
			// dispatch functions below don't bail out on !m.inetOnline.
			m.inetOnline = true
			m.offlineToastShown = false
			m.lookup.wlForceCheck = true
			m.lookup.qrzForceCheck = true
			m.toasts.Success("Internet: connected")
			// Push dashboard immediately so the map switches from
			// offline CRS to tiled Web Mercator without waiting
			// for the next throttled tick cycle.
			lastDashboardPushTick = 0
			lastFastTick = 0
			m.pushDashboardState()
			var cmds []tea.Cmd
			if c := m.maybeDXC(); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.maybeHTTP(); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.maybeCheckWavelog(); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.maybeCheckCallbook(); c != nil {
				cmds = append(cmds, c)
			}
			if c := m.maybeFetchSolar(); c != nil {
				cmds = append(cmds, c)
			}
			if len(cmds) > 0 {
				return true, tea.Batch(cmds...)
			}
			return true, nil
		} else if !bool(r) {
			if !m.offlineToastShown {
				m.offlineToastShown = true
				m.toasts.Warn("Internet: not available — working in offline mode")
			}
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
		if r.online {
			m.lookup.wlFailCount = 0
		} else {
			m.lookup.wlFailCount++
		}
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
				if n.Enabled && n.QSOSent {
					applog.Info("Sending Wavelog success notification", "call", r.call)
					if desktopAvailable() {
						if err := beeep.Notify("CQOps — Wavelog", fmt.Sprintf("QSO %s sent to Wavelog", r.call), ""); err != nil {
							applog.Info("Wavelog notification failed", "error", err.Error())
						}
					}
				}
			}
		} else {
			if r.err != nil {
				m.toasts.Error(fmt.Sprintf("Wavelog: %s — %s", r.call, r.err.Error()))
			} else {
				m.toasts.Error(fmt.Sprintf("Wavelog: %s failed", r.call))
			}
			if n.Enabled && n.AllErrors {
				msg := fmt.Sprintf("QSO %s upload failed", r.call)
				if r.err != nil {
					msg = fmt.Sprintf("QSO %s: %s", r.call, r.err.Error())
				}
				applog.Info("Sending Wavelog error notification", "call", r.call)
				if desktopAvailable() {
					if err := beeep.Alert("CQOps — Wavelog Error", msg, ""); err != nil {
						applog.Info("Wavelog error notification failed", "error", err.Error())
					}
				}
			}
		}
		// Immediately refresh the QSO list so the Recent QSOs table picks up
		// the updated Wavelog status. Also flag needRefresh so the logbook
		// editor (if open) reloads on the next tick.
		m.needRefresh = true
		return true, m.refreshQSOS()
	case wsjtxEnrichDoneMsg:
		m.needRefresh = true
		return true, m.refreshQSOS()
	case qrzStatusMsg:
		m.lookup.qrzOnline = r.online
		return true, nil
	case httpStatusMsg:
		if r.client != nil {
			m.http.client = r.client
		}
		if r.online {
			m.http.online = true
			m.http.err = nil
			// Push initial state NOW — bypass throttle so first SSE snapshot has full data.
			lastDashboardPushTick = -10
			lastFastTick = -1
			m.pushDashboardState()
			if m.http.client != nil {
				m.toasts.Success("HTTP server: listening on " + m.http.client.Addr())
			}
			applog.Info("HTTP server: online")
			// Refresh QSO list from DB so dashboard recent matches TUI.
			return true, m.refreshQSOS()
		}
		// Server failed to start — report the error.
		m.http.online = false
		m.http.err = r.err
		if r.err != nil {
			m.toasts.Error("HTTP server: " + r.err.Error())
			applog.Error("HTTP server: failed", "error", r.err)
		}
		m.rc.status = ""
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
	case gpsTickMsg:
		return true, m.handleGPSTick()
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
				if !strings.Contains(err.Error(), "database is closed") {
					applog.Warn("PSK Reporter: DB insert failed", "error", err)
				}
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
			// Push per-band stats to dashboard.
			if m.http.client != nil && m.http.online {
				byBand := make(map[string]int)
				for _, rpt := range r.reports {
					band := qso.DeriveBand(rpt.Frequency)
					if band != "" {
						byBand[band]++
					}
				}
				m.http.client.State().SetPSK(dashboard.PSKInfo{
					Total:  len(r.reports),
					ByBand: byBand,
				})
			}
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
		// Only refresh QSOs when on a screen that displays them — avoids
		// unnecessary DB queries on DXC, PSK, BPL, and other screens.
		// Keep the flag set when the current screen can't show QSOs so
		// the refresh fires as soon as the user navigates to a QSO screen
		// (fixes stale recent QSOs after logbook create/switch).
		if m.screen == screenQSO || m.screen == screenPartner || m.screen == screenLogbookEditor {
			m.needRefresh = false
			cmd = tea.Batch(cmd, m.refreshQSOS())
		}
	}
	if m.lookup.qrzNeed {
		call := m.lookup.qrzCall
		applog.Debug("DXC: handlePendingRequests qrzNeed",
			"call", call,
			"qrzEnabled", m.App.Config.Integrations.Callbook.QRZ.Enabled,
			"qrzUser", m.App.Config.Integrations.Callbook.QRZ.User != "",
		)
		if call == "" {
			m.lookup.qrzNeed = false
			return cmd, false
		}
		m.lookup.qrzNeed = false
		// Always dispatch callbook + DXC spot lookups, even without QRZ.
		if c := m.callbookLookup(call); c != nil {
			return tea.Batch(cmd, c, m.dxcSpotLookupCmd(call)), true
		}
		return tea.Batch(cmd, m.dxcSpotLookupCmd(call)), true
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
	// Auto-trigger REF database rebuild when enabled and empty or when
	// the search column needs backfill (diacritic-insensitive search).
	if m.App != nil && m.App.Config.General.UseRef &&
		m.App.RefDB != nil && !m.ref.building && !m.ref.ready {
		if n, err := m.App.RefDB.Count(); err == nil && n == 0 {
			if c := m.startRefRebuildCmd(); c != nil {
				return tea.Batch(cmd, c), true
			}
		} else if n > 0 {
			needBackfill, _ := m.App.RefDB.NeedsSearchBackfill()
			if needBackfill {
				if c := m.startRefRebuildCmd(); c != nil {
					return tea.Batch(cmd, c), true
				}
			} else {
				m.ref.ready = true
			}
		}
	}
	return cmd, false
}

// handleLookupResultMsg processes async lookup result messages (QRZ, Wavelog,
// logbook stats, PSK spots, REF rebuild, DXC spot/fill/tune, BPL tune/export,
// QSO refresh). Extracted from Update() to keep the main loop manageable.
func (m *Model) handleLookupResultMsg(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch r := msg.(type) {
	case callbookResultMsg:
		m.fillCallbookData(r)
		cmd = tea.Batch(cmd, m.updateFilteredTable())
		m.contestAutoFocusExchRcvd()
		if m.photo.partnerPicNeedLoad {
			m.photo.partnerPicNeedLoad = false
			w := m.photo.partnerPicW
			h := m.photo.partnerPicH
			if w < 25 {
				w = 40
			}
			if h < 4 {
				h = 15
			}
			cmd = tea.Batch(cmd, m.photo.partnerPicViewer.SetSize(w, h),
				m.photo.partnerPicViewer.SetURL(m.photo.partnerPicURL))
		}
		return m, cmd
	case wlResultMsg:
		wlCmd := m.fillWLData(r)
		cmd = tea.Batch(cmd, wlCmd)
		m.contestAutoFocusExchRcvd()
		return m, cmd
	case logbookStatsMsg:
		m.handleLogbookStats(r)
		return m, cmd
	case pskSpotsLoadedMsg:
		if r.err == nil && r.spotKey != "" {
			m.psk.spots = r.spots
			m.psk.spotKey = r.spotKey
		}
		return m, cmd
	case refRebuildMsg:
		m.ref.building = false
		m.ref.refNamesDirty = true
		if r.err != nil {
			applog.Warn("REF: rebuild failed", "error", r.err)
			m.toasts.Error("REF: database build failed")
		} else {
			m.ref.ready = true
			applog.Info("REF: rebuild complete", "total", r.total)
			m.toasts.Success(fmt.Sprintf("REF: database ready — %d references", r.total))
		}
		return m, cmd
	case dxcSpotLookupMsg:
		m.fillDXCFreq(r)
		return m, cmd
	case dxcSpotsStoredMsg:
		m.handleDXCSpotsStored(r)
		return m, cmd
	case dxcTuneResultMsg:
		m.handleTuneResult(r.err, r.freqMHz, r.mode, r.verify)
		return m, cmd
	case bplTuneResultMsg:
		m.handleTuneResult(r.err, r.freqMHz, r.mode, r.verify)
		return m, cmd
	case bplExportMsg:
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("Band Plan: export failed — %v", r.err))
		} else {
			m.toasts.Success(fmt.Sprintf("Band Plan: exported to %s", r.path))
		}
		return m, cmd
	case qsoRefreshedMsg:
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("Refresh failed: %v", r.err))
		} else {
			m.qsos = r.qsos
			m.recentQSOs.SetQSOS(r.qsos)
			m.rc.pathSig = ""
			m.rc.logStatsSig = ""
			if !m.callRecentQSOs.filterSuppressed && m.callRecentQSOs.IsFiltered() {
				filtered, filterErr := store.SearchQSOsByCall(m.App.DB, m.callRecentQSOs.filterCall, 200)
				if filterErr == nil {
					m.callRecentQSOs.SetFilterCall(m.callRecentQSOs.filterCall, filtered)
				}
			}
			m.callRecentQSOs.filterSuppressed = false
		}
		return m, cmd
	default:
		return nil, cmd
	}
}

// handleTuneResult shows the appropriate toast for a rig tune operation
// (shared by dxcTuneResultMsg and bplTuneResultMsg — identical handling).
func (m *Model) handleTuneResult(err error, freqMHz float64, mode, verify string) {
	if err != nil {
		if strings.Contains(err.Error(), "cancelled") {
			m.toasts.Warn(fmt.Sprintf("Rig: tune cancelled — %v", err))
		} else {
			m.toasts.Error(fmt.Sprintf("Rig: tune failed — %v", err))
		}
		return
	}
	msg := fmt.Sprintf("Rig: tuned to %.5f MHz", freqMHz)
	if mode != "" {
		msg += " " + mode
	}
	if verify != "" {
		m.toasts.Warn("Rig: tuning failed")
	} else {
		m.toasts.Success(msg)
	}
}
