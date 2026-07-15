package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

// =============================================================================
// Logbook and rig cycling
// =============================================================================

// cycleLogbook switches to the next logbook in alphabetical order (by callsign).
func (m *Model) cycleLogbook() tea.Cmd {
	ids := config.SortedLogbookIDs(m.App.Config)
	if len(ids) <= 1 {
		m.toasts.Info("Only one logbook configured")
		return nil
	}

	// Find current and move to next.
	idx := 0
	for i, id := range ids {
		if id == m.App.Config.State.ActiveLogbook {
			idx = (i + 1) % len(ids)
			break
		}
	}
	next := ids[idx]

	if err := m.App.SwitchLogbook(next); err != nil {
		m.toasts.Error("Switch to " + config.LogbookDisplayName(m.App.Logbook) + " failed: " + err.Error())
		return nil
	}
	displayName := config.LogbookDisplayName(m.App.Logbook)
	m.toasts.Success("Logbook: " + displayName)
	applog.Info("Logbook cycled", "name", displayName)
	m.rc.status = ""
	m.invalidatePartnerMapCache()
	m.rc.logStatsSig = ""
	m.rc.pathSig = ""
	m.rc.pathLine = ""
	m.lookup.wlPrivateData = nil // WL data is logbook-specific
	m.lookup.wlForceCheck = true

	// Clear contest exchange fields, then re-apply prefill if the new
	// logbook has an active contest with prefilling enabled.
	m.fields[fieldExchSent].SetValue("")
	m.fields[fieldExchRcvd].SetValue("")
	m.prefillContestExchange()
	m.needRefresh = true

	// Recheck dupe and new-call status against the new logbook.
	if strings.TrimSpace(m.fields[fieldCall].Value()) != "" {
		m.checkDupe()
	}
	var cmds []tea.Cmd
	cmds = append(cmds, m.refreshQSOS())
	// Request recent DXC spots for the new logbook so the DXC table
	// isn't left empty after the DB switch clears old spots.
	if m.dxc.online && m.dxc.client != nil {
		cmds = append(cmds, func() tea.Msg {
			m.dxc.client.RequestRecent(50)
			return nil
		})
	}
	// Force-push all dashboard panels so the website reflects the new
	// logbook immediately — not on the next 5 s throttle cycle.
	m.forcePushDashboardAll()
	return tea.Batch(cmds...)
}

// cycleRig cycles to the next rig preset in alphabetical order (by model).
func (m *Model) cycleRig() tea.Cmd {
	ids := config.SortedRigIDs(m.App.Config)
	if len(ids) == 0 {
		m.toasts.Info("No rigs configured")
		return nil
	}
	if len(ids) == 1 {
		rp := m.App.Config.Rigs[ids[0]]
		m.toasts.Info("Only one rig: " + config.RigDisplayName(&rp))
		return nil
	}

	// Find current and move to next.
	current := m.App.Logbook.Station.RigName
	idx := 0
	for i, id := range ids {
		if id == current {
			idx = (i + 1) % len(ids)
			break
		}
	}
	next := ids[idx]
	rp := m.App.Config.Rigs[next]

	m.App.Logbook.Station.RigName = next
	lb := m.App.Config.Logbooks[m.App.LogbookName]
	lb.Station.RigName = next
	m.App.Config.Logbooks[m.App.LogbookName] = lb

	if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
		m.toasts.Error("Save rig failed: " + err.Error())
		return nil
	}
	m.toasts.Success("Rig: " + config.RigDisplayName(&rp))
	applog.Info("Rig cycled", "name", config.RigDisplayName(&rp))
	m.rc.status = ""
	m.invalidatePartnerMapCache()
	m.rc.pathSig = ""
	m.refreshRigClient()   // reconnect/disconnect for the new rig
	m.refreshRotorClient() // rotor may have changed too
	m.App.MaybeRestartWSJTX(rp.WsjtxEnabled, rp.WsjtxUDPHost, rp.WsjtxUDPPort)
	// Push rig/station change to dashboard — light, no DB queries.
	if m.http.online {
		lastFastTick = 0
		m.pushDashboardFast()
	}
	return nil
}

// restartWSJTXForActiveRig reads the active rig's WSJT-X config and
// calls MaybeRestartWSJTX.  Used after rig editor save/close so WSJT-X
// starts/stops immediately instead of waiting for the periodic retry.
func (m *Model) restartWSJTXForActiveRig() {
	if m.App == nil || m.App.Logbook == nil || m.App.Config == nil {
		return
	}
	rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]
	if !ok {
		return
	}
	m.App.MaybeRestartWSJTX(rp.WsjtxEnabled, rp.WsjtxUDPHost, rp.WsjtxUDPPort)
}
