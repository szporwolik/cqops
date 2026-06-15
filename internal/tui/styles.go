package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette defines the semantic color tokens used throughout the TUI.
// Flat theme — relies on terminal's default dark background for performance.
type Palette struct {
	Text      color.Color // primary readable text
	TextMuted color.Color // subdued labels, secondary text
	TextDim   color.Color // very dim, disabled text
	Primary   color.Color // primary accent: cyan
	Success   color.Color // connected / OK / saved
	Warning   color.Color // pending / warning
	Error     color.Color // disconnected / error
	Info      color.Color // informational
	Accent    color.Color // subtle secondary accent (purple/violet)
	Border    color.Color // normal panel borders
	Cursor    color.Color // cursor/focus highlight (pink)
}

var P = Palette{
	Text:      lipgloss.Color("#DDE3EA"),
	TextMuted: lipgloss.Color("#9CA3AF"),
	TextDim:   lipgloss.Color("#6B7280"),
	Primary:   lipgloss.Color("#22D3EE"),
	Success:   lipgloss.Color("#22C55E"),
	Warning:   lipgloss.Color("#F59E0B"),
	Error:     lipgloss.Color("#EF4444"),
	Info:      lipgloss.Color("#38BDF8"),
	Accent:    lipgloss.Color("#A78BFA"),
	Border:    lipgloss.Color("#566170"),
	Cursor:    lipgloss.Color("212"),
}

// Styles collects all named semantic styles for the application.
type Styles struct {
	// Status bar
	StatusApp   lipgloss.Style
	StatusLabel lipgloss.Style
	StatusValue lipgloss.Style
	StatusTime  lipgloss.Style

	// Tabs
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	TabDisabled lipgloss.Style

	// Typography
	Title   lipgloss.Style
	Label   lipgloss.Style
	Value   lipgloss.Style
	Dim     lipgloss.Style
	Help    lipgloss.Style
	Error   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style
	Info    lipgloss.Style

	// Form
	FormLabel lipgloss.Style
	Input     lipgloss.Style
	Cursor    lipgloss.Style

	// Toasts
	ToastInfo    lipgloss.Style
	ToastSuccess lipgloss.Style
	ToastWarning lipgloss.Style
	ToastError   lipgloss.Style

	// Wizard
	WizardHeader lipgloss.Style
	WizardAccent lipgloss.Style

	// Log viewer
	LogInfo  lipgloss.Style
	LogWarn  lipgloss.Style
	LogError lipgloss.Style
	LogDebug lipgloss.Style

	// Map
	MapOwn     lipgloss.Style
	MapPartner lipgloss.Style
	MapBoth    lipgloss.Style
	MapGrid    lipgloss.Style

	// Confirm dialog
	ConfirmTitle  lipgloss.Style
	ConfirmMsg    lipgloss.Style
	ConfirmBtn    lipgloss.Style
	ConfirmBtnDim lipgloss.Style
	ConfirmDanger lipgloss.Style
	ConfirmHint   lipgloss.Style
	ConfirmHelp   lipgloss.Style
}

var S = Styles{
	StatusApp:   lipgloss.NewStyle().Foreground(P.Primary).Bold(true).Padding(0, 1),
	StatusLabel: lipgloss.NewStyle().Foreground(P.TextMuted),
	StatusValue: lipgloss.NewStyle().Foreground(P.Text),
	StatusTime:  lipgloss.NewStyle().Foreground(P.Text).Padding(0, 1),

	TabActive:   lipgloss.NewStyle().Bold(true).Foreground(P.Text).Padding(0, 1),
	TabInactive: lipgloss.NewStyle().Foreground(P.TextMuted).Padding(0, 1),
	TabDisabled: lipgloss.NewStyle().Foreground(P.TextDim).Padding(0, 1),

	Title:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	Label:   lipgloss.NewStyle().Foreground(P.TextMuted),
	Value:   lipgloss.NewStyle().Foreground(P.Text),
	Dim:     lipgloss.NewStyle().Foreground(P.TextDim),
	Help:    lipgloss.NewStyle().Foreground(P.TextMuted),
	Success: lipgloss.NewStyle().Foreground(P.Success),
	Warning: lipgloss.NewStyle().Foreground(P.Warning),
	Error:   lipgloss.NewStyle().Foreground(P.Error),
	Info:    lipgloss.NewStyle().Foreground(P.Info),

	FormLabel: lipgloss.NewStyle().Width(13).Foreground(P.TextMuted),
	Input:     lipgloss.NewStyle().Foreground(P.Text),
	Cursor:    lipgloss.NewStyle().Foreground(P.Cursor),

	ToastInfo:    lipgloss.NewStyle().Foreground(P.Info),
	ToastSuccess: lipgloss.NewStyle().Foreground(P.Success),
	ToastWarning: lipgloss.NewStyle().Foreground(P.Warning),
	ToastError:   lipgloss.NewStyle().Foreground(P.Error),

	WizardHeader: lipgloss.NewStyle().Bold(true).Foreground(P.Text),
	WizardAccent: lipgloss.NewStyle().Bold(true).Foreground(P.Primary),

	LogInfo:  lipgloss.NewStyle().Foreground(P.Info),
	LogWarn:  lipgloss.NewStyle().Foreground(P.Warning),
	LogError: lipgloss.NewStyle().Foreground(P.Error),
	LogDebug: lipgloss.NewStyle().Foreground(P.TextDim),

	MapOwn:     lipgloss.NewStyle().Foreground(P.Info).Bold(true),
	MapPartner: lipgloss.NewStyle().Foreground(P.Accent).Bold(true),
	MapBoth:    lipgloss.NewStyle().Foreground(P.Info).Bold(true),
	MapGrid:    lipgloss.NewStyle().Foreground(P.TextMuted),

	ConfirmTitle:  lipgloss.NewStyle().Bold(true).Foreground(P.Primary),
	ConfirmMsg:    lipgloss.NewStyle().Foreground(P.Text),
	ConfirmBtn:    lipgloss.NewStyle().Bold(true).Foreground(P.Text).Background(P.Primary).Padding(0, 2),
	ConfirmBtnDim: lipgloss.NewStyle().Foreground(P.TextDim).Padding(0, 2),
	ConfirmDanger: lipgloss.NewStyle().Bold(true).Foreground(P.Text).Background(P.Error).Padding(0, 2),
	ConfirmHint:   lipgloss.NewStyle().Foreground(P.TextDim),
	ConfirmHelp:   lipgloss.NewStyle().Foreground(P.TextDim),
}

// Package-level style aliases.
var (
	ErrorStyle   = S.Error
	SuccessStyle = S.Success
	HelpStyle    = S.Help
	LabelStyle   = S.Label
	ValueStyle   = S.Value
	CursorStyle  = S.Cursor
	InputStyle   = S.Input
	DimStyle     = S.Dim
)

var (
	// fieldFocusedLabel is the label style for active form fields (pink).
	fieldFocusedLabel = lipgloss.NewStyle().Width(13).Foreground(P.Cursor)
	// fieldFocusedPrefix is the "> " marker for active form fields.
	fieldFocusedPrefix = lipgloss.NewStyle().Foreground(P.Cursor)
	// fieldUnfocusedPrefix is the "  " marker for inactive form fields.
	fieldUnfocusedPrefix = lipgloss.NewStyle().Foreground(P.TextMuted)

	// pathInfoStyle is used for the short-path info line when grids are set.
	pathInfoStyle = lipgloss.NewStyle().Foreground(P.Info)
	// pathMutedStyle is used for the short-path info line when no path.
	pathMutedStyle = lipgloss.NewStyle().Foreground(P.TextMuted)

	// Border box style — simple border, no background fill.
	borderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(P.Border).
			Padding(0, 1)

	// confirmBoxStyle for dialog overlays.
	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(P.Border).
			Padding(1, 2)

	// statusDotOn / statusDotOff — pre-allocated integration indicator styles.
	statusDotOnStyle  = lipgloss.NewStyle().Foreground(P.Success)
	statusDotOffStyle = lipgloss.NewStyle().Foreground(P.Error)

	// utcLabelStyle — "UTC " prefix in status bar.
	utcLabelStyle = lipgloss.NewStyle().Foreground(P.TextMuted)

	// profileBarBase — right-aligned dim profile line.
	profileBarBase = lipgloss.NewStyle().Align(lipgloss.Right).Foreground(P.TextDim)

	// wizardCenterBase — centered layout helper for wizard.
	wizardCenterBase = lipgloss.NewStyle().Align(lipgloss.Center)
)
