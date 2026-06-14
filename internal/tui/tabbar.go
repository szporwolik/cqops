package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// tabView renders the function-key tab bar below the status bar.
func (m *Model) tabView() string {
	hasPartner := m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""

	type tab struct {
		label    string
		active   bool
		disabled bool
	}
	tabs := []tab{
		{"F1 QSO Form", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 Partner", m.screen == screenPartner && hasPartner, !hasPartner},
		{"F7 Log Editor", m.screen == screenLogbookEditor, false},
		{"F8 Config", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit, false},
		{"F9 Logs", m.screen == screenLogView, false},
	}

	var parts []string
	for _, t := range tabs {
		s := S.TabInactive
		if t.active {
			s = S.TabActive
		}
		if t.disabled {
			s = S.TabDisabled
		}
		parts = append(parts, s.Render(" "+t.label+" "))
	}

	row := strings.Join(parts, S.TabGap.Render(" "))
	return S.TabBar.Width(m.width).Render(row)
}

// renderProfileLine returns station details (operator, rig, antenna, etc.).
// Returns empty string when no details are configured.
func (m *Model) renderProfileLine() string {
	s := m.App.Logbook.Station
	var parts []string
	if s.Operator != "" {
		parts = append(parts, "Op "+s.Operator)
	}
	if s.Rig != "" {
		parts = append(parts, "Rig "+s.Rig)
	}
	if s.Antenna != "" {
		parts = append(parts, "Ant "+s.Antenna)
	}
	if s.Grid != "" {
		parts = append(parts, "Grid "+formatLocator(s.Grid))
	}
	if s.SOTARef != "" {
		parts = append(parts, "SOTA "+s.SOTARef)
	}
	if s.POTARef != "" {
		parts = append(parts, "POTA "+s.POTARef)
	}
	if s.WWFFRef != "" {
		parts = append(parts, "WWFF "+s.WWFFRef)
	}
	if len(parts) == 0 {
		return ""
	}
	return DimStyle.Render("  " + strings.Join(parts, " \u00b7 "))
}

// renderProfileBar returns the right-aligned profile line.
func (m *Model) renderProfileBar() string {
	if m.confirm == nil {
		line := m.renderProfileLine()
		if line == "" {
			return ""
		}
		return lipgloss.NewStyle().
			Width(m.width).
			MaxHeight(1).
			Align(lipgloss.Right).
			Render(line)
	}
	return ""
}

// renderTabBar is the canonical entry point for tab bar rendering.
func (m *Model) renderTabBar() string { return m.tabView() }
