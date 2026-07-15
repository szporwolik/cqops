package tui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Global key bindings (F1-F10, etc.) — independent of current screen
// =============================================================================

// handleGlobalKeys processes top-level function key bindings (F1-F10, etc.)
// that are independent of the current screen. Returns true if the key was handled.
func (m *Model) handleGlobalKeys(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	// Block tab switching during Wavelog download (full-screen operation).
	if m.ui.logbookEditor != nil && m.ui.logbookEditor.isDownloadActive() {
		// Only allow F10 (quit) to pass through.
		if !key.Matches(msg, m.keys.Quit) {
			return nil, true
		}
	}

	switch {
	case m.handlePaneNav(msg):
		return nil, true

	case key.Matches(msg, m.keys.Quit):
		applog.Debug("tab: F10 quit requested")
		dlg := NewDialog("Quit CQOps", "Exit the application?",
			Option{Label: "Quit", Value: "quit"},
			Option{Label: "Cancel", Value: "cancel"},
		)
		applog.Debug("App: quit dialog shown")
		m.confirm = &dlg
		m.screen = screenQSO
		return nil, true

	case key.Matches(msg, m.keys.Help):
		// Defer help overlay until initialisation is complete (first tick
		// has dispatched all startup commands). Pressing ? during early
		// startup must not interrupt or delay initialisation.
		if m.tickCount < 1 {
			return nil, true
		}
		m.help.ShowAll = !m.help.ShowAll
		// When dismissing help, force-start services that may have been
		// blocked during initialisation (observed: DXC skips connection
		// when ? is pressed before internet is confirmed).
		if !m.help.ShowAll && m.inetOnline && m.tickCount < 60 {
			cmd := m.forceStartServices()
			return cmd, true
		}
		// When toggling help on, dismiss any open dialog.
		if m.help.ShowAll {
			m.confirm = nil
			m.spotDialog = nil
		}
		return nil, true

	case key.Matches(msg, m.keys.QSOForm):
		applog.Debug("tab: F1 QSO")
		if m.screen == screenQSO {
			m.focusField(fieldCall)
		} else {
			m.screen = screenQSO
		}
		return nil, true

	case key.Matches(msg, m.keys.Partner):
		call := qso.NormalizeCall(m.fields[fieldCall].Value())

		// Cycle: Partner → Image → Partner (when photo available).
		if m.screen == screenImage {
			m.screen = screenPartner
			m.photo.lastErr = nil
			m.photo.lastURL = ""
			m.photo.viewerLastW = 0
			m.photo.viewerLastH = 0
			// Reset photo dimension tracking so handlePartnerUpdate
			// re-applies SetSize with correct inline dimensions.
			m.photo.partnerPicW = 0
			m.photo.partnerPicH = 0
			m.photo.partnerPicLastW = 0
			m.photo.partnerPicLastH = 0
			m.photo.partnerPicNeedSize = true
			m.invalidatePartnerMapCache()
			return nil, true
		}
		if m.screen == screenPartner && m.lookup.partnerData != nil && m.lookup.partnerData.ImageURL != "" {
			applog.Debug("F2: opening image view", "url", m.lookup.partnerData.ImageURL)
			m.screen = screenImage
			m.photo.lastURL = m.lookup.partnerData.ImageURL
			m.photo.partnerPicURL = ""
			m.photo.partnerPicNeedLoad = false
			w := m.width
			h := m.height - 3 // full content area (ContentH = TerminalH - 3)
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			m.photo.viewerLastW = w
			m.photo.viewerLastH = h
			return tea.Batch(
				m.photo.viewer.SetSize(w, h),
				m.photo.viewer.SetURL(m.lookup.partnerData.ImageURL),
			), true
		}

		if call == "" {
			m.toasts.Warn("No callsign entered")
			applog.Debug("F2 Partner: no callsign")
			return nil, true
		}
		// Validate before committing.
		if !qso.IsValidCall(call) {
			m.toasts.Warn("Not a valid callsign")
			return nil, true
		}
		applog.Debug("tab: F2 Partner Details")
		m.commitCall()
		m.scpMatches = nil
		m.scpCacheKey = ""
		m.dxccAutoFill()
		band := strings.TrimSpace(m.fields[fieldBand].Value())
		mode := qso.NormalizeRigMode(m.fields[fieldMode].Value())

		callChanged := m.lookup.partnerData == nil || !strings.EqualFold(m.lookup.partnerData.Callsign, call)
		wlCallChanged := m.lookup.wlLookupCall == "" || !strings.EqualFold(m.lookup.wlLookupCall, call)
		bandChanged := band != m.lookup.wlLastBand
		modeChanged := mode != m.lookup.wlLastMode

		if callChanged {
			m.lookup.partnerData = nil
		}
		// Only invalidate WL data when the call, band, or mode actually
		// changed from the last WL lookup — not when QRZ partner data
		// happens to be nil.
		if wlCallChanged || bandChanged || modeChanged {
			m.lookup.wlPrivateData = nil
			m.lookup.wlLookupDone = false
		}
		m.screen = screenPartner
		m.invalidatePartnerMapCache()

		// Only trigger lookups when the call actually changed — avoid
		// redundant network calls on every tab switch.
		if callChanged || m.lookup.partnerData == nil {
			return m.lookupCallCmd(call), true
		}
		return nil, true

	case key.Matches(msg, m.keys.PSKReporter):
		if !m.inetOnline {
			m.toasts.Warn("PSK Reporter: no internet connection")
			return nil, true
		}
		applog.Debug("tab: F5 PSK Reporter")
		m.screen = screenPSKReporter
		return nil, true

	case key.Matches(msg, m.keys.Ref):
		if !m.isREFReady() {
			m.toasts.Warn("REF database not available — enable in General settings")
			return nil, true
		}
		applog.Debug("tab: F6 REF")
		m.screen = screenRef
		return nil, true

	case key.Matches(msg, m.keys.BPL):
		applog.Debug("tab: F7 BPL")
		m.screen = screenBPL
		return nil, true

	case key.Matches(msg, m.keys.Config):
		if m.screen == screenMainMenu {
			applog.Debug("tab: F9 close Config")
			m.screen = screenQSO
		} else {
			applog.Debug("tab: F9 Config")
			m.ui.mainMenu = NewMainMenu()
			m.ui.mainMenu.width = m.width
			m.ui.mainMenu.height = m.height
			m.screen = screenMainMenu
		}
		return nil, true

	case key.Matches(msg, m.keys.DXC):
		if !m.App.Config.Integrations.DXC.Enabled {
			m.toasts.Warn("DX Cluster not configured")
			return nil, true
		}
		if !m.dxc.online {
			m.toasts.Warn("DXC: not connected")
			return nil, true
		}
		applog.Debug("tab: F4 DXC")
		m.dxc.tableReady = false // force rebuild with fresh data
		// Set default continent filter from station config on first open.
		// Fall back to DXCC prefix lookup of own callsign if not configured.
		cont := m.App.Logbook.Station.Continent
		if cont == "" && m.App.DXCC != nil {
			if p := m.dxccLookup(m.App.Logbook.Station.Callsign); p != nil && p.Continent != "" {
				cont = p.Continent
			}
		}
		if m.dxc.contFilter == "" && cont != "" {
			m.dxc.contFilter = cont
			// Sync contIdx to match.
			choices := m.dxcContChoices()
			for i, c := range choices {
				if c == cont {
					m.dxc.contIdx = i
					break
				}
			}
		}
		m.screen = screenDXC
		return nil, true

	case key.Matches(msg, m.keys.LogEditor):
		applog.Debug("tab: F8 Editor")
		m.initLogbookEditor()
		m.screen = screenLogbookEditor
		return nil, true

	case key.Matches(msg, m.keys.Logs):
		applog.Debug("tab: Ctrl+F9 Log Viewer")
		m.ui.logViewer = NewLogViewer(config.LogbookDisplayName(m.App.Logbook))
		m.ui.logViewer.width = m.width
		m.ui.logViewer.height = m.height
		m.screen = screenLogView
		return nil, true

	default:
		if !m.isSubmodelActive() {
			if key.Matches(msg, m.keys.Delete) {
				m.clearForm()
				return nil, true
			}
			if key.Matches(msg, m.keys.Lookup) {
				return m.commitAndLookup(), true
			}
			if key.Matches(msg, m.keys.CycleLogbook) {
				return m.cycleLogbook(), true
			}
			if key.Matches(msg, m.keys.CycleRig) {
				return m.cycleRig(), true
			}
			// Favorite slots: ctrl+shift+digit saves, ctrl+digit recalls.
			if cmd, handled := m.handleFavoriteKey(msg); handled {
				return cmd, true
			}
		}
	}
	return nil, false
}

// forceStartServices kicks off DXC and HTTP connections when they may have
// been missed during early startup (e.g. when help overlay was opened before
// internet was confirmed). Safe to call repeatedly — each service guards
// against duplicate connections internally.
func (m *Model) forceStartServices() tea.Cmd {
	var cmds []tea.Cmd
	if c := m.maybeDXC(); c != nil {
		cmds = append(cmds, c)
	}
	if c := m.maybeHTTP(); c != nil {
		cmds = append(cmds, c)
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// =============================================================================
// QSO form key bindings
// =============================================================================

// handleFormKey processes QSO form-specific key bindings when no sub-screen is active.
// Returns a command and true if the key was handled.
func (m *Model) handleFormKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	// drainPending returns any lookup command queued by onFieldExit (Tab/arrow
	// out of the call field), batched with the caller's existing command.
	drainPending := func(existing tea.Cmd) tea.Cmd {
		if c := m.lookup.pendingLookupCmd; c != nil {
			m.lookup.pendingLookupCmd = nil
			if existing != nil {
				return tea.Batch(existing, c)
			}
			return c
		}
		return existing
	}

	var persistCmd tea.Cmd

	// ── Rotor manual control (only when connected, QSO screen) ──
	if m.rotor.connected && m.screen == screenQSO {
		if cmd, handled := m.handleRotorKey(msg); handled {
			return cmd, true
		}
	}

	switch {
	case m.keepFocused:
		switch msg.String() {
		case "space", "enter":
			if m.keepSubFocus == 0 {
				m.keepComment = !m.keepComment
				persistCmd = m.persistKeepComment()
			} else {
				m.retainForm = !m.retainForm
			}
		case "tab":
			m.nextField()
		case "shift+tab":
			m.prevField()
		case "down":
			m.nextRowField()
		case "up":
			m.prevRowField()
		case "ctrl+t":
			if m.keepSubFocus == 0 {
				m.keepComment = !m.keepComment
				persistCmd = m.persistKeepComment()
			} else {
				m.retainForm = !m.retainForm
			}
		case "ctrl+k":
			m.keepComment = !m.keepComment
			persistCmd = m.persistKeepComment()
		case "ctrl+h":
			m.retainForm = !m.retainForm
		}
		return drainPending(persistCmd), true

	// Global QSO form toggles — work regardless of focus.
	case msg.String() == "ctrl+k":
		m.keepComment = !m.keepComment
		return m.persistKeepComment(), true
	case msg.String() == "ctrl+h":
		m.retainForm = !m.retainForm
		return nil, true

	// Tab jumps horizontally across columns; Down/Up walk vertically.
	case msg.String() == "tab":
		m.nextField()
		return drainPending(nil), true
	case msg.String() == "shift+tab":
		m.prevField()
		return drainPending(nil), true
	case msg.String() == "down":
		m.nextRowField()
		return drainPending(nil), true
	case msg.String() == "up":
		m.prevRowField()
		return drainPending(nil), true

	case key.Matches(msg, m.keys.Save):
		return m.saveQSO(), true

	case key.Matches(msg, m.keys.Spot):
		return m.openSpotDialog(), true

	case key.Matches(msg, m.keys.Enter):
		// Enter logs what's in the form. Dupe check + two-press confirmation
		// is handled inside saveQSO. Lookups (QRZ, Wavelog) are dispatched
		// automatically via onFieldExit → tick loop — no need to trigger them here.
		return m.saveQSO(), true

	case key.Matches(msg, m.keys.Delete):
		m.clearForm()
		return nil, true

	case key.Matches(msg, m.keys.Retain):
		m.keepComment = !m.keepComment
		return m.persistKeepComment(), true

	case msg.String() == "ctrl+c":
		m.cycleActiveContest()
		// Contest change may affect dupe detection and form fields,
		// but doesn't change today/recent QSOs in the dashboard.
		// Push fast so the active QSO flags recompute.
		if m.http.online {
			lastFastTick = 0
			m.pushDashboardFast()
		}
		return m.refreshQSOS(), true

	case msg.String() == "ctrl+o":
		m.cycleActiveOperator()
		// Operator change only affects the operator field — light push.
		if m.http.online {
			lastFastTick = 0
			m.pushDashboardFast()
		}
		return nil, true

	case msg.String() == "ctrl+p":
		m.fillFromDXCSpot()
		return nil, true

	case key.Matches(msg, m.keys.Partner):
		call := m.commitCall()
		if call == "" {
			raw := strings.TrimSpace(m.fields[fieldCall].Value())
			if raw != "" {
				m.toasts.Warn("Not a valid callsign")
			}
			return nil, true
		}
		m.scpMatches = nil
		m.scpCacheKey = ""
		m.dxccAutoFill()
		m.screen = screenPartner
		m.invalidatePartnerMapCache()
		// Only trigger lookups when the call changed.
		if m.lookup.partnerData == nil || !strings.EqualFold(m.lookup.partnerData.Callsign, call) {
			return m.lookupCallCmd(call), true
		}
		return nil, true

	case key.Matches(msg, m.keys.PSKReporter):
		if !m.inetOnline {
			m.toasts.Warn("PSK Reporter: no internet connection")
			return nil, true
		}
		applog.Debug("tab: F5 PSK Reporter")
		m.screen = screenPSKReporter
		return nil, true

	case key.Matches(msg, m.keys.Lookup):
		return m.commitAndLookup(), true

	case key.Matches(msg, m.keys.CycleUp):
		m.cycleFieldUp()
		return nil, true

	case key.Matches(msg, m.keys.CycleDown):
		m.cycleFieldDown()
		return nil, true

	case key.Matches(msg, m.keys.RigTuneUp):
		return m.tuneRigStep(+1), true

	case key.Matches(msg, m.keys.RigTuneDown):
		return m.tuneRigStep(-1), true

	default:
		m.updateFocused(msg)
	}

	// Re-trigger WL lookup when band or mode changes while partner data is already loaded.
	curBand := strings.TrimSpace(m.fields[fieldBand].Value())
	curMode := strings.TrimSpace(m.fields[fieldMode].Value())
	call := qso.NormalizeCall(m.fields[fieldCall].Value())
	if call != "" && (curBand != m.lookup.wlLastBand || curMode != m.lookup.wlLastMode) && m.lookup.wlPrivateData != nil {
		m.lookup.wlNeed = true
		m.lookup.wlCall = call
	}
	// Dispatch any lookup command queued by onFieldExit (Tab/arrow out of call field).
	if c := m.lookup.pendingLookupCmd; c != nil {
		m.lookup.pendingLookupCmd = nil
		return c, true
	}
	return nil, false
}

// handleRotorKey processes rotor control key bindings when the rotor is
// connected and the QSO screen is active.
//
//	Alt+,/.       → adjust azimuth ±5°
//	Alt+;/'       → adjust elevation ±5°
//	Alt+\         → point rotor to calculated path bearing
func (m *Model) handleRotorKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	const step = 5.0

	// Base off the current target (if moving) or the polled position,
	// rounded to nearest integer so targets stay clean.
	baseAz := math.Round(m.rotor.azimuth)
	baseEl := math.Round(m.rotor.elevation)
	if m.rotor.targetAz != 0 {
		baseAz = math.Round(m.rotor.targetAz)
	}
	if m.rotor.targetEl != 0 {
		baseEl = math.Round(m.rotor.targetEl)
	}

	switch msg.String() {
	case "alt+,":
		az := clampAz(baseAz - step)
		applog.Debug("rotor: left", "az", az, "key", msg.String())
		return m.rotorSetPositionCmd(az, baseEl), true

	case "alt+.":
		az := clampAz(baseAz + step)
		applog.Debug("rotor: right", "az", az, "key", msg.String())
		return m.rotorSetPositionCmd(az, baseEl), true

	case "alt+;":
		el := clampEl(baseEl + step)
		applog.Debug("rotor: up", "el", el, "key", msg.String())
		return m.rotorSetPositionCmd(baseAz, el), true

	case "alt+'":
		el := clampEl(baseEl - step)
		applog.Debug("rotor: down", "el", el, "key", msg.String())
		return m.rotorSetPositionCmd(baseAz, el), true

	case "alt+\\":
		ownGrid := formatLocator(m.effectiveGrid())
		partnerGrid := formatLocator(m.fields[fieldGrid].Value())
		bearing := gridBearingDeg(ownGrid, partnerGrid)
		if bearing < 0 {
			m.toasts.Warn("Rotator: no path — enter partner grid first")
			return nil, true
		}
		az := clampAz(math.Round(bearing))
		applog.Debug("rotor: path bearing", "az", az, "from", ownGrid, "to", partnerGrid, "key", msg.String())
		m.toasts.Info(fmt.Sprintf("Rotator: turning to %.0f\u00b0", bearing))
		return m.rotorSetPositionCmd(az, math.Round(m.rotor.elevation)), true

	case "alt+/":
		if m.rotor.client == nil {
			return nil, false
		}
		applog.Debug("rotor: stop", "key", msg.String())
		m.toasts.Info("Rotator: stopped")
		m.rotor.targetAz = 0
		m.rotor.targetEl = 0
		m.rc.status = ""
		client := m.rotor.client
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := client.Stop(ctx); err != nil {
				applog.Debug("rotor: stop failed", "error", err)
			}
			return nil
		}, true
	}
	return nil, false
}

// rotorSetPositionCmd returns a tea.Cmd that commands the rotor to turn.
func (m *Model) rotorSetPositionCmd(az, el float64) tea.Cmd {
	if m.rotor.client == nil {
		return nil
	}
	m.rotor.targetAz = math.Round(az)
	m.rotor.targetEl = math.Round(el)
	client := m.rotor.client
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := client.SetPosition(ctx, az, el); err != nil {
			applog.Debug("rotor: set position failed", "az", az, "el", el, "error", err)
		}
		return nil
	}
}

// paneScreens returns the ordered list of available screens for Ctrl+←/→
// navigation.  Screens that are not currently usable (e.g. DXC offline,
// Partner without callsign) are excluded so Ctrl+arrows only cycle through
// active panes.
func (m *Model) paneScreens() []screenKind {
	hasPartner := m.lookup.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""
	dxcOnline := m.App.Config.Integrations.DXC.Enabled && m.dxc.online

	var screens []screenKind
	screens = append(screens, screenQSO) // always
	if hasPartner || m.screen == screenPartner {
		screens = append(screens, screenPartner)
	}
	if dxcOnline || m.screen == screenDXC {
		screens = append(screens, screenDXC)
	}
	if m.inetOnline || m.screen == screenPSKReporter {
		screens = append(screens, screenPSKReporter)
	}
	if m.isREFReady() || m.screen == screenRef {
		screens = append(screens, screenRef)
	}
	screens = append(screens, screenBPL)           // always
	screens = append(screens, screenLogbookEditor) // always
	screens = append(screens, screenMainMenu)      // always
	return screens
}

// handlePaneNav cycles the active screen left/right with Ctrl+←/→.
// Only cycles through currently available screens.
func (m *Model) handlePaneNav(msg tea.KeyPressMsg) bool {
	k := msg.String()
	if k != "ctrl+left" && k != "ctrl+right" {
		return false
	}

	// Image is an overlay on Partner, not a pane. Navigate as Partner.
	screen := m.screen
	if screen == screenImage {
		screen = screenPartner
	}

	screens := m.paneScreens()
	if len(screens) == 0 {
		return false
	}
	idx := -1
	for i, s := range screens {
		if s == screen {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	if k == "ctrl+left" {
		idx--
	} else {
		idx++
	}
	if idx < 0 {
		idx = len(screens) - 1
	}
	if idx >= len(screens) {
		idx = 0
	}
	target := screens[idx]

	// Lazily initialise sub-screens that need it — same logic as the
	// dedicated F-key handlers.
	switch target {
	case screenLogbookEditor:
		if m.ui.logbookEditor == nil {
			m.initLogbookEditor()
		}
	case screenMainMenu:
		if m.ui.mainMenu == nil {
			m.ui.mainMenu = NewMainMenu()
			m.ui.mainMenu.width = m.width
			m.ui.mainMenu.height = m.height
		}
	}

	// When leaving the image screen via pane nav, clear photo state.
	if m.screen == screenImage && target != screenImage {
		m.photo.lastErr = nil
		m.photo.lastURL = ""
		m.photo.viewerLastW = 0
		m.photo.viewerLastH = 0
		m.photo.partnerPicW = 0
		m.photo.partnerPicH = 0
		m.photo.partnerPicLastW = 0
		m.photo.partnerPicLastH = 0
		m.photo.partnerPicNeedSize = true
		m.invalidatePartnerMapCache()
	}

	m.screen = target
	applog.Debug("pane: ctrl+arrow", "screen", m.screen)
	return true
}

// initLogbookEditor creates a fresh logbook editor with current config.
func (m *Model) initLogbookEditor() {
	wl := m.App.Logbook.Wavelog
	wlURL, wlKey, wlStationID := "", "", ""
	if wl != nil {
		wlURL, wlKey, wlStationID = wl.URL, wl.APIKey, wl.StationProfileID
	}
	wlLastID := int64(0)
	if m.App.Logbook.Wavelog != nil {
		wlLastID = m.App.Logbook.Wavelog.LastFetchedID
	}
	m.ui.logbookEditor = NewLogbookEditor(LogbookEditorConfig{
		DB:              m.App.DB,
		WLURL:           wlURL,
		WLKey:           wlKey,
		WLStationID:     wlStationID,
		WLLastFetchedID: wlLastID,
		StationOperator: m.activeOperatorCallsign(),
		StationGrid:     m.effectiveGrid(),
		StationCall:     m.App.Logbook.Station.Callsign,
	})
	m.ui.logbookEditor.width = m.width
	m.ui.logbookEditor.height = m.height
	if m.App.Logbook.ActiveContest != "" {
		ct := m.App.Config.Contests[m.App.Logbook.ActiveContest]
		m.ui.logbookEditor.SetContestID(m.App.Logbook.ActiveContest, config.ContestDisplayName(&ct), ct.ContestID, ct.Date)
	} else {
		m.ui.logbookEditor.SetContestID("", "", "", "")
	}
	m.ui.logbookEditor.loadPage()
}

func clampAz(a float64) float64 {
	for a < 0 {
		a += 360
	}
	for a >= 360 {
		a -= 360
	}
	return a
}

func clampEl(e float64) float64 {
	if e < -90 {
		return -90
	}
	if e > 90 {
		return 90
	}
	return e
}

// persistKeepComment syncs the retain-comment checkbox state to the
// in-memory config and returns a tea.Cmd that writes it to disk async.
// When retain is off, the retained comment is cleared so it doesn't
// survive across restarts.
func (m *Model) persistKeepComment() tea.Cmd {
	if m.App == nil || m.App.Config == nil {
		return nil
	}
	m.App.Config.State.RetainComment = m.keepComment
	if m.keepComment {
		m.App.Config.State.RetainedComment = m.fields[fieldComment].Value()
	} else {
		m.App.Config.State.RetainedComment = ""
	}
	cfgPath := m.App.ConfigPath
	cfg := m.App.Config
	return func() tea.Msg {
		if err := config.Save(cfgPath, cfg); err != nil {
			applog.Warn("Failed to save retain comment state", "error", err)
		}
		return nil
	}
}
