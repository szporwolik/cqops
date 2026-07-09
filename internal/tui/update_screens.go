package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/ftl/hamradio/bandplan"
	"github.com/ftl/hamradio/dxcc"
	"github.com/ftl/hamradio/scp"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
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
	if m.ui.rigChooser.needsRefresh {
		m.ui.rigChooser.needsRefresh = false
		m.refreshRigClient()
		m.refreshRotorClient()
		m.restartWSJTXForActiveRig()
	}
	if m.ui.rigChooser.done {
		m.screen = screenMainMenu
		m.refreshRigClient()
		m.refreshRotorClient()
		m.restartWSJTXForActiveRig()
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
		m.needRefresh = true // contest changes may affect active contest / filtering
	}
	if m.ui.contestChooser.done {
		m.screen = screenMainMenu
	}
	return m, cmd
}

func (m *Model) handleOperatorUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.operatorChooser.width = m.width
	m.ui.operatorChooser.height = m.height

	_, opCmd := m.ui.operatorChooser.Update(msg)
	cmd = tea.Batch(cmd, opCmd)
	if m.ui.operatorChooser.done {
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
			m.App.Config.General.Units = m.ui.configMenu.distanceUnit
			m.App.Config.General.Timezone = m.ui.configMenu.timezone
			m.App.Config.General.RenderMap = m.ui.configMenu.renderMap
			m.App.Config.General.DrawGrayline = m.ui.configMenu.drawGrayline
			m.App.Config.General.PictureAtQRZPane = m.ui.configMenu.pictureAtQRZ
			m.App.Config.General.SolarAtQSOPane = m.ui.configMenu.solarAtQSO
			m.App.Config.General.UseCTY = m.ui.configMenu.useCTY
			m.App.Config.General.UseSCP = m.ui.configMenu.useSCP
			m.App.Config.General.UseRef = m.ui.configMenu.useRef
			m.App.Config.General.Debug = m.ui.configMenu.debugMode
			m.App.Config.General.KittyGraphics = m.ui.configMenu.kittyGraphics
			applog.SetDebugMode(m.ui.configMenu.debugMode)
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

func (m *Model) handleIntegrationUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	m.ui.integrationMenu.width = m.width
	m.ui.integrationMenu.height = m.height
	m.ui.integrationMenu.inetOnline = m.inetOnline
	m.ui.integrationMenu.aprsOnline = m.aprsConnected()
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
			dxcE, dxcHost, dxcPort, dxcLogin, qrzE, qrzUser, qrzPass, httpE, httpAddr, httpPort, httpHdr1, httpHdr2, httpLogo, httpEvtStart := m.ui.integrationMenu.Values()

			// Restart the HTTP server when address, port, or enabled
			// state actually change, OR when the server should be running
			// but isn't (silent crash recovery). Header/logo changes are
			// picked up by pushDashboardState — no restart needed.
			needHTTPRestart := httpE != m.App.Config.Integrations.HTTPServer.Enabled ||
				httpAddr != m.App.Config.Integrations.HTTPServer.Address ||
				httpPort != m.App.Config.Integrations.HTTPServer.Port ||
				(httpE && !m.http.online)

			m.App.Config.Integrations.DXC.Enabled = dxcE
			m.App.Config.Integrations.DXC.Host = dxcHost
			m.App.Config.Integrations.DXC.Port = dxcPort
			m.App.Config.Integrations.DXC.Login = dxcLogin
			m.App.Config.Integrations.QRZ.Enabled = qrzE
			m.App.Config.Integrations.QRZ.User = qrzUser
			m.App.Config.Integrations.QRZ.Pass = qrzPass
			m.App.Config.Integrations.HTTPServer.Enabled = httpE
			m.App.Config.Integrations.HTTPServer.Address = httpAddr
			m.App.Config.Integrations.HTTPServer.Port = httpPort
			m.App.Config.Integrations.HTTPServer.Header1 = httpHdr1
			m.App.Config.Integrations.HTTPServer.Header2 = httpHdr2
			m.App.Config.Integrations.HTTPServer.ClubLogo = httpLogo
			m.App.Config.Integrations.HTTPServer.EventStart = httpEvtStart

			// GPS integration.
			gpsWasEnabled := m.App.Config.Integrations.GPS.Enabled
			gpsWasService := m.App.Config.Integrations.GPS.Service
			m.App.Config.Integrations.GPS.Enabled = m.ui.integrationMenu.gpsEnabled
			m.App.Config.Integrations.GPS.Service = m.ui.integrationMenu.gpsServiceName()
			m.App.Config.Integrations.GPS.GridPrecision = m.ui.integrationMenu.gpsGridPrecision
			m.App.Config.Integrations.GPS.Port = m.ui.integrationMenu.gpsPort.Value()
			m.App.Config.Integrations.GPS.BaudRate = m.ui.integrationMenu.gpsBaudRate
			m.App.Config.Integrations.GPS.DTR = m.ui.integrationMenu.gpsDTR
			m.App.Config.Integrations.GPS.RTS = m.ui.integrationMenu.gpsRTS
			m.App.Config.Integrations.GPS.GPSDHost = m.ui.integrationMenu.gpsdHost.Value()
			m.App.Config.Integrations.GPS.GPSDPort = m.ui.integrationMenu.gpsdPort.Value()

			// APRS
			aprsWasEnabled := m.App.Config.Integrations.APRS.Enabled
			aprsWasService := m.App.Config.Integrations.APRS.Service
			aprsWasServer := m.App.Config.Integrations.APRS.Server
			aprsWasKISSHost := m.App.Config.Integrations.APRS.KISSServerHost
			aprsWasKISSPort := m.App.Config.Integrations.APRS.KISSServerPort
			aprsWasPort := m.App.Config.Integrations.APRS.Port
			aprsWasBaud := m.App.Config.Integrations.APRS.BaudRate
			m.App.Config.Integrations.APRS.Enabled = m.ui.integrationMenu.aprsEnabled
			m.App.Config.Integrations.APRS.Service = m.ui.integrationMenu.aprsServiceName()
			m.App.Config.Integrations.APRS.Server = m.ui.integrationMenu.aprsServer.Value()
			m.App.Config.Integrations.APRS.KISSServerHost = m.ui.integrationMenu.aprsKISSHost.Value()
			m.App.Config.Integrations.APRS.KISSServerPort = m.ui.integrationMenu.aprsKISSPort.Value()
			m.App.Config.Integrations.APRS.Port = m.ui.integrationMenu.aprsPort.Value()
			m.App.Config.Integrations.APRS.BaudRate = m.ui.integrationMenu.aprsBaudRate
			m.App.Config.Integrations.APRS.DataBits = m.ui.integrationMenu.aprsDataBits
			m.App.Config.Integrations.APRS.Parity = m.ui.integrationMenu.aprsParityName()
			m.App.Config.Integrations.APRS.StopBits = m.ui.integrationMenu.aprsStopBitsName()
			m.App.Config.Integrations.APRS.DTR = m.ui.integrationMenu.aprsDTR
			m.App.Config.Integrations.APRS.RTS = m.ui.integrationMenu.aprsRTS

			m.saveConfig("Settings saved")
			applog.Info("Integration config saved, restarting services")

			m.resetDXC()
			if needHTTPRestart {
				m.restartHTTPServer()
			}
			// GPS: start, stop, or restart based on config changes.
			gpsNowEnabled := m.App.Config.Integrations.GPS.Enabled
			gpsServiceChanged := gpsNowEnabled && gpsWasEnabled &&
				m.App.Config.Integrations.GPS.Service != gpsWasService
			switch {
			case gpsNowEnabled && (!gpsWasEnabled || gpsServiceChanged):
				cmd = tea.Batch(cmd, m.startGPS())
			case !gpsNowEnabled && gpsWasEnabled:
				m.stopGPS()
			}
			// APRS: restart if config changed.
			if m.App.Config.Integrations.APRS.Enabled != aprsWasEnabled ||
				m.App.Config.Integrations.APRS.Service != aprsWasService ||
				m.App.Config.Integrations.APRS.Server != aprsWasServer ||
				m.App.Config.Integrations.APRS.KISSServerHost != aprsWasKISSHost ||
				m.App.Config.Integrations.APRS.KISSServerPort != aprsWasKISSPort ||
				m.App.Config.Integrations.APRS.Port != aprsWasPort ||
				m.App.Config.Integrations.APRS.BaudRate != aprsWasBaud {
				m.App.MaybeRestartAPRS()
			}
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
		case "operator":
			m.ui.operatorChooser = NewOperatorChooser(m.App, m.toasts)
			m.ui.operatorChooser.width = m.width
			m.ui.operatorChooser.height = m.height
			m.screen = screenOperator
		case "integration":
			m.ui.integrationMenu = NewIntegrationMenu(m.App.Config)
			m.ui.integrationMenu.width = m.width
			m.ui.integrationMenu.height = m.height
			m.screen = screenIntegration
		case "notifications":
			m.ui.notifMenu = NewNotificationsMenu(m.App.Config)
			m.ui.notifMenu.width = m.width
			m.ui.notifMenu.height = m.height
			m.screen = screenNotifications
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
	// After returning from F2/ESC, force SetSize on the next frame
	// using whatever dimensions viewPartner() last computed. Don't
	// invent approximate sizes — that misaligns the Kitty grid.
	if m.photo.partnerPicNeedSize {
		m.photo.partnerPicNeedSize = false
		m.photo.partnerPicLastW = 0
		m.photo.partnerPicLastH = 0
	}
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
			m.photo.partnerPicURL = ""
			m.photo.partnerPicNeedLoad = false
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
	call := strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Callsign))

	// Lazy-init per-callsign fetch timestamp map.
	if m.psk.lastFetchByCall == nil {
		m.psk.lastFetchByCall = make(map[string]time.Time)
	}

	// Detect callsign change (e.g. logbook cycled) — reset caches
	// and allow an immediate fetch for the new callsign. The fetching
	// flag prevents concurrent fetches; rapid logbook toggling is rare
	// in normal ham radio operation.
	if call != "" && call != m.psk.lastCall {
		m.psk.fetched = false
		m.psk.lastCall = call
		m.psk.spots = nil
		m.psk.spotKey = ""
		m.psk.view = ""
		m.psk.viewKey = ""
	}

	// Trigger initial fetch when first entering the tab (not yet fetched, not already fetching).
	if !m.psk.fetched && !m.psk.fetching && m.inetOnline {
		if call != "" {
			m.psk.fetching = true
			return m, tea.Batch(cmd, m.pskFetchCmd())
		}
	}
	// Auto-refresh: if data for this callsign is older than 5 minutes,
	// trigger a background refresh (per-callsign, not global).
	if m.psk.fetched && !m.psk.fetching && m.inetOnline {
		last := m.psk.lastFetchByCall[call]
		if !last.IsZero() && time.Since(last) >= 5*time.Minute {
			m.psk.fetching = true
			return m, tea.Batch(cmd, m.pskFetchCmd())
		}
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
			m.pskResetCaches()
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
			m.pskResetCaches()
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
			m.pskResetCaches()
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
	band := qso.DeriveBand(freqHz / 1_000_000)
	if band == "" {
		return "other"
	}
	return band
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
	m.pskResetCaches()
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
			if m.App.Logbook.ActiveContest != "" {
				ct := m.App.Config.Contests[m.App.Logbook.ActiveContest]
				m.ui.logbookEditor.SetContestID(m.App.Logbook.ActiveContest, config.ContestDisplayName(&ct), ct.ContestID)
			} else {
				m.ui.logbookEditor.SetContestID("", "", "")
			}
			return m, m.refreshQSOS()
		}
	}

	// Detect resize — pageSize depends on terminal height, so reload.
	oldW, oldH := m.ui.logbookEditor.width, m.ui.logbookEditor.height
	m.ui.logbookEditor.width = m.width
	m.ui.logbookEditor.height = m.height
	m.ui.logbookEditor.Offline = m.Offline || !m.inetOnline
	if m.width != oldW || m.height != oldH {
		m.ui.logbookEditor.needsReload = true
	}
	_, editorCmd := m.ui.logbookEditor.Update(msg)
	var refreshCmd tea.Cmd
	if em, ok := msg.(editorMsg); ok {
		if em.toastWarn != "" {
			m.toasts.Warn(em.toastWarn)
		}
		if em.err != nil && em.wlQSOID == 0 {
			m.toasts.Error(em.err.Error())
		}
		if em.deleted != 0 {
			m.toasts.Success(fmt.Sprintf("QSO %s from %s deleted", em.delCall, em.delDate))
			refreshCmd = m.refreshQSOS()
		}
		if em.saved != 0 {
			m.toasts.Success(fmt.Sprintf("QSO %s from %s saved", em.saveCall, em.saveDate))
			refreshCmd = m.refreshQSOS()
		}
		if em.purged {
			m.toasts.Success("Logbook purged")
			m.ui.logbookEditor.wlLastFetchedID = 0
			m.ui.logbookEditor.needsReload = true
			refreshCmd = m.refreshQSOS()
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
	return m, tea.Batch(cmd, editorCmd, refreshCmd)
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
	bplTabPORT  = 5 // portable SOTA/POTA starting areas
	bplTabCount = 6
)

var bplTabNames = []string{"Ham Radio - HF", "Ham Radio - VHF", "Citizen Band - CB", "Personal Mobile Radio - PMR", "Broadcast", "Portable"}
var bplTabShortNames = []string{"HF", "VHF", "CB", "PMR", "BC", "PORT"}

type bplState struct {
	scroll  int
	cursor  int
	tab     int    // active filter tab
	bandSel int    // selected HF band index for detail view (HAM tab)
	search  string // search/filter substring (empty = no filter)

	// Cache.
	cachedView string
	cachedSig  string // "tab|region|w|h|scroll|cursor|bandSel|search"

	// Pre-built line lists — avoid rebuilding hundreds of fmt.Sprintf calls
	// on every tab switch or resize. Only rebuild when tab or region changes.
	cachedLines    []string
	cachedLinesKey string // "tab|region|search"

	// Cached row count — avoids rebuilding full band plan just to count rows.
	cachedRowCount  int
	cachedRowRegion int

	// Tune state.
	tuneCancel context.CancelFunc // cancels previous in-flight BPL tune
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

// portablePreset is a suggested portable starting area for a band+mode.
// These are NOT official SOTA/POTA channels — always check bandplan, listen,
// ask QRL, and self-spot your exact frequency.
type portablePreset struct {
	Band     string // "40m", "20m", etc.
	CW       string // suggested CW start, e.g. "7.032"
	CWRange  string // "7.030–7.035"
	CWNote   string // optional extra info
	SSB      string // suggested SSB start, e.g. "7.118"
	SSBRange string // "7.090–7.130"
	SSBNote  string // optional extra info
}

// portablePresets holds per-IARU-region portable SOTA/POTA starting areas.
// Frequencies in MHz, sourced from IARU band plans and practical field reports.
var portablePresets = map[int][]portablePreset{
	1: { // IARU Region 1 — Europe, Africa, Middle East, Northern Asia
		{"40m", "7.032", "7.030–7.035", "", "7.118", "7.090–7.130", "7.090 QRP CoA; 7.118/7.144 common EU portable"},
		{"30m", "10.118", "10.116–10.120", "QRP CoA 10.116", "", "", "SSB not permitted"},
		{"20m", "14.062", "14.060–14.065", "", "14.285", "14.250–14.300", ""},
		{"17m", "18.086", "18.086–18.090", "", "18.130", "18.130–18.150", ""},
		{"15m", "21.062", "21.060–21.065", "", "21.285", "21.285–21.300", ""},
		{"10m", "28.062", "28.060–28.065", "", "28.450", "28.400–28.500", "28.360 QRP CoA"},
	},
	2: { // IARU Region 2 — Americas
		{"40m", "7.032", "7.030–7.035", "", "7.285", "7.200–7.290", "Avoid 7.118 as generic Americas default"},
		{"30m", "10.116", "10.116–10.120", "", "", "", "SSB not permitted"},
		{"20m", "14.060", "14.060–14.065", "", "14.285", "14.250–14.300", ""},
		{"17m", "18.086", "18.086–18.090", "", "18.130", "18.130–18.150", ""},
		{"15m", "21.060", "21.060–21.065", "", "21.285", "21.285–21.300", ""},
		{"10m", "28.060", "28.060–28.065", "", "28.450", "28.400–28.500", "28.360 QRP CoA"},
	},
	3: { // IARU Region 3 — Asia-Pacific
		{"40m", "7.032", "7.030–7.035", "", "7.090", "7.090–7.120", "7.090 QRP CoA; 7.095 DX phone CoA"},
		{"30m", "10.116", "10.116–10.120", "Some admins allow phone; not global default", "", "", ""},
		{"20m", "14.060", "14.060–14.065", "", "14.285", "14.250–14.300", ""},
		{"17m", "18.086", "18.086–18.090", "", "18.130", "18.130–18.150", ""},
		{"15m", "21.060", "21.060–21.065", "", "21.285", "21.285–21.300", "21.295 DX phone CoA"},
		{"10m", "28.060", "28.055–28.065", "28.055 QRS CoA", "28.450", "28.400–28.500", "28.360 QRP CoA"},
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
