package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type IntegrationMenu struct {
	// DXC
	dxcEnabled bool
	dxcHost    textinput.Model
	dxcPort    textinput.Model
	dxcLogin   textinput.Model

	// WSJT-X
	wsjtxEnabled bool
	host         textinput.Model
	port         textinput.Model

	focus  int
	done   bool
	saved  bool
	goBack bool
	width  int
	height int
}

const (
	imDXCChk    = 0
	imDXCHost   = 1
	imDXCPort   = 2
	imDXCLogin  = 3
	imWSJTXChk  = 4
	imWSJTXHost = 5
	imWSJTXPort = 6
	imMax       = 7
)

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
	dxcHost := newTextinput()
	dxcHost.CharLimit = 60
	dxcHost.SetWidth(28)
	dxcHost.Placeholder = "dxspider.co.uk"
	if cfg.DXC.Host != "" {
		dxcHost.SetValue(cfg.DXC.Host)
	} else {
		dxcHost.SetValue("dxspider.co.uk")
	}

	dxcPort := newTextinput()
	dxcPort.CharLimit = 6
	dxcPort.SetWidth(28)
	dxcPort.Placeholder = "7300"
	if cfg.DXC.Port != "" {
		dxcPort.SetValue(cfg.DXC.Port)
	} else {
		dxcPort.SetValue("7300")
	}

	dxcLogin := newTextinput()
	dxcLogin.CharLimit = 20
	dxcLogin.SetWidth(28)
	dxcLogin.Placeholder = "callsign"
	if cfg.DXC.Login != "" {
		dxcLogin.SetValue(cfg.DXC.Login)
	}

	host := newTextinput()
	host.CharLimit = 40
	host.SetWidth(28)
	host.Placeholder = "127.0.0.1"
	host.SetValue("127.0.0.1")
	if cfg.WSJTX.Enabled && cfg.WSJTX.UDPHost != "" {
		host.SetValue(cfg.WSJTX.UDPHost)
	}

	port := newTextinput()
	port.CharLimit = 6
	port.SetWidth(28)
	port.Placeholder = "2233"
	port.SetValue("2233")
	if cfg.WSJTX.UDPPort > 0 {
		port.SetValue(strconv.Itoa(cfg.WSJTX.UDPPort))
	}

	return &IntegrationMenu{
		dxcEnabled:   cfg.DXC.Enabled,
		dxcHost:      dxcHost,
		dxcPort:      dxcPort,
		dxcLogin:     dxcLogin,
		wsjtxEnabled: cfg.WSJTX.Enabled,
		host:         host,
		port:         port,
		focus:        0,
	}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		k := msg.String()
		switch k {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case "ctrl+s", "\x13":
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
			case imWSJTXChk:
				im.wsjtxEnabled = !im.wsjtxEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			}
			switch im.focus {
			case imDXCHost:
				im.dxcHost, _ = im.dxcHost.Update(msg)
			case imDXCPort:
				im.dxcPort, _ = im.dxcPort.Update(msg)
			case imDXCLogin:
				im.dxcLogin, _ = im.dxcLogin.Update(msg)
			case imWSJTXHost:
				im.host, _ = im.host.Update(msg)
			case imWSJTXPort:
				im.port, _ = im.port.Update(msg)
			}
		case "enter":
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
			case imWSJTXHost:
				im.host, _ = im.host.Update(msg)
			case imWSJTXPort:
				im.port, _ = im.port.Update(msg)
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
	case imDXCChk:
		return true
	case imDXCHost, imDXCPort, imDXCLogin:
		return im.dxcEnabled
	case imWSJTXChk:
		return true
	case imWSJTXHost, imWSJTXPort:
		return im.wsjtxEnabled
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
	blurTextinputs(&im.dxcHost, &im.dxcPort, &im.dxcLogin, &im.host, &im.port)
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case imDXCHost:
		im.dxcHost.Focus()
	case imDXCPort:
		im.dxcPort.Focus()
	case imDXCLogin:
		im.dxcLogin.Focus()
	case imWSJTXHost:
		im.host.Focus()
	case imWSJTXPort:
		im.port.Focus()
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

	// DXC checkbox
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
		b.WriteString(padOrTrunc(im.renderField(imDXCHost, "  Host:", &im.dxcHost), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCPort, "  Port:", &im.dxcPort), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCLogin, "  Login:", &im.dxcLogin), boxW))
	}

	b.WriteString("\n")
	b.WriteString(padOrTrunc("", boxW))
	b.WriteString("\n")

	// WSJT-X checkbox
	checkbox := "[ ]"
	if im.wsjtxEnabled {
		checkbox = "[x]"
	}
	wsjtxPrefix := "  "
	wsjtxLabel := S.FormLabelWide.Align(lipgloss.Left).Render("WSJT-X:")
	if im.focus == imWSJTXChk {
		wsjtxPrefix = S.FormPrefixOn.Render("> ")
		wsjtxLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("WSJT-X:")
		checkbox = CursorStyle.Render(checkbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, wsjtxPrefix, wsjtxLabel, " ", checkbox),
		boxW))

	if im.wsjtxEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imWSJTXHost, "  UDP Host:", &im.host), boxW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imWSJTXPort, "  UDP Port:", &im.port), boxW))
	}

	body := drawMenuWithHeader("Configuration \u2014 Integrations", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

// renderField renders a labelled textinput line — consistent with callbook menu.
func (im *IntegrationMenu) renderField(focusIdx int, label string, ti *textinput.Model) string {
	raw := strings.TrimSpace(ti.Value())
	var val string
	if im.focus == focusIdx {
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
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

// Values returns DXC and WSJT-X config values.
func (im *IntegrationMenu) Values() (dxcEnabled bool, dxcHost, dxcPort, dxcLogin string, wsjtxEnabled bool, wsjtxHost string, wsjtxPort int) {
	p := 2233
	if v, err := strconv.Atoi(strings.TrimSpace(im.port.Value())); err == nil && v > 0 {
		p = v
	}
	return im.dxcEnabled,
		strings.TrimSpace(im.dxcHost.Value()),
		strings.TrimSpace(im.dxcPort.Value()),
		strings.TrimSpace(im.dxcLogin.Value()),
		im.wsjtxEnabled, strings.TrimSpace(im.host.Value()), p
}
