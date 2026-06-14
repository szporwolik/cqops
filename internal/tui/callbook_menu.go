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
	un.Prompt = ""
	un.Placeholder = "QRZ.com username"
	un.SetValue(cfg.QRZUser)

	pw := textinput.New()
	pw.CharLimit = 40
	pw.Prompt = ""
	pw.Placeholder = "QRZ.com password"
	pw.SetValue(cfg.QRZPass)

	// Apply surface background to textinput styles (same pattern as QSO form)
	us := un.Styles()
	us.Focused.Text = us.Focused.Text.Background(P.Surface)
	us.Focused.Placeholder = us.Focused.Placeholder.Background(P.Surface)
	us.Focused.Prompt = us.Focused.Prompt.Background(P.Surface)
	us.Focused.Suggestion = us.Focused.Suggestion.Background(P.Surface)
	us.Blurred.Text = us.Blurred.Text.Background(P.Surface)
	us.Blurred.Placeholder = us.Blurred.Placeholder.Background(P.Surface)
	us.Blurred.Prompt = us.Blurred.Prompt.Background(P.Surface)
	us.Blurred.Suggestion = us.Blurred.Suggestion.Background(P.Surface)
	un.SetStyles(us)

	ps := pw.Styles()
	ps.Focused.Text = ps.Focused.Text.Background(P.Surface)
	ps.Focused.Placeholder = ps.Focused.Placeholder.Background(P.Surface)
	ps.Focused.Prompt = ps.Focused.Prompt.Background(P.Surface)
	ps.Focused.Suggestion = ps.Focused.Suggestion.Background(P.Surface)
	ps.Blurred.Text = ps.Blurred.Text.Background(P.Surface)
	ps.Blurred.Placeholder = ps.Blurred.Placeholder.Background(P.Surface)
	ps.Blurred.Prompt = ps.Blurred.Prompt.Background(P.Surface)
	ps.Blurred.Suggestion = ps.Blurred.Suggestion.Background(P.Surface)
	pw.SetStyles(ps)

	un.Focus()
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

	case tea.KeyPressMsg:
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
		case " ", "space":
			if cm.focus == 0 {
				cm.enabled = !cm.enabled
				return cm, nil
			}
			// fall through to default for text input focus
			switch cm.focus {
			case 1:
				cm.user, _ = cm.user.Update(msg)
			case 2:
				cm.pass, _ = cm.pass.Update(msg)
			}
		case "enter":
			switch cm.focus {
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
	return "Ctrl+S to save  Space to toggle  ↑↓/Tab to navigate  Esc to go back"
}

// renderField renders a labelled textinput line with cursor indicator.
// renderField renders a labelled textinput line. Values always use
// InputStyle (plain styled text) — never ti.View() — so focused and
// blurred fields are pixel-identical. Focus is shown by the "> " prefix
// and pink label colour.
func (cm *CallbookMenu) renderField(focusIdx int, label string, ti *textinput.Model) string {
	gap := lipgloss.NewStyle().Background(P.Surface).Render(" ")
	val := InputStyle.Render(strings.TrimSpace(ti.Value()))
	padded := fit(label, 14)
	if cm.focus == focusIdx {
		return CursorStyle.Render("> ") + CursorStyle.Render(padded) + gap + val
	}
	return "  " + LabelStyle.Render(padded) + gap + val
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
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	var b strings.Builder
	b.WriteString(menuTitle("Configuration — Callbook", w))
	b.WriteString("\n\n")

	bg := lipgloss.NewStyle().Background(P.Surface)
	checkbox := "[ ]"
	if cm.enabled {
		checkbox = "[x]"
	}
	if cm.focus == 0 {
		checkbox = CursorStyle.Render(checkbox)
	} else {
		checkbox = bg.Render(checkbox)
	}
	// QRZ checkbox — show "> " marker when focused.
	qrPrefix := "  "
	if cm.focus == 0 {
		qrPrefix = CursorStyle.Render("> ")
	}
	b.WriteString(menuLine(qrPrefix+LabelStyle.Render(fit("Use QRZ:", 14))+bg.Render(" ")+checkbox, w))
	if cm.enabled {
		b.WriteString("\n")
		b.WriteString(menuLine(cm.renderField(1, "Username:", &cm.user), w))
		b.WriteString("\n")
		b.WriteString(menuLine(cm.renderField(2, "Password:", &cm.pass), w))

		// Test button
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !cm.inetOnline {
			btnLine = "  " + DimStyle.Render(btnText) + bg.Render(" ") + DimStyle.Render("(offline)")
		} else if cm.focus == 3 {
			btnLine = CursorStyle.Render("> ") + CursorStyle.Render(btnText)
		} else {
			btnLine = "  " + InputStyle.Render(btnText)
		}
		b.WriteString(menuLine(btnLine, w))

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

	return tea.NewView(fillBody(b.String(), contentH))
}
