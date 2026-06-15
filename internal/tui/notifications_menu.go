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
	cursor        int
	done          bool
	saved         bool
	goBack        bool
	width         int
	height        int
}

const notifItemCount = 5 // 4 checkboxes + 1 button

func NewNotificationsMenu(cfg *config.Config) *NotificationsMenu {
	n := cfg.General.Notifications
	return &NotificationsMenu{
		enabled:       n.Enabled,
		qso:           n.QSO,
		wavelog:       n.Wavelog,
		wavelogErrors: n.WavelogErrors,
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
			}
		case "enter":
			if nm.cursor == 4 {
				nm.sendTestNotification()
			}
		}
	}
	return nm, nil
}

func (nm *NotificationsMenu) sendTestNotification() {
	title := "CQOPS — Test Notification"
	body := "This is a test notification from CQOPS."
	applog.Info("Test notification sent")
	if err := beeep.Notify(title, body, ""); err != nil {
		applog.Info("Test notification failed", "error", err.Error())
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
	if boxW < 40 {
		boxW = 40
	}

	var b strings.Builder

	// Row 0: master toggle
	nm.renderCheckbox(&b, boxW, 0, "System notifications", nm.enabled, false)

	// Sub-options — dimmed when master is off, indented.
	nm.renderCheckbox(&b, boxW, 1, "  Notify when QSO logged", nm.qso && nm.enabled, !nm.enabled)
	nm.renderCheckbox(&b, boxW, 2, "  Notify when sent to Wavelog", nm.wavelog && nm.enabled, !nm.enabled)
	nm.renderCheckbox(&b, boxW, 3, "  Notify on Wavelog errors", nm.wavelogErrors && nm.enabled, !nm.enabled)

	// Row 4: Test notification button
	btn := "[ Test notification ]"
	if nm.cursor == 4 {
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center,
				S.FormPrefixOn.Render("> "),
				CursorStyle.Render(btn)),
			boxW))
	} else {
		b.WriteString(padOrTrunc("    "+InputStyle.Render(btn), boxW))
	}
	b.WriteString("\n")

	body := drawMenuBox(b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

func (nm *NotificationsMenu) renderCheckbox(b *strings.Builder, boxW, cursor int, label string, checked, disabled bool) {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}

	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	if nm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		checkbox = CursorStyle.Render(checkbox)
	}
	if disabled {
		lbl = DimStyle.Render(S.FormLabelWide.Align(lipgloss.Left).Render(label))
	}

	line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", checkbox)
	b.WriteString(padOrTrunc(line, boxW))
	b.WriteString("\n")
}
