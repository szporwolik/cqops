package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// handleTick processes periodic tick messages: ADIF ingestion, WSJT-X status,
// toast expiry, date/time auto-update, and scheduled health checks.
func (m *Model) handleTick(cmd tea.Cmd) tea.Cmd {
	m.adifMu.Lock()
	adif := m.pendingADIF
	m.pendingADIF = ""
	m.adifMu.Unlock()
	if adif != "" {
		applog.Info("WSJT-X: processing pending ADIF")
		cmd = tea.Batch(cmd, m.logQSOFromADIF(adif))
	}
	m.adifMu.Lock()
	sp := m.pendingStatus
	m.pendingStatus = statusPending{}
	m.adifMu.Unlock()
	if sp.hasData {
		m.applyWSJTXStatus(sp.call, sp.grid, sp.freq, sp.mode, sp.submode, sp.report)
	}
	m.toasts.Expire()
	m.autoUpdateDateTime()
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
		if r.ok {
			store.UpdateWavelogStatus(m.App.DB, r.qID, "yes")
			m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", r.call))
		} else {
			store.UpdateWavelogStatus(m.App.DB, r.qID, "no")
			m.toasts.Warn(fmt.Sprintf("Wavelog: %s failed", r.call))
		}
		return true
	case qrzStatusMsg:
		m.qrzOnline = r.online
		return true
	case flrigResultMsg:
		m.applyFlrigResult(r)
		return true
	}
	return false
}

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
		m.logViewer = NewLogViewer(m.App.LogbookName)
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
		if call == "" || !m.App.Config.QRZ.Enabled {
			return cmd, false
		}
		if m.App.Config.QRZ.User == "" {
			m.toasts.Warn("QRZ not configured")
			return cmd, false
		}
		return tea.Batch(cmd, m.qrzLookup(call), m.wlLookup(call)), true
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

// --- Screen-specific update handlers ---

func (m *Model) handleChooserUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.chooser.width = m.width
	m.chooser.height = m.height
	_, chooserCmd := m.chooser.Update(msg)
	cmd = tea.Batch(cmd, chooserCmd)
	if m.chooser.done {
		m.screen = screenMainMenu
		m.wlForceCheck = true
		m.needRefresh = true
	}
	// Logbook was switched via Enter in the chooser — force WL check.
	if _, ok := msg.(logbookSwitchedMsg); ok {
		m.wlForceCheck = true
		m.needRefresh = true
	}
	return m, cmd
}

func (m *Model) handleRigEditUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.rigChooser.width = m.width
	m.rigChooser.height = m.height
	_, rigCmd := m.rigChooser.Update(msg)
	cmd = tea.Batch(cmd, rigCmd)
	if m.rigChooser.done {
		m.screen = screenMainMenu
		m.refreshFlrigClient()
	}
	return m, cmd
}

func (m *Model) handleConfigUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.configMenu.width = m.width
	m.configMenu.height = m.height
	_, configCmd := m.configMenu.Update(msg)
	cmd = tea.Batch(cmd, configCmd)
	if m.configMenu.done {
		m.screen = screenQSO
		if m.configMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.configMenu.saved {
			m.App.Config.General.DistanceUnit = m.configMenu.distanceUnit
			m.saveConfig("Settings saved")
			m.screen = screenMainMenu
		}
	}
	return m, cmd
}

func (m *Model) handleCallbookUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.callbookMenu.width = m.width
	m.callbookMenu.height = m.height
	m.callbookMenu.inetOnline = m.inetOnline
	_, callbookCmd := m.callbookMenu.Update(msg)
	if m.callbookMenu.done {
		m.screen = screenQSO
		if m.callbookMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.callbookMenu.saved {
			m.App.Config.QRZ.User = m.callbookMenu.user.Value()
			m.App.Config.QRZ.Pass = m.callbookMenu.pass.Value()
			m.App.Config.QRZ.Enabled = m.callbookMenu.enabled
			m.saveConfig("Settings saved")
			m.screen = screenMainMenu
		}
	}
	return m, tea.Batch(cmd, callbookCmd)
}

func (m *Model) handleIntegrationUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.integrationMenu.width = m.width
	m.integrationMenu.height = m.height
	_, integrationCmd := m.integrationMenu.Update(msg)
	if m.integrationMenu.done {
		m.screen = screenQSO
		if m.integrationMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.integrationMenu.saved {
			wsjtxE, wsjtxH, wsjtxP := m.integrationMenu.Values()
			m.App.Config.WSJTX.Enabled = wsjtxE
			m.App.Config.WSJTX.UDPHost = wsjtxH
			m.App.Config.WSJTX.UDPPort = wsjtxP
			m.saveConfig("Settings saved")
			applog.Info("Integration config saved, restarting services")
			m.App.MaybeRestartWSJTX()
			m.screen = screenMainMenu
		}
	}
	return m, tea.Batch(cmd, integrationCmd)
}

func (m *Model) handleMainMenuUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.mainMenu.width = m.width
	m.mainMenu.height = m.height
	_, mainCmd := m.mainMenu.Update(msg)
	cmd = tea.Batch(cmd, mainCmd)
	if m.mainMenu.action != "" {
		action := m.mainMenu.action
		m.mainMenu.action = ""
		switch action {
		case "general":
			m.configMenu = NewGeneralMenu(m.App.Config)
			m.screen = screenConfig
		case "callbook":
			m.callbookMenu = NewCallbookMenu(m.App.Config)
			m.screen = screenCallbook
		case "logbook":
			m.chooser = NewLogbookChooser(m.App, m.toasts)
			m.screen = screenChooser
		case "rig":
			m.rigChooser = NewRigChooser(m.App, m.toasts)
			m.screen = screenRigEdit
		case "integration":
			m.integrationMenu = NewIntegrationMenu(m.App.Config)
			m.screen = screenIntegration
		}
	}
	if m.mainMenu.done {
		m.screen = screenQSO
	}
	return m, cmd
}

func (m *Model) handlePartnerUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1", "esc":
			m.screen = screenQSO
			return m, cmd
		case "f7":
			m.mainMenu = NewMainMenu()
			m.screen = screenMainMenu
			return m, cmd
		}
	}
	return m, cmd
}

func (m *Model) handleLogbookEditorUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.logbookEditor.width = m.width
	m.logbookEditor.height = m.height
	_, editorCmd := m.logbookEditor.Update(msg)
	if em, ok := msg.(editorMsg); ok {
		if em.err != nil && em.wlQSOID == 0 {
			m.toasts.Error(em.err.Error())
		}
		if em.deleted != 0 {
			m.toasts.Success(fmt.Sprintf("QSO %s from %s deleted", em.delCall, em.delDate))
		}
		if em.saved != 0 {
			m.toasts.Success(fmt.Sprintf("QSO %s from %s saved", em.saveCall, em.saveDate))
		}
		if em.purged {
			m.toasts.Success("Logbook purged")
		}
		if em.wlQSOID != 0 {
			if em.wlOK {
				m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", em.wlCall))
				m.logbookEditor.UpdateWLStatus(em.wlQSOID, "yes")
				m.logbookEditor.needsReload = true
			} else {
				m.toasts.Warn(fmt.Sprintf("Wavelog: %s failed", em.wlCall))
				m.logbookEditor.UpdateWLStatus(em.wlQSOID, "no")
			}
		}
		if m.logbookEditor.wlSkipped > 0 {
			m.toasts.Warn(fmt.Sprintf("Wavelog: %s", m.logbookEditor.wlSkipDetail))
			m.logbookEditor.wlSkipped = 0
			m.logbookEditor.wlSkipDetail = ""
		}
	}
	if m.logbookEditor.needsReload {
		m.logbookEditor.needsReload = false
		qsos, _ := store.ListAllQSOs(m.App.DB)
		m.logbookEditor.SetQSOS(qsos)
		m.needRefresh = true
	}
	if m.logbookEditor.done {
		m.screen = screenQSO
		m.needRefresh = true
	}
	return m, tea.Batch(cmd, editorCmd)
}

func (m *Model) handleLogViewUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.logViewer.width = m.width
	m.logViewer.height = m.height
	_, logCmd := m.logViewer.Update(msg)
	cmd = tea.Batch(cmd, logCmd)
	if m.logViewer.done {
		m.screen = screenQSO
	}
	return m, cmd
}

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
