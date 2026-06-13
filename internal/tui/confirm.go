package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// ConfirmKind classifies the prompt's nature for styling.
type ConfirmKind int

const (
	ConfirmInfo    ConfirmKind = iota // neutral (quit, upload)
	ConfirmWarning                    // caution (overwrite)
	ConfirmDanger                     // destructive (delete, purge)
)

// Confirm holds the state of a yes/no prompt.
type Confirm struct {
	Title     string
	Message   string
	YesLabel  string
	NoLabel   string
	ChooseYes bool // true = Yes focused, false = No focused
	Kind      ConfirmKind
}

// NewConfirm builds a prompt. For dangerous actions the default is No.
func NewConfirm(title, msg, yes, no string, kind ConfirmKind) Confirm {
	return Confirm{
		Title:     title,
		Message:   msg,
		YesLabel:  yes,
		NoLabel:   no,
		ChooseYes: kind != ConfirmDanger,
		Kind:      kind,
	}
}

// FocusYes moves focus to the Yes button.
func (c *Confirm) FocusYes() { c.ChooseYes = true }

// FocusNo moves focus to the No button.
func (c *Confirm) FocusNo() { c.ChooseYes = false }

// Toggle toggles the selection.
func (c *Confirm) Toggle() { c.ChooseYes = !c.ChooseYes }

// Selected returns true when Yes is chosen.
func (c Confirm) Selected() bool { return c.ChooseYes }

// View renders the confirm dialog.
func (c Confirm) View(width int) string {
	if width < 40 {
		return c.viewCompact(width)
	}
	return c.viewBoxed(width)
}

// viewBoxed renders a bordered dialog centered in the available width.
func (c Confirm) viewBoxed(width int) string {
	const maxW = 56
	const minW = 34
	w := width - 8
	if w > maxW {
		w = maxW
	}
	if w < minW {
		w = minW
	}

	// Title with separator line
	title := S.ConfirmTitle.Render(" " + c.Title + " ")
	titleW := lipgloss.Width(title)
	dashW := w - titleW - 4 // 4 for "── " prefix and " " suffix
	if dashW < 1 {
		dashW = 1
	}
	titleLine := "── " + title + " " + strings.Repeat("─", dashW)

	// Message
	msg := S.ConfirmMsg.Render(c.Message)

	// Buttons
	yesStyle := S.ConfirmBtnDim
	noStyle := S.ConfirmBtnDim
	if c.ChooseYes {
		if c.Kind == ConfirmDanger {
			yesStyle = S.ConfirmDanger
		} else {
			yesStyle = S.ConfirmBtn
		}
	} else {
		noStyle = S.ConfirmBtn
	}
	yesBtn := yesStyle.Render(" " + c.YesLabel + " ")
	noBtn := noStyle.Render(" " + c.NoLabel + " ")
	btns := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, noBtn)

	// Hint
	hint := S.ConfirmHelp.Render("←/→ choose  •  enter confirm  •  esc cancel")

	// Assemble inner content
	inner := lipgloss.JoinVertical(lipgloss.Center,
		"", msg, "", btns, "", hint,
	)

	// Ensure each line is exactly w chars wide
	bodyLines := strings.Split(titleLine+"\n"+inner, "\n")
	for i, line := range bodyLines {
		lw := lipgloss.Width(line)
		if lw < w {
			bodyLines[i] = line + strings.Repeat(" ", w-lw)
		}
	}
	plain := strings.Join(bodyLines, "\n")

	// Wrap in bordered box
	box := S.ConfirmBox.Width(w).Render(plain)

	// Center horizontally with lipgloss
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(P.Background)),
	)
}

// viewCompact renders an inline prompt for narrow terminals.
func (c Confirm) viewCompact(width int) string {
	title := S.ConfirmTitle.Render(c.Title)
	msg := S.ConfirmMsg.Render(" " + c.Message + " ")
	yesStyle := S.ConfirmBtnDim
	noStyle := S.ConfirmBtnDim
	if c.ChooseYes {
		if c.Kind == ConfirmDanger {
			yesStyle = S.ConfirmDanger
		} else {
			yesStyle = S.ConfirmBtn
		}
	} else {
		noStyle = S.ConfirmBtn
	}
	yesBtn := yesStyle.Render(" " + c.YesLabel + " ")
	noBtn := noStyle.Render(" " + c.NoLabel + " ")
	line := title + msg + yesBtn + "  " + noBtn
	if lipgloss.Width(line) > width {
		line = title + msg
	}
	return line
}
