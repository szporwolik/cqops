package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuItem struct {
	label string
	desc  string
}

type MainMenu struct {
	items  []menuItem
	cursor int
	done   bool
	action string
	width  int
	height int
}

func NewMainMenu() *MainMenu {
	return &MainMenu{
		items: []menuItem{
			{"General Options", "Callsign, operator, locator, defaults"},
			{"Callbook / QRZ.com", "QRZ username, password, lookup behavior"},
			{"Logbook Configuration", "Logs, paths, active log profile"},
			{"Rig Configuration", "flrig / rigctld / manual radio data"},
		},
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

	bodyW := m.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	cursor := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	showDesc := bodyW >= 60

	var b strings.Builder

	title := "── Configuration "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
	b.WriteString("\n\n")

	for i, item := range m.items {
		prefix := "  "
		label := item.label
		if i == m.cursor {
			prefix = cursor.Render("> ")
			label = inputStyle.Render(item.label)
		}

		line := prefix + label
		if showDesc {
			pad := 26 - lipgloss.Width(prefix) - lipgloss.Width(item.label)
			if pad < 1 {
				pad = 1
			}
			line += strings.Repeat(" ", pad) + dim.Render(item.desc)
		}
		b.WriteString(line + "\n")
	}

	if showDesc {
		b.WriteString(fmt.Sprintf("\n  %s", dim.Render("↑↓ to navigate  Enter to select")))
	}

	return b.String()
}
