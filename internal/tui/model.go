package tui

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture/pictureurl"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type field int

// gridSource tracks where the QSO form grid value originated.
type gridSource string

const (
	gridSourceNone   gridSource = ""
	gridSourceQRZ    gridSource = "QRZ.com"
	gridSourceManual gridSource = "manual"
	gridSourceSOTA   gridSource = "SOTA"
	gridSourcePOTA   gridSource = "POTA"
	gridSourceWWFF   gridSource = "WWFF"
	gridSourceIOTA   gridSource = "IOTA"
)

const (
	healthCheckTicks  = 60                      // ticks between health checks (1 min)
	rigStatusTimeout  = 1500 * time.Millisecond // context timeout for rig status
	rigDefaultTimeout = 1000                    // default rig poll timeout (ms)
)

const (
	fieldDate field = iota
	fieldTime
	fieldCall
	fieldRSTSent
	fieldRSTRcvd
	fieldFreq
	fieldBand
	fieldExchSent
	fieldExchRcvd
	fieldMode
	fieldSubmode
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
	fieldSIG
	fieldSIGInfo
	fieldComment
	fieldCount // sentinel: must be last; equals number of fields above
)

var fieldNames = []string{
	"Date UTC", "Time UTC", "Call", "RST sent", "RST rcvd", "Frequency", "Band",
	"Exch sent", "Exch rcvd",
	"Mode", "Submode", "Name", "QTH", "Grid", "Country", "Power W", "Freq RX",
	"SOTA Ref", "POTA Ref", "WWFF Ref", "IOTA", "SIG", "SIG Info", "Comment",
}

type screenKind int

const (
	screenQSO screenKind = iota
	screenPartner
	screenImage
	screenPSKReporter
	screenMainMenu
	screenConfig
	screenIntegration
	screenChooser
	screenRigEdit
	screenContest
	screenOperator
	screenLogView
	screenLogbookEditor
	screenNotifications
	screenDXC
	screenRef
	screenBPL
)

type Model struct {
	App            *app.App
	screen         screenKind
	fields         [fieldCount]textinput.Model
	focus          field
	qsos           []qso.QSO
	toasts         *ToastQueue
	err            error
	width          int
	height         int
	quitting       bool
	rig            rigState
	rotor          rotorState
	dateTimeAuto   bool
	tickCount      int
	inetOnline     bool
	versionChecked bool // true after first GitHub version check
	Offline        bool // when true, skip all network-dependent operations
	wsjtx          wsjtxState
	needRefresh    bool
	dupe           bool // true when call/band/mode match an existing QSO today
	dupeConfirmed  bool // true after first Enter on a dupe; second Enter proceeds
	adifQ          adifQueue
	ui             uiComponents
	photo          photoState
	mapView        *mapRenderer // embedded world map renderer
	confirm        *DialogModel // active confirmation dialog (quit, etc.)
	spotDialog     *SpotDialog  // active DX spot dialog

	// PSK Reporter.
	psk pskState

	// Solar data — hourly fetch from hamqsl.com.
	solar solarState

	// DX Cluster — telnet connection to dxspider.co.uk.
	dxc dxcState

	// REF — SOTA/POTA/WWFF reference lookup.
	ref refState

	// BPL — band plan display (F7).
	bpl bplState

	// lastDataCheck is the last time CTY.DAT / SCP files were checked for updates.
	lastDataCheck time.Time

	// SCP (Super Check Partial) auto-complete state.
	scpMatches  []string // current prefix matches for the call field
	scpCacheKey string   // the callsign prefix that produced scpMatches

	// Render cache — avoids redundant layout, style, and view computation.
	rc renderCache

	lookup       lookupState
	keepComment  bool // "Keep" checkbox — retains comment field content across QSOs
	keepFocused  bool // true when the Keep/Retain checkbox row has focus
	keepSubFocus int  // 0=Keep, 1=Retain — which checkbox in the row is active
	retainForm   bool // "Retain" checkbox — prevents form clearing after QSO save
	gridSource   gridSource

	keys       KeyMap
	help       help.Model
	recentQSOs *RecentQSOs // read-only Recent QSOs view
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

// logbookStatsMsg carries the async result of GetLogbookStats.
type logbookStatsMsg struct {
	stats store.LogbookStats
	sig   string
}

type dxcTuneResultMsg struct {
	call    string
	freqMHz float64
	mode    string
	verify  string // non-empty when actual frequency differs from requested
	err     error
}

// bplTuneResultMsg is sent after a BPL space-to-tune completes.
type bplTuneResultMsg struct {
	freqMHz float64
	mode    string
	verify  string // non-empty when actual frequency differs from requested
	err     error
}
type inetResultMsg bool

// versionCheckMsg carries the result of a GitHub release version check.
type versionCheckMsg struct {
	latest string // latest release tag, empty if check failed
}

type statusPending struct {
	call, grid, mode, submode, report, txMessage string
	freq                                         uint64
	transmitting                                 bool
	hasData                                      bool
}

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	// Apply config debug mode at startup (CLI --debug flag may have
	// already set it, but config value takes precedence).
	applog.SetDebugMode(a.Config.General.Debug)

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
			ti.CharLimit = 40
		case fieldSIG:
			ti.CharLimit = 30
		case fieldSIGInfo:
			ti.CharLimit = 40
		}
		m.fields[i] = ti
	}

	m.focus = fieldCall
	m.keys = DefaultKeyMap()
	m.help = help.New()
	// Brighten help overlay content: key in value color, desc in muted label color.
	m.help.Styles.FullKey = S.Input
	m.help.Styles.FullDesc = lipgloss.NewStyle().Foreground(P.TextMuted)
	m.help.Styles.FullSeparator = lipgloss.NewStyle().Foreground(P.TextMuted)

	m.recentQSOs = NewRecentQSOs(initialQSOS)
	transport := &imageTransport{base: http.DefaultTransport}
	m.photo.viewer = pictureurl.NewWithConfig(pictureurl.Config{
		CacheLimit: 4,
		UserAgent:  "CQOps/1.0 (ham-radio-logger)",
		HTTPClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	})
	m.photo.partnerPicViewer = pictureurl.NewWithConfig(pictureurl.Config{
		CacheLimit: 4,
		UserAgent:  "CQOps/1.0 (ham-radio-logger)",
		HTTPClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	})
	m.mapView = newMapRenderer()
	m.psk.filterMins = pskFilterSteps[0] // default 5 min
	m.ref = newRefState()
	if dir, err := config.CacheDir(); err == nil {
		m.psk.cacheDir = dir
		m.solar.cacheDir = dir
	}
	m.psk.lastFetchByCall = make(map[string]time.Time)
	m.applyBeepOnError()
	m.keepComment = a.Config.State.RetainComment
	if m.keepComment && a.Config.State.RetainedComment != "" {
		m.fields[fieldComment].SetValue(a.Config.State.RetainedComment)
	}
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
	m.refreshRigClient()
	m.refreshRotorClient()
	m.App.WSJTX.OnADIF = func(adif string) {
		m.adifQ.mu.Lock()
		m.adifQ.adifs = append(m.adifQ.adifs, adif)
		// Persist to disk immediately so QSOs survive crashes.
		// Failures are silent — the in-memory queue is authoritative.
		m.savePendingADIFsLocked()
		m.adifQ.mu.Unlock()
	}
	// Recover any ADIF records left on disk from a previous crash.
	if saved := loadPendingADIFs(); len(saved) > 0 {
		m.adifQ.adifs = append(m.adifQ.adifs, saved...)
		applog.Info("WSJT-X: recovered pending ADIF records from disk", "count", len(saved))
	}
	m.App.WSJTX.OnStatus = func(call, grid string, freq uint64, mode, submode, report, txMessage string, transmitting bool) {
		m.adifQ.mu.Lock()
		m.adifQ.status = statusPending{call: call, grid: grid, freq: freq, mode: mode, submode: submode, report: report, txMessage: txMessage, transmitting: transmitting, hasData: true}
		m.adifQ.mu.Unlock()
	}
	// Read WSJT-X config from the active rig preset (per-rig).
	wsjtxE, wsjtxH, wsjtxP := false, "127.0.0.1", 2233
	if rp, ok := m.App.Config.Rigs[m.App.Logbook.Station.RigName]; ok {
		wsjtxE = rp.WsjtxEnabled
		if rp.WsjtxUDPHost != "" {
			wsjtxH = rp.WsjtxUDPHost
		}
		if rp.WsjtxUDPPort > 0 {
			wsjtxP = rp.WsjtxUDPPort
		}
	}
	if wsjtxE {
		applog.Info("WSJT-X: callbacks registered, restarting listener")
		m.App.MaybeRestartWSJTX(wsjtxE, wsjtxH, wsjtxP)
	} else {
		applog.Debug("wsjt-x: disabled")
	}
	cmds := []tea.Cmd{tickCmd(), m.photo.viewer.Init(), m.emitWindowIconCmd()}
	if !m.Offline {
		cmds = append(cmds, checkInetCmd())
	}
	return tea.Batch(cmds...)
}

// emitWindowIconCmd emits OSC 0 which sets both the terminal window title and
// the icon name. This is the only escape sequence that many terminals (xterm,
// Windows Terminal, Konsole, GNOME Terminal, Kitty) use for the taskbar/dock
// icon label. It's a one-shot — subsequent title updates via WindowTitle use
// OSC 2 (title only) and won't overwrite the icon name.
func (m *Model) emitWindowIconCmd() tea.Cmd {
	return func() tea.Msg {
		fmt.Fprintf(os.Stderr, "\x1b]0;%s\x07", m.windowTitle())
		return nil
	}
}
func tickCmd() tea.Cmd {
	return tea.Tick(1000*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// isSubmodelActive returns true when any sub-screen is visible.
func (m *Model) isSubmodelActive() bool {
	return m.screen != screenQSO
}

// saveConfig persists the app configuration and shows a toast.
func (m *Model) saveConfig(msg string) {
	if err := m.App.Config.Validate(); err != nil {
		m.toasts.Error("Settings save failed: " + err.Error())
		applog.Error("Config validation failed before save", "error", err)
		return
	}
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
		if c := m.photo.viewer.Update(msg); c != nil {
			cmd = tea.Batch(cmd, c)
		}
		// Update focused textinput width so scrolling stays correct.
		// Set width on ALL form fields so View() doesn't need to mutate them.
		if m.screen == screenQSO {
			if m.width > 60 {
				colW := (m.width - 4) / 3
				if colW > 41 {
					colW = 41
				}
				vw := colW - 14 // subtract label+icon padding
				if vw < 3 {
					vw = 3
				}
				if vw > 40 {
					vw = 40
				}
				for f := field(0); f < fieldCount; f++ {
					ti := m.fields[f]
					ti.SetWidth(vw)
					m.fields[f] = ti
				}
			} else if m.width > 20 {
				vw := m.width - 16
				if vw < 3 {
					vw = 3
				}
				if vw > 40 {
					vw = 40
				}
				for f := field(0); f < fieldCount; f++ {
					ti := m.fields[f]
					ti.SetWidth(vw)
					m.fields[f] = ti
				}
			}
		}
	}

	// Active spot dialog — blocks key input while open, but lets ticks
	// through so toasts continue to expire.
	if m.spotDialog != nil {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			updated, spotCmd := m.spotDialog.Update(keyMsg)
			*m.spotDialog = updated.(SpotDialog)
			if m.spotDialog.Done() {
				if m.spotDialog.Result.Confirmed {
					cmd = tea.Batch(cmd, m.sendSpotCmd(m.spotDialog.Call, m.spotDialog.FreqKhz, m.spotDialog.Result.Comment))
				}
				m.spotDialog = nil
			}
			if spotCmd != nil {
				cmd = tea.Batch(cmd, spotCmd)
			}
			return m, cmd
		}
		// Non-key messages fall through for tick / toast expiry / async processing.
	}

	// Active help overlay — blocks all input except dismiss keys.
	if m.help.ShowAll {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if key.Matches(keyMsg, m.keys.Help) || key.Matches(keyMsg, m.keys.Cancel) {
				m.help.ShowAll = false
			}
			// Consume all keys while help is shown.
		}
		return m, cmd
	}

	// Active confirmation dialog — highest priority, blocks everything else
	if m.confirm != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			updated, _ := m.confirm.Update(msg)
			*m.confirm = updated.(DialogModel)
			if m.confirm.Done() {
				if m.confirm.Result.Confirmed && m.confirm.Result.Value == "quit" {
					m.shutdownConnections()
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
	if c := m.photo.partnerPicViewer.Update(msg); c != nil {
		cmd = tea.Batch(cmd, c)
	}

	// Async result messages (internet, Wavelog, rig)
	if handled, asyncCmd := m.handleAsyncMessages(msg); handled {
		if asyncCmd != nil {
			cmd = tea.Batch(cmd, asyncCmd)
		}
		if _, ok := msg.(inetResultMsg); ok && m.inetOnline {
			cmd = tea.Batch(cmd, m.maybeCheckVersion())
		}
		if _, ok := msg.(rigPollMsg); ok {
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
		m.contestAutoFocusExchRcvd()
		if m.photo.partnerPicNeedLoad {
			m.photo.partnerPicNeedLoad = false
			w := m.photo.partnerPicW
			h := m.photo.partnerPicH
			if w < 25 {
				w = 40
			}
			if h < 4 {
				h = 15
			}
			cmd = tea.Batch(cmd, m.photo.partnerPicViewer.SetSize(w, h),
				m.photo.partnerPicViewer.SetURL(m.photo.partnerPicURL))
		}
		return m, cmd
	case wlResultMsg:
		wlCmd := m.fillWLData(r)
		cmd = tea.Batch(cmd, wlCmd)
		m.contestAutoFocusExchRcvd()
		return m, cmd
	case logbookStatsMsg:
		m.handleLogbookStats(r)
		return m, cmd
	case pskSpotsLoadedMsg:
		if r.err == nil && r.spotKey != "" {
			m.psk.spots = r.spots
			m.psk.spotKey = r.spotKey
		}
		return m, cmd
	case refRebuildMsg:
		m.ref.building = false
		m.ref.refNamesDirty = true
		if r.err != nil {
			applog.Warn("REF: rebuild failed", "error", r.err)
			m.toasts.Error("REF database build failed")
		} else {
			m.ref.ready = true
			applog.Info("REF: rebuild complete", "total", r.total)
			m.toasts.Success(fmt.Sprintf("REF database ready — %d references", r.total))
		}
		return m, cmd
	case dxcSpotLookupMsg:
		m.fillDXCFreq(r)
		return m, cmd
	case dxcSpotsStoredMsg:
		m.handleDXCSpotsStored(r)
		return m, cmd
	case dxcTuneResultMsg:
		if r.err != nil {
			if strings.Contains(r.err.Error(), "cancelled") {
				m.toasts.Warn(fmt.Sprintf("Tune cancelled: %v", r.err))
			} else {
				m.toasts.Error(fmt.Sprintf("Tune failed: %v", r.err))
			}
		} else {
			msg := fmt.Sprintf("Rig tuned to %.5f MHz", r.freqMHz)
			if r.mode != "" {
				msg += " " + r.mode
			}
			if r.verify != "" {
				m.toasts.Warn("Rig tuning failed")
			} else {
				m.toasts.Success(msg)
			}
		}
		return m, cmd
	case bplTuneResultMsg:
		if r.err != nil {
			if strings.Contains(r.err.Error(), "cancelled") {
				m.toasts.Warn(fmt.Sprintf("Tune cancelled: %v", r.err))
			} else {
				m.toasts.Error(fmt.Sprintf("Tune failed: %v", r.err))
			}
		} else {
			msg := fmt.Sprintf("Rig tuned to %.5f MHz", r.freqMHz)
			if r.mode != "" {
				msg += " " + r.mode
			}
			if r.verify != "" {
				m.toasts.Warn("Rig tuning failed")
			} else {
				m.toasts.Success(msg)
			}
		}
		return m, cmd
	case bplExportMsg:
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("Export failed: %v", r.err))
		} else {
			m.toasts.Success(fmt.Sprintf("Band plan exported to %s", r.path))
		}
		return m, cmd
	case qsoRefreshedMsg:
		if r.err != nil {
			m.toasts.Error(fmt.Sprintf("Refresh failed: %v", r.err))
		} else {
			m.qsos = r.qsos
			m.recentQSOs.SetQSOS(r.qsos)
			m.rc.pathSig = ""
			m.rc.logStatsSig = ""
			if !m.recentQSOs.filterSuppressed && m.recentQSOs.IsFiltered() {
				filtered, filterErr := store.SearchQSOsByCall(m.App.DB, m.recentQSOs.filterCall, 200)
				if filterErr == nil {
					m.recentQSOs.SetFilterCall(m.recentQSOs.filterCall, filtered)
				}
			}
			m.recentQSOs.filterSuppressed = false
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
	case screenContest:
		return m.handleContestUpdate(msg, cmd)
	case screenOperator:
		return m.handleOperatorUpdate(msg, cmd)
	case screenConfig:
		return m.handleConfigUpdate(msg, cmd)
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
		if m.lookup.partnerData != nil && m.lookup.partnerData.ImageURL != "" &&
			m.lookup.partnerData.ImageURL != m.photo.lastURL {
			m.photo.lastURL = m.lookup.partnerData.ImageURL
			w := m.width
			h := contentHeight(m.height)
			if w < 20 {
				w = 80
			}
			if h < 10 {
				h = 10
			}
			cmd = tea.Batch(cmd, m.photo.viewer.SetSize(w, h-1), m.photo.viewer.SetURL(m.lookup.partnerData.ImageURL))
		}
		// Log image errors once and show toast.
		if err := m.photo.viewer.Err(); err != nil && m.photo.lastErr != err {
			m.photo.lastErr = err
			applog.Warn("Image load failed", "error", err.Error())
			m.toasts.Warn("Photo unavailable — unsupported format")
		}
		if m.photo.viewer.Err() == nil {
			m.photo.lastErr = nil
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
			if c := m.photo.viewer.SetSize(w, h); c != nil {
				cmd = tea.Batch(cmd, c)
			}
		}
		c := m.photo.viewer.Update(msg)
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
	case screenRef:
		return m.handleRefUpdate(msg, cmd)
	case screenBPL:
		return m.handleBPLUpdate(msg, cmd)
	}

	// Forward paste messages to the focused textinput so clipboard paste works.
	if pasteMsg, ok := msg.(tea.PasteMsg); ok && m.screen == screenQSO && !m.keepFocused {
		f := m.focus
		ti, c := m.fields[f].Update(pasteMsg)
		m.fields[f] = ti
		if c != nil {
			cmd = tea.Batch(cmd, c)
		}
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
	if m.rc.lastLayoutW == m.width && m.rc.lastLayoutH == m.height && m.rc.lastLayoutSc == m.screen {
		layout = m.rc.lastLayout
	} else {
		layout = MeasureLayout(m)
		m.rc.lastLayout = layout
		m.rc.lastLayoutW = m.width
		m.rc.lastLayoutH = m.height
		m.rc.lastLayoutSc = m.screen
	}

	if layout.TerminalW < 75 || layout.TerminalH < 24 {
		msg := fmt.Sprintf("\n  CQOps — Terminal too small: %dx%d (min 75x24)\n\n  Press F10 and then Enter to quit",
			layout.TerminalW, layout.TerminalH)
		return tea.NewView(ErrorStyle.Render(msg))
	}

	// Render fixed bars — cache when screen and width haven't changed.
	// Status bar has a 1-second TTL because it contains the UTC clock.
	cacheBars := m.rc.barW == m.width && m.rc.barSc == m.screen &&
		m.rc.barOp == m.App.Logbook.ActiveOperator &&
		m.rc.barLog == m.App.LogbookName &&
		m.rc.barRig == m.App.Logbook.Station.RigName &&
		m.rc.barTx == m.wsjtx.tx && m.rc.barTxMsg == m.wsjtx.txMsg
	if !cacheBars {
		m.rc.status = ""
	}
	if m.rc.status == "" || m.rc.statusSec != time.Now().UTC().Second() {
		m.rc.status = m.renderStatusBar()
		m.rc.statusSec = time.Now().UTC().Second()
	}
	// Tab bar depends on partner data / call field / connectivity — cached.
	m.rc.tabs = m.renderTabBar()
	// Help bar has dynamic suffix (QSO counter, scroll info) — cached.
	m.rc.help = m.renderHelpBar()
	m.rc.barW = m.width
	m.rc.barSc = m.screen
	m.rc.barOp = m.App.Logbook.ActiveOperator
	m.rc.barLog = m.App.LogbookName
	m.rc.barRig = m.App.Logbook.Station.RigName
	m.rc.barTx = m.wsjtx.tx
	m.rc.barTxMsg = m.wsjtx.txMsg

	var mainParts []string
	addRow := func(s string) {
		if s != "" {
			mainParts = append(mainParts, s)
		}
	}
	addRow(m.rc.status)
	addRow(m.rc.tabs)

	body := m.buildBodyForScreen(layout)
	if body == "" {
		body = DimStyle.Render("\u2014")
	}
	addRow(body)
	addRow(m.rc.help)
	mainView := lipgloss.JoinVertical(lipgloss.Left, mainParts...)

	// Composite confirm dialog as a centered overlay if active
	if m.confirm != nil {
		mainView = RenderDialogOverlay(mainView, *m.confirm, layout.TerminalW, layout.TerminalH)
	}

	// Composite spot dialog as a centered overlay if active
	if m.spotDialog != nil {
		mainView = RenderSpotDialogOverlay(mainView, *m.spotDialog, layout.TerminalW, layout.TerminalH)
	}

	// Composite toasts as a floating overlay in the bottom-right corner.
	// Clip mainView to terminal height first so the compositor canvas
	// matches the visible terminal exactly.
	mainView = lipgloss.NewStyle().MaxHeight(layout.TerminalH).Render(mainView)
	finalView := RenderToastOverlay(mainView, m.toasts.Active(), layout.TerminalW, layout.TerminalH)

	// Help overlay — floating bottom-left, above toasts, when ? is pressed.
	if m.help.ShowAll {
		finalView = m.renderHelpOverlay(finalView, layout)
	}

	v := tea.NewView(finalView)
	v.AltScreen = true
	v.WindowTitle = m.windowTitle()
	return v
}

// viewImage renders the partner photo full-screen.
func (m *Model) viewImage(l Layout) string {
	content := m.photo.viewer.View().Content
	if m.photo.viewer.Err() != nil || content == "" {
		msg := ""
		if m.photo.viewer.Err() != nil {
			err := m.photo.viewer.Err()
			msg = err.Error()
			if strings.Contains(msg, "unexpected Content-Type") {
				msg = "QRZ photo not available — unsupported format"
			} else if strings.Contains(msg, "no such host") {
				msg = "Cannot reach image server"
			} else if strings.Contains(msg, "timeout") {
				msg = "Image download timed out"
			}
		}
		if msg == "" && m.photo.lastURL != "" {
			msg = "Loading photo\u2026"
		} else if msg == "" && m.lookup.partnerData != nil {
			msg = "No photo for " + m.lookup.partnerData.Callsign
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
		if m.lookup.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != "" {
			body = m.viewPartner()
		}
	case screenImage:
		body = m.viewImage(l)
	case screenPSKReporter:
		body = m.viewPSKReporter()
	case screenMainMenu:
		body = m.ui.mainMenu.View().Content
	case screenConfig:
		body = m.ui.configMenu.View().Content
	case screenIntegration:
		body = m.ui.integrationMenu.View().Content
	case screenChooser:
		body = m.ui.chooser.View().Content
	case screenRigEdit:
		body = m.ui.rigChooser.View().Content
	case screenContest:
		body = m.ui.contestChooser.View().Content
	case screenOperator:
		body = m.ui.operatorChooser.View().Content
	case screenLogView:
		body = m.ui.logViewer.View().Content
	case screenLogbookEditor:
		body = m.ui.logbookEditor.View().Content
	case screenNotifications:
		body = m.ui.notifMenu.View().Content
	case screenDXC:
		body = m.dxcView()
	case screenRef:
		body = m.viewRef()
	case screenBPL:
		body = m.viewBPL(l)
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
	// Cap to same max as QSO form for visual consistency.
	if tableW > partnerMapMaxW {
		tableW = partnerMapMaxW
	}
	if tableH < 3 {
		tableH = 3
	}

	// SCP auto-complete suggestions — shown below the QSO form when the
	// call field is focused and SCP is enabled.
	var scpBox string
	if m.focus == fieldCall && len(m.scpMatches) > 0 {
		scpH := len(m.scpMatches)
		if scpH > 12 {
			scpH = 12
		}
		if tableH-scpH < 5 {
			scpH = tableH - 5
		}
		if scpH > 0 {
			scpLines := m.scpMatches[:scpH]
			// Highlight the matching prefix in each callsign with Info (cyan)
			// and dim the rest.
			prefix := m.scpCacheKey
			var highlighted []string
			for _, c := range scpLines {
				if prefix != "" && strings.HasPrefix(strings.ToUpper(c), strings.ToUpper(prefix)) {
					match := c[:len(prefix)]
					rest := c[len(prefix):]
					highlighted = append(highlighted, S.Info.Render(match)+DimStyle.Render(rest))
				} else {
					highlighted = append(highlighted, DimStyle.Render(c))
				}
			}
			scpContent := strings.Join(highlighted, DimStyle.Render("  "))
			// Match the form row width (QSO form + solar panel if present).
			scpW := lipgloss.Width(formRow)
			if scpW < 40 {
				scpW = 40
			}
			scpBox = drawBorderedBox(scpContent, scpW)
			scpBoxH := lipgloss.Height(scpBox)
			tableH -= scpBoxH
		}
	}

	// REF names — resolved from SOTA/POTA/WWFF/IOTA fields.
	var refBox string
	refLine := m.buildRefNamesLine()
	if refLine != "" {
		refW := lipgloss.Width(formRow)
		if refW < 40 {
			refW = 40
		}
		refBox = drawBorderedBox(refLine, refW)
		refBoxH := lipgloss.Height(refBox)
		tableH -= refBoxH
	}

	// Contest mode box — shown when a contest is active.
	var contestBox string
	contestLine := m.buildContestLine()
	if contestLine != "" {
		contestW := lipgloss.Width(formRow)
		if contestW < 40 {
			contestW = 40
		}
		contestBox = contestBoxStyle.Width(contestW).Render(contestLine)
		contestBoxH := lipgloss.Height(contestBox)
		tableH -= contestBoxH
	}

	m.recentQSOs.SetSize(tableW, tableH)

	var parts []string
	if profileLine != "" {
		parts = append(parts, profileLine)
	}
	parts = append(parts, formRow)
	if scpBox != "" {
		parts = append(parts, scpBox)
	}
	if contestBox != "" {
		parts = append(parts, contestBox)
	}
	if refBox != "" {
		parts = append(parts, refBox)
	}
	parts = append(parts, m.recentQSOs.View())
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// buildContestLine returns the contest info line for the QSO screen, or "" if
// no contest is active.
func (m *Model) buildContestLine() string {
	id := m.App.Logbook.ActiveContest
	if id == "" {
		return ""
	}
	ct, ok := m.App.Config.Contests[id]
	if !ok {
		return ""
	}
	return fmt.Sprintf(" Contest: %s   Contest ID: %s   Next QSO seq: %d",
		config.ContestDisplayName(&ct), ct.ContestID, ct.NextQSO)
}

// cycleActiveContest rotates through all active contests (excluding None).
// Persists the new active contest to config silently.
func (m *Model) cycleActiveContest() {
	ids := config.ActiveContestIDs(m.App.Config, m.App.LogbookName)
	current := m.App.Logbook.ActiveContest

	// No active contests — nothing to cycle.
	if len(ids) == 0 {
		m.toasts.Warn("No contests configured — create one in F9 → Contests")
		return
	}

	if current == "" {
		m.App.SetActiveContest(ids[0])
		ct := m.App.Config.Contests[ids[0]]
		m.toasts.Info(fmt.Sprintf("Contest: %s", config.ContestDisplayName(&ct)))
		m.needRefresh = true
		config.Save(m.App.ConfigPath, m.App.Config)
		return
	}

	for i, id := range ids {
		if id == current {
			if i+1 < len(ids) {
				m.App.SetActiveContest(ids[i+1])
				ct := m.App.Config.Contests[ids[i+1]]
				m.toasts.Info(fmt.Sprintf("Contest: %s", config.ContestDisplayName(&ct)))
			} else {
				m.App.SetActiveContest("")
				m.toasts.Info("Contest: None")
			}
			m.needRefresh = true
			config.Save(m.App.ConfigPath, m.App.Config)
			return
		}
	}

	// Current contest not found in active list (possibly deleted or set to
	// not-in-use) — clear and wrap to first.
	m.App.SetActiveContest("")
	m.toasts.Info("Contest: None")
	m.needRefresh = true
	config.Save(m.App.ConfigPath, m.App.Config)
}

// activeOperatorCallsign returns the callsign of the active operator,
// or an empty string when no active operator is selected.
func (m *Model) activeOperatorCallsign() string {
	if m.App.Logbook.ActiveOperator != "" {
		if op, ok := m.App.Config.Operators[m.App.Logbook.ActiveOperator]; ok {
			return op.Callsign
		}
	}
	return ""
}

// cycleActiveOperator cycles the active operator for the current logbook
// through: None → first operator → second → … → None.
func (m *Model) cycleActiveOperator() {
	ids := config.SortedOperatorIDs(m.App.Config)
	current := m.App.Logbook.ActiveOperator

	if len(ids) == 0 {
		m.toasts.Warn("No operators configured — add one in F9 → Operators")
		return
	}

	if current == "" {
		m.App.SetActiveOperator(ids[0])
		op := m.App.Config.Operators[ids[0]]
		m.toasts.Info(fmt.Sprintf("Operator: %s", config.OperatorDisplayName(&op)))
		applog.Info("Operator cycled", "callsign", op.Callsign)
		config.Save(m.App.ConfigPath, m.App.Config)
		return
	}

	for i, id := range ids {
		if id == current {
			if i+1 < len(ids) {
				m.App.SetActiveOperator(ids[i+1])
				op := m.App.Config.Operators[ids[i+1]]
				m.toasts.Info(fmt.Sprintf("Operator: %s", config.OperatorDisplayName(&op)))
				applog.Info("Operator cycled", "callsign", op.Callsign)
			} else {
				m.App.SetActiveOperator("")
				m.toasts.Info("Operator: None")
				applog.Info("Operator cycled", "callsign", "None")
			}
			config.Save(m.App.ConfigPath, m.App.Config)
			return
		}
	}

	// Current operator not found — clear.
	m.App.SetActiveOperator("")
	m.toasts.Info("Operator: None")
	applog.Info("Operator cycled (not found)", "callsign", "None")
	config.Save(m.App.ConfigPath, m.App.Config)
}

// prefillContestExchange fills the exchange sent/rcvd fields from the active
// contest's prefill settings. Replaces special markers with current QSO
// and station values. Next QSO is NOT incremented here — that happens on
// successful QSO save.
func (m *Model) prefillContestExchange() {
	id := m.App.Logbook.ActiveContest
	if id == "" {
		return
	}
	ct, ok := m.App.Config.Contests[id]
	if !ok {
		return
	}

	// Sent exchange: always regenerate (serial increments each QSO).
	if ct.PrefillExchange && ct.ExchangeSent != "" {
		val := m.resolveExchangeMarkers(ct.ExchangeSent, "sent", ct.NextQSO)
		m.fields[fieldExchSent].SetValue(val)
	}

	// Received exchange: only fill if empty (don't overwrite user input).
	if ct.PrefillExchangeRcvd && ct.ExchangeRcvd != "" && strings.TrimSpace(m.fields[fieldExchRcvd].Value()) == "" {
		val := strings.TrimSpace(m.resolveExchangeMarkers(ct.ExchangeRcvd, "rcvd", ct.NextQSO))
		m.fields[fieldExchRcvd].SetValue(val)
	}
}

// resolveExchangeMarkers replaces special markers in an exchange template with
// values from the current QSO form and station config. Supported markers:
//
//	@rst     — RST sent or received (depending on direction)
//	@serial  — next contest QSO sequence number (zero-padded to 3 digits)
//	@cqz     — DX station CQ zone (from form)
//	@mycqz   — station CQ zone (from logbook config)
//	@itu     — DX station ITU zone (from form)
//	@myitu   — station ITU zone (from logbook config)
//	@grid    — DX station grid square (from form)
//	@mygrid  — station grid square (from logbook config)
//
// Backward-compatible: ### is also replaced with @serial.
func (m *Model) resolveExchangeMarkers(tmpl, direction string, nextQSO int) string {
	// Build replacement map.
	rep := make(map[string]string)

	// @serial — contest sequence number.
	// For received exchange, keep @serial as a format placeholder so the
	// operator knows where to type the received serial; the actual number
	// is parsed from user input at save time via ParseSerial.
	if direction == "sent" {
		seq := fmt.Sprintf("%03d", nextQSO)
		if nextQSO > 999 {
			seq = fmt.Sprintf("%d", nextQSO)
		}
		rep["@serial"] = seq
	} else {
		rep["@serial"] = ""
	}

	// @rst — RST depending on direction.
	if direction == "sent" {
		rep["@rst"] = strings.TrimSpace(m.fields[fieldRSTSent].Value())
	} else {
		rep["@rst"] = strings.TrimSpace(m.fields[fieldRSTRcvd].Value())
	}

	// @cqz / @itu / @grid — from the QSO form or DXCC lookup.
	call := strings.TrimSpace(m.fields[fieldCall].Value())
	if call != "" && m.App != nil && m.App.DXCC != nil && m.App.Config.General.UseCTY {
		if p := m.dxccLookup(call); p != nil {
			rep["@cqz"] = fmt.Sprintf("%d", int(p.CQZone))
			rep["@itu"] = fmt.Sprintf("%d", int(p.ITUZone))
		}
	}
	// Fall back to empty if no DXCC match.
	if _, ok := rep["@cqz"]; !ok {
		rep["@cqz"] = ""
	}
	if _, ok := rep["@itu"]; !ok {
		rep["@itu"] = ""
	}
	rep["@grid"] = strings.TrimSpace(m.fields[fieldGrid].Value())

	// @mycqz / @myitu / @mygrid — from station config.
	rep["@mycqz"] = qso.ItoaOrEmpty(m.App.Logbook.Station.CQZone)
	rep["@myitu"] = qso.ItoaOrEmpty(m.App.Logbook.Station.ITUZone)
	rep["@mygrid"] = strings.ToUpper(strings.TrimSpace(m.App.Logbook.Station.Grid))

	// Optional markers that resolved to empty → render "?".
	for k, v := range rep {
		if v == "" && k != "@serial" && k != "@rst" {
			rep[k] = "?"
		}
	}

	result := tmpl
	for marker, value := range rep {
		result = strings.ReplaceAll(result, marker, value)
	}
	return result
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
	if len(m.adifQ.adifs) == 0 {
		os.Remove(path)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	for _, adif := range m.adifQ.adifs {
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
