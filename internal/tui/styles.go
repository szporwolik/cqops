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

	// Tabs — now rendered via lipgloss tab border pattern in tabbar.go.
	// No longer using S.Tab* styles.

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
	FormLabel       lipgloss.Style
	FormLabelWide   lipgloss.Style // wider label for menus (17 cells)
	FormFocused     lipgloss.Style
	FormFocusedWide lipgloss.Style // wider focused label for menus (17 cells)
	FormLabelXL     lipgloss.Style // extra-wide for notifications (36 cells)
	FormFocusedXL   lipgloss.Style // extra-wide focused for notifications (36 cells)
	FormLabelGen    lipgloss.Style // medium for General settings (30 cells)
	FormFocusedGen  lipgloss.Style // medium focused for General settings (30 cells)
	FormLabelCtx    lipgloss.Style // wider for Contest (28 cells)
	FormFocusedCtx  lipgloss.Style // wider focused for Contest (28 cells)
	FormPrefixOn    lipgloss.Style
	FormPrefixOff   lipgloss.Style
	Input           lipgloss.Style
	Cursor          lipgloss.Style

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

	// Band plan tabs
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	TabSeparator lipgloss.Style

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
	StatusApp:   lipgloss.NewStyle().Foreground(P.Text).Bold(true).Padding(0, 1),
	StatusLabel: lipgloss.NewStyle().Foreground(P.TextMuted),
	StatusValue: lipgloss.NewStyle().Foreground(P.Text),
	StatusTime:  lipgloss.NewStyle().Foreground(P.Text).Padding(0, 1),

	Title:   lipgloss.NewStyle().Bold(true).Foreground(P.Primary).Padding(0, 1),
	Label:   lipgloss.NewStyle().Foreground(P.TextMuted),
	Value:   lipgloss.NewStyle().Foreground(P.Text),
	Dim:     lipgloss.NewStyle().Foreground(P.TextDim),
	Help:    lipgloss.NewStyle().Foreground(P.TextMuted),
	Success: lipgloss.NewStyle().Foreground(P.Success),
	Warning: lipgloss.NewStyle().Foreground(P.Warning),
	Error:   lipgloss.NewStyle().Foreground(P.Error),
	Info:    lipgloss.NewStyle().Foreground(P.Info),

	FormLabel:       lipgloss.NewStyle().Width(11).Foreground(P.TextMuted),
	FormLabelWide:   lipgloss.NewStyle().Width(22).Foreground(P.TextMuted),
	FormFocused:     lipgloss.NewStyle().Width(11).Foreground(P.Cursor),
	FormFocusedWide: lipgloss.NewStyle().Width(22).Foreground(P.Cursor),
	FormPrefixOn:    lipgloss.NewStyle().Foreground(P.Cursor),
	FormPrefixOff:   lipgloss.NewStyle().Foreground(P.TextMuted),
	Input:           lipgloss.NewStyle().Foreground(P.Text),
	Cursor:          lipgloss.NewStyle().Foreground(P.Cursor),

	// Extra-wide variants for menus with long labels (Notifications).
	FormLabelXL:   lipgloss.NewStyle().Width(36).Foreground(P.TextMuted),
	FormFocusedXL: lipgloss.NewStyle().Width(36).Foreground(P.Cursor),

	// Medium variants for General settings — wider than Wide, narrower than XL.
	FormLabelGen:   lipgloss.NewStyle().Width(30).Foreground(P.TextMuted),
	FormFocusedGen: lipgloss.NewStyle().Width(30).Foreground(P.Cursor),

	// Wider variants for Contest submenu — accommodate long exchange labels.
	FormLabelCtx:   lipgloss.NewStyle().Width(28).Foreground(P.TextMuted),
	FormFocusedCtx: lipgloss.NewStyle().Width(28).Foreground(P.Cursor),

	ToastInfo:    lipgloss.NewStyle().Foreground(P.Info),
	ToastSuccess: lipgloss.NewStyle().Foreground(P.Success),
	ToastWarning: lipgloss.NewStyle().Foreground(P.Warning),
	ToastError:   lipgloss.NewStyle().Foreground(P.Error),

	WizardHeader: lipgloss.NewStyle().Bold(true).Foreground(P.Text),
	WizardAccent: lipgloss.NewStyle().Bold(true).Foreground(P.Text),

	LogInfo:  lipgloss.NewStyle().Foreground(P.Info),
	LogWarn:  lipgloss.NewStyle().Foreground(P.Warning),
	LogError: lipgloss.NewStyle().Foreground(P.Error),
	LogDebug: lipgloss.NewStyle().Foreground(P.TextDim),

	// Map markers — bold on black background for maximum contrast over the map.
	MapOwn:     lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Background(lipgloss.Color("0")).Bold(true),
	MapPartner: lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Background(lipgloss.Color("0")).Bold(true),
	MapBoth:    lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Background(lipgloss.Color("0")).Bold(true),
	MapGrid:    lipgloss.NewStyle().Foreground(P.TextMuted),

	TabActive:    lipgloss.NewStyle().Bold(true).Foreground(P.Cursor),
	TabInactive:  lipgloss.NewStyle().Foreground(P.TextMuted),
	TabSeparator: lipgloss.NewStyle().Foreground(P.TextDim),

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
	// PartnerBlock is a pre-built style for wrapping the partner view to terminal width.
	PartnerBlock = lipgloss.NewStyle().Align(lipgloss.Left)
)

var (
	// fieldFocusedLabel is the label style for active form fields (pink).
	// These are thin aliases to S.Form* — kept for backward compat with QSO form.
	fieldFocusedLabel    = S.FormFocused
	fieldFocusedPrefix   = S.FormPrefixOn
	fieldUnfocusedPrefix = S.FormPrefixOff

	// pathInfoStyle is used for the short-path info line when grids are set.
	pathInfoStyle = lipgloss.NewStyle().Foreground(P.Info)
	// pathMutedStyle is used for the short-path info line when no path.
	pathMutedStyle = lipgloss.NewStyle().Foreground(P.TextMuted)

	// Border box style — rounded border, no background fill.
	borderBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(P.Border).
			Padding(0, 1)

	// confirmBoxStyle for dialog overlays.
	confirmBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(P.Border).
			Padding(1, 2)

	// dialogBtnAlignStyle — base style for dialog button row centering.
	dialogBtnAlignStyle = lipgloss.NewStyle().Align(lipgloss.Center)

	// contestBoxStyle for contest info box on QSO screen.
	contestBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(P.Warning).
			Padding(0, 1)

	// statusDotOn / statusDotOff — pre-allocated integration indicator styles.
	// Airbus philosophy: default (white) when online, red when offline.
	statusDotOnStyle   = lipgloss.NewStyle().Foreground(P.Text)
	statusDotOffStyle  = lipgloss.NewStyle().Foreground(P.Error)
	statusDotWarnStyle = lipgloss.NewStyle().Foreground(P.Warning)

	// wizardCenterBase — centered layout helper for wizard.
	wizardCenterBase = lipgloss.NewStyle().Align(lipgloss.Center)
)
