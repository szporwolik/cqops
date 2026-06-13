package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

func (le *LogbookEditor) doBatchUpload() tea.Cmd {
	wlCall := le.wlStationCall
	logOp := le.logStationOp
	logGrid := le.logStationGrid

	// Collect unsent QSOs
	var unsent []qso.QSO
	for _, q := range le.qsos {
		if q.WavelogUploaded != "yes" {
			unsent = append(unsent, q)
		}
	}
	if len(unsent) == 0 {
		return func() tea.Msg {
			return editorMsg{wlOK: true, wlCall: "all sent", err: nil}
		}
	}

	// Detect mismatches against Wavelog station / logbook station defaults
	var mismatch []qso.QSO
	var fields []string
	hasCallMismatch := false
	hasOpMismatch := false
	hasGridMismatch := false
	for _, q := range unsent {
		callDiff := wlCall != "" && q.StationCallsign != "" && !strings.EqualFold(q.StationCallsign, wlCall)
		opDiff := logOp != "" && q.Operator != "" && !strings.EqualFold(q.Operator, logOp)
		gridDiff := logGrid != "" && q.MyGridSquare != "" && !strings.EqualFold(q.MyGridSquare, logGrid)
		if callDiff || opDiff || gridDiff {
			mismatch = append(mismatch, q)
			if callDiff {
				hasCallMismatch = true
			}
			if opDiff {
				hasOpMismatch = true
			}
			if gridDiff {
				hasGridMismatch = true
			}
		}
	}
	if hasCallMismatch {
		fields = append(fields, "callsign")
	}
	if hasOpMismatch {
		fields = append(fields, "operator")
	}
	if hasGridMismatch {
		fields = append(fields, "grid")
	}

	if len(mismatch) > 0 {
		le.mismatchQSOs = mismatch
		le.mismatchFields = fields
		le.mode = edModeConfirmNormalize
		return nil
	}

	return le.uploadBatch(unsent)
}

func (le *LogbookEditor) doNormalizeAndUpload() tea.Cmd {
	db := le.db
	mismatch := le.mismatchQSOs
	wlCall := le.wlStationCall
	logOp := le.logStationOp
	logGrid := le.logStationGrid

	// Collect all unsent QSOs (some may not be mismatched but still unsent)
	var unsent []qso.QSO
	for _, q := range le.qsos {
		if q.WavelogUploaded != "yes" {
			unsent = append(unsent, q)
		}
	}

	// Build list of IDs to normalize
	var normIDs []int64
	for _, q := range mismatch {
		normIDs = append(normIDs, q.ID)
	}

	applog.InfoDetail("Wavelog: normalizing station fields", fmt.Sprintf("count=%d call=%s op=%s grid=%s", len(normIDs), wlCall, logOp, logGrid))

	return func() tea.Msg {
		if err := store.NormalizeStationFields(db, normIDs, wlCall, logOp, logGrid); err != nil {
			applog.Error("Wavelog: normalization failed", "error", err)
			return editorMsg{wlOK: false, err: fmt.Errorf("normalize: %w", err)}
		}
		// Also update in-memory QSO list so the list view reflects changes
		for i := range le.qsos {
			for _, mid := range normIDs {
				if le.qsos[i].ID == mid {
					le.qsos[i].StationCallsign = wlCall
					le.qsos[i].Operator = logOp
					le.qsos[i].MyGridSquare = logGrid
					break
				}
			}
		}
		return editorMsg{normalized: len(normIDs)}
	}
}

func (le *LogbookEditor) uploadBatch(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	wlCall := le.wlStationCall

	// Build batch ADIF (override station callsign to match Wavelog station)
	var adifStr string
	for _, q := range unsent {
		adifStr += q.ToADIFWithStation(wlCall)
	}

	applog.InfoDetail("Wavelog: batch upload", fmt.Sprintf("count=%d", len(unsent)))

	return func() tea.Msg {
		result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
		if err != nil {
			// If the error contains duplicates, fall back to individual uploads
			// so that duplicates don't block new QSOs.
			errStr := err.Error()
			if strings.Contains(errStr, "Duplicate for") {
				applog.Warn("Wavelog: batch had duplicates, falling back to individual uploads", "count", len(unsent))
				return le.uploadIndividual(unsent)()
			}
			applog.Error("Wavelog: batch upload failed", "count", len(unsent), "error", err)
			return editorMsg{wlOK: false, err: err, wlCall: fmt.Sprintf("%d QSOs", len(unsent))}
		}
		if result != nil && result.AllDuplicates {
			for _, q := range unsent {
				store.UpdateWavelogStatus(db, q.ID, "yes")
			}
			applog.InfoDetail("Wavelog: batch already present (duplicates)", fmt.Sprintf("count=%d", len(unsent)))
			return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: fmt.Sprintf("%d QSOs (already on Wavelog)", len(unsent))}
		}
		for _, q := range unsent {
			store.UpdateWavelogStatus(db, q.ID, "yes")
		}
		applog.InfoDetail("Wavelog: batch upload OK", fmt.Sprintf("count=%d", len(unsent)))
		return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: fmt.Sprintf("%d QSOs", len(unsent))}
	}
}

// uploadIndividual sends each unsent QSO one at a time, silently handling
// duplicates. Used as fallback when a batch upload encounters mixed results.
func (le *LogbookEditor) uploadIndividual(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	wlCall := le.wlStationCall

	return func() tea.Msg {
		okCount := 0
		dupCount := 0
		failCount := 0
		var lastErr error

		for _, q := range unsent {
			adifStr := q.ToADIFWithStation(wlCall)
			result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
			if err != nil {
				applog.Warn("Wavelog: individual upload failed", "qso_id", q.ID, "call", q.Call, "error", err)
				store.UpdateWavelogStatus(db, q.ID, "no")
				failCount++
				lastErr = err
				continue
			}
			if result != nil && result.AllDuplicates {
				applog.Debug("Wavelog: duplicate (silent)", "qso_id", q.ID, "call", q.Call)
				store.UpdateWavelogStatus(db, q.ID, "yes")
				dupCount++
				continue
			}
			store.UpdateWavelogStatus(db, q.ID, "yes")
			okCount++
		}

		applog.InfoDetail("Wavelog: individual upload complete",
			fmt.Sprintf("ok=%d dup=%d fail=%d", okCount, dupCount, failCount))

		if failCount > 0 && okCount == 0 && dupCount == 0 {
			return editorMsg{wlOK: false, err: lastErr, wlCall: fmt.Sprintf("%d failed", failCount)}
		}
		msg := fmt.Sprintf("%d ok", okCount)
		if dupCount > 0 {
			msg += fmt.Sprintf(", %d already present", dupCount)
		}
		if failCount > 0 {
			msg += fmt.Sprintf(", %d failed", failCount)
		}
		return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: msg}
	}
}

func (le *LogbookEditor) doUploadToWavelog() tea.Cmd {
	if le.wlURL == "" || le.wlKey == "" || le.wlStationID == "" {
		return func() tea.Msg {
			return editorMsg{wlOK: false, err: fmt.Errorf("Wavelog not configured")}
		}
	}
	q := le.readEditForm()
	adifStr := q.ToADIFWithStation(le.wlStationCall)
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	qID := q.ID
	call := q.Call

	return func() tea.Msg {
		applog.InfoDetail("Wavelog: uploading from editor", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
		if err != nil {
			applog.Error("Wavelog: editor upload failed", "qso_id", qID, "call", call, "error", err)
			store.UpdateWavelogStatus(le.db, qID, "no")
			return editorMsg{wlQSOID: qID, wlCall: call, wlOK: false, err: err}
		}
		if result != nil && result.AllDuplicates {
			store.UpdateWavelogStatus(le.db, qID, "yes")
			applog.InfoDetail("Wavelog: editor QSO already present (duplicate)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
			return editorMsg{wlQSOID: qID, wlCall: call, wlOK: true}
		}
		store.UpdateWavelogStatus(le.db, qID, "yes")
		applog.InfoDetail("Wavelog: editor upload OK", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		return editorMsg{wlQSOID: qID, wlCall: call, wlOK: true}
	}
}
