package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Global key bindings (F1-F10, etc.) — independent of current screen
// =============================================================================

// handleGlobalKeys processes top-level function key bindings (F1-F10, etc.)
// that are independent of the current screen. Returns true if the key was handled.
func (m *Model) handleGlobalKeys(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		applog.Debug("tab: F10 quit requested")
		dlg := NewDialog("Quit CQOPS", "Exit the application?",
			Option{Label: "Quit", Value: "quit"},
			Option{Label: "Cancel", Value: "cancel"},
		)
		m.confirm = &dlg
		m.screen = screenQSO
		return nil, true

	case key.Matches(msg, m.keys.QSOForm):
		applog.Debug("tab: F1 QSO Form")
		m.screen = screenQSO
		return nil, true

	case key.Matches(msg, m.keys.Partner):
		call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))

		// Cycle: Partner → Image → Partner (when photo available).
		if m.screen == screenImage {
			m.screen = screenPartner
			return nil, true
		}
		if m.screen == screenPartner && m.partnerData != nil && m.partnerData.ImageURL != "" {
			applog.Debug("F2: opening image view", "url", m.partnerData.ImageURL)
			m.screen = screenImage
			w := m.width
			h := m.height - 5 // header + footer overhead
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			h-- // bottom hint row
			return tea.Batch(
				m.imageViewer.SetSize(w, h),
				m.imageViewer.SetURL(m.partnerData.ImageURL),
			), true
		}

		if call == "" {
			m.toasts.Warn("No callsign entered")
			applog.Debug("F2 Partner: no callsign")
			return nil, true
		}
		applog.Debug("tab: F2 Partner Details")
		band := strings.TrimSpace(m.fields[fieldBand].Value())
		mode := strings.TrimSpace(m.fields[fieldMode].Value())

		callChanged := m.partnerData == nil || !strings.EqualFold(m.partnerData.Callsign, call)
		bandChanged := band != m.wlLastBand
		modeChanged := mode != m.wlLastMode

		if callChanged {
			m.partnerData = nil
		}
		if callChanged || bandChanged || modeChanged {
			m.wlPrivateData = nil
			m.wlLookupDone = false
		}
		m.screen = screenPartner
		m.invalidatePartnerMapCache()

		var cmds []tea.Cmd
		if callChanged && m.App.Config.QRZ.User != "" && m.App.Config.QRZ.Enabled {
			cmds = append(cmds, m.qrzLookup(call))
		}
		wl := m.App.Logbook.Wavelog
		if (callChanged || bandChanged || modeChanged) && wl != nil && wl.Enabled && wl.APIKey != "" {
			cmds = append(cmds, m.wlLookup(call))
		}
		if len(cmds) > 0 {
			return tea.Batch(cmds...), true
		}
		return nil, true

	case key.Matches(msg, m.keys.Config):
		if m.screen == screenMainMenu {
			applog.Debug("tab: F7 close Config")
			m.screen = screenQSO
		} else {
			applog.Debug("tab: F7 Config")
			m.mainMenu = NewMainMenu()
			m.screen = screenMainMenu
		}
		return nil, true

	case key.Matches(msg, m.keys.LogEditor):
		applog.Debug("tab: F6 Log Editor")
		wl := m.App.Logbook.Wavelog
		wlURL, wlKey, wlStationID := "", "", ""
		if wl != nil {
			wlURL, wlKey, wlStationID = wl.URL, wl.APIKey, wl.StationProfileID
		}
		m.logbookEditor = NewLogbookEditor(m.App.DB, wlURL, wlKey, wlStationID, m.App.Logbook.Station.Operator, m.App.Logbook.Station.Grid)
		m.logbookEditor.width = m.width
		m.logbookEditor.height = m.height
		qsos, _ := store.ListAllQSOs(m.App.DB)
		m.logbookEditor.SetQSOS(qsos)
		m.screen = screenLogbookEditor
		return nil, true

	case key.Matches(msg, m.keys.Logs):
		applog.Debug("tab: F8 Log Viewer")
		m.logViewer = NewLogViewer(config.LogbookDisplayName(m.App.Logbook))
		m.logViewer.width = m.width
		m.logViewer.height = m.height
		m.screen = screenLogView
		return nil, true

	default:
		if !m.isSubmodelActive() {
			if key.Matches(msg, m.keys.Delete) {
				m.clearForm()
				return nil, true
			}
			if key.Matches(msg, m.keys.Lookup) {
				call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
				if call == "" {
					return nil, true
				}
				var cmds []tea.Cmd
				if m.App.Config.QRZ.User != "" && m.App.Config.QRZ.Enabled {
					cmds = append(cmds, m.qrzLookup(call))
				}
				wl := m.App.Logbook.Wavelog
			if wl != nil && wl.Enabled && wl.APIKey != "" {
					cmds = append(cmds, m.wlLookup(call))
				}
				if len(cmds) > 0 {
					return tea.Batch(cmds...), true
				}
			}
			if key.Matches(msg, m.keys.CycleLogbook) {
				return m.cycleLogbook(), true
			}
			if key.Matches(msg, m.keys.CycleRig) {
				return m.cycleRig(), true
			}
		}
	}
	return nil, false
}

// =============================================================================
// QSO form key bindings
// =============================================================================

// handleFormKey processes QSO form-specific key bindings when no sub-screen is active.
// Returns a command and true if the key was handled.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	switch {
	case m.retainFocused:
		switch msg.String() {
		case "space", "enter":
			m.retainComment = !m.retainComment
		case "tab", "down":
			m.nextField()
		case "shift+tab", "up":
			m.prevField()
		case "ctrl+r":
			m.retainComment = !m.retainComment
		}
		return nil, true

	case key.Matches(msg, m.keys.PrevField):
		m.prevField()
		return nil, true

	case key.Matches(msg, m.keys.NextField):
		m.nextField()
		return nil, true

	case key.Matches(msg, m.keys.Save):
		return m.saveQSO(), true

	case key.Matches(msg, m.keys.Delete):
		m.clearForm()
		return nil, true

	case key.Matches(msg, m.keys.Retain):
		m.retainComment = !m.retainComment
		return nil, true

	case msg.String() == "ctrl+c":
		m.mainMenu = NewMainMenu()
		m.screen = screenMainMenu
		return nil, true

	case key.Matches(msg, m.keys.FocusCall):
		m.focusField(fieldCall)
		return nil, true

	case key.Matches(msg, m.keys.Partner):
		call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		var cmds []tea.Cmd
		if call != "" && m.App.Config.QRZ.User != "" && m.App.Config.QRZ.Enabled && m.partnerData == nil {
			cmds = append(cmds, m.qrzLookup(call))
		}
		cmds = append(cmds, m.wlLookup(call))
		if len(cmds) > 0 {
			return tea.Batch(cmds...), true
		}
		return nil, true

	case key.Matches(msg, m.keys.Lookup):
		call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		var cmds []tea.Cmd
		if call != "" && m.App.Config.QRZ.User != "" && m.App.Config.QRZ.Enabled {
			cmds = append(cmds, m.qrzLookup(call))
		}
		cmds = append(cmds, m.wlLookup(call))
		if len(cmds) > 0 {
			return tea.Batch(cmds...), true
		}
		return nil, true

	case key.Matches(msg, m.keys.Enter):
		return m.saveQSO(), true

	case key.Matches(msg, m.keys.CycleUp):
		m.cycleFieldUp()
		return nil, true

	case key.Matches(msg, m.keys.CycleDown):
		m.cycleFieldDown()
		return nil, true

	default:
		m.updateFocused(msg)
	}

	// Re-trigger WL lookup when band or mode changes while partner data is already loaded.
	curBand := strings.TrimSpace(m.fields[fieldBand].Value())
	curMode := strings.TrimSpace(m.fields[fieldMode].Value())
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if call != "" && (curBand != m.wlLastBand || curMode != m.wlLastMode) && m.wlPrivateData != nil {
		m.wlNeed = true
		m.wlCall = call
	}
	return nil, false
}
