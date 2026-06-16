package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Core update pipeline handlers
// =============================================================================
//
// These are called in sequence from the main Update() in model.go:
//   1. handleTick        — periodic tick messages
//   2. handleAsyncMessages — async result messages (internet, Wavelog, flrig)
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
	m.adifMu.Lock()
	adifs := m.pendingADIFs
	m.pendingADIFs = nil
	sp := m.pendingStatus
	m.pendingStatus = statusPending{}
	m.adifMu.Unlock()

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
			m.adifMu.Lock()
			m.pendingADIFs = append(m.pendingADIFs, adif)
			m.adifMu.Unlock()
		}
	}

	// Persist any remaining (unprocessed or retry) ADIFs.
	m.adifMu.Lock()
	m.savePendingADIFsLocked()
	m.adifMu.Unlock()

	if sp.hasData {
		m.applyWSJTXStatus(sp.call, sp.grid, sp.freq, sp.mode, sp.submode, sp.report)
	}
	m.toasts.Expire()
	// Only update the QSO form clock when the form is visible.
	if m.screen == screenQSO {
		m.autoUpdateDateTime()
	}
	m.tickCount++
	return tea.Batch(tickCmd(), m.maybeCheckInet(), m.pollFlrig(), m.maybeCheckWavelog(), m.maybeCheckQRZ(), cmd)
}

// handleAsyncMessages processes async result messages (internet check, Wavelog status,
// Wavelog upload results, flrig results). Returns true if the message was consumed.
func (m *Model) handleAsyncMessages(msg tea.Msg) bool {
	switch r := msg.(type) {
	case inetResultMsg:
		m.inetOnline = bool(r)
		return true
	case wlStatusMsg:
		m.wlOnline = r.online
		if r.stationName != "" {
			m.wlStationName = r.stationName
		}
		if r.stationLabel != "" {
			m.wlStationLabel = r.stationLabel
		}
		return true
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
		return true
	case qrzStatusMsg:
		m.qrzOnline = r.online
		return true
	case flrigResultMsg:
		m.applyFlrigResult(r)
		return true
	case pskFetchMsg:
		m.pskFetching = false
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
			m.pskLastFetch = r.fetchTime
			m.pskLastCall = call
			m.pskFetched = true
			m.pskSpotKey = ""
			m.pskViewKey = ""
			m.pskSpots = nil
			m.toasts.Success(fmt.Sprintf("PSK Reporter: %d spots updated", len(r.reports)))
		}
		return true
	}
	return false
}

// handlePendingRequests processes deferred actions (QSO refresh, QRZ lookup, WL lookup)
// that were flagged during normal message handling.
func (m *Model) handlePendingRequests(cmd tea.Cmd) (tea.Cmd, bool) {
	if m.needRefresh {
		m.needRefresh = false
		return tea.Batch(cmd, m.refreshQSOS()), true
	}
	if m.qrzNeed {
		m.qrzNeed = false
		call := m.qrzCall
		if call == "" || !m.App.Config.QRZ.Enabled || m.App.Config.QRZ.User == "" {
			return cmd, false
		}
		return tea.Batch(cmd, m.lookupCallCmd(call)), true
	}
	if m.wlNeed {
		m.wlNeed = false
		call := m.wlCall
		if call != "" {
			return tea.Batch(cmd, m.wlLookup(call)), true
		}
	}
	return cmd, false
}
