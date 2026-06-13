package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/config"
)

type IntegrationMenu struct {
	enabled bool
	host    textinput.Model
	port    textinput.Model
	focus   int
	done    bool
	saved   bool
	goBack  bool
	width   int
	height  int
}

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
	host := textinput.New()
	host.CharLimit = 40
	host.Placeholder = "127.0.0.1"
	host.SetValue("127.0.0.1")
	if cfg.WSJTX.Enabled && cfg.WSJTX.UDPHost != "" {
		host.SetValue(cfg.WSJTX.UDPHost)
	}

	port := textinput.New()
	port.CharLimit = 6
	port.Placeholder = "2233"
	port.SetValue("2233")
	if cfg.WSJTX.UDPPort > 0 {
		port.SetValue(strconv.Itoa(cfg.WSJTX.UDPPort))
	}
	host.Focus()

	return &IntegrationMenu{enabled: cfg.WSJTX.Enabled, host: host, port: port, focus: 1}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case "ctrl+s", "\x13":
			im.done = true
			im.saved = true
			return im, nil
		case " ", "enter":
			if im.focus == 0 {
				im.enabled = !im.enabled
				return im, nil
			}
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
	im.focus = (im.focus + 1) % 3
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) prev() {
	if im.focus == 0 {
		im.focus = 2
	} else {
		im.focus--
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) blurAll() {
	im.host.Blur()
	im.port.Blur()
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case 0:
	case 1:
		im.host.Focus()
	case 2:
		im.port.Focus()
	}
}

func (im *IntegrationMenu) FooterText() string {
	return "Ctrl+S to save  Space/Enter to toggle  Tab/↓/↑ to navigate  Esc to go back"
}

func (im *IntegrationMenu) View() string {
	if im.done {
		return ""
	}
	bodyW := im.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder
	title := "── Configuration — Integration "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")

	checkbox := "[ ]"
	if im.enabled {
		checkbox = "[x]"
	}
	if im.focus == 0 {
		checkbox = cursorStyle.Render(checkbox)
	}
	b.WriteString(formLabelStyle.Render("WSJT-X:") + " " + checkbox)

	if im.enabled {
		b.WriteString("\n\n")
		if im.focus == 1 {
			b.WriteString(cursorStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(formLabelStyle.Render("UDP Host:"))
		b.WriteString(inputStyle.Render(im.host.View()))

		b.WriteString("\n\n")
		if im.focus == 2 {
			b.WriteString(cursorStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(formLabelStyle.Render("UDP Port:"))
		b.WriteString(inputStyle.Render(im.port.View()))
	}

	b.WriteString(fmt.Sprintf("\n\n  %s", SubtleStyle.Render("default: 127.0.0.1:2233")))

	return b.String()
}

func (im *IntegrationMenu) Values() (bool, string, int) {
	p := 2233
	if v, err := strconv.Atoi(strings.TrimSpace(im.port.Value())); err == nil && v > 0 {
		p = v
	}
	return im.enabled, strings.TrimSpace(im.host.Value()), p
}
