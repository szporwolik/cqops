package tui

import (
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
	m.wlForceCheck = true
	m.needRefresh = true
	return nil
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
	m.toasts.Success("Rig: " + rp.Model + " (" + rp.Antenna + ")")
	applog.Info("Rig cycled", "name", rp.Model)
	return nil
}
