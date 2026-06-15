package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// =============================================================================
// LogbookEditor view rendering
// =============================================================================

func (le *LogbookEditor) View() tea.View {
	if le.done {
		return tea.NewView("")
	}
	bodyW := le.width // full terminal width for wider table
	if bodyW < 30 {
		bodyW = 30
	}

	switch le.mode {
	case edModeConfirmDelete:
		if le.dialog == nil {
			q := le.qsos[le.table.Cursor()]
			d := NewDialog("Delete QSO", q.Call+" from "+formatDate(q.QSODate),
				DangerOption("Delete", "delete"),
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmPurge:
		if le.dialog == nil {
			d := NewDialog("Purge Logbook", "All QSOs will be permanently deleted.",
				DangerOption("Purge", "purge"),
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmWLSend:
		if le.dialog == nil {
			unsent := 0
			for _, q := range le.qsos {
				if q.WavelogUploaded != "yes" {
					unsent++
				}
			}
			d := NewDialog("Send to Wavelog", fmt.Sprintf("%d unsent QSOs", unsent),
				Option{Label: "Send", Value: "wlsend"},
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeConfirmNormalize:
		return tea.NewView(le.viewNormalizeConfirm(bodyW))
	case edModeConfirmWLDownload:
		if le.dialog == nil {
			d := NewDialog("Download from Wavelog", "Pull QSOs from Wavelog into local logbook.\nDuplicates will be replaced with Wavelog versions.",
				Option{Label: "Download", Value: "wldownload"},
				Option{Label: "Cancel", Value: "cancel"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeWLDownloading:
		if le.dlMsgCh == nil {
			// Stale mode — no download in progress, fall back to list.
			le.mode = edModeList
			le.dialog = nil
		} else if le.dialog == nil {
			d := NewDialog("Wavelog Download", "Downloading ADIF from Wavelog…\nThis may take a while for large logbooks.",
				Option{Label: "Abort", Value: "abort"},
			)
			le.dialog = &d
		} else if le.dlProgress == 0 {
			le.dialog.Message = "Downloading ADIF from Wavelog…\nThis may take a while for large logbooks."
		} else {
			le.dialog.Message = fmt.Sprintf("Downloaded %d QSOs (%d%% of file)",
				le.dlProgress, le.dlTotal)
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeWLDownloadResult:
		if le.dialog == nil {
			msg := fmt.Sprintf("Downloaded %d QSOs.", le.wlDownloadCount)
			if le.wlDownloadDupes > 0 {
				msg += fmt.Sprintf("\n%d already in logbook, skipped.", le.wlDownloadDupes)
			}
			if le.wlDownloadErr != "" {
				msg = "Download failed: " + le.wlDownloadErr
			}
			d := NewDialog("Wavelog Download", msg,
				Option{Label: "OK", Value: "ok"},
			)
			le.dialog = &d
		}
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeEdit:
		contentH := contentHeight(le.height)
		if contentH < 10 {
			contentH = 10
		}
		return tea.NewView(le.viewEdit(bodyW, contentH))
	default:
		if !le.built && len(le.qsos) > 0 {
			le.buildTable()
		}
		contentH := contentHeight(le.height)
		inner := lipgloss.NewStyle().
			MaxWidth(bodyW - 2).
			Height(contentH - 2).
			Render(le.table.View())
		return tea.NewView(drawBorderedBox(inner, bodyW))
	}
}

// viewWithDialog renders the list view with the confirm dialog composited on top.
func (le *LogbookEditor) viewWithDialog(bodyW int) string {
	// Build the base list view
	if !le.built && len(le.qsos) > 0 {
		le.buildTable()
	}
	contentH := contentHeight(le.height)
	if contentH < 5 {
		contentH = 5
	}
	body := drawBorderedBox(
		lipgloss.NewStyle().
			MaxWidth(bodyW-2).
			Height(contentH-2).
			Render(le.table.View()),
		bodyW,
	)
	if le.dialog != nil {
		return RenderDialogOverlay(body, *le.dialog, bodyW, le.height)
	}
	return body
}

func (le *LogbookEditor) viewNormalizeConfirm(bodyW int) string {
	return confirmBoxStyle.Width(bodyW).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			S.ConfirmTitle.Render(fmt.Sprintf("Normalize %d QSOs", len(le.mismatchQSOs))),
			"",
			S.ConfirmMsg.Render(fmt.Sprintf("%d unsent QSOs will be normalised.", len(le.mismatchQSOs))),
			S.ConfirmHelp.Render("y = yes  ·  any other key = cancel"),
		),
	)
}
