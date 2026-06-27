package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

// OperatorForm is a simple form for editing an operator callsign and name.
type OperatorForm struct {
	Callsign textinput.Model
	Name     textinput.Model
	focus    int // 0 = callsign, 1 = name
}

// NewOperatorForm creates a form with default values.
func NewOperatorForm() OperatorForm {
	cs := textinput.New()
	cs.Placeholder = "e.g. SP9XXX"
	cs.CharLimit = 16
	cs.SetWidth(16)
	nm := textinput.New()
	nm.Placeholder = "Optional display name"
	nm.CharLimit = 30
	nm.SetWidth(30)
	return OperatorForm{Callsign: cs, Name: nm}
}

// SetOperator fills the form from an existing operator.
func (f *OperatorForm) SetOperator(op *config.Operator) {
	f.Callsign.SetValue(op.Callsign)
	f.Name.SetValue(op.Name)
}

func (f *OperatorForm) nextField() {
	f.focus = (f.focus + 1) % 2
	if f.focus == 0 {
		f.Callsign.Focus()
		f.Name.Blur()
	} else {
		f.Callsign.Blur()
		f.Name.Focus()
	}
}

func (f *OperatorForm) prevField() {
	f.focus = (f.focus + 1) % 2 // identical to nextField for 2 fields
	if f.focus == 0 {
		f.Callsign.Focus()
		f.Name.Blur()
	} else {
		f.Callsign.Blur()
		f.Name.Focus()
	}
}

// HandleKey handles keyboard input and returns a command when Ctrl+S is pressed.
func (f *OperatorForm) HandleKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg

	if k.String() == "ctrl+s" || k.String() == "\x13" {
		return func() tea.Msg { return enterOnLastFieldMsg{} }
	}

	if k.String() == "tab" || msg.Code == tea.KeyDown {
		f.nextField()
		return nil
	}
	if k.String() == "shift+tab" || msg.Code == tea.KeyUp {
		f.prevField()
		return nil
	}

	if f.focus == 0 {
		f.Callsign, _ = f.Callsign.Update(msg)
		f.Callsign.SetValue(strings.ToUpper(f.Callsign.Value()))
	} else {
		f.Name, _ = f.Name.Update(msg)
	}
	return nil
}

// BlurAll removes focus from all fields.
func (f *OperatorForm) BlurAll() {
	f.Callsign.Blur()
	f.Name.Blur()
	f.focus = -1
}

// SetWidth adjusts the width of the form fields.
func (f *OperatorForm) SetWidth(w int) {
	if w < 20 {
		w = 20
	}
	f.Callsign.SetWidth(min(w-14, 16))
	f.Name.SetWidth(min(w-14, 30))
}

// Validate checks the form and returns an error message or empty string.
func (f *OperatorForm) Validate() string {
	call := strings.TrimSpace(f.Callsign.Value())
	if call == "" {
		return "Callsign is required"
	}
	return ""
}

// ValidateCall checks whether the callsign looks like a standard ham callsign
// (has both letters and digits). Returns a warning string or empty if ok.
func (f *OperatorForm) ValidateCall() string {
	call := strings.TrimSpace(f.Callsign.Value())
	if call != "" && !qso.IsValidCall(call) {
		return "Callsign \"" + call + "\" doesn't look like a standard callsign (no digit) — saved anyway"
	}
	return ""
}

// Values returns the callsign and name from the form.
func (f *OperatorForm) Values() (callsign, name string) {
	return strings.TrimSpace(f.Callsign.Value()), strings.TrimSpace(f.Name.Value())
}

// View renders the form.
func (f *OperatorForm) View() string {
	var lines []string
	csLbl := S.FormLabel.Render("Callsign")
	nmLbl := S.FormLabel.Render("Name")
	if f.focus == 0 {
		csLbl = fieldFocusedLabel.Render("Callsign")
	} else {
		nmLbl = fieldFocusedLabel.Render("Name")
	}
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, csLbl, " ", f.Callsign.View()))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, nmLbl, " ", f.Name.View()))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Focus sets focus to the callsign field.
func (f *OperatorForm) Focus() {
	f.focus = 0
	f.Callsign.Focus()
	f.Name.Blur()
}
