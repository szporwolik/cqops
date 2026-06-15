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
		le.ensureDialog(
			"Delete QSO",
			func() string {
				q := le.qsos[le.table.Cursor()]
				return q.Call + " from " + formatDate(q.QSODate)
			}(),
			DangerOption("Delete", "delete"),
			Option{Label: "Cancel", Value: "cancel"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeConfirmPurge:
		le.ensureDialog("Purge Logbook", "All QSOs will be permanently deleted.",
			DangerOption("Purge", "purge"),
			Option{Label: "Cancel", Value: "cancel"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeConfirmWLSend:
		le.ensureDialog("Send to Wavelog",
			func() string {
				unsent := 0
				for _, q := range le.qsos {
					if q.WavelogUploaded != "yes" {
						unsent++
					}
				}
				return fmt.Sprintf("%d unsent QSOs", unsent)
			}(),
			Option{Label: "Send", Value: "wlsend"},
			Option{Label: "Cancel", Value: "cancel"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeConfirmNormalize:
		le.ensureDialog(
			fmt.Sprintf("Normalize %d QSOs", len(le.mismatchQSOs)),
			fmt.Sprintf("%d unsent QSOs will be normalised.", len(le.mismatchQSOs)),
			Option{Label: "Normalize", Value: "normalize"},
			Option{Label: "Cancel", Value: "cancel"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeConfirmWLDownload:
		le.ensureDialog("Download from Wavelog",
			"Pull QSOs from Wavelog into local logbook.\nDuplicates will be replaced with Wavelog versions.",
			Option{Label: "Download", Value: "wldownload"},
			Option{Label: "Cancel", Value: "cancel"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeWLDownloading:
		if le.dlMsgCh == nil {
			le.mode = edModeList
			le.dialog = nil
		} else {
			msg := "Downloading ADIF from Wavelog…\nThis may take a while for large logbooks."
			if le.dlProgress > 0 {
				msg = fmt.Sprintf("Downloaded %d QSOs (%d%% of file)", le.dlProgress, le.dlTotal)
			}
			le.ensureDialog("Wavelog Download", msg,
				Option{Label: "Abort", Value: "abort"},
			)
		}
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeWLDownloadResult:
		msg := fmt.Sprintf("Downloaded %d QSOs.", le.wlDownloadCount)
		if le.wlDownloadDupes > 0 {
			msg += fmt.Sprintf("\n%d already in logbook, skipped.", le.wlDownloadDupes)
		}
		if le.wlDownloadErr != "" {
			msg = "Download failed: " + le.wlDownloadErr
		}
		le.ensureDialog("Wavelog Download", msg,
			Option{Label: "OK", Value: "ok"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))
	case edModeEdit:
		contentH := contentHeight(le.height)
		if contentH < 10 {
			contentH = 10
		}
		return tea.NewView(le.viewEdit(bodyW, contentH))
	default:
		// Rebuild the table when dimensions change (resize).
		if le.built && (le.width != le.builtW || le.height != le.builtH) {
			le.built = false
		}
		if !le.built && len(le.qsos) > 0 {
			le.buildTable()
		}
		// Cache the rendered table — large QSO sets are paginated.
		// Invalidate when QSO data, page, dimensions, or cursor change.
		sig := fmt.Sprintf("%d|%d|%d|%d|%d", le.width, le.height, len(le.qsos), le.currentPage, le.table.Cursor())
		if le.cachedSig == sig && le.cachedView != "" {
			return tea.NewView(le.cachedView)
		}
		contentH := contentHeight(le.height)
		// Spacer row (reserved for future use) + table.
		spacer := lipgloss.NewStyle().Width(bodyW).Render("")
		tablePart := lipgloss.NewStyle().
			MaxWidth(bodyW).
			Height(contentH - 1).
			Render(le.table.View())
		le.cachedView = lipgloss.JoinVertical(lipgloss.Left, spacer, tablePart)
		le.cachedSig = sig
		return tea.NewView(le.cachedView)
	}
}

// viewWithDialog renders the list view with the confirm dialog composited on top.
func (le *LogbookEditor) viewWithDialog(bodyW int) string {
	// Rebuild the table when dimensions change (resize).
	if le.built && (le.width != le.builtW || le.height != le.builtH) {
		le.built = false
	}
	if !le.built && len(le.qsos) > 0 {
		le.buildTable()
	}
	contentH := contentHeight(le.height)
	if contentH < 5 {
		contentH = 5
	}
	// Plain table (no border — the dialog provides its own).
	body := lipgloss.NewStyle().
		MaxWidth(bodyW).
		Height(contentH).
		Render(le.table.View())
	if le.dialog != nil {
		return RenderDialogOverlay(body, *le.dialog, bodyW, contentH)
	}
	return body
}

// ensureDialog creates the dialog if it doesn't exist yet (idempotent).
func (le *LogbookEditor) ensureDialog(title, message string, options ...Option) {
	if le.dialog == nil {
		d := NewDialog(title, message, options...)
		le.dialog = &d
	}
}
