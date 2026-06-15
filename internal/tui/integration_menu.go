package tui

import (
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type IntegrationMenu struct {
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

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
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
			case 0: // WSJTX checkbox
				im.wsjtxEnabled = !im.wsjtxEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			}
			switch im.focus {
			case 1:
				im.host, _ = im.host.Update(msg)
			case 2:
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
			case 1:
				im.host, _ = im.host.Update(msg)
			case 2:
				im.port, _ = im.port.Update(msg)
			}
		}
	}
	return im, nil
}

func (im *IntegrationMenu) next() {
	for {
		im.focus = wrapNext(im.focus, 3)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) prev() {
	for {
		im.focus = wrapPrev(im.focus, 3)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) isPositionVisible(pos int) bool {
	switch pos {
	case 0: // WSJTX checkbox — always visible
		return true
	case 1, 2: // WSJTX host/port — only when enabled
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
	blurTextinputs(&im.host, &im.port)
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case 1:
		im.host.Focus()
	case 2:
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

	var b strings.Builder
	b.WriteString(menuTitle("Settings — Integration", w))
	b.WriteString("\n\n")

	// ── WSJT-X section ──
	checkbox := "[ ]"
	if im.wsjtxEnabled {
		checkbox = "[x]"
	}
	wsjtxPrefix := "  "
	if im.focus == 0 {
		wsjtxPrefix = CursorStyle.Render("> ")
		checkbox = CursorStyle.Render(checkbox)
	}
	b.WriteString(menuLine(wsjtxPrefix+LabelStyle.Render(fit("WSJT-X:", 14))+" "+checkbox, w))

	if im.wsjtxEnabled {
		b.WriteString("\n")
		b.WriteString(menuLine(im.renderField(1, "  UDP Host:", &im.host), w))
		b.WriteString("\n")
		b.WriteString(menuLine(im.renderField(2, "  UDP Port:", &im.port), w))
	}

	return tea.NewView(fillBody(b.String(), contentH))
}

// renderField renders a labelled textinput line — consistent with callbook menu.
func (im *IntegrationMenu) renderField(focusIdx int, label string, ti *textinput.Model) string {
	val := InputStyle.Render(strings.TrimSpace(ti.Value()))
	if im.focus == focusIdx {
		val = ti.View()
	}
	padded := fit(label, 14)
	if im.focus == focusIdx {
		return CursorStyle.Render("> ") + CursorStyle.Render(padded) + " " + val
	}
	return "  " + LabelStyle.Render(padded) + " " + val
}

// Values returns WSJT-X config values.
func (im *IntegrationMenu) Values() (wsjtxEnabled bool, wsjtxHost string, wsjtxPort int) {
	p := 2233
	if v, err := strconv.Atoi(strings.TrimSpace(im.port.Value())); err == nil && v > 0 {
		p = v
	}
	return im.wsjtxEnabled, strings.TrimSpace(im.host.Value()), p
}
