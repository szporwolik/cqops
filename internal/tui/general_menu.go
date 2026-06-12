package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
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
	var b strings.Builder
	b.WriteString(titleStyle.Render("General Options"))
	b.WriteString("\n\n")

	imgCheck := "[ ]"
	imgLine := "[ ] Render maps and partner images"
	if gm.renderImages {
		imgCheck = "[x]"
		imgLine = "[x] Render maps and partner images"
	}
	if gm.cursor == 0 {
		imgLine = cursorStyle.Render(imgCheck) + imgLine[3:]
	}
	b.WriteString(imgLine)
	b.WriteString("\n\n")

	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	opt := "Distance unit: " + unitVal
	if gm.cursor == 1 {
		opt = cursorStyle.Render("> ") + opt
	} else {
		opt = "  " + opt
	}
	b.WriteString(opt)

	return b.String()
}
