package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
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
			fmt.Sprintf("%d unsent QSOs", le.wlUnsentCount),
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
		if !le.dlActive {
			// Download already completed but mode not yet advanced — show results.
			le.mode = edModeList
			le.dialog = nil
		} else {
			msg := "Downloading from Wavelog…"
			if le.dlCurrent > 0 && le.dlTotal > 0 {
				if le.dlCurrent != le.dlLastCur || le.dlTotal != le.dlLastTot {
					pct := le.dlCurrent * 100 / le.dlTotal
					le.dlCachedMsg = fmt.Sprintf("Processing QSO %d / %d (%d%%)",
						le.dlCurrent, le.dlTotal, pct)
					le.dlLastCur = le.dlCurrent
					le.dlLastTot = le.dlTotal
				}
				msg = le.dlCachedMsg
			}
			// Update message every frame so it reflects latest progress.
			if le.dialog == nil {
				d := NewDialog("Wavelog Download", msg,
					Option{Label: "Abort", Value: "abort"},
				)
				le.dialog = &d
			} else {
				le.dialog.Message = msg
			}
		}
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeWLDownloadResult:
		if errText := strings.TrimSpace(le.wlDownloadErr); errText != "" {
			msg := "Download failed: " + errText
			le.ensureDialog("Wavelog Download", msg,
				Option{Label: "OK", Value: "ok"},
			)
		} else {
			msg := fmt.Sprintf("Downloaded %d QSOs.", le.wlDownloadCount)
			if le.wlDownloadDupes > 0 {
				msg += fmt.Sprintf("\n%d duplicates skipped.", le.wlDownloadDupes)
			}
			if le.wlDownloadFailed > 0 {
				msg += fmt.Sprintf("\n%d failed.", le.wlDownloadFailed)
			}
			le.ensureDialog("Wavelog Download", msg,
				Option{Label: "OK", Value: "ok"},
			)
		}
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeImporting:
		if !le.dlActive {
			// Import already completed but mode not yet advanced — show results.
			// Carry the live progress counter into the result; the done handler
			// may not have run yet (channel close vs message ordering).
			if le.impInserted == 0 {
				le.impInserted = le.dlCurrent
			}
			le.mode = edModeImportResult
		} else {
			msg := "Importing ADIF…"
			if le.dlCurrent > 0 && le.dlTotal > 0 {
				msg = fmt.Sprintf("Inserted %d QSOs (~%d in file)",
					le.dlCurrent, le.dlTotal)
				if le.impDupes > 0 {
					msg += fmt.Sprintf(" — %d dupes", le.impDupes)
				}
			}
			if le.dialog == nil {
				d := NewDialog("ADIF Import", msg,
					Option{Label: "Abort", Value: "abort"},
				)
				le.dialog = &d
			} else {
				le.dialog.Message = msg
			}
		}
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeExporting:
		if !le.dlActive {
			if le.impInserted == 0 {
				le.impInserted = le.dlCurrent
			}
			le.mode = edModeExportResult
		} else {
			msg := "Exporting ADIF…"
			if le.dlCurrent > 0 && le.dlTotal > 0 {
				msg = fmt.Sprintf("Written %d / %d QSOs",
					le.dlCurrent, le.dlTotal)
			}
			if le.dialog == nil {
				d := NewDialog("ADIF Export", msg,
					Option{Label: "Abort", Value: "abort"},
				)
				le.dialog = &d
			} else {
				le.dialog.Message = msg
			}
		}
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeExportResult:
		msg := fmt.Sprintf("Exported %d QSOs.", le.impInserted)
		path := le.exportPath
		if path != "" {
			msg += fmt.Sprintf("\nFile: %s", filepath.Base(path))
		}
		if errText := strings.TrimSpace(le.impErr); errText != "" {
			msg = "Export failed: " + errText
		}
		le.ensureDialog("ADIF Export", msg,
			Option{Label: "OK", Value: "ok"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeImportResult:
		msg := fmt.Sprintf("Imported %d QSOs.", le.impInserted)
		if le.impDupes > 0 {
			msg += fmt.Sprintf("\n%d duplicates skipped.", le.impDupes)
		}
		if le.impFailed > 0 {
			msg += fmt.Sprintf("\n%d failed.", le.impFailed)
		}
		if errText := strings.TrimSpace(le.impErr); errText != "" {
			msg = "Import failed: " + errText
		}
		le.ensureDialog("ADIF Import", msg,
			Option{Label: "OK", Value: "ok"},
		)
		return tea.NewView(le.viewWithDialog(bodyW))

	case edModeEdit:
		contentH := contentHeight(le.height)
		if contentH < 10 {
			contentH = 10
		}
		return tea.NewView(le.viewEdit(bodyW, contentH))
	case edModeExport:
		return tea.NewView(le.viewExport(bodyW))
	case edModeImport:
		return tea.NewView(le.viewImport(bodyW))
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
		var sb strings.Builder
		sb.WriteString(strconv.Itoa(le.width))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.height))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(len(le.qsos)))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.currentPage))
		sb.WriteByte('|')
		sb.WriteString(strconv.Itoa(le.table.Cursor()))
		sb.WriteByte('|')
		sb.WriteString(le.contestID)
		sig := sb.String()
		if le.cachedSig == sig && le.cachedView != "" {
			return tea.NewView(le.cachedView)
		}
		contentH := contentHeight(le.height)

		// Contest info line — warning-colored row when contest is active.
		var headerLines []string
		if le.contestID != "" {
			contestLine := S.Warning.Render(fmt.Sprintf(" Contest: %s   Contest ID: %s",
				le.contestName, le.contestAdifID))
			if le.cachedSpacerStyleW != bodyW {
				le.cachedSpacerStyle = lipgloss.NewStyle().Width(bodyW)
				le.cachedSpacerStyleW = bodyW
			}
			headerLines = append(headerLines, le.cachedSpacerStyle.Render(contestLine))
			contentH-- // consume one row for the contest info line
		}

		// Spacer row + table.
		if le.cachedSpacerStyleW != bodyW {
			le.cachedSpacerStyle = lipgloss.NewStyle().Width(bodyW)
			le.cachedSpacerStyleW = bodyW
		}
		spacer := le.cachedSpacerStyle.Render("")
		// Table already handles its own width via WithWidth(bodyW); only
		// constrain height here. Applying Width/MaxWidth on top causes
		// lipgloss to word-wrap the table output instead of truncating.
		tablePart := lipgloss.NewStyle().
			Height(contentH - 1 - len(headerLines)).
			Render(le.table.View())

		allParts := append(headerLines, spacer, tablePart)
		le.cachedView = lipgloss.JoinVertical(lipgloss.Left, allParts...)
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
	// Spacer row + table (no border — the dialog provides its own).
	if le.cachedSpacerStyleW != bodyW {
		le.cachedSpacerStyle = lipgloss.NewStyle().Width(bodyW).MaxWidth(bodyW)
		le.cachedSpacerStyleW = bodyW
	}
	spacer := le.cachedSpacerStyle.Render("")
	// Table part uses its own cached style (no Width, only MaxWidth+Height)
	// to avoid forcing the table to pad to bodyW which can cause layout issues.
	// Only constrain height — table manages its own width.
	th := contentH - 1
	if le.cachedTablePartH != th {
		le.cachedTablePartStyle = lipgloss.NewStyle().Height(th)
		le.cachedTablePartH = th
	}
	tablePart := le.cachedTablePartStyle.Render(le.table.View())
	body := lipgloss.JoinVertical(lipgloss.Left, spacer, tablePart)
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
		applog.Debug("LogEditor: dialog shown", "title", title, "options", len(options))
	}
}

// viewExport renders the ADIF export directory picker screen.
func (le *LogbookEditor) viewExport(bodyW int) string {
	boxW := bodyW
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	// Leave room for the bordered menu box (title + border + padding ≈ 5 rows).
	// fpH = contentHeight - header(1) - border(2) - padding(2) - pathLine(1) - margin
	// filepicker View() renders Height()+1 rows (off-by-one in lib).
	// Overhead: header(1) + border(2) + padding(2) + pathLine(1) = 6, +1 fp bug = 7.
	fpH := contentHeight(le.height) - 7
	if fpH < 4 {
		fpH = 4
	}
	le.filePicker.SetHeight(fpH)

	pathLine := S.Info.Render("Path: ") + ValueStyle.Render(le.filePicker.CurrentDirectory)
	fpView := le.filePicker.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		pathLine,
		fpView,
	)
	return drawMenuWithHeader("ADIF Export - choose folder", content, boxW)
}

// viewImport renders the ADIF import file picker screen.
func (le *LogbookEditor) viewImport(bodyW int) string {
	boxW := bodyW
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	fpH := contentHeight(le.height) - 7
	if fpH < 4 {
		fpH = 4
	}
	le.filePicker.SetHeight(fpH)

	pathLine := S.Info.Render("Path: ") + ValueStyle.Render(le.filePicker.CurrentDirectory)
	fpView := le.filePicker.View()

	content := lipgloss.JoinVertical(lipgloss.Left,
		pathLine,
		fpView,
	)
	return drawMenuWithHeader("ADIF Import - choose .adi/.adif file", content, boxW)
}
