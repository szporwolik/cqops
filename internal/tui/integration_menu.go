package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
)

type IntegrationMenu struct {
	// DXC
	dxcEnabled bool
	dxcHost    textinput.Model
	dxcPort    textinput.Model
	dxcLogin   textinput.Model

	// QRZ
	qrzEnabled    bool
	qrzUser       textinput.Model
	qrzPass       textinput.Model
	qrzTesting    bool
	qrzTestResult string
	inetOnline    bool

	focus  int
	done   bool
	saved  bool
	goBack bool
	width  int
	height int

	// saveError is set when Ctrl+S is blocked by validation.
	// The parent reads it to show a toast, then clears it.
	SaveError string
}

const (
	imDXCChk   = 0
	imDXCHost  = 1
	imDXCPort  = 2
	imDXCLogin = 3
	imQRZChk   = 4
	imQRZUser  = 5
	imQRZPass  = 6
	imQRZTest  = 7
	imMax      = 8
)

type callbookTestMsg struct {
	ok  bool
	err error
}

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
	dxcHost := newTextinput()
	dxcHost.CharLimit = 60
	dxcHost.SetWidth(28)
	dxcHost.Placeholder = "dxspider.co.uk"
	if cfg.Integrations.DXC.Host != "" {
		dxcHost.SetValue(cfg.Integrations.DXC.Host)
	} else {
		dxcHost.SetValue("dxspider.co.uk")
	}

	dxcPort := newTextinput()
	dxcPort.CharLimit = 6
	dxcPort.SetWidth(28)
	dxcPort.Placeholder = "7300"
	if cfg.Integrations.DXC.Port != "" {
		dxcPort.SetValue(cfg.Integrations.DXC.Port)
	} else {
		dxcPort.SetValue("7300")
	}

	dxcLogin := newTextinput()
	dxcLogin.CharLimit = 20
	dxcLogin.SetWidth(28)
	dxcLogin.Placeholder = "callsign"
	if cfg.Integrations.DXC.Login != "" {
		dxcLogin.SetValue(cfg.Integrations.DXC.Login)
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

	return &IntegrationMenu{
		dxcEnabled: cfg.Integrations.DXC.Enabled,
		dxcHost:    dxcHost,
		dxcPort:    dxcPort,
		dxcLogin:   dxcLogin,
		qrzEnabled: cfg.Integrations.QRZ.Enabled,
		qrzUser:    qrzUser,
		qrzPass:    qrzPass,
		focus:      0,
	}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height

	case callbookTestMsg:
		im.qrzTesting = false
		if msg.err != nil {
			im.qrzTestResult = friendlyQRZError(msg.err)
			applog.Error("QRZ test failed", "error", msg.err.Error())
		} else if msg.ok {
			im.qrzTestResult = "OK - QRZ.com connected"
			applog.Info("QRZ test OK")
		} else {
			im.qrzTestResult = "No data returned"
			applog.Warn("QRZ test: no data returned")
		}

	case tea.KeyPressMsg:
		k := msg.String()
		if im.qrzTesting {
			return im, nil
		}
		switch k {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case "ctrl+s", "\x13":
			// Validate DXC fields when DXC is enabled.
			if im.dxcEnabled {
				if strings.TrimSpace(im.dxcHost.Value()) == "" {
					im.SaveError = "DXC host (server) is required when DXC is enabled"
					return im, nil
				}
				if strings.TrimSpace(im.dxcPort.Value()) == "" {
					im.SaveError = "DXC port is required when DXC is enabled"
					return im, nil
				}
				if strings.TrimSpace(im.dxcLogin.Value()) == "" {
					im.SaveError = "DXC login (callsign) is required when DXC is enabled"
					return im, nil
				}
			}
			// Validate QRZ fields when QRZ is enabled.
			if im.qrzEnabled {
				if strings.TrimSpace(im.qrzUser.Value()) == "" {
					im.SaveError = "QRZ username is required when QRZ is enabled"
					return im, nil
				}
				if im.qrzPass.Value() == "" {
					im.SaveError = "QRZ password is required when QRZ is enabled"
					return im, nil
				}
			}
			im.done = true
			im.saved = true
			return im, nil
		case " ", "space":
			switch im.focus {
			case imDXCChk:
				im.dxcEnabled = !im.dxcEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			case imQRZChk:
				im.qrzEnabled = !im.qrzEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			}
			// Fall through to text input for editable fields.
			switch im.focus {
			case imDXCHost:
				im.dxcHost, _ = im.dxcHost.Update(msg)
			case imDXCPort:
				im.dxcPort, _ = im.dxcPort.Update(msg)
			case imDXCLogin:
				im.dxcLogin, _ = im.dxcLogin.Update(msg)
			case imQRZUser:
				im.qrzUser, _ = im.qrzUser.Update(msg)
			case imQRZPass:
				im.qrzPass, _ = im.qrzPass.Update(msg)
			}
		case "enter":
			if im.focus == imQRZTest {
				if !im.inetOnline {
					im.qrzTestResult = "No internet connection"
					return im, nil
				}
				user := strings.TrimSpace(im.qrzUser.Value())
				pass := im.qrzPass.Value()
				if user == "" || pass == "" {
					im.qrzTestResult = "Username and password required"
					return im, nil
				}
				im.qrzTesting = true
				im.qrzTestResult = "Testing..."
				return im, func() tea.Msg {
					data, err := qrz.Lookup(user, pass, "SP9MOA")
					return callbookTestMsg{ok: err == nil && data != nil, err: err}
				}
			}
			im.next()
		case "tab", "down":
			im.next()
		case "shift+tab", "up":
			im.prev()
		default:
			switch im.focus {
			case imDXCHost:
				im.dxcHost, _ = im.dxcHost.Update(msg)
			case imDXCPort:
				im.dxcPort, _ = im.dxcPort.Update(msg)
			case imDXCLogin:
				im.dxcLogin, _ = im.dxcLogin.Update(msg)
			case imQRZUser:
				im.qrzUser, _ = im.qrzUser.Update(msg)
			case imQRZPass:
				im.qrzPass, _ = im.qrzPass.Update(msg)
			}
		}
	}
	return im, nil
}

func (im *IntegrationMenu) next() {
	for {
		im.focus = wrapNext(im.focus, imMax)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) prev() {
	for {
		im.focus = wrapPrev(im.focus, imMax)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) isPositionVisible(pos int) bool {
	switch pos {
	case imDXCChk, imQRZChk:
		return true
	case imDXCHost, imDXCPort, imDXCLogin:
		return im.dxcEnabled
	case imQRZUser, imQRZPass, imQRZTest:
		return im.qrzEnabled
	}
	return true
}

func (im *IntegrationMenu) fixFocus() {
	if im.isPositionVisible(im.focus) {
		return
	}
	im.next()
}

func (im *IntegrationMenu) blurAll() {
	blurTextinputs(&im.dxcHost, &im.dxcPort, &im.dxcLogin, &im.qrzUser, &im.qrzPass)
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case imDXCHost:
		im.dxcHost.Focus()
	case imDXCPort:
		im.dxcPort.Focus()
	case imDXCLogin:
		im.dxcLogin.Focus()
	case imQRZUser:
		im.qrzUser.Focus()
	case imQRZPass:
		im.qrzPass.Focus()
	}
}

func (im *IntegrationMenu) View() tea.View {
	if im.done {
		return tea.NewView("")
	}
	w := im.width
	if w < 40 {
		w = 80
	}
	h := im.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	boxW := w - 2
	if boxW < 40 {
		boxW = 40
	}

	var b strings.Builder

	// --- DXC section ---
	dxcCheckbox := "[ ]"
	if im.dxcEnabled {
		dxcCheckbox = "[x]"
	}
	dxcPrefix := "  "
	dxcLabel := S.FormLabelWide.Align(lipgloss.Left).Render("DX Cluster:")
	if im.focus == imDXCChk {
		dxcPrefix = S.FormPrefixOn.Render("> ")
		dxcLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("DX Cluster:")
		dxcCheckbox = CursorStyle.Render(dxcCheckbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, dxcPrefix, dxcLabel, " ", dxcCheckbox),
		boxW))

	if im.dxcEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCHost, "  Host:", &im.dxcHost, false), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCPort, "  Port:", &im.dxcPort, false), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCLogin, "  Login:", &im.dxcLogin, false), boxW))
	}

	b.WriteString("\n")
	b.WriteString(padOrTrunc("", boxW))
	b.WriteString("\n")

	// --- QRZ section ---
	qrzCheckbox := "[ ]"
	if im.qrzEnabled {
		qrzCheckbox = "[x]"
	}
	qrzPrefix := "  "
	qrzLabel := S.FormLabelWide.Align(lipgloss.Left).Render("QRZ.com:")
	if im.focus == imQRZChk {
		qrzPrefix = S.FormPrefixOn.Render("> ")
		qrzLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("QRZ.com:")
		qrzCheckbox = CursorStyle.Render(qrzCheckbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, qrzPrefix, qrzLabel, " ", qrzCheckbox),
		boxW))

	if im.qrzEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imQRZUser, "  Username:", &im.qrzUser, false), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imQRZPass, "  Password:", &im.qrzPass, true), boxW))

		// Test button
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !im.inetOnline {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("(offline)")
		} else if im.focus == imQRZTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, boxW))

		if im.qrzTestResult != "" {
			b.WriteString("\n    ")
			if im.qrzTesting {
				b.WriteString(DimStyle.Render(im.qrzTestResult))
			} else if strings.HasPrefix(im.qrzTestResult, "OK") {
				b.WriteString(SuccessStyle.Render(im.qrzTestResult))
			} else {
				b.WriteString(ErrorStyle.Render(im.qrzTestResult))
			}
		}
	}

	body := drawMenuWithHeader("Configuration \u2014 Integrations", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

// renderField renders a labelled textinput line with cursor indicator.
// When masked is true, the value is shown as asterisks when not focused.
func (im *IntegrationMenu) renderField(focusIdx int, label string, ti *textinput.Model, masked bool) string {
	raw := strings.TrimSpace(ti.Value())
	var val string
	if im.focus == focusIdx {
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
	} else if masked {
		val = ValueStyle.Render(strings.Repeat("*", len(raw)))
	} else {
		val = ValueStyle.Render(raw)
	}
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	if im.focus == focusIdx {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

// Values returns DXC and QRZ config values.
func (im *IntegrationMenu) Values() (dxcEnabled bool, dxcHost, dxcPort, dxcLogin string, qrzEnabled bool, qrzUser, qrzPass string) {
	return im.dxcEnabled,
		strings.TrimSpace(im.dxcHost.Value()),
		strings.TrimSpace(im.dxcPort.Value()),
		strings.TrimSpace(im.dxcLogin.Value()),
		im.qrzEnabled,
		strings.TrimSpace(im.qrzUser.Value()),
		im.qrzPass.Value()
}

// friendlyQRZError wraps raw network errors from QRZ lookups into
// user-readable messages.
func friendlyQRZError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "QRZ:") {
		return msg
	}
	if strings.Contains(msg, "no such host") {
		return "Cannot reach QRZ.com - check your internet connection"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") {
		return "QRZ.com timed out - try again later"
	}
	if strings.Contains(msg, "connection refused") {
		return "Cannot connect to QRZ.com - try again later"
	}
	return "QRZ lookup failed - " + msg
}
