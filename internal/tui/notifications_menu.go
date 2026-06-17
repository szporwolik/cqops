package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

type NotificationsMenu struct {
	enabled       bool
	qso           bool
	wavelog       bool
	wavelogErrors bool
	beepOnError   bool
	cursor        int
	done          bool
	saved         bool
	goBack        bool
	width         int
	height        int
}

const notifItemCount = 7 // 5 checkboxes + 2 buttons

func NewNotificationsMenu(cfg *config.Config) *NotificationsMenu {
	n := cfg.General.Notifications
	return &NotificationsMenu{
		enabled:       n.Enabled,
		qso:           n.QSO,
		wavelog:       n.Wavelog,
		wavelogErrors: n.WavelogErrors,
		beepOnError:   n.BeepOnError,
	}
}

func (nm *NotificationsMenu) Init() tea.Cmd { return nil }

func (nm *NotificationsMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		nm.width, nm.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			nm.done = true
			nm.goBack = true
			return nm, nil
		case "ctrl+s", "\x13":
			nm.done = true
			nm.saved = true
			return nm, nil
		case "up", "k":
			if nm.cursor == 0 {
				nm.cursor = notifItemCount - 1
			} else {
				nm.cursor--
			}
		case "down", "j":
			if nm.cursor == notifItemCount-1 {
				nm.cursor = 0
			} else {
				nm.cursor++
			}
		case " ", "space":
			switch nm.cursor {
			case 0:
				nm.enabled = !nm.enabled
			case 1:
				if nm.enabled {
					nm.qso = !nm.qso
				}
			case 2:
				if nm.enabled {
					nm.wavelog = !nm.wavelog
				}
			case 3:
				if nm.enabled {
					nm.wavelogErrors = !nm.wavelogErrors
				}
			case 4:
				nm.beepOnError = !nm.beepOnError
			}
		case "enter":
			if nm.cursor == 5 {
				nm.sendTestNotification()
			} else if nm.cursor == 6 {
				nm.sendTestBeep()
			}
		}
	}
	return nm, nil
}

func (nm *NotificationsMenu) sendTestNotification() {
	title := "CQOps — Test Notification"
	body := "This is a test notification from CQOps."
	applog.Info("Test notification sent")
	if err := beeep.Notify(title, body, ""); err != nil {
		applog.Warn("Test notification failed", "error", err.Error())
	}
}

func (nm *NotificationsMenu) sendTestBeep() {
	applog.Info("Test beep triggered")
	if err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration); err != nil {
		applog.Warn("Test beep failed", "error", err.Error())
	}
}

func (nm *NotificationsMenu) View() tea.View {
	if nm.done {
		return tea.NewView("")
	}
	w := nm.width
	if w < 40 {
		w = 80
	}
	h := nm.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	boxW := w - 2
	if boxW < 56 {
		boxW = 56
	}

	var b strings.Builder

	// Row 0: master toggle
	nm.renderCheckbox(&b, boxW, 0, "System notifications", nm.enabled, false)

	// Sub-options — dimmed when master is off, indented.
	nm.renderCheckbox(&b, boxW, 1, "  Notify when QSO logged", nm.qso && nm.enabled, !nm.enabled)
	nm.renderCheckbox(&b, boxW, 2, "  Notify when sent to Wavelog", nm.wavelog && nm.enabled, !nm.enabled)
	nm.renderCheckbox(&b, boxW, 3, "  Notify on Wavelog errors", nm.wavelogErrors && nm.enabled, !nm.enabled)

	// Row 4: Beep on errors — same indent as sub-options.
	nm.renderCheckbox(&b, boxW, 4, "  Beep on all errors", nm.beepOnError, false)

	// Button helper — keeps fixed padding so buttons never shift on focus.
	renderBtn := func(idx int, text string) {
		focused := nm.cursor == idx
		prefix := "    "
		styled := InputStyle.Render(text)
		if focused {
			prefix = S.FormPrefixOn.Render("> ") + "  "
			styled = CursorStyle.Render(text)
		}
		b.WriteString(padOrTrunc(prefix+styled, boxW))
		b.WriteString("\n")
	}

	// Row 5: Test notification button
	renderBtn(5, "[ Test notification ]")

	// Row 6: Test beep button
	renderBtn(6, "[ Test beep ]")

	body := drawMenuWithHeader("Configuration \u2014 Notifications", b.String(), w)
	return tea.NewView(lipgloss.NewStyle().MaxHeight(contentH).Render(fillBody(body, contentH)))
}

func (nm *NotificationsMenu) renderCheckbox(b *strings.Builder, boxW, cursor int, label string, checked, disabled bool) {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}

	prefix := "  "
	lbl := S.FormLabelXL.Align(lipgloss.Left).Render(label)
	if nm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedXL.Align(lipgloss.Left).Render(label)
		checkbox = CursorStyle.Render(checkbox)
	}
	if disabled {
		lbl = DimStyle.Render(S.FormLabelXL.Align(lipgloss.Left).Render(label))
	}

	line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", checkbox)
	b.WriteString(padOrTrunc(line, boxW))
	b.WriteString("\n")
}
