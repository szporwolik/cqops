package tui

import (
	"database/sql"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// reconcileThreshold is the minimum unsent count that triggers the remote
// reconciliation pass before a batch upload. Small batches rely on the
// server's own duplicate detection instead of the extra list fetch.
const reconcileThreshold = 25

// uploadPrepMsg carries the prepared unsent QSO set from the background
// upload-preparation worker. The worker only reads the database and returns
// data — the editor state is updated on the update loop.
//
// gen and db bind the message to the editor instance and database the
// operation was started on: a result delivered to a replacement editor
// (e.g. after a logbook switch) must be discarded, never applied against
// another logbook's rows or Wavelog configuration.
type uploadPrepMsg struct {
	gen           uint64
	db            *sql.DB
	unsent        []qso.QSO
	skipped       int
	firstSkipCall string
	firstSkipDate string
	err           error
}

func (le *LogbookEditor) doBatchUpload() tea.Cmd {
	// Capture everything the background worker needs — it must not read the
	// editor from the command goroutine. The generation and database bind
	// the result to this editor instance.
	db := le.db
	release := le.dbLease(db)
	gen := le.gen
	qsos := le.qsos

	applog.Info("Wavelog: batch upload starting")
	return func() tea.Msg {
		defer release()
		if db != nil {
			total, err := store.CountUnsentQSOs(db)
			if err != nil {
				applog.Error("Wavelog: batch upload — cannot count unsent QSOs", "error", err)
				return uploadPrepMsg{err: fmt.Errorf("cannot read logbook: %w", err), gen: gen, db: db}
			}
			applog.Info("Wavelog: batch upload — unsent rows in log", "unsent", total)

			// Fetch only the eligible (never-uploaded) rows — a large
			// historical logbook stays cheap to prepare.
			rows, err := store.ListUnsentQSOs(db)
			if err != nil {
				applog.Error("Wavelog: batch upload — cannot list unsent QSOs", "error", err)
				return uploadPrepMsg{err: fmt.Errorf("cannot read logbook: %w", err), gen: gen, db: db}
			}
			msg := buildUploadPrep(rows)
			msg.gen = gen
			msg.db = db
			return msg
		}

		// No database — filter the in-memory list (tests).
		var all []qso.QSO
		for _, q := range qsos {
			if q.WavelogID == 0 {
				all = append(all, q)
			}
		}
		msg := buildUploadPrep(all)
		msg.gen = gen
		msg.db = db
		return msg
	}
}

// buildUploadPrep splits the eligible rows into uploadable QSOs and those
// skipped for missing required fields.
func buildUploadPrep(eligible []qso.QSO) uploadPrepMsg {
	var unsent []qso.QSO
	var skipped int
	var firstSkipCall, firstSkipDate string
	for _, q := range eligible {
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
	return uploadPrepMsg{unsent: unsent, skipped: skipped, firstSkipCall: firstSkipCall, firstSkipDate: firstSkipDate}
}

// handleUploadPrep consumes the prepared unsent set on the update loop:
// reports skipped rows, shows the normalize dialog on station-field
// mismatches, or launches the actual upload. Results bound to a different
// editor generation or database (logbook switched while preparation ran)
// are discarded — they must never be uploaded against another logbook.
func (le *LogbookEditor) handleUploadPrep(msg uploadPrepMsg) (tea.Model, tea.Cmd) {
	if msg.gen != le.gen || msg.db != le.db {
		applog.Warn("Wavelog: discarding upload-prep result for a replaced editor",
			fmt.Sprintf("msg_gen=%d editor_gen=%d", msg.gen, le.gen))
		return le, nil
	}
	if msg.err != nil {
		return le, func() tea.Msg { return editorMsg{wlOK: false, err: msg.err} }
	}
	if msg.skipped > 0 {
		applog.Warn("Wavelog: skipped QSOs with missing fields", "count", msg.skipped)
		le.wlSkipped = msg.skipped
		if msg.skipped == 1 {
			le.wlSkipDetail = fmt.Sprintf("%s %s — missing band", msg.firstSkipCall, msg.firstSkipDate)
		} else {
			le.wlSkipDetail = fmt.Sprintf("%d QSOs skipped (e.g. %s %s — missing band)", msg.skipped, msg.firstSkipCall, msg.firstSkipDate)
		}
	}
	if len(msg.unsent) == 0 {
		applog.Info("Wavelog: batch upload — all already sent")
		return le, func() tea.Msg {
			return editorMsg{wlOK: true, wlCall: "all sent", err: nil}
		}
	}

	applog.Info("Wavelog: batch upload — unsent QSOs", "unsent", len(msg.unsent), "skipped", msg.skipped)

	mismatch, fields := le.detectUploadMismatches(msg.unsent)
	if len(mismatch) > 0 {
		le.mismatchQSOs = mismatch
		le.mismatchFields = fields
		le.mode = edModeConfirmNormalize
		return le, nil
	}
	return le, le.uploadBatch(msg.unsent)
}

// detectUploadMismatches compares the unsent set against the logbook station
// defaults and returns mismatching QSOs plus the field names involved.
func (le *LogbookEditor) detectUploadMismatches(unsent []qso.QSO) ([]qso.QSO, []string) {
	var mismatch []qso.QSO
	var fields []string
	hasCallMismatch := false
	hasOpMismatch := false
	hasGridMismatch := false
	for _, q := range unsent {
		callDiff := le.logStationCall != "" && q.StationCallsign != "" && !strings.EqualFold(q.StationCallsign, le.logStationCall)
		opDiff := le.logStationOp != "" && q.Operator != "" && !strings.EqualFold(q.Operator, le.logStationOp)
		gridDiff := le.logStationGrid != "" && q.MyGridSquare != "" && !strings.EqualFold(q.MyGridSquare, le.logStationGrid)
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
	return mismatch, fields
}

func (le *LogbookEditor) doNormalizeAndUpload() tea.Cmd {
	db := le.db
	release := le.dbLease(db)
	gen := le.gen
	mismatch := le.mismatchQSOs

	// Normalize only the fields the confirmation explicitly listed as
	// mismatching. Station callsigns are preserved unless a callsign
	// mismatch was detected and confirmed — empty means "keep stored".
	wlCall := ""
	logOp := ""
	logGrid := ""
	for _, f := range le.mismatchFields {
		switch f {
		case "callsign":
			wlCall = le.logStationCall
		case "operator":
			logOp = le.logStationOp
		case "grid":
			logGrid = le.logStationGrid
		}
	}

	// Build list of IDs to normalize
	var normIDs []int64
	for _, q := range mismatch {
		normIDs = append(normIDs, q.ID)
	}

	applog.InfoDetail("Wavelog: normalizing station fields", fmt.Sprintf("count=%d call=%s op=%s grid=%s", len(normIDs), wlCall, logOp, logGrid))

	return func() tea.Msg {
		defer release()
		if err := store.NormalizeStationFields(db, normIDs, wlCall, logOp, logGrid); err != nil {
			applog.Error("Wavelog: normalization failed", "error", err)
			return editorMsg{wlOK: false, err: fmt.Errorf("normalize: %w", err), gen: gen}
		}
		// The worker is limited to database work: it returns the changed
		// fields and affected ids as immutable result data. The owner loop
		// applies them to the in-memory list after validating the
		// generation — a worker must never mutate le.qsos, which the owner
		// loop reads concurrently (table rebuilds).
		return editorMsg{
			normalized: len(normIDs), gen: gen,
			normIDs: normIDs, normCall: wlCall, normOp: logOp, normGrid: logGrid,
		}
	}
}

func (le *LogbookEditor) uploadBatch(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	release := le.dbLease(db)
	gen := le.gen
	opSession := le.editSession

	const chunkSize = 50

	// Empty list — should not happen in production but callers may pass nil.
	if len(unsent) == 0 {
		return func() tea.Msg {
			return editorMsg{wlOK: false, err: fmt.Errorf("no QSOs to upload")}
		}
	}

	return func() tea.Msg {
		defer release()
		totalOK := 0
		totalDup := 0
		totalUnresolved := 0
		totalFail := 0
		totalRecon := 0
		var lastErr error

		// Migrated logs: rows carry no remote id even though they already
		// exist on Wavelog. Reconcile against the remote list first so those
		// are never re-uploaded — they just learn their id locally. Only
		// genuinely new QSOs reach the upload loop below.
		if len(unsent) > reconcileThreshold {
			if ids, rerr := wavelog.FetchAllQSOIDs(url, key, sid); rerr == nil {
				uploadable := make([]qso.QSO, 0, len(unsent))
				for _, q := range unsent {
					k := wavelog.MakeQSOIDKey(q.Call, q.Band, q.Mode, q.QSODate, q.TimeOn)
					if rid, ok := ids[k]; ok && rid > 0 {
						serr := store.SetWavelogID(db, q.ID, rid)
						if serr == nil {
							totalRecon++
							continue
						}
						applog.Error("Wavelog: failed to store reconciled id — uploading instead", "qso_id", q.ID, "error", serr)
					}
					uploadable = append(uploadable, q)
				}
				unsent = uploadable
				applog.InfoDetail("Wavelog: pre-upload reconciliation",
					fmt.Sprintf("matched=%d remaining=%d", totalRecon, len(unsent)))
			} else {
				applog.Warn("Wavelog: pre-upload reconciliation failed — uploading directly", "error", rerr)
			}
		}
		if len(unsent) == 0 {
			return editorMsg{wlOK: true, wlCall: fmt.Sprintf("%d QSOs (already on Wavelog)", totalRecon), gen: gen, opSession: opSession}
		}

		for start := 0; start < len(unsent); start += chunkSize {
			end := start + chunkSize
			if end > len(unsent) {
				end = len(unsent)
			}
			chunk := unsent[start:end]

			// Build chunk ADIF.
			var adifStr string
			for _, q := range chunk {
				adifStr += q.ToADIF()
			}

			applog.InfoDetail("Wavelog: batch upload chunk", fmt.Sprintf("count=%d total=%d/%d", len(chunk), end, len(unsent)))

			// Strip MY_GRIDSQUARE — Wavelog uses the station profile grid.
			result, err := wavelog.PostQSOWithResult(url, key, sid, stripMyGridsquare(adifStr))
			if err != nil {
				errStr := strings.ToLower(err.Error())
				if strings.Contains(errStr, "duplicate") {
					applog.Warn("Wavelog: chunk had duplicates, falling back to individual", "count", len(chunk))
					msg := le.uploadIndividual(chunk)()
					if em, ok := msg.(editorMsg); ok {
						totalOK += em.wlSentCount
						totalDup += em.wlDupCount
						totalUnresolved += em.wlUnresolvedCount
						totalFail += em.wlFailCount
						if em.err != nil {
							lastErr = em.err
						}
					}
					continue
				}
				applog.Error("Wavelog: chunk upload failed", "count", len(chunk), "error", err)
				totalFail += len(chunk)
				lastErr = err
				continue
			}
			// The remote accepted the chunk; how many of those remote ids
			// made it into the local database decides whether each QSO
			// counts as sent or as unresolved.
			stored := backfillBatchIDs(url, key, sid, db, chunk)
			if result != nil && result.AllDuplicates {
				totalDup += stored
			} else {
				totalOK += stored
			}
			totalUnresolved += len(chunk) - stored
		}

		// Build the summary with explicit tallies. A failed chunk leaves its
		// QSOs unsent locally, so a partial success must never be reported
		// as success for the entire input count.
		if totalOK+totalDup+totalUnresolved == 0 && lastErr != nil {
			return editorMsg{wlOK: false, err: lastErr, wlFailCount: totalFail,
				wlCall: fmt.Sprintf("%d QSOs", len(unsent)), gen: gen, opSession: opSession}
		}
		var parts []string
		if totalOK > 0 {
			parts = append(parts, fmt.Sprintf("%d sent", totalOK))
		}
		already := totalDup + totalRecon
		if already > 0 {
			parts = append(parts, fmt.Sprintf("%d already on Wavelog", already))
		}
		if totalUnresolved > 0 {
			parts = append(parts, fmt.Sprintf("%d accepted but remote id not stored", totalUnresolved))
		}
		if totalFail > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", totalFail))
		}
		applog.InfoDetail("Wavelog: chunked upload done",
			fmt.Sprintf("ok=%d dup=%d unresolved=%d fail=%d recon=%d total=%d",
				totalOK, totalDup, totalUnresolved, totalFail, totalRecon, len(unsent)))
		return editorMsg{wlQSOID: unsent[0].ID,
			wlOK:              true,
			wlCall:            strings.Join(parts, ", "),
			wlSentCount:       totalOK,
			wlDupCount:        totalDup,
			wlFailCount:       totalFail,
			wlUnresolvedCount: totalUnresolved,
			gen:               gen,
			opSession:         opSession,
		}
	}
}

// uploadIndividual sends each unsent QSO one at a time, silently handling
// duplicates. Used as fallback when a batch upload encounters mixed results.
func (le *LogbookEditor) uploadIndividual(unsent []qso.QSO) tea.Cmd {
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	db := le.db
	release := le.dbLease(db)
	gen := le.gen

	return func() tea.Msg {
		defer release()
		sentCount := 0
		dupCount := 0
		failCount := 0
		unresolved := 0
		var lastErr error

		for _, q := range unsent {
			ok, isDup, rid, err := postQSOSingle(url, key, sid, &q, db)
			if !ok {
				applog.Warn("Wavelog: individual upload failed", "qso_id", q.ID, "call", q.Call, "error", err)
				failCount++
				lastErr = err
				continue
			}
			// Remote acceptance and local remote-id persistence are
			// separate outcomes: without a persisted id the QSO still
			// looks unsent locally and is re-offered on the next upload.
			if rid == 0 {
				unresolved++
				continue
			}
			if isDup {
				dupCount++
			} else {
				sentCount++
			}
		}

		applog.InfoDetail("Wavelog: individual upload complete",
			fmt.Sprintf("sent=%d dup=%d unresolved=%d fail=%d", sentCount, dupCount, unresolved, failCount))

		if failCount > 0 && sentCount+dupCount+unresolved == 0 {
			return editorMsg{wlOK: false, err: lastErr, wlFailCount: failCount, wlCall: fmt.Sprintf("%d failed", failCount), gen: gen}
		}
		var parts []string
		if sentCount > 0 {
			parts = append(parts, fmt.Sprintf("%d sent", sentCount))
		}
		if dupCount > 0 {
			parts = append(parts, fmt.Sprintf("%d already present", dupCount))
		}
		if unresolved > 0 {
			parts = append(parts, fmt.Sprintf("%d accepted but remote id not stored", unresolved))
		}
		if failCount > 0 {
			parts = append(parts, fmt.Sprintf("%d failed", failCount))
		}
		return editorMsg{
			wlQSOID:           unsent[0].ID,
			wlOK:              true,
			wlCall:            strings.Join(parts, ", "),
			wlSentCount:       sentCount,
			wlDupCount:        dupCount,
			wlFailCount:       failCount,
			wlUnresolvedCount: unresolved,
			gen:               gen,
		}
	}
}

func (le *LogbookEditor) doUploadToWavelog() tea.Cmd {
	if le.wlURL == "" || le.wlKey == "" || le.wlStationID == "" {
		return func() tea.Msg {
			return editorMsg{wlOK: false, err: fmt.Errorf("wavelog not configured")}
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
	url, key, sid := le.wlURL, le.wlKey, le.wlStationID
	qID := q.ID
	call := q.Call
	gen := le.gen
	opSession := le.editSession
	db := le.db
	release := le.dbLease(db)

	return func() tea.Msg {
		defer release()
		ok, isDup, _, err := postQSOSingle(url, key, sid, q, db)
		return editorMsg{wlQSOID: qID, wlCall: call, wlOK: ok, wlDup: isDup, err: err, gen: gen, opSession: opSession}
	}
}

// backfillBatchIDs stores remote ids for a successfully uploaded chunk and
// returns how many chunk QSOs now have a persisted remote id. Bulk ADIF
// import summaries carry no ids, so the newest window of the JSON list is
// fetched and matched by dedupe fields. QSOs whose id could not be persisted
// count as unresolved: the remote accepted them, but locally they still look
// unsent and will be re-offered on the next upload.
func backfillBatchIDs(url, key, sid string, db *sql.DB, chunk []qso.QSO) int {
	ids, berr := wavelog.FindQSOIDs(url, key, sid, len(chunk)+25)
	if berr != nil {
		applog.Warn("Wavelog: batch id backfill failed", "error", berr)
		return 0
	}
	stored := 0
	for _, q := range chunk {
		k := wavelog.MakeQSOIDKey(q.Call, q.Band, q.Mode, q.QSODate, q.TimeOn)
		if rid, ok := ids[k]; ok {
			if serr := store.SetWavelogID(db, q.ID, rid); serr != nil {
				applog.Error("Wavelog: failed to store remote id", "qso_id", q.ID, "error", serr)
			} else {
				stored++
			}
		}
	}
	return stored
}
