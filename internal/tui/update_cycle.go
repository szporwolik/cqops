package tui

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

// =============================================================================
// Logbook and rig cycling
// =============================================================================

// cycleLogbook switches to the next logbook in alphabetical order.
func (m *Model) cycleLogbook() tea.Cmd {
	names := make([]string, 0, len(m.App.Config.Logbooks))
	for n := range m.App.Config.Logbooks {
		names = append(names, n)
	}
	slices.Sort(names)
	if len(names) <= 1 {
		m.toasts.Info("Only one logbook configured")
		return nil
	}

	// Find current and move to next.
	idx := 0
	for i, n := range names {
		if n == m.App.Config.State.ActiveLogbook {
			idx = (i + 1) % len(names)
			break
		}
	}
	next := names[idx]

	if err := m.App.SwitchLogbook(next); err != nil {
		m.toasts.Error("Switch to " + next + " failed: " + err.Error())
		return nil
	}
	m.toasts.Success("Logbook: " + next)
	applog.Info("Logbook cycled", "name", next)
	m.wlForceCheck = true
	m.needRefresh = true
	return nil
}

// cycleRig cycles to the next rig preset in alphabetical order.
func (m *Model) cycleRig() tea.Cmd {
	names := make([]string, 0, len(m.App.Config.Rigs))
	for n := range m.App.Config.Rigs {
		names = append(names, n)
	}
	slices.Sort(names)
	if len(names) == 0 {
		m.toasts.Info("No rigs configured")
		return nil
	}
	if len(names) == 1 {
		m.toasts.Info("Only one rig: " + names[0])
		return nil
	}

	// Find current and move to next.
	current := m.App.Logbook.Station.RigName
	idx := 0
	for i, n := range names {
		if n == current {
			idx = (i + 1) % len(names)
			break
		}
	}
	next := names[idx]
	rp := m.App.Config.Rigs[next]

	m.App.Logbook.Station.RigName = next
	lb := m.App.Config.Logbooks[m.App.LogbookName]
	lb.Station.RigName = next
	m.App.Config.Logbooks[m.App.LogbookName] = lb

	if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
		m.toasts.Error("Save rig failed: " + err.Error())
		return nil
	}
	m.toasts.Success("Rig: " + next + " (" + rp.Model + ")")
	applog.Info("Rig cycled", "name", next, "model", rp.Model)
	return nil
}
