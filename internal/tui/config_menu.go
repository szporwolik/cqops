package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/config"
)

type ConfigMenu struct {
	apiKey textinput.Model
	width  int
	height int
	done   bool
}

func NewConfigMenu(cfg *config.Config) *ConfigMenu {
	ak := textinput.New()
	ak.CharLimit = 60
	ak.Placeholder = "Enter QRZ.com API key"
	if cfg.QRZAPIKey != "" {
		ak.SetValue(cfg.QRZAPIKey)
	}
	ak.Focus()

	return &ConfigMenu{
		apiKey: ak,
	}
}

func (cm *ConfigMenu) Init() tea.Cmd { return nil }

func (cm *ConfigMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cm.width = msg.Width
		cm.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			cm.done = true
			return cm, nil
		case "enter":
			cm.done = true
			return cm, nil
		default:
			cm.apiKey, _ = cm.apiKey.Update(msg)
		}
	}
	return cm, nil
}

func (cm *ConfigMenu) View() string {
	if cm.done {
		return ""
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n\n")
	b.WriteString(formLabelStyle.Render("QRZ API Key:"))
	b.WriteString(inputStyle.Render(cm.apiKey.View()))
	b.WriteString("\n\n\n")
	b.WriteString(helpStyle.Render("Enter to save  |  Esc to cancel"))
	return b.String()
}
