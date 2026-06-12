package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/config"
)

type CallbookMenu struct {
	user    textinput.Model
	pass    textinput.Model
	enabled bool
	focus   int
	done    bool
	saved   bool
	goBack  bool
	width   int
	height  int
}

func NewCallbookMenu(cfg *config.Config) *CallbookMenu {
	un := textinput.New(); un.CharLimit = 30; un.Placeholder = "QRZ.com username"; un.SetValue(cfg.QRZUser); un.Focus()
	pw := textinput.New(); pw.CharLimit = 40; pw.Placeholder = "QRZ.com password"; pw.SetValue(cfg.QRZPass)
	return &CallbookMenu{user: un, pass: pw, enabled: cfg.QRZEnabled}
}

func (cm *CallbookMenu) Init() tea.Cmd { return nil }

func (cm *CallbookMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: cm.width, cm.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": cm.done = true; cm.goBack = true; return cm, nil
		case "ctrl+s", "\x13": cm.done = true; cm.saved = true; return cm, nil
		case " ", "enter":
			if cm.focus == 0 { cm.enabled = !cm.enabled; return cm, nil }
			cm.next()
		case "tab", "down": cm.next()
		case "shift+tab", "up": cm.prev()
		default:
			switch cm.focus {
			case 1: cm.user, _ = cm.user.Update(msg)
			case 2: cm.pass, _ = cm.pass.Update(msg)
			}
		}
	}
	return cm, nil
}

func (cm *CallbookMenu) next() { cm.focus = (cm.focus + 1) % 3; cm.blurAll(); cm.focusField() }
func (cm *CallbookMenu) prev() { if cm.focus == 0 { cm.focus = 2 } else { cm.focus-- }; cm.blurAll(); cm.focusField() }
func (cm *CallbookMenu) blurAll() { cm.user.Blur(); cm.pass.Blur() }
func (cm *CallbookMenu) focusField() {
	switch cm.focus { case 0: /* checkbox */; case 1: cm.user.Focus(); case 2: cm.pass.Focus() }
}

func (cm *CallbookMenu) FooterText() string {
	return "Ctrl+S to save  Space/Enter to toggle  Tab/↓/↑ to navigate  Esc to go back"
}

func (cm *CallbookMenu) View() string {
	if cm.done { return "" }
	bodyW := cm.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder
	title := "── Callbook "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")
	checkbox := "[ ]"
	if cm.enabled { checkbox = "[x]" }
	if cm.focus == 0 { checkbox = cursorStyle.Render(checkbox) }
	b.WriteString(formLabelStyle.Render("Use QRZ:") + " " + checkbox)
	if cm.enabled {
		b.WriteString("\n\n")
		if cm.focus == 1 { b.WriteString(cursorStyle.Render("> ")) } else { b.WriteString("  ") }
		b.WriteString(formLabelStyle.Render("Username:"))
		b.WriteString(inputStyle.Render(cm.user.View()))
		b.WriteString("\n\n")
		if cm.focus == 2 { b.WriteString(cursorStyle.Render("> ")) } else { b.WriteString("  ") }
		b.WriteString(formLabelStyle.Render("Password:"))
		b.WriteString(inputStyle.Render(cm.pass.View()))
	}
	return b.String()
}
