package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type field int

const (
	rigPollInterval     = 15                      // ticks between flrig polls
	healthCheckTicks    = 300                     // ticks between health checks
	flrigStatusTimeout  = 1500 * time.Millisecond // context timeout for flrig status
	flrigDefaultTimeout = 1000                    // default flrig HTTP timeout (ms)
)

const (
	fieldDate field = iota
	fieldTime
	fieldCall
	fieldFreq
	fieldBand
	fieldMode
	fieldSubmode
	fieldRSTSent
	fieldRSTRcvd
	fieldName
	fieldQTH
	fieldGrid
	fieldCountry
	fieldTXPower
	fieldFreqRx
	fieldSOTA
	fieldPOTA
	fieldWWFF
	fieldIOTA
	fieldComment
	fieldCount // sentinel: must be last; equals number of fields above
)

var fieldNames = []string{
	"Date UTC", "Time UTC", "Call", "Frequency", "Band", "Mode", "Submode",
	"RST sent", "RST rcvd", "Name", "QTH", "Grid", "DXCC", "Power W", "Freq RX",
	"SOTA Ref", "POTA Ref", "WWFF Ref", "IOTA", "Comment",
}

type screenKind int

const (
	screenQSO screenKind = iota
	screenPartner
	screenMainMenu
	screenConfig
	screenCallbook
	screenIntegration
	screenChooser
	screenRigEdit
	screenLogView
	screenLogbookEditor
)

type Model struct {
	App             *app.App
	screen          screenKind
	fields          [fieldCount]textinput.Model
	focus           field
	qsos            []qso.QSO
	toasts          *ToastQueue
	err             error
	width           int
	height          int
	quitting        bool
	rigConnected    bool
	rigFreq         float64
	rigMode         string
	rigPower        float64
	rigBlink        bool
	rigSkipTicks    int
	rigPolling      bool
	dateTimeAuto    bool
	tickCount       int
	inetOnline      bool
	wsjtxOnline     bool
	wsjtxStatus     string
	needRefresh     bool
	pendingADIF     string
	pendingStatus   statusPending
	adifMu          sync.Mutex
	chooser         *LogbookChooser
	rigChooser      *RigChooser
	configMenu      *GeneralMenu
	callbookMenu    *CallbookMenu
	integrationMenu *IntegrationMenu
	mainMenu        *MainMenu
	logViewer       *LogViewer
	logbookEditor   *LogbookEditor
	confirm         *DialogModel // active confirmation dialog (quit, etc.)
	partnerData     *qrz.CallData
	wlPrivateData   *wavelog.PrivateLookupResult // Wavelog callsign lookup
	wlLookupDone    bool                         // true when any WL lookup result received
	wlLastBand      string                       // band used in last WL query
	wlLastMode      string                       // mode used in last WL query
	flrigClient     FlrigClient
	qrzNeed         bool
	qrzCall         string
	qrzLastLook     time.Time
	qrzLastCall     string // last looked-up callsign
	wlNeed          bool   // re-trigger WL lookup after band/mode change
	wlCall          string // callsign for pending WL lookup
	wlLastLook      time.Time
	wlLastCall      string // last looked-up callsign
	retainComment   bool
	retainFocused   bool // true when the Retain checkbox has focus (instead of a text field)
	wlOnline        bool
	wlStationName   string // e.g. "JO30oo / DJ7NT"
	wlStationLabel  string // e.g. "Debug location"

	qrzOnline bool
	keys            KeyMap
	help            help.Model
	recentQSOs      *RecentQSOs // read-only Recent QSOs view

	// Partner/map rendering cache — avoids expensive ASCII map generation on every View().
	partnerMapCache    string
	partnerMapCacheSig string
}

type tickMsg time.Time
type qrzResultMsg struct {
	Call string
	Data *qrz.CallData
	Err  error
}

type qrzStatusMsg struct {
	online bool
}
type wlResultMsg struct {
	Call string
	Data *wavelog.PrivateLookupResult
	Err  error
}
type inetResultMsg bool
type statusPending struct {
	call, grid, mode, submode, report string
	freq                              uint64
	hasData                           bool
}

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	m := &Model{App: a, qsos: initialQSOS, toasts: NewToastQueue(), dateTimeAuto: true, width: 80, height: 24}
	now := time.Now().UTC()
	for i := field(0); i < fieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 40
		switch i {
		case fieldCall:
			ti.Focus()
		case fieldBand:
			ti.CharLimit = 8
		case fieldFreq, fieldFreqRx:
			ti.CharLimit = 16
		case fieldMode:
			ti.CharLimit = 12
		case fieldSubmode:
			ti.CharLimit = 16
		case fieldDate:
			ti.CharLimit = 10
			ti.SetValue(now.Format("2006-01-02"))
		case fieldTime:
			ti.CharLimit = 8
			ti.SetValue(now.Format("15:04:05"))
		case fieldGrid:
			ti.CharLimit = 8
		case fieldCountry:
			ti.CharLimit = 20
		case fieldName:
			ti.CharLimit = 30
		case fieldQTH:
			ti.CharLimit = 30
		case fieldTXPower:
			ti.CharLimit = 8
		case fieldComment:
			ti.CharLimit = 60
		case fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA:
			ti.CharLimit = 20
		}
		m.fields[i] = ti
	}



	m.focus = fieldCall
	m.keys = DefaultKeyMap()
	m.help = help.New()

	// Ensure textinput fields use Surface background (panel color, not app bg)
	for i := field(0); i < fieldCount; i++ {
		s := m.fields[i].Styles()
		s.Focused.Text = s.Focused.Text.Background(P.Surface)
		s.Focused.Placeholder = s.Focused.Placeholder.Background(P.Surface)
		s.Focused.Prompt = s.Focused.Prompt.Background(P.Surface)
		s.Blurred.Text = s.Blurred.Text.Background(P.Surface)
		s.Blurred.Placeholder = s.Blurred.Placeholder.Background(P.Surface)
		s.Blurred.Prompt = s.Blurred.Prompt.Background(P.Surface)
		m.fields[i].SetStyles(s)
	}
	m.recentQSOs = NewRecentQSOs(initialQSOS)
	return m
}

func (m *Model) Init() tea.Cmd {
	m.refreshFlrigClient()
	m.App.WSJTX.OnADIF = func(adif string) {
		m.adifMu.Lock()
		m.pendingADIF = adif
		m.adifMu.Unlock()
	}
	m.App.WSJTX.OnStatus = func(call, grid string, freq uint64, mode, submode, report string) {
		m.adifMu.Lock()
		m.pendingStatus = statusPending{call: call, grid: grid, freq: freq, mode: mode, submode: submode, report: report, hasData: true}
		m.adifMu.Unlock()
	}
	if m.App.Config.WSJTX.Enabled {
		applog.Info("WSJT-X: callbacks registered, restarting listener")
		m.App.MaybeRestartWSJTX()
	} else {
		applog.Debug("wsjt-x: disabled")
	}
	return tea.Batch(tickCmd(), checkInetCmd())
}
func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// hideAllSubmodels returns to the QSO form screen.
func (m *Model) hideAllSubmodels() {
	m.screen = screenQSO
}

// isSubmodelActive returns true when any sub-screen is visible.
func (m *Model) isSubmodelActive() bool {
	return m.screen != screenQSO
}

// saveConfig persists the app configuration and shows a toast.
func (m *Model) saveConfig(msg string) {
	if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
		m.toasts.Error("Settings save failed: " + err.Error())
	} else {
		if msg != "" {
			m.toasts.Success(msg)
		} else {
			m.toasts.Success("Settings saved")
		}
		applog.Info("Settings saved")
	}
}

// partnerMapCacheKey computes a cache key from all inputs that affect
// the partner/map rendered output. When this key matches, the cached
// map output can be reused without expensive ASCII generation.

// invalidatePartnerMapCache clears the partner map cache.

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// WindowSizeMsg — store dimensions first; invalidate map cache on resize
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.invalidatePartnerMapCache()
	}

	// Active confirmation dialog — highest priority, blocks everything else
	if m.confirm != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			updated, _ := m.confirm.Update(msg)
			*m.confirm = updated.(DialogModel)
			if m.confirm.Done() {
				if m.confirm.Result.Confirmed && m.confirm.Result.Value == "quit" {
					return m, tea.Quit
				}
				m.confirm = nil
			}
		}
		return m, cmd
	}

	// Tick processing
	if _, ok := msg.(tickMsg); ok {
		cmd = m.handleTick(cmd)
	}

	// Async result messages (internet, Wavelog, flrig)
	if m.handleAsyncMessages(msg) {
		if _, ok := msg.(flrigResultMsg); ok {
			return m, cmd
		}
	}

	// Global function keys (F1-F10, etc.)
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if handledCmd, handled := m.handleGlobalKeys(keyMsg); handled {
			if handledCmd != nil {
				cmd = tea.Batch(cmd, handledCmd)
			}
			return m, cmd
		}
	}

	// Screen-specific routing
	switch m.screen {
	case screenChooser:
		return m.handleChooserUpdate(msg, cmd)
	case screenRigEdit:
		return m.handleRigEditUpdate(msg, cmd)
	case screenConfig:
		return m.handleConfigUpdate(msg, cmd)
	case screenCallbook:
		return m.handleCallbookUpdate(msg, cmd)
	case screenIntegration:
		return m.handleIntegrationUpdate(msg, cmd)
	case screenMainMenu:
		return m.handleMainMenuUpdate(msg, cmd)
	case screenPartner:
		return m.handlePartnerUpdate(msg, cmd)
	case screenLogbookEditor:
		return m.handleLogbookEditorUpdate(msg, cmd)
	case screenLogView:
		return m.handleLogViewUpdate(msg, cmd)
	}

	// QSO form result messages (QRZ, Wavelog lookups)
	switch r := msg.(type) {
	case qrzResultMsg:
		m.fillQRZData(r)
		return m, cmd
	case wlResultMsg:
		m.fillWLData(r)
		return m, cmd
	}

	// QSO form key handling
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if formCmd, handled := m.handleFormKey(keyMsg); handled {
			if formCmd != nil {
				cmd = tea.Batch(cmd, formCmd)
			}
			return m, cmd
		}
	}

	// Deferred pending requests
	if pendingCmd, handled := m.handlePendingRequests(cmd); handled {
		return m, pendingCmd
	}

	return m, cmd
}

func (m *Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	if m.err != nil {
		return tea.NewView(errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)))
	}

	// Measure all fixed zones and calculate content area dimensions
	layout := MeasureLayout(m)

	if layout.TerminalW < 75 || layout.TerminalH < 24 {
		msg := fmt.Sprintf("\n  CQOps — Terminal too small: %dx%d (min 75x24)\n\n  Press F10 and then Enter to quit",
			layout.TerminalW, layout.TerminalH)
		return tea.NewView(lipgloss.NewStyle().Foreground(P.Error).Render(msg))
	}

	// 1. Status bar
	statusBar := m.renderStatusBar()

	// 2. Profile/context line — right-aligned above tabs for readability
	profileLine := m.renderProfileBar()

	// 3. Tab bar
	tabBar := m.renderTabBar()

	var body string
	body = m.buildBodyForScreen(layout)
	if body == "" {
		body = DimStyle.Render("\u2014")
	}

	helpBar := m.renderHelpBar()

	// Fill body area with surface background to content height
	body = S.ContentBase.
		Height(layout.ContentH).
		Width(layout.TerminalW).
		Render(body)

	// Build the main view without toasts or dialogs (those are composited on top).
	// Only include non-empty rows to avoid blank lines from empty zones.
	var mainParts []string
	addRow := func(s string) {
		if s != "" {
			mainParts = append(mainParts, s)
		}
	}
	addRow(statusBar)
	addRow(profileLine)
	addRow(tabBar)
	addRow(body)
	addRow(helpBar)
	mainView := lipgloss.JoinVertical(lipgloss.Left, mainParts...)

	// Composite confirm dialog as a centered overlay if active
	if m.confirm != nil {
		mainView = RenderDialogOverlay(mainView, *m.confirm, layout.TerminalW, layout.TerminalH)
	}

	// Composite toasts as a floating overlay in the bottom-right corner.
	// Clip mainView to terminal height first so the compositor canvas
	// matches the visible terminal exactly.
	mainView = lipgloss.NewStyle().MaxHeight(layout.TerminalH).Render(mainView)
	finalView := RenderToastOverlay(mainView, m.toasts.Active(), layout.TerminalW, layout.TerminalH)

	v := tea.NewView(finalView)
	v.AltScreen = true
	v.WindowTitle = m.windowTitle()
	v.BackgroundColor = P.Background
	return v
}

// buildBodyForScreen returns the content string for the active screen,
// using Layout dimensions for proper sizing.
func (m *Model) buildBodyForScreen(l Layout) string {
	switch m.screen {
	case screenQSO:
		return m.buildQSOFormWithLayout(l)
	case screenPartner:
		if m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != "" {
			return m.viewPartner()
		}
	case screenMainMenu:
		return m.mainMenu.View().Content
	case screenConfig:
		return m.configMenu.View().Content
	case screenCallbook:
		return m.callbookMenu.View().Content
	case screenIntegration:
		return m.integrationMenu.View().Content
	case screenChooser:
		return m.chooser.View().Content
	case screenRigEdit:
		return m.rigChooser.View().Content
	case screenLogView:
		return m.logViewer.View().Content
	case screenLogbookEditor:
		return m.logbookEditor.View().Content
	}
	return ""
}

// buildQSOFormWithLayout renders the QSO form, short path info, and recent
// QSOs using layout-derived dimensions. The short path gets its own bordered
// box between the form and the table for visual separation.
func (m *Model) buildQSOFormWithLayout(l Layout) string {
	w := l.TerminalW // full width
	borderW := w - 2 // content width inside borders
	formW := borderW - 2 // 1-char padding each side
	if formW < 20 {
		formW = borderW
	}

	// QSO form with manual border — no │ leak
	form := m.viewForm(formW)
	formBox := drawBorderedBox(strings.TrimRight(form, "\n"), borderW, w)

	// Path row with manual border
	pathContent := m.formPathRow(formW)
	pathBox := drawBorderedBox(pathContent, borderW, w)

	formRenderedH := lipgloss.Height(formBox)
	pathRenderedH := lipgloss.Height(pathBox)
	recentH := l.ContentH - formRenderedH - pathRenderedH
	if recentH < 5 {
		recentH = 5
	}

	tableW := w - 2
	tableH := recentH - 2
	if tableH < 3 {
		tableH = 3
	}
	m.recentQSOs.SetSize(tableW, tableH)

	recentBox := drawBorderedBox(m.recentQSOs.View(), borderW, w)

	return formBox + "\n" + pathBox + "\n" + recentBox
}

// formPartnerData builds a CallData from the current QSO form fields.
