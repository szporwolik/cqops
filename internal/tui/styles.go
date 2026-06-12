package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Accent      lipgloss.Color
	Success     lipgloss.Color
	Error       lipgloss.Color
	Warning     lipgloss.Color
	Value       lipgloss.Color
	Background  lipgloss.Color
	Muted       lipgloss.Color
	Dim         lipgloss.Color
	Label       lipgloss.Color
	Section     lipgloss.Color
	Subtle      lipgloss.Color
	ActiveTab   lipgloss.Color
	Info        lipgloss.Color
	Debug       lipgloss.Color
}

var DefaultTheme = Theme{
	Accent:     "86",
	Success:    "46",
	Error:      "196",
	Warning:    "214",
	Value:      "229",
	Background: "236",
	Muted:      "241",
	Dim:        "238",
	Label:      "243",
	Section:    "244",
	Subtle:     "245",
	ActiveTab:  "62",
	Info:       "252",
	Debug:      "240",
}

var (
	th = &DefaultTheme

	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(th.Accent).Padding(0, 1)

	HeaderStyle = lipgloss.NewStyle().Foreground(th.Section).Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().Foreground(th.Error)

	WarningStyle = lipgloss.NewStyle().Foreground(th.Warning)

	SuccessStyle = lipgloss.NewStyle().Foreground(th.Success)

	HelpStyle = lipgloss.NewStyle().Foreground(th.Muted)

	LabelStyle = lipgloss.NewStyle().Foreground(th.Label)

	ValueStyle = lipgloss.NewStyle().Foreground(th.Value)

	CursorStyle = lipgloss.NewStyle().Foreground(th.Accent)

	InputStyle = lipgloss.NewStyle().Foreground(th.Value)

	SectionStyle = lipgloss.NewStyle().Foreground(th.Section)

	DimStyle = lipgloss.NewStyle().Foreground(th.Dim)

	SubtleStyle = lipgloss.NewStyle().Foreground(th.Subtle)

	BarStyle = lipgloss.NewStyle().Background(th.Background)

	ActiveTabStyle = lipgloss.NewStyle().Background(th.ActiveTab).Foreground(th.Value).Bold(true).Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().Background(th.Background).Foreground(th.Muted).Padding(0, 2)

	DisabledTabStyle = lipgloss.NewStyle().Background(th.Background).Foreground(th.Dim).Padding(0, 2)
)

var borderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(th.Accent)

var titleStyle = TitleStyle
var headerStyle = HeaderStyle
var errorStyle = ErrorStyle
var warningStyle = WarningStyle
var successStyle = SuccessStyle
var helpStyle = HelpStyle
var formLabelStyle = lipgloss.NewStyle().Width(22).Foreground(th.Label)
var inputStyle = InputStyle
var cursorStyle = CursorStyle
var sectionStyle = SectionStyle
var dimStyle = DimStyle
var selectedStyle = lipgloss.NewStyle().Foreground(th.Value).Bold(true)
