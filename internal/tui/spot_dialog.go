package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// SpotDialog is a modal dialog for sending a DX spot.
type SpotDialog struct {
	Call    string
	FreqKhz float64

	input    textinput.Model
	selected int // 0 = Accept, 1 = Cancel
	done     bool
	width    int
	height   int

	Result SpotDialogResult
}

type SpotDialogResult struct {
	Confirmed bool
	Comment   string
	Cancelled bool
}

func NewSpotDialog(call string, freqKhz float64, comment string) SpotDialog {
	ti := textinput.New()
	ti.Placeholder = "spot comment (optional)"
	ti.SetValue(comment)
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(48)
	return SpotDialog{Call: call, FreqKhz: freqKhz, input: ti}
}

func (d SpotDialog) Init() tea.Cmd { return nil }

func (d SpotDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		return d, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "right":
			d.selected = 1 - d.selected
			return d, nil
		case "enter":
			d.done = true
			if d.selected == 0 {
				d.Result = SpotDialogResult{
					Confirmed: true,
					Comment:   strings.TrimSpace(d.input.Value()),
				}
			} else {
				d.Result = SpotDialogResult{Cancelled: true}
			}
			return d, nil
		case "esc":
			d.done = true
			d.Result = SpotDialogResult{Cancelled: true}
			return d, nil
		default:
			var cmd tea.Cmd
			d.input, cmd = d.input.Update(msg)
			return d, cmd
		}
	}
	return d, nil
}

func (d SpotDialog) Done() bool { return d.done }

func (d SpotDialog) View() tea.View { return tea.NewView(d.render()) }

func (d SpotDialog) render() string {
	const modalW = 50
	contentW := modalW - 6

	title := S.ConfirmTitle.Width(contentW).Align(lipgloss.Center).
		Render(fmt.Sprintf("Spot %s", d.Call))

	info := S.ConfirmMsg.Width(contentW).Align(lipgloss.Center).
		Render(fmt.Sprintf("%.1f kHz", d.FreqKhz))

	d.input.SetWidth(contentW)
	inputView := d.input.View()

	btnAccept := S.ConfirmBtnDim
	btnCancel := S.ConfirmBtnDim
	if d.selected == 0 {
		btnAccept = S.ConfirmBtn
	} else {
		btnCancel = S.ConfirmBtn
	}
	btns := lipgloss.JoinHorizontal(lipgloss.Center,
		btnAccept.Render(" Accept "),
		btnCancel.Render(" Cancel "),
	)
	btns = dialogBtnAlignStyle.Width(contentW).Render(btns)

	body := lipgloss.JoinVertical(lipgloss.Top,
		title,
		"",
		info,
		"",
		inputView,
		"",
		btns,
	)
	return confirmBoxStyle.Width(modalW).Render(body)
}

// RenderSpotDialogOverlay composites the spot dialog as a centered overlay.
func RenderSpotDialogOverlay(mainView string, dlg SpotDialog, viewW, viewH int) string {
	dlg.width = viewW
	dlg.height = viewH
	dialog := dlg.render()

	placed := lipgloss.Place(viewW, viewH, lipgloss.Center, lipgloss.Center, dialog,
		lipgloss.WithWhitespaceChars(" "),
	)

	base := lipgloss.NewLayer(mainView)
	dialogLayer := lipgloss.NewLayer(placed).X(0).Y(0).Z(1)

	return lipgloss.NewCompositor(base, dialogLayer).Render()
}
