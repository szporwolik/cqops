package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

type StationForm struct {
	Callsign    textinput.Model
	Operator    textinput.Model
	Locator     textinput.Model
	SOTARef     textinput.Model
	POTARef     textinput.Model
	WWFFRef     textinput.Model
	WlEnabled   bool
	wlCbFocus   bool // true when the WL checkbox has focus
	wlBtnFocus  int  // 0=none, 1=Update, 2=Test
	WlURL       textinput.Model
	WlKey       textinput.Model
	WlStationID textinput.Model
}

// Wavelog button action messages sent when a button is activated via Enter.
type wlUpdateAction struct{}
type wlTestAction struct{}
type wlCycleStation struct{}

func NewStationForm(callsignPlaceholder, opPlaceholder, locatorPlaceholder string) *StationForm {
	mkTI := func(limit int, width int, placeholder string) textinput.Model {
		ti := newTextinput()
		ti.CharLimit = limit
		ti.SetWidth(width)
		ti.Placeholder = placeholder
		return ti
	}

	cs := mkTI(20, 28, callsignPlaceholder)
	cs.Focus()
	op := mkTI(20, 28, opPlaceholder)
	lc := mkTI(8, 28, locatorPlaceholder)
	sr := mkTI(20, 28, "e.g. SP/TA-001")
	pr := mkTI(20, 28, "e.g. SP-0001")
	wr := mkTI(20, 28, "e.g. SPFF-0001")

	wu := mkTI(80, 28, "https://log.example.com")
	wk := mkTI(64, 28, "Wavelog API key")
	ws := mkTI(80, 60, "press Update to fetch")

	for _, ti := range []*textinput.Model{&cs, &op, &lc, &sr, &pr, &wr, &wu, &wk, &ws} {
		applyTextinputSurfaceStyle(ti)
	}

	return &StationForm{
		Callsign:    cs,
		Operator:    op,
		Locator:     lc,
		SOTARef:     sr,
		POTARef:     pr,
		WWFFRef:     wr,
		WlURL:       wu,
		WlKey:       wk,
		WlStationID: ws,
	}
}

func (f *StationForm) Update(msg tea.KeyPressMsg) {
	switch {
	case f.Callsign.Focused():
		f.Callsign, _ = f.Callsign.Update(msg)
		f.Callsign.SetValue(strings.ToUpper(f.Callsign.Value()))
	case f.Operator.Focused():
		f.Operator, _ = f.Operator.Update(msg)
		f.Operator.SetValue(strings.ToUpper(f.Operator.Value()))
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
	case f.WlURL.Focused():
		f.WlURL, _ = f.WlURL.Update(msg)
	case f.WlKey.Focused():
		f.WlKey, _ = f.WlKey.Update(msg)
	case f.WlStationID.Focused():
		// Station ID is read-only — updated via Update button or Space cycle.
	}
}

func (f *StationForm) NextInput() {
	switch {
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.Locator.Focus()
	case f.Operator.Focused():
		f.Operator.Blur()
		f.SOTARef.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Operator.Focus()
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.POTARef.Focus()
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.WWFFRef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.wlCbFocus = true
	case f.wlCbFocus:
		f.wlCbFocus = false
		f.WlURL.Focus()
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
		f.wlBtnFocus = 2
	case f.wlBtnFocus == 2:
		f.wlBtnFocus = 0
		f.Callsign.Focus()
	}
}

func (f *StationForm) PrevInput() {
	switch {
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.wlBtnFocus = 2
	case f.wlBtnFocus == 2:
		f.wlBtnFocus = 1
	case f.wlBtnFocus == 1:
		f.wlBtnFocus = 0
		f.WlStationID.Focus()
	case f.Operator.Focused():
		f.Operator.Blur()
		f.Locator.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Callsign.Focus()
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.Operator.Focus()
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.SOTARef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.POTARef.Focus()
	case f.wlCbFocus:
		f.wlCbFocus = false
		f.WWFFRef.Focus()
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
	return f.wlBtnFocus == 2
}

func (f *StationForm) BlurAll() {
	blurTextinputs(&f.Callsign, &f.Operator, &f.Locator, &f.SOTARef, &f.POTARef, &f.WWFFRef,
		&f.WlURL, &f.WlKey, &f.WlStationID)
	f.wlCbFocus = false
	f.wlBtnFocus = 0
}

func (f *StationForm) Values() (callsign, operator, locator, sotaRef, potaRef, wwffRef string,
	wlEnabled bool, wlURL, wlKey, wlStationID string) {
	return strings.ToUpper(strings.TrimSpace(f.Callsign.Value())),
		strings.ToUpper(strings.TrimSpace(f.Operator.Value())),
		formatLocator(f.Locator.Value()),
		strings.TrimSpace(f.SOTARef.Value()),
		strings.TrimSpace(f.POTARef.Value()),
		strings.TrimSpace(f.WWFFRef.Value()),
		f.WlEnabled,
		strings.TrimSpace(f.WlURL.Value()),
		strings.TrimSpace(f.WlKey.Value()),
		strings.TrimSpace(f.WlStationID.Value())
}

func (f *StationForm) SetValues(callsign, operator, locator, sotaRef, potaRef, wwffRef string) {
	f.Callsign.SetValue(callsign)
	f.Operator.SetValue(operator)
	f.Locator.SetValue(locator)
	f.SOTARef.SetValue(sotaRef)
	f.POTARef.SetValue(potaRef)
	f.WWFFRef.SetValue(wwffRef)
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

func (f *StationForm) View() tea.View {
	bg := lipgloss.NewStyle().Background(P.Surface)

	type fieldDef struct {
		label string
		ti    *textinput.Model
	}
	var fields []fieldDef
	fields = append(fields,
		fieldDef{"Callsign:", &f.Callsign},
		fieldDef{"Grid locator:", &f.Locator},
		fieldDef{"Operator (optional):", &f.Operator},
		fieldDef{"SOTA Ref (optional):", &f.SOTARef},
		fieldDef{"POTA Ref (optional):", &f.POTARef},
		fieldDef{"WWFF Ref (optional):", &f.WWFFRef},
	)

	var b strings.Builder
	for _, field := range fields {
		b.WriteString(f.renderFieldLine(field.label, field.ti))
	}

	// Wavelog checkbox — focusable via Tab, Space to toggle.
	wlCheckbox := "[ ]"
	if f.WlEnabled {
		wlCheckbox = "[x]"
	}
	wlCbPrefix := "  "
	wlCbLabel := LabelStyle.Render(fit("Wavelog:", 22))
	if f.wlCbFocus {
		wlCbPrefix = CursorStyle.Render("> ")
		wlCbLabel = CursorStyle.Render(fit("Wavelog:", 22))
		wlCheckbox = CursorStyle.Render(wlCheckbox)
	} else {
		wlCheckbox = bg.Render(wlCheckbox)
	}
	b.WriteString(menuLine(wlCbPrefix+wlCbLabel+bg.Render(" ")+wlCheckbox, 80))
	b.WriteString("\n")

	if f.WlEnabled {
		wlFields := []fieldDef{
			{"  API URL:", &f.WlURL},
			{"  API Key:", &f.WlKey},
			{"  Station ID:", &f.WlStationID},
		}
		for _, field := range wlFields {
			b.WriteString(f.renderFieldLine(field.label, field.ti))
		}

		updateBtn := "[ Update ]"
		updateHint := "fetch stations from Wavelog"
		if f.wlBtnFocus == 1 {
			b.WriteString(menuLine("  "+CursorStyle.Render("> ")+CursorStyle.Render(updateBtn)+bg.Render(" ")+SubtleStyle.Render(updateHint), 80))
		} else {
			b.WriteString(menuLine("    "+InputStyle.Render(updateBtn)+bg.Render(" ")+SubtleStyle.Render(updateHint), 80))
		}
		b.WriteString("\n")

		testBtn := "[ Test ]"
		testHint := "verify connection and station"
		if f.wlBtnFocus == 2 {
			b.WriteString(menuLine("  "+CursorStyle.Render("> ")+CursorStyle.Render(testBtn)+bg.Render(" ")+SubtleStyle.Render(testHint), 80))
		} else {
			b.WriteString(menuLine("    "+InputStyle.Render(testBtn)+bg.Render(" ")+SubtleStyle.Render(testHint), 80))
		}
	}

	return tea.NewView(b.String())
}

func (f *StationForm) renderFieldLine(label string, ti *textinput.Model) string {
	bg := lipgloss.NewStyle().Background(P.Surface)
	focused := ti.Focused()
	raw := strings.TrimSpace(ti.Value())

	lbl := LabelStyle.Render(fit(label, 22))
	if focused {
		lbl = CursorStyle.Render(fit(label, 22))
	}

	var val string
	if focused {
		val = ti.View()
	} else if raw == "" {
		val = SubtleStyle.Render("\u2014")
	} else {
		val = ValueStyle.Render(raw)
	}
	prefix := "  "
	if focused {
		prefix = CursorStyle.Render("> ")
	}
	return menuLine(prefix+lbl+bg.Render(" ")+val, 80) + "\n"
}

func (f *StationForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg
	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}
	if k.String() == " " || k.String() == "space" {
		if f.wlCbFocus {
			f.WlEnabled = !f.WlEnabled
			return nil
		}
	}
	// Space on Station ID field cycles through fetched stations.
	if (k.String() == " " || k.String() == "space") && f.WlStationID.Focused() {
		return func() tea.Msg { return wlCycleStation{} }
	}
	if k.String() == "enter" {
		switch f.wlBtnFocus {
		case 1:
			return func() tea.Msg { return wlUpdateAction{} }
		case 2:
			return func() tea.Msg { return wlTestAction{} }
		}
	}
	if k.String() == "tab" || msg.Code == tea.KeyDown || k.String() == "enter" {
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

func (f *StationForm) Validate() error {
	cs, _, gr, _, _, _, _, _, _, _ := f.Values()
	if cs == "" {
		return fmt.Errorf("callsign is required")
	}
	if gr == "" {
		return fmt.Errorf("grid locator is required")
	}
	return nil
}
