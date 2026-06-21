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
	wlCall := ""
	logOp := le.logStationOp
	logGrid := le.logStationGrid

	// Load ALL QSOs from the database, not just the current page.
	// The page-sized le.qsos slice would miss QSOs on other pages.
	var allQSOS []qso.QSO
	if le.db != nil {
		var err error
		allQSOS, err = store.ListAllQSOs(le.db)
		if err != nil {
			applog.Error("Wavelog: batch upload — cannot list QSOs", "error", err)
			return func() tea.Msg {
				return editorMsg{wlOK: false, err: fmt.Errorf("cannot read logbook: %w", err)}
			}
		}
	} else {
		// Fallback for tests without a real database.
		allQSOS = le.qsos
	}
	applog.Info("Wavelog: batch upload starting", "total_qsos", len(allQSOS))

	// Collect unsent QSOs, skip those with missing required fields.
	var unsent []qso.QSO
	var skipped int
	var firstSkipCall, firstSkipDate string
	for _, q := range allQSOS {
		if q.WavelogUploaded != "yes" {
			if q.Band == "" || q.Mode == "" || q.QSODate == "" {
				applog.Warn("Wavelog: skipping QSO with missing required field",
					"id", q.ID, "call", q.Call, "band", q.Band, "mode", q.Mode, "date", q.QSODate)
				if skipped == 0 {
					firstSkipCall = q.Call
					firstSkipDate = q.QSODate
				}
				skipped++
				continue
			}
			unsent = append(unsent, q)
		}
	}
	if skipped > 0 {
		applog.Warn("Wavelog: skipped QSOs with missing fields", "count", skipped)
		le.wlSkipped = skipped
		if skipped == 1 {
			le.wlSkipDetail = fmt.Sprintf("%s %s — missing band", firstSkipCall, firstSkipDate)
		} else {
			le.wlSkipDetail = fmt.Sprintf("%d QSOs skipped (e.g. %s %s — missing band)", skipped, firstSkipCall, firstSkipDate)
		}
	}
	if len(unsent) == 0 {
		applog.Info("Wavelog: batch upload — all already sent")
		return func() tea.Msg {
			return editorMsg{wlOK: true, wlCall: "all sent", err: nil}
		}
	}

	applog.Info("Wavelog: batch upload — unsent QSOs", "unsent", len(unsent), "skipped", skipped)

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
	wlCall := ""
	logOp := le.logStationOp
	logGrid := le.logStationGrid

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

	// Build batch ADIF.
	var adifStr string
	for _, q := range unsent {
		adifStr += q.ToADIF()
	}

	applog.InfoDetail("Wavelog: batch upload", fmt.Sprintf("count=%d", len(unsent)))

	return func() tea.Msg {
		result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
		if err != nil {
			// If the error contains duplicates, fall back to individual uploads
			// so that duplicates don't block new QSOs.
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "duplicate") {
				applog.Warn("Wavelog: batch had duplicates, falling back to individual uploads", "count", len(unsent))
				return le.uploadIndividual(unsent)()
			}
			applog.Error("Wavelog: batch upload failed", "count", len(unsent), "error", err)
			return editorMsg{wlOK: false, err: err, wlCall: fmt.Sprintf("%d QSOs", len(unsent))}
		}
		if result != nil && result.AllDuplicates {
			for _, q := range unsent {
				if dbErr := store.UpdateWavelogStatus(db, q.ID, "yes"); dbErr != nil {
					applog.Error("Wavelog: batch upload — failed to update status", "qso_id", q.ID, "error", dbErr)
				}
			}
			applog.InfoDetail("Wavelog: batch already present (duplicates)", fmt.Sprintf("count=%d", len(unsent)))
			return editorMsg{wlQSOID: unsent[0].ID, wlOK: true, wlCall: fmt.Sprintf("%d QSOs (already on Wavelog)", len(unsent))}
		}
		for _, q := range unsent {
			if dbErr := store.UpdateWavelogStatus(db, q.ID, "yes"); dbErr != nil {
				applog.Error("Wavelog: batch upload — failed to update status", "qso_id", q.ID, "error", dbErr)
			}
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

	return func() tea.Msg {
		okCount := 0
		dupCount := 0
		failCount := 0
		var lastErr error

		for _, q := range unsent {
			adifStr := q.ToADIF()
			ok, isDup, err := postQSO(url, key, sid, adifStr, q.ID, q.Call, db)
			if !ok {
				applog.Warn("Wavelog: individual upload failed", "qso_id", q.ID, "call", q.Call, "error", err)
				failCount++
				lastErr = err
				continue
			}
			if isDup {
				dupCount++
			} else {
				okCount++
			}
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
	if q.Band == "" || q.Mode == "" || q.QSODate == "" {
		applog.Warn("Wavelog: editor upload skipped — missing required field",
			"id", q.ID, "call", q.Call, "band", q.Band, "mode", q.Mode)
		return func() tea.Msg {
			return editorMsg{wlQSOID: q.ID, wlCall: q.Call, wlOK: false,
				err: fmt.Errorf("missing required field: band/mode/date")}
		}
	}
	adifStr := q.ToADIF()
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	qID := q.ID
	call := q.Call

	return func() tea.Msg {
		ok, isDup, err := postQSO(url, key, sid, adifStr, qID, call, le.db)
		return editorMsg{wlQSOID: qID, wlCall: call, wlOK: ok, wlDup: isDup, err: err}
	}
}
