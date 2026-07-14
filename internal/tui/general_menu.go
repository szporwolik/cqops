package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type GeneralMenu struct {
	distanceUnit  string
	timezone      string
	tzIndex       int
	renderMap     bool
	drawGrayline  bool
	pictureAtQRZ  bool
	solarAtQSO    bool
	useSCP        bool
	useRef        bool
	debugMode     bool
	kittyGraphics bool
	cursor        int
	done          bool
	saved         bool
	goBack        bool
	width         int
	height        int
}

func NewGeneralMenu(cfg *config.Config) *GeneralMenu {
	du := cfg.General.Units
	if du != "imperial" {
		du = "metric"
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
		distanceUnit:  du,
		timezone:      tz,
		tzIndex:       tzIdx,
		renderMap:     cfg.General.RenderMap,
		drawGrayline:  cfg.General.DrawGrayline,
		pictureAtQRZ:  cfg.General.PictureAtPartnerPane,
		solarAtQSO:    cfg.General.SolarAtQSOPane,
		useSCP:        cfg.General.UseSCP,
		useRef:        cfg.General.UseRef,
		debugMode:     cfg.General.Debug,
		kittyGraphics: cfg.General.KittyGraphics,
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
				gm.cursor = 10
			}
		case "down", "j":
			if gm.cursor < 10 {
				gm.cursor++
			} else {
				gm.cursor = 0
			}
		case " ", "space":
			switch gm.cursor {
			case 0:
				if gm.distanceUnit == "metric" {
					gm.distanceUnit = "imperial"
				} else {
					gm.distanceUnit = "metric"
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
				gm.useSCP = !gm.useSCP
			case 7:
				gm.useRef = !gm.useRef
			case 8:
				gm.kittyGraphics = !gm.kittyGraphics
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

	// --- Info box (same pattern as callbook menu) ---
	infoMaxW := boxW - 6
	if infoMaxW < 30 {
		infoMaxW = 30
	}
	if infoMaxW > partnerMapMaxW-10 {
		infoMaxW = partnerMapMaxW - 10
	}
	infoText := "Match these settings to your hardware \u2014 " +
		"partner map, grayline, and photo rendering " +
		"can increase CPU load on low-end machines. " +
		"Kitty graphics require a compatible terminal " +
		"(Kitty, Ghostty, or WezTerm)."
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
	b.WriteString(infoBox)
	b.WriteString("\n")

	// Row 0: Units — toggles Metric/Imperial on space.
	unitVal := "Metric"
	if gm.distanceUnit == "imperial" {
		unitVal = "Imperial"
	}
	unitHint := ""
	if gm.cursor == 0 {
		unitHint = "distance, speed, elevation"
	}
	gm.renderSettingRow(&b, boxW, 0, "Units", unitVal, unitHint)

	// Row 1: Timezone — shows current value, cycles on space.
	tzHint := ""
	if gm.cursor == 1 {
		tzHint = "QSO date/time reference"
	}
	gm.renderSettingRow(&b, boxW, 1, "Timezone", gm.timezone, tzHint)

	// Row 2-4: Checkbox options.
	gm.renderCheckbox(&b, boxW, 2, "Render partner map", "Shows station on world map with bearing", gm.renderMap)
	gm.renderCheckbox(&b, boxW, 3, "Render grayline at partner map", "Day/night terminator overlay", gm.drawGrayline)
	gm.renderCheckbox(&b, boxW, 4, "Render partner picture", "Shows photo from callbook if available", gm.pictureAtQRZ)
	gm.renderCheckbox(&b, boxW, 5, "Solar data next to QSO form", "SFI, A, K indices in QSO pane", gm.solarAtQSO)
	gm.renderCheckbox(&b, boxW, 6, "Use Super Check Partial", "Callsign autocomplete from contest logs", gm.useSCP)
	gm.renderCheckbox(&b, boxW, 7, "Use SOTA/POTA/IOTA database", "Reference lookup for awards", gm.useRef)
	gm.renderKittyCheckbox(&b, boxW, 8, "Kitty graphics", "Experimental — requires Kitty, Ghostty, or WezTerm", gm.kittyGraphics)
	gm.renderCheckbox(&b, boxW, 9, "Debug Mode", "Verbose logging for troubleshooting", gm.debugMode)

	body := drawMenuWithHeader("Configuration \u2014 General Settings", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}

func (gm *GeneralMenu) renderCheckbox(b *strings.Builder, boxW, cursor int, label, hint string, checked bool) {
	cb := "[ ]"
	if checked {
		cb = "[x]"
	}
	prefix := S.FormPrefixOff.Render("  ")
	lbl := S.FormLabelGen.Align(lipgloss.Left).Render(label)
	if gm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedGen.Align(lipgloss.Left).Render(label)
		cb = CursorStyle.Render(cb) + " " + DimStyle.Render("(Space)")
		if hint != "" {
			cb = cb + " " + DimStyle.Render(hint)
		}
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", cb),
		boxW-5))
	b.WriteString("\n")
}

func (gm *GeneralMenu) renderKittyCheckbox(b *strings.Builder, boxW, cursor int, label, hint string, checked bool) {
	cb := "[ ]"
	if checked {
		cb = "[x]"
	}
	prefix := S.FormPrefixOff.Render("  ")
	lbl := S.FormLabelGen.Align(lipgloss.Left).Render(label)
	if gm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedGen.Align(lipgloss.Left).Render(label)
		cb = CursorStyle.Render(cb) + " " + DimStyle.Render("(Space)")
		if hint != "" {
			cb = cb + " " + DimStyle.Render(hint)
		}
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", cb),
		boxW-5))
	b.WriteString("\n")
}

func (gm *GeneralMenu) renderSettingRow(b *strings.Builder, boxW, cursor int, label, value, hint string) {
	prefix := S.FormPrefixOff.Render("  ")
	lbl := S.FormLabelGen.Align(lipgloss.Left).Render(label)
	val := ValueStyle.Render(value)
	if gm.cursor == cursor {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedGen.Align(lipgloss.Left).Render(label)
		val = CursorStyle.Render(value) + " " + DimStyle.Render("(Space)")
		if hint != "" {
			val += " " + DimStyle.Render(hint)
		}
	}
	// Start truncation 5 chars early as safety margin against ANSI-width miscalc.
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val),
		boxW-5))
	b.WriteString("\n")
}
