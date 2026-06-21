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
	renderMap    bool
	drawGrayline bool
	pictureAtQRZ bool
	solarAtQSO   bool
	useCTY       bool
	useSCP       bool
	useRef       bool
	debugMode    bool
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
	return &GeneralMenu{
		distanceUnit: du,
		timezone:     tz,
		tzIndex:      tzIdx,
		renderMap:    cfg.General.RenderMap,
		drawGrayline: cfg.General.DrawGrayline,
		pictureAtQRZ: cfg.General.PictureAtQRZPane,
		solarAtQSO:   cfg.General.SolarAtQSOPane,
		useCTY:       cfg.General.UseCTY,
		useSCP:       cfg.General.UseSCP,
		useRef:       cfg.General.UseRef,
		debugMode:    cfg.General.Debug,
	}
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
			} else {
				gm.cursor = 9
			}
		case "down", "j":
			if gm.cursor < 9 {
				gm.cursor++
			} else {
				gm.cursor = 0
			}
		case " ", "space":
			switch gm.cursor {
			case 0:
				if gm.distanceUnit == "km" {
					gm.distanceUnit = "mi"
				} else {
					gm.distanceUnit = "km"
				}
			case 1:
				gm.tzIndex++
				if gm.tzIndex >= len(config.Timezones) {
					gm.tzIndex = 0
				}
				gm.timezone = config.Timezones[gm.tzIndex]
			case 2:
				gm.renderMap = !gm.renderMap
			case 3:
				gm.drawGrayline = !gm.drawGrayline
			case 4:
				gm.pictureAtQRZ = !gm.pictureAtQRZ
			case 5:
				gm.solarAtQSO = !gm.solarAtQSO
			case 6:
				gm.useCTY = !gm.useCTY
			case 7:
				gm.useSCP = !gm.useSCP
			case 8:
				gm.useRef = !gm.useRef
			case 9:
				gm.debugMode = !gm.debugMode
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

	// Row 0: Distance unit — shows current value, toggles on space.
	unitVal := "Kilometers (km)"
	if gm.distanceUnit == "mi" {
		unitVal = "Miles (mi)"
	}
	gm.renderSettingRow(&b, boxW, 0, "Distance unit", unitVal)

	// Row 1: Timezone — shows current value, cycles on space.
	gm.renderSettingRow(&b, boxW, 1, "Timezone", gm.timezone)

	// Row 2-4: Checkbox options.
	gm.renderCheckbox(&b, boxW, 2, "Render map", gm.renderMap)
	gm.renderCheckbox(&b, boxW, 3, "Draw grayline", gm.drawGrayline)
	gm.renderCheckbox(&b, boxW, 4, "Picture at QRZ pane", gm.pictureAtQRZ)
	gm.renderCheckbox(&b, boxW, 5, "Solar at QSO pane", gm.solarAtQSO)
	gm.renderCheckbox(&b, boxW, 6, "Use CTY.DAT country data", gm.useCTY)
	gm.renderCheckbox(&b, boxW, 7, "Use Super Check Partial", gm.useSCP)
	gm.renderCheckbox(&b, boxW, 8, "Use REF database", gm.useRef)
	gm.renderCheckbox(&b, boxW, 9, "Debug Mode", gm.debugMode)

	body := drawMenuWithHeader("Configuration \u2014 General Settings", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

func (gm *GeneralMenu) renderCheckbox(b *strings.Builder, boxW, cursor int, label string, checked bool) {
	checkbox := "[ ]"
	if checked {
		checkbox = "[x]"
	}
	prefix := "  "
	lbl := S.FormLabelGen.Align(lipgloss.Left).Render(label)
	if gm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedGen.Align(lipgloss.Left).Render(label)
		checkbox = CursorStyle.Render(checkbox)
	}
	b.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", checkbox), boxW))
	b.WriteString("\n")
}

func (gm *GeneralMenu) renderSettingRow(b *strings.Builder, boxW, cursor int, label, value string) {
	prefix := "  "
	lbl := S.FormLabelGen.Align(lipgloss.Left).Render(label)
	lblW := lipgloss.Width(lbl)
	// Values get remaining space: boxW minus 2 (prefix) minus lblW (label) minus 1 (space) minus 6 (border+padding).
	valW := boxW - 2 - lblW - 1 - 6
	if valW < 8 {
		valW = 8
	}
	val := ValueStyle.Width(valW).MaxWidth(valW).Inline(true).Render(value)
	if gm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedGen.Align(lipgloss.Left).Render(label)
		lblW = lipgloss.Width(lbl)
		valW = boxW - 2 - lblW - 1 - 6
		if valW < 8 {
			valW = 8
		}
		val = CursorStyle.Width(valW).MaxWidth(valW).Inline(true).Render(value)
	}
	b.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val), boxW))
	b.WriteString("\n")
}
