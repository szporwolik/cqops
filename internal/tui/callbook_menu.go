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
	un := newTextinput()
	un.CharLimit = 30
	un.Placeholder = "QRZ.com username"
	un.SetValue(cfg.QRZ.User)

	pw := newTextinput()
	pw.CharLimit = 40
	pw.Placeholder = "QRZ.com password"
	pw.EchoMode = textinput.EchoPassword
	pw.EchoCharacter = '*'
	pw.SetValue(cfg.QRZ.Pass)

	// Apply surface background to textinput styles
	applyTextinputSurfaceStyle(&un)
	applyTextinputSurfaceStyle(&pw)

	un.Focus()
	return &CallbookMenu{user: un, pass: pw, enabled: cfg.QRZ.Enabled}
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
			cm.testResult = friendlyQRZError(msg.err)
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

func (cm *CallbookMenu) next() { cm.focus = wrapNext(cm.focus, 4); cm.blurAll(); cm.focusField() }
func (cm *CallbookMenu) prev()  { cm.focus = wrapPrev(cm.focus, 4); cm.blurAll(); cm.focusField() }
func (cm *CallbookMenu) blurAll()  { blurTextinputs(&cm.user, &cm.pass) }
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

// renderField renders a labelled textinput line with cursor indicator.
// renderField renders a labelled textinput line with cursor indicator.
// When masked is true, the value is shown as asterisks when not focused.
func (cm *CallbookMenu) renderField(focusIdx int, label string, ti *textinput.Model, masked bool) string {
	gap := lipgloss.NewStyle().Background(P.Surface).Render(" ")
	raw := strings.TrimSpace(ti.Value())
	var val string
	if cm.focus == focusIdx {
		val = ti.View() // respects EchoMode/EchoCharacter when focused
	} else if raw == "" {
		val = SubtleStyle.Render("\u2014")
	} else if masked {
		val = ValueStyle.Render(strings.Repeat("*", len(raw)))
	} else {
		val = ValueStyle.Render(raw)
	}
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
		b.WriteString(menuLine(cm.renderField(1, "  Username:", &cm.user, false), w))
		b.WriteString("\n")
		b.WriteString(menuLine(cm.renderField(2, "  Password:", &cm.pass, true), w))

		// Test button — indented under QRZ.
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !cm.inetOnline {
			btnLine = "    " + DimStyle.Render(btnText) + bg.Render(" ") + DimStyle.Render("(offline)")
		} else if cm.focus == 3 {
			btnLine = CursorStyle.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(menuLine(btnLine, w))

		if cm.testResult != "" {
			b.WriteString("\n    ")
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

// friendlyQRZError wraps raw network errors from QRZ lookups into
// user-readable messages. QRZ API errors (already prefixed "QRZ: …")
// pass through unchanged.
func friendlyQRZError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// QRZ API errors are already user-friendly.
	if strings.Contains(msg, "QRZ:") {
		return msg
	}
	// Network-level errors — borrow wavelog's friendly patterns.
	if strings.Contains(msg, "no such host") {
		return "Cannot reach QRZ.com — check your internet connection"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") {
		return "QRZ.com timed out — try again later"
	}
	if strings.Contains(msg, "connection refused") {
		return "Cannot connect to QRZ.com — try again later"
	}
	return "QRZ lookup failed — " + msg
}
