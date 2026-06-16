package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
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
			m.App.Config.General.RenderMap = m.configMenu.renderMap
			m.App.Config.General.DrawGrayline = m.configMenu.drawGrayline
			m.App.Config.General.PictureAtQRZPane = m.configMenu.pictureAtQRZ
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
			m.App.Config.General.Notifications.BeepOnError = m.notifMenu.beepOnError
			m.applyBeepOnError()
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
			m.configMenu.width = m.width
			m.configMenu.height = m.height
			m.screen = screenConfig
		case "notifications":
			m.notifMenu = NewNotificationsMenu(m.App.Config)
			m.notifMenu.width = m.width
			m.notifMenu.height = m.height
			m.screen = screenNotifications
		case "callbook":
			m.callbookMenu = NewCallbookMenu(m.App.Config)
			m.callbookMenu.width = m.width
			m.callbookMenu.height = m.height
			m.screen = screenCallbook
		case "logbook":
			m.chooser = NewLogbookChooser(m.App, m.toasts)
			m.chooser.width = m.width
			m.chooser.height = m.height
			m.screen = screenChooser
		case "rig":
			m.rigChooser = NewRigChooser(m.App, m.toasts)
			m.rigChooser.width = m.width
			m.rigChooser.height = m.height
			m.screen = screenRigEdit
		case "integration":
			m.integrationMenu = NewIntegrationMenu(m.App.Config)
			m.integrationMenu.width = m.width
			m.integrationMenu.height = m.height
			m.screen = screenIntegration
		}
	}
	if m.mainMenu.done {
		m.screen = screenQSO
	}
	return m, cmd
}

func (m *Model) handlePartnerUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Resize inline photo viewer when dimensions change (terminal resize etc).
	w := m.partnerPicW
	h := m.partnerPicH
	if w >= 25 && h >= 4 && (w != m.partnerPicLastW || h != m.partnerPicLastH) {
		m.partnerPicLastW = w
		m.partnerPicLastH = h
		cmd = tea.Batch(cmd, m.partnerPicViewer.SetSize(w, h))
	}
	// Dispatch inline photo load when partner/URL changes.
	if m.partnerPicNeedLoad {
		m.partnerPicNeedLoad = false
		if w < 25 {
			w = 25
		}
		if h < 4 {
			h = 4
		}
		cmd = tea.Batch(cmd,
			m.partnerPicViewer.SetSize(w, h),
			m.partnerPicViewer.SetURL(m.lastPartnerPicURL),
		)
	}
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
			m.mainMenu.width = m.width
			m.mainMenu.height = m.height
			m.screen = screenMainMenu
			return m, cmd
		}
	}
	return m, cmd
}

func (m *Model) handlePSKReporterUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Trigger initial fetch when first entering the tab (not yet fetched, not already fetching).
	if !m.pskFetched && !m.pskFetching && m.inetOnline {
		call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
		if call != "" {
			m.pskFetching = true
			return m, tea.Batch(cmd, m.pskFetchCmd())
		}
	}
	// Auto-refresh: if data is older than 5 minutes, trigger a background refresh.
	if m.pskFetched && !m.pskFetching && m.inetOnline &&
		!m.pskLastFetch.IsZero() && time.Since(m.pskLastFetch) >= 5*time.Minute {
		m.pskFetching = true
		return m, tea.Batch(cmd, m.pskFetchCmd())
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.invalidatePartnerMapCache()
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1", "esc":
			m.screen = screenQSO
			return m, cmd
		case "f5":
			// Refresh PSK data via async command — never block UI.
			if !m.pskFetching && m.inetOnline {
				m.pskFetching = true
				m.toasts.Success("PSK Reporter: fetching\u2026")
				return m, m.pskFetchCmd()
			}
			return m, cmd
		case "home", "end":
			// Cycle through band filters — only bands with spots.
			bands := m.pskAvailableBands()
			if len(bands) == 0 {
				return m, cmd
			}
			dir := 1
			if msg.String() == "end" {
				dir = -1
			}
			cur := -1
			for i, b := range bands {
				if b == m.pskBandFilter {
					cur = i
					break
				}
			}
			next := cur + dir
			if next >= len(bands) {
				next = 0
			}
			if next < 0 {
				next = len(bands) - 1
			}
			m.pskBandFilter = bands[next]
			m.pskSelected = 0
			label := m.pskBandFilter
			if label == "" {
				label = "all bands"
			}
			m.toasts.Info(fmt.Sprintf("PSK Reporter: %s", label))
			return m, cmd
		case "pgup", "pgdown":
			// Cycle time filter.
			dir := 1
			if msg.String() == "pgup" {
				dir = -1
			}
			cur := -1
			for i, s := range pskFilterSteps {
				if s == m.pskFilterMins {
					cur = i
					break
				}
			}
			if cur >= 0 {
				next := cur + dir
				if next >= len(pskFilterSteps) {
					next = 0
				}
				if next < 0 {
					next = len(pskFilterSteps) - 1
				}
				m.pskFilterMins = pskFilterSteps[next]
			}
			m.pskSelected = 0
			m.toasts.Info(fmt.Sprintf("PSK Reporter: last %d min", m.pskFilterMins))
			return m, cmd
		case "up", "k":
			if m.pskSelected > 0 {
				m.pskSelected--
			}
			return m, cmd
		case "down", "j":
			m.pskSelected++
			return m, cmd
		case "insert":
			m.pskCycleMode(1)
			return m, cmd
		case "delete":
			m.pskCycleMode(-1)
			return m, cmd
		}
	}
	return m, cmd
}

// pskAvailableBands returns the band filter options: "" (all) plus each band
// that has at least one spot in the cached result set.
func (m *Model) pskAvailableBands() []string {
	if len(m.pskSpots) == 0 {
		return nil
	}
	seen := map[string]bool{"": true} // "all" is always first
	for _, r := range m.pskSpots {
		band := freqToBandName(r.Frequency)
		if band != "" {
			seen[band] = true
		}
	}
	var bands []string
	for b := range seen {
		bands = append(bands, b)
	}
	sort.Strings(bands)
	return bands
}

func freqToBandName(freqHz float64) string {
	freqkHz := freqHz / 1000
	switch {
	case freqkHz >= 1800 && freqkHz < 2000:
		return "160m"
	case freqkHz >= 3500 && freqkHz < 4000:
		return "80m"
	case freqkHz >= 7000 && freqkHz < 7300:
		return "40m"
	case freqkHz >= 14000 && freqkHz < 14350:
		return "20m"
	case freqkHz >= 21000 && freqkHz < 21450:
		return "15m"
	case freqkHz >= 28000 && freqkHz < 29700:
		return "10m"
	default:
		return "other"
	}
}

func (m *Model) pskCycleMode(dir int) {
	if len(m.pskSpots) == 0 {
		return
	}
	seen := map[string]bool{"": true}
	for _, r := range m.pskSpots {
		if r.Mode != "" {
			seen[strings.ToUpper(r.Mode)] = true
		}
	}
	var modes []string
	for mode := range seen {
		modes = append(modes, mode)
	}
	sort.Strings(modes)
	cur := -1
	for i, mode := range modes {
		if mode == m.pskModeFilter {
			cur = i
			break
		}
	}
	next := cur + dir
	if next >= len(modes) {
		next = 0
	}
	if next < 0 {
		next = len(modes) - 1
	}
	m.pskModeFilter = modes[next]
	m.pskSelected = 0
	label := m.pskModeFilter
	if label == "" {
		label = "all modes"
	}
	m.toasts.Info(fmt.Sprintf("PSK Reporter: %s", label))
}

func (m *Model) handleLogbookEditorUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Block all non-editor keys during full-screen operations (download).
	if m.logbookEditor.isDownloadActive() {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			_, editorCmd := m.logbookEditor.Update(msg)
			cmd = tea.Batch(cmd, editorCmd)
			return m, cmd
		}
	}
	// Detect resize — pageSize depends on terminal height, so reload.
	oldW, oldH := m.logbookEditor.width, m.logbookEditor.height
	m.logbookEditor.width = m.width
	m.logbookEditor.height = m.height
	if m.width != oldW || m.height != oldH {
		m.logbookEditor.needsReload = true
	}
	_, editorCmd := m.logbookEditor.Update(msg)
	if em, ok := msg.(editorMsg); ok {
		if em.toastWarn != "" {
			m.toasts.Warn(em.toastWarn)
		}
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
			m.logbookEditor.wlLastFetchedID = 0
			m.logbookEditor.needsReload = true
			if m.App.Logbook.Wavelog != nil {
				m.App.Logbook.Wavelog.LastFetchedID = 0
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					applog.Warn("Failed to reset Wavelog last_fetched_id after purge", "error", err)
				}
			}
			m.needRefresh = true
		}
		if em.wlQSOID != 0 {
			if em.wlOK {
				if em.wlDup {
					m.toasts.Success(fmt.Sprintf("Wavelog: %s already present", em.wlCall))
				} else {
					m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", em.wlCall))
				}
				m.logbookEditor.UpdateWLStatus(em.wlQSOID, "yes")
				m.logbookEditor.needsReload = true
			} else {
				if em.err != nil {
					m.toasts.Error(fmt.Sprintf("Wavelog: %s — %s", em.wlCall, em.err.Error()))
				} else {
					m.toasts.Error(fmt.Sprintf("Wavelog: %s failed", em.wlCall))
				}
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
			// Download finished — editor's Update already set wlDownloadCount/Dupes.
			if !em.dlAborted && em.dlCount > 0 {
				m.needRefresh = true
			}
		} else if em.dlErr != "" {
			m.logbookEditor.wlDownloadErr = em.dlErr
			m.logbookEditor.mode = edModeWLDownloadResult
		}
	}
	if m.logbookEditor.needsReload {
		m.logbookEditor.needsReload = false
		m.logbookEditor.loadPage()
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
