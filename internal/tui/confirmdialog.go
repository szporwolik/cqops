package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ── Option ──

// Option represents a single action button in the dialog.
type Option struct {
	Label  string // display text, e.g. "Quit", "Cancel"
	Value  string // returned in Result when selected
	Danger bool   // renders with danger/error styling when focused
}

// ── Result ──

// Result is returned when the dialog is dismissed.
type Result struct {
	Confirmed bool   // true if an option was confirmed (not cancelled)
	Value     string // the Value of the selected Option
	Cancelled bool   // true if dismissed via Esc
}

// ── DialogModel ──

// DialogModel is a reusable modal confirmation dialog.
// It implements tea.DialogModel and can be composed into any parent DialogModel.
type DialogModel struct {
	Title   string
	Message string
	Options []Option

	selected int  // index into Options
	width    int  // terminal width from WindowSizeMsg
	height   int  // terminal height from WindowSizeMsg
	done     bool // true when confirmed or cancelled

	// Result is populated when the dialog is dismissed.
	Result Result
}

// New creates a confirmation dialog. At least one Option is required.
func NewDialog(title, message string, options ...Option) DialogModel {
	return DialogModel{
		Title:    title,
		Message:  message,
		Options:  options,
		selected: 0,
	}
}

// Init implements tea.Model.
func (m DialogModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m DialogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "right", "tab":
			m.selected = (m.selected + 1) % len(m.Options)
		case "shift+tab":
			m.selected--
			if m.selected < 0 {
				m.selected = len(m.Options) - 1
			}
		case "enter":
			m.done = true
			m.Result = Result{
				Confirmed: true,
				Value:     m.Options[m.selected].Value,
			}
		case "esc":
			m.done = true
			m.Result = Result{Cancelled: true}
		case "q":
			// Only quit on 'q' if no danger option is focused
			if !m.Options[m.selected].Danger {
				m.done = true
				m.Result = Result{Cancelled: true}
			}
		}
	}
	return m, nil
}

// Done returns true when the dialog has been dismissed.
func (m DialogModel) Done() bool { return m.done }

// ── View ──

// View renders the dialog as a centered bordered modal.
// The backdrop/dimming is handled by the parent via RenderDialogOverlay.
func (m DialogModel) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the dialog content string — a centered, bordered modal.
func (m DialogModel) render() string {
	w := m.width
	if w < 40 {
		w = 80
	}

	// Modal width: capped at 56, minimum 30
	modalW := w - 12
	if modalW > 56 {
		modalW = 56
	}
	if modalW < 30 {
		modalW = 30
	}

	// Title — centered inside the modal
	title := S.ConfirmTitle.
		Width(modalW - 4). // inside border + padding
		Align(lipgloss.Center).
		Render(m.Title)

	// Message
	msg := S.ConfirmMsg.
		Width(modalW - 4).
		Align(lipgloss.Center).
		Render(m.Message)

	// Buttons — size only to label + padding, centered as a group
	var btnParts []string
	for i, opt := range m.Options {
		s := S.ConfirmBtnDim
		if i == m.selected {
			if opt.Danger {
				s = S.ConfirmDanger
			} else {
				s = S.ConfirmBtn
			}
		}
		btnParts = append(btnParts, s.Render(" "+opt.Label+" "))
	}
	btns := lipgloss.JoinHorizontal(lipgloss.Center, btnParts...)
	btns = lipgloss.NewStyle().Width(modalW - 4).Align(lipgloss.Center).Render(btns)

	// Hint
	hint := S.ConfirmHint.
		Width(modalW - 4).
		Align(lipgloss.Center).
		Render("←/→ select  •  enter confirm  •  esc cancel")

	// Assemble modal body vertically
	modal := lipgloss.JoinVertical(lipgloss.Top,
		title,
		"",
		msg,
		"",
		btns,
		"",
		hint,
	)

	// Wrap in bordered box
	return confirmBoxStyle.Width(modalW).Render(modal)
}

// ── Helpers ──

// DangerOption creates an Option that renders with danger styling when focused.
func DangerOption(label, value string) Option {
	return Option{Label: label, Value: value, Danger: true}
}

// RenderDialogOverlay composites the dialog as a centered overlay on top of
// the main view using lipgloss compositing. The dialog is placed exactly
// centered both horizontally and vertically over the terminal space.
func RenderDialogOverlay(mainView string, dlg DialogModel, viewW, viewH int) string {
	dlg.width = viewW
	dlg.height = viewH
	dialog := dlg.render()

	// Use lipgloss.Place to center the dialog in the terminal space
	placed := lipgloss.Place(viewW, viewH, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
	)

	base := lipgloss.NewLayer(mainView)
	dialogLayer := lipgloss.NewLayer(placed).X(0).Y(0).Z(1)

	return lipgloss.NewCompositor(base, dialogLayer).Render()
}
