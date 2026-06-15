package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// Screen-specific update handlers
// =============================================================================
//
// Each handler routes tea.Msg to the appropriate sub-component and manages
// screen transitions, config saves, and cleanup on exit.

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
			m.App.Config.General.Timezone = m.configMenu.timezone
			m.saveConfig("Settings saved")
			m.screen = screenMainMenu
		}
	}
	return m, cmd
}

func (m *Model) handleNotificationsUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.notifMenu.width = m.width
	m.notifMenu.height = m.height
	_, notifCmd := m.notifMenu.Update(msg)
	cmd = tea.Batch(cmd, notifCmd)
	if m.notifMenu.done {
		m.screen = screenQSO
		if m.notifMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.notifMenu.saved {
			m.App.Config.General.Notifications.Enabled = m.notifMenu.enabled
			m.App.Config.General.Notifications.QSO = m.notifMenu.qso
			m.App.Config.General.Notifications.Wavelog = m.notifMenu.wavelog
			m.App.Config.General.Notifications.WavelogErrors = m.notifMenu.wavelogErrors
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
		case "notifications":
			m.notifMenu = NewNotificationsMenu(m.App.Config)
			m.screen = screenNotifications
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
			// Reset Wavelog download index so next download fetches everything.
			m.logbookEditor.wlLastFetchedID = 0
			if m.App.Logbook.Wavelog != nil {
				m.App.Logbook.Wavelog.LastFetchedID = 0
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					applog.Warn("Failed to reset Wavelog last_fetched_id after purge", "error", err)
				}
			}
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
		if em.dlLastID != 0 {
			m.logbookEditor.wlLastFetchedID = em.dlLastID
			if m.App.Logbook.Wavelog != nil {
				m.App.Logbook.Wavelog.LastFetchedID = em.dlLastID
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					applog.Warn("Failed to persist Wavelog last_fetched_id", "error", err)
				}
			}
		}
		if em.dlDone {
			// Download finished — handled by editor's Update; just refresh.
			if !em.dlAborted && em.dlCount > 0 {
				m.needRefresh = true
			}
		} else if em.dlErr != "" {
			m.logbookEditor.wlDownloadErr = em.dlErr
			m.logbookEditor.mode = edModeWLDownloadResult
		} else if em.dlCount > 0 || em.dlLastID != 0 {
			m.logbookEditor.wlDownloadCount = em.dlCount
			m.logbookEditor.wlDownloadDupes = em.dlDupes
			m.logbookEditor.wlDownloadErr = ""
			m.logbookEditor.mode = edModeWLDownloadResult
			m.needRefresh = true
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
