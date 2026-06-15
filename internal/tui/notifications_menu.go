package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
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

	var b strings.Builder
	b.WriteString(menuTitle("Settings — Notifications", w))
	b.WriteString("\n\n")

	// Row 0: master toggle
	nm.renderCheckbox(&b, w, 0, "System notifications", nm.enabled, false)

	// Sub-options — dimmed when master is off, indented.
	// Row 1: QSO logged
	nm.renderCheckbox(&b, w, 1, "  Notify when QSO is logged", nm.qso && nm.enabled, !nm.enabled)

	// Row 2: Wavelog sent
	nm.renderCheckbox(&b, w, 2, "  Notify when QSO is sent to Wavelog", nm.wavelog && nm.enabled, !nm.enabled)

	// Row 3: Wavelog errors
	nm.renderCheckbox(&b, w, 3, "  Notify on Wavelog errors", nm.wavelogErrors && nm.enabled, !nm.enabled)

	// Row 4: Test notification button
	btn := "[ Test notification ]"
	if nm.cursor == 4 {
		b.WriteString(menuLine("  "+CursorStyle.Render("> ")+CursorStyle.Render(btn), w))
	} else {
		b.WriteString(menuLine("    "+InputStyle.Render(btn), w))
	}
	b.WriteString("\n")

	return tea.NewView(fillBody(b.String(), contentH))
}

func (nm *NotificationsMenu) renderCheckbox(b *strings.Builder, w, cursor int, label string, checked, disabled bool) {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}
	labelFit := fit(label, 38)

	if disabled {
		if nm.cursor == cursor {
			b.WriteString(menuLine(CursorStyle.Render("> ")+DimStyle.Render(labelFit)+" "+checkbox, w))
		} else {
			b.WriteString(menuLine("  "+DimStyle.Render(labelFit)+" "+checkbox, w))
		}
	} else if nm.cursor == cursor {
		checkbox = CursorStyle.Render(checkbox)
		b.WriteString(menuLine(CursorStyle.Render("> ")+CursorStyle.Render(labelFit)+" "+checkbox, w))
	} else {
		b.WriteString(menuLine("  "+LabelStyle.Render(labelFit)+" "+checkbox, w))
	}
	b.WriteString("\n")
}
