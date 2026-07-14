package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

type StationForm struct {
	Name        textinput.Model
	Callsign    textinput.Model
	Operator    textinput.Model // display-only; driven by opFocus/opIdx + SetOperators
	Locator     textinput.Model
	SOTARef     textinput.Model
	POTARef     textinput.Model
	WWFFRef     textinput.Model
	IARURegion  int
	iaruFocus   bool // true when IARU region selector has focus
	Continent   string
	contIdx     int  // index into continent list for cycling
	contFocus   bool // true when continent selector has focus
	CQZone      textinput.Model
	ITUZone     textinput.Model
	DXCC        textinput.Model
	SIG         textinput.Model
	SIGInfo     textinput.Model
	WlEnabled   bool
	wlCbFocus   bool // true when the WL checkbox has focus
	wlBtnFocus  int  // 0=none, 1=Update, 2=Test
	WlURL       textinput.Model
	WlKey       textinput.Model
	WlStationID textinput.Model
	// APRS fields.
	AprsEnabled      bool
	aprsCbFocus      bool // true when APRS checkbox has focus
	aprsBtnFocus     int  // 0=none, 1=Test
	AprsServer       textinput.Model
	AprsPasscode     textinput.Model
	AprsRadiusKm     textinput.Model
	AprsSendLoc      bool
	aprsSendLocFocus bool
	AprsCallsign     textinput.Model
	AprsIntervalMin  textinput.Model
	AprsSymbol       textinput.Model
	AprsComment      textinput.Model
	width            int // terminal width for responsive layout

	// GPS Grid — use GPS-derived grid when checked and GPS has fix.
	GPSGrid      bool
	gpsGridFocus bool
	HideGPSGrid  bool // set true to hide checkbox (e.g. in wizard)

	// Operator cycling (Space-toggleable, like Continent/IARU).
	operators []config.Operator
	opIdx     int  // index into operators; -1 = None
	opFocus   bool // true when operator selector has focus
}

// Wavelog button action messages sent when a button is activated via Enter.
type wlUpdateAction struct{}
type wlTestAction struct{}
type wlCycleStation struct{}

// APRS button action message.
type aprsTestAction struct{}

// scrollFormToEnd is emitted when a toggle (APRS/Wavelog checkbox) reveals
// new fields below the fold; the parent should scroll the viewport to the end.
type scrollFormToEnd struct{}

func NewStationForm(callsignPlaceholder, opPlaceholder, locatorPlaceholder string) *StationForm {
	mkTI := func(limit int, width int, placeholder string) textinput.Model {
		ti := newTextinput()
		ti.CharLimit = limit
		ti.SetWidth(width)
		ti.Placeholder = placeholder
		return ti
	}

	nm := mkTI(30, 28, "e.g. Home QTH, Field Day")
	nm.Focus()
	cs := mkTI(20, 28, callsignPlaceholder)
	cs.Blur()
	op := mkTI(20, 28, opPlaceholder)
	lc := mkTI(10, 28, locatorPlaceholder)
	sr := mkTI(20, 28, "e.g. SP/TA-001")
	pr := mkTI(20, 28, "e.g. SP-0001")
	wr := mkTI(20, 28, "e.g. SPFF-0001")
	cz := mkTI(4, 28, "1-40")
	iz := mkTI(4, 28, "1-90")
	dx := mkTI(6, 28, "e.g. 269")
	sg := mkTI(10, 28, "e.g. SOTA")
	si := mkTI(20, 28, "e.g. SP/TQ-001")

	wu := mkTI(80, 28, "https://log.example.com")
	wk := mkTI(64, 28, "Wavelog API key")
	ws := mkTI(80, 60, "press Update to fetch")

	// APRS defaults.
	asrv := mkTI(60, 28, "euro.aprs2.net:14580")
	apc := mkTI(20, 28, "APRS passcode")
	arad := mkTI(5, 28, "50")
	acall := mkTI(12, 28, "N0CALL-10")
	aint := mkTI(3, 28, "15")
	asym := mkTI(6, 28, "/-")
	acmt := mkTI(40, 28, "Field Day")

	return &StationForm{
		Name:            nm,
		Callsign:        cs,
		Operator:        op,
		Locator:         lc,
		SOTARef:         sr,
		POTARef:         pr,
		WWFFRef:         wr,
		Continent:       "EU",
		CQZone:          cz,
		ITUZone:         iz,
		DXCC:            dx,
		SIG:             sg,
		SIGInfo:         si,
		WlURL:           wu,
		WlKey:           wk,
		WlStationID:     ws,
		AprsServer:      asrv,
		AprsPasscode:    apc,
		AprsRadiusKm:    arad,
		AprsCallsign:    acall,
		AprsIntervalMin: aint,
		AprsSymbol:      asym,
		AprsComment:     acmt,
		opIdx:           -1,
	}
}

// SetOperators provides the operator list for the Space-toggleable selector.
func (f *StationForm) SetOperators(ops []config.Operator) {
	f.operators = ops
	// Try to keep the current selection if it still exists.
	if f.opIdx >= 0 && f.opIdx < len(ops) {
		f.Operator.SetValue(config.OperatorDisplayName(&ops[f.opIdx]))
	} else if f.opIdx < 0 || len(ops) == 0 {
		f.opIdx = -1
		f.Operator.SetValue("")
	}
}

// updateOperatorDisplay sets the Operator textinput to show the current selection.
func (f *StationForm) updateOperatorDisplay() {
	if f.opIdx >= 0 && f.opIdx < len(f.operators) {
		f.Operator.SetValue(config.OperatorDisplayName(&f.operators[f.opIdx]))
	} else {
		f.Operator.SetValue("")
		f.opIdx = -1
	}
}

// SelectedOperatorCallsign returns the callsign of the selected operator, or "" for none.
func (f *StationForm) SelectedOperatorCallsign() string {
	if f.opIdx >= 0 && f.opIdx < len(f.operators) {
		return f.operators[f.opIdx].Callsign
	}
	return ""
}

func (f *StationForm) Update(msg tea.KeyPressMsg) {
	switch {
	case f.Name.Focused():
		f.Name, _ = f.Name.Update(msg)
	case f.Callsign.Focused():
		f.Callsign, _ = f.Callsign.Update(msg)
		f.Callsign.SetValue(strings.ToUpper(f.Callsign.Value()))
	case f.opFocus:
		if msg.String() == " " || msg.String() == "space" || msg.String() == "enter" {
			if len(f.operators) == 0 {
				f.opIdx = -1
			} else {
				f.opIdx++
				if f.opIdx >= len(f.operators) {
					f.opIdx = -1 // wrap to None
				}
			}
			f.updateOperatorDisplay()
		}
	case f.Locator.Focused():
		f.Locator, _ = f.Locator.Update(msg)
		f.Locator.SetValue(formatLocator(f.Locator.Value()))
	case f.SOTARef.Focused():
		f.SOTARef, _ = f.SOTARef.Update(msg)
		f.SOTARef.SetValue(strings.ToUpper(f.SOTARef.Value()))
	case f.POTARef.Focused():
		f.POTARef, _ = f.POTARef.Update(msg)
		f.POTARef.SetValue(strings.ToUpper(f.POTARef.Value()))
	case f.WWFFRef.Focused():
		f.WWFFRef, _ = f.WWFFRef.Update(msg)
		f.WWFFRef.SetValue(strings.ToUpper(f.WWFFRef.Value()))
	case f.iaruFocus:
		if msg.String() == " " || msg.String() == "space" || msg.String() == "enter" {
			f.IARURegion++
			if f.IARURegion > 3 {
				f.IARURegion = 1
			}
		}
	case f.contFocus:
		if msg.String() == " " || msg.String() == "space" || msg.String() == "enter" {
			continents := continentList()
			f.contIdx = (f.contIdx + 1) % len(continents)
			f.Continent = continents[f.contIdx]
		}
	case f.CQZone.Focused():
		f.CQZone, _ = f.CQZone.Update(msg)
	case f.ITUZone.Focused():
		f.ITUZone, _ = f.ITUZone.Update(msg)
	case f.DXCC.Focused():
		f.DXCC, _ = f.DXCC.Update(msg)
		f.DXCC.SetValue(strings.ToUpper(f.DXCC.Value()))
	case f.SIG.Focused():
		f.SIG, _ = f.SIG.Update(msg)
		f.SIG.SetValue(strings.ToUpper(f.SIG.Value()))
	case f.SIGInfo.Focused():
		f.SIGInfo, _ = f.SIGInfo.Update(msg)
		f.SIGInfo.SetValue(strings.ToUpper(f.SIGInfo.Value()))
	case f.WlURL.Focused():
		f.WlURL, _ = f.WlURL.Update(msg)
	case f.WlKey.Focused():
		f.WlKey, _ = f.WlKey.Update(msg)
	case f.WlStationID.Focused():
		// Station ID is read-only — updated via Update button or Space cycle.

	// APRS fields.
	case f.AprsCallsign.Focused():
		f.AprsCallsign, _ = f.AprsCallsign.Update(msg)
		f.AprsCallsign.SetValue(strings.ToUpper(f.AprsCallsign.Value()))
	case f.AprsPasscode.Focused():
		f.AprsPasscode, _ = f.AprsPasscode.Update(msg)
	case f.AprsRadiusKm.Focused():
		f.AprsRadiusKm, _ = f.AprsRadiusKm.Update(msg)
	case f.AprsIntervalMin.Focused():
		f.AprsIntervalMin, _ = f.AprsIntervalMin.Update(msg)
	case f.AprsSymbol.Focused():
		f.AprsSymbol, _ = f.AprsSymbol.Update(msg)
	case f.AprsComment.Focused():
		f.AprsComment, _ = f.AprsComment.Update(msg)
	}
}

func (f *StationForm) NextInput() {
	switch {
	case f.Name.Focused():
		f.Name.Blur()
		f.Callsign.Focus()
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.Locator.Focus()
	case f.opFocus:
		f.opFocus = false
		f.SOTARef.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		if f.HideGPSGrid {
			f.iaruFocus = true
		} else {
			f.gpsGridFocus = true
		}
	case f.gpsGridFocus:
		f.gpsGridFocus = false
		f.iaruFocus = true
	case f.iaruFocus:
		f.iaruFocus = false
		f.contFocus = true
	case f.contFocus:
		f.contFocus = false
		f.opFocus = true
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.POTARef.Focus()
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.WWFFRef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.CQZone.Focus()
	case f.CQZone.Focused():
		f.CQZone.Blur()
		f.ITUZone.Focus()
	case f.ITUZone.Focused():
		f.ITUZone.Blur()
		f.DXCC.Focus()
	case f.DXCC.Focused():
		f.DXCC.Blur()
		f.SIG.Focus()
	case f.SIG.Focused():
		f.SIG.Blur()
		f.SIGInfo.Focus()
	case f.SIGInfo.Focused():
		f.SIGInfo.Blur()
		f.wlCbFocus = true
	case f.wlCbFocus:
		f.wlCbFocus = false
		if f.WlEnabled {
			f.WlURL.Focus()
		} else {
			f.Name.Focus()
		}
	case f.WlURL.Focused():
		f.WlURL.Blur()
		f.WlKey.Focus()
	case f.WlKey.Focused():
		f.WlKey.Blur()
		f.WlStationID.Focus()
	case f.WlStationID.Focused():
		f.WlStationID.Blur()
		f.wlBtnFocus = 1
	case f.wlBtnFocus == 1:
		f.wlBtnFocus = 0
		if f.HideGPSGrid {
			f.Name.Focus()
		} else {
			f.aprsCbFocus = true
		}
	// APRS section — skipped when HideGPSGrid.
	case f.aprsCbFocus:
		f.aprsCbFocus = false
		if f.AprsEnabled {
			f.AprsCallsign.Focus()
		} else {
			f.Name.Focus()
		}
	case f.AprsCallsign.Focused():
		f.AprsCallsign.Blur()
		f.AprsPasscode.Focus()
	case f.AprsPasscode.Focused():
		f.AprsPasscode.Blur()
		f.aprsSendLocFocus = true
	case f.aprsSendLocFocus:
		f.aprsSendLocFocus = false
		f.AprsIntervalMin.Focus()
	case f.AprsIntervalMin.Focused():
		f.AprsIntervalMin.Blur()
		f.AprsRadiusKm.Focus()
	case f.AprsRadiusKm.Focused():
		f.AprsRadiusKm.Blur()
		f.AprsSymbol.Focus()
	case f.AprsSymbol.Focused():
		f.AprsSymbol.Blur()
		f.AprsComment.Focus()
	case f.AprsComment.Focused():
		f.AprsComment.Blur()
		f.aprsBtnFocus = 1
	case f.aprsBtnFocus == 1:
		f.aprsBtnFocus = 0
		f.Name.Focus()
	}
}

func (f *StationForm) PrevInput() {
	switch {
	case f.Name.Focused():
		f.Name.Blur()
		if f.HideGPSGrid {
			f.wlBtnFocus = 1
		} else if f.AprsEnabled {
			f.aprsBtnFocus = 1
		} else {
			f.aprsCbFocus = true
		}
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.Name.Focus()
	// APRS section — backwards.
	case f.aprsBtnFocus == 1:
		f.aprsBtnFocus = 0
		f.AprsComment.Focus()
	case f.AprsComment.Focused():
		f.AprsComment.Blur()
		f.AprsSymbol.Focus()
	case f.AprsSymbol.Focused():
		f.AprsSymbol.Blur()
		f.AprsRadiusKm.Focus()
	case f.AprsRadiusKm.Focused():
		f.AprsRadiusKm.Blur()
		f.AprsIntervalMin.Focus()
	case f.AprsIntervalMin.Focused():
		f.AprsIntervalMin.Blur()
		f.aprsSendLocFocus = true
	case f.aprsSendLocFocus:
		f.aprsSendLocFocus = false
		f.AprsPasscode.Focus()
	case f.AprsPasscode.Focused():
		f.AprsPasscode.Blur()
		f.AprsCallsign.Focus()
	case f.AprsCallsign.Focused():
		f.AprsCallsign.Blur()
		f.aprsCbFocus = true
	case f.aprsCbFocus:
		f.aprsCbFocus = false
		if f.WlEnabled {
			f.wlBtnFocus = 1
		} else {
			f.wlCbFocus = true
		}
	// Wavelog section — backwards.
	case f.wlBtnFocus == 1:
		f.wlBtnFocus = 0
		f.WlStationID.Focus()
	case f.opFocus:
		f.opFocus = false
		f.contFocus = true
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Callsign.Focus()
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.opFocus = true
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.SOTARef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.POTARef.Focus()
	case f.iaruFocus:
		f.iaruFocus = false
		if f.HideGPSGrid {
			f.Locator.Focus()
		} else {
			f.gpsGridFocus = true
		}
	case f.gpsGridFocus:
		f.gpsGridFocus = false
		f.Locator.Focus()
	case f.contFocus:
		f.contFocus = false
		f.iaruFocus = true
	case f.CQZone.Focused():
		f.CQZone.Blur()
		f.WWFFRef.Focus()
	case f.ITUZone.Focused():
		f.ITUZone.Blur()
		f.CQZone.Focus()
	case f.DXCC.Focused():
		f.DXCC.Blur()
		f.ITUZone.Focus()
	case f.wlCbFocus:
		f.wlCbFocus = false
		f.SIGInfo.Focus()
	case f.SIG.Focused():
		f.SIG.Blur()
		f.DXCC.Focus()
	case f.SIGInfo.Focused():
		f.SIGInfo.Blur()
		f.SIG.Focus()
	case f.WlURL.Focused():
		f.WlURL.Blur()
		f.wlCbFocus = true
	case f.WlKey.Focused():
		f.WlKey.Blur()
		f.WlURL.Focus()
	case f.WlStationID.Focused():
		f.WlStationID.Blur()
		f.WlKey.Focus()
	}
}

func (f *StationForm) OnLastField() bool {
	return f.aprsBtnFocus == 1
}

func (f *StationForm) BlurAll() {
	blurTextinputs(&f.Name, &f.Callsign, &f.Operator, &f.Locator, &f.SOTARef, &f.POTARef, &f.WWFFRef,
		&f.CQZone, &f.ITUZone, &f.DXCC, &f.SIG, &f.SIGInfo,
		&f.WlURL, &f.WlKey, &f.WlStationID,
		&f.AprsServer, &f.AprsPasscode, &f.AprsRadiusKm, &f.AprsCallsign, &f.AprsIntervalMin, &f.AprsSymbol, &f.AprsComment)
	f.wlCbFocus = false
	f.aprsCbFocus = false
	f.aprsSendLocFocus = false
	f.gpsGridFocus = false
	f.iaruFocus = false
	f.contFocus = false
	f.opFocus = false
	f.wlBtnFocus = 0
	f.aprsBtnFocus = 0
}

func (f *StationForm) Values() (name, callsign, operator, locator, sotaRef, potaRef, wwffRef string,
	wlEnabled bool, wlURL, wlKey, wlStationID string, iaruRegion, cqZone, ituZone, dxcc int,
	sig, sigInfo, continent string) {

	var cz, iz, dx int
	fmt.Sscanf(strings.TrimSpace(f.CQZone.Value()), "%d", &cz)
	fmt.Sscanf(strings.TrimSpace(f.ITUZone.Value()), "%d", &iz)
	fmt.Sscanf(strings.TrimSpace(f.DXCC.Value()), "%d", &dx)

	return strings.TrimSpace(f.Name.Value()),
		strings.ToUpper(strings.TrimSpace(f.Callsign.Value())),
		f.SelectedOperatorCallsign(),
		formatLocator(f.Locator.Value()),
		strings.TrimSpace(f.SOTARef.Value()),
		strings.TrimSpace(f.POTARef.Value()),
		strings.TrimSpace(f.WWFFRef.Value()),
		f.WlEnabled,
		strings.TrimSpace(f.WlURL.Value()),
		strings.TrimSpace(f.WlKey.Value()),
		strings.TrimSpace(f.WlStationID.Value()),
		f.IARURegion,
		cz, iz, dx,
		strings.ToUpper(strings.TrimSpace(f.SIG.Value())),
		strings.ToUpper(strings.TrimSpace(f.SIGInfo.Value())),
		f.Continent
}

func (f *StationForm) SetValues(name, callsign, operator, locator, sotaRef, potaRef, wwffRef string, iaruRegion, cqZone, ituZone, dxcc int, sig, sigInfo, continent string) {
	f.Name.SetValue(name)
	f.Callsign.SetValue(callsign)
	// Set operator from callsign lookup.
	f.opIdx = -1
	if operator != "" {
		for i, op := range f.operators {
			if strings.EqualFold(op.Callsign, operator) {
				f.opIdx = i
				break
			}
		}
	}
	f.updateOperatorDisplay()
	f.Locator.SetValue(locator)
	f.SOTARef.SetValue(sotaRef)
	f.POTARef.SetValue(potaRef)
	f.WWFFRef.SetValue(wwffRef)
	f.IARURegion = iaruRegion
	if cqZone > 0 {
		f.CQZone.SetValue(fmt.Sprintf("%d", cqZone))
	} else {
		f.CQZone.SetValue("")
	}
	if ituZone > 0 {
		f.ITUZone.SetValue(fmt.Sprintf("%d", ituZone))
	} else {
		f.ITUZone.SetValue("")
	}
	if dxcc > 0 {
		f.DXCC.SetValue(fmt.Sprintf("%d", dxcc))
	} else {
		f.DXCC.SetValue("")
	}
	f.SIG.SetValue(sig)
	f.SIGInfo.SetValue(sigInfo)
	if continent != "" {
		f.Continent = continent
		// Sync contIdx to match the continent.
		for i, c := range continentList() {
			if c == continent {
				f.contIdx = i
				break
			}
		}
	}
}

func (f *StationForm) SetWavelogValues(wl *config.WavelogConfig) {
	if wl != nil {
		f.WlEnabled = wl.Enabled
		f.WlURL.SetValue(wl.URL)
		f.WlKey.SetValue(wl.APIKey)
		f.WlStationID.SetValue(wl.StationProfileID)
	} else {
		f.WlEnabled = false
		f.WlURL.SetValue("")
		f.WlKey.SetValue("")
		f.WlStationID.SetValue("")
	}
}

// APRSValues returns the APRS configuration from the form fields.
func (f *StationForm) APRSValues() *config.APRSConfig {
	rad, _ := parseInt(f.AprsRadiusKm.Value())
	iv, _ := parseInt(f.AprsIntervalMin.Value())
	if iv < 5 {
		iv = 5
	}
	if iv > 180 {
		iv = 180
	}
	return &config.APRSConfig{
		Enabled:      f.AprsEnabled,
		Callsign:     strings.ToUpper(strings.TrimSpace(f.AprsCallsign.Value())),
		Passcode:     strings.TrimSpace(f.AprsPasscode.Value()),
		RadiusKm:     rad,
		SendLocation: f.AprsSendLoc,
		IntervalMin:  iv,
		Symbol:       strings.TrimSpace(f.AprsSymbol.Value()),
		Comment:      strings.TrimSpace(f.AprsComment.Value()),
	}
}

// SetAPRSValues populates the form fields from an APRS config.
func (f *StationForm) SetAPRSValues(aprs *config.APRSConfig) {
	if aprs != nil {
		f.AprsEnabled = aprs.Enabled
		f.AprsCallsign.SetValue(aprs.Callsign)
		f.AprsPasscode.SetValue(aprs.Passcode)
		if aprs.RadiusKm > 0 {
			f.AprsRadiusKm.SetValue(fmt.Sprintf("%d", aprs.RadiusKm))
		} else {
			f.AprsRadiusKm.SetValue("50")
		}
		f.AprsSendLoc = aprs.SendLocation
		if aprs.IntervalMin >= 5 {
			f.AprsIntervalMin.SetValue(fmt.Sprintf("%d", aprs.IntervalMin))
		} else {
			f.AprsIntervalMin.SetValue("15")
		}
		f.AprsSymbol.SetValue(aprs.Symbol)
		f.AprsComment.SetValue(aprs.Comment)
	} else {
		f.AprsEnabled = false
		f.AprsCallsign.SetValue("")
		f.AprsPasscode.SetValue("")
		f.AprsRadiusKm.SetValue("50")
		f.AprsSendLoc = false
		f.AprsIntervalMin.SetValue("15")
		f.AprsSymbol.SetValue("/-")
		f.AprsComment.SetValue("")
	}
}

// parseInt is a small helper to read an integer from a string value.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	return n, err
}

func (f *StationForm) View() tea.View {
	availW := f.width
	if availW < 40 {
		availW = 80
	}

	type fieldDef struct {
		label string
		ti    *textinput.Model
	}
	var b strings.Builder

	// Station name
	b.WriteString(f.renderFieldLine("Name:", &f.Name, availW))
	// Callsign
	b.WriteString(f.renderFieldLine("Callsign:", &f.Callsign, availW))
	// Grid locator
	b.WriteString(f.renderFieldLine("Grid locator:", &f.Locator, availW))
	// GPS Grid checkbox — below grid locator. Hidden in wizard.
	if !f.HideGPSGrid {
		gpsCb := "[ ]"
		if f.GPSGrid {
			gpsCb = "[x]"
		}
		gpsPrefix := "  "
		gpsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Grid from GPS:")
		if f.gpsGridFocus {
			gpsPrefix = S.FormPrefixOn.Render("> ")
			gpsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Grid from GPS:")
			gpsCb = CursorStyle.Render(gpsCb) + " " + DimStyle.Render("(Space)")
			gpsCb += " " + DimStyle.Render("Auto-fill grid from GPS when available")
		}
		b.WriteString(padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, gpsPrefix, gpsLabel, " ", gpsCb), availW))
		b.WriteString("\n")
	}

	// IARU Region display (focusable, Space/Enter to cycle) — right after grid.
	iaruLabel := "IARU Region:"
	if f.IARURegion < 1 || f.IARURegion > 3 {
		f.IARURegion = 1
	}
	iaruVal := fmt.Sprintf("%d — %s", f.IARURegion, iaruRegionName(f.IARURegion))
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(iaruLabel)
	val := ValueStyle.Render(iaruVal)
	if f.iaruFocus {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(iaruLabel)
		val = CursorStyle.Render(iaruVal) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val),
		availW))
	b.WriteString("\n")

	// Continent selector — focusable, Space/Enter to cycle.
	contLabel := "Continent:"
	contVal := f.Continent + " — " + continentName(f.Continent)
	cPrefix := "  "
	cLbl := S.FormLabelWide.Align(lipgloss.Left).Render(contLabel)
	cVal := ValueStyle.Render(contVal)
	if f.contFocus {
		cPrefix = S.FormPrefixOn.Render("> ")
		cLbl = S.FormFocusedWide.Align(lipgloss.Left).Render(contLabel)
		cVal = CursorStyle.Render(contVal) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, cPrefix, cLbl, " ", cVal),
		availW))
	b.WriteString("\n")

	// Operator selector — Space-toggleable, like Continent/IARU.
	opLabel := "Operator (opt):"
	var opVal string
	if f.opIdx >= 0 && f.opIdx < len(f.operators) {
		opVal = config.OperatorDisplayName(&f.operators[f.opIdx])
	} else {
		opVal = DimStyle.Render("None")
	}
	opPrefix := "  "
	opLbl := S.FormLabelWide.Align(lipgloss.Left).Render(opLabel)
	displayVal := ValueStyle.Render(opVal)
	if f.opFocus {
		opPrefix = S.FormPrefixOn.Render("> ")
		opLbl = S.FormFocusedWide.Align(lipgloss.Left).Render(opLabel)
		if f.opIdx >= 0 {
			displayVal = CursorStyle.Render(opVal) + " " + DimStyle.Render("(Space)")
		} else {
			displayVal = CursorStyle.Render(DimStyle.Render("None")) + " " + DimStyle.Render("(Space)")
		}
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, opPrefix, opLbl, " ", displayVal),
		availW))
	b.WriteString("\n")

	// Remaining text fields.
	remFields := []fieldDef{
		{"SOTA Ref (opt):", &f.SOTARef},
		{"POTA Ref (opt):", &f.POTARef},
		{"WWFF Ref (opt):", &f.WWFFRef},
	}
	for _, field := range remFields {
		b.WriteString(f.renderFieldLine(field.label, field.ti, availW))
	}

	// CQ Zone, ITU Zone, DXCC, SIG, SIG Info — text inputs.
	zoneFields := []fieldDef{
		{"CQ Zone (opt):", &f.CQZone},
		{"ITU Zone (opt):", &f.ITUZone},
		{"DXCC ID (opt):", &f.DXCC},
		{"SIG (opt):", &f.SIG},
		{"SIG Info (opt):", &f.SIGInfo},
	}
	for _, field := range zoneFields {
		b.WriteString(f.renderFieldLine(field.label, field.ti, availW))
	}

	// Wavelog checkbox
	wlCheckbox := "[ ]"
	if f.WlEnabled {
		wlCheckbox = "[x]"
	}
	wlCbPrefix := "  "
	wlCbLabel := S.FormLabelWide.Align(lipgloss.Left).Render("Wavelog:")
	if f.wlCbFocus {
		wlCbPrefix = S.FormPrefixOn.Render("> ")
		wlCbLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("Wavelog:")
		wlCheckbox = CursorStyle.Render(wlCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, wlCbPrefix, wlCbLabel, " ", wlCheckbox),
		availW))
	b.WriteString("\n")

	if f.WlEnabled {
		wlFields := []fieldDef{
			{"  API URL:", &f.WlURL},
			{"  API Key:", &f.WlKey},
		}
		for _, field := range wlFields {
			b.WriteString(f.renderFieldLine(field.label, field.ti, availW))
		}

		// Station ID — read-only, with (Space) hint and truncation.
		{
			const labelW = 2 + 17
			raw := strings.TrimSpace(f.WlStationID.Value())
			prefix := "  "
			lbl := S.FormLabelWide.Align(lipgloss.Left).Render("  Station ID:")
			// Reserve space for label + (Space) hint + safety margin.
			reserved := labelW + 1 + len(" (Space)") + 5
			maxVW := availW - reserved
			if maxVW < 6 {
				maxVW = 6
			}
			var val string
			if f.WlStationID.Focused() {
				prefix = S.FormPrefixOn.Render("> ")
				lbl = S.FormFocusedWide.Align(lipgloss.Left).Render("  Station ID:")
				val = CursorStyle.Render(truncateText(raw, maxVW)) + " " + DimStyle.Render("(Space)")
			} else if raw == "" {
				val = DimStyle.Render("\u2014")
			} else {
				val = ValueStyle.Render(truncateText(raw, maxVW))
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val),
				availW-5))
			b.WriteString("\n")
		}

		// Button helper — fixed padding so buttons never shift on focus.
		renderBtn := func(focusVal int, text, hint string) {
			prefix := "    "
			styled := InputStyle.Render(text)
			if f.wlBtnFocus == focusVal {
				prefix = S.FormPrefixOn.Render("> ") + "  "
				styled = CursorStyle.Render(text)
			}
			line := prefix + styled + " " + DimStyle.Render(hint)
			b.WriteString(padOrTrunc(line, availW))
			b.WriteString("\n")
		}
		renderBtn(1, "[ Update ]", "fetch stations from Wavelog")
	}

	// APRS section — hidden in wizard.
	if !f.HideGPSGrid {
		// APRS checkbox
		aprsCheckbox := "[ ]"
		if f.AprsEnabled {
			aprsCheckbox = "[x]"
		}
		aprsCbPrefix := "  "
		aprsCbLabel := S.FormLabelWide.Align(lipgloss.Left).Render("APRS:")
		if f.aprsCbFocus {
			aprsCbPrefix = S.FormPrefixOn.Render("> ")
			aprsCbLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("APRS:")
			aprsCheckbox = CursorStyle.Render(aprsCheckbox) + " " + DimStyle.Render("(Space)")
			aprsCheckbox += " " + DimStyle.Render("Note: APRS must be enabled in the Integrations menu")
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, aprsCbPrefix, aprsCbLabel, " ", aprsCheckbox),
			availW))
		b.WriteString("\n")

		if f.AprsEnabled {
			aprsFields := []fieldDef{
				{"  Callsign:", &f.AprsCallsign},
				{"  Passcode:", &f.AprsPasscode},
			}
			for _, field := range aprsFields {
				b.WriteString(f.renderFieldLine(field.label, field.ti, availW))
			}
			// TX Beacon checkbox.
			locCheckbox := "[ ]"
			if f.AprsSendLoc {
				locCheckbox = "[x]"
			}
			locPrefix := "  "
			locLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Send beacons:")
			if f.aprsSendLocFocus {
				locPrefix = S.FormPrefixOn.Render("> ")
				locLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Send beacons:")
				locCheckbox = CursorStyle.Render(locCheckbox) + " " + DimStyle.Render("(Space)")
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, locPrefix, locLabel, " ", locCheckbox),
				availW))
			b.WriteString("\n")

			aprsFields2 := []fieldDef{
				{"  Interval (min):", &f.AprsIntervalMin},
				{"  Radius (km):", &f.AprsRadiusKm},
				{"  Symbol:", &f.AprsSymbol},
				{"  Comment:", &f.AprsComment},
			}
			for _, field := range aprsFields2 {
				b.WriteString(f.renderFieldLine(field.label, field.ti, availW))
			}

			// Test button.
			prefix := "    "
			styled := InputStyle.Render("[ Test ]")
			hint := DimStyle.Render("verify APRS connection")
			if f.aprsBtnFocus == 1 {
				prefix = S.FormPrefixOn.Render("> ") + "  "
				styled = CursorStyle.Render("[ Test ]")
			}
			line := prefix + styled + " " + hint
			b.WriteString(padOrTrunc(line, availW))
			b.WriteString("\n")
		}
	}

	return tea.NewView(b.String())
}

func (f *StationForm) renderFieldLine(label string, ti *textinput.Model, availW int) string {
	focused := ti.Focused()
	raw := strings.TrimSpace(ti.Value())

	// labelW: 2-char prefix + 17-char label (FormLabelWide/FormFocusedWide).
	const labelW = 2 + 17
	const maxVW = 40 // max value width, matches QSO form

	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	vw := availW - labelW - 1
	if vw < 3 {
		vw = 3
	}
	if vw > maxVW {
		vw = maxVW
	}

	var val string
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		ti.SetWidth(vw)
		if lipgloss.Width(raw) > vw {
			ti.SetWidth(vw - 1)
		}
		ti.SetCursor(ti.Position())
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
	} else {
		val = ValueStyle.Render(truncateText(raw, vw))
	}
	return padOrTrunc(lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val), availW) + "\n"
}

// HandlePaste forwards clipboard-paste content to the currently focused
// text input field. Non-text focus states (opFocus, iaruFocus, contFocus,
// wlCbFocus, wlBtnFocus) are ignored — paste only makes sense for editable
// text fields.
func (f *StationForm) HandlePaste(content string) tea.Cmd {
	msg := tea.PasteMsg{Content: content}
	switch {
	case f.Name.Focused():
		f.Name, _ = f.Name.Update(msg)
	case f.Callsign.Focused():
		f.Callsign, _ = f.Callsign.Update(msg)
		f.Callsign.SetValue(strings.ToUpper(f.Callsign.Value()))
	case f.Locator.Focused():
		f.Locator, _ = f.Locator.Update(msg)
		f.Locator.SetValue(formatLocator(f.Locator.Value()))
	case f.SOTARef.Focused():
		f.SOTARef, _ = f.SOTARef.Update(msg)
		f.SOTARef.SetValue(strings.ToUpper(f.SOTARef.Value()))
	case f.POTARef.Focused():
		f.POTARef, _ = f.POTARef.Update(msg)
		f.POTARef.SetValue(strings.ToUpper(f.POTARef.Value()))
	case f.WWFFRef.Focused():
		f.WWFFRef, _ = f.WWFFRef.Update(msg)
		f.WWFFRef.SetValue(strings.ToUpper(f.WWFFRef.Value()))
	case f.CQZone.Focused():
		f.CQZone, _ = f.CQZone.Update(msg)
	case f.ITUZone.Focused():
		f.ITUZone, _ = f.ITUZone.Update(msg)
	case f.DXCC.Focused():
		f.DXCC, _ = f.DXCC.Update(msg)
		f.DXCC.SetValue(strings.ToUpper(f.DXCC.Value()))
	case f.SIG.Focused():
		f.SIG, _ = f.SIG.Update(msg)
		f.SIG.SetValue(strings.ToUpper(f.SIG.Value()))
	case f.SIGInfo.Focused():
		f.SIGInfo, _ = f.SIGInfo.Update(msg)
		f.SIGInfo.SetValue(strings.ToUpper(f.SIGInfo.Value()))
	case f.WlURL.Focused():
		f.WlURL, _ = f.WlURL.Update(msg)
	case f.WlKey.Focused():
		f.WlKey, _ = f.WlKey.Update(msg)
	default:
		return nil // Non-text focus — no paste target.
	}
	return nil
}

func (f *StationForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg
	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}
	if k.String() == " " || k.String() == "space" {
		if f.wlCbFocus {
			f.WlEnabled = !f.WlEnabled
			if f.WlEnabled {
				return func() tea.Msg { return scrollFormToEnd{} }
			}
			return nil
		}
		if f.aprsCbFocus {
			f.AprsEnabled = !f.AprsEnabled
			if f.AprsEnabled {
				return func() tea.Msg { return scrollFormToEnd{} }
			}
			return nil
		}
		if f.aprsSendLocFocus {
			f.AprsSendLoc = !f.AprsSendLoc
			return nil
		}
		if f.gpsGridFocus {
			f.GPSGrid = !f.GPSGrid
			return nil
		}
	}
	// Space on Station ID field cycles through fetched stations.
	if (k.String() == " " || k.String() == "space") && f.WlStationID.Focused() {
		return func() tea.Msg { return wlCycleStation{} }
	}
	if k.String() == "enter" {
		if f.wlBtnFocus == 1 {
			return func() tea.Msg { return wlUpdateAction{} }
		}
		if f.aprsBtnFocus == 1 {
			return func() tea.Msg { return aprsTestAction{} }
		}
	}
	if k.String() == "tab" || msg.Code == tea.KeyDown {
		f.NextInput()
		return nil
	}
	if k.String() == "shift+tab" || msg.Code == tea.KeyUp {
		f.PrevInput()
		return nil
	}
	f.Update(msg)
	return nil
}

type enterOnLastFieldMsg struct{}

// ScrollFraction returns 0.0 (top) to 1.0 (bottom) indicating the relative
// position of the currently focused field within the form. Used by the parent
// to auto-scroll a viewport so the active field stays visible.
func (f *StationForm) ScrollFraction() float64 {
	switch {
	case f.Name.Focused():
		return 0.0
	case f.Callsign.Focused():
		return 0.04
	case f.opFocus:
		return 0.08
	case f.Locator.Focused():
		return 0.12
	case f.gpsGridFocus:
		return 0.145
	case f.iaruFocus:
		return 0.17
	case f.contFocus:
		return 0.22
	case f.SOTARef.Focused():
		return 0.27
	case f.POTARef.Focused():
		return 0.32
	case f.WWFFRef.Focused():
		return 0.37
	case f.CQZone.Focused():
		return 0.42
	case f.ITUZone.Focused():
		return 0.47
	case f.DXCC.Focused():
		return 0.52
	case f.SIG.Focused():
		return 0.57
	case f.SIGInfo.Focused():
		return 0.62
	case f.wlCbFocus:
		return 0.67
	case f.WlURL.Focused():
		return 0.71
	case f.WlKey.Focused():
		return 0.75
	case f.WlStationID.Focused():
		return 0.79
	case f.wlBtnFocus > 0:
		return 0.83
	case f.aprsCbFocus:
		return 0.87
	case f.AprsCallsign.Focused():
		return 0.90
	case f.AprsPasscode.Focused():
		return 0.93
	case f.aprsSendLocFocus:
		return 0.945
	case f.AprsIntervalMin.Focused():
		return 0.955
	case f.AprsRadiusKm.Focused():
		return 0.965
	case f.AprsSymbol.Focused():
		return 0.975
	case f.AprsComment.Focused():
		return 0.985
	case f.aprsBtnFocus > 0:
		return 1.0
	default:
		return 0.5
	}
}

func (f *StationForm) Validate() error {
	nm, cs, _, gr, _, _, _, _, _, _, _, _, _, _, _, _, _, cont := f.Values()
	if nm == "" {
		return fmt.Errorf("station name is required")
	}
	if cs == "" {
		return fmt.Errorf("callsign is required")
	}
	if !qso.IsValidCall(cs) {
		return fmt.Errorf("invalid callsign")
	}
	if gr == "" {
		return fmt.Errorf("grid locator is required")
	}
	if !qso.IsValidLocator(gr) {
		return fmt.Errorf("invalid locator")
	}
	if cont == "" {
		return fmt.Errorf("continent is required")
	}
	return nil
}

// ValidateField returns an error hint for the given render field label, or ""
// if the field value is valid. Used for inline UI feedback.
func (f *StationForm) ValidateField(label string) string {
	_, cs, _, gr, _, _, _, _, _, _, _, _, _, _, _, _, _, cont := f.Values()
	switch label {
	case "Callsign:":
		if cs != "" && !qso.IsValidCall(cs) {
			return "Invalid callsign"
		}
	case "Grid locator:":
		if gr != "" && !qso.IsValidLocator(gr) {
			return "Invalid locator"
		}
	case "Continent:":
		if cont == "" {
			return "Required"
		}
	}
	return ""
}

// continentList returns the list of continent codes for cycling.
func continentList() []string {
	return []string{"EU", "NA", "SA", "AS", "AF", "OC", "AN"}
}

// iaruRegionName returns the human-readable name for an IARU region number.
func iaruRegionName(r int) string {
	switch r {
	case 1:
		return "Europe, Africa, Middle East, N. Asia"
	case 2:
		return "Americas"
	case 3:
		return "Asia-Pacific"
	default:
		return "Unknown"
	}
}
