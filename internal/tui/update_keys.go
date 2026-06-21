package tui

import (
	"strings"

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

	case key.Matches(msg, m.keys.QSOForm):
		applog.Debug("tab: F1 QSO")
		if m.screen == screenQSO {
			m.focusField(fieldCall)
		} else {
			m.screen = screenQSO
		}
		return nil, true

	case key.Matches(msg, m.keys.Partner):
		call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))

		// Cycle: Partner → Image → Partner (when photo available).
		if m.screen == screenImage {
			m.screen = screenPartner
			m.photo.lastErr = nil
			m.photo.lastURL = ""
			return nil, true
		}
		if m.screen == screenPartner && m.lookup.partnerData != nil && m.lookup.partnerData.ImageURL != "" {
			applog.Debug("F2: opening image view", "url", m.lookup.partnerData.ImageURL)
			m.screen = screenImage
			m.photo.lastURL = m.lookup.partnerData.ImageURL
			w := m.width
			h := m.height - 4 // header/tab/help overhead
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			h-- // bottom hint row
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
		mode := strings.TrimSpace(m.fields[fieldMode].Value())

		callChanged := m.lookup.partnerData == nil || !strings.EqualFold(m.lookup.partnerData.Callsign, call)
		bandChanged := band != m.lookup.wlLastBand
		modeChanged := mode != m.lookup.wlLastMode

		if callChanged {
			m.lookup.partnerData = nil
		}
		if callChanged || bandChanged || modeChanged {
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

	// case key.Matches(msg, m.keys.CON):
	// 	applog.Debug("tab: F3 CON")
	// 	m.screen = screenCON
	// 	return nil, true

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
		if !m.App.Config.Integrations.DXC.Enabled || !m.dxc.online {
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
		wl := m.App.Logbook.Wavelog
		wlURL, wlKey, wlStationID := "", "", ""
		if wl != nil {
			wlURL, wlKey, wlStationID = wl.URL, wl.APIKey, wl.StationProfileID
		}
		wlLastID := int64(0)
		if m.App.Logbook.Wavelog != nil {
			wlLastID = m.App.Logbook.Wavelog.LastFetchedID
		}
		m.ui.logbookEditor = NewLogbookEditor(m.App.DB, wlURL, wlKey, wlStationID, wlLastID, m.App.Logbook.Station.Operator, m.App.Logbook.Station.Grid)
		m.ui.logbookEditor.width = m.width
		m.ui.logbookEditor.height = m.height
		// Apply active contest filter.
		if m.App.Logbook.ActiveContest != "" {
			ct := m.App.Config.Contests[m.App.Logbook.ActiveContest]
			m.ui.logbookEditor.SetContestID(m.App.Logbook.ActiveContest, config.ContestDisplayName(&ct), ct.ContestID)
		} else {
			m.ui.logbookEditor.SetContestID("", "", "")
		}
		m.ui.logbookEditor.loadPage()
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
	switch {
	case m.retainFocused:
		switch msg.String() {
		case "space", "enter":
			m.retainComment = !m.retainComment
			persistCmd = m.persistRetainComment()
		case "tab":
			m.nextField()
		case "shift+tab":
			m.prevField()
		case "down":
			m.nextRowField()
		case "up":
			m.prevRowField()
		case "ctrl+t":
			m.retainComment = !m.retainComment
			persistCmd = m.persistRetainComment()
		}
		return drainPending(persistCmd), true

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
		m.retainComment = !m.retainComment
		return m.persistRetainComment(), true

	case msg.String() == "ctrl+c":
		m.cycleActiveContest()
		return m.refreshQSOS(), true

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

	default:
		m.updateFocused(msg)
	}

	// Re-trigger WL lookup when band or mode changes while partner data is already loaded.
	curBand := strings.TrimSpace(m.fields[fieldBand].Value())
	curMode := strings.TrimSpace(m.fields[fieldMode].Value())
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
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

// persistRetainComment syncs the retain-comment checkbox state to the
// in-memory config and returns a tea.Cmd that writes it to disk async.
func (m *Model) persistRetainComment() tea.Cmd {
	if m.App == nil || m.App.Config == nil {
		return nil
	}
	m.App.Config.State.RetainComment = m.retainComment
	m.App.Config.State.RetainedComment = m.fields[fieldComment].Value()
	cfgPath := m.App.ConfigPath
	cfg := m.App.Config
	return func() tea.Msg {
		if err := config.Save(cfgPath, cfg); err != nil {
			applog.Warn("Failed to save retain comment state", "error", err)
		}
		return nil
	}
}
