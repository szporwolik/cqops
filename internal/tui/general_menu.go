package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type GeneralMenu struct {
	distanceUnit string
	cursor       int
	done         bool
	saved        bool
	goBack       bool
	width        int
	height       int
}

func NewGeneralMenu(cfg *config.Config) *GeneralMenu {
	du := cfg.General.DistanceUnit
	if du != "mi" {
		du = "km"
	}
	return &GeneralMenu{distanceUnit: du}
}

func (gm *GeneralMenu) Init() tea.Cmd { return nil }

func (gm *GeneralMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		gm.width, gm.height = msg.Width, msg.Height
	case tea.KeyPressMsg:
		switch msg.String() {
		case "esc":
			gm.done = true
			gm.goBack = true
			return gm, nil
		case "ctrl+s", "\x13":
			gm.done = true
			gm.saved = true
			return gm, nil
		case "up", "k":
			if gm.cursor > 0 {
				gm.cursor--
			}
		case "down", "j":
			// Only one config option — cursor stays at 0
		case " ", "space":
			if gm.cursor == 0 {
				if gm.distanceUnit == "km" {
					gm.distanceUnit = "mi"
				} else {
					gm.distanceUnit = "km"
				}
			}
		case "enter":
			// no-op: Enter does not save
		}
	}
	return gm, nil
}

func (gm *GeneralMenu) View() tea.View {
	if gm.done {
		return tea.NewView("")
	}
	w := gm.width
	if w < 40 {
		w = 80
	}
	h := gm.height
	if h < 10 {
		h = 24
	}
	contentH := contentHeight(h)
	if contentH < 3 {
		contentH = 3
	}

	var b strings.Builder
	b.WriteString(menuTitle("Configuration — General", w))
	b.WriteString("\n\n")

	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	prefix := "  "
	unitLabel := LabelStyle.Render("Distance unit:")
	unitDisplay := ValueStyle.Render(unitVal)
	if gm.cursor == 0 {
		prefix = CursorStyle.Render("> ")
		unitLabel = CursorStyle.Render("Distance unit:")
		unitDisplay = CursorStyle.Render(unitVal)
	}
	line := prefix + unitLabel +
		lipgloss.NewStyle().Background(P.Surface).Render(" ") +
		unitDisplay
	b.WriteString(menuLine(line, w))
	b.WriteString("\n")

	return tea.NewView(fillBody(b.String(), contentH))
}
