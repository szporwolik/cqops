package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ftl/hamradio"
	"github.com/ftl/hamradio/bandplan"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/ref"
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

func (m *Model) handleContestUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.contestChooser.width = m.width
	m.ui.contestChooser.height = m.height
	_, contestCmd := m.ui.contestChooser.Update(msg)
	cmd = tea.Batch(cmd, contestCmd)
	if m.ui.contestChooser.needsSave {
		m.ui.contestChooser.needsSave = false
		m.saveConfig("Contests updated")
	}
	if m.ui.contestChooser.done {
		m.screen = screenMainMenu
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
			m.App.Config.General.UseRef = m.ui.configMenu.useRef
			m.saveConfig("Settings saved")
			m.reloadDataFiles()
			// Handle REF database enable/disable.
			if m.App.Config.General.UseRef {
				if m.App.RefDB == nil {
					cacheDir, _ := config.CacheDir()
					refPath := filepath.Join(cacheDir, "ref.db")
					if rdb, err := ref.Open(refPath); err == nil {
						m.App.RefDB = rdb
					}
				}
				// Start async rebuild if database is empty (won't block UI).
				if c := m.startRefRebuildCmd(); c != nil {
					cmd = tea.Batch(cmd, c)
				}
			} else {
				// UseRef disabled — close database and reset state.
				if m.App.RefDB != nil {
					m.App.RefDB.Close()
					m.App.RefDB = nil
				}
				m.ref.ready = false
				m.ref.building = false
				m.ref.searched = false
				m.ref.rows = nil
			}
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
		if _, statErr := os.Stat(ctyPath); os.IsNotExist(statErr) {
			applog.Info("DXCC: downloading on first enable")
			if dlErr := dxcc.Download(dxcc.DefaultURL, ctyPath); dlErr != nil {
				applog.Warn("DXCC: download failed", "error", dlErr.Error())
			}
		}
		if prefixes, loadErr := dxcc.LoadLocal(ctyPath); loadErr == nil {
			m.App.DXCC = prefixes
			applog.Info("DXCC: prefix data loaded on demand")
		} else {
			applog.Info("DXCC: no cached data yet — will fetch when online")
		}
	}

	if m.App.Config.General.UseSCP && m.App.SCP == nil {
		scpPath := filepath.Join(cacheDir, "MASTER.SCP")
		if _, statErr := os.Stat(scpPath); os.IsNotExist(statErr) {
			applog.Info("SCP: downloading on first enable")
			if dlErr := scp.Download(scp.DefaultURL, scpPath); dlErr != nil {
				applog.Warn("SCP: download failed", "error", dlErr.Error())
			}
		}
		if db, loadErr := scp.LoadLocal(scpPath); loadErr == nil {
			m.App.SCP = db
			applog.Info("SCP: callsign database loaded on demand")
		} else {
			applog.Info("SCP: no cached data yet — will fetch when online")
		}
	}

	if m.App.Config.General.UseRef && m.App.RefDB == nil {
		refPath := filepath.Join(cacheDir, "ref.db")
		if rdb, openErr := ref.Open(refPath); openErr == nil {
			m.App.RefDB = rdb
			applog.Info("REF: database opened on demand")
			// Check if already populated.
			if n, err := rdb.Count(); err == nil && n > 0 {
				m.ref.ready = true
			}
		} else {
			applog.Info("REF: cannot open database — will rebuild when online")
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
		case "contest":
			m.ui.contestChooser = NewContestChooser(m.App, m.toasts)
			m.ui.contestChooser.width = m.width
			m.ui.contestChooser.height = m.height
			m.screen = screenContest
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

	// Handle Ctrl+C contest cycling on the log editor screen (list mode only).
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(keyMsg, m.keys.CycleContest) && !m.ui.logbookEditor.IsEditing() {
			m.cycleActiveContest()
			// Refresh the editor's contest filter.
			if m.App.Config.State.ActiveContest != "" {
				ct := m.App.Config.Contests[m.App.Config.State.ActiveContest]
				m.ui.logbookEditor.SetContestID(m.App.Config.State.ActiveContest, config.ContestDisplayName(&ct), ct.ContestID)
			} else {
				m.ui.logbookEditor.SetContestID("", "", "")
			}
			return m, nil
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
		if em.dlDone && !em.dlAborted && em.dlErr == "" {
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

// =============================================================================
// BPL — Band Plan display (F7)
// =============================================================================

// bplState holds the scroll state for the band plan table.
// bplTab constants for the BPL filter tabs.
const (
	bplTabHAM   = 0 // amateur HF bands
	bplTabVHF   = 1 // VHF/UHF detailed
	bplTabCB    = 2 // CB channels
	bplTabPMR   = 3 // PMR446
	bplTabBRC   = 4 // broadcast presets
	bplTabCount = 5
)

var bplTabNames = []string{"Ham Radio - HF", "Ham Radio - VHF", "Citizen Band - CB", "Personal Mobile Radio - PMR", "Broadcast"}

type bplState struct {
	scroll  int
	cursor  int
	tab     int    // active filter tab
	bandSel int    // selected HF band index for detail view (HAM tab)
	search  string // search/filter substring (empty = no filter)

	// Cache.
	cachedView string
	cachedSig  string // "tab|region|w|h|scroll|cursor|bandSel|search"
}

// bandOrder is the display order for HF bands, low to high frequency.
var bandOrder = []bandplan.BandName{
	bandplan.Band160m,
	bandplan.Band80m,
	bandplan.Band60m,
	bandplan.Band40m,
	bandplan.Band30m,
	bandplan.Band20m,
	bandplan.Band17m,
	bandplan.Band15m,
	bandplan.Band12m,
	bandplan.Band10m,
}

// emcomFreqs holds EMCOM (Emergency Communication) center-of-activity
// frequencies in MHz, per IARU region. Empty string = no EMCOM for that band.
var emcomFreqs = map[int]map[bandplan.BandName]string{
	1: { // Region 1 — Europe/Africa
		bandplan.Band80m: "3.760",
		bandplan.Band40m: "7.110",
		bandplan.Band20m: "14.300",
		bandplan.Band17m: "18.160",
		bandplan.Band15m: "21.360",
	},
	2: { // Region 2 — Americas
		bandplan.Band80m: "3.750 / 3.985",
		bandplan.Band40m: "7.060 / 7.240 / 7.275",
		bandplan.Band20m: "14.300",
		bandplan.Band17m: "18.160",
		bandplan.Band15m: "21.360",
	},
	3: { // Region 3 — Asia-Pacific
		bandplan.Band80m: "3.600",
		bandplan.Band40m: "7.110",
		bandplan.Band20m: "14.300",
		bandplan.Band17m: "18.160",
		bandplan.Band15m: "21.360",
	},
}

// qrpFreq holds a QRP center of activity entry.
type qrpFreq struct {
	Mode string
	Freq string
}

// qrpFreqs holds QRP center-of-activity frequencies per IARU region.
// Empty/nil slice means no QRP entry for that band in that region.
var qrpFreqs = map[int]map[bandplan.BandName][]qrpFreq{
	1: { // Region 1 — Europe/Africa
		bandplan.Band160m: {{"CW QRP", "1.836"}},
		bandplan.Band80m:  {{"CW QRP", "3.560"}, {"SSB QRP", "3.690"}},
		bandplan.Band40m:  {{"CW QRP", "7.030"}, {"SSB QRP", "7.090"}},
		bandplan.Band30m:  {{"CW QRP", "10.116"}},
		bandplan.Band20m:  {{"CW QRP", "14.060"}, {"SSB QRP", "14.285"}},
		bandplan.Band17m:  {{"CW QRP", "18.086"}, {"SSB QRP", "18.130"}},
		bandplan.Band15m:  {{"CW QRP", "21.060"}, {"SSB QRP", "21.285"}},
		bandplan.Band12m:  {{"CW QRP", "24.906"}, {"SSB QRP", "24.950"}},
		bandplan.Band10m:  {{"CW QRP", "28.060"}, {"SSB QRP", "28.360"}},
	},
	2: { // Region 2 — Americas
		bandplan.Band160m: {{"CW QRP", "1.812"}, {"SSB QRP", "1.910"}},
		bandplan.Band80m:  {{"CW QRP", "3.560"}, {"SSB QRP", "3.690"}},
		bandplan.Band40m:  {{"CW QRP", "7.030"}, {"SSB QRP", "7.090 + 7.285"}},
		bandplan.Band30m:  {{"CW QRP", "10.116"}},
		bandplan.Band20m:  {{"CW QRP", "14.060"}, {"SSB QRP", "14.285"}},
		bandplan.Band17m:  {{"CW QRP", "18.086"}, {"SSB QRP", "18.130"}},
		bandplan.Band15m:  {{"CW QRP", "21.060"}, {"SSB QRP", "21.285"}},
		bandplan.Band12m:  {{"CW QRP", "24.906"}, {"SSB QRP", "24.950"}},
		bandplan.Band10m:  {{"CW QRP", "28.060"}, {"SSB QRP", "28.360"}},
	},
	3: { // Region 3 — Asia-Pacific
		bandplan.Band160m: {{"CW QRP", "1.836"}},
		bandplan.Band80m:  {{"CW QRP", "3.560"}, {"SSB QRP", "3.690"}},
		bandplan.Band40m:  {{"CW QRP", "7.030"}, {"SSB QRP", "7.090"}},
		bandplan.Band30m:  {{"CW QRP", "10.116"}},
		bandplan.Band20m:  {{"CW QRP", "14.060"}, {"SSB QRP", "14.285"}},
		bandplan.Band17m:  {{"CW QRP", "18.086"}, {"SSB QRP", "18.130"}},
		bandplan.Band15m:  {{"CW QRP", "21.060"}},
		bandplan.Band12m:  {{"CW QRP", "24.906"}, {"SSB QRP", "24.950"}},
		bandplan.Band10m:  {{"SSB QRP", "28.360"}},
	},
}

// sstvFreqs holds SSTV/image centre-of-activity frequencies per IARU region.
var sstvFreqs = map[int]map[bandplan.BandName]string{
	1: { // Region 1 — Europe/Africa
		bandplan.Band80m: "3.735",
		bandplan.Band40m: "7.165",
		bandplan.Band20m: "14.230",
		bandplan.Band15m: "21.340",
		bandplan.Band10m: "28.680",
	},
	2: { // Region 2 — Americas
		bandplan.Band80m: "3.735 / 3.845",
		bandplan.Band40m: "7.165",
		bandplan.Band20m: "14.230",
		bandplan.Band15m: "21.340",
		bandplan.Band10m: "28.680",
	},
	3: { // Region 3 — Asia-Pacific
		bandplan.Band20m: "14.230",
	},
}

// ibpFreqs holds IBP (International Beacon Project) frequencies — same worldwide.
var ibpFreqs = map[bandplan.BandName]string{
	bandplan.Band20m: "14.100",
	bandplan.Band17m: "18.110",
	bandplan.Band15m: "21.150",
	bandplan.Band12m: "24.930",
	bandplan.Band10m: "28.200",
}

// qrsFreqs holds QRS (slow CW) centres of activity — same worldwide.
var qrsFreqs = map[bandplan.BandName]string{
	bandplan.Band80m: "3.555",
	bandplan.Band20m: "14.055",
	bandplan.Band15m: "21.055",
	bandplan.Band10m: "28.055",
}

// vhfCall holds a simple VHF/UHF band overview or calling frequency entry.
// Used for 10m, 6m, and 4m overviews only — 2m/70cm use vhfSeg based data.
type vhfCall struct {
	Band    string // band name, e.g. "4m", "10m"
	FromMHz string // lower band edge (empty for calling entries)
	ToMHz   string // upper band edge (empty for calling entries)
	Freq    string // calling frequency
	Mode    string // e.g. "CALL", "DX", or empty for band header
	Note    string // description
}

// vhfCalling holds simple VHF/UHF band overview and calling frequency data.
// 2m and 70cm are handled by the detailed vhfSeg-based bandplans below.
var vhfCalling = map[int][]vhfCall{
	1: { // Region 1 — Europe/Africa
		{Band: "4m", FromMHz: "70.000", ToMHz: "70.500"},
		{Band: "", Freq: "70.200", Mode: "CALL", Note: "CW/SSB calling"},
		{Band: "", Freq: "70.450", Mode: "CALL", Note: "FM calling"},
		{Band: "6m", FromMHz: "50.000", ToMHz: "52.000"},
		{Band: "", Freq: "50.110", Mode: "DX", Note: "Intercontinental DX calling"},
		{Band: "", Freq: "50.150", Mode: "CALL", Note: "SSB centre/calling"},
	},
	2: { // Region 2 — Americas
		{Band: "6m", FromMHz: "50.000", ToMHz: "54.000"},
		{Band: "", Freq: "50.110", Mode: "DX", Note: "Intercontinental DX calling"},
		{Band: "", Freq: "50.125", Mode: "CALL", Note: "SSB calling"},
	},
	3: { // Region 3 — Asia-Pacific
		{Band: "6m", FromMHz: "50.000", ToMHz: "54.000"},
		{Band: "", Freq: "50.110", Mode: "DX", Note: "Intercontinental DX calling"},
		{Band: "", Freq: "50.150", Mode: "CALL", Note: "SSB centre/calling"},
	},
}

// vhfSeg is a VHF/UHF sub-band segment entry.
// Band headers use Kind="RNG"; sub-rows use Kind tags like SSB, FM, RPT, APR, etc.
type vhfSeg struct {
	Band    string // "2m", "70cm" for header; "" for sub-rows
	Kind    string // "RNG", "SSB", "FM", "RPT", "DIG", "APR", "SAT", "BCN", "IMG", "LRA"
	FromMHz string // segment start or specific frequency
	ToMHz   string // segment end (empty for single-frequency entries)
	Freq    string // specific frequency (calling, APRS etc.)
	Note    string // extra info
}

// vhf2mSeeds holds detailed 2m bandplan seeds per IARU region.
var vhf2mSeeds = map[int][]vhfSeg{
	1: { // Region 1 — 144.000–146.000 MHz
		{Band: "2m", Kind: "RNG", FromMHz: "144.000", ToMHz: "146.000", Note: "FM 12.5 kHz; rpt shift −600 kHz"},
		{Kind: "SAT", FromMHz: "144.000", ToMHz: "144.025", Note: "satellite downlink"},
		{Kind: "SSB", FromMHz: "144.025", ToMHz: "144.100", Freq: "144.050", Note: "CW/weak signal; CW calling"},
		{Kind: "SSB", FromMHz: "144.100", ToMHz: "144.150", Note: "MGM/CW, EME/weak signal"},
		{Kind: "SSB", FromMHz: "144.150", ToMHz: "144.400", Freq: "144.300", Note: "SSB/CW/MGM weak signal; SSB CoA"},
		{Kind: "BCN", FromMHz: "144.400", ToMHz: "144.490", Note: "beacons"},
		{Kind: "IMG", Freq: "144.500", Note: "SSTV/image CoA"},
		{Kind: "DIG", Freq: "144.600", Note: "data/MGM CoA"},
		{Kind: "DIG", FromMHz: "144.794", ToMHz: "144.9625", Note: "digital communications"},
		{Kind: "APR", Freq: "144.800", Note: "APRS Europe / R1 common"},
		{Kind: "RPT", FromMHz: "144.975", ToMHz: "145.194", Note: "repeater inputs"},
		{Kind: "FM", FromMHz: "145.206", ToMHz: "145.5625", Note: "FM/DV simplex"},
		{Kind: "FM", Freq: "145.375", Note: "DV calling"},
		{Kind: "FM", Freq: "145.500", Note: "FM calling"},
		{Kind: "RPT", FromMHz: "145.575", ToMHz: "145.7935", Note: "repeater outputs"},
		{Kind: "SAT", FromMHz: "145.800", ToMHz: "146.000", Note: "satellite exclusive"},
	},
	2: { // Region 2 — 144.000–148.000 MHz
		{Band: "2m", Kind: "RNG", FromMHz: "144.000", ToMHz: "148.000", Note: "FM 15/20 kHz typ; rpt ±600 kHz; local overrides vary"},
		{Kind: "SAT", FromMHz: "144.000", ToMHz: "144.025", Note: "satellite"},
		{Kind: "SSB", FromMHz: "144.000", ToMHz: "144.150", Note: "CW/MGM/EME/weak signal"},
		{Kind: "SSB", FromMHz: "144.180", ToMHz: "144.275", Freq: "144.200", Note: "weak signal; SSB/CW exclusive calling"},
		{Kind: "SSB", FromMHz: "144.300", ToMHz: "144.360", Freq: "144.300", Note: "SSB/CW calling"},
		{Kind: "APR", FromMHz: "144.360", ToMHz: "144.400", Freq: "144.390", Note: "digital/APRS CoA"},
		{Kind: "BCN", FromMHz: "144.400", ToMHz: "144.500", Note: "beacons/ACDS"},
		{Kind: "RPT", FromMHz: "144.600", ToMHz: "144.900", Note: "repeater inputs, output +600 kHz"},
		{Kind: "RPT", FromMHz: "145.200", ToMHz: "145.500", Note: "repeater outputs, input −600 kHz"},
		{Kind: "SAT", FromMHz: "145.800", ToMHz: "146.000", Note: "satellite exclusive"},
		{Kind: "RPT", FromMHz: "146.000", ToMHz: "146.390", Note: "repeater inputs, output +600 kHz"},
		{Kind: "FM", FromMHz: "146.390", ToMHz: "146.600", Freq: "146.520", Note: "FM/DV simplex; FM calling"},
		{Kind: "RPT", FromMHz: "146.600", ToMHz: "146.990", Note: "repeater outputs"},
		{Kind: "RPT", FromMHz: "146.990", ToMHz: "147.400", Note: "repeater inputs"},
		{Kind: "RPT", FromMHz: "147.590", ToMHz: "148.000", Note: "repeater outputs"},
	},
	3: { // Region 3 — 144.000–148.000 MHz
		{Band: "2m", Kind: "RNG", FromMHz: "144.000", ToMHz: "148.000", Note: "less channelized; national rules apply"},
		{Kind: "DIG", FromMHz: "144.000", ToMHz: "144.025", Note: "narrowband/digimodes; satellite caution"},
		{Kind: "SSB", FromMHz: "144.025", ToMHz: "144.035", Note: "EME/weak signal"},
		{Kind: "ALL", FromMHz: "144.035", ToMHz: "145.800", Freq: "144.100", Note: "all modes; suggested DX calling 144.100"},
		{Kind: "APR", Freq: "144.390", Note: "APRS used by several R3 societies"},
		{Kind: "APR", Freq: "144.640", Note: "APRS used by several R3 societies"},
		{Kind: "APR", Freq: "144.800", Note: "APRS suggested spot frequency"},
		{Kind: "SAT", Freq: "145.825", Note: "suggested amateur-satellite APRS"},
		{Kind: "SAT", FromMHz: "145.800", ToMHz: "146.000", Note: "satellites"},
		{Kind: "ALL", FromMHz: "146.000", ToMHz: "148.000", Note: "all modes; national rules apply"},
	},
}

// vhf70cmSeeds holds detailed 70cm bandplan seeds per IARU region.
var vhf70cmSeeds = map[int][]vhfSeg{
	1: { // Region 1 — 430.000–440.000 MHz
		{Band: "70cm", Kind: "RNG", FromMHz: "430.000", ToMHz: "440.000", Note: "FM 12.5/25 kHz; rpt 1.6/2.0/7.6 MHz shifts"},
		{Kind: "RPT", FromMHz: "430.025", ToMHz: "430.375", Note: "repeater outputs, 1.6 MHz shift"},
		{Kind: "DIG", FromMHz: "430.400", ToMHz: "430.575", Note: "digital communications"},
		{Kind: "RPT", FromMHz: "431.050", ToMHz: "431.825", Note: "repeater inputs, 7.6 MHz shift"},
		{Kind: "RPT", FromMHz: "431.625", ToMHz: "431.975", Note: "repeater inputs, 1.6 MHz shift"},
		{Kind: "SSB", FromMHz: "432.000", ToMHz: "432.100", Freq: "432.050", Note: "CW/MGM; CW CoA"},
		{Kind: "SSB", FromMHz: "432.100", ToMHz: "432.400", Freq: "432.200", Note: "CW/SSB/MGM; SSB CoA"},
		{Kind: "BCN", FromMHz: "432.400", ToMHz: "432.490", Note: "beacons"},
		{Kind: "APR", Freq: "432.500", Note: "new APRS frequency"},
		{Kind: "RPT", FromMHz: "432.600", ToMHz: "432.975", Note: "repeater inputs, 2 MHz shift"},
		{Kind: "RPT", FromMHz: "433.000", ToMHz: "433.375", Note: "FM/DV repeater inputs, 1.6 MHz shift"},
		{Kind: "IMG", Freq: "433.400", Note: "SSTV FM/AFSK"},
		{Kind: "FM", Freq: "433.450", Note: "DV calling"},
		{Kind: "FM", Freq: "433.500", Note: "FM calling"},
		{Kind: "DIG", FromMHz: "433.625", ToMHz: "433.775", Note: "digital communication channels"},
		{Kind: "LRA", Freq: "433.775", Note: "LoRa APRS node→gateway (R1 proposed)"},
		{Kind: "LRA", Freq: "433.900", Note: "LoRa APRS gateway→node (R1 proposed)"},
		{Kind: "DIG", Freq: "434.000", Note: "digital experiments centre"},
		{Kind: "RPT", FromMHz: "434.600", ToMHz: "434.9875", Note: "repeater outputs"},
		{Kind: "SAT", FromMHz: "435.000", ToMHz: "438.000", Note: "satellite / DATV / data"},
		{Kind: "RPT", FromMHz: "438.650", ToMHz: "439.425", Note: "repeater outputs, 7.6 MHz shift"},
		{Kind: "DIG", FromMHz: "439.800", ToMHz: "439.975", Note: "digital links"},
		{Kind: "LRA", Freq: "439.9125", Note: "UK LoRa APRS (country-specific)"},
	},
	2: { // Region 2 — 420.000–450.000 MHz
		{Band: "70cm", Kind: "RNG", FromMHz: "420.000", ToMHz: "450.000", Note: "broad; many local options; country override required"},
		{Kind: "ALL", FromMHz: "420.000", ToMHz: "432.000", Note: "ATV/experimental/local option"},
		{Kind: "SSB", FromMHz: "432.000", ToMHz: "432.025", Note: "CW EME"},
		{Kind: "SSB", FromMHz: "432.025", ToMHz: "432.100", Note: "CW/MGM EME and weak signal"},
		{Kind: "SSB", FromMHz: "432.100", ToMHz: "432.300", Freq: "432.100", Note: "CW/SSB weak signal; SSB/CW calling"},
		{Kind: "BCN", FromMHz: "432.300", ToMHz: "432.400", Note: "beacons"},
		{Kind: "BCN", FromMHz: "432.400", ToMHz: "432.420", Note: "digital beacons/ACDS"},
		{Kind: "SSB", FromMHz: "432.420", ToMHz: "433.000", Note: "CW/SSB/DM"},
		{Kind: "DIG", FromMHz: "433.000", ToMHz: "433.050", Note: "ACDS"},
		{Kind: "DIG", FromMHz: "433.050", ToMHz: "433.100", Note: "IVG"},
		{Kind: "ALL", FromMHz: "433.100", ToMHz: "435.000", Note: "local option"},
		{Kind: "SAT", FromMHz: "435.000", ToMHz: "438.000", Note: "satellite exclusive"},
		{Kind: "ALL", FromMHz: "438.000", ToMHz: "450.000", Note: "local option"},
		// US common overrides (marked as local, not IARU).
		{Kind: "FM", Freq: "446.000", Note: "FM calling (US local practice)"},
		{Kind: "RPT", FromMHz: "442.000", ToMHz: "445.000", Note: "US common repeater outputs (+5 MHz shift)"},
		{Kind: "RPT", FromMHz: "447.000", ToMHz: "450.000", Note: "US common repeater outputs (+5 MHz shift)"},
	},
	3: { // Region 3 — 430.000–440.000 MHz
		{Band: "70cm", Kind: "RNG", FromMHz: "430.000", ToMHz: "440.000", Note: "broad; ISM coexistence; country override required"},
		{Kind: "ALL", FromMHz: "430.000", ToMHz: "431.900", Note: "all modes"},
		{Kind: "SSB", FromMHz: "431.900", ToMHz: "432.240", Note: "EME/weak signal"},
		{Kind: "ALL", FromMHz: "432.240", ToMHz: "435.000", Note: "all modes"},
		{Kind: "SAT", FromMHz: "435.000", ToMHz: "438.000", Note: "satellite all modes"},
		{Kind: "ALL", FromMHz: "438.000", ToMHz: "440.000", Note: "all modes; ISM coexistence — avoid interference"},
	},
}

// r3APRSKnown holds Region 3 country-specific APRS frequencies (not a global standard).
var r3APRSKnown = []vhfSeg{
	{Kind: "APR", Freq: "144.640", Note: "JP 9600 baud"},
	{Kind: "APR", Freq: "144.660", Note: "JP 1200 baud"},
	{Kind: "APR", Freq: "145.175", Note: "AU/Tasmania"},
	{Kind: "APR", Freq: "144.575", Note: "NZ"},
	{Kind: "APR", Freq: "144.640", Note: "CN/HK/TW"},
	{Kind: "APR", Freq: "144.620", Note: "KR"},
	{Kind: "APR", Freq: "145.525", Note: "TH"},
}

// dxAvoid holds a DX/DXpedition frequency segment to avoid.
type dxAvoid struct {
	Name string
	Freq string
}

// dxAvoidFreqs holds frequencies/segments to avoid per IARU region.
var dxAvoidFreqs = map[int]map[bandplan.BandName][]dxAvoid{
	1: { // Region 1 — Europe/Africa
		bandplan.Band20m: {{"DX", "14.195 ±5 kHz"}, {"EMG", "14.300"}},
		bandplan.Band40m: {{"EMG", "7.110"}, {"DX", "7.175–7.200"}},
	},
	3: { // Region 3 — Asia-Pacific
		bandplan.Band40m: {{"EMG", "7.110"}},
	},
}

// =============================================================================
// Non-amateur service profiles — CB, PMR446, FRS/GMRS
// =============================================================================
// These are NOT ham bands. They are regulated by CEPT/FCC/ACMA/national rules
// and are included for awareness only. The "NOT A HAM BAND" warning highlights
// that transmitting here requires appropriate equipment and licensing.

// cbChan holds a 27 MHz CB channel.
type cbChan struct {
	Ch   int
	Freq string // e.g. "26.965"
	Tag  string // e.g. "EMG", "CALL", "SSB", "ROAD"
}

// cbChannels is the common 40-channel 27 MHz CB plan (CEPT/FCC/many R3 countries).
// Channels 23–25 are historically out of order (old 23-chan numbering).
var cbChannels = []cbChan{
	{1, "26.965", ""},
	{2, "26.975", ""},
	{3, "26.985", ""},
	{4, "27.005", ""},
	{5, "27.015", ""},
	{6, "27.025", ""},
	{7, "27.035", ""},
	{8, "27.055", ""},
	{9, "27.065", "EMG"},
	{10, "27.075", ""},
	{11, "27.085", "Call"},
	{12, "27.105", ""},
	{13, "27.115", ""},
	{14, "27.125", "Call"},
	{15, "27.135", ""},
	{16, "27.155", ""},
	{17, "27.165", ""},
	{18, "27.175", ""},
	{19, "27.185", "Road"},
	{20, "27.205", ""},
	{21, "27.215", ""},
	{22, "27.225", ""},
	{23, "27.255", ""},
	{24, "27.235", ""},
	{25, "27.245", ""},
	{26, "27.265", ""},
	{27, "27.275", ""},
	{28, "27.285", ""},
	{29, "27.295", ""},
	{30, "27.305", ""},
	{31, "27.315", ""},
	{32, "27.325", ""},
	{33, "27.335", ""},
	{34, "27.345", ""},
	{35, "27.355", "SSB"},
	{36, "27.365", "SSB"},
	{37, "27.375", "SSB"},
	{38, "27.385", "SSB Call"},
	{39, "27.395", "SSB"},
	{40, "27.405", ""},
}

// pmrChan holds a PMR446 channel.
type pmrChan struct {
	Ch   int
	Freq string // e.g. "446.00625"
}

// pmr446Analog is the 16-channel analog FM PMR446 plan (CEPT/R1).
var pmr446Analog = []pmrChan{
	{1, "446.00625"},
	{2, "446.01875"},
	{3, "446.03125"},
	{4, "446.04375"},
	{5, "446.05625"},
	{6, "446.06875"},
	{7, "446.08125"},
	{8, "446.09375"},
	{9, "446.10625"},
	{10, "446.11875"},
	{11, "446.13125"},
	{12, "446.14375"},
	{13, "446.15625"},
	{14, "446.16875"},
	{15, "446.18175"},
	{16, "446.19375"},
}

// frsChan holds an FRS/GMRS channel.
type frsChan struct {
	Ch    int
	Freq  string
	FRS   bool
	GMRS  bool
	Tag   string
	RptIn string // GMRS repeater input (+5 MHz), empty if none
}

// frsGmrsChannels is the 22-channel FRS/GMRS plan (FCC/USA/R2).
var frsGmrsChannels = []frsChan{
	{1, "462.5625", true, true, "", ""},
	{2, "462.5875", true, true, "", ""},
	{3, "462.6125", true, true, "", ""},
	{4, "462.6375", true, true, "", ""},
	{5, "462.6625", true, true, "", ""},
	{6, "462.6875", true, true, "", ""},
	{7, "462.7125", true, true, "", ""},
	{8, "467.5625", true, false, "", ""},
	{9, "467.5875", true, false, "", ""},
	{10, "467.6125", true, false, "", ""},
	{11, "467.6375", true, false, "", ""},
	{12, "467.6625", true, false, "", ""},
	{13, "467.6875", true, false, "", ""},
	{14, "467.7125", true, false, "", ""},
	{15, "462.5500", true, true, "", "467.5500"},
	{16, "462.5750", true, true, "", "467.5750"},
	{17, "462.6000", true, true, "", "467.6000"},
	{18, "462.6250", true, true, "", "467.6250"},
	{19, "462.6500", true, true, "", "467.6500"},
	{20, "462.6750", true, true, "TRAVEL/EMG", "467.6750"},
	{21, "462.7000", true, true, "", "467.7000"},
	{22, "462.7250", true, true, "", "467.7250"},
}

// nonHamProfile describes a non-amateur service profile for a region hint.
type nonHamProfile struct {
	ID      string // e.g. "CB_CEPT_EU", "PMR446", "FRS_GMRS"
	Label   string // display label
	RangeLo string // e.g. "26.960"
	RangeHi string // e.g. "27.410"
	Mod     string // modulation
	Note    string // extra info
}

// nonHamProfiles returns the non-ham profiles relevant to an IARU region hint.
func nonHamProfiles(region int) []nonHamProfile {
	switch region {
	case 1:
		return []nonHamProfile{
			{"CB_CEPT_EU", "CB (CEPT/EU)", "26.960", "27.410", "FM/AM/SSB", "40 ch, 26.965–27.405 MHz"},
			{"PMR446_ANALOG", "PMR446 Analog", "446.00625", "446.19375", "FM", "16 ch, 12.5 kHz, 500 mW ERP"},
			{"PMR446_DIGITAL", "PMR446 Digital", "446.000", "446.200", "DMR/dPMR", "32×6.25 kHz or 16×12.5 kHz"},
		}
	case 2:
		return []nonHamProfile{
			{"CB_FCC_US", "CB (FCC/US)", "26.965", "27.405", "AM/FM/SSB", "40 ch, FCC CBRS"},
			{"FRS_GMRS", "FRS / GMRS", "462.5500", "467.7250", "FM", "22 ch; FRS licence-free, GMRS licensed"},
		}
	case 3:
		return []nonHamProfile{
			{"CB_HF_AU", "CB HF (AU)", "26.965", "27.405", "AM/SSB", "40 ch, ACMA HF CB"},
			{"CB_UHF_AU", "CB UHF (AU)", "476.425", "477.4125", "FM", "80 ch, Australian UHF CB"},
		}
	default:
		return nil
	}
}

// =============================================================================
// Broadcast emergency/off-grid presets (BRC) — receive-only reference data
// =============================================================================
// These are NOT ham bands. SW schedules are seasonal and should be checked
// against HFCC / broadcaster schedules when possible.

// bcastPreset is an emergency/off-grid broadcast frequency preset.
type bcastPreset struct {
	FreqKHz     int    // frequency in kHz
	Band        string // "LW", "MW", "SW"
	Station     string // station name
	Area        string // target area / use
	Priority    int    // 1 = top emergency/reference, 2 = secondary
	Reliability string // "high", "seasonal", "check_status"
	Note        string // optional extra info
}

// bcastPresets returns broadcast presets for an IARU region hint.
// These are regional emergency/news/time-signal SW/MW/LW stations useful
// for off-grid awareness. Marked BRC (broadcast / receive-only).
func bcastPresets(region int) []bcastPreset {
	switch region {
	case 1:
		return []bcastPreset{
			{225, "LW", "Polskie Radio Jedynka", "PL/Central Europe", 1, "high", "emergency-news fallback"},
			{7220, "SW", "RRI English", "Europe night", 1, "seasonal", ""},
			{9740, "SW", "RRI English", "Europe evening/night", 1, "seasonal", ""},
			{11960, "SW", "RRI English", "Europe morning", 1, "seasonal", ""},
			{15180, "SW", "RRI English", "Europe day/evening", 1, "seasonal", ""},
			{3955, "SW", "BBC World Service", "Europe possible", 2, "seasonal", ""},
			{9690, "SW", "Radio Exterior de España", "Europe/Africa/Atlantic", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Central Europe", 2, "", ""},
			{756, "MW", "Deutschlandfunk", "Central Europe", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe", 2, "", ""},
		}
	case 2:
		return []bcastPreset{
			{2500, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{5000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{10000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{15000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{20000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{3330, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{7850, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{14670, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{6180, "SW", "Rádio Nacional da Amazônia", "Brazil/Amazonia", 2, "seasonal", ""},
			{11780, "SW", "Rádio Nacional da Amazônia", "Brazil/Amazonia", 2, "seasonal", ""},
			{11620, "SW", "RRI English", "NA East/West", 2, "seasonal", ""},
			{11900, "SW", "RRI English", "NA East", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe (DX)", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Europe (DX)", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe (DX)", 2, "", ""},
		}
	case 3:
		return []bcastPreset{
			{7390, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", "emergency/news fallback"},
			{11725, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{13755, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{15720, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{17675, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{13580, "SW", "RRI English", "Japan", 2, "seasonal", ""},
			{11650, "SW", "RRI English", "Japan", 2, "seasonal", ""},
			{15410, "SW", "Akashvani / AIR", "South/Central Asia", 2, "seasonal", ""},
			{15280, "SW", "Akashvani / AIR", "Asia", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe (DX)", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Europe (DX)", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe (DX)", 2, "", ""},
		}
	default:
		return nil
	}
}

// bcastPresetsAll returns all broadcast presets from all regions, deduplicated
// by frequency+band. The Broadcast tab always shows the global list.
func bcastPresetsAll() []bcastPreset {
	seen := make(map[string]bool)
	var all []bcastPreset
	for r := 1; r <= 3; r++ {
		for _, bc := range bcastPresets(r) {
			key := fmt.Sprintf("%s|%d", bc.Band, bc.FreqKHz)
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, bc)
		}
	}
	return all
}

func (m *Model) handleBPLUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.bpl.cachedSig = "" // invalidate cache on resize
		return m, cmd
	case tea.KeyPressMsg:
		k := msg
		switch {
		case k.String() == "f1" || k.String() == "esc":
			m.screen = screenQSO
			return m, cmd
		case k.String() == "ctrl+e":
			cmd = tea.Batch(cmd, m.exportBPL())
			return m, cmd
		case k.String() == "left" || msg.Code == tea.KeyLeft || k.String() == "h":
			if m.bpl.tab > 0 {
				m.bpl.tab--
			} else {
				m.bpl.tab = bplTabCount - 1
			}
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "right" || msg.Code == tea.KeyRight || k.String() == "l":
			if m.bpl.tab < bplTabCount-1 {
				m.bpl.tab++
			} else {
				m.bpl.tab = 0
			}
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "tab":
			m.bpl.tab = (m.bpl.tab + 1) % bplTabCount
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "/":
			m.bpl.search = ""
			m.bpl.cachedSig = ""
		case k.String() == "up" || msg.Code == tea.KeyUp || k.String() == "k":
			if m.bpl.cursor > 0 {
				m.bpl.cursor--
				m.bpl.cachedSig = ""
			}
		case k.String() == "down" || msg.Code == tea.KeyDown || k.String() == "j":
			m.bpl.cursor++
			m.bpl.cachedSig = ""
		case k.String() == "pgup" || msg.Code == tea.KeyPgUp:
			m.bpl.cursor -= 10
			if m.bpl.cursor < 0 {
				m.bpl.cursor = 0
			}
			m.bpl.cachedSig = ""
		case k.String() == "pgdown" || msg.Code == tea.KeyPgDown:
			m.bpl.cursor += 10
			m.bpl.cachedSig = ""
		case k.String() == "home" || msg.Code == tea.KeyHome:
			m.bpl.cursor = 0
			m.bpl.cachedSig = ""
		case k.String() == "end" || msg.Code == tea.KeyEnd:
			m.bpl.cursor = 9999 // clamped later
			m.bpl.cachedSig = ""
		}
	}
	return m, cmd
}

// renderBPLContent applies scroll, cursor clamping, and cursor highlight
// to a full list of lines, returning the visible window as a string.
func (m *Model) renderBPLContent(lines []string) string {
	maxVisible := contentHeight(m.height) - 7
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Clamp cursor.
	if len(lines) == 0 {
		m.bpl.cursor = 0
		m.bpl.scroll = 0
		return ""
	}
	if m.bpl.cursor >= len(lines) {
		m.bpl.cursor = len(lines) - 1
	}
	if m.bpl.cursor < 0 {
		m.bpl.cursor = 0
	}

	// Keep cursor in view.
	if m.bpl.cursor < m.bpl.scroll {
		m.bpl.scroll = m.bpl.cursor
	}
	if m.bpl.cursor >= m.bpl.scroll+maxVisible {
		m.bpl.scroll = m.bpl.cursor - maxVisible + 1
	}
	if m.bpl.scroll < 0 {
		m.bpl.scroll = 0
	}
	maxScroll := len(lines) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.bpl.scroll > maxScroll {
		m.bpl.scroll = maxScroll
	}

	// Build visible window with cursor highlight.
	end := m.bpl.scroll + maxVisible
	if end > len(lines) {
		end = len(lines)
	}
	var b strings.Builder
	cursorLine := m.bpl.cursor - m.bpl.scroll
	for i := m.bpl.scroll; i < end; i++ {
		if i-m.bpl.scroll == cursorLine {
			b.WriteString(">")
			b.WriteString(lines[i])
		} else {
			b.WriteString(" ")
			b.WriteString(lines[i])
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *Model) viewBPL(l Layout) string {
	w := l.TerminalW
	if w < 40 {
		w = 80
	}
	ch := contentHeight(m.height)
	if ch < 8 {
		ch = 8
	}
	region := 1
	if m.App != nil && m.App.Logbook != nil {
		r := m.App.Logbook.Station.IARURegion
		if r >= 1 && r <= 3 {
			region = r
		}
	}

	// Cache: rebuild only when state changes.
	sig := fmt.Sprintf("%d|%d|%d|%d|%d|%d|%d|%s",
		m.bpl.tab, region, w, ch, m.bpl.scroll, m.bpl.cursor, m.bpl.bandSel, m.bpl.search)
	if m.bpl.cachedSig == sig && m.bpl.cachedView != "" {
		return m.bpl.cachedView
	}

	// Tab bar.
	var tabParts []string
	for i, name := range bplTabNames {
		if i == m.bpl.tab {
			tabParts = append(tabParts, S.TabActive.Render(name))
		} else {
			tabParts = append(tabParts, S.TabInactive.Render(name))
		}
	}
	tabBar := strings.Join(tabParts, " "+S.TabSeparator.Render("│")+" ")
	header := S.Title.Width(w).Render("Bandplan Iaru Region " + fmt.Sprintf("%d", region))

	// Render tab content — each sub-view returns full line list, renderBPLContent handles scroll/cursor.
	var lines []string
	switch m.bpl.tab {
	case bplTabHAM:
		lines = m.viewBPLHAM(region)
	case bplTabVHF:
		lines = m.viewBPLVHF(region)
	case bplTabCB:
		lines = m.viewBPLCB(region)
	case bplTabPMR:
		lines = m.viewBPLPMR(region)
	case bplTabBRC:
		lines = m.viewBPLBRC(region)
	}
	body := m.renderBPLContent(lines)

	// Disclaimer footer.
	footer := DimStyle.Width(w).Render(" Listen first. Check national rules. VHF/UHF often needs country/local overrides.")
	content := header + "\n " + tabBar + "\n\n" + body + "\n\n" + footer

	m.bpl.cachedView = fillBody(content, ch)
	m.bpl.cachedSig = sig
	return m.bpl.cachedView
}

// viewBPLHAM returns lines for the amateur HF band plan (160m–10m).
func (m *Model) viewBPLHAM(region int) []string {
	bp := bplForRegion(region)
	freqStr := func(f hamradio.Frequency) string { return fmt.Sprintf("%.3f", float64(f)/1e6) }

	var lines []string
	for _, name := range bandOrder {
		b, ok := bp[name]
		if !ok {
			continue
		}
		// Band summary line — just band name and range.
		summary := fmt.Sprintf("%-5s %s–%s", string(b.Name), freqStr(b.From), freqStr(b.To))
		lines = append(lines, summary)

		// Detail rows for this band.
		for _, p := range b.Portions {
			mode := shortModeTag(string(p.Mode))
			bw := ""
			if p.MaxBandwidth > 0 {
				bw = fmt.Sprintf(" BW %.0f Hz", float64(p.MaxBandwidth))
			}
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s–%s %s%s", freqStr(p.From), freqStr(p.To), mode, bw)))
		}
		// Special frequencies under this band.
		if emcom, ok := emcomFreqs[region]; ok {
			if f, ok := emcom[name]; ok {
				lines = append(lines, S.Error.Render(fmt.Sprintf("  %s EMG  emergency — avoid normal QSO", f)))
			}
		}
		if qrps, ok := qrpFreqs[region]; ok {
			if entries, ok := qrps[name]; ok {
				for _, e := range entries {
					lines = append(lines, fmt.Sprintf("  %s QRP %s centre", e.Freq, e.Mode))
				}
			}
		}
		if f, ok := qrsFreqs[name]; ok {
			lines = append(lines, fmt.Sprintf("  %s QRS slow CW centre", f))
		}
		if f, ok := ibpFreqs[name]; ok {
			lines = append(lines, fmt.Sprintf("  %s IBP beacon — avoid TX", f))
		}
		if f, ok := sstvFreqs[region]; ok {
			if freq, ok := f[name]; ok {
				lines = append(lines, fmt.Sprintf("  %s IMG SSTV/image", freq))
			}
		}
		if avoids, ok := dxAvoidFreqs[region]; ok {
			if entries, ok := avoids[name]; ok {
				for _, e := range entries {
					// Skip AVOID if already covered by an EMCOM entry (dedup).
					if e.Name == "EMG" {
						if emcom, ok2 := emcomFreqs[region]; ok2 {
							if _, ok3 := emcom[name]; ok3 {
								continue
							}
						}
					}
					if e.Name == "DX" {
						lines = append(lines, S.Warning.Render(fmt.Sprintf("  %s Avoid - reserved for DX", e.Freq)))
					} else {
						lines = append(lines, S.Error.Render(fmt.Sprintf("  %s AVOID %s", e.Freq, e.Name)))
					}
				}
			}
		}
	}

	return lines
}

// viewBPLVHF renders the VHF/UHF band plan (6m/4m/2m/70cm).
func (m *Model) viewBPLVHF(region int) []string {
	var lines []string

	// 6m and 4m overview from vhfCalling.
	if vhf, ok := vhfCalling[region]; ok {
		for _, v := range vhf {
			if v.Band != "" {
				lines = append(lines, fmt.Sprintf("%s %s–%s", v.Band, v.FromMHz, v.ToMHz))
			} else {
				tag := shortModeTag(v.Mode)
				lines = append(lines, fmt.Sprintf("  %s MHz  %s %s", v.Freq, tag, v.Note))
			}
		}
	}

	// 2m detailed.
	if segs, ok := vhf2mSeeds[region]; ok {
		lines = append(lines, "")
		for _, s := range segs {
			if s.Band != "" {
				lines = append(lines, fmt.Sprintf("%s %s–%s  %s", s.Band, s.FromMHz, s.ToMHz, s.Note))
			} else if s.ToMHz != "" {
				freq := s.Freq
				if freq != "" {
					freq = " CoA " + freq
				}
				sev := severityStyle(s.Kind)
				lines = append(lines, sev.Render(fmt.Sprintf("  %s–%s %s%s  %s", s.FromMHz, s.ToMHz, s.Kind, freq, s.Note)))
			} else {
				sev := severityStyle(s.Kind)
				note := s.Note
				if s.Kind == "LRA" {
					note += " (country-specific)"
				}
				lines = append(lines, sev.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, note)))
			}
		}
	}

	// 70cm detailed.
	if segs, ok := vhf70cmSeeds[region]; ok {
		lines = append(lines, "")
		for _, s := range segs {
			if s.Band != "" {
				lines = append(lines, fmt.Sprintf("%s %s–%s  %s", s.Band, s.FromMHz, s.ToMHz, s.Note))
			} else if s.ToMHz != "" {
				freq := s.Freq
				if freq != "" {
					freq = " CoA " + freq
				}
				sev := severityStyle(s.Kind)
				lines = append(lines, sev.Render(fmt.Sprintf("  %s–%s %s%s  %s", s.FromMHz, s.ToMHz, s.Kind, freq, s.Note)))
			} else {
				sev := severityStyle(s.Kind)
				note := s.Note
				if s.Kind == "LRA" {
					note += " (country-specific)"
				}
				lines = append(lines, sev.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, note)))
			}
		}
	}

	// R3 APRS.
	if region == 3 {
		lines = append(lines, "")
		lines = append(lines, "R3 APRS (country-specific):")
		for _, s := range r3APRSKnown {
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, s.Note)))
		}
	}
	return lines
}

// viewBPLCB renders CB channels.
func (m *Model) viewBPLCB(region int) []string {
	var lines []string
	profiles := nonHamProfiles(region)
	var cbProfile *nonHamProfile
	for i := range profiles {
		if profiles[i].ID == "CB_CEPT_EU" || profiles[i].ID == "CB_FCC_US" || profiles[i].ID == "CB_HF_AU" {
			cbProfile = &profiles[i]
			break
		}
	}

	if cbProfile != nil {
		lines = append(lines, S.Error.Render("NOT A HAM BAND")+" — "+cbProfile.Label)
		lines = append(lines, fmt.Sprintf("%s–%s MHz  %s  %s", cbProfile.RangeLo, cbProfile.RangeHi, cbProfile.Mod, cbProfile.Note))
		lines = append(lines, "")
		// Build fixed-width rows: "Ch XX  XXX.XXX  TAG" → 20 chars per channel.
		var chRows []string
		for _, ch := range cbChannels {
			tag := ch.Tag
			row := fmt.Sprintf("Ch %-2d  %s", ch.Ch, ch.Freq)
			if tag != "" {
				row += "  "
				if tag == "EMG" {
					row += S.Error.Render(tag)
				} else {
					row += tag
				}
			}
			chRows = append(chRows, row)
		}
		// 2-column layout: two per line, separated by 3 spaces.
		mid := (len(chRows) + 1) / 2
		for i := 0; i < mid; i++ {
			left := chRows[i]
			right := ""
			if i+mid < len(chRows) {
				right = chRows[i+mid]
			}
			// Pad left to consistent width for alignment.
			leftPad := lipgloss.NewStyle().Width(26).Render(left)
			lines = append(lines, leftPad+"   "+right)
		}
	} else {
		// UHF CB for R3.
		for _, p := range profiles {
			if p.ID == "CB_UHF_AU" {
				lines = append(lines, S.Error.Render("NOT A HAM BAND")+" — "+p.Label)
				lines = append(lines, fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			}
		}
	}
	return lines
}

// viewBPLPMR renders PMR446 channels and FRS/GMRS.
func (m *Model) viewBPLPMR(region int) []string {
	var lines []string
	profiles := nonHamProfiles(region)

	// No PMR profiles at all — show region-specific note.
	hasPMR := false
	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" || p.ID == "PMR446_DIGITAL" || p.ID == "FRS_GMRS" {
			hasPMR = true
			break
		}
	}
	if !hasPMR {
		if region == 3 {
			lines = append(lines, DimStyle.Render("No Asia-wide PMR446 equivalent. PMR446 exists in some Asian countries,"))
			lines = append(lines, DimStyle.Render("but check the specific country's licence-free radio allocation."))
		} else {
			lines = append(lines, DimStyle.Render("No licence-free radio profiles for this region."))
		}
		return lines
	}

	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" {
			lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+p.Label)
			lines = append(lines, fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			lines = append(lines, "")
			var chRows []string
			for _, ch := range pmr446Analog {
				chRows = append(chRows, fmt.Sprintf("Ch %-2d  %s", ch.Ch, ch.Freq))
			}
			mid := (len(chRows) + 1) / 2
			for i := 0; i < mid; i++ {
				right := ""
				if i+mid < len(chRows) {
					right = chRows[i+mid]
				}
				leftPad := lipgloss.NewStyle().Width(26).Render(chRows[i])
				lines = append(lines, leftPad+"   "+right)
			}
		}
		if p.ID == "PMR446_DIGITAL" {
			lines = append(lines, "")
			lines = append(lines, DimStyle.Render(p.Label+": "+p.RangeLo+"–"+p.RangeHi+" MHz  "+p.Mod+"  "+p.Note))
		}
	}
	// FRS/GMRS for R2.
	firstFRS := true
	for _, p := range profiles {
		if p.ID == "FRS_GMRS" {
			if firstFRS && len(lines) == 0 {
				// No PMR content — don't add blank separator line.
			} else {
				lines = append(lines, "")
			}
			lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+p.Label)
			lines = append(lines, fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			for _, ch := range frsGmrsChannels {
				svc := ""
				if ch.FRS && ch.GMRS {
					svc = "F+G"
				} else if ch.FRS {
					svc = "FRS"
				} else if ch.GMRS {
					svc = "GMR"
				}
				row := fmt.Sprintf("Ch %-2d %s  %-3s", ch.Ch, ch.Freq, svc)
				if ch.Tag != "" {
					row += " " + ch.Tag
				}
				if ch.RptIn != "" {
					row += " RPT+" + ch.RptIn
				}
				lines = append(lines, row)
			}
		}
	}
	return lines
}

// viewBPLBRC renders broadcast receive-only presets.
func (m *Model) viewBPLBRC(region int) []string {
	var lines []string
	bcasts := bcastPresetsAll()
	if len(bcasts) == 0 {
		return []string{DimStyle.Render("  No broadcast presets for this region.")}
	}
	lines = append(lines, S.Warning.Render("BROADCAST ONLY — receive-only reference"))
	lines = append(lines, DimStyle.Render("SW schedules are seasonal; check HFCC for current data."))
	lines = append(lines, "")

	// Sort by frequency ascending.
	sorted := make([]bcastPreset, len(bcasts))
	copy(sorted, bcasts)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FreqKHz < sorted[j].FreqKHz })

	for _, bc := range sorted {
		freqMHz := fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0)
		note := bc.Area
		if bc.Reliability == "seasonal" {
			note += " [seasonal]"
		}
		if bc.Note != "" {
			note += "  " + bc.Note
		}
		lines = append(lines, fmt.Sprintf("  %-6s %s MHz  %s  %s", bc.Band, freqMHz, bc.Station, note))
	}
	return lines
}

// shortModeTag returns a compact 3-letter tag for a mode string.
func shortModeTag(mode string) string {
	switch mode {
	case "CW":
		return "CW"
	case "Digital", "DIGITAL":
		return "DIG"
	case "Phone", "PHONE", "SSB":
		return "PHN"
	default:
		if len(mode) > 3 {
			return mode[:3]
		}
		return mode
	}
}

// severityStyle returns the appropriate Lip Gloss style for a segment kind.
func severityStyle(kind string) lipgloss.Style {
	switch kind {
	case "EMG", "AVOID":
		return S.Error
	case "DX":
		return S.Warning
	case "SAT":
		return S.Warning
	case "RNG":
		return S.Value
	default:
		return DimStyle
	}
}

// bplRowCount returns the total number of rows in the band plan table.
// Kept for export compatibility; the TUI no longer uses a single table.
func (m *Model) bplRowCount() int {
	region := 1
	if m.App != nil && m.App.Logbook != nil {
		r := m.App.Logbook.Station.IARURegion
		if r >= 1 && r <= 3 {
			region = r
		}
	}
	return len(bplRows(region))
}

// bplTableHeight returns the available table height for the band plan.
func bplTableHeight(termH int) int {
	h := contentHeight(termH) - 2
	if h < 3 {
		h = 3
	}
	return h
}

// bplForRegion returns the bandplan for the given IARU region.
func bplForRegion(r int) bandplan.Bandplan {
	switch r {
	case 2:
		return bandplan.IARURegion2
	case 3:
		return bandplan.IARURegion3
	default:
		return bandplan.IARURegion1
	}
}

// bplExportMsg carries the result of a band plan export.
type bplExportMsg struct {
	path string
	err  error
}

// exportBPL writes the full band plan as a Markdown document to cqops_bandplan.md
// in the CQOPS config directory, overwriting any existing file.
func (m *Model) exportBPL() tea.Cmd {
	return func() tea.Msg {
		region := 1
		if m.App != nil && m.App.Logbook != nil {
			r := m.App.Logbook.Station.IARURegion
			if r >= 1 && r <= 3 {
				region = r
			}
		}
		dir, err := config.ConfigDir()
		if err != nil {
			return bplExportMsg{err: err}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return bplExportMsg{err: err}
		}
		path := filepath.Join(dir, "cqops_bandplan.md")
		content := m.buildBPLMarkdown(region)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return bplExportMsg{err: err}
		}
		return bplExportMsg{path: path}
	}
}

// buildBPLMarkdown generates a Markdown document with the full band plan.
func (m *Model) buildBPLMarkdown(region int) string {
	var b strings.Builder

	// Main header.
	b.WriteString(fmt.Sprintf("# CQOps - Iaru Region %d - Bandplan\n\n", region))
	b.WriteString("> Band plans are guidance, not a licence. Check national rules. Listen first.\n")
	b.WriteString("> VHF/UHF repeaters, APRS and LoRa often need country/local overrides.\n")
	b.WriteString("> CB/PMR/BRC are non-amateur services; BRC is receive-only.\n\n")

	// HAM HF section.
	b.WriteString("## Ham Radio - HF\n\n")
	b.WriteString("| Band | From MHz | To MHz | Mode | From | To | BW Hz / Note |\n")
	b.WriteString("|------|----------|--------|------|------|----|-------------|\n")
	m.writeBPLMarkdownRows(&b, region)

	// VHF section.
	b.WriteString("\n## Ham Radio - VHF\n\n")
	b.WriteString("| Band | From MHz | To MHz | Mode | From | To | Note |\n")
	b.WriteString("|------|----------|--------|------|------|----|------|\n")
	m.writeVHFMarkdownRows(&b, region)

	// CB section.
	b.WriteString("\n## Citizen Band - CB\n\n")
	b.WriteString("| Ch | Freq MHz | Tag |\n")
	b.WriteString("|---|----------|-----|\n")
	m.writeCBMarkdownRows(&b, region)

	// PMR section.
	b.WriteString("\n## Personal Mobile Radio - PMR\n\n")
	m.writePMRMarkdownRows(&b, region)

	// Broadcast section.
	b.WriteString("\n## Broadcast\n\n")
	b.WriteString("| Band | Freq MHz | Station | Area |\n")
	b.WriteString("|------|----------|---------|------|\n")
	m.writeBRCMarkdownRows(&b, region)

	return b.String()
}

func (m *Model) writeBPLMarkdownRows(b *strings.Builder, region int) {
	bp := bplForRegion(region)
	freqStr := func(f hamradio.Frequency) string { return fmt.Sprintf("%.3f", float64(f)/1e6) }
	bwStr := func(bw hamradio.Frequency) string {
		if bw <= 0 {
			return ""
		}
		return fmt.Sprintf("%.0f", float64(bw))
	}
	for _, name := range bandOrder {
		bd, ok := bp[name]
		if !ok {
			continue
		}
		// Band header row.
		fmt.Fprintf(b, "| **%s** | %s | %s | | | | |\n", string(bd.Name), freqStr(bd.From), freqStr(bd.To))
		for _, p := range bd.Portions {
			fmt.Fprintf(b, "| | | | %s | %s | %s | %s |\n", string(p.Mode), freqStr(p.From), freqStr(p.To), bwStr(p.MaxBandwidth))
		}
		if emcom, ok := emcomFreqs[region]; ok {
			if f, ok := emcom[name]; ok {
				fmt.Fprintf(b, "| | | | **EMG** | %s | | emergency — avoid normal QSO |\n", f)
			}
		}
		if qrps, ok := qrpFreqs[region]; ok {
			if entries, ok := qrps[name]; ok {
				for _, e := range entries {
					fmt.Fprintf(b, "| | | | %s | %s | | QRP centre |\n", e.Mode, e.Freq)
				}
			}
		}
		if f, ok := qrsFreqs[name]; ok {
			fmt.Fprintf(b, "| | | | QRS | %s | | slow CW centre |\n", f)
		}
		if f, ok := ibpFreqs[name]; ok {
			fmt.Fprintf(b, "| | | | **IBP** | %s | | beacon — avoid TX |\n", f)
		}
		if sstv, ok := sstvFreqs[region]; ok {
			if f, ok := sstv[name]; ok {
				fmt.Fprintf(b, "| | | | IMG | %s | | SSTV/image |\n", f)
			}
		}
		if avoids, ok := dxAvoidFreqs[region]; ok {
			if entries, ok := avoids[name]; ok {
				for _, e := range entries {
					if e.Name == "EMG" {
						if emcom, ok2 := emcomFreqs[region]; ok2 {
							if _, ok3 := emcom[name]; ok3 {
								continue
							}
						}
					}
					if e.Name == "DX" {
						fmt.Fprintf(b, "| | | | **Avoid** | %s | | reserved for DX |\n", e.Freq)
					} else {
						fmt.Fprintf(b, "| | | | **AVOID** | %s | | %s |\n", e.Freq, e.Name)
					}
				}
			}
		}
	}
}

func (m *Model) writeVHFMarkdownRows(b *strings.Builder, region int) {
	// 6m/4m overview.
	if vhf, ok := vhfCalling[region]; ok {
		for _, v := range vhf {
			if v.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | | | | %s |\n", v.Band, v.FromMHz, v.ToMHz, v.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", v.Mode, v.Freq, v.Note)
			}
		}
	}
	// 2m.
	if segs, ok := vhf2mSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | %s | | | %s |\n", s.Band, s.FromMHz, s.ToMHz, s.Kind, s.Note)
			} else if s.ToMHz != "" {
				fmt.Fprintf(b, "| | %s | %s | %s | %s | | %s |\n", s.FromMHz, s.ToMHz, s.Kind, s.Freq, s.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
			}
		}
	}
	// 70cm.
	if segs, ok := vhf70cmSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | %s | | | %s |\n", s.Band, s.FromMHz, s.ToMHz, s.Kind, s.Note)
			} else if s.ToMHz != "" {
				fmt.Fprintf(b, "| | %s | %s | %s | %s | | %s |\n", s.FromMHz, s.ToMHz, s.Kind, s.Freq, s.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
			}
		}
	}
	// R3 APRS.
	if region == 3 {
		for _, s := range r3APRSKnown {
			fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
		}
	}
}

func (m *Model) writeCBMarkdownRows(b *strings.Builder, region int) {
	profiles := nonHamProfiles(region)
	for _, p := range profiles {
		if p.ID == "CB_CEPT_EU" || p.ID == "CB_FCC_US" || p.ID == "CB_HF_AU" {
			b.WriteString(fmt.Sprintf("\n**%s** — %s–%s MHz, %s\n\n", p.Label, p.RangeLo, p.RangeHi, p.Mod))
		}
	}
	for _, ch := range cbChannels {
		tag := ch.Tag
		if tag == "" {
			tag = "-"
		}
		fmt.Fprintf(b, "| %d | %s | %s |\n", ch.Ch, ch.Freq, tag)
	}
	// UHF CB for R3.
	for _, p := range profiles {
		if p.ID == "CB_UHF_AU" {
			b.WriteString(fmt.Sprintf("\n**%s** — %s–%s MHz, %s, %s\n\n", p.Label, p.RangeLo, p.RangeHi, p.Mod, p.Note))
		}
	}
}

func (m *Model) writePMRMarkdownRows(b *strings.Builder, region int) {
	profiles := nonHamProfiles(region)
	hasPMR := false
	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" || p.ID == "PMR446_DIGITAL" || p.ID == "FRS_GMRS" {
			hasPMR = true
			break
		}
	}
	if !hasPMR {
		b.WriteString("No licence-free radio profiles for this region.\n\n")
		if region == 3 {
			b.WriteString("PMR446 exists in some Asian countries, but check the specific country's licence-free radio allocation.\n\n")
		}
		return
	}
	for _, p := range profiles {
		switch p.ID {
		case "PMR446_ANALOG":
			b.WriteString(fmt.Sprintf("### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			b.WriteString("| Ch | Freq MHz |\n")
			b.WriteString("|----|----------|\n")
			for _, ch := range pmr446Analog {
				fmt.Fprintf(b, "| %d | %s |\n", ch.Ch, ch.Freq)
			}
		case "PMR446_DIGITAL":
			b.WriteString(fmt.Sprintf("\n### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
		case "FRS_GMRS":
			b.WriteString(fmt.Sprintf("### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			b.WriteString("| Ch | Freq MHz | Service | Tag | Repeater |\n")
			b.WriteString("|----|----------|---------|-----|----------|\n")
			for _, ch := range frsGmrsChannels {
				svc := ""
				if ch.FRS && ch.GMRS {
					svc = "FRS+GMRS"
				} else if ch.FRS {
					svc = "FRS"
				} else if ch.GMRS {
					svc = "GMRS"
				}
				tag := ch.Tag
				if tag == "" {
					tag = "-"
				}
				rpt := ch.RptIn
				if rpt == "" {
					rpt = "-"
				}
				fmt.Fprintf(b, "| %d | %s | %s | %s | %s |\n", ch.Ch, ch.Freq, svc, tag, rpt)
			}
		}
	}
}

func (m *Model) writeBRCMarkdownRows(b *strings.Builder, region int) {
	presets := bcastPresetsAll()
	sorted := make([]bcastPreset, len(presets))
	copy(sorted, presets)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].FreqKHz < sorted[j].FreqKHz })
	for _, bc := range sorted {
		freqMHz := fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0)
		area := bc.Area
		if bc.Reliability == "seasonal" {
			area += " [seasonal]"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s |\n", bc.Band, freqMHz, bc.Station, area)
	}
}

// stripANSI removes ANSI escape sequences from a string.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	state := 0
	for i := 0; i < len(s); i++ {
		switch state {
		case 0:
			if s[i] == '\x1b' {
				state = 1
			} else {
				b.WriteByte(s[i])
			}
		case 1:
			if s[i] == '[' {
				state = 2
			} else {
				state = 0
			}
		case 2:
			if s[i] >= '@' && s[i] <= '~' {
				state = 0
			}
		}
	}
	return b.String()
}

// bplColumnWidths returns the 7 column widths for the given available width.
// Mirrors the distribution logic in viewBPL.
func bplColumnWidths(availableW int) []int {
	cols := []int{6, 10, 10, 10, 10, 10, 9}
	minTotal := 0
	for _, w := range cols {
		minTotal += w
	}
	gaps := len(cols) - 1
	extra := availableW - gaps - minTotal
	if extra > 0 {
		distributed := 0
		for i := range cols {
			var share int
			switch i {
			case 0: // Band
				share = extra * 1 / 10
			case 1: // From MHz
				share = extra * 1 / 10
			case 2: // To MHz
				share = extra * 1 / 10
			case 3: // Mode
				share = extra * 2 / 10
			case 4: // From
				share = extra * 1 / 10
			case 5: // To
				share = extra * 1 / 10
			case 6: // BW Hz
				share = extra * 3 / 10
			}
			cols[i] += share
			distributed += share
		}
		if leftover := extra - distributed; leftover > 0 {
			cols[len(cols)-1] += leftover
		}
	}
	return cols
}

// bplRows builds all table rows for the BPL page (HF + VHF + non-ham + BRC).
// This is shared between viewBPL (TUI) and buildBPLText (export) so the export
// matches exactly what is shown on F7.
func bplRows(region int) []table.Row {
	bp := bplForRegion(region)

	freqStr := func(f hamradio.Frequency) string {
		return fmt.Sprintf("%.3f", float64(f)/1e6)
	}
	bwStr := func(bw hamradio.Frequency) string {
		if bw <= 0 {
			return ""
		}
		return fmt.Sprintf("%.0f", float64(bw))
	}

	var rows []table.Row

	// --- HF bandplan ---
	for _, name := range bandOrder {
		b, ok := bp[name]
		if !ok {
			continue
		}
		rows = append(rows, table.Row{
			string(b.Name),
			freqStr(b.From), freqStr(b.To),
			"", "", "", "",
		})
		for _, p := range b.Portions {
			rows = append(rows, table.Row{
				"",
				"", "",
				string(p.Mode),
				freqStr(p.From), freqStr(p.To),
				bwStr(p.MaxBandwidth),
			})
		}
		if emcom, ok := emcomFreqs[region]; ok {
			if freq, ok := emcom[name]; ok {
				rows = append(rows, table.Row{"", "", "", "EMCOM", freq, "", ""})
			}
		}
		if qrps, ok := qrpFreqs[region]; ok {
			if entries, ok := qrps[name]; ok {
				for _, e := range entries {
					rows = append(rows, table.Row{"", "", "", e.Mode, e.Freq, "", ""})
				}
			}
		}
		if sstv, ok := sstvFreqs[region]; ok {
			if freq, ok := sstv[name]; ok {
				rows = append(rows, table.Row{"", "", "", "SSTV", freq, "", ""})
			}
		}
		if freq, ok := ibpFreqs[name]; ok {
			rows = append(rows, table.Row{"", "", "", "IBP", freq, "", ""})
		}
		if freq, ok := qrsFreqs[name]; ok {
			rows = append(rows, table.Row{"", "", "", "QRS", freq, "", ""})
		}
		if avoids, ok := dxAvoidFreqs[region]; ok {
			if entries, ok := avoids[name]; ok {
				for _, e := range entries {
					rows = append(rows, table.Row{"", "", "", "AVOID", e.Freq, "", e.Name})
				}
			}
		}
	}

	// --- VHF/UHF overview (10m/6m/4m) ---
	if vhf, ok := vhfCalling[region]; ok {
		for _, v := range vhf {
			if v.Band != "" {
				rows = append(rows, table.Row{v.Band, v.FromMHz, v.ToMHz, "", "", "", v.Note})
			} else {
				rows = append(rows, table.Row{"", "", "", v.Mode, v.Freq, "", v.Note})
			}
		}
	}

	// --- Detailed 2m bandplan ---
	if segs, ok := vhf2mSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				rows = append(rows, table.Row{s.Band, s.FromMHz, s.ToMHz, s.Kind, "", "", s.Note})
			} else if s.ToMHz != "" {
				rows = append(rows, table.Row{"", s.FromMHz, s.ToMHz, s.Kind, s.Freq, "", s.Note})
			} else {
				rows = append(rows, table.Row{"", "", "", s.Kind, s.Freq, "", s.Note})
			}
		}
	}

	// --- Detailed 70cm bandplan ---
	if segs, ok := vhf70cmSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				rows = append(rows, table.Row{s.Band, s.FromMHz, s.ToMHz, s.Kind, "", "", s.Note})
			} else if s.ToMHz != "" {
				rows = append(rows, table.Row{"", s.FromMHz, s.ToMHz, s.Kind, s.Freq, "", s.Note})
			} else {
				rows = append(rows, table.Row{"", "", "", s.Kind, s.Freq, "", s.Note})
			}
		}
	}

	// --- R3 APRS country frequencies ---
	if region == 3 {
		for _, s := range r3APRSKnown {
			rows = append(rows, table.Row{"", "", "", s.Kind, s.Freq, "", s.Note})
		}
	}

	warnStyle := S.Error

	// --- Non-amateur service profiles ---
	for _, p := range nonHamProfiles(region) {
		rows = append(rows, table.Row{"", "", "", "", "", "", ""})
		rows = append(rows, table.Row{
			"", "", "",
			warnStyle.Render("NOT A HAM BAND"),
			warnStyle.Render(p.Label), "", p.Note,
		})
		rows = append(rows, table.Row{p.ID, p.RangeLo, p.RangeHi, p.Mod, "", "", ""})
		switch p.ID {
		case "CB_CEPT_EU", "CB_FCC_US", "CB_HF_AU":
			for _, ch := range cbChannels {
				rows = append(rows, table.Row{
					fmt.Sprintf("Ch %d", ch.Ch), "", "", "", ch.Freq, "", ch.Tag,
				})
			}
		case "CB_UHF_AU":
			rows = append(rows, table.Row{"", "", "", "", "476.425–477.4125", "", "80 ch FM (country-specific)"})
		case "PMR446_ANALOG":
			for _, ch := range pmr446Analog {
				rows = append(rows, table.Row{fmt.Sprintf("Ch %d", ch.Ch), "", "", "FM", ch.Freq, "", ""})
			}
		case "PMR446_DIGITAL":
			rows = append(rows, table.Row{"", "", "", "", "446.000–446.200", "", "DMR Tier I / dPMR446; 32×6.25 kHz ch"})
		case "FRS_GMRS":
			for _, ch := range frsGmrsChannels {
				svc := ""
				if ch.FRS && ch.GMRS {
					svc = "FRS+GMRS"
				} else if ch.FRS {
					svc = "FRS"
				} else if ch.GMRS {
					svc = "GMRS"
				}
				note := ch.Tag
				if ch.RptIn != "" {
					if note != "" {
						note += " "
					}
					note += "RPT+" + ch.RptIn
				}
				rows = append(rows, table.Row{
					fmt.Sprintf("Ch %d", ch.Ch), "", "", svc, ch.Freq, "", note,
				})
			}
		}
	}

	// --- Broadcast presets (BRC) ---
	if bcasts := bcastPresetsAll(); len(bcasts) > 0 {
		rows = append(rows, table.Row{"", "", "", "", "", "", ""})
		rows = append(rows, table.Row{
			"", "", "",
			warnStyle.Render("BROADCAST ONLY"),
			"", "", "Receive-only reference; seasonal schedules",
		})
		rows = append(rows, table.Row{"Band", "Freq MHz", "", "BRC", "Station", "", "Area / note"})
		for _, bc := range bcasts {
			prio := "P2"
			if bc.Priority == 1 {
				prio = "P1"
			}
			note := bc.Area
			if bc.Reliability == "seasonal" {
				note += "  [seasonal]"
			} else if bc.Reliability == "check_status" {
				note += "  [check status]"
			}
			if bc.Note != "" {
				note += "  " + bc.Note
			}
			rows = append(rows, table.Row{
				bc.Band,
				fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0), "",
				"BRC " + prio,
				bc.Station, "", note,
			})
		}
	}

	return rows
}

// =============================================================================
// CON placeholder pane (F3)
// =============================================================================

func (m *Model) handleCONUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, cmd
	case tea.KeyPressMsg:
		switch msg.String() {
		case "f1", "esc":
			m.screen = screenQSO
			return m, cmd
		}
	}
	return m, cmd
}

func (m *Model) viewCON(l Layout) string {
	return fillBody(DimStyle.Width(l.TerminalW).Align(lipgloss.Center).Render("CON — placeholder"), l.ContentH)
}
