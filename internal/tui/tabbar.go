package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// tabView renders the function-key tab bar below the status bar.
// F1-F2 tabs are left-aligned; F6-F8 tabs are right-aligned.
func (m *Model) tabView() string {
	hasPartner := m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""

	type tab struct {
		label    string
		active   bool
		disabled bool
	}

	leftTabs := []tab{
		{"F1 QSO Form", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 Partner", (m.screen == screenPartner || m.screen == screenImage) && hasPartner, !hasPartner},
	}
	rightTabs := []tab{
		{"F6 Log Editor", m.screen == screenLogbookEditor, false},
		{"F7 Config", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit || m.screen == screenNotifications, false},
		{"F8 Logs", m.screen == screenLogView, false},
	}

	renderGroup := func(tabs []tab) string {
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
		return strings.Join(parts, S.TabGap.Render(" "))
	}

	left := renderGroup(leftTabs)
	right := renderGroup(rightTabs)

	fillerW := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if fillerW < 1 {
		fillerW = 1
	}
	return left + strings.Repeat(" ", fillerW) + right
}

// renderProfileLine returns station details (operator, rig, antenna, etc.).
// Returns empty string when no details are configured.
func (m *Model) renderProfileLine() string {
	s := m.App.Logbook.Station
	var parts []string
	if s.Operator != "" {
		parts = append(parts, "Op "+s.Operator)
	}
	rigModel := s.RigModel(m.App.Config.Rigs)
	rigAnt := s.RigAntenna(m.App.Config.Rigs)
	if rigModel != "" {
		parts = append(parts, "Rig "+rigModel)
	}
	if rigAnt != "" {
		parts = append(parts, "Ant "+rigAnt)
	}
	wl := m.App.Logbook.Wavelog
	if wl != nil && wl.Enabled && m.wlOnline {
		parts = append(parts, "WL "+config.LogbookDisplayName(m.App.Logbook))
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
	return lipgloss.NewStyle().Foreground(P.TextDim).Render("  " + strings.Join(parts, " \u00b7 "))
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
