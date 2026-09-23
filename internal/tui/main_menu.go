package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Pre-allocated menu cell base style — only Width changes per render.
var menuCellBaseStyle = lipgloss.NewStyle().Align(lipgloss.Left)

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
		// Order follows operational relevance: station identity and
		// operation first, online services next, preferences last.
		items: []menuItem{
			{"Logbooks", "Callsign, grid, Wavelog, APRS per logbook"},
			{"Operators", "Multi-operator callsign profiles"},
			{"Rigs", "flrig, rigctld, WSJT-X, antenna"},
			{"Contests", "Contest profiles, exchanges, serials"},
			{"Integrations", "APRS, DX Cluster, PSK, GPS, HTTP dashboard"},
			{"Callbook", "QRZ, HamQTH, Callook, Wavelog lookup"},
			{"General", "Units, timezone, map, solar data, debug, notifications"},
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
		case "esc":
			m.done = true
		case "enter":
			m.setAction(m.cursor)
		case "1", "2", "3", "4", "5", "6", "7":
			// Digit quick-select — F9 menu, then 1-7.
			m.setAction(int(msg.String()[0] - '1'))
		case "up":
			if m.cursor == 0 {
				m.cursor = len(m.items) - 1
			} else {
				m.cursor--
			}
		case "down":
			if m.cursor == len(m.items)-1 {
				m.cursor = 0
			} else {
				m.cursor++
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.items) - 1
		}
	}
	return m, nil
}

// setAction maps a menu position to the selected action.
func (m *MainMenu) setAction(i int) {
	switch i {
	case 0:
		m.action = "logbook"
	case 1:
		m.action = "operator"
	case 2:
		m.action = "rig"
	case 3:
		m.action = "contest"
	case 4:
		m.action = "integration"
	case 5:
		m.action = "callbook"
	case 6:
		m.action = "general"
	}
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
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}

	showDesc := w >= 60
	// Fixed label column so descriptions align vertically.
	const labelW = 13

	// --- Info box (same pattern as other config menus) ---
	infoMaxW := boxW - 6
	if infoMaxW < 30 {
		infoMaxW = 30
	}
	infoText := "CQOps is designed to work out of the box " +
		"with sensible defaults for most setups. " +
		"Need more help? See " +
		osc8Link("https://docs.cqops.com", "docs.cqops.com") +
		" for the full documentation."
	infoLines := wrapLines(infoText, infoMaxW)
	var infoContent strings.Builder
	for i, line := range infoLines {
		infoContent.WriteString(DimStyle.Render(line))
		if i < len(infoLines)-1 {
			infoContent.WriteString("\n")
		}
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.Border)
	infoBox := boxStyle.Render(infoContent.String())

	var b strings.Builder
	b.WriteString(infoBox)
	b.WriteString("\n")
	for i, item := range m.items {
		prefix := "  "
		label := item.label
		if i == m.cursor {
			prefix = S.FormPrefixOn.Render("> ")
			label = CursorStyle.Render(item.label)
		}
		labelCell := menuCellBaseStyle.Width(labelW).Render(label)
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
