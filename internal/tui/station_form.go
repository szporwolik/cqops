package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type StationForm struct {
	Callsign textinput.Model
	Operator textinput.Model
	Locator  textinput.Model
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

	return &StationForm{
		Callsign: cs,
		Operator: op,
		Locator:  lc,
	}
}

func (f *StationForm) Update(msg tea.KeyMsg) {
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
		f.Callsign.Focus()
	}
}

func (f *StationForm) PrevInput() {
	switch {
	case f.Callsign.Focused():
		f.Callsign.Blur()
		f.Locator.Focus()
	case f.Operator.Focused():
		f.Operator.Blur()
		f.Callsign.Focus()
	case f.Locator.Focused():
		f.Locator.Blur()
		f.Operator.Focus()
	}
}

func (f *StationForm) OnLastField() bool {
	return f.Locator.Focused()
}

func (f *StationForm) Values() (callsign, operator, locator string) {
	return strings.ToUpper(strings.TrimSpace(f.Callsign.Value())),
		strings.ToUpper(strings.TrimSpace(f.Operator.Value())),
		strings.ToUpper(strings.TrimSpace(f.Locator.Value()))
}

func (f *StationForm) SetValues(callsign, operator, locator string) {
	f.Callsign.SetValue(callsign)
	f.Operator.SetValue(operator)
	f.Locator.SetValue(locator)
}

func (f *StationForm) View() string {
	var b strings.Builder
	b.WriteString(formLabelStyle.Render("Callsign:"))
	b.WriteString(inputStyle.Render(f.Callsign.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Operator (optional):"))
	b.WriteString(inputStyle.Render(f.Operator.View()))
	b.WriteString("\n\n")

	b.WriteString(formLabelStyle.Render("Grid (locator):"))
	b.WriteString(inputStyle.Render(f.Locator.View()))
	return b.String()
}

func (f *StationForm) HandleKey(msg tea.KeyMsg) tea.Cmd {
	k := msg
	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}
	if k.String() == "tab" || msg.Type == tea.KeyDown || k.String() == "enter" {
		f.NextInput()
		return nil
	}
	if k.String() == "shift+tab" || msg.Type == tea.KeyUp {
		f.PrevInput()
		return nil
	}
	f.Update(msg)
	return nil
}

type enterOnLastFieldMsg struct{}

func (f *StationForm) Validate() error {
	cs, _, gr := f.Values()
	if cs == "" {
		return fmt.Errorf("callsign is required")
	}
	if gr == "" {
		return fmt.Errorf("grid locator is required")
	}
	return nil
}
