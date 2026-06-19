package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
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
	m.ui.chooser.width = m.width
	m.ui.chooser.height = m.height
	_, chooserCmd := m.ui.chooser.Update(msg)
	cmd = tea.Batch(cmd, chooserCmd)
	if m.ui.chooser.done {
		m.screen = screenMainMenu
		m.lookup.wlForceCheck = true
		m.needRefresh = true
	}
	// Logbook was switched via Enter in the chooser — force WL check.
	if _, ok := msg.(logbookSwitchedMsg); ok {
		m.lookup.wlForceCheck = true
		m.needRefresh = true
	}
	return m, cmd
}

func (m *Model) handleRigEditUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.rigChooser.width = m.width
	m.ui.rigChooser.height = m.height
	_, rigCmd := m.ui.rigChooser.Update(msg)
	cmd = tea.Batch(cmd, rigCmd)
	if m.ui.rigChooser.done {
		m.screen = screenMainMenu
		m.refreshFlrigClient()
	}
	return m, cmd
}

func (m *Model) handleConfigUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.configMenu.width = m.width
	m.ui.configMenu.height = m.height
	_, configCmd := m.ui.configMenu.Update(msg)
	cmd = tea.Batch(cmd, configCmd)
	if m.ui.configMenu.done {
		m.screen = screenQSO
		if m.ui.configMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.ui.configMenu.saved {
			m.App.Config.General.DistanceUnit = m.ui.configMenu.distanceUnit
			m.App.Config.General.Timezone = m.ui.configMenu.timezone
			m.App.Config.General.RenderMap = m.ui.configMenu.renderMap
			m.App.Config.General.DrawGrayline = m.ui.configMenu.drawGrayline
			m.App.Config.General.PictureAtQRZPane = m.ui.configMenu.pictureAtQRZ
			m.App.Config.General.SolarAtQSOPane = m.ui.configMenu.solarAtQSO
			m.App.Config.General.UseCTY = m.ui.configMenu.useCTY
			m.App.Config.General.UseSCP = m.ui.configMenu.useSCP
			m.saveConfig("Settings saved")
			m.reloadDataFiles()
			m.screen = screenMainMenu
		}
	}
	return m, cmd
}

func (m *Model) handleNotificationsUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.notifMenu.width = m.width
	m.ui.notifMenu.height = m.height
	_, notifCmd := m.ui.notifMenu.Update(msg)
	cmd = tea.Batch(cmd, notifCmd)
	if m.ui.notifMenu.done {
		m.screen = screenQSO
		if m.ui.notifMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.ui.notifMenu.saved {
			m.App.Config.General.Notifications.Enabled = m.ui.notifMenu.enabled
			m.App.Config.General.Notifications.QSO = m.ui.notifMenu.qso
			m.App.Config.General.Notifications.Wavelog = m.ui.notifMenu.wavelog
			m.App.Config.General.Notifications.WavelogErrors = m.ui.notifMenu.wavelogErrors
			m.App.Config.General.Notifications.BeepOnError = m.ui.notifMenu.beepOnError
			m.applyBeepOnError()
			m.saveConfig("Settings saved")
			m.screen = screenMainMenu
		}
	}
	return m, cmd
}

func (m *Model) handleCallbookUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.callbookMenu.width = m.width
	m.ui.callbookMenu.height = m.height
	m.ui.callbookMenu.inetOnline = m.inetOnline
	_, callbookCmd := m.ui.callbookMenu.Update(msg)
	if m.ui.callbookMenu.done {
		m.screen = screenQSO
		if m.ui.callbookMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.ui.callbookMenu.saved {
			m.App.Config.QRZ.User = m.ui.callbookMenu.user.Value()
			m.App.Config.QRZ.Pass = m.ui.callbookMenu.pass.Value()
			m.App.Config.QRZ.Enabled = m.ui.callbookMenu.enabled
			m.saveConfig("Settings saved")
			m.screen = screenMainMenu
		}
	}
	return m, tea.Batch(cmd, callbookCmd)
}

func (m *Model) handleIntegrationUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.integrationMenu.width = m.width
	m.ui.integrationMenu.height = m.height
	_, integrationCmd := m.ui.integrationMenu.Update(msg)

	// Show validation errors from the menu.
	if m.ui.integrationMenu.SaveError != "" {
		m.toasts.Error(m.ui.integrationMenu.SaveError)
		m.ui.integrationMenu.SaveError = ""
	}

	if m.ui.integrationMenu.done {
		m.screen = screenQSO
		if m.ui.integrationMenu.goBack {
			m.screen = screenMainMenu
		}
		if m.ui.integrationMenu.saved {
			dxcE, dxcHost, dxcPort, dxcLogin, wsjtxE, wsjtxH, wsjtxP := m.ui.integrationMenu.Values()
			m.App.Config.DXC.Enabled = dxcE
			m.App.Config.DXC.Host = dxcHost
			m.App.Config.DXC.Port = dxcPort
			m.App.Config.DXC.Login = dxcLogin
			m.App.Config.WSJTX.Enabled = wsjtxE
			m.App.Config.WSJTX.UDPHost = wsjtxH
			m.App.Config.WSJTX.UDPPort = wsjtxP
			m.saveConfig("Settings saved")
			applog.Info("Integration config saved, restarting services")
			m.App.MaybeRestartWSJTX()
			m.resetDXC()
			m.screen = screenMainMenu
		}
	}
	return m, tea.Batch(cmd, integrationCmd)
}

// reloadDataFiles loads DXCC prefix data and SCP callsign database from
// cached files when the user enables UseCTY or UseSCP in settings. This
// avoids requiring an app restart for those features to become active.
func (m *Model) reloadDataFiles() {
	cacheDir, err := config.CacheDir()
	if err != nil {
		applog.Debug("reloadDataFiles: cannot determine cache dir", "error", err)
		return
	}

	if m.App.Config.General.UseCTY && m.App.DXCC == nil {
		ctyPath := filepath.Join(cacheDir, "cty.dat")
		if prefixes, loadErr := dxcc.LoadLocal(ctyPath); loadErr == nil {
			m.App.DXCC = prefixes
			applog.Info("DXCC: prefix data loaded on demand")
		} else {
			applog.Info("DXCC: no cached data yet — will fetch when online")
		}
	}

	if m.App.Config.General.UseSCP && m.App.SCP == nil {
		scpPath := filepath.Join(cacheDir, "MASTER.SCP")
		if db, loadErr := scp.LoadLocal(scpPath); loadErr == nil {
			m.App.SCP = db
			applog.Info("SCP: callsign database loaded on demand")
		} else {
			applog.Info("SCP: no cached data yet — will fetch when online")
		}
	}
}

func (m *Model) handleMainMenuUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.mainMenu.width = m.width
	m.ui.mainMenu.height = m.height
	_, mainCmd := m.ui.mainMenu.Update(msg)
	cmd = tea.Batch(cmd, mainCmd)
	if m.ui.mainMenu.action != "" {
		action := m.ui.mainMenu.action
		m.ui.mainMenu.action = ""
		switch action {
		case "general":
			m.ui.configMenu = NewGeneralMenu(m.App.Config)
			m.ui.configMenu.width = m.width
			m.ui.configMenu.height = m.height
			m.screen = screenConfig
		case "notifications":
			m.ui.notifMenu = NewNotificationsMenu(m.App.Config)
			m.ui.notifMenu.width = m.width
			m.ui.notifMenu.height = m.height
			m.screen = screenNotifications
		case "callbook":
			m.ui.callbookMenu = NewCallbookMenu(m.App.Config)
			m.ui.callbookMenu.width = m.width
			m.ui.callbookMenu.height = m.height
			m.screen = screenCallbook
		case "logbook":
			m.ui.chooser = NewLogbookChooser(m.App, m.toasts)
			m.ui.chooser.width = m.width
			m.ui.chooser.height = m.height
			m.screen = screenChooser
		case "rig":
			m.ui.rigChooser = NewRigChooser(m.App, m.toasts)
			m.ui.rigChooser.width = m.width
			m.ui.rigChooser.height = m.height
			m.screen = screenRigEdit
		case "integration":
			m.ui.integrationMenu = NewIntegrationMenu(m.App.Config)
			m.ui.integrationMenu.width = m.width
			m.ui.integrationMenu.height = m.height
			m.screen = screenIntegration
		}
	}
	if m.ui.mainMenu.done {
		m.screen = screenQSO
	}
	return m, cmd
}

func (m *Model) handlePartnerUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Resize inline photo viewer when dimensions change (terminal resize etc).
	w := m.photo.partnerPicW
	h := m.photo.partnerPicH
	if w >= 25 && h >= 4 && (w != m.photo.partnerPicLastW || h != m.photo.partnerPicLastH) {
		m.photo.partnerPicLastW = w
		m.photo.partnerPicLastH = h
		cmd = tea.Batch(cmd, m.photo.partnerPicViewer.SetSize(w, h))
	}
	// Dispatch inline photo load when partner/URL changes.
	if m.photo.partnerPicNeedLoad {
		m.photo.partnerPicNeedLoad = false
		if w < 25 {
			w = 25
		}
		if h < 4 {
			h = 4
		}
		cmd = tea.Batch(cmd,
			m.photo.partnerPicViewer.SetSize(w, h),
			m.photo.partnerPicViewer.SetURL(m.photo.partnerPicURL),
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
			m.ui.mainMenu = NewMainMenu()
			m.ui.mainMenu.width = m.width
			m.ui.mainMenu.height = m.height
			m.screen = screenMainMenu
			return m, cmd
		}
	}
	return m, cmd
}

func (m *Model) handlePSKReporterUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Trigger initial fetch when first entering the tab (not yet fetched, not already fetching).
	if !m.psk.fetched && !m.psk.fetching && m.inetOnline {
		call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))
		if call != "" {
			m.psk.fetching = true
			return m, tea.Batch(cmd, m.pskFetchCmd())
		}
	}
	// Auto-refresh: if data is older than 5 minutes, trigger a background refresh.
	if m.psk.fetched && !m.psk.fetching && m.inetOnline &&
		!m.psk.lastFetch.IsZero() && time.Since(m.psk.lastFetch) >= 5*time.Minute {
		m.psk.fetching = true
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
			if !m.psk.fetching && m.inetOnline {
				m.psk.fetching = true
				m.toasts.Info("PSK Reporter: fetching\u2026")
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
				if b == m.psk.bandFilter {
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
			m.psk.bandFilter = bands[next]
			m.psk.selected = 0
			label := m.psk.bandFilter
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
				if s == m.psk.filterMins {
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
				m.psk.filterMins = pskFilterSteps[next]
			}
			m.psk.selected = 0
			m.toasts.Info(fmt.Sprintf("PSK Reporter: last %d min", m.psk.filterMins))
			return m, cmd
		case "up", "k":
			if m.psk.selected > 0 {
				m.psk.selected--
			}
			return m, cmd
		case "down", "j":
			m.psk.selected++
			return m, cmd
		case "insert":
			m.pskCycleMode(1)
			return m, cmd
		case "delete":
			m.pskCycleMode(-1)
			return m, cmd
		case "backspace":
			// Clear all filters.
			m.psk.filterMins = pskFilterSteps[0]
			m.psk.bandFilter = ""
			m.psk.modeFilter = ""
			m.psk.selected = 0
			m.psk.spotKey = ""
			m.psk.viewKey = ""
			m.psk.view = ""
			return m, cmd
		}
	}
	return m, cmd
}

// pskAvailableBands returns the band filter options: "" (all) plus each band
// that has at least one spot in the cached result set.
func (m *Model) pskAvailableBands() []string {
	if len(m.psk.spots) == 0 {
		return nil
	}
	seen := map[string]bool{"": true} // "all" is always first
	for _, r := range m.psk.spots {
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
	if len(m.psk.spots) == 0 {
		return
	}
	seen := map[string]bool{"": true}
	for _, r := range m.psk.spots {
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
		if mode == m.psk.modeFilter {
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
	m.psk.modeFilter = modes[next]
	m.psk.selected = 0
	label := m.psk.modeFilter
	if label == "" {
		label = "all modes"
	}
	m.toasts.Info(fmt.Sprintf("PSK Reporter: %s", label))
}

func (m *Model) handleLogbookEditorUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	// Block all non-editor keys during full-screen operations (download).
	if m.ui.logbookEditor.isDownloadActive() {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			_, editorCmd := m.ui.logbookEditor.Update(msg)
			cmd = tea.Batch(cmd, editorCmd)
			return m, cmd
		}
	}
	// Detect resize — pageSize depends on terminal height, so reload.
	oldW, oldH := m.ui.logbookEditor.width, m.ui.logbookEditor.height
	m.ui.logbookEditor.width = m.width
	m.ui.logbookEditor.height = m.height
	if m.width != oldW || m.height != oldH {
		m.ui.logbookEditor.needsReload = true
	}
	_, editorCmd := m.ui.logbookEditor.Update(msg)
	if em, ok := msg.(editorMsg); ok {
		if em.toastWarn != "" {
			m.toasts.Warn(em.toastWarn)
		}
		if em.toastOK != "" {
			m.toasts.Success(em.toastOK)
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
			m.ui.logbookEditor.wlLastFetchedID = 0
			m.ui.logbookEditor.needsReload = true
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
				m.ui.logbookEditor.UpdateWLStatus(em.wlQSOID, "yes")
				m.ui.logbookEditor.needsReload = true
			} else {
				if em.err != nil {
					m.toasts.Error(fmt.Sprintf("Wavelog: %s — %s", em.wlCall, em.err.Error()))
				} else {
					m.toasts.Error(fmt.Sprintf("Wavelog: %s failed", em.wlCall))
				}
				m.ui.logbookEditor.UpdateWLStatus(em.wlQSOID, "no")
			}
		}
		if m.ui.logbookEditor.wlSkipped > 0 {
			m.toasts.Warn(fmt.Sprintf("Wavelog: %s", m.ui.logbookEditor.wlSkipDetail))
			m.ui.logbookEditor.wlSkipped = 0
			m.ui.logbookEditor.wlSkipDetail = ""
		}
		if em.dlLastID != 0 {
			m.ui.logbookEditor.wlLastFetchedID = em.dlLastID
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
			m.ui.logbookEditor.wlDownloadErr = em.dlErr
			m.ui.logbookEditor.mode = edModeWLDownloadResult
		}
	}
	if m.ui.logbookEditor.needsReload {
		m.ui.logbookEditor.needsReload = false
		m.ui.logbookEditor.loadPage()
		m.needRefresh = true
	}
	if m.ui.logbookEditor.done {
		m.screen = screenQSO
		m.needRefresh = true
	}
	return m, tea.Batch(cmd, editorCmd)
}

func (m *Model) handleLogViewUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.logViewer.width = m.width
	m.ui.logViewer.height = m.height
	_, logCmd := m.ui.logViewer.Update(msg)
	cmd = tea.Batch(cmd, logCmd)
	if m.ui.logViewer.done {
		m.screen = screenQSO
	}
	return m, cmd
}
