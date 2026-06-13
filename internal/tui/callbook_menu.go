package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
)

type CallbookMenu struct {
	user       textinput.Model
	pass       textinput.Model
	enabled    bool
	focus      int
	done       bool
	saved      bool
	goBack     bool
	testing    bool
	testResult string
	inetOnline bool
	width      int
	height     int
}

func NewCallbookMenu(cfg *config.Config) *CallbookMenu {
	un := textinput.New()
	un.CharLimit = 30
	un.Placeholder = "QRZ.com username"
	un.SetValue(cfg.QRZUser)
	un.Focus()
	pw := textinput.New()
	pw.CharLimit = 40
	pw.Placeholder = "QRZ.com password"
	pw.SetValue(cfg.QRZPass)
	return &CallbookMenu{user: un, pass: pw, enabled: cfg.QRZEnabled}
}

func (cm *CallbookMenu) Init() tea.Cmd { return nil }

type callbookTestMsg struct {
	ok  bool
	err error
}

func (cm *CallbookMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cm.width, cm.height = msg.Width, msg.Height

	case callbookTestMsg:
		cm.testing = false
		if msg.err != nil {
			cm.testResult = msg.err.Error()
			applog.Error("QRZ test failed", "error", msg.err.Error())
		} else if msg.ok {
			cm.testResult = "OK — QRZ.com connected"
			applog.Info("QRZ test OK")
		} else {
			cm.testResult = "No data returned"
			applog.Warn("QRZ test: no data returned")
		}

	case tea.KeyMsg:
		k := msg.String()
		if cm.testing {
			return cm, nil
		}
		switch k {
		case "esc":
			cm.done = true
			cm.goBack = true
			return cm, nil
		case "ctrl+s", "\x13":
			cm.done = true
			cm.saved = true
			return cm, nil
		case " ", "enter":
			switch cm.focus {
			case 0:
				cm.enabled = !cm.enabled
			case 3:
				if !cm.inetOnline {
					cm.testResult = "No internet connection"
					return cm, nil
				}
				user := strings.TrimSpace(cm.user.Value())
				pass := cm.pass.Value()
				if user == "" || pass == "" {
					cm.testResult = "Username and password required"
					return cm, nil
				}
				cm.testing = true
				cm.testResult = "Testing…"
				return cm, func() tea.Msg {
					data, err := qrz.Lookup(user, pass, "SP9MOA")
					return callbookTestMsg{ok: err == nil && data != nil, err: err}
				}
			default:
				cm.next()
			}
		case "tab", "down":
			cm.next()
		case "shift+tab", "up":
			cm.prev()
		default:
			switch cm.focus {
			case 1:
				cm.user, _ = cm.user.Update(msg)
			case 2:
				cm.pass, _ = cm.pass.Update(msg)
			}
		}
	}
	return cm, nil
}

func (cm *CallbookMenu) next() { cm.focus = (cm.focus + 1) % 4; cm.blurAll(); cm.focusField() }
func (cm *CallbookMenu) prev() {
	if cm.focus == 0 {
		cm.focus = 3
	} else {
		cm.focus--
	}
	cm.blurAll()
	cm.focusField()
}
func (cm *CallbookMenu) blurAll() { cm.user.Blur(); cm.pass.Blur() }
func (cm *CallbookMenu) focusField() {
	switch cm.focus {
	case 0: /* checkbox */
	case 1:
		cm.user.Focus()
	case 2:
		cm.pass.Focus()
	case 3: /* test button */
	}
}

func (cm *CallbookMenu) FooterText() string {
	if cm.testing {
		return "Testing QRZ.com connection…"
	}
	return "Ctrl+S to save  Space/Enter to toggle/select  Tab/↓/↑ to navigate  Esc to go back"
}

func (cm *CallbookMenu) View() string {
	if cm.done {
		return ""
	}
	bodyW := cm.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder
	title := "── Configuration — Callbook "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")
	checkbox := "[ ]"
	if cm.enabled {
		checkbox = "[x]"
	}
	if cm.focus == 0 {
		checkbox = cursorStyle.Render(checkbox)
	}
	b.WriteString(formLabelStyle.Render("Use QRZ:"))
	b.WriteString(" ")
	b.WriteString(checkbox)
	if cm.enabled {
		b.WriteString("\n\n")
		if cm.focus == 1 {
			b.WriteString(cursorStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(formLabelStyle.Render("Username:"))
		b.WriteString(inputStyle.Render(cm.user.View()))
		b.WriteString("\n\n")
		if cm.focus == 2 {
			b.WriteString(cursorStyle.Render("> "))
		} else {
			b.WriteString("  ")
		}
		b.WriteString(formLabelStyle.Render("Password:"))
		b.WriteString(inputStyle.Render(cm.pass.View()))

		// Test button
		b.WriteString("\n\n")
		btnText := "[ Test Connection ]"
		if !cm.inetOnline {
			b.WriteString("  ")
			b.WriteString(DimStyle.Render(btnText))
			b.WriteString(DimStyle.Render(" (offline)"))
		} else if cm.focus == 3 {
			b.WriteString(cursorStyle.Render("> "))
			b.WriteString(cursorStyle.Render(btnText))
		} else {
			b.WriteString("  ")
			b.WriteString(InputStyle.Render(btnText))
		}

		if cm.testResult != "" {
			b.WriteString("\n  ")
			if cm.testing {
				b.WriteString(SubtleStyle.Render(cm.testResult))
			} else if strings.HasPrefix(cm.testResult, "OK") {
				b.WriteString(SuccessStyle.Render(cm.testResult))
			} else {
				b.WriteString(ErrorStyle.Render(cm.testResult))
			}
		}
	}

	return b.String()
}
