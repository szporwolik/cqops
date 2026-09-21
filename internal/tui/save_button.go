package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// saveBackButton is a reusable Save & Back button appended to the end of
// config forms and edit/create screens. It follows the same convention as
// the setup wizard's Save & Next button: a visible line with a (Space)
// hint, reached with Tab/Down from the last field and Shift+Tab/Up from the
// first field, activated with Space or Enter.
type saveBackButton struct {
	Focus bool
}

// line renders the button. label is e.g. "Save & Back".
func (b *saveBackButton) line(label string, width int) string {
	w := width - 4
	if w < 20 {
		w = 20
	}
	line := fmt.Sprintf("[ %s ]", label)
	if b.Focus {
		line = S.FormPrefixOn.Render("> ") + CursorStyle.Render(line)
	} else {
		line = InputStyle.Render(line)
	}
	line += " " + DimStyle.Render("(Space)")
	return lipgloss.NewStyle().Width(w).Align(lipgloss.Center).Render(line)
}

// activate reports whether the keypress triggers the button.
func (b *saveBackButton) activate(k tea.KeyPressMsg) bool {
	return k.String() == "enter" || k.String() == " " || k.String() == "space" || k.Code == tea.KeySpace
}

// next reports Tab or Down — used to move focus onto/away from the button.
func (b *saveBackButton) next(k tea.KeyPressMsg) bool {
	return k.String() == "tab" || k.Code == tea.KeyDown
}

// prev reports Shift+Tab or Up.
func (b *saveBackButton) prev(k tea.KeyPressMsg) bool {
	return k.String() == "shift+tab" || k.Code == tea.KeyUp
}
