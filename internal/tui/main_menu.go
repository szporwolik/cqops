package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type MainMenu struct {
	items  []string
	cursor int
	done   bool
	action string
	width  int
	height int
}

func NewMainMenu() *MainMenu {
	return &MainMenu{
		items: []string{"General Options", "Callbook / QRZ.com", "Logbook Configuration", "Rig Configuration"},
	}
}

func (m *MainMenu) Init() tea.Cmd { return nil }

func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			switch m.cursor {
			case 0: m.action = "general"
			case 1: m.action = "callbook"
			case 2: m.action = "logbook"
			case 3: m.action = "rig"
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *MainMenu) FooterText() string {
	return "Enter to select  F1 QSO Form  F10 Quit"
}

func (m *MainMenu) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("Configuration — Menu"))
	b.WriteString("\n\n")
	for i, item := range m.items {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}
		b.WriteString(prefix + item + "\n")
	}
	return b.String()
}
