package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette defines the semantic color tokens used throughout the TUI.
// Modern dark field-console theme — readable on low-end hardware, SSH, VNC, RDP.
type Palette struct {
	Background  color.Color // root app background (dark charcoal, not black)
	Surface     color.Color // panel/card background
	SurfaceAlt  color.Color // elevated section background
	Text        color.Color // primary readable text
	TextMuted   color.Color // subdued labels, secondary text
	TextDim     color.Color // very dim, disabled text
	Primary     color.Color // primary accent: cyan for active fields
	PrimaryAlt  color.Color // softer blue variant
	Success     color.Color // connected / OK / saved
	Warning     color.Color // pending / warning / action hint
	Error       color.Color // disconnected / error
	Info        color.Color // informational
	Accent      color.Color // subtle secondary accent (purple/violet)
	Border      color.Color // normal panel borders
	BorderFocus color.Color // focused/active border
	SelectedBg  color.Color // selected row/table background
	SelectedFg  color.Color // selected row/table text
}

var P = Palette{
	Background:  lipgloss.Color("#1E2024"), // root app background (darker charcoal)
	Surface:     lipgloss.Color("#24272D"), // panel/surface
	SurfaceAlt:  lipgloss.Color("#2B2F36"), // elevated/section
	Text:        lipgloss.Color("#DDE3EA"), // main readable text — brighter
	TextMuted:   lipgloss.Color("#9CA3AF"), // labels, secondary — legible
	TextDim:     lipgloss.Color("#6B7280"), // disabled/very muted
	Primary:     lipgloss.Color("#22D3EE"), // radio cyan accent
	PrimaryAlt:  lipgloss.Color("#60A5FA"), // soft blue
	Success:     lipgloss.Color("#22C55E"), // green
	Warning:     lipgloss.Color("#F59E0B"), // amber
	Error:       lipgloss.Color("#EF4444"), // red
	Info:        lipgloss.Color("#38BDF8"), // cyan info
	Accent:      lipgloss.Color("#A78BFA"), // subtle purple
	Border:      lipgloss.Color("#566170"), // steel grey border — slightly brighter
	BorderFocus: lipgloss.Color("#22D3EE"), // cyan active border
	SelectedBg:  lipgloss.Color("#164E63"), // dark teal selected row
	SelectedFg:  lipgloss.Color("#F3F4F6"), // near-white selected text
}

// Styles collects all named semantic styles for the application.
type Styles struct {
	StatusApp      lipgloss.Style
	StatusLabel    lipgloss.Style
	StatusValue    lipgloss.Style
	StatusFill     lipgloss.Style
	StatusRight    lipgloss.Style
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
	ConfirmHint    lipgloss.Style
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
	ContentBase    lipgloss.Style
	QSOFormBox     lipgloss.Style
	RecentQSOsBox  lipgloss.Style
	MapBox         lipgloss.Style
}

var S = Styles{
	StatusApp:   lipgloss.NewStyle().Foreground(P.Primary).Bold(true).Padding(0, 1),
	StatusLabel: lipgloss.NewStyle().Foreground(P.TextMuted),
	StatusValue: lipgloss.NewStyle().Foreground(P.Text),
	StatusFill:  lipgloss.NewStyle().Background(P.Surface),
	StatusRight: lipgloss.NewStyle().Foreground(P.TextDim),
	StatusTime:  lipgloss.NewStyle().Foreground(P.Text).Padding(0, 1),

	TabActive:   lipgloss.NewStyle().Bold(true).Foreground(P.Text).Background(P.Surface).Padding(0, 1),
	TabInactive: lipgloss.NewStyle().Foreground(P.TextMuted).Padding(0, 1),
	TabDisabled: lipgloss.NewStyle().Foreground(P.TextDim).Padding(0, 1),
	TabGap:      lipgloss.NewStyle(),
	TabBar:      lipgloss.NewStyle(),

	Section: lipgloss.NewStyle().Foreground(P.TextDim),
	Title:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	Label:   lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	Value:   lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	Dim:     lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Surface),
	Help:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	Success: lipgloss.NewStyle().Foreground(P.Success).Background(P.Surface),
	Warning: lipgloss.NewStyle().Foreground(P.Warning).Background(P.Surface),
	Error:   lipgloss.NewStyle().Foreground(P.Error).Background(P.Surface),
	Info:    lipgloss.NewStyle().Foreground(P.Info).Background(P.Surface),
	Debug:   lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Surface),

	FormLabel: lipgloss.NewStyle().Width(13).Foreground(P.TextMuted).Background(P.Surface),
	Input:     lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	Cursor:    lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(P.Surface),

	ToastInfo:    lipgloss.NewStyle().Foreground(P.Info),
	ToastSuccess: lipgloss.NewStyle().Foreground(P.Success),
	ToastWarning: lipgloss.NewStyle().Foreground(P.Warning),
	ToastError:   lipgloss.NewStyle().Foreground(P.Error),

	WizardActive:   lipgloss.NewStyle().Foreground(P.Primary),
	WizardInactive: lipgloss.NewStyle().Foreground(P.TextDim),
	WizardHeader:   lipgloss.NewStyle().Bold(true).Foreground(P.Text),
	WizardAccent:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	WizardDim:      lipgloss.NewStyle().Foreground(P.TextMuted),
	WizardTag:      lipgloss.NewStyle().Foreground(P.Text),
	WizardSelected: lipgloss.NewStyle().Bold(true).Foreground(P.Text),

	LogInfo:  lipgloss.NewStyle().Foreground(P.Info),
	LogWarn:  lipgloss.NewStyle().Foreground(P.Warning),
	LogError: lipgloss.NewStyle().Foreground(P.Error),
	LogDebug: lipgloss.NewStyle().Foreground(P.TextDim),

	MapOwn:     lipgloss.NewStyle().Foreground(P.Info).Bold(true).Background(P.Surface),
	MapPartner: lipgloss.NewStyle().Foreground(P.Accent).Bold(true).Background(P.Surface),
	MapBoth:    lipgloss.NewStyle().Foreground(P.Info).Bold(true).Background(P.Surface),
	MapGrid:    lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),

	ConfirmBox: lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.Border).
		Background(P.Surface).
		Padding(1, 2),
	ConfirmTitle:  lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Background(P.Surface),
	ConfirmMsg:    lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	ConfirmBtn:    lipgloss.NewStyle().Bold(true).Foreground(P.Background).Background(P.Primary).Padding(0, 2),
	ConfirmBtnDim: lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.SurfaceAlt).Padding(0, 2),
	ConfirmDanger: lipgloss.NewStyle().Bold(true).Foreground(P.Text).Background(P.Error).Padding(0, 2),
	ConfirmHint:   lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Surface),
	ConfirmHelp:   lipgloss.NewStyle().Foreground(P.TextDim),

	BarStyle:    lipgloss.NewStyle().Background(P.Surface),
	TitleStyle:  lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	HeaderStyle: lipgloss.NewStyle().Foreground(P.TextDim).Padding(0, 1),
	ErrorStyle:  lipgloss.NewStyle().Foreground(P.Error),
	WarningStyle: lipgloss.NewStyle().Foreground(P.Warning),
	SuccessStyle: lipgloss.NewStyle().Foreground(P.Success),
	HelpStyle:   lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	LabelStyle:  lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),
	ValueStyle:  lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	CursorStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(P.Surface),
	InputStyle:  lipgloss.NewStyle().Foreground(P.Text).Background(P.Surface),
	DimStyle:    lipgloss.NewStyle().Foreground(P.TextDim).Background(P.Surface),
	SubtleStyle: lipgloss.NewStyle().Foreground(P.TextMuted).Background(P.Surface),

	QSOFormBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.Border).
		Background(P.Surface).
		Padding(0, 1),
	RecentQSOsBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.Border).
		Background(P.Surface),
	MapBox: lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(P.Border).
		Background(P.Surface).
		Padding(0, 0),

	ContentBase: lipgloss.NewStyle().Background(P.Surface),
}

// Package-level style aliases — backed by S.* as the single source of truth.
var (
	ErrorStyle     = S.ErrorStyle
	SuccessStyle   = S.SuccessStyle
	HelpStyle      = S.HelpStyle
	LabelStyle     = S.LabelStyle
	ValueStyle     = S.ValueStyle
	CursorStyle    = S.CursorStyle
	InputStyle     = S.InputStyle
	DimStyle       = S.DimStyle
	SubtleStyle    = S.SubtleStyle
	TitleStyle     = S.TitleStyle
	HeaderStyle    = S.HeaderStyle
	SectionStyle   = S.Section
	formLabelStyle = S.FormLabel
	inputStyle     = S.InputStyle
	cursorStyle    = S.CursorStyle
	errorStyle     = S.ErrorStyle
)
