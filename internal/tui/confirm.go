package tui

import (
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
	return c.viewSheet(width)
}

// viewSheet renders a full-width sheet-style dialog using the reusable Sheet helper.
func (c Confirm) viewSheet(width int) string {
	msg := S.ConfirmMsg.Render(c.Message)

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

	body := lipgloss.JoinVertical(lipgloss.Center, msg, "", btns)
	return Sheet(width, c.Title, body)
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
	btns := lipgloss.JoinHorizontal(lipgloss.Top, yesBtn, noBtn)
	line := lipgloss.JoinHorizontal(lipgloss.Top, title, msg, btns)
	if lipgloss.Width(line) > width {
		line = lipgloss.JoinHorizontal(lipgloss.Top, title, msg)
	}
	return line
}

// =============================================================================
// Sheet — reusable full-width overlay panel (Bubble Tea / Lip Gloss native)
// =============================================================================

// Sheet renders a full-width overlay panel with a bold title, body content,
// and page-matching background. Use this for any modal/dialog/prompt to keep
// the look consistent without hand-crafting borders or separators.
//
//	body := lipgloss.JoinVertical(lipgloss.Center, "Are you sure?", "", btns)
//	output := Sheet(width, "Confirm", body)
func Sheet(width int, title, body string) string {
	titleLine := S.ConfirmTitle.Render(title)

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleLine,
		"",
		body,
	)

	return lipgloss.NewStyle().
		Width(width).
		Background(P.Background).
		Padding(1, 2).
		Render(content)
}

// RenderConfirmOverlay composites a confirm dialog as a full-width sheet
// over the main view. The dialog has no border and matches the page background,
// appearing as a natural overlay with key hints in the footer help bar.
func RenderConfirmOverlay(mainView string, c Confirm, viewW, viewH int) string {
	dialog := c.View(viewW)

	dialogH := lipgloss.Height(dialog)

	// Center vertically over the full view
	y := (viewH - dialogH) / 2
	if y < 0 {
		y = 0
	}

	base := lipgloss.NewLayer(mainView)
	dialogLayer := lipgloss.NewLayer(dialog).
		X(0).
		Y(y).
		Z(1)

	return lipgloss.NewCompositor(base, dialogLayer).Render()
}
