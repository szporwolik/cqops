package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/hamqth"
	"github.com/szporwolik/cqops/internal/qrzcom"
	"github.com/szporwolik/cqops/internal/qrzru"
)

// callbookRows is the row style shared by the callbook menu's checkbox rows.
var callbookRows = rowStyle{label: S.FormLabelWide, focused: S.FormFocusedWide}

// CallbookMenu is a scrollable sub-menu for callbook provider settings.
// Currently QRZ.com is the only provider; the menu is structured so
// additional providers can be added as separate sections later.
type CallbookMenu struct {
	// General options
	baseCallFallback bool

	// Logbook provider
	logEnabled  bool
	logPriority textinput.Model

	// QRZ fields
	qrzEnabled    bool
	qrzUser       textinput.Model
	qrzPass       textinput.Model
	qrzPriority   textinput.Model
	qrzTesting    bool
	qrzTestResult string
	inetOnline    bool

	// HamQTH fields
	hamqthEnabled    bool
	hamqthUser       textinput.Model
	hamqthPass       textinput.Model
	hamqthPriority   textinput.Model
	hamqthTesting    bool
	hamqthTestResult string

	// Callook fields
	callookEnabled  bool
	callookPriority textinput.Model

	// QRZ.RU fields
	qrzruEnabled    bool
	qrzruUser       textinput.Model
	qrzruPass       textinput.Model
	qrzruPriority   textinput.Model
	qrzruTesting    bool
	qrzruTestResult string

	// Wavelog provider
	wlEnabled    bool
	wlConfigured bool // true only when a logbook has Wavelog configured
	wlPriority   textinput.Model

	fm     menuFocus
	done   bool
	saved  bool
	goBack bool
	width  int
	height int

	// saveError is set when Ctrl+S is blocked by validation.
	SaveError string

	// TestToast is set by QRZ/HamQTH test handlers; parent shows toast.
	TestToast string

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastBodyContent string
}

const (
	cmBaseCall        = 0
	cmQRZChk          = 1
	cmQRZUser         = 2
	cmQRZPass         = 3
	cmQRZPriority     = 4
	cmQRZTest         = 5
	cmHamQTHChk       = 6
	cmHamQTHUser      = 7
	cmHamQTHPass      = 8
	cmHamQTHPriority  = 9
	cmHamQTHTest      = 10
	cmCallookChk      = 11
	cmCallookPriority = 12
	cmQRZRuChk        = 13
	cmQRZRuUser       = 14
	cmQRZRuPass       = 15
	cmQRZRuPriority   = 16
	cmQRZRuTest       = 17
	cmLogChk          = 18
	cmLogPriority     = 19
	cmWavelogChk      = 20
	cmWavelogPriority = 21
	cmMax             = 22
)

func NewCallbookMenu(cfg *config.Config) *CallbookMenu {
	logPriority := newTextinput()
	logPriority.CharLimit = 5
	logPriority.SetWidth(6)
	logPriority.Placeholder = strconv.Itoa(config.DefaultLogbookPriority)
	if cfg.Integrations.Callbook.Logbook.Priority != 0 {
		logPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.Logbook.Priority))
	} else {
		logPriority.SetValue(strconv.Itoa(config.DefaultLogbookPriority))
	}
	logEnabled := cfg.Integrations.Callbook.Logbook.Enabled
	// Default to enabled on first run (Priority=0 means never configured).
	if !logEnabled && cfg.Integrations.Callbook.Logbook.Priority == 0 {
		logEnabled = true
	}

	qrzUser := newTextinput()
	qrzUser.CharLimit = 30
	qrzUser.SetWidth(28)
	qrzUser.Placeholder = "QRZ.com username"
	qrzUser.SetValue(cfg.Integrations.Callbook.QRZ.User)

	qrzPass := newTextinput()
	qrzPass.CharLimit = 40
	qrzPass.SetWidth(28)
	qrzPass.Placeholder = "QRZ.com password"
	qrzPass.EchoMode = textinput.EchoPassword
	qrzPass.EchoCharacter = '*'
	qrzPass.SetValue(cfg.Integrations.Callbook.QRZ.Pass)

	qrzPriority := newTextinput()
	qrzPriority.CharLimit = 5
	qrzPriority.SetWidth(6)
	qrzPriority.Placeholder = strconv.Itoa(config.DefaultQRZPriority)
	qrzPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.QRZ.Priority))
	if cfg.Integrations.Callbook.QRZ.Priority == 0 {
		qrzPriority.SetValue(strconv.Itoa(config.DefaultQRZPriority))
	}

	// HamQTH provider.
	hamqthUser := newTextinput()
	hamqthUser.CharLimit = 30
	hamqthUser.SetWidth(28)
	hamqthUser.Placeholder = "HamQTH username"
	hamqthUser.SetValue(cfg.Integrations.Callbook.HamQTH.User)

	hamqthPass := newTextinput()
	hamqthPass.CharLimit = 40
	hamqthPass.SetWidth(28)
	hamqthPass.Placeholder = "HamQTH password"
	hamqthPass.EchoMode = textinput.EchoPassword
	hamqthPass.EchoCharacter = '*'
	hamqthPass.SetValue(cfg.Integrations.Callbook.HamQTH.Pass)

	hamqthPriority := newTextinput()
	hamqthPriority.CharLimit = 5
	hamqthPriority.SetWidth(6)
	hamqthPriority.Placeholder = strconv.Itoa(config.DefaultHamQTHPriority)
	hamqthPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.HamQTH.Priority))
	if cfg.Integrations.Callbook.HamQTH.Priority == 0 {
		hamqthPriority.SetValue(strconv.Itoa(config.DefaultHamQTHPriority))
	}

	// Callook.info provider (no auth required, US callsigns only).
	callookPriority := newTextinput()
	callookPriority.CharLimit = 5
	callookPriority.SetWidth(6)
	callookPriority.Placeholder = strconv.Itoa(config.DefaultCallookPriority)
	callookPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.Callook.Priority))
	if cfg.Integrations.Callbook.Callook.Priority == 0 {
		callookPriority.SetValue(strconv.Itoa(config.DefaultCallookPriority))
	}

	// QRZ.RU provider (free, RU and surrounding countries).
	qrzruUser := newTextinput()
	qrzruUser.CharLimit = 30
	qrzruUser.SetWidth(28)
	qrzruUser.Placeholder = "QRZ.ru API login"
	qrzruUser.SetValue(cfg.Integrations.Callbook.QRZRu.User)

	qrzruPass := newTextinput()
	qrzruPass.CharLimit = 40
	qrzruPass.SetWidth(28)
	qrzruPass.Placeholder = "QRZ.ru API password"
	qrzruPass.EchoMode = textinput.EchoPassword
	qrzruPass.EchoCharacter = '*'
	qrzruPass.SetValue(cfg.Integrations.Callbook.QRZRu.Pass)

	qrzruPriority := newTextinput()
	qrzruPriority.CharLimit = 5
	qrzruPriority.SetWidth(6)
	qrzruPriority.Placeholder = strconv.Itoa(config.DefaultQRZRuPriority)
	qrzruPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.QRZRu.Priority))
	if cfg.Integrations.Callbook.QRZRu.Priority == 0 {
		qrzruPriority.SetValue(strconv.Itoa(config.DefaultQRZRuPriority))
	}

	// Default base-call fallback and Callook.info to enabled on fresh config.
	// Use LogbookCallbook priority as a proxy for "callbook section never configured".
	freshCallbook := cfg.Integrations.Callbook.Logbook.Priority == 0 &&
		cfg.Integrations.Callbook.QRZ.Priority == 0 &&
		cfg.Integrations.Callbook.HamQTH.Priority == 0 &&
		cfg.Integrations.Callbook.Callook.Priority == 0 &&
		cfg.Integrations.Callbook.QRZRu.Priority == 0
	baseFallback := cfg.Integrations.Callbook.BaseCallFallback
	if freshCallbook {
		baseFallback = true
	}
	callookEnabled := cfg.Integrations.Callbook.Callook.Enabled
	if freshCallbook {
		callookEnabled = true
	}

	// Wavelog callbook provider.
	wlPriority := newTextinput()
	wlPriority.CharLimit = 5
	wlPriority.SetWidth(6)
	wlPriority.Placeholder = strconv.Itoa(config.DefaultWavelogPriority)
	wlPriority.SetValue(strconv.Itoa(cfg.Integrations.Callbook.Wavelog.Priority))
	if cfg.Integrations.Callbook.Wavelog.Priority == 0 {
		wlPriority.SetValue(strconv.Itoa(config.DefaultWavelogPriority))
	}
	wlEnabled := cfg.Integrations.Callbook.Wavelog.Enabled
	wlConfigured := false
	for _, lb := range cfg.Logbooks {
		if lb.Wavelog != nil && lb.Wavelog.Enabled && lb.Wavelog.URL != "" && lb.Wavelog.APIKey != "" {
			wlConfigured = true
			break
		}
	}

	// CTY.DAT is always enabled — it's an offline prefix database that
	// fills country and grid instantly before other providers run.

	return &CallbookMenu{
		baseCallFallback: baseFallback,
		logEnabled:       logEnabled,
		logPriority:      logPriority,
		qrzEnabled:       cfg.Integrations.Callbook.QRZ.Enabled,
		qrzUser:          qrzUser,
		qrzPass:          qrzPass,
		qrzPriority:      qrzPriority,
		hamqthEnabled:    cfg.Integrations.Callbook.HamQTH.Enabled,
		hamqthUser:       hamqthUser,
		hamqthPass:       hamqthPass,
		hamqthPriority:   hamqthPriority,
		callookEnabled:   callookEnabled,
		callookPriority:  callookPriority,
		qrzruEnabled:     cfg.Integrations.Callbook.QRZRu.Enabled,
		qrzruUser:        qrzruUser,
		qrzruPass:        qrzruPass,
		qrzruPriority:    qrzruPriority,
		wlEnabled:        wlEnabled,
		wlConfigured:     wlConfigured,
		wlPriority:       wlPriority,
	}
}

func (cm *CallbookMenu) Init() tea.Cmd { return nil }

func (cm *CallbookMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cm.width, cm.height = msg.Width, msg.Height

	case callbookTestMsg:
		switch msg.provider {
		case "hamqth":
			cm.hamqthTesting = false
			if msg.err != nil {
				cm.hamqthTestResult = friendlyHTestError(msg.err)
				cm.TestToast = friendlyHTestError(msg.err)
				applog.Error("HamQTH test failed", "error", msg.err.Error())
			} else if msg.ok {
				cm.hamqthTestResult = "OK — HamQTH connected"
				cm.TestToast = "HamQTH: connection verified"
				applog.Info("HamQTH test OK")
			} else {
				cm.hamqthTestResult = "Connected OK — OK1HRA not found (API works)"
				cm.TestToast = "HamQTH: connected, but test lookup returned no data"
				applog.Warn("HamQTH test: no data returned")
			}
		case "qrzru":
			cm.qrzruTesting = false
			if msg.err != nil {
				cm.qrzruTestResult = friendlyQRZError(msg.err)
				cm.TestToast = friendlyQRZError(msg.err)
				applog.Error("QRZ.RU test failed", "error", msg.err.Error())
			} else if msg.ok {
				cm.qrzruTestResult = "OK — QRZ.RU connected"
				cm.TestToast = "QRZ.RU: connection verified"
				applog.Info("QRZ.RU test OK")
			} else {
				cm.qrzruTestResult = "No data returned"
				cm.TestToast = "QRZ.RU: connected, but lookup returned no data"
				applog.Warn("QRZ.RU test: no data returned")
			}
		default:
			cm.qrzTesting = false
			if msg.err != nil {
				cm.qrzTestResult = friendlyQRZError(msg.err)
				cm.TestToast = friendlyQRZError(msg.err)
				applog.Error("QRZ test failed", "error", msg.err.Error())
			} else if msg.ok {
				cm.qrzTestResult = "OK - QRZ.com connected"
				cm.TestToast = "QRZ: connection verified"
				applog.Info("QRZ test OK")
			} else {
				cm.qrzTestResult = "No data returned"
				cm.TestToast = "QRZ: connected, but lookup returned no data"
				applog.Warn("QRZ test: no data returned")
			}
		}

	case tea.KeyPressMsg:
		k := msg.String()
		if cm.qrzTesting || cm.hamqthTesting || cm.qrzruTesting {
			return cm, nil
		}
		// Shared navigation: Tab/Down, Shift+Tab/Up, and the Save & Back
		// button (Space/Enter saves through the same validation as Ctrl+S).
		if handled, cmd := cm.fm.onKey(msg, cm, func() tea.Cmd { return cm.trySave() }); handled {
			return cm, cmd
		}
		switch k {
		case "esc":
			cm.done = true
			cm.goBack = true
			return cm, nil
		case "ctrl+s", "\x13":
			return cm, cm.trySave()
		case " ", "space":
			// Space triggers Test buttons in parallel with Enter.
			if cm.fm.row == cmQRZTest || cm.fm.row == cmHamQTHTest || cm.fm.row == cmQRZRuTest {
				m2, c := cm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
				return m2.(*CallbookMenu), c
			}
			switch cm.fm.row {
			case cmBaseCall:
				cm.baseCallFallback = !cm.baseCallFallback
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmLogChk:
				cm.logEnabled = !cm.logEnabled
				if !cm.isPositionVisible(cm.fm.row) {
					cm.fm.fixFocus(cm)
				}
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmQRZChk:
				cm.qrzEnabled = !cm.qrzEnabled
				if !cm.isPositionVisible(cm.fm.row) {
					cm.fm.fixFocus(cm)
				}
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmWavelogChk:
				cm.wlEnabled = !cm.wlEnabled
				if !cm.isPositionVisible(cm.fm.row) {
					cm.fm.fixFocus(cm)
				}
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmHamQTHChk:
				cm.hamqthEnabled = !cm.hamqthEnabled
				if !cm.isPositionVisible(cm.fm.row) {
					cm.fm.fixFocus(cm)
				}
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmCallookChk:
				cm.callookEnabled = !cm.callookEnabled
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			case cmQRZRuChk:
				cm.qrzruEnabled = !cm.qrzruEnabled
				if !cm.isPositionVisible(cm.fm.row) {
					cm.fm.fixFocus(cm)
				}
				scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
				return cm, nil
			}
			cm.forwardToFocused(msg)
		case "tab", "down":
			_ = cm.fm.next(cm)
			scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
		case "shift+tab", "up":
			_ = cm.fm.prev(cm)
			scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
		case "enter":
			if cm.fm.row == cmQRZTest {
				if !cm.inetOnline {
					cm.qrzTestResult = "No internet connection"
					return cm, nil
				}
				user := strings.TrimSpace(cm.qrzUser.Value())
				pass := cm.qrzPass.Value()
				if user == "" || pass == "" {
					cm.qrzTestResult = "Username and password required"
					return cm, nil
				}
				cm.qrzTesting = true
				cm.qrzTestResult = "Testing..."
				return cm, func() tea.Msg {
					data, err := qrzcom.Lookup(user, pass, "SP9MOA")
					return callbookTestMsg{ok: err == nil && data != nil, err: err, provider: "qrz"}
				}
			}
			if cm.fm.row == cmHamQTHTest {
				if !cm.inetOnline {
					cm.hamqthTestResult = "No internet connection"
					return cm, nil
				}
				user := strings.TrimSpace(cm.hamqthUser.Value())
				pass := cm.hamqthPass.Value()
				if user == "" || pass == "" {
					cm.hamqthTestResult = "Username and password required"
					return cm, nil
				}
				cm.hamqthTesting = true
				cm.hamqthTestResult = "Testing..."
				return cm, func() tea.Msg {
					data, err := hamqth.Lookup(user, pass, "OK1HRA")
					return callbookTestMsg{ok: err == nil && data != nil, err: err, provider: "hamqth"}
				}
			}
			if cm.fm.row == cmQRZRuTest {
				if !cm.inetOnline {
					cm.qrzruTestResult = "No internet connection"
					return cm, nil
				}
				user := strings.TrimSpace(cm.qrzruUser.Value())
				pass := cm.qrzruPass.Value()
				if user == "" || pass == "" {
					cm.qrzruTestResult = "API login and password required"
					return cm, nil
				}
				cm.qrzruTesting = true
				cm.qrzruTestResult = "Testing..."
				return cm, func() tea.Msg {
					client := qrzru.NewClientWithPriority(user, pass, 35)
					data, err := client.Lookup("RA3ZZ")
					return callbookTestMsg{ok: err == nil && data != nil, err: err, provider: "qrzru"}
				}
			}
			cm.fm.next(cm)
			scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
		default:
			cm.forwardToFocused(msg)
			cm.vp, _ = cm.vp.Update(msg)
		}
	default:
		cm.forwardToFocused(msg)
	}
	return cm, nil
}

func (cm *CallbookMenu) forwardToFocused(msg tea.Msg) {
	switch cm.fm.row {
	case cmLogPriority:
		cm.logPriority, _ = cm.logPriority.Update(msg)
	case cmQRZUser:
		cm.qrzUser, _ = cm.qrzUser.Update(msg)
	case cmQRZPass:
		cm.qrzPass, _ = cm.qrzPass.Update(msg)
	case cmQRZPriority:
		cm.qrzPriority, _ = cm.qrzPriority.Update(msg)
	case cmHamQTHUser:
		cm.hamqthUser, _ = cm.hamqthUser.Update(msg)
	case cmHamQTHPass:
		cm.hamqthPass, _ = cm.hamqthPass.Update(msg)
	case cmHamQTHPriority:
		cm.hamqthPriority, _ = cm.hamqthPriority.Update(msg)
	case cmCallookPriority:
		cm.callookPriority, _ = cm.callookPriority.Update(msg)
	case cmQRZRuUser:
		cm.qrzruUser, _ = cm.qrzruUser.Update(msg)
	case cmQRZRuPass:
		cm.qrzruPass, _ = cm.qrzruPass.Update(msg)
	case cmQRZRuPriority:
		cm.qrzruPriority, _ = cm.qrzruPriority.Update(msg)
	case cmWavelogPriority:
		cm.wlPriority, _ = cm.wlPriority.Update(msg)
	}
}

func (cm *CallbookMenu) isPositionVisible(pos int) bool {
	switch pos {
	case cmBaseCall, cmQRZChk, cmHamQTHChk, cmCallookChk, cmQRZRuChk, cmLogChk, cmWavelogChk:
		return true
	case cmLogPriority:
		return cm.logEnabled
	case cmQRZUser, cmQRZPass, cmQRZPriority, cmQRZTest:
		return cm.qrzEnabled
	case cmHamQTHUser, cmHamQTHPass, cmHamQTHPriority, cmHamQTHTest:
		return cm.hamqthEnabled
	case cmCallookPriority:
		return cm.callookEnabled
	case cmQRZRuUser, cmQRZRuPass, cmQRZRuPriority, cmQRZRuTest:
		return cm.qrzruEnabled
	case cmWavelogPriority:
		return cm.wlEnabled && cm.wlConfigured
	}
	return true
}

// focusableRows implementation.
func (cm *CallbookMenu) rowCount() int         { return cmMax }
func (cm *CallbookMenu) rowVisible(i int) bool { return cm.isPositionVisible(i) }

func (cm *CallbookMenu) blurAll() {
	blurTextinputs(&cm.logPriority, &cm.qrzUser, &cm.qrzPass, &cm.qrzPriority,
		&cm.hamqthUser, &cm.hamqthPass, &cm.hamqthPriority,
		&cm.callookPriority, &cm.qrzruUser, &cm.qrzruPass, &cm.qrzruPriority,
		&cm.wlPriority)
}

func (cm *CallbookMenu) focusRow(i int) tea.Cmd {
	switch i {
	case cmLogPriority:
		cm.logPriority.Focus()
	case cmQRZUser:
		cm.qrzUser.Focus()
	case cmQRZPass:
		cm.qrzPass.Focus()
	case cmQRZPriority:
		cm.qrzPriority.Focus()
	case cmHamQTHUser:
		cm.hamqthUser.Focus()
	case cmHamQTHPass:
		cm.hamqthPass.Focus()
	case cmHamQTHPriority:
		cm.hamqthPriority.Focus()
	case cmCallookPriority:
		cm.callookPriority.Focus()
	case cmQRZRuUser:
		cm.qrzruUser.Focus()
	case cmQRZRuPass:
		cm.qrzruPass.Focus()
	case cmQRZRuPriority:
		cm.qrzruPriority.Focus()
	case cmWavelogPriority:
		cm.wlPriority.Focus()
	}
	return nil
}

// trySave validates the enabled callbook providers and closes the menu.
// Used by both Ctrl+S and the Save & Back button.
func (cm *CallbookMenu) trySave() tea.Cmd {
	// Validate logbook priority.
	lps := strings.TrimSpace(cm.logPriority.Value())
	if lps != "" {
		p, err := strconv.Atoi(lps)
		if err != nil || p < 0 || p > 100 {
			cm.SaveError = "Callbook: priority must be 0\u2013100"
			return nil
		}
	}
	if cm.qrzEnabled {
		if strings.TrimSpace(cm.qrzUser.Value()) == "" {
			cm.SaveError = "QRZ: username is required when enabled"
			return nil
		}
		if cm.qrzPass.Value() == "" {
			cm.SaveError = "QRZ: password is required when enabled"
			return nil
		}
		ps := strings.TrimSpace(cm.qrzPriority.Value())
		if ps != "" {
			p, err := strconv.Atoi(ps)
			if err != nil || p < 0 || p > 100 {
				cm.SaveError = "Callbook: priority must be 0\u2013100"
				return nil
			}
		}
	}
	if cm.hamqthEnabled {
		if strings.TrimSpace(cm.hamqthUser.Value()) == "" {
			cm.SaveError = "HamQTH: username is required when enabled"
			return nil
		}
		if cm.hamqthPass.Value() == "" {
			cm.SaveError = "HamQTH: password is required when enabled"
			return nil
		}
		ps := strings.TrimSpace(cm.hamqthPriority.Value())
		if ps != "" {
			p, err := strconv.Atoi(ps)
			if err != nil || p < 0 || p > 100 {
				cm.SaveError = "Callbook: priority must be 0\u2013100"
				return nil
			}
		}
	}
	if cm.qrzruEnabled {
		if strings.TrimSpace(cm.qrzruUser.Value()) == "" {
			cm.SaveError = "QRZ.RU: API login is required when enabled"
			return nil
		}
		if cm.qrzruPass.Value() == "" {
			cm.SaveError = "QRZ.RU: API password is required when enabled"
			return nil
		}
		ps := strings.TrimSpace(cm.qrzruPriority.Value())
		if ps != "" {
			p, err := strconv.Atoi(ps)
			if err != nil || p < 0 || p > 100 {
				cm.SaveError = "Callbook: priority must be 0\u2013100"
				return nil
			}
		}
	}
	cm.done = true
	cm.saved = true
	return nil
}

func (cm *CallbookMenu) View() tea.View {
	if cm.done {
		return tea.NewView("")
	}
	w := cm.width
	if w < 40 {
		w = 80
	}
	h := cm.height
	if h < 10 {
		h = 24
	}

	lineW := w - 2 - 4
	if lineW < 36 {
		lineW = 36
	}
	if lineW > partnerMapMaxW-4 {
		lineW = partnerMapMaxW - 4
	}

	// Compute dimensions first.
	boxW := w - 2
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // viewport width inside menuBoxStyle
	if vpW < 20 {
		vpW = 20
	}

	var b strings.Builder

	// --- Info box ---
	// Wrap text to fit viewport minus border overhead.
	infoMaxW := vpW - 2 // 2 for border (no padding on this box)
	if infoMaxW < 30 {
		infoMaxW = 30
	}
	infoText := "Callsign lookup providers with priority-based " +
		"search order (higher = tried first). Login details " +
		"are safe on shared stations \u2014 read-only, encrypted."
	infoBox(&b, infoText, infoMaxW)

	// --- Base call fallback ---
	checkboxRow(&b, lineW, cm.fm.row == cmBaseCall, "Base call fallback:", cm.baseCallFallback, "Fallback to base callsign", false, callbookRows)

	// --- QRZ.com ---
	checkboxRow(&b, lineW, cm.fm.row == cmQRZChk, "QRZ.com:", cm.qrzEnabled, "Paid, XML subscription required", false, callbookRows)

	if cm.qrzEnabled {
		b.WriteString(padOrTrunc(cm.renderField(cmQRZUser, "  Username:", &cm.qrzUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZPass, "  Password:", &cm.qrzPass, true), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZPriority, "  Priority:", &cm.qrzPriority, false), lineW))

		// Test button
		b.WriteString("\n")
		cm.providerTestButton(&b, lineW, cmQRZTest)
	}

	// --- HamQTH ---
	checkboxRow(&b, lineW, cm.fm.row == cmHamQTHChk, "HamQTH:", cm.hamqthEnabled, "Recommended free global callbook", false, callbookRows)

	if cm.hamqthEnabled {
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHUser, "  Username:", &cm.hamqthUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHPass, "  Password:", &cm.hamqthPass, true), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHPriority, "  Priority:", &cm.hamqthPriority, false), lineW))

		// Test button
		b.WriteString("\n")
		cm.providerTestButton(&b, lineW, cmHamQTHTest)
	}

	// --- Callook.info ---
	checkboxRow(&b, lineW, cm.fm.row == cmCallookChk, "Callook.info:", cm.callookEnabled, "Free, US callsigns only", false, callbookRows)

	if cm.callookEnabled {
		b.WriteString(padOrTrunc(cm.renderField(cmCallookPriority, "  Priority:", &cm.callookPriority, false), lineW))
		b.WriteString("\n")
	}

	// --- QRZ.RU ---
	checkboxRow(&b, lineW, cm.fm.row == cmQRZRuChk, "QRZ.RU:", cm.qrzruEnabled, "Free, Russia, Eastern Europe and surroundings", false, callbookRows)

	if cm.qrzruEnabled {
		b.WriteString(padOrTrunc(cm.renderField(cmQRZRuUser, "  API login:", &cm.qrzruUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZRuPass, "  API password:", &cm.qrzruPass, true), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZRuPriority, "  Priority:", &cm.qrzruPriority, false), lineW))

		// Test button
		b.WriteString("\n")
		cm.providerTestButton(&b, lineW, cmQRZRuTest)
	}

	// --- Local Logbook ---
	checkboxRow(&b, lineW, cm.fm.row == cmLogChk, "Logbook:", cm.logEnabled, "Offline fallback, past contacts may be stale", false, callbookRows)

	if cm.logEnabled {
		b.WriteString(padOrTrunc(cm.renderField(cmLogPriority, "  Priority:", &cm.logPriority, false), lineW))
		b.WriteString("\n")
	}

	// --- Wavelog ---
	if cm.wlConfigured {
		checkboxRow(&b, lineW, cm.fm.row == cmWavelogChk, "Wavelog:", cm.wlEnabled, "Low priority — data may come from QRZ/HamQTH/QRZ.RU", false, callbookRows)

		if cm.wlEnabled {
			b.WriteString(padOrTrunc(cm.renderField(cmWavelogPriority, "  Priority:", &cm.wlPriority, false), lineW))
			b.WriteString("\n")
		}
	} // wlConfigured

	// Save & Back button at the end of the menu.
	b.WriteString("\n")
	b.WriteString(cm.fm.btn.line("Save & Back", lineW))

	body := b.String()
	if body == "" {
		body = " "
	}

	contentH := contentHeight(h)
	if contentH < 8 {
		contentH = 8
	}
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	cm.vp.SetWidth(vpW)
	cm.vp.SetHeight(vpH)
	if body != cm.lastBodyContent {
		cm.vp.SetContent(body)
		cm.lastBodyContent = body
		cm.vp.GotoTop()
		scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
	}
	if cm.vp.PastBottom() {
		scrollViewportToFraction(&cm.vp, cm.fm.scrollFraction(cm))
	}
	header := S.Title.Width(boxW).Render("Configuration \u2014 Callbook")
	vpContent := cm.vp.View()
	if hint := scrollHint(cm.vp); hint != "" {
		hintLine := DimStyle.Width(vpW).Render(hint)
		vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	}
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, "", box))
}

// providerTestButton renders a "[ Test Connection ]" row for a callbook
// provider. Offline state dims the button; otherwise it uses the shared
// button renderer with the (Space)/Enter focus marker.
func (cm *CallbookMenu) providerTestButton(b *strings.Builder, w, testPos int) {
	btnText := "[ Test Connection ]"
	if !cm.inetOnline {
		b.WriteString(padOrTrunc("    "+DimStyle.Render(btnText)+" "+DimStyle.Render("(offline)"), w))
		b.WriteString("\n")
		return
	}
	buttonRow(b, w, cm.fm.row == testPos, btnText)
}

func (cm *CallbookMenu) renderField(pos int, label string, ti *textinput.Model, hidden bool) string {
	prefix := "  "
	if cm.fm.row == pos {
		prefix = S.FormPrefixOn.Render("> ")
	}
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	valW := 28
	val := ""
	if hidden {
		val = strings.Repeat("•", len(ti.Value()))
	} else {
		val = ti.Value()
	}
	if cm.fm.row == pos {
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		val = CursorStyle.Width(valW).MaxWidth(valW).Render(ti.View())
	} else {
		val = ValueStyle.Width(valW).MaxWidth(valW).Render(val)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

// wrapLines splits text into lines no wider than maxW at word boundaries.
func wrapLines(text string, maxW int) []string {
	if maxW <= 0 {
		return []string{text}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	var lines []string
	words := strings.Fields(text)
	var cur strings.Builder
	for _, w := range words {
		if cur.Len() == 0 {
			cur.WriteString(w)
			continue
		}
		if lipgloss.Width(cur.String()+" "+w) > maxW {
			lines = append(lines, cur.String())
			cur.Reset()
			cur.WriteString(w)
		} else {
			cur.WriteString(" ")
			cur.WriteString(w)
		}
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	return lines
}

// ToConfig writes the callbook menu state back into the config.
func (cm *CallbookMenu) ToConfig(cfg *config.Config) {
	cfg.Integrations.Callbook.BaseCallFallback = cm.baseCallFallback
	cfg.Integrations.Callbook.Logbook.Enabled = cm.logEnabled
	ps := strings.TrimSpace(cm.logPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.Logbook.Priority = p
		}
	}
	cfg.Integrations.Callbook.QRZ.Enabled = cm.qrzEnabled
	cfg.Integrations.Callbook.QRZ.User = strings.TrimSpace(cm.qrzUser.Value())
	cfg.Integrations.Callbook.QRZ.Pass = cm.qrzPass.Value()
	ps = strings.TrimSpace(cm.qrzPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.QRZ.Priority = p
		}
	} else {
		cfg.Integrations.Callbook.QRZ.Priority = 50
	}
	cfg.Integrations.Callbook.HamQTH.Enabled = cm.hamqthEnabled
	cfg.Integrations.Callbook.HamQTH.User = strings.TrimSpace(cm.hamqthUser.Value())
	cfg.Integrations.Callbook.HamQTH.Pass = cm.hamqthPass.Value()
	ps = strings.TrimSpace(cm.hamqthPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.HamQTH.Priority = p
		}
	} else {
		cfg.Integrations.Callbook.HamQTH.Priority = 45
	}
	cfg.Integrations.Callbook.Callook.Enabled = cm.callookEnabled
	ps = strings.TrimSpace(cm.callookPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.Callook.Priority = p
		}
	} else {
		cfg.Integrations.Callbook.Callook.Priority = 30
	}
	cfg.Integrations.Callbook.QRZRu.Enabled = cm.qrzruEnabled
	cfg.Integrations.Callbook.QRZRu.User = strings.TrimSpace(cm.qrzruUser.Value())
	cfg.Integrations.Callbook.QRZRu.Pass = cm.qrzruPass.Value()
	ps = strings.TrimSpace(cm.qrzruPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.QRZRu.Priority = p
		}
	} else {
		cfg.Integrations.Callbook.QRZRu.Priority = 35
	}
	cfg.Integrations.Callbook.Wavelog.Enabled = cm.wlEnabled
	ps = strings.TrimSpace(cm.wlPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.Callbook.Wavelog.Priority = p
		}
	} else {
		cfg.Integrations.Callbook.Wavelog.Priority = 10
	}
}
