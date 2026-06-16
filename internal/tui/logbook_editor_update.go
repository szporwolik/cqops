package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
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
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		le.width = msg.Width
		le.height = msg.Height
		le.buildTable()

	case editorMsg:
		// Batch download progress — only when a download is actually active.
		if le.dlActive && !msg.dlDone && msg.dlErr == "" {
			le.dlProgress = msg.dlProgress
			le.dlTotal = msg.dlTotal
			le.dlCurrent = msg.dlCount
			if le.mode != edModeWLDownloading {
				le.mode = edModeWLDownloading
			}
			return le, le.readDownloadMsg
		}

		// Download complete (or aborted).
		if le.dlActive && msg.dlDone {
			le.dlActive = false
			le.dlCancel = nil
			le.dialog = nil // clear stale dialog so results can render
			if msg.dlAborted {
				le.wlDownloadCount = msg.dlCount
				le.wlDownloadDupes = msg.dlDupes
				le.wlDownloadErr = ""
				le.mode = edModeWLDownloadResult
				le.needsReload = true
			} else if msg.dlErr != "" {
				le.wlDownloadErr = msg.dlErr
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

		// Download result — route keys to the dialog (OK button).
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
		}
	}

	// During download, always keep the channel reader alive.  Other messages
	// (ticks, flrig polls, etc.) would otherwise replace readDownloadMsg.
	if le.dlActive {
		return le, le.readDownloadMsg
	}
	return le, nil
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
	const batchInterval = 1 // report progress every QSO for smooth updates

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

		qs := qso.NewQSO()
		if v := r[adifield.CALL]; v != "" {
			qs.Call = strings.ToUpper(v)
		}
		if v := r[adifield.BAND]; v != "" {
			qs.Band = qso.NormalizeBand(v)
		}
		if v := r[adifield.MODE]; v != "" {
			qs.Mode = strings.ToUpper(v)
		}
		if v := r[adifield.SUBMODE]; v != "" {
			qs.Submode = strings.ToUpper(v)
		}
		if v := r[adifield.QSO_DATE]; v != "" {
			qs.QSODate = v
		}
		if v := r[adifield.TIME_ON]; v != "" {
			qs.TimeOn = v
		}
		if v := r[adifield.TIME_OFF]; v != "" {
			qs.TimeOff = v
		}
		if v := r[adifield.FREQ]; v != "" {
			fmt.Sscanf(v, "%f", &qs.Freq)
		}
		if v := r[adifield.FREQ_RX]; v != "" {
			fmt.Sscanf(v, "%f", &qs.FreqRx)
		}
		if v := r[adifield.RST_SENT]; v != "" {
			qs.RSTSent = v
		}
		if v := r[adifield.RST_RCVD]; v != "" {
			qs.RSTRcvd = v
		}
		if v := r[adifield.GRIDSQUARE]; v != "" {
			qs.GridSquare = v
		}
		if v := r[adifield.NAME]; v != "" {
			qs.Name = v
		}
		if v := r[adifield.QTH]; v != "" {
			qs.QTH = v
		}
		if v := r[adifield.COUNTRY]; v != "" {
			qs.Country = v
		}
		if v := r[adifield.COMMENT]; v != "" {
			qs.Comment = v
		}
		if v := r[adifield.NOTES]; v != "" {
			qs.Notes = v
		}
		if v := r[adifield.TX_PWR]; v != "" {
			qs.TXPower = v
		}
		if v := r[adifield.SOTA_REF]; v != "" {
			qs.SOTARef = v
		}
		if v := r[adifield.POTA_REF]; v != "" {
			qs.POTARef = v
		}
		if v := r[adifield.WWFF_REF]; v != "" {
			qs.WWFFRef = v
		}
		if v := r[adifield.IOTA]; v != "" {
			qs.IOTA = v
		}
		if v := r[adifield.MY_SOTA_REF]; v != "" {
			qs.MySOTARef = v
		}
		if v := r[adifield.MY_POTA_REF]; v != "" {
			qs.MyPOTARef = v
		}
		if v := r[adifield.MY_WWFF_REF]; v != "" {
			qs.MyWWFFRef = v
		}
		if v := r[adifield.STATION_CALLSIGN]; v != "" {
			qs.StationCallsign = strings.ToUpper(v)
		}
		if v := r[adifield.OPERATOR]; v != "" {
			qs.Operator = strings.ToUpper(v)
		}
		if v := r[adifield.MY_GRIDSQUARE]; v != "" {
			qs.MyGridSquare = v
		}
		if v := r[adifield.MY_RIG]; v != "" {
			qs.MyRig = v
		}
		if v := r[adifield.MY_ANTENNA]; v != "" {
			qs.MyAntenna = v
		}
		if v := r[adifield.DISTANCE]; v != "" {
			fmt.Sscanf(v, "%f", &qs.Distance)
		}
		qs.Source = "wavelog"
		qs.WavelogUploaded = "yes"

		if qs.Call == "" {
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
