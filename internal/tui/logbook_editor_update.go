package tui

import (
	"fmt"
	"strings"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Editor messages and update logic
// =============================================================================

type editorMsg struct {
	deleted       int64
	delCall       string
	delDate       string
	saved         int64
	saveCall      string
	saveDate      string
	purged        bool
	wlQSOID       int64
	wlCall        string
	wlOK          bool
	normalized    int
	skipped       int
	skipReason    string
	err           error
	dlCount       int
	dlDupes       int
	dlLastID      int64
	dlErr         string
}

func (le *LogbookEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		le.width = msg.Width
		le.height = msg.Height
		le.buildTable()

	case editorMsg:
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

		// Download result — any key dismisses.
		if le.mode == edModeWLDownloadResult {
			le.mode = edModeList
			le.needsReload = true
			return le, nil
		}

		// Normalize confirm — y=yes, anything else=cancel.
		if le.mode == edModeConfirmNormalize {
			if k == "y" {
				return le, le.doNormalizeAndUpload()
			}
			le.mode = edModeList
			return le, nil
		}

		// Confirm modes — route keys to the dialog with left/right navigation.
		if le.isConfirmMode() && le.dialog != nil {
			updated, _ := le.dialog.Update(msg)
			d := updated.(DialogModel)
			*le.dialog = d
			if d.Done() {
				if d.Result.Confirmed {
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
			case "w":
				return le, le.doUploadToWavelog()
			case "esc", "f6":
				le.mode = edModeList
			case "tab", "down":
				le.nextField()
			case "shift+tab", "up":
				le.prevField()
			default:
				if le.focus != qefWLStatus {
					le.fields[le.focus], _ = le.fields[le.focus].Update(msg)
				}
			}
			return le, nil
		}

		// modeList — table handles all navigation
		switch k {
		case "f6", "esc":
			le.done = true
		case "up", "down", "left", "right", "pgup", "pgdown", "home", "end", "k", "j", "h", "l":
			var cmd tea.Cmd
			le.table, cmd = le.table.Update(msg)
			return le, cmd
		case "delete":
			if len(le.qsos) > 0 {
				le.mode = edModeConfirmDelete
			}
		case "w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" && len(le.qsos) > 0 {
				le.mode = edModeConfirmWLSend
			}
		case "ctrl+w":
			if le.wlURL != "" && le.wlKey != "" && le.wlStationID != "" {
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
			le.mode = edModeConfirmPurge
		}
	}

	return le, nil
}

func (le *LogbookEditor) doConfirm() tea.Cmd {
	if le.dialog == nil {
		return nil
	}
	val := le.dialog.Result.Value
	le.dialog = nil
	switch val {
	case "wlsend":
		le.mode = edModeList
		return le.doBatchUpload()
	case "wldownload":
		le.mode = edModeList
		return le.doWavelogDownload()
	case "purge":
		le.mode = edModeList
		le.wlLastFetchedID = 0
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
	case "delete":
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
// and inserts new QSOs. Local duplicates are replaced with the Wavelog version.
func (le *LogbookEditor) doWavelogDownload() tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	fetchFromID := le.wlLastFetchedID
	db := le.db

	applog.InfoDetail("Wavelog: starting contacts download",
		fmt.Sprintf("url=%s station_id=%s from_id=%d", url, sid, fetchFromID))

	return func() tea.Msg {
		result, err := wavelog.FetchContacts(url, key, sid, fetchFromID)
		if err != nil {
			applog.ErrorDetail("Wavelog: contacts download failed",
				fmt.Sprintf("url=%s station_id=%s from_id=%d error=%v", url, sid, fetchFromID, err))
			return editorMsg{dlErr: err.Error()}
		}

		if result.ADIF == "" || result.ExportedQSOs == 0 {
			applog.Info("Wavelog: no new contacts to download")
			return editorMsg{dlCount: 0, dlLastID: result.LastFetchedID()}
		}

		// Parse ADIF records — same pattern as parseWSJTXADIF.
		var inserted, dupes int
		s := adif.NewScanner(strings.NewReader(result.ADIF))
		for s.Scan() {
			if s.IsHeader() {
				continue
			}
			r := s.Record()
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

			// Check for duplicate in local DB
			if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
				applog.Info("Wavelog: replacing local duplicate",
					"local_id", existingID, "call", qs.Call, "band", qs.Band, "date", qs.QSODate)
				if err := store.DeleteQSO(db, existingID); err != nil {
					applog.Error("Wavelog: failed to delete local duplicate", "id", existingID, "error", err)
				}
				dupes++
			}

			if _, err := store.InsertQSO(db, qs); err != nil {
				applog.Error("Wavelog: failed to insert downloaded QSO", "call", qs.Call, "error", err)
				continue
			}
			inserted++
		}

		applog.Info("Wavelog: contacts download complete",
			"inserted", inserted, "dupes_replaced", dupes, "last_id", result.LastFetchedID())

		return editorMsg{
			dlCount:  inserted,
			dlDupes:  dupes,
			dlLastID: result.LastFetchedID(),
		}
	}
}
