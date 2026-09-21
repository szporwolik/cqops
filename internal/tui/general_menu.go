package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// generalRowCount is the number of rows in the General settings menu:
// Units, Timezone, and eight toggles.
const generalRowCount = 10

// generalRows is the row style shared by every General settings row.
var generalRows = rowStyle{label: S.FormLabelGen, focused: S.FormFocusedGen}

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
	fm            menuFocus
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

// focusableRows implementation — all ten rows are always visible and none
// carry a textinput.
func (gm *GeneralMenu) rowCount() int        { return generalRowCount }
func (gm *GeneralMenu) rowVisible(int) bool  { return true }
func (gm *GeneralMenu) blurAll()             {}
func (gm *GeneralMenu) focusRow(int) tea.Cmd { return nil }

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
		case "enter":
			gm.done = true
			gm.saved = true
			return gm, nil
		}
		if handled, cmd := gm.fm.onKey(msg, gm, func() tea.Cmd {
			gm.done = true
			gm.saved = true
			return nil
		}); handled {
			return gm, cmd
		}
		switch msg.String() {
		case " ", "space":
			switch gm.fm.row {
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
	infoBox(&b, infoText, infoMaxW)

	rowW := boxW - 5 // truncation safety margin against ANSI-width miscalc
	focused := func(row int) bool { return gm.fm.row == row }

	// Row 0: Units — toggles Metric/Imperial on space.
	unitVal := "Metric"
	if gm.distanceUnit == "imperial" {
		unitVal = "Imperial"
	}
	unitHint := ""
	if focused(0) {
		unitHint = "distance, speed, elevation"
	}
	valueRow(&b, rowW, focused(0), "Units", unitVal, unitHint, generalRows)

	// Row 1: Timezone — shows current value, cycles on space.
	tzHint := ""
	if focused(1) {
		tzHint = "QSO date/time reference"
	}
	valueRow(&b, rowW, focused(1), "Timezone", gm.timezone, tzHint, generalRows)

	// Row 2-9: Checkbox options.
	checkboxRow(&b, rowW, focused(2), "Render partner map", gm.renderMap, "Shows station on world map with bearing", false, generalRows)
	checkboxRow(&b, rowW, focused(3), "Render grayline at partner map", gm.drawGrayline, "Day/night terminator overlay", false, generalRows)
	checkboxRow(&b, rowW, focused(4), "Render partner picture", gm.pictureAtQRZ, "Shows photo from callbook if available", false, generalRows)
	checkboxRow(&b, rowW, focused(5), "Solar data next to QSO form", gm.solarAtQSO, "SFI, A, K indices in QSO pane", false, generalRows)
	checkboxRow(&b, rowW, focused(6), "Use Super Check Partial", gm.useSCP, "Callsign autocomplete from contest logs", false, generalRows)
	checkboxRow(&b, rowW, focused(7), "Use SOTA/POTA/IOTA database", gm.useRef, "Reference lookup for awards", false, generalRows)
	checkboxRow(&b, rowW, focused(8), "Kitty graphics", gm.kittyGraphics, "Experimental — requires Kitty, Ghostty, or WezTerm", false, generalRows)
	checkboxRow(&b, rowW, focused(9), "Debug Mode", gm.debugMode, "Verbose logging for troubleshooting", false, generalRows)

	// Save & Back button at the end of the menu.
	b.WriteString("\n")
	b.WriteString(gm.fm.btn.line("Save & Back", boxW-4))

	body := drawMenuWithHeader("Configuration \u2014 General Settings", b.String(), w)
	return tea.NewView(fillBody(body, contentH))
}
