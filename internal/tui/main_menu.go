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
			{"General", "Language, timezone, distance units"},
			{"Station", "Callsign, operator, locator, CQ/ITU zones"},
			{"Logbooks", "Logs, station profiles, paths"},
			{"Rigs", "Radio models, antennas, flrig"},
			{"Contests", "Contest profiles, exchanges, serials"},
			{"Integration", "DX Cluster, QRZ.com, Wavelog"},
			{"Notifications", "Desktop alert preferences"},
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
				m.action = "station"
			case 2:
				m.action = "logbook"
			case 3:
				m.action = "rig"
			case 4:
				m.action = "contest"
			case 5:
				m.action = "integration"
			case 6:
				m.action = "notifications"
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

	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	boxW := w - 2
	if boxW < 40 {
		boxW = 40
	}

	showDesc := w >= 60
	// Fixed label column so descriptions align vertically.
	const labelW = 13

	var b strings.Builder
	for i, item := range m.items {
		prefix := "  "
		label := item.label
		if i == m.cursor {
			prefix = S.FormPrefixOn.Render("> ")
			label = CursorStyle.Render(item.label)
		}
		labelCell := lipgloss.NewStyle().Width(labelW).Align(lipgloss.Left).Render(label)
		line := prefix + labelCell
		if showDesc {
			line += "  " + DimStyle.Render(item.desc)
		}
		b.WriteString(padOrTrunc(line, boxW))
		if i < len(m.items)-1 {
			b.WriteString("\n")
		}
	}

	body := drawMenuWithHeader("Configuration", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}
