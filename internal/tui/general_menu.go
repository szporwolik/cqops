package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	du := cfg.DistanceUnit
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
	case tea.KeyMsg:
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
			if gm.cursor < 1 {
				gm.cursor++
			}
		case " ", "enter":
			if gm.cursor == 0 {
				if gm.distanceUnit == "km" {
					gm.distanceUnit = "mi"
				} else {
					gm.distanceUnit = "km"
				}
			}
		}
	}
	return gm, nil
}

func (gm *GeneralMenu) FooterText() string {
	return "Space to toggle  ↑↓ to navigate  Ctrl+S to save  Esc to go back"
}

func (gm *GeneralMenu) View() string {
	if gm.done {
		return ""
	}
	bodyW := gm.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder
	title := "── Configuration — General "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")

	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	if gm.cursor == 0 {
		b.WriteString(cursorStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(formLabelStyle.Render("Distance unit:"))
	b.WriteString(" ")
	b.WriteString(inputStyle.Render(unitVal))

	return b.String()
}
