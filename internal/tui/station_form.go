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
		if f.WlEnabled {
			f.WlURL.Focus()
		} else {
			f.Callsign.Focus()
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
		if f.WlEnabled {
			f.wlBtnFocus = 2
		} else {
			f.wlCbFocus = true
		}
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
	type fieldDef struct {
		label string
		ti    *textinput.Model
	}
	var fields []fieldDef
	fields = append(fields,
		fieldDef{"Callsign:", &f.Callsign},
		fieldDef{"Grid locator:", &f.Locator},
		fieldDef{"Operator (opt):", &f.Operator},
		fieldDef{"SOTA Ref (opt):", &f.SOTARef},
		fieldDef{"POTA Ref (opt):", &f.POTARef},
		fieldDef{"WWFF Ref (opt):", &f.WWFFRef},
	)

	var b strings.Builder
	for _, field := range fields {
		b.WriteString(padOrTrunc(f.renderFieldLine(field.label, field.ti), 80))
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
		wlCheckbox = CursorStyle.Render(wlCheckbox)
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, wlCbPrefix, wlCbLabel, " ", wlCheckbox),
		80))
	b.WriteString("\n")

	if f.WlEnabled {
		wlFields := []fieldDef{
			{"  API URL:", &f.WlURL},
			{"  API Key:", &f.WlKey},
			{"  Station ID:", &f.WlStationID},
		}
		for _, field := range wlFields {
			b.WriteString(padOrTrunc(f.renderFieldLine(field.label, field.ti), 80))
		}

		updateBtn := "[ Update ]"
		updateHint := "fetch stations from Wavelog"
		if f.wlBtnFocus == 1 {
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center,
					S.FormPrefixOn.Render("> "),
					CursorStyle.Render(updateBtn),
					" ",
					DimStyle.Render(updateHint)),
				80))
		} else {
			b.WriteString(padOrTrunc("    "+InputStyle.Render(updateBtn)+" "+DimStyle.Render(updateHint), 80))
		}
		b.WriteString("\n")

		testBtn := "[ Test ]"
		testHint := "verify connection and station"
		if f.wlBtnFocus == 2 {
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center,
					S.FormPrefixOn.Render("> "),
					CursorStyle.Render(testBtn),
					" ",
					DimStyle.Render(testHint)),
				80))
		} else {
			b.WriteString(padOrTrunc("    "+InputStyle.Render(testBtn)+" "+DimStyle.Render(testHint), 80))
		}
	}

	return tea.NewView(b.String())
}

func (f *StationForm) renderFieldLine(label string, ti *textinput.Model) string {
	focused := ti.Focused()
	raw := strings.TrimSpace(ti.Value())

	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	var val string
	if focused {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
	} else {
		val = ValueStyle.Render(raw)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val) + "\n"
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
