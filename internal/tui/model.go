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
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
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
	screenImage
	screenMainMenu
	screenConfig
	screenCallbook
	screenIntegration
	screenChooser
	screenRigEdit
	screenLogView
	screenLogbookEditor
	screenNotifications
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
	notifMenu       *NotificationsMenu
	mainMenu        *MainMenu
	logViewer       *LogViewer
	logbookEditor   *LogbookEditor
	imageViewer     pictureurl.Model // terminal image viewer for partner photos
	lastImageErr    error            // dedup image error logging
	confirm         *DialogModel     // active confirmation dialog (quit, etc.)

	// Layout cache — avoids redundant MeasureLayout() calls when terminal size
	// and screen haven't changed between frames.
	lastLayout   Layout
	lastLayoutW  int
	lastLayoutH  int
	lastLayoutSc screenKind

	// Bar caches — avoids rebuilding status/profile/tabs/help on every frame.
	// Status bar has a 1-second TTL because it contains the UTC clock.
	cachedStatus    string
	cachedStatusSec int
	cachedProfile   string
	cachedTabs      string
	cachedHelp      string
	cachedBarSc     screenKind
	cachedBarW      int

	partnerData    *qrz.CallData
	wlPrivateData  *wavelog.PrivateLookupResult // Wavelog callsign lookup
	wlLookupDone   bool                         // true when any WL lookup result received
	wlLastBand     string                       // band used in last WL query
	wlLastMode     string                       // mode used in last WL query
	flrigClient    FlrigClient
	qrzNeed        bool
	qrzCall        string
	qrzLastLook    time.Time
	qrzLastCall    string // last looked-up callsign
	wlNeed         bool   // re-trigger WL lookup after band/mode change
	wlCall         string // callsign for pending WL lookup
	wlLastLook     time.Time
	wlLastCall     string // last looked-up callsign
	retainComment  bool
	retainFocused  bool // true when the Retain checkbox has focus (instead of a text field)
	wlOnline       bool
	wlForceCheck   bool   // force Wavelog check on next tick
	wlStationName  string // e.g. "JO30oo / DJ7NT"
	wlStationLabel string // e.g. "Debug location"

	qrzOnline  bool
	keys       KeyMap
	help       help.Model
	recentQSOs *RecentQSOs // read-only Recent QSOs view

	// Partner/map rendering cache — avoids expensive ASCII map generation on every View().
	partnerMapCache    string
	partnerMapCacheSig string

	// QSO count cache — avoids SQL queries during View().
	qsoCounts      store.QSOCounts
	qsoCountsValid bool

	// Path line cache — avoids locator parsing every View().
	cachedPathLine string
	cachedPathSig  string
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
		ti := newTextinput()
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

	m.recentQSOs = NewRecentQSOs(initialQSOS)
	m.imageViewer = pictureurl.NewWithConfig(pictureurl.Config{
		CacheLimit: 4,
	})
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
	return tea.Batch(tickCmd(), checkInetCmd(), m.imageViewer.Init())
}
func tickCmd() tea.Cmd {
	return tea.Tick(2000*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// WindowSizeMsg — store dimensions first; invalidate map cache on resize
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.invalidatePartnerMapCache()
		// Forward size to image viewer.
		if c := m.imageViewer.Update(msg); c != nil {
			cmd = tea.Batch(cmd, c)
		}
		// Update focused textinput width so scrolling stays correct.
		if m.screen == screenQSO && !m.retainFocused {
			if m.width > 60 {
				m.fields[m.focus].SetWidth(m.width/3 - 16)
			} else if m.width > 20 {
				m.fields[m.focus].SetWidth(m.width - 16)
			}
		}
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

	// Lookup result messages (QRZ, Wavelog) — must be processed before
	// screen-specific routing so they work regardless of which screen is
	// active (e.g. partner screen). Each screen's handler returns early
	// for unrecognised messages, which would silently drop these.
	switch r := msg.(type) {
	case qrzResultMsg:
		m.fillQRZData(r)
		return m, cmd
	case wlResultMsg:
		m.fillWLData(r)
		return m, cmd
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
	case screenImage:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
			m.screen = screenPartner
			return m, cmd
		}
		// Log image errors once and show toast.
		if err := m.imageViewer.Err(); err != nil && m.lastImageErr != err {
			m.lastImageErr = err
			applog.Warn("Image load failed", "error", err.Error())
			m.toasts.Warn("Photo unavailable — unsupported format")
		}
		if m.imageViewer.Err() == nil {
			m.lastImageErr = nil
		}
		// Reapply size on resize while viewing image.
		if _, ok := msg.(tea.WindowSizeMsg); ok {
			w := m.width
			h := m.height - 4
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			if c := m.imageViewer.SetSize(w, h); c != nil {
				cmd = tea.Batch(cmd, c)
			}
		}
		c := m.imageViewer.Update(msg)
		if c != nil {
			cmd = tea.Batch(cmd, c)
		}
		return m, cmd
	case screenLogbookEditor:
		return m.handleLogbookEditorUpdate(msg, cmd)
	case screenLogView:
		return m.handleLogViewUpdate(msg, cmd)
	case screenNotifications:
		return m.handleNotificationsUpdate(msg, cmd)
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
		return tea.NewView(ErrorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)))
	}

	// Measure all fixed zones and calculate content area dimensions.
	// Cache the layout when terminal size and screen haven't changed.
	var layout Layout
	if m.lastLayoutW == m.width && m.lastLayoutH == m.height && m.lastLayoutSc == m.screen {
		layout = m.lastLayout
	} else {
		layout = MeasureLayout(m)
		m.lastLayout = layout
		m.lastLayoutW = m.width
		m.lastLayoutH = m.height
		m.lastLayoutSc = m.screen
	}

	if layout.TerminalW < 75 || layout.TerminalH < 24 {
		msg := fmt.Sprintf("\n  CQOps — Terminal too small: %dx%d (min 75x24)\n\n  Press F10 and then Enter to quit",
			layout.TerminalW, layout.TerminalH)
		return tea.NewView(ErrorStyle.Render(msg))
	}

	// Render fixed bars — cache when screen and width haven't changed.
	// Status bar has a 1-second TTL because it contains the UTC clock.
	cacheBars := m.cachedBarW == m.width && m.cachedBarSc == m.screen
	if !cacheBars {
		m.cachedStatus = ""
	}
	if m.cachedStatus == "" || m.cachedStatusSec != time.Now().UTC().Second() {
		m.cachedStatus = m.renderStatusBar()
		m.cachedStatusSec = time.Now().UTC().Second()
	}
	if m.cachedProfile == "" || !cacheBars {
		m.cachedProfile = m.renderProfileBar()
	}
	if m.cachedTabs == "" || !cacheBars {
		m.cachedTabs = m.renderTabBar()
	}
	// Help bar has dynamic suffix (QSO counter, scroll info) — always fresh.
	m.cachedHelp = m.renderHelpBar()
	m.cachedBarW = m.width
	m.cachedBarSc = m.screen

	var mainParts []string
	addRow := func(s string) {
		if s != "" {
			mainParts = append(mainParts, s)
		}
	}
	addRow(m.cachedStatus)
	if m.cachedProfile != "" {
		addRow(m.cachedProfile)
	}
	addRow(m.cachedTabs)

	body := m.buildBodyForScreen(layout)
	if body == "" {
		body = DimStyle.Render("\u2014")
	}
	addRow(body)
	addRow(m.cachedHelp)
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
	return v
}

// viewImage renders the partner photo full-screen.
func (m *Model) viewImage(l Layout) string {
	content := m.imageViewer.View().Content
	if m.imageViewer.Err() != nil {
		err := m.imageViewer.Err()
		msg := err.Error()
		if strings.Contains(msg, "unexpected Content-Type") {
			msg = "QRZ photo not available — unsupported format"
		} else if strings.Contains(msg, "no such host") {
			msg = "Cannot reach image server"
		} else if strings.Contains(msg, "timeout") {
			msg = "Image download timed out"
		}
		content = lipgloss.NewStyle().
			Width(l.TerminalW).
			Height(l.ContentH).
			Align(lipgloss.Center, lipgloss.Center).
			Foreground(P.TextMuted).
			Render(msg)
	} else {
		content = lipgloss.NewStyle().
			Width(l.TerminalW).
			Height(l.ContentH).
			Render(content)
	}
	return content
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
	case screenImage:
		return m.viewImage(l)
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
	case screenNotifications:
		return m.notifMenu.View().Content
	}
	return ""
}

// buildQSOFormWithLayout renders the QSO form, short path info, and recent
// QSOs using layout-derived dimensions.
func (m *Model) buildQSOFormWithLayout(l Layout) string {
	w := l.TerminalW
	borderW := w - 2
	formW := borderW - 2
	if formW < 20 {
		formW = borderW
	}

	formBox := drawBorderedBox(m.viewForm(formW), w)
	pathBox := drawBorderedBox(m.formPathRow(formW), w)

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
	recentBox := drawBorderedBox(m.recentQSOs.View(), w)

	return lipgloss.JoinVertical(lipgloss.Left, formBox, pathBox, recentBox)
}

// formPartnerData builds a CallData from the current QSO form fields.
