package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
)

// notifItemCount is the number of rows in the Notifications menu:
// the master toggle, three sub-options, two test buttons and a beep toggle.
const notifItemCount = 7

// notifRows is the row style shared by every Notifications row.
var notifRows = rowStyle{label: S.FormLabelXL, focused: S.FormFocusedXL}

type NotificationsMenu struct {
	enabled     bool
	qso         bool
	wavelog     bool
	allErrors   bool
	beepOnError bool
	fm          menuFocus
	done        bool
	saved       bool
	goBack      bool
	width       int
	height      int

	cachedClipStyle lipgloss.Style
	cachedClipH     int

	// statusMsg is set by test actions; parent reads and shows toast, then clears.
	statusMsg string
}

func NewNotificationsMenu(cfg *config.Config) *NotificationsMenu {
	n := cfg.General.Notifications
	return &NotificationsMenu{
		enabled:     n.Enabled,
		qso:         n.QSO,
		wavelog:     n.QSOSent,
		allErrors:   n.AllErrors,
		beepOnError: n.BeepOnError,
	}
}

// focusableRows implementation — all rows are always visible and none carry
// a textinput.
func (nm *NotificationsMenu) rowCount() int        { return notifItemCount }
func (nm *NotificationsMenu) rowVisible(int) bool  { return true }
func (nm *NotificationsMenu) blurAll()             {}
func (nm *NotificationsMenu) focusRow(int) tea.Cmd { return nil }

func (nm *NotificationsMenu) Init() tea.Cmd { return nil }

func (nm *NotificationsMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		nm.width, nm.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		nm.statusMsg = "" // clear status on any key
		switch msg.String() {
		case "esc":
			nm.done = true
			nm.goBack = true
			return nm, nil
		case "ctrl+s", "\x13":
			nm.done = true
			nm.saved = true
			return nm, nil
		}
		if handled, cmd := nm.fm.onKey(msg, nm, func() tea.Cmd {
			nm.done = true
			nm.saved = true
			return nil
		}); handled {
			return nm, cmd
		}
		switch msg.String() {
		case " ", "space":
			switch nm.fm.row {
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
					nm.allErrors = !nm.allErrors
				}
			case 4:
				// Space triggers the Test buttons in parallel with Enter.
				nm.sendTestNotification()
			case 5:
				nm.beepOnError = !nm.beepOnError
			case 6:
				nm.sendTestBeep()
			}
		case "enter":
			switch nm.fm.row {
			case 4:
				nm.sendTestNotification()
			case 6:
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
	if !desktopAvailable() {
		nm.statusMsg = "Notifications unavailable — no desktop environment detected (D-Bus/GUI required)"
		applog.Warn("Test notification skipped: desktop unavailable")
		return
	}
	if err := beeep.Notify(title, body, ""); err != nil {
		applog.Warn("Test notification failed", "error", err.Error())
		nm.statusMsg = "Notification failed: " + err.Error()
	} else {
		nm.statusMsg = "Test notification sent — check your desktop"
	}
}

func (nm *NotificationsMenu) sendTestBeep() {
	applog.Info("Test beep triggered")
	if !desktopAvailable() {
		nm.statusMsg = "Notifications: beep unavailable — no desktop environment detected (D-Bus/GUI required)"
		applog.Warn("Test beep skipped: desktop unavailable")
		return
	}
	if err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration); err != nil {
		applog.Warn("Test beep failed", "error", err.Error())
		nm.statusMsg = "Notifications: beep failed — " + err.Error()
	} else {
		nm.statusMsg = "Notifications: beep played"
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
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}

	// --- Info box (same pattern as other config menus) ---
	infoMaxW := boxW - 6
	if infoMaxW < 30 {
		infoMaxW = 30
	}
	infoText := "Notifications keep you informed about ongoing " +
		"activity and errors without watching the screen. " +
		"A beep on critical errors is recommended for " +
		"unattended operation or field use."
	var b strings.Builder
	infoBox(&b, infoText, infoMaxW)

	focused := func(row int) bool { return nm.fm.row == row }

	// Row 0: master toggle.
	checkboxRow(&b, boxW, focused(0), "System notifications", nm.enabled, "", false, notifRows)

	// Sub-options — dimmed when master is off, indented.
	checkboxRow(&b, boxW, focused(1), "  Notify when QSO logged", nm.qso && nm.enabled, "", !nm.enabled, notifRows)
	checkboxRow(&b, boxW, focused(2), "  Notify when QSO sent", nm.wavelog && nm.enabled, "", !nm.enabled, notifRows)
	checkboxRow(&b, boxW, focused(3), "  Notify on all errors", nm.allErrors && nm.enabled, "", !nm.enabled, notifRows)

	// Row 4: Test notification button (before Beep on errors).
	buttonRow(&b, boxW, focused(4), "[ Test notification ]")

	// Row 5: Beep on errors.
	checkboxRow(&b, boxW, focused(5), "  Beep on all errors", nm.beepOnError, "", false, notifRows)

	// Row 6: Test beep button.
	buttonRow(&b, boxW, focused(6), "[ Test beep ]")

	// Save & Back button at the end of the menu.
	b.WriteString("\n")
	b.WriteString(nm.fm.btn.line("Save & Back", boxW-4))

	body := drawMenuWithHeader("Configuration \u2014 Notifications", b.String(), w)
	if nm.cachedClipH != contentH {
		nm.cachedClipStyle = lipgloss.NewStyle().MaxHeight(contentH)
		nm.cachedClipH = contentH
	}
	return tea.NewView(nm.cachedClipStyle.Render(fillBody(body, contentH)))
}
