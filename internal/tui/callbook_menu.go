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
)

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

	// Wavelog provider
	wlEnabled    bool
	wlConfigured bool // true only when a logbook has Wavelog configured
	wlPriority   textinput.Model

	focus  int
	done   bool
	saved  bool
	goBack bool
	width  int
	height int

	// saveError is set when Ctrl+S is blocked by validation.
	SaveError string

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastBodyContent string
}

const (
	cmBaseCall        = 0
	cmLogChk          = 1
	cmLogPriority     = 2
	cmQRZChk          = 3
	cmQRZUser         = 4
	cmQRZPass         = 5
	cmQRZPriority     = 6
	cmQRZTest         = 7
	cmHamQTHChk       = 8
	cmHamQTHUser      = 9
	cmHamQTHPass      = 10
	cmHamQTHPriority  = 11
	cmHamQTHTest      = 12
	cmWavelogChk      = 13
	cmWavelogPriority = 14
	cmMax             = 15
)

func NewCallbookMenu(cfg *config.Config) *CallbookMenu {
	logPriority := newTextinput()
	logPriority.CharLimit = 5
	logPriority.SetWidth(6)
	logPriority.Placeholder = "100"
	if cfg.Integrations.LogbookCallbook.Priority != 0 {
		logPriority.SetValue(strconv.Itoa(cfg.Integrations.LogbookCallbook.Priority))
	} else {
		logPriority.SetValue("100") // default: tried before online providers
	}
	logEnabled := cfg.Integrations.LogbookCallbook.Enabled
	// Default to enabled on first run (Priority=0 means never configured).
	if !logEnabled && cfg.Integrations.LogbookCallbook.Priority == 0 {
		logEnabled = true
	}

	qrzUser := newTextinput()
	qrzUser.CharLimit = 30
	qrzUser.SetWidth(28)
	qrzUser.Placeholder = "QRZ.com username"
	qrzUser.SetValue(cfg.Integrations.QRZ.User)

	qrzPass := newTextinput()
	qrzPass.CharLimit = 40
	qrzPass.SetWidth(28)
	qrzPass.Placeholder = "QRZ.com password"
	qrzPass.EchoMode = textinput.EchoPassword
	qrzPass.EchoCharacter = '*'
	qrzPass.SetValue(cfg.Integrations.QRZ.Pass)

	qrzPriority := newTextinput()
	qrzPriority.CharLimit = 5
	qrzPriority.SetWidth(6)
	qrzPriority.Placeholder = "50"
	qrzPriority.SetValue(strconv.Itoa(cfg.Integrations.QRZ.Priority))
	if cfg.Integrations.QRZ.Priority == 0 {
		qrzPriority.SetValue("50")
	}

	// HamQTH provider.
	hamqthUser := newTextinput()
	hamqthUser.CharLimit = 30
	hamqthUser.SetWidth(28)
	hamqthUser.Placeholder = "HamQTH username"
	hamqthUser.SetValue(cfg.Integrations.HamQTH.User)

	hamqthPass := newTextinput()
	hamqthPass.CharLimit = 40
	hamqthPass.SetWidth(28)
	hamqthPass.Placeholder = "HamQTH password"
	hamqthPass.EchoMode = textinput.EchoPassword
	hamqthPass.EchoCharacter = '*'
	hamqthPass.SetValue(cfg.Integrations.HamQTH.Pass)

	hamqthPriority := newTextinput()
	hamqthPriority.CharLimit = 5
	hamqthPriority.SetWidth(6)
	hamqthPriority.Placeholder = "45"
	hamqthPriority.SetValue(strconv.Itoa(cfg.Integrations.HamQTH.Priority))
	if cfg.Integrations.HamQTH.Priority == 0 {
		hamqthPriority.SetValue("45")
	}

	// Default base-call fallback to true on first run (fresh config).
	baseFallback := cfg.Integrations.Callbook.BaseCallFallback
	if cfg.Integrations.LogbookCallbook.Priority == 0 && !cfg.Integrations.LogbookCallbook.Enabled {
		baseFallback = true
	}

	// Wavelog callbook provider.
	wlPriority := newTextinput()
	wlPriority.CharLimit = 5
	wlPriority.SetWidth(6)
	wlPriority.Placeholder = "10"
	wlPriority.SetValue(strconv.Itoa(cfg.Integrations.WavelogCallbook.Priority))
	if cfg.Integrations.WavelogCallbook.Priority == 0 {
		wlPriority.SetValue("10")
	}
	wlEnabled := cfg.Integrations.WavelogCallbook.Enabled
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
		qrzEnabled:       cfg.Integrations.QRZ.Enabled,
		qrzUser:          qrzUser,
		qrzPass:          qrzPass,
		qrzPriority:      qrzPriority,
		hamqthEnabled:    cfg.Integrations.HamQTH.Enabled,
		hamqthUser:       hamqthUser,
		hamqthPass:       hamqthPass,
		hamqthPriority:   hamqthPriority,
		wlEnabled:        wlEnabled,
		wlConfigured:     wlConfigured,
		wlPriority:       wlPriority,
		focus:            0,
	}
}

func (cm *CallbookMenu) Init() tea.Cmd { return nil }

func (cm *CallbookMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cm.width, cm.height = msg.Width, msg.Height

	case callbookTestMsg:
		if msg.provider == "hamqth" {
			cm.hamqthTesting = false
			if msg.err != nil {
				cm.hamqthTestResult = friendlyHTestError(msg.err)
				applog.Error("HamQTH test failed", "error", msg.err.Error())
			} else if msg.ok {
				cm.hamqthTestResult = "OK — HamQTH connected"
				applog.Info("HamQTH test OK")
			} else {
				cm.hamqthTestResult = "Connected OK — OK1HRA not found (API works)"
				applog.Warn("HamQTH test: no data returned")
			}
		} else {
			cm.qrzTesting = false
			if msg.err != nil {
				cm.qrzTestResult = friendlyQRZError(msg.err)
				applog.Error("QRZ test failed", "error", msg.err.Error())
			} else if msg.ok {
				cm.qrzTestResult = "OK - QRZ.com connected"
				applog.Info("QRZ test OK")
			} else {
				cm.qrzTestResult = "No data returned"
				applog.Warn("QRZ test: no data returned")
			}
		}

	case tea.KeyPressMsg:
		k := msg.String()
		if cm.qrzTesting || cm.hamqthTesting {
			return cm, nil
		}
		switch k {
		case "esc":
			cm.done = true
			cm.goBack = true
			return cm, nil
		case "ctrl+s", "\x13":
			// Validate logbook priority.
			lps := strings.TrimSpace(cm.logPriority.Value())
			if lps != "" {
				p, err := strconv.Atoi(lps)
				if err != nil || p < 0 || p > 100 {
					cm.SaveError = "Priority must be 0\u2013100"
					return cm, nil
				}
			}
			if cm.qrzEnabled {
				if strings.TrimSpace(cm.qrzUser.Value()) == "" {
					cm.SaveError = "QRZ username is required when enabled"
					return cm, nil
				}
				if cm.qrzPass.Value() == "" {
					cm.SaveError = "QRZ password is required when enabled"
					return cm, nil
				}
				ps := strings.TrimSpace(cm.qrzPriority.Value())
				if ps != "" {
					p, err := strconv.Atoi(ps)
					if err != nil || p < 0 || p > 100 {
						cm.SaveError = "Priority must be 0\u2013100"
						return cm, nil
					}
				}
			}
			if cm.hamqthEnabled {
				if strings.TrimSpace(cm.hamqthUser.Value()) == "" {
					cm.SaveError = "HamQTH username is required when enabled"
					return cm, nil
				}
				if cm.hamqthPass.Value() == "" {
					cm.SaveError = "HamQTH password is required when enabled"
					return cm, nil
				}
				ps := strings.TrimSpace(cm.hamqthPriority.Value())
				if ps != "" {
					p, err := strconv.Atoi(ps)
					if err != nil || p < 0 || p > 100 {
						cm.SaveError = "Priority must be 0\u2013100"
						return cm, nil
					}
				}
			}
			cm.done = true
			cm.saved = true
			return cm, nil
		case " ", "space":
			switch cm.focus {
			case cmBaseCall:
				cm.baseCallFallback = !cm.baseCallFallback
				cm.autoScrollViewport()
				return cm, nil
			case cmLogChk:
				cm.logEnabled = !cm.logEnabled
				if !cm.isPositionVisible(cm.focus) {
					cm.fixFocus()
				}
				cm.autoScrollViewport()
				return cm, nil
			case cmQRZChk:
				cm.qrzEnabled = !cm.qrzEnabled
				if !cm.isPositionVisible(cm.focus) {
					cm.fixFocus()
				}
				cm.autoScrollViewport()
				return cm, nil
			case cmWavelogChk:
				cm.wlEnabled = !cm.wlEnabled
				if !cm.isPositionVisible(cm.focus) {
					cm.fixFocus()
				}
				cm.autoScrollViewport()
				return cm, nil
			case cmHamQTHChk:
				cm.hamqthEnabled = !cm.hamqthEnabled
				if !cm.isPositionVisible(cm.focus) {
					cm.fixFocus()
				}
				cm.autoScrollViewport()
				return cm, nil
			}
			cm.forwardToFocused(msg)
		case "tab", "down":
			cm.next()
			cm.autoScrollViewport()
		case "shift+tab", "up":
			cm.prev()
			cm.autoScrollViewport()
		case "enter":
			if cm.focus == cmQRZTest {
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
			if cm.focus == cmHamQTHTest {
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
			cm.next()
			cm.autoScrollViewport()
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
	switch cm.focus {
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
	case cmWavelogPriority:
		cm.wlPriority, _ = cm.wlPriority.Update(msg)
	}
}

func (cm *CallbookMenu) next() {
	for {
		cm.focus = wrapNext(cm.focus, cmMax)
		if cm.isPositionVisible(cm.focus) {
			break
		}
	}
	cm.blurAll()
	cm.focusField()
}

func (cm *CallbookMenu) prev() {
	for {
		cm.focus = wrapPrev(cm.focus, cmMax)
		if cm.isPositionVisible(cm.focus) {
			break
		}
	}
	cm.blurAll()
	cm.focusField()
}

func (cm *CallbookMenu) isPositionVisible(pos int) bool {
	switch pos {
	case cmBaseCall, cmLogChk, cmQRZChk, cmHamQTHChk, cmWavelogChk:
		return true
	case cmLogPriority:
		return cm.logEnabled
	case cmQRZUser, cmQRZPass, cmQRZPriority, cmQRZTest:
		return cm.qrzEnabled
	case cmHamQTHUser, cmHamQTHPass, cmHamQTHPriority, cmHamQTHTest:
		return cm.hamqthEnabled
	case cmWavelogPriority:
		return cm.wlEnabled && cm.wlConfigured
	}
	return true
}

func (cm *CallbookMenu) fixFocus() {
	if cm.isPositionVisible(cm.focus) {
		return
	}
	cm.next()
}

func (cm *CallbookMenu) blurAll() {
	blurTextinputs(&cm.logPriority, &cm.qrzUser, &cm.qrzPass, &cm.qrzPriority,
		&cm.hamqthUser, &cm.hamqthPass, &cm.hamqthPriority, &cm.wlPriority)
}

func (cm *CallbookMenu) focusField() {
	switch cm.focus {
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
	case cmWavelogPriority:
		cm.wlPriority.Focus()
	}
}

func (cm *CallbookMenu) scrollFraction() float64 {
	visible := 0
	rank := -1
	for i := 0; i < cmMax; i++ {
		if cm.isPositionVisible(i) {
			visible++
		}
		if i == cm.focus {
			rank = visible
		}
	}
	if visible <= 1 || rank <= 0 {
		return 0
	}
	return float64(rank-1) / float64(visible-1)
}

func (cm *CallbookMenu) autoScrollViewport() {
	total := cm.vp.TotalLineCount()
	visible := cm.vp.VisibleLineCount()
	if total <= visible {
		cm.vp.SetYOffset(0)
		return
	}
	frac := cm.scrollFraction()
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := int(float64(maxOffset) * frac)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	cm.vp.SetYOffset(offset)
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
	infoLines := wrapLines(infoText, infoMaxW)
	var infoContent strings.Builder
	for i, line := range infoLines {
		infoContent.WriteString(DimStyle.Render(line))
		if i < len(infoLines)-1 {
			infoContent.WriteString("\n")
		}
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.Border)
	infoBox := boxStyle.Render(infoContent.String())
	b.WriteString(infoBox)

	b.WriteString("\n")

	// --- Base call fallback ---
	baseCb := "[ ]"
	if cm.baseCallFallback {
		baseCb = "[x]"
	}
	basePrefix := "  "
	baseLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Base call fallback:")
	baseInfo := DimStyle.Render("(look up SP9MOA if SP9MOA/P not found)")
	if cm.focus == cmBaseCall {
		basePrefix = S.FormPrefixOn.Render("> ")
		baseLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Base call fallback:")
		baseCb = CursorStyle.Render(baseCb) + " " + DimStyle.Render("(Space)") + " " + baseInfo
	} else {
		baseCb = baseCb + " " + baseInfo
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, basePrefix, baseLabel, " ", baseCb),
		lineW))

	b.WriteString("\n")

	// --- Local Logbook ---
	logCb := "[ ]"
	if cm.logEnabled {
		logCb = "[x]"
	}
	logPrefix := "  "
	logLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Logbook:")
	logInfo := DimStyle.Render("(local search in past contacts)")
	if cm.focus == cmLogChk {
		logPrefix = S.FormPrefixOn.Render("> ")
		logLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Logbook:")
		logCb = CursorStyle.Render(logCb) + " " + DimStyle.Render("(Space)") + " " + logInfo
	} else {
		logCb = logCb + " " + logInfo
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, logPrefix, logLabel, " ", logCb),
		lineW))

	if cm.logEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmLogPriority, "  Priority:", &cm.logPriority, false), lineW))
	}

	b.WriteString("\n")
	b.WriteString("")

	// --- QRZ.com ---
	qrzCheckbox := "[ ]"
	if cm.qrzEnabled {
		qrzCheckbox = "[x]"
	}
	qrzPrefix := "  "
	qrzLabel := S.FormLabelWide.Align(lipgloss.Left).Render("QRZ.com:")
	qrzInfo := DimStyle.Render("(requires XML subscription)")
	if cm.focus == cmQRZChk {
		qrzPrefix = S.FormPrefixOn.Render("> ")
		qrzLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("QRZ.com:")
		qrzCheckbox = CursorStyle.Render(qrzCheckbox) + " " + DimStyle.Render("(Space)") + " " + qrzInfo
	} else {
		qrzCheckbox = qrzCheckbox + " " + qrzInfo
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, qrzPrefix, qrzLabel, " ", qrzCheckbox),
		lineW))

	if cm.qrzEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZUser, "  Username:", &cm.qrzUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZPass, "  Password:", &cm.qrzPass, true), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmQRZPriority, "  Priority:", &cm.qrzPriority, false), lineW))

		// Test button
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !cm.inetOnline {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("(offline)")
		} else if cm.focus == cmQRZTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, lineW))

		if cm.qrzTestResult != "" {
			b.WriteString("\n    ")
			if cm.qrzTesting {
				b.WriteString(DimStyle.Render(cm.qrzTestResult))
			} else if strings.HasPrefix(cm.qrzTestResult, "OK") {
				b.WriteString(SuccessStyle.Render(cm.qrzTestResult))
			} else {
				b.WriteString(ErrorStyle.Render(cm.qrzTestResult))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString("")

	// --- HamQTH ---
	hamqthCb := "[ ]"
	if cm.hamqthEnabled {
		hamqthCb = "[x]"
	}
	hamqthPrefix := "  "
	hamqthLabel := S.FormLabelWide.Align(lipgloss.Left).Render("HamQTH:")
	hamqthInfo := DimStyle.Render("(free service)")
	if cm.focus == cmHamQTHChk {
		hamqthPrefix = S.FormPrefixOn.Render("> ")
		hamqthLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("HamQTH:")
		hamqthCb = CursorStyle.Render(hamqthCb) + " " + DimStyle.Render("(Space)") + " " + hamqthInfo
	} else {
		hamqthCb = hamqthCb + " " + hamqthInfo
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, hamqthPrefix, hamqthLabel, " ", hamqthCb),
		lineW))

	if cm.hamqthEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHUser, "  Username:", &cm.hamqthUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHPass, "  Password:", &cm.hamqthPass, true), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(cm.renderField(cmHamQTHPriority, "  Priority:", &cm.hamqthPriority, false), lineW))

		// Test button
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !cm.inetOnline {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("(offline)")
		} else if cm.focus == cmHamQTHTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, lineW))

		if cm.hamqthTestResult != "" {
			b.WriteString("\n    ")
			if cm.hamqthTesting {
				b.WriteString(DimStyle.Render(cm.hamqthTestResult))
			} else if strings.HasPrefix(cm.hamqthTestResult, "OK") {
				b.WriteString(SuccessStyle.Render(cm.hamqthTestResult))
			} else {
				b.WriteString(ErrorStyle.Render(cm.hamqthTestResult))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString("")

	// --- Wavelog ---
	if cm.wlConfigured {
		wlCb := "[ ]"
		if cm.wlEnabled {
			wlCb = "[x]"
		}
		wlPrefix := "  "
		wlLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Wavelog:")
		wlInfo := DimStyle.Render("(requires Wavelog integration enabled per logbook)")
		if cm.focus == cmWavelogChk {
			wlPrefix = S.FormPrefixOn.Render("> ")
			wlLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Wavelog:")
			wlCb = CursorStyle.Render(wlCb) + " " + DimStyle.Render("(Space)") + " " + wlInfo
		} else {
			wlCb = wlCb + " " + wlInfo
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, wlPrefix, wlLabel, " ", wlCb),
			lineW))

		if cm.wlEnabled {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(cm.renderField(cmWavelogPriority, "  Priority:", &cm.wlPriority, false), lineW))
		}
	} // wlConfigured

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
		cm.autoScrollViewport()
	}
	if cm.vp.PastBottom() {
		cm.autoScrollViewport()
	}
	header := S.Title.Width(boxW).Render("Configuration \u2014 Callbook")
	vpContent := cm.vp.View()
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, "", box))
}

func (cm *CallbookMenu) renderField(pos int, label string, ti *textinput.Model, hidden bool) string {
	prefix := "  "
	if cm.focus == pos {
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
	if cm.focus == pos {
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
	cfg.Integrations.LogbookCallbook.Enabled = cm.logEnabled
	ps := strings.TrimSpace(cm.logPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.LogbookCallbook.Priority = p
		}
	}
	cfg.Integrations.QRZ.Enabled = cm.qrzEnabled
	cfg.Integrations.QRZ.User = strings.TrimSpace(cm.qrzUser.Value())
	cfg.Integrations.QRZ.Pass = cm.qrzPass.Value()
	ps = strings.TrimSpace(cm.qrzPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.QRZ.Priority = p
		}
	} else {
		cfg.Integrations.QRZ.Priority = 50
	}
	cfg.Integrations.HamQTH.Enabled = cm.hamqthEnabled
	cfg.Integrations.HamQTH.User = strings.TrimSpace(cm.hamqthUser.Value())
	cfg.Integrations.HamQTH.Pass = cm.hamqthPass.Value()
	ps = strings.TrimSpace(cm.hamqthPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.HamQTH.Priority = p
		}
	} else {
		cfg.Integrations.HamQTH.Priority = 45
	}
	cfg.Integrations.WavelogCallbook.Enabled = cm.wlEnabled
	ps = strings.TrimSpace(cm.wlPriority.Value())
	if ps != "" {
		if p, err := strconv.Atoi(ps); err == nil {
			cfg.Integrations.WavelogCallbook.Priority = p
		}
	} else {
		cfg.Integrations.WavelogCallbook.Priority = 10
	}
}
