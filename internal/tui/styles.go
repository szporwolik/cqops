package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette defines the semantic color tokens used throughout the TUI.
// Colours are chosen for a modern, readable ham-radio portable-operations
// console: dark field-radio background, muted grey labels, readable
// white/soft foreground, cyan/blue for active fields, green for OK,
// red for errors, amber for warnings.
type Palette struct {
	Background  color.Color // terminal background (very dark grey)
	Surface     color.Color // card/panel background
	SurfaceAlt  color.Color // alternate surface for contrast
	Text        color.Color // primary readable text
	TextMuted   color.Color // subdued labels, secondary text
	TextDim     color.Color // very dim, disabled text
	Primary     color.Color // primary accent: active field, selection
	PrimaryAlt  color.Color // softer primary variant
	Success     color.Color // connected / OK / saved
	Warning     color.Color // pending / warning
	Error       color.Color // disconnected / error
	Info        color.Color // informational / transient
	Accent      color.Color // subtle purple accent for special elements
	FieldBg     color.Color // form field background
	ActiveField color.Color // focused/active field border or indicator
}

var P = Palette{
	Background:  lipgloss.Color("#1a1b1e"), // very dark field-radio grey
	Surface:     lipgloss.Color("#25262a"), // dark panel
	SurfaceAlt:  lipgloss.Color("#2d2e32"), // slightly lighter panel
	Text:        lipgloss.Color("#e4e4e7"), // soft white foreground
	TextMuted:   lipgloss.Color("#909098"), // muted grey labels
	TextDim:     lipgloss.Color("#5c5c66"), // very dim/disabled
	Primary:     lipgloss.Color("#3b9eff"), // active field cyan-blue
	PrimaryAlt:  lipgloss.Color("#2d7dd2"), // darker primary
	Success:     lipgloss.Color("#34d399"), // green OK/connected
	Warning:     lipgloss.Color("#fbbf24"), // amber pending/warning
	Error:       lipgloss.Color("#ef4444"), // red disconnected/error
	Info:        lipgloss.Color("#67e8f9"), // cyan info
	Accent:      lipgloss.Color("#a78bfa"), // subtle purple accent
	FieldBg:     lipgloss.Color("#1e1f23"), // form field background
	ActiveField: lipgloss.Color("#60a5fa"), // focused field indicator
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
	QSOFormBox     lipgloss.Style // bordered box around QSO entry form
	RecentQSOsBox  lipgloss.Style // bordered box around recent QSOs table
	MapBox         lipgloss.Style // bordered box around map (no extra h-padding)
}

var S = Styles{
	StatusApp:      lipgloss.NewStyle().Foreground(lipgloss.Color("#1a1b1e")).Background(P.Primary).Bold(true).Padding(0, 1),
	StatusLabel:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	StatusValue:    lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	StatusFill:     lipgloss.NewStyle().Background(P.Surface),
	StatusRight:    lipgloss.NewStyle().Foreground(P.Text).Background(P.Accent),
	StatusDotOn:    lipgloss.NewStyle().Foreground(P.Success).Background(P.Accent),
	StatusDotOff:   lipgloss.NewStyle().Foreground(P.Error).Background(P.Accent),
	StatusTime:     lipgloss.NewStyle().Foreground(lipgloss.Color("#faf5ff")).Background(P.Accent).Padding(0, 1),
	TabActive:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1a1b1e")).Background(P.Primary).Padding(0, 1),
	TabInactive:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.SurfaceAlt).Padding(0, 1),
	TabDisabled:    lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Background).Padding(0, 1),
	TabGap:         lipgloss.NewStyle().Background(P.Background),
	TabBar:         lipgloss.NewStyle().Background(P.Background),
	Section:        lipgloss.NewStyle().Foreground(P.TextDim),
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
	FormLabel:      lipgloss.NewStyle().Width(13).Foreground(P.TextMuted),
	Input:          lipgloss.NewStyle().Foreground(P.Text),
	Cursor:         lipgloss.NewStyle().Foreground(P.Primary),
	ToastInfo:      lipgloss.NewStyle().Foreground(P.Info),
	ToastSuccess:   lipgloss.NewStyle().Foreground(P.Success),
	ToastWarning:   lipgloss.NewStyle().Foreground(P.Warning),
	ToastError:     lipgloss.NewStyle().Foreground(P.Error),
	WizardActive:   lipgloss.NewStyle().Foreground(P.Primary),
	WizardInactive: lipgloss.NewStyle().Foreground(P.TextDim),
	WizardHeader:   lipgloss.NewStyle().Bold(true).Foreground(P.Text),
	WizardAccent:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	WizardDim:      lipgloss.NewStyle().Foreground(P.TextMuted),
	WizardTag:      lipgloss.NewStyle().Italic(true).Foreground(P.Text),
	WizardSelected: lipgloss.NewStyle().Bold(true).Foreground(P.Text),
	LogInfo:        lipgloss.NewStyle().Foreground(P.Info),
	LogWarn:        lipgloss.NewStyle().Foreground(P.Warning),
	LogError:       lipgloss.NewStyle().Foreground(P.Error),
	LogDebug:       lipgloss.NewStyle().Foreground(P.TextDim),
	MapOwn:         lipgloss.NewStyle().Foreground(P.Info).Bold(true),
	MapPartner:     lipgloss.NewStyle().Foreground(P.Accent).Bold(true),
	MapBoth:        lipgloss.NewStyle().Foreground(P.Info).Bold(true),
	MapGrid:        lipgloss.NewStyle().Foreground(P.TextMuted),
	ConfirmBox: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.TextMuted).
		Background(P.Surface).
		Padding(1, 2),
	ConfirmTitle:  lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	ConfirmMsg:    lipgloss.NewStyle().Foreground(P.Text),
	ConfirmBtn:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1a1b1e")).Background(P.Primary).Padding(0, 1),
	ConfirmBtnDim: lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.SurfaceAlt).Padding(0, 1),
	ConfirmDanger: lipgloss.NewStyle().Bold(true).Foreground(P.Text).Background(P.Error).Padding(0, 1),
	ConfirmHelp:   lipgloss.NewStyle().Foreground(P.TextDim),
	BarStyle:      lipgloss.NewStyle().Background(P.Surface),
	TitleStyle:    lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	HeaderStyle:   lipgloss.NewStyle().Foreground(P.TextDim).Padding(0, 1),
	ErrorStyle:    lipgloss.NewStyle().Foreground(P.Error),
	WarningStyle:  lipgloss.NewStyle().Foreground(P.Warning),
	SuccessStyle:  lipgloss.NewStyle().Foreground(P.Success),
	HelpStyle:     lipgloss.NewStyle().Foreground(P.TextMuted),
	LabelStyle:    lipgloss.NewStyle().Foreground(P.TextMuted),
	ValueStyle:    lipgloss.NewStyle().Foreground(P.Text),
	CursorStyle:   lipgloss.NewStyle().Foreground(P.Primary),
	InputStyle:    lipgloss.NewStyle().Foreground(P.Text),
	DimStyle:      lipgloss.NewStyle().Foreground(P.TextDim),
	SubtleStyle:   lipgloss.NewStyle().Foreground(P.TextMuted),
	QSOFormBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim).
		Padding(0, 1),
	RecentQSOsBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim),
	MapBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.TextDim).
		Padding(0, 0), // no extra padding — map needs every cell
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
