package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type GeneralMenu struct {
	distanceUnit string
	timezone     string
	tzIndex      int
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
	tz := cfg.General.Timezone
	tzIdx := 0
	for i, candidate := range config.Timezones {
		if candidate == tz {
			tzIdx = i
			break
		}
	}
	if tz == "" {
		tz = "UTC"
	}
	return &GeneralMenu{distanceUnit: du, timezone: tz, tzIndex: tzIdx}
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
			if gm.cursor == 0 {
				gm.cursor = 1
			} else {
				gm.cursor = 0
			}
		case "down", "j":
			if gm.cursor == 0 {
				gm.cursor = 1
			} else {
				gm.cursor = 0
			}
		case " ", "space":
			if gm.cursor == 0 {
				if gm.distanceUnit == "km" {
					gm.distanceUnit = "mi"
				} else {
					gm.distanceUnit = "km"
				}
			} else if gm.cursor == 1 {
				gm.tzIndex++
				if gm.tzIndex >= len(config.Timezones) {
					gm.tzIndex = 0
				}
				gm.timezone = config.Timezones[gm.tzIndex]
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

	boxW := w - 2
	if boxW < 40 {
		boxW = 40
	}

	var b strings.Builder

	// Distance unit row
	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	b.WriteString(formCheckbox("Distance unit", unitVal, gm.cursor == 0, boxW))
	b.WriteString("\n")

	// Timezone row
	b.WriteString(formCheckbox("Timezone", gm.timezone, gm.cursor == 1, boxW))
	b.WriteString("\n")

	body := drawMenuBox(b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

// formCheckbox renders a label + value row for toggle-style menu items.
// When focused, the "> " cursor prefix appears.
func formCheckbox(label, value string, focused bool, width int) string {
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	val := ValueStyle.Render(value)
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		val = CursorStyle.Render(value)
	}
	line := lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
	return padOrTrunc(line, width)
}
