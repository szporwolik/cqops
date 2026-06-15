package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// Pre-allocated confirm dialog key bindings — reused across all confirm screens.
var confirmBindings = []key.Binding{
	key.NewBinding(key.WithKeys("←/→"), key.WithHelp("←/→", "choose")),
	key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// helpView renders the bottom help/footer bar with context-sensitive key bindings.
func (m *Model) helpView() string {
	// Global confirm dialog (quit, etc.)
	if m.confirm != nil {
		return HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
	}

	// Internal confirm dialogs: rig/chooser delete confirmations.
	if m.rigChooser != nil && m.rigChooser.mode == rigChooserConfirmDelete && m.rigChooser.dialog != nil {
		return HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
	}
	if m.chooser != nil && m.chooser.mode == chooserConfirmDelete && m.chooser.dialog != nil {
		return HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
	}
	if m.logbookEditor != nil && m.logbookEditor.isConfirmMode() && m.logbookEditor.dialog != nil {
		return HelpStyle.Render(m.help.ShortHelpView(confirmBindings))
	}

	bindings := m.ActiveBindings()
	if len(bindings) == 0 {
		bindings = []key.Binding{m.keys.Quit}
	}
	helpText := m.help.ShortHelpView(bindings)
	if helpText == "" {
		helpText = m.help.ShortHelpView([]key.Binding{m.keys.Quit})
	}

	// Dynamic suffix (counter / scroll info) — always computed, never cached.
	suffix := m.helpSuffix()
	if suffix != "" {
		suffixW := lipgloss.Width(suffix)
		spacerW := m.width - lipgloss.Width(helpText) - suffixW - 2
		if spacerW < 1 {
			spacerW = 1
		}
		return HelpStyle.Render(helpText + strings.Repeat(" ", spacerW) + suffix)
	}

	return HelpStyle.Render(helpText)
}

// helpSuffix returns the dynamic right-aligned suffix for the help bar
// (QSO counter in log editor, scroll info in log viewer). This is
// computed every frame and not cached.
func (m *Model) helpSuffix() string {
	if m.screen == screenLogbookEditor && m.logbookEditor != nil {
		cursor := m.logbookEditor.CursorPos()
		total := m.logbookEditor.QSOCount()
		if total > 0 {
			return fmt.Sprintf("QSO %d/%d", cursor+1, total)
		}
	}
	if m.screen == screenLogView && m.logViewer != nil {
		return m.logViewer.ScrollInfo()
	}
	return ""
}

// renderHelpBar is the canonical entry point for help bar rendering.
func (m *Model) renderHelpBar() string { return m.helpView() }
