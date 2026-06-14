package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	cs.SetWidth(28)
	cs.Placeholder = callsignPlaceholder
	cs.Focus()
	cs.Prompt = ""

	op := textinput.New()
	op.CharLimit = 20
	op.SetWidth(28)
	op.Placeholder = opPlaceholder
	op.Prompt = ""

	lc := textinput.New()
	lc.CharLimit = 8
	lc.SetWidth(28)
	lc.Placeholder = locatorPlaceholder
	lc.Prompt = ""

	sr := textinput.New()
	sr.CharLimit = 20
	sr.SetWidth(28)
	sr.Placeholder = "e.g. SP/TA-001"
	sr.Prompt = ""

	pr := textinput.New()
	pr.CharLimit = 20
	pr.SetWidth(28)
	pr.Placeholder = "e.g. SP-0001"
	pr.Prompt = ""

	wr := textinput.New()
	wr.CharLimit = 20
	wr.SetWidth(28)
	wr.Placeholder = "e.g. SPFF-0001"
	wr.Prompt = ""

	// Apply Surface background to textinput styles BEFORE storing in struct
	// (textinput.Model is a value type — copies made after SetStyles are stale).
	for _, ti := range []*textinput.Model{&cs, &op, &lc, &sr, &pr, &wr} {
		s := ti.Styles()
		s.Focused.Text = s.Focused.Text.Background(P.Surface)
		s.Focused.Placeholder = s.Focused.Placeholder.Background(P.Surface)
		s.Focused.Prompt = s.Focused.Prompt.Background(P.Surface)
		s.Blurred.Text = s.Blurred.Text.Background(P.Surface)
		s.Blurred.Placeholder = s.Blurred.Placeholder.Background(P.Surface)
		s.Blurred.Prompt = s.Blurred.Prompt.Background(P.Surface)
		ti.SetStyles(s)
	}

	sf := &StationForm{
		Callsign: cs,
		Operator: op,
		Locator:  lc,
		SOTARef:  sr,
		POTARef:  pr,
		WWFFRef:  wr,
	}
	return sf
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
		f.Locator.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Callsign.Focus()
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
		formatLocator(f.Locator.Value()),
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
	bg := lipgloss.NewStyle().Background(P.Surface)
	fields := []struct {
		label string
		ti    *textinput.Model
	}{
		{"Callsign:", &f.Callsign},
		{"Grid locator:", &f.Locator},
		{"Operator (optional):", &f.Operator},
		{"SOTA Ref (optional):", &f.SOTARef},
		{"POTA Ref (optional):", &f.POTARef},
		{"WWFF Ref (optional):", &f.WWFFRef},
	}

	var b strings.Builder
	for _, field := range fields {
		focused := field.ti.Focused()
		raw := strings.TrimSpace(field.ti.Value())
		label := LabelStyle.Render(fit(field.label, 22))
		if focused {
			label = CursorStyle.Render(fit(field.label, 22))
		}
		// Render value: textinput view when focused, ValueStyle when not
		// (matching the edit QSO form pattern).
		// Wrap focused view with InputStyle to catch any ANSI resets from
		// placeholder rendering.
		var val string
		if focused {
			val = field.ti.View()
		} else if raw == "" {
			val = SubtleStyle.Render("\u2014")
		} else {
			val = ValueStyle.Render(raw)
		}
		prefix := "  "
		if focused {
			prefix = CursorStyle.Render("> ")
		}
		line := prefix + label + bg.Render(" ") + val
		b.WriteString(menuLine(line, 80))
		b.WriteString("\n")
	}
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
