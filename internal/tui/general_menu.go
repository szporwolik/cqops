package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/config"
)

type GeneralMenu struct {
	renderImages bool
	done         bool
	saved        bool
	goBack       bool
	width        int
	height       int
}

func NewGeneralMenu(cfg *config.Config) *GeneralMenu {
	return &GeneralMenu{renderImages: cfg.RenderImages}
}

func (gm *GeneralMenu) Init() tea.Cmd { return nil }

func (gm *GeneralMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: gm.width, gm.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc": gm.done = true; gm.goBack = true; return gm, nil
		case "ctrl+s", "\x13": gm.done = true; gm.saved = true; return gm, nil
		case " ": gm.renderImages = !gm.renderImages
		}
	}
	return gm, nil
}

func (gm *GeneralMenu) FooterText() string {
	return "Space/Enter to save  Esc to go back"
}

func (gm *GeneralMenu) View() string {
	if gm.done { return "" }
	var b strings.Builder
	b.WriteString(titleStyle.Render("General Options"))
	b.WriteString("\n\n")
	checkbox := "[ ]"
	if gm.renderImages {
		checkbox = "[x] Render images (partners, maps)"
	} else {
		checkbox = "[ ] Render images (partners, maps)"
	}
	b.WriteString(checkbox)
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Space to toggle and save  Ctrl+S to save  Esc to go back"))
	return b.String()
}
