package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
			{"Callbook", "Lookup providers and credentials"},
			{"Logbook Configuration", "Logs, paths, active log profile"},
			{"Rig Configuration", "flrig / rigctld / manual radio data"},
			{"Integration", "WSJT-X / external tools connectivity"},
		},
	}
}

func (m *MainMenu) Init() tea.Cmd { return nil }

func (m *MainMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			switch m.cursor {
			case 0:
				m.action = "general"
			case 1:
				m.action = "callbook"
			case 2:
				m.action = "logbook"
			case 3:
				m.action = "rig"
			case 4:
				m.action = "integration"
			}
		case "up", "k":
			if m.cursor == 0 {
				m.cursor = len(m.items) - 1
			} else {
				m.cursor--
			}
		case "down", "j":
			if m.cursor == len(m.items)-1 {
				m.cursor = 0
			} else {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m *MainMenu) View() tea.View {
	if m.done {
		return tea.NewView("")
	}

	w := m.width
	if w < 40 {
		w = 80
	}
	h := m.height
	if h < 10 {
		h = 24
	}

	// Available content height: status+profile+tab+help = 4 fixed rows.
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	dim := SubtleStyle
	showDesc := w >= 60

	var b strings.Builder

	// Title header — Surface background fills the full width so no leaking
	// character after the text.
	b.WriteString(menuTitle("Configuration", w))
	b.WriteString("\n\n")

	for i, item := range m.items {
		prefix := "  "
		label := item.label
		if i == m.cursor {
			// Selected row: pink "> " marker and pink option name.
			prefix = CursorStyle.Render("> ")
			label = CursorStyle.Render(item.label)
		}

		line := prefix + label
		if showDesc {
			pad := 26 - lipgloss.Width(prefix) - lipgloss.Width(item.label)
			if pad < 1 {
				pad = 1
			}
			line += lipgloss.NewStyle().Background(P.Surface).Render(strings.Repeat(" ", pad)) + dim.Render(item.desc)
		}
		b.WriteString(menuLine(line, w))
		b.WriteString("\n")
	}

	menuH := lipgloss.Height(b.String())
	fillerH := contentH - menuH
	if fillerH < 0 {
		fillerH = 0
	}
	if fillerH > 0 {
		b.WriteString(strings.Repeat("\n", fillerH))
	}

	return tea.NewView(b.String())
}
