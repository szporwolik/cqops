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

// dispatchViewFetches batches the DB reads that View() flagged as cache
// misses. View() must stay free of DB I/O, so it only records what it needs
// and the query runs here as a command.
func (m *Model) dispatchViewFetches(cmd tea.Cmd) tea.Cmd {
	if m.App.DB == nil {
		return cmd
	}
	if m.rc.logStatsNeedFetch {
		m.rc.logStatsNeedFetch = false
		cmd = tea.Batch(cmd, m.fetchLogbookStatsCmd(
			m.rc.logStatsFetchCall, m.rc.logStatsFetchBand, m.rc.logStatsFetchMode))
	}
	if m.rc.dxcSpotsNeedFetch {
		m.rc.dxcSpotsNeedFetch = false
		cmd = tea.Batch(cmd, m.fetchDXCPathSpotsCmd(m.rc.dxcSpotsFetchBand))
	}
	if m.rc.dxcDupeNeedFetch {
		m.rc.dxcDupeNeedFetch = false
		cmd = tea.Batch(cmd, m.fetchDXCPathDupesCmd(
			m.rc.dxcDupeFetchDate, m.rc.dxcDupeFetchContest, m.App.LogbookName, m.dxc.dupeGen))
	}
	return cmd
}

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
	// APRS lifecycle work is single-owner: the debounce timer and the APRS
	// workers hand their events back here, and all start/stop/restart
	// transitions run on this main loop.
	if m.App != nil {
		m.App.RunPendingAPRS()
	}
	// WSJT-X watchdog: if no status received in 15 seconds, mark offline.
	if m.wsjtx.online && time.Since(m.wsjtx.lastSeen) > 15*time.Second {
		m.wsjtx.online = false
		m.wsjtx.tx = false
		m.wsjtx.txMsg = ""
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
	// Push live rig state (freq/mode/power from the QSO form) to the
	// Wavelog radio endpoint every 15 seconds.
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled && m.lookup.wlOnline && m.lookup.wlRadioID != 0 &&
		time.Since(m.lookup.lastRadioPush) >= wlRadioPushInterval {
		if c := m.pushWavelogRadioCmd(); c != nil {
			m.lookup.lastRadioPush = time.Now()
			cmd = tea.Batch(cmd, c)
		}
	}
	m.toasts.Expire()
	// Only update the QSO form clock when the form is visible.
	if m.screen == screenQSO {
		m.autoUpdateDateTime()
	}
	m.tickCount++
	// Dispatch async DB fetches flagged by the last View().
	cmd = m.dispatchViewFetches(cmd)
	// Refresh logbook-wide counts once per tick (total QSOs, today's QSOs).
	// Also refreshes at midnight when the date rolls over.
	if m.App.DB != nil {
		today := time.Now().UTC().Format("20060102")
		if m.rc.logbookStatsDate != today {
			total, todayCount, err := store.LogbookCounts(m.App.DB, today)
			if err != nil {
				// Keep the previous counts and retry next tick, so a transient
				// lock does not render the logbook as empty.
				applog.Warn("Logbook counts refresh failed", "error", err)
			} else {
				m.rc.logbookStatsDate = today
				m.rc.logbookTotal, m.rc.logbookToday = total, todayCount
			}
		}
	}
	// Dispatch async PSK spot DB load if a View() cache miss was recorded.
	if m.psk.needDBLoad && m.App.DB != nil {
		m.psk.needDBLoad = false
		cmd = tea.Batch(cmd, m.loadPSKSpotsCmd(
			m.psk.pendingCall, m.psk.pendingCutoff, m.psk.pendingSpotKey))
	}
	// Poll GPS every 60 ticks (~60 s).  GPS position changes slowly; a
	// faster poll wastes CPU on low-end hardware without improving accuracy.
	// Also poll on the first tick so the status bar reflects the actual
	// connection state immediately after startup.
	if m.tickCount == 1 || m.tickCount%60 == 0 {
		if gpsCmd := m.handleGPSTick(); gpsCmd != nil {
			cmd = tea.Batch(cmd, gpsCmd)
		}
	}

	// Consolidate periodic commands — only batch non-nil commands to reduce
	// closure allocation and tea.Batch overhead on low-end hardware.
	cmds := []tea.Cmd{tickCmd()}
	// Rapid internet check triggered by a network service (DXC, APRS)
	// failing with a hard network error. Fires an immediate health check
	// instead of waiting up to 60 s for the next scheduled poll.
	if m.triggerRapidCheck {
		m.triggerRapidCheck = false
		if m.inetOnline {
			cmds = append(cmds, checkInetCmd())
		}
	}
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
	// Periodic DXCC backfill — catch any QSOs that escaped enrichment
	// (legacy imports, ADIF loads, Wavelog downloads before Big CTY was
	// available). Runs every 30 ticks (~30 s), processes 50 rows max.
	if m.App.BigCTY != nil && m.App.DB != nil && m.tickCount%30 == 0 {
		n, _ := backfillMissingDXCCLimit(m.App.DB, m.App.BigCTY, 50)
		if n > 0 {
			applog.Debug("DXCC: periodic backfill updated", "count", n)
		}
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
		if bool(r) {
			// Internet reachable — reset streak and mark online immediately.
			prevOnline := m.inetOnline
			m.inetOnline = true
			m.inetConfirmed = true
			m.App.InetOnline = true
			m.inetFailStreak = 0
			if !prevOnline {
				// Just came back — dispatch all network services.
				m.offlineToastShown = false
				m.lookup.wlForceCheck = true
				m.lookup.qrzForceCheck = true
				m.toasts.Success("Internet: connected")
				// Push every panel immediately so the dashboard gets fresh
				// QSO data alongside the online map switch. Resets all
				// throttles and fingerprints — unlike pushDashboardState(),
				// which may skip today/recent/APRS due to rate limits.
				m.forcePushDashboardAll()
				// Restart APRS — it was stopped when we went offline.
				m.App.MaybeRestartAPRS()
				var cmds []tea.Cmd
				// Make DXC reconnect immediately — reset its internal retry
				// backoff so it doesn't wait for its own delay cycle.
				m.dxc.lastAttempt = time.Time{}
				m.dxc.reconnectIdx = 0
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
			}
			return true, nil
		}
		// Internet unreachable — require 2 consecutive failures before
		// marking offline. A single transient blip is ignored.
		m.inetFailStreak++
		if m.inetFailStreak >= 2 && m.inetOnline {
			m.inetOnline = false
			m.App.InetOnline = false
			if !m.offlineToastShown {
				m.offlineToastShown = true
				m.toasts.Warn("Internet: not available — working in offline mode")
			}
			// Push dashboard immediately so the map switches from
			// tiled Web Mercator to offline equirectangular fallback.
			lastDashboardPushTick = 0
			m.pushDashboardState()
		}
		return true, nil
	case versionCheckMsg:
		if r.latest != "" {
			current := version.Resolved()
			if versionNewer(r.latest, current) {
				m.toasts.Warn(fmt.Sprintf("CQOps %s available — visit github.com/szporwolik/cqops/releases", r.latest))
			}
		}
		return true, nil
	case refDataMsg:
		// Reference data refreshed in the background — install on the main
		// loop only. Never assigned by the worker itself.
		var extra tea.Cmd
		if r.bigCTY != nil && m.App.Config.General.UseCTY {
			m.App.BigCTY = r.bigCTY
			if m.App.DB != nil {
				extra = dxccBackfillCmd(m.App.DB, r.bigCTY)
			}
		}
		if r.scp != nil && m.App.Config.General.UseSCP {
			m.App.SCP = r.scp
		}
		if r.refDB != nil {
			if m.App.Config.General.UseRef && m.App.RefDB == nil {
				m.App.RefDB = r.refDB
				if n, err := r.refDB.Count(); err == nil && n > 0 {
					m.ref.ready = true
				}
			} else {
				// Lost a race — another path opened the database already.
				r.refDB.Close()
			}
		}
		return true, extra
	case dxccBackfillMsg:
		if r.count > 0 {
			applog.Info("DXCC: backfilled missing dxcc", "count", r.count)
		}
		return true, nil
	case callFilterResultMsg:
		m.applyCallFilterResult(r)
		return true, nil
	case wlStatusMsg:
		m.lookup.wlOnline = r.online
		m.lookup.wlStatusErr = r.err
		if r.online {
			m.lookup.wlFailCount = 0
		} else {
			m.lookup.wlFailCount++
		}
		if r.err != "" && m.lookup.wlWarnShown != r.err {
			// Surface the reason once (e.g. the v1-key migration notice).
			m.toasts.Warn(r.err)
			m.lookup.wlWarnShown = r.err
		} else if r.err == "" {
			m.lookup.wlWarnShown = ""
		}
		if r.stationName != "" {
			m.lookup.wlStationName = r.stationName
		}
		if r.stationLabel != "" {
			m.lookup.wlStationLabel = r.stationLabel
		}
		var radioCmd tea.Cmd
		if r.online && m.lookup.wlRadioID == 0 {
			// Connection confirmed — make sure the CQOps radio exists so
			// rig-state pushes can start.
			radioCmd = m.ensureWavelogRadioCmd()
		}
		return true, radioCmd
	case wlUploadResultMsg:
		// The upload worker's database lease transferred with this result and
		// must cover the whole upload → id retry → reconciliation chain. It
		// is handed to the follow-up chain (reconcile or retry); with no
		// follow-up it is released immediately.
		var reconcile tea.Cmd
		if r.ok && r.changed {
			reconcile = m.queueContactReconcile(r.db, r.url, r.key, r.logbook, r.qID, r.release, nil)
			r.release = nil
		}
		// Remote acceptance without local id persistence is unresolved —
		// retry the id attach once; a failed retry leaves the row re-offered
		// on the next upload cycle. The retry carries the accepted snapshot
		// (identity + revision), the originating logbook, and the lease.
		var retry tea.Cmd
		if r.ok && r.unresolved && !r.retried {
			retry = m.retryIDAttachCmd(r.db, r.url, r.key, r.sid, r.logbook, r.snap, r.uploadedRev, r.release)
			r.release = nil
		}
		if r.release != nil {
			r.release()
		}
		// A result for a logbook the user has switched away from: the editor
		// now shows a different database (IDs may collide), so skip all UI
		// updates and notifications.
		if r.logbook != "" && r.logbook != m.App.LogbookName {
			return true, tea.Batch(reconcile, retry)
		}
		if r.qID != 0 && m.ui.logbookEditor != nil && !r.unresolved {
			m.ui.logbookEditor.UpdateWLStatus(r.qID, r.ok, r.remoteID)
		}
		// Enrichment changed fields without changing QSO ids — force-push
		// so the dashboard shows the enriched rows. Runs on the owner loop;
		// the worker never touches dashboard state.
		m.pushDashboardRecentAndToday()
		n := m.App.Config.General.Notifications
		if r.ok {
			if r.unresolved {
				m.toasts.Warn(fmt.Sprintf("Wavelog: %s accepted but remote id not stored — will retry", r.call))
			} else if r.isDup {
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
		return true, tea.Batch(m.refreshQSOS(), reconcile, retry)
	case wsjtxEnrichDoneMsg:
		// Enrichment finished for a logbook the user has switched away from
		// — refreshing would reload the new logbook's rows for nothing.
		if r.logbook != "" && r.logbook != m.App.LogbookName {
			return true, nil
		}
		// Enrichment changed fields without changing QSO ids — force-push
		// so the dashboard shows the enriched rows.
		m.pushDashboardRecentAndToday()
		m.needRefresh = true
		return true, m.refreshQSOS()
	case qrzStatusMsg:
		m.lookup.qrzOnline = r.online
		return true, nil
	case stationSyncDoneMsg:
		// Handled globally — the chooser may be closed before the station
		// fetch completes, and the result must still apply.
		return true, m.handleStationSyncDone(r)
	case logbookSwitchedMsg:
		// The switch bookkeeping follows every logbook change (cycled,
		// chooser, created) even when the chooser screen is gone.
		return true, m.handleLogbookSwitched()
	case httpStatusMsg:
		if r.client != nil {
			m.http.client = r.client
		}
		if r.online {
			m.http.online = true
			m.http.err = nil
			// Push initial state NOW — bypass throttle so first SSE snapshot has full data.
			lastDashboardPushTick = -10
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
			if len(r.reports) > 0 {
				m.toasts.Info(fmt.Sprintf("PSK Reporter: %d spots updated", len(r.reports)))
			}
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
	case wlRadioEnsureMsg, wlRadioPushMsg:
		return m.handleWavelogRadioMsg(msg), nil
	case solarFetchMsg:
		m.handleSolarResult(r)
		return true, nil
	case dxcStatusMsg:
		return true, m.handleDXCStatus(r)
	}
	return false, nil
}

// handlePendingRequests processes deferred actions (QSO refresh, QRZ lookup, WL lookup)
// that were flagged during normal message handling.
func (m *Model) handlePendingRequests(cmd tea.Cmd) (tea.Cmd, bool) {
	// Run before any early return so a fetch flagged by the last View() is
	// serviced on this update instead of waiting for the next tick.
	cmd = m.dispatchViewFetches(cmd)
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
	case dxcPathSpotsMsg:
		m.handleDXCPathSpots(r)
		return m, cmd
	case dxcPathDupesMsg:
		m.handleDXCPathDupes(r)
		return m, cmd
	case workedSummaryMsg:
		m.handleWorkedSummary(r)
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
		// A late result from a previous logbook must never replace the
		// current logbook's table.
		if r.logbook != "" && r.logbook != m.App.LogbookName {
			applog.Debug("QSO refresh: stale result discarded", "from", r.logbook, "current", m.App.LogbookName)
			return m, cmd
		}
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("QSO: refresh failed — %v", r.err))
		} else {
			m.qsos = r.qsos
			m.recentQSOs.SetQSOS(r.qsos)
			applog.Debug("QSOs refreshed", "count", len(r.qsos))
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
