package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
)

// helpView renders the bottom help/footer bar with context-sensitive key bindings.
func (m *Model) helpView() string {
	if m.confirm != nil {
		bindings := []key.Binding{
			key.NewBinding(key.WithKeys("←/→"), key.WithHelp("←/→", "choose")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		}
		return HelpStyle.Render(m.help.ShortHelpView(bindings))
	}

	bindings := m.ActiveBindings()
	if len(bindings) == 0 {
		bindings = []key.Binding{m.keys.Quit}
	}
	helpText := m.help.ShortHelpView(bindings)
	if helpText == "" {
		helpText = m.help.ShortHelpView([]key.Binding{m.keys.Quit})
	}

	// Log editor: append QSO counter.
	if m.screen == screenLogbookEditor && m.logbookEditor != nil {
		cursor := m.logbookEditor.CursorPos()
		total := m.logbookEditor.QSOCount()
		if total > 0 {
			counter := fmt.Sprintf("QSO %d/%d", cursor+1, total)
			counterW := lipgloss.Width(counter)
			spacerW := m.width - lipgloss.Width(helpText) - counterW - 2
			if spacerW < 1 {
				spacerW = 1
			}
			return HelpStyle.Render(helpText + strings.Repeat(" ", spacerW) + counter)
		}
	}
	// Log viewer: append scroll info.
	if m.screen == screenLogView && m.logViewer != nil {
		info := m.logViewer.ScrollInfo()
		if info != "" {
			infoW := lipgloss.Width(info)
			spacerW := m.width - lipgloss.Width(helpText) - infoW - 2
			if spacerW < 1 {
				spacerW = 1
			}
			return HelpStyle.Render(helpText + strings.Repeat(" ", spacerW) + info)
		}
	}
	return HelpStyle.Render(helpText)
}

// renderHelpBar is the canonical entry point for help bar rendering.
// The \x1b[0m reset clears any background colour that may have leaked
// from the body content above (e.g. trailing ANSI sequences from table
// cells, form fields, or border characters).
func (m *Model) renderHelpBar() string { return "\x1b[0m" + m.helpView() }
