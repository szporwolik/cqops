package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette defines the semantic color tokens used throughout the TUI.
type Palette struct {
	Background color.Color
	Surface    color.Color
	SurfaceAlt color.Color
	Text       color.Color
	TextMuted  color.Color
	TextDim    color.Color
	Primary    color.Color
	PrimaryAlt color.Color
	Success    color.Color
	Warning    color.Color
	Error      color.Color
	Info       color.Color
}

var P = Palette{
	Background: lipgloss.Color("232"),
	Surface:    lipgloss.Color("235"),
	SurfaceAlt: lipgloss.Color("234"),
	Text:       lipgloss.Color("252"),
	TextMuted:  lipgloss.Color("246"),
	TextDim:    lipgloss.Color("240"),
	Primary:    lipgloss.Color("39"),
	PrimaryAlt: lipgloss.Color("33"),
	Success:    lipgloss.Color("46"),
	Warning:    lipgloss.Color("214"),
	Error:      lipgloss.Color("196"),
	Info:       lipgloss.Color("51"),
}

// Styles collects all named semantic styles for the application.
type Styles struct {
	StatusApp      lipgloss.Style
	StatusLabel    lipgloss.Style
	StatusValue    lipgloss.Style
	StatusFill     lipgloss.Style
	StatusRight    lipgloss.Style
	StatusDotOn    lipgloss.Style
	StatusDotOff   lipgloss.Style
	StatusTime     lipgloss.Style
	TabActive      lipgloss.Style
	TabInactive    lipgloss.Style
	TabDisabled    lipgloss.Style
	TabGap         lipgloss.Style
	TabBar         lipgloss.Style
	Section        lipgloss.Style
	Title          lipgloss.Style
	Label          lipgloss.Style
	Value          lipgloss.Style
	Dim            lipgloss.Style
	Help           lipgloss.Style
	Error          lipgloss.Style
	Success        lipgloss.Style
	Warning        lipgloss.Style
	Info           lipgloss.Style
	Debug          lipgloss.Style
	FormLabel      lipgloss.Style
	Input          lipgloss.Style
	Cursor         lipgloss.Style
	ToastInfo      lipgloss.Style
	ToastSuccess   lipgloss.Style
	ToastWarning   lipgloss.Style
	ToastError     lipgloss.Style
	WizardActive   lipgloss.Style
	WizardInactive lipgloss.Style
	WizardHeader   lipgloss.Style
	WizardAccent   lipgloss.Style
	WizardDim      lipgloss.Style
	WizardTag      lipgloss.Style
	WizardSelected lipgloss.Style
	LogInfo        lipgloss.Style
	LogWarn        lipgloss.Style
	LogError       lipgloss.Style
	LogDebug       lipgloss.Style
	MapOwn         lipgloss.Style
	MapPartner     lipgloss.Style
	MapBoth        lipgloss.Style
	MapGrid        lipgloss.Style
	ConfirmBox     lipgloss.Style
	ConfirmTitle   lipgloss.Style
	ConfirmMsg     lipgloss.Style
	ConfirmBtn     lipgloss.Style
	ConfirmBtnDim  lipgloss.Style
	ConfirmDanger  lipgloss.Style
	ConfirmHelp    lipgloss.Style
	BarStyle       lipgloss.Style
	TitleStyle     lipgloss.Style
	HeaderStyle    lipgloss.Style
	ErrorStyle     lipgloss.Style
	WarningStyle   lipgloss.Style
	SuccessStyle   lipgloss.Style
	HelpStyle      lipgloss.Style
	LabelStyle     lipgloss.Style
	ValueStyle     lipgloss.Style
	CursorStyle    lipgloss.Style
	InputStyle     lipgloss.Style
	DimStyle       lipgloss.Style
	SubtleStyle    lipgloss.Style
}

var S = Styles{
	StatusApp:      lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(P.Primary).Bold(true).Padding(0, 1),
	StatusLabel:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	StatusValue:    lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	StatusFill:     lipgloss.NewStyle().Background(P.Surface),
	StatusRight:    lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("99")).Padding(0, 1),
	StatusDotOn:    lipgloss.NewStyle().Foreground(P.Success).Background(lipgloss.Color("99")),
	StatusDotOff:   lipgloss.NewStyle().Foreground(P.Error).Background(lipgloss.Color("99")),
	StatusTime:     lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("99")).Padding(0, 1),
	TabActive:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(P.Primary).Padding(0, 1),
	TabInactive:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.SurfaceAlt).Padding(0, 1),
	TabDisabled:    lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Background).Padding(0, 1),
	TabGap:         lipgloss.NewStyle().Background(P.Background),
	TabBar:         lipgloss.NewStyle().Background(P.Background),
	Section:        lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
	Title:          lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	Label:          lipgloss.NewStyle().Foreground(P.TextMuted),
	Value:          lipgloss.NewStyle().Foreground(P.Text),
	Dim:            lipgloss.NewStyle().Foreground(P.TextDim),
	Help:           lipgloss.NewStyle().Foreground(P.TextMuted),
	Error:          lipgloss.NewStyle().Foreground(P.Error),
	Success:        lipgloss.NewStyle().Foreground(P.Success),
	Warning:        lipgloss.NewStyle().Foreground(P.Warning),
	Info:           lipgloss.NewStyle().Foreground(P.Info),
	Debug:          lipgloss.NewStyle().Foreground(P.TextDim),
	FormLabel:      lipgloss.NewStyle().Width(22).Foreground(P.TextMuted),
	Input:          lipgloss.NewStyle().Foreground(P.Text),
	Cursor:         lipgloss.NewStyle().Foreground(P.Primary),
	ToastInfo:      lipgloss.NewStyle().Foreground(P.Info),
	ToastSuccess:   lipgloss.NewStyle().Foreground(P.Success),
	ToastWarning:   lipgloss.NewStyle().Foreground(P.Warning),
	ToastError:     lipgloss.NewStyle().Foreground(P.Error),
	WizardActive:   lipgloss.NewStyle().Foreground(P.Primary),
	WizardInactive: lipgloss.NewStyle().Foreground(P.TextDim),
	WizardHeader:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
	WizardAccent:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	WizardDim:      lipgloss.NewStyle().Foreground(P.TextMuted),
	WizardTag:      lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("229")),
	WizardSelected: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")),
	LogInfo:        lipgloss.NewStyle().Foreground(P.Info),
	LogWarn:        lipgloss.NewStyle().Foreground(P.Warning),
	LogError:       lipgloss.NewStyle().Foreground(P.Error),
	LogDebug:       lipgloss.NewStyle().Foreground(P.TextDim),
	MapOwn:         lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
	MapPartner:     lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true),
	MapBoth:        lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true),
	MapGrid:        lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
	ConfirmBox: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.TextMuted).
		Background(P.Surface).
		Padding(1, 2),
	ConfirmTitle:  lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	ConfirmMsg:    lipgloss.NewStyle().Foreground(P.Text),
	ConfirmBtn:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0")).Background(P.Primary).Padding(0, 1),
	ConfirmBtnDim: lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.SurfaceAlt).Padding(0, 1),
	ConfirmDanger: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(P.Error).Padding(0, 1),
	ConfirmHelp:   lipgloss.NewStyle().Foreground(P.TextDim),
	BarStyle:      lipgloss.NewStyle().Background(P.Surface),
	TitleStyle:    lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	HeaderStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Padding(0, 1),
	ErrorStyle:    lipgloss.NewStyle().Foreground(P.Error),
	WarningStyle:  lipgloss.NewStyle().Foreground(P.Warning),
	SuccessStyle:  lipgloss.NewStyle().Foreground(P.Success),
	HelpStyle:     lipgloss.NewStyle().Foreground(P.TextMuted),
	LabelStyle:    lipgloss.NewStyle().Foreground(P.TextMuted),
	ValueStyle:    lipgloss.NewStyle().Foreground(P.Text),
	CursorStyle:   lipgloss.NewStyle().Foreground(P.Primary),
	InputStyle:    lipgloss.NewStyle().Foreground(P.Text),
	DimStyle:      lipgloss.NewStyle().Foreground(P.TextDim),
	SubtleStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
}

// Legacy package-level aliases.
var (
	th               = P
	TitleStyle       = S.TitleStyle
	HeaderStyle      = S.HeaderStyle
	ErrorStyle       = S.ErrorStyle
	WarningStyle     = S.WarningStyle
	SuccessStyle     = S.SuccessStyle
	HelpStyle        = S.HelpStyle
	LabelStyle       = S.LabelStyle
	ValueStyle       = S.ValueStyle
	CursorStyle      = S.CursorStyle
	InputStyle       = S.InputStyle
	DimStyle         = S.DimStyle
	SubtleStyle      = S.SubtleStyle
	BarStyle         = S.BarStyle
	titleStyle       = S.TitleStyle
	headerStyle      = S.HeaderStyle
	errorStyle       = S.ErrorStyle
	helpStyle        = S.HelpStyle
	formLabelStyle   = S.FormLabel
	inputStyle       = S.InputStyle
	cursorStyle      = S.CursorStyle
	SectionStyle     = S.Section
	ActiveTabStyle   = S.TabActive
	InactiveTabStyle = S.TabInactive
	DisabledTabStyle = S.TabDisabled
)
