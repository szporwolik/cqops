package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type StationForm struct {
	Callsign textinput.Model
	Operator textinput.Model
	Locator  textinput.Model
	SOTARef  textinput.Model
	POTARef  textinput.Model
	WWFFRef  textinput.Model
}

func NewStationForm(callsignPlaceholder, opPlaceholder, locatorPlaceholder string) *StationForm {
	cs := textinput.New()
	cs.CharLimit = 20
	cs.Placeholder = callsignPlaceholder
	cs.Focus()
	cs.Prompt = ""

	op := textinput.New()
	op.CharLimit = 20
	op.Placeholder = opPlaceholder
	op.Prompt = ""

	lc := textinput.New()
	lc.CharLimit = 8
	lc.Placeholder = locatorPlaceholder
	lc.Prompt = ""

	sr := textinput.New()
	sr.CharLimit = 20
	sr.Placeholder = "e.g. SP/TA-001"
	sr.Prompt = ""

	pr := textinput.New()
	pr.CharLimit = 20
	pr.Placeholder = "e.g. SP-0001"
	pr.Prompt = ""

	wr := textinput.New()
	wr.CharLimit = 20
	wr.Placeholder = "e.g. SPFF-0001"
	wr.Prompt = ""

	return &StationForm{
		Callsign: cs,
		Operator: op,
		Locator:  lc,
		SOTARef:  sr,
		POTARef:  pr,
		WWFFRef:  wr,
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
		f.Locator.SetValue(strings.ToUpper(f.Locator.Value()))
	case f.SOTARef.Focused():
		f.SOTARef, _ = f.SOTARef.Update(msg)
		f.SOTARef.SetValue(strings.ToUpper(f.SOTARef.Value()))
	case f.POTARef.Focused():
		f.POTARef, _ = f.POTARef.Update(msg)
		f.POTARef.SetValue(strings.ToUpper(f.POTARef.Value()))
	case f.WWFFRef.Focused():
		f.WWFFRef, _ = f.WWFFRef.Update(msg)
		f.WWFFRef.SetValue(strings.ToUpper(f.WWFFRef.Value()))
	}
}

func (f *StationForm) NextInput() {
	switch {
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.Operator.Focus()
	case f.Operator.Focused():
		f.Operator.Blur()
		f.Locator.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.SOTARef.Focus()
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.POTARef.Focus()
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.WWFFRef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.Callsign.Focus()
	}
}

func (f *StationForm) PrevInput() {
	switch {
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.WWFFRef.Focus()
	case f.Operator.Focused():
		f.Operator.Blur()
		f.Callsign.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Operator.Focus()
	case f.SOTARef.Focused():
		f.SOTARef.Blur()
		f.Locator.Focus()
	case f.POTARef.Focused():
		f.POTARef.Blur()
		f.SOTARef.Focus()
	case f.WWFFRef.Focused():
		f.WWFFRef.Blur()
		f.POTARef.Focus()
	}
}

func (f *StationForm) OnLastField() bool {
	return f.WWFFRef.Focused()
}

func (f *StationForm) Values() (callsign, operator, locator, sotaRef, potaRef, wwffRef string) {
	return strings.ToUpper(strings.TrimSpace(f.Callsign.Value())),
		strings.ToUpper(strings.TrimSpace(f.Operator.Value())),
		strings.ToUpper(strings.TrimSpace(f.Locator.Value())),
		strings.TrimSpace(f.SOTARef.Value()),
		strings.TrimSpace(f.POTARef.Value()),
		strings.TrimSpace(f.WWFFRef.Value())
}

func (f *StationForm) SetValues(callsign, operator, locator, sotaRef, potaRef, wwffRef string) {
	f.Callsign.SetValue(callsign)
	f.Operator.SetValue(operator)
	f.Locator.SetValue(locator)
	f.SOTARef.SetValue(sotaRef)
	f.POTARef.SetValue(potaRef)
	f.WWFFRef.SetValue(wwffRef)
}

func (f *StationForm) View() tea.View {
	var b strings.Builder
	b.WriteString(formLabelStyle.Render("Callsign:"))
	b.WriteString(inputStyle.Render(f.Callsign.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Operator (optional):"))
	b.WriteString(inputStyle.Render(f.Operator.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Grid (locator):"))
	b.WriteString(inputStyle.Render(f.Locator.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("SOTA Ref (optional):"))
	b.WriteString(inputStyle.Render(f.SOTARef.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("POTA Ref (optional):"))
	b.WriteString(inputStyle.Render(f.POTARef.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("WWFF Ref (optional):"))
	b.WriteString(inputStyle.Render(f.WWFFRef.View()))
	return tea.NewView(b.String())
}

func (f *StationForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg
	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
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
	cs, _, gr, _, _, _ := f.Values()
	if cs == "" {
		return fmt.Errorf("callsign is required")
	}
	if gr == "" {
		return fmt.Errorf("grid locator is required")
	}
	return nil
}
