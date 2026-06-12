package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/config"
)

type GeneralMenu struct {
	renderImages bool
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
	return &GeneralMenu{renderImages: cfg.RenderImages, distanceUnit: du}
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
				gm.renderImages = !gm.renderImages
			} else {
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
	title := "── General Options "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
	b.WriteString("\n\n")

	imgCheck := "[ ]"
	if gm.renderImages {
		imgCheck = "[x]"
	}
	if gm.cursor == 0 {
		imgCheck = cursorStyle.Render(imgCheck)
	}
	b.WriteString(formLabelStyle.Render("Render images:"))
	b.WriteString(" " + imgCheck)
	b.WriteString("\n\n")

	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	if gm.cursor == 1 {
		b.WriteString(cursorStyle.Render("> "))
	} else {
		b.WriteString("  ")
	}
	b.WriteString(formLabelStyle.Render("Distance unit:"))
	b.WriteString(" " + inputStyle.Render(unitVal))

	return b.String()
}
