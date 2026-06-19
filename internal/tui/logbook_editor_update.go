package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	tea "charm.land/bubbletea/v2"
	adif "github.com/farmergreg/adif/v5"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Editor messages and update logic
// =============================================================================

type editorMsg struct {
	deleted    int64
	delCall    string
	delDate    string
	saved      int64
	saveCall   string
	saveDate   string
	purged     bool
	wlQSOID    int64
	wlCall     string
	wlOK       bool
	wlDup      bool
	normalized int
	skipped    int
	skipReason string
	err        error
	dlCount    int
	dlDupes    int
	dlFailed   int
	dlLastID   int64
	dlErr      string
	// Batch download progress
	dlProgress int
	dlTotal    int
	dlDone     bool
	dlAborted  bool
	// Simple toast from the editor.
	toastWarn string
	toastOK   string
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		le.width = msg.Width
		le.height = msg.Height
		le.buildTable()

	case editorMsg:
		// Batch download/import/export progress — only when a download is actually active.
		if le.dlActive && !msg.dlDone && msg.dlErr == "" {
			le.dlProgress = msg.dlProgress
			le.dlTotal = msg.dlTotal
			le.dlCurrent = msg.dlCount
			if le.mode == edModeImport {
				le.mode = edModeImporting
			} else if le.mode == edModeExport {
				le.mode = edModeExporting
			} else if le.mode != edModeWLDownloading && le.mode != edModeImporting && le.mode != edModeExporting {
				le.mode = edModeWLDownloading
			}
			return le, le.readDownloadMsg
		}

		// Download/import/export error received before done signal.
		// Capture the error immediately so it's not lost before the channel closes.
		if le.dlActive && !msg.dlDone {
			if errText := strings.TrimSpace(msg.dlErr); errText != "" {
				if le.mode == edModeImporting || le.mode == edModeImport {
					le.impErr = errText
				} else if le.mode == edModeExporting || le.mode == edModeExport {
					le.impErr = errText // reuse impErr for export errors
				} else {
					le.wlDownloadErr = errText
				}
				return le, le.readDownloadMsg
			}
		}

		// Download/import/export complete (or aborted).
		if le.dlActive && msg.dlDone {
			le.dlActive = false
			le.dlCancel = nil
			le.dialog = nil // clear stale dialog so results can render

			// ADIF export result.
			if le.mode == edModeExporting || le.mode == edModeExport {
				le.impInserted = msg.dlCount // reuse as exported count
				if msg.dlAborted {
					le.impErr = ""
				} else if msg.dlErr != "" {
					le.impErr = msg.dlErr
				} else {
					le.impErr = ""
				}
				le.mode = edModeExportResult
				le.needsReload = true
				return le, nil
			}

			// ADIF import result.
			if le.mode == edModeImporting || le.mode == edModeImport {
				le.impInserted = msg.dlCount
				le.impDupes = msg.dlDupes
				le.impFailed = msg.dlFailed
				if msg.dlAborted {
					le.impErr = ""
				} else if msg.dlErr != "" {
					le.impErr = msg.dlErr
				} else {
					le.impErr = ""
				}
				le.mode = edModeImportResult
				le.needsReload = true
				return le, nil
			}

			// Wavelog download result.
			if msg.dlAborted {
				le.wlDownloadCount = msg.dlCount
				le.wlDownloadDupes = msg.dlDupes
				le.wlDownloadErr = ""
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			} else if msg.dlErr != "" {
				le.wlDownloadErr = msg.dlErr
				le.mode = edModeWLDownloadResult
			} else if le.wlDownloadErr != "" {
				// Error was already captured from a previous dlErr message;
				// keep it and transition to result screen.
				le.mode = edModeWLDownloadResult
			} else {
				le.wlDownloadCount = msg.dlCount
				le.wlDownloadDupes = msg.dlDupes
				le.wlDownloadFailed = msg.dlFailed
				le.wlDownloadErr = ""
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			}
			return le, nil
		}

		if msg.err != nil {
			// error handled by caller via toast
		}
		if msg.deleted != 0 || msg.saved != 0 || msg.purged || msg.wlCall != "" {
			le.mode = edModeList
			le.needsReload = true
		}
		if msg.normalized > 0 {
			// Normalization done, now upload all unsent QSOs (skip invalid).
			var unsent []qso.QSO
			for _, q := range le.qsos {
				if q.WavelogUploaded != "yes" {
					if q.Band == "" || q.Mode == "" || q.QSODate == "" {
						continue
					}
					unsent = append(unsent, q)
				}
			}
			return le, le.uploadBatch(unsent)
		}

	case tea.KeyPressMsg:
		k := msg.String()

		// File export/import mode — route ALL keys to the filepicker handler.
		if le.mode == edModeExport || le.mode == edModeImport {
			return le.handleFilePickerUpdate(msg)
		}

		// Download progress — route keys to the dialog (Abort button).
		if le.dlActive && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				if le.dlCancel != nil {
					close(le.dlCancel)
					le.dlCancel = nil
				}
				// Don't touch dlMsgCh — the goroutine needs it to send the
				// final dlDone message.  The editorMsg handler will clean up.
			}
			return le, le.readDownloadMsg
		}

		// Download/import result — route keys to the dialog (OK button).
		if le.mode == edModeWLDownloadResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}
		if le.mode == edModeImportResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}
		if le.mode == edModeExportResult && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				le.dialog = nil
				le.mode = edModeList
				le.needsReload = true
			}
			return le, nil
		}

		// Confirm modes — route keys to the dialog with left/right navigation.
		if le.isConfirmMode() && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				if d.Result.Confirmed && d.Result.Value != "cancel" {
					return le, le.doConfirm()
				}
				le.dialog = nil
				le.mode = edModeList
			}
			return le, nil
		}

		if le.mode == edModeEdit {
			switch k {
			case "ctrl+s":
				return le, le.doSave()
			case "esc", "f6":
				le.mode = edModeList
			case "tab", "down":
				le.nextField()
			case "shift+tab", "up":
				le.prevField()
			default:
				if le.focus != qefWLStatus && le.focus != qefSource {
					le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
				}
			}
			return le, nil
		}

		// modeList — table handles navigation; we intercept page transitions.
		switch k {
		case "f6", "esc":
			le.done = true
		case "pgup":
			le.goToPage(le.currentPage - 1)
		case "pgdown":
			le.goToPage(le.currentPage + 1)
		case "up", "down", "left", "right", "home", "end", "k", "j", "h", "l":
			// Before passing to table, check for page boundary overflow.
			cursor := le.table.Cursor()
			if k == "down" || k == "j" {
				if cursor >= len(le.qsos)-1 && le.currentPage < le.totalPages() {
					le.goToPage(le.currentPage + 1)
					return le, nil
				}
			}
			if k == "up" || k == "k" {
				if cursor <= 0 && le.currentPage > 1 {
					le.goToPage(le.currentPage - 1)
					// Set cursor to last row of the new page.
					if len(le.qsos) > 0 {
						le.table.SetCursor(len(le.qsos) - 1)
					}
					return le, nil
				}
			}
			var cmd tea.Cmd
			le.table, cmd = le.table.Update(msg)
			return le, cmd
		case "delete":
			if len(le.qsos) > 0 {
				le.dialog = nil
				le.mode = edModeConfirmDelete
			}
		case "w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" {
				if len(le.qsos) == 0 {
					return le, func() tea.Msg { return editorMsg{toastWarn: "Logbook is empty — nothing to upload"} }
				}
				le.dialog = nil
				le.mode = edModeConfirmWLSend
			}
		case "ctrl+w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" {
				le.dialog = nil
				le.mode = edModeConfirmWLDownload
			}
		case "e", "enter":
			if len(le.qsos) > 0 {
				idx := le.table.Cursor()
				if idx >= len(le.qsos) {
					idx = 0
				}
				q := le.qsos[idx]
				le.editing = &q
				le.fillEditForm(&q)
				le.focus = qefCall
				le.fields[le.focus].Focus()
				le.mode = edModeEdit
			}
		case "p":
			le.dialog = nil
			le.mode = edModeConfirmPurge
		case "ctrl+e":
			le.filePicker = filepicker.New()
			le.filePicker.FileAllowed = false
			le.filePicker.DirAllowed = true
			le.filePicker.AutoHeight = false
			le.filePicker.ShowHidden = false
			if home, err := os.UserHomeDir(); err == nil {
				le.filePicker.CurrentDirectory = home
			}
			le.mode = edModeExport
			return le, le.filePicker.Init()
		case "ctrl+i", "tab":
			// ctrl+i and Tab may be indistinguishable in some terminals.
			// Only trigger import in list mode (Tab in edit mode is for field navigation).
			if le.mode == edModeList {
				le.filePicker = filepicker.New()
				le.filePicker.AllowedTypes = []string{".adi", ".adif"}
				le.filePicker.FileAllowed = true
				le.filePicker.DirAllowed = true
				le.filePicker.AutoHeight = false
				le.filePicker.ShowHidden = false
				if home, err := os.UserHomeDir(); err == nil {
					le.filePicker.CurrentDirectory = home
				}
				le.mode = edModeImport
				return le, le.filePicker.Init()
			}
		}
	}

	// During download, always keep the channel reader alive.  Other messages
	// (ticks, flrig polls, etc.) would otherwise replace readDownloadMsg.
	if le.dlActive {
		return le, le.readDownloadMsg
	}

	// File export/import mode — route non-key messages to the filepicker.
	if le.mode == edModeExport || le.mode == edModeImport {
		if _, isKey := msg.(tea.KeyPressMsg); !isKey {
			return le.handleFilePickerUpdate(msg)
		}
		return le, nil
	}

	return le, nil
}

func (le *LogbookEditor) handleFilePickerUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+e", "ctrl+i", "esc":
			le.mode = edModeList
			return le, nil
		case "enter":
			if le.mode == edModeExport {
				// Export: confirm current directory, start async export with progress.
				dir := le.filePicker.CurrentDirectory
				if p := le.filePicker.Path; p != "" {
					dir = p
				}
				ts := time.Now().UTC().Format("20060102_150405")
				name := "cqops"
				if le.logStationOp != "" {
					name = strings.ToLower(strings.ReplaceAll(le.logStationOp, " ", "_"))
				}
				path := filepath.Join(dir, fmt.Sprintf("%s_%s.adi", ts, name))
				le.exportPath = path
				// Start async export with progress dialog.
				le.dlProgress = 0
				le.dlTotal = 0
				le.dlCurrent = 0
				le.dlActive = true
				le.dlMsgCh = make(chan editorMsg, 4)
				le.dlCancel = make(chan struct{})
				le.mode = edModeExporting
				go le.runExport(path)
				return le, le.readDownloadMsg
			}
			// Import: let filepicker handle selection via DidSelectFile below.
		}
	}

	var cmd tea.Cmd
	le.filePicker, cmd = le.filePicker.Update(msg)

	// Import mode: check if a file was selected.
	if le.mode == edModeImport {
		if didSelect, path := le.filePicker.DidSelectFile(msg); didSelect && path != "" {
			// Validate that the selected file is an ADIF file.
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".adi" && ext != ".adif" {
				return le, func() tea.Msg { return editorMsg{toastWarn: "Only .adi / .adif files can be imported"} }
			}
			// Start async import with progress dialog.
			le.dlProgress = 0
			le.dlTotal = 0
			le.dlCurrent = 0
			le.dlActive = true
			le.dlMsgCh = make(chan editorMsg, 4)
			le.dlCancel = make(chan struct{})
			le.impInserted = 0
			le.impDupes = 0
			le.impFailed = 0
			le.impErr = ""
			le.mode = edModeImporting
			go le.runImport(path)
			return le, le.readDownloadMsg
		}
	}

	return le, cmd
}

func (le *LogbookEditor) doConfirm() tea.Cmd {
	if le.dialog == nil {
		return nil
	}
	mode := le.mode // capture before clearing
	applog.Debug("LogEditor: dialog response", "mode", mode, "value", le.dialog.Result.Value)
	le.dialog = nil
	switch mode {
	case edModeConfirmNormalize:
		return le.doNormalizeAndUpload()
	case edModeConfirmWLSend:
		le.mode = edModeList
		return le.doBatchUpload()
	case edModeConfirmWLDownload:
		le.mode = edModeWLDownloading
		le.dlProgress = 0
		le.dlTotal = 0
		return le.doWavelogDownload()
	case edModeConfirmPurge:
		le.mode = edModeList
		le.wlLastFetchedID = 0
		// Clear any lingering download state.
		le.dlProgress = 0
		le.dlTotal = 0
		le.dlMsgCh = nil
		if le.dlCancel != nil {
			close(le.dlCancel)
			le.dlCancel = nil
		}
		applog.Warn("LogbookEditor: purging all QSOs")
		return func() tea.Msg {
			err := store.PurgeQSOs(le.db)
			if err != nil {
				applog.Error("LogbookEditor: purge failed", "error", err.Error())
			} else {
				applog.Info("LogbookEditor: all QSOs purged")
			}
			return editorMsg{purged: true, err: err}
		}
	case edModeConfirmDelete:
		q := le.qsos[le.table.Cursor()]
		call := q.Call
		date := formatDate(q.QSODate)
		id := q.ID
		le.mode = edModeList
		applog.Info("LogbookEditor: deleting QSO", "id", id, "call", call, "date", date)
		return func() tea.Msg {
			err := store.DeleteQSO(le.db, id)
			if err != nil {
				applog.Error("LogbookEditor: delete failed", "id", id, "call", call, "error", err.Error())
			} else {
				applog.Info("LogbookEditor: QSO deleted", "id", id, "call", call)
			}
			return editorMsg{deleted: id, delCall: call, delDate: date, err: err}
		}
	}
	le.mode = edModeList
	return nil
}

func (le *LogbookEditor) doSave() tea.Cmd {
	q := le.readEditForm()
	call := q.Call
	date := formatDate(q.QSODate)
	id := q.ID
	applog.Info("LogbookEditor: saving QSO", "id", id, "call", call, "date", date)
	return func() tea.Msg {
		err := store.UpdateQSO(le.db, q)
		if err != nil {
			applog.Error("LogbookEditor: save failed", "id", id, "call", call, "error", err.Error())
		} else {
			applog.Info("LogbookEditor: QSO saved", "id", id, "call", call)
		}
		return editorMsg{saved: id, saveCall: call, saveDate: date, err: err}
	}
}

// doWavelogDownload fetches contacts from Wavelog, deduplicates against local DB,
// and inserts new QSOs in batches. Progress is reported via editorMsg so the UI
// can show a live counter. The user can abort by pressing any key.
func (le *LogbookEditor) doWavelogDownload() tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	fetchFromID := le.wlLastFetchedID

	// Channels for progress and cancellation.
	le.dlMsgCh = make(chan editorMsg, 4)
	le.dlCancel = make(chan struct{})
	le.dlActive = true

	applog.InfoDetail("Wavelog: starting contacts download",
		fmt.Sprintf("url=%s station_id=%s from_id=%d", url, sid, fetchFromID))

	// Start the download goroutine.
	go le.runDownload(url, key, sid, fetchFromID)

	// Return a Cmd that reads the first progress message.
	return le.readDownloadMsg
}

// readDownloadMsg reads the next message from the download channel.
// Returns the message to the Bubble Tea runtime for processing.
func (le *LogbookEditor) readDownloadMsg() tea.Msg {
	msg, ok := <-le.dlMsgCh
	if !ok {
		return editorMsg{dlDone: true}
	}
	return msg
}

// runDownload performs the actual HTTP fetch, saves ADIF to a temp file,
// then processes it line-by-line. Progress is reported per batch (every 50 QSOs
// or when done). The goroutine checks le.dlCancel between records; on abort
// processing stops immediately. Downloads are idempotent — duplicates are
// detected and skipped.
func (le *LogbookEditor) runDownload(url, key, sid string, fetchFromID int64) {
	// Capture the channel locally so the UI handler can't nil it out from
	// under us (setting le.dlMsgCh = nil would break sends and close).
	msgCh := le.dlMsgCh
	defer close(msgCh)

	db := le.db

	// Send initial message so the dialog appears immediately.
	msgCh <- editorMsg{dlProgress: 0, dlTotal: 0}

	result, err := wavelog.FetchContacts(url, key, sid, fetchFromID)
	if err != nil {
		applog.ErrorDetail("Wavelog: contacts download failed",
			fmt.Sprintf("url=%s station_id=%s from_id=%d error=%v", url, sid, fetchFromID, err))
		msgCh <- editorMsg{dlErr: err.Error()}
		return
	}

	// Clean up temp file when done.
	defer os.Remove(result.ADIFPath)

	if result.ADIFPath == "" || result.ExportedQSOs == 0 {
		applog.Info("Wavelog: no new contacts to download")
		msgCh <- editorMsg{dlCount: 0, dlLastID: result.LastFetchedID(), dlDone: true}
		return
	}

	// Open the temp ADIF file for line-by-line scanning.
	f, err := os.Open(result.ADIFPath)
	if err != nil {
		applog.Error("Wavelog: failed to open temp ADIF file", "error", err)
		msgCh <- editorMsg{dlErr: "failed to read downloaded data"}
		return
	}
	defer f.Close()

	var inserted, dupes, failed int
	totalExported := result.ExportedQSOs
	const batchInterval = 50 // report progress every 50 QSOs for smooth but efficient UI

	applog.Info("Wavelog: scanning ADIF", "exported", totalExported, "size_bytes", result.ADIFSize)

	// Stream-parse ADIF records one at a time — never loads all QSOs into memory.
	scanner := adif.NewScanner(f)
	processed := 0 // total records seen (including skipped/dupes)
	for scanner.Scan() {
		// Check for abort between records.
		select {
		case <-le.dlCancel:
			msgCh <- editorMsg{dlCount: inserted, dlDupes: dupes, dlAborted: true, dlDone: true}
			return
		default:
		}

		if scanner.IsHeader() {
			continue
		}
		r := scanner.Record()
		processed++

		qs := qso.ParseADIFRecord(r, "wavelog")
		qs.WavelogUploaded = "yes"

		// Enrich: compute distance/bearing if both grids are available.
		if myGrid := strings.TrimSpace(le.logStationGrid); myGrid != "" && qs.GridSquare != "" {
			qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
			qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
		}

		if err := qso.ValidateImportRecord(qs); err != nil {
			applog.Warn("Wavelog: skipping invalid imported QSO", "call", qs.Call, "reason", err)
			failed++
			continue
		}

		if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
			applog.Warn("Wavelog: duplicate QSO in ADIF — already imported this session",
				"local_id", existingID, "call", qs.Call, "band", qs.Band, "date", qs.QSODate)
			dupes++
			continue
		}

		// Retry on SQLITE_BUSY — the main thread may briefly hold a write lock
		// (e.g. during loadPage after purge).  A short sleep + retry resolves
		// nearly all transient lock conflicts.
		var insertErr error
		for attempt := 0; attempt < 3; attempt++ {
			_, insertErr = store.InsertQSO(db, qs)
			if insertErr == nil {
				break
			}
			if !strings.Contains(insertErr.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if insertErr != nil {
			applog.Error("Wavelog: failed to insert downloaded QSO", "call", qs.Call, "error", insertErr)
			failed++
			continue
		}
		inserted++

		// Report progress every batchInterval QSOs, and always for the first
		// one (so the UI transitions from "Downloading…" to "Processing QSO 1").
		if inserted == 1 || inserted%batchInterval == 0 {
			msgCh <- editorMsg{dlProgress: totalExported, dlTotal: totalExported, dlCount: inserted}
		}

		// Log milestone every 500 QSOs so we can trace progress in logs.
		if inserted%500 == 0 && inserted > 0 {
			applog.Info("Wavelog: download progress", "inserted", inserted, "dupes", dupes, "processed", processed)
		}
	}

	// Send final progress (catch remainder if last batch was partial).
	if inserted%batchInterval != 0 {
		msgCh <- editorMsg{dlProgress: totalExported, dlTotal: totalExported, dlCount: inserted}
	}

	if err := scanner.Err(); err != nil {
		applog.Error("Wavelog: ADIF scanner error", "error", err, "inserted", inserted, "dupes", dupes)
	}

	applog.Info("Wavelog: contacts download complete",
		"inserted", inserted, "dupes", dupes, "failed", failed, "processed", processed, "last_id", result.LastFetchedID())
	if dupes > 0 {
		applog.Warn("Wavelog: ADIF export contains duplicate QSO records — skipped during import",
			"dupe_count", dupes,
			"note", "These QSOs exist more than once in the Wavelog database and should be cleaned up at the source.")
	}

	msgCh <- editorMsg{
		dlCount:  inserted,
		dlDupes:  dupes,
		dlFailed: failed,
		dlLastID: result.LastFetchedID(),
		dlDone:   true,
	}
}

// runImport performs ADIF import from a local file with progress reporting.
// It mirrors runDownload but reads from a local file instead of Wavelog HTTP.
func (le *LogbookEditor) runImport(path string) {
	msgCh := le.dlMsgCh
	defer close(msgCh)

	db := le.db

	// Count total records for progress estimation.
	totalRecords := countADIFRecords(path)
	msgCh <- editorMsg{dlProgress: totalRecords, dlTotal: totalRecords, dlCount: 0}

	f, err := os.Open(path)
	if err != nil {
		applog.Error("ADIF import: failed to open file", "path", path, "error", err)
		msgCh <- editorMsg{dlErr: "cannot open file: " + err.Error()}
		return
	}
	defer f.Close()

	var inserted, dupes, failed int
	const batchInterval = 50 // report every 50 QSOs for smooth but efficient UI

	applog.Info("ADIF import: scanning", "path", path, "estimated_records", totalRecords)

	scanner := adif.NewScanner(f)
	for scanner.Scan() {
		// Check for abort between records.
		select {
		case <-le.dlCancel:
			msgCh <- editorMsg{dlCount: inserted, dlDupes: dupes, dlAborted: true, dlDone: true}
			return
		default:
		}

		if scanner.IsHeader() {
			continue
		}
		r := scanner.Record()
		qs := qso.ParseADIFRecord(r, "import")

		// Enrich: compute distance/bearing if both grids are available.
		if myGrid := strings.TrimSpace(le.logStationGrid); myGrid != "" && qs.GridSquare != "" {
			qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
			qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
		}

		if err := qso.ValidateImportRecord(qs); err != nil {
			applog.Warn("ADIF import: skipping invalid QSO", "call", qs.Call, "reason", err)
			failed++
			continue
		}

		if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
			applog.Warn("ADIF import: duplicate QSO skipped",
				"local_id", existingID, "call", qs.Call, "band", qs.Band, "date", qs.QSODate)
			dupes++
			continue
		}

		var insertErr error
		for attempt := 0; attempt < 3; attempt++ {
			_, insertErr = store.InsertQSO(db, qs)
			if insertErr == nil {
				break
			}
			if !strings.Contains(insertErr.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if insertErr != nil {
			applog.Error("ADIF import: failed to insert QSO", "call", qs.Call, "error", insertErr)
			failed++
			continue
		}
		inserted++

		// Report progress every batchInterval QSOs, and always for the first
		// one so the UI transitions from "Importing…" to showing a count.
		if inserted == 1 || inserted%batchInterval == 0 {
			msgCh <- editorMsg{dlProgress: totalRecords, dlTotal: totalRecords, dlCount: inserted, dlDupes: dupes}
		}

		if inserted%500 == 0 && inserted > 0 {
			applog.Info("ADIF import: progress", "inserted", inserted, "dupes", dupes, "failed", failed)
		}
	}

	if err := scanner.Err(); err != nil {
		applog.Error("ADIF import: scanner error", "error", err, "inserted", inserted, "dupes", dupes)
	}

	applog.Info("ADIF import: complete", "inserted", inserted, "dupes", dupes, "failed", failed, "path", path)

	msgCh <- editorMsg{
		dlCount:  inserted,
		dlDupes:  dupes,
		dlFailed: failed,
		dlDone:   true,
	}
}

// countADIFRecords quickly estimates the number of ADIF records in a file
// by counting "<CALL:" occurrences. This is fast and doesn't parse the full ADIF.
func countADIFRecords(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	// Read first 1MB to estimate record count.
	buf := make([]byte, 1024*1024)
	n, _ := f.Read(buf)
	if n == 0 {
		return 0
	}

	count := 0
	data := string(buf[:n])
	// Count ADIF field markers to estimate records — rough but fast.
	for {
		idx := strings.Index(data, "<CALL:")
		if idx == -1 {
			break
		}
		count++
		data = data[idx+1:]
	}

	// If file is larger than 1MB, extrapolate.
	fi, err := f.Stat()
	if err == nil && fi.Size() > int64(n) {
		count = count * int(fi.Size()) / n
	}

	return count
}

// runExport performs ADIF export to a local file with progress reporting.
func (le *LogbookEditor) runExport(path string) {
	msgCh := le.dlMsgCh
	defer close(msgCh)

	db := le.db

	// Count total QSOs for progress.
	counts, err := store.CountQSOs(db)
	total := 0
	if err == nil {
		total = counts.Total
	}
	msgCh <- editorMsg{dlProgress: total, dlTotal: total, dlCount: 0}

	if total == 0 {
		msgCh <- editorMsg{dlErr: "logbook is empty"}
		return
	}

	f, err := os.Create(path)
	if err != nil {
		applog.Error("ADIF export: failed to create file", "path", path, "error", err)
		msgCh <- editorMsg{dlErr: "cannot create file: " + err.Error()}
		return
	}
	defer f.Close()

	// Write ADIF header. Per ADIF spec, the first character must not be '<'
	// or the file is treated as having no header.
	_, err = fmt.Fprintf(f, "CQOps ADIF Export\n<ADIF_VER:5>3.1.4<PROGRAMID:5>CQOps<PROGRAMVERSION:%d>%s<EOH>\n",
		len(version.Version), version.Version)
	if err != nil {
		applog.Error("ADIF export: failed to write header", "path", path, "error", err)
		msgCh <- editorMsg{dlErr: "write error: " + err.Error()}
		return
	}

	// Stream QSOs from DB with pagination to avoid loading all into memory.
	const pageSize = 500
	written := 0
	for offset := 0; offset < total; offset += pageSize {
		select {
		case <-le.dlCancel:
			msgCh <- editorMsg{dlCount: written, dlAborted: true, dlDone: true}
			return
		default:
		}

		limit := pageSize
		if offset+limit > total {
			limit = total - offset
		}
		qsos, err := store.ListQSOsPage(db, limit, offset)
		if err != nil {
			applog.Error("ADIF export: failed to list QSOs", "offset", offset, "error", err)
			msgCh <- editorMsg{dlErr: "database read error: " + err.Error()}
			return
		}
		for _, q := range qsos {
			if _, err := fmt.Fprintln(f, q.ToADIF()); err != nil {
				applog.Error("ADIF export: write error", "offset", offset, "error", err)
				msgCh <- editorMsg{dlErr: "write error: " + err.Error()}
				return
			}
			written++
		}
		// Report progress after each page.
		msgCh <- editorMsg{dlProgress: total, dlTotal: total, dlCount: written}

		if written%1000 == 0 && written > 0 {
			applog.Info("ADIF export: progress", "written", written, "total", total)
		}
	}

	applog.Info("ADIF export: complete", "path", path, "count", written)
	msgCh <- editorMsg{dlCount: written, dlDone: true}
}
