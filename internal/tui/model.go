package tui

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/dxc"
	"github.com/szporwolik/cqops/internal/psk"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/solar"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type field int

const (
	healthCheckTicks    = 600                     // ticks between health checks (10 min)
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
	"RST sent", "RST rcvd", "Name", "QTH", "Grid", "Country", "Power W", "Freq RX",
	"SOTA Ref", "POTA Ref", "WWFF Ref", "IOTA", "Comment",
}

type screenKind int

const (
	screenQSO screenKind = iota
	screenPartner
	screenImage
	screenPSKReporter
	screenMainMenu
	screenConfig
	screenCallbook
	screenIntegration
	screenChooser
	screenRigEdit
	screenLogView
	screenLogbookEditor
	screenNotifications
	screenDXC
)

type Model struct {
	App                *app.App
	screen             screenKind
	fields             [fieldCount]textinput.Model
	focus              field
	qsos               []qso.QSO
	toasts             *ToastQueue
	err                error
	width              int
	height             int
	quitting           bool
	rigConnected       bool
	rigFreq            float64
	rigMode            string
	rigPower           float64
	rigBlink           bool
	rigSkipTicks       int
	rigPolling         bool
	dateTimeAuto       bool
	tickCount          int
	inetOnline         bool
	wsjtxOnline        bool
	wsjtxTx            bool   // true when WSJT-X is transmitting (from StatusMessage)
	wsjtxTxMsg         string // last TX message from WSJT-X (e.g. "CQ SP9XXX JO90")
	wsjtxLastSeen      time.Time
	wsjtxStatus        string
	needRefresh        bool
	pendingADIFs       []string // queued WSJT-X ADIF records — processed on tick
	pendingStatus      statusPending
	adifMu             sync.Mutex
	chooser            *LogbookChooser
	rigChooser         *RigChooser
	configMenu         *GeneralMenu
	callbookMenu       *CallbookMenu
	integrationMenu    *IntegrationMenu
	notifMenu          *NotificationsMenu
	mainMenu           *MainMenu
	logViewer          *LogViewer
	logbookEditor      *LogbookEditor
	imageViewer        pictureurl.Model // terminal image viewer for partner photos
	lastImageErr       error            // dedup image error logging
	lastImageURL       string           // track photo URL to detect partner changes
	lastPartnerPicURL  string           // track inline photo URL on Partner page
	partnerPicViewer   pictureurl.Model // inline photo viewer for Partner page (wide screens)
	partnerPicNeedLoad bool             // set when photo URL changes; consumed by Update
	partnerPicW        int              // photo box content width (computed in View, used in Update)
	partnerPicH        int              // photo box content height
	partnerPicLastW    int              // last SetSize width sent to viewer
	partnerPicLastH    int              // last SetSize height sent to viewer
	mapView            *mapRenderer     // embedded world map renderer
	confirm            *DialogModel     // active confirmation dialog (quit, etc.)

	// PSK Reporter.

	pskLastFetch  time.Time
	pskLastCall   string
	pskFilterMins int
	pskBandFilter string // "" = all bands, or band name like "20m"
	pskModeFilter string // "" = all modes, or mode name like "FT8"
	pskSelected   int
	pskFetched    bool
	pskFetching   bool // true while async HTTP fetch is in flight
	pskCacheDir   string

	// PSK caches — avoids SQL + Lip Gloss work on every frame.
	pskSpots   []psk.Report
	pskSpotKey string
	pskView    string
	pskViewKey string

	// Solar data — hourly fetch from hamqsl.com.
	solarData       *solar.Data
	solarLastFetch  time.Time
	solarFetching   bool
	solarFailed     bool // true after all retries exhausted
	solarCacheDir   string
	cachedSolarView string
	cachedSolarSig  string

	// DX Cluster — telnet connection to dxspider.co.uk.
	dxcClient       *dxc.Client
	dxcOnline       bool
	dxcConnecting   bool
	dxcLastAttempt  time.Time
	dxcReconnectIdx int
	dxcLastPurge    time.Time
	dxcTable        table.Model
	dxcTableReady   bool
	dxcBuiltW       int
	dxcBuiltH       int
	dxcSpotCount    int
	dxcBandFilter   string // "" = all, band name = filter, "other" = unclassified
	dxcTimeFilter   int    // minutes, 0 = all
	dxcTimeIdx      int    // index into dxcTimeWindows
	dxcBandIdx      int    // index into dxcBandChoices
	dxcModeFilter   string // "" = all, mode name = filter
	dxcModeIdx      int    // index into mode choices

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
	dxcNeed        bool   // re-trigger DXC freq lookup after live spot arrives
	dxcCall        string // callsign for pending DXC lookup
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

	// Partner view cache — avoids rebuilding the entire partner page on every render.
	partnerViewCache    string
	partnerViewCacheSig string

	// Path line cache — avoids locator parsing every View().
	// pathCall/pathGrid are the committed values, updated only on field exit.
	cachedPathLine string
	cachedPathSig  string
	pathCall       string
	pathGrid       string

	// Form column style cache — avoids re-creating lipgloss styles every frame.
	cachedFormColW         int
	cachedFormColStyle     lipgloss.Style
	cachedFormCommentStyle lipgloss.Style

	// Partner logbook stats cache — recomputed on call/band/mode change or QSO save.
	cachedLogStats    store.LogbookStats
	cachedLogStatsSig string

	// Per-frame view caches — avoids rebuilding lipgloss output when inputs unchanged.
	cachedFormView string
	cachedFormSig  string
	cachedTabView  string
	cachedTabSig   string
	cachedHelpView string
	cachedHelpSig  string
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
type dxcSpotLookupMsg struct {
	call string
	freq float64 // 0 if not found
}

type dxcTuneResultMsg struct {
	call    string
	freqMHz float64
	mode    string
	err     error
}
type inetResultMsg bool
type statusPending struct {
	call, grid, mode, submode, report, txMessage string
	freq                                         uint64
	transmitting                                 bool
	hasData                                      bool
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
	transport := &imageTransport{base: http.DefaultTransport}
	m.imageViewer = pictureurl.NewWithConfig(pictureurl.Config{
		CacheLimit: 4,
		UserAgent:  "CQOps/1.0 (ham-radio-logger)",
		HTTPClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	})
	m.partnerPicViewer = pictureurl.NewWithConfig(pictureurl.Config{
		CacheLimit: 4,
		UserAgent:  "CQOps/1.0 (ham-radio-logger)",
		HTTPClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	})
	m.mapView = newMapRenderer()
	m.pskFilterMins = pskFilterSteps[0] // default 5 min
	if dir, err := config.CacheDir(); err == nil {
		m.pskCacheDir = dir
		m.solarCacheDir = dir
	}
	m.applyBeepOnError()
	return m
}

// applyBeepOnError wires the system beep to all ERROR-level log calls
// when BeepOnError is enabled in the notifications config.
func (m *Model) applyBeepOnError() {
	if m.App.Config.General.Notifications.BeepOnError {
		applog.SetBeepFunc(func() { beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration) })
	} else {
		applog.SetBeepFunc(nil)
	}
}

func (m *Model) Init() tea.Cmd {
	m.refreshFlrigClient()
	m.App.WSJTX.OnADIF = func(adif string) {
		m.adifMu.Lock()
		m.pendingADIFs = append(m.pendingADIFs, adif)
		// Persist to disk immediately so QSOs survive crashes.
		// Failures are silent — the in-memory queue is authoritative.
		m.savePendingADIFsLocked()
		m.adifMu.Unlock()
	}
	// Recover any ADIF records left on disk from a previous crash.
	if saved := loadPendingADIFs(); len(saved) > 0 {
		m.pendingADIFs = append(m.pendingADIFs, saved...)
		applog.Info("WSJT-X: recovered pending ADIF records from disk", "count", len(saved))
	}
	m.App.WSJTX.OnStatus = func(call, grid string, freq uint64, mode, submode, report, txMessage string, transmitting bool) {
		m.adifMu.Lock()
		m.pendingStatus = statusPending{call: call, grid: grid, freq: freq, mode: mode, submode: submode, report: report, txMessage: txMessage, transmitting: transmitting, hasData: true}
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
	return tea.Tick(1000*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
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

	// Inline partner photo viewer — process its messages globally so
	// loads complete regardless of which screen is active.
	if c := m.partnerPicViewer.Update(msg); c != nil {
		cmd = tea.Batch(cmd, c)
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
		cmd = tea.Batch(cmd, m.updateFilteredTable())
		if m.partnerPicNeedLoad {
			m.partnerPicNeedLoad = false
			w := m.partnerPicW
			h := m.partnerPicH
			if w < 25 {
				w = 40
			}
			if h < 4 {
				h = 15
			}
			cmd = tea.Batch(cmd, m.partnerPicViewer.SetSize(w, h),
				m.partnerPicViewer.SetURL(m.lastPartnerPicURL))
		}
		return m, cmd
	case wlResultMsg:
		m.fillWLData(r)
		return m, cmd
	case dxcSpotLookupMsg:
		m.fillDXCFreq(r)
		return m, cmd
	case dxcSpotsStoredMsg:
		m.handleDXCSpotsStored(r)
		return m, cmd
	case dxcTuneResultMsg:
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("Tune failed: %v", r.err))
		} else {
			msg := fmt.Sprintf("Rig tuned to %.5f MHz", r.freqMHz)
			if r.mode != "" {
				msg += " " + r.mode
			}
			m.toasts.Success(msg)
		}
		return m, cmd
	}

	// Deferred pending requests (QRZ lookup, WL lookup, QSO refresh) —
	// must run before screen-specific routing so they work regardless of
	// which screen is active.
	if pendingCmd, handled := m.handlePendingRequests(cmd); handled {
		return m, pendingCmd
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
	case screenPSKReporter:
		return m.handlePSKReporterUpdate(msg, cmd)
	case screenImage:
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
			m.screen = screenPartner
			return m, cmd
		}
		// Detect new partner photo URL while viewing image.
		if m.partnerData != nil && m.partnerData.ImageURL != "" &&
			m.partnerData.ImageURL != m.lastImageURL {
			m.lastImageURL = m.partnerData.ImageURL
			w := m.width
			h := contentHeight(m.height)
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			cmd = tea.Batch(cmd, m.imageViewer.SetSize(w, h-1), m.imageViewer.SetURL(m.partnerData.ImageURL))
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
			h := contentHeight(m.height)
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
	case screenDXC:
		return m.handleDXCUpdate(msg, cmd)
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
	// Tab bar depends on partner data / call field — always fresh.
	m.cachedTabs = m.renderTabBar()
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
	if m.imageViewer.Err() != nil || content == "" {
		msg := ""
		if m.imageViewer.Err() != nil {
			err := m.imageViewer.Err()
			msg = err.Error()
			if strings.Contains(msg, "unexpected Content-Type") {
				msg = "QRZ photo not available — unsupported format"
			} else if strings.Contains(msg, "no such host") {
				msg = "Cannot reach image server"
			} else if strings.Contains(msg, "timeout") {
				msg = "Image download timed out"
			}
		}
		if msg == "" && m.lastImageURL != "" {
			msg = "Loading photo\u2026"
		} else if msg == "" && m.partnerData != nil {
			msg = "No photo for " + m.partnerData.Callsign
		}
		if msg == "" {
			msg = "No image"
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
// using Layout dimensions for proper sizing. Height is clamped to contentH
// so the bottom bars never shift when content changes (e.g. toggle in menus).
func (m *Model) buildBodyForScreen(l Layout) string {
	var body string
	switch m.screen {
	case screenQSO:
		body = m.buildQSOFormWithLayout(l)
	case screenPartner:
		if m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != "" {
			body = m.viewPartner()
		}
	case screenImage:
		body = m.viewImage(l)
	case screenPSKReporter:
		body = m.viewPSKReporter()
	case screenMainMenu:
		body = m.mainMenu.View().Content
	case screenConfig:
		body = m.configMenu.View().Content
	case screenCallbook:
		body = m.callbookMenu.View().Content
	case screenIntegration:
		body = m.integrationMenu.View().Content
	case screenChooser:
		body = m.chooser.View().Content
	case screenRigEdit:
		body = m.rigChooser.View().Content
	case screenLogView:
		body = m.logViewer.View().Content
	case screenLogbookEditor:
		body = m.logbookEditor.View().Content
	case screenNotifications:
		body = m.notifMenu.View().Content
	case screenDXC:
		body = m.dxcView()
	}
	if body == "" {
		return ""
	}
	// Clamp to contentH so toggling a menu item never shifts the bottom bars.
	return lipgloss.NewStyle().MaxHeight(l.ContentH).Render(body)
}

// buildQSOFormWithLayout renders the QSO form, short path info, and recent
// QSOs using layout-derived dimensions. The form border matches content width
// and is left-aligned; path row and table fill the full width.
func (m *Model) buildQSOFormWithLayout(l Layout) string {
	w := l.TerminalW
	formW := w - 4 // max available content width inside border
	if formW < 20 {
		formW = w - 2
	}
	// Cap form width to match partner page max (partnerMapMaxW).
	// Keeps F1/F2 cycling visually consistent on large monitors.
	if formW > partnerMapMaxW-4 {
		formW = partnerMapMaxW - 4
	}

	formContent := m.viewForm(formW)
	// Border uses the same max width as partner page for visual consistency
	// when cycling F1/F2 on large monitors.
	boxW := lipgloss.Width(formContent) + 4
	if boxW < partnerMapMaxW && formW >= partnerMapMaxW-4 {
		boxW = partnerMapMaxW
	}
	formBox := drawBorderedBox(formContent, boxW)

	// Solar panel — right-side column on wide screens (≥166 cols),
	// gated by the General → Solar at QSO pane config option.
	var formRow string
	if w >= 166 && m.App.Config.General.SolarAtQSOPane {
		solarW := w - 2 - boxW - 0 + 2 // no gap, 2px wider panel
		if solarW < 32 {
			solarW = 32
		}
		solarPanel := m.renderSolarPanel(solarW)
		if solarPanel != "" {
			leftH := lipgloss.Height(formBox)
			solarPanel = lipgloss.Place(
				lipgloss.Width(solarPanel),
				leftH,
				lipgloss.Top, lipgloss.Left,
				solarPanel,
			)
			formRow = lipgloss.JoinHorizontal(lipgloss.Top, formBox, solarPanel)
		}
	}
	if formRow == "" {
		formRow = formBox
	}

	profileLine := m.formPathRow(w - 2)
	profileH := 0
	if profileLine != "" {
		profileH = 1
	}

	formH := lipgloss.Height(formRow)
	tableH := l.ContentH - profileH - formH
	if tableH < 5 {
		tableH = 5
	}

	tableW := w - 2
	if tableH < 3 {
		tableH = 3
	}
	m.recentQSOs.SetSize(tableW, tableH)

	var parts []string
	if profileLine != "" {
		parts = append(parts, profileLine)
	}
	parts = append(parts, formRow)
	parts = append(parts, m.recentQSOs.View())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// formPartnerData builds a CallData from the current QSO form fields.

// imageTransport wraps an http.RoundTripper and strips non-image Content-Type
// headers from responses. Some image hosts (e.g. older QRZ CDN) serve PNG files
// as application/octet-stream, which pictureurl rejects. Stripping the header
// lets Go's image decoder identify the format from magic bytes instead.
type imageTransport struct {
	base http.RoundTripper
}

func (t *imageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "image/") {
		resp.Header.Del("Content-Type")
	}
	return resp, nil
}

// pendingADIFPath returns the path to the pending ADIF backup file.
func pendingADIFPath() string {
	dir, err := config.DataDir()
	if err != nil {
		return ""
	}
	return dir + "/pending_adifs.jsonl"
}

// savePendingADIFsLocked writes the current ADIF queue to a backup file.
// Must be called with adifMu held.
func (m *Model) savePendingADIFsLocked() {
	path := pendingADIFPath()
	if path == "" {
		return
	}
	if len(m.pendingADIFs) == 0 {
		os.Remove(path)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	for _, adif := range m.pendingADIFs {
		// Escape newlines so each ADIF is exactly one line.
		line := strings.ReplaceAll(adif, "\n", "\\n")
		fmt.Fprintln(f, line)
	}
}

// loadPendingADIFs reads the backup file and returns any saved ADIF records.
func loadPendingADIFs() []string {
	path := pendingADIFPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	os.Remove(path) // consumed
	var adifs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Unescape newlines.
		adifs = append(adifs, strings.ReplaceAll(line, "\\n", "\n"))
	}
	return adifs
}
