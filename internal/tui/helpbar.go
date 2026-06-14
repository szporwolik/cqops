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
		return HelpStyle.Render("\u2190/\u2192 choose  \u2022  enter confirm  \u2022  esc cancel")
	}

	// Sub-components with their own footer text take priority over screen-level bindings.
	if m.screen == screenCallbook && m.callbookMenu != nil {
		return HelpStyle.Render(m.callbookMenu.FooterText())
	}
	if m.screen == screenRigEdit && m.rigChooser != nil {
		return HelpStyle.Render(m.rigChooser.FooterText())
	}
	if m.screen == screenChooser && m.chooser != nil {
		return HelpStyle.Render(m.chooser.FooterText())
	}
	if m.screen == screenImage {
		return HelpStyle.Render("F2 / Esc to return to partner details")
	}

	bindings := m.ActiveBindings()
	if len(bindings) == 0 {
		bindings = []key.Binding{m.keys.Quit}
	}
	helpText := m.help.ShortHelpView(bindings)
	if helpText == "" {
		helpText = m.help.ShortHelpView([]key.Binding{m.keys.Quit})
	}
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
