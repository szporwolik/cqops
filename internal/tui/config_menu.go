package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/config"
)

type ConfigMenu struct {
	user   textinput.Model
	pass   textinput.Model
	focus  int
	width  int
	height int
	done   bool
}

func NewConfigMenu(cfg *config.Config) *ConfigMenu {
	un := textinput.New(); un.CharLimit = 30; un.Placeholder = "QRZ.com username"; un.SetValue(cfg.QRZUser); un.Focus()
	pw := textinput.New(); pw.CharLimit = 40; pw.Placeholder = "QRZ.com password"; pw.SetValue(cfg.QRZPass)
	return &ConfigMenu{user: un, pass: pw}
}

func (cm *ConfigMenu) Init() tea.Cmd { return nil }

func (cm *ConfigMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: cm.width, cm.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc": cm.done = true; return cm, nil
		case "ctrl+s", "\x13": cm.done = true; return cm, nil
		case "tab", "down": cm.next()
		case "shift+tab", "up": cm.prev()
		default:
			switch cm.focus {
			case 0: cm.user, _ = cm.user.Update(msg)
			case 1: cm.pass, _ = cm.pass.Update(msg)
			}
		}
	}
	return cm, nil
}

func (cm *ConfigMenu) next() { cm.focus = (cm.focus + 1) % 2; cm.blurAll(); cm.focusField() }
func (cm *ConfigMenu) prev() { if cm.focus == 0 { cm.focus = 1 } else { cm.focus-- }; cm.blurAll(); cm.focusField() }
func (cm *ConfigMenu) blurAll() { cm.user.Blur(); cm.pass.Blur() }
func (cm *ConfigMenu) focusField() {
	switch cm.focus { case 0: cm.user.Focus(); case 1: cm.pass.Focus() }
}

func (cm *ConfigMenu) FooterText() string {
	return "Ctrl+S to save  Tab/↓/↑ to navigate  Esc to go back"
}

func (cm *ConfigMenu) View() string {
	if cm.done { return "" }
	var b strings.Builder
	b.WriteString(titleStyle.Render("Configuration — General Options"))
	b.WriteString("\n\n")
	b.WriteString("QRZ.com callsign lookup credentials:")
	b.WriteString("\n\n")
	b.WriteString(formLabelStyle.Render("Username:"))
	b.WriteString(inputStyle.Render(cm.user.View()))
	b.WriteString("\n\n")
	b.WriteString(formLabelStyle.Render("Password:"))
	b.WriteString(inputStyle.Render(cm.pass.View()))
	return b.String()
}
