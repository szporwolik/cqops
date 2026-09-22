package tui

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// remoteRefreshRequest is the identity captured when a remote-copy fetch
// starts. Every field is validated before the result may touch the database
// or the form: a late response from another logbook, editor instance, edit
// session, or row is discarded instead of overwriting local data.
type remoteRefreshRequest struct {
	gen     uint64  // editor generation that issued the fetch
	db      *sql.DB // database the fetch ran against
	localID int64   // local QSO id the fetch was issued for
	rev     uint64  // edit revision when the fetch started
	session uint64  // edit session when the fetch started — a reopened contact rejects older sessions
}

// fetchRemoteCopy loads the latest copy of a Wavelog QSO (by remote id) so
// the edit form can show what the server currently holds. The editor
// generation, database, local id, edit revision and edit session are
// captured here so a late result can be bound back to exactly the state
// that issued it.
func (le *LogbookEditor) fetchRemoteCopy(remoteID, localID int64) tea.Cmd {
	url, key := le.wlURL, le.wlKey
	rev := le.editRev
	gen := le.gen
	db := le.db
	session := le.editSession
	return func() tea.Msg {
		data, err := wavelog.GetQSO(url, key, remoteID)
		if err != nil {
			return editorMsg{wlFetchQSOID: localID, wlFetchErr: wavelog.FriendlyError(err).Error(),
				wlFetchRev: rev, wlFetchGen: gen, wlFetchDB: db, wlFetchSession: session}
		}
		return editorMsg{wlFetchQSOID: localID, wlFetchQSO: data,
			wlFetchRev: rev, wlFetchGen: gen, wlFetchDB: db, wlFetchSession: session}
	}
}

// ApplyRemoteRefresh merges the freshly fetched remote copy into the QSO being
// edited: the local row is updated, the form refilled and the list reload
// scheduled. Returns applied=false (a no-op) when the result is stale — the
// request identity no longer matches this editor (generation or database),
// the user moved to another row, typed since the fetch began, or the local
// row carries pending unsynced changes. In all those cases the operator's
// local state wins over the stale server copy.
func (le *LogbookEditor) ApplyRemoteRefresh(data *wavelog.QSOData, req remoteRefreshRequest) (bool, error) {
	// A response from a replaced editor or database must never touch this
	// editor's rows — logbook switching creates a fresh editor instance.
	if req.gen != 0 && req.gen != le.gen {
		applog.Warn("Wavelog: discarding remote refresh for a replaced editor",
			fmt.Sprintf("req_gen=%d editor_gen=%d remote_id=%d", req.gen, le.gen, data.ID))
		return false, nil
	}
	if req.db != nil && req.db != le.db {
		applog.Warn("Wavelog: discarding remote refresh for a replaced database",
			fmt.Sprintf("remote_id=%d", data.ID))
		return false, nil
	}
	// Reopening the same contact starts a new edit session — a refresh from
	// an earlier session must never apply, even when generation, database,
	// row and revision all coincide.
	if req.session != 0 && req.session != le.editSession {
		applog.InfoDetail("Wavelog: ignored remote refresh from a previous edit session",
			fmt.Sprintf("req_session=%d current_session=%d local_id=%d remote_id=%d",
				req.session, le.editSession, req.localID, data.ID))
		return false, nil
	}
	if le.editing == nil || le.mode != edModeEdit || le.editing.WavelogID != data.ID {
		return false, nil // user moved on — ignore the late result
	}
	if req.localID != 0 && req.localID != le.editing.ID {
		// Another contact is being edited now (even one with the same
		// remote id) — the fetched copy belongs to a different local row.
		applog.InfoDetail("Wavelog: ignored remote refresh for another row",
			fmt.Sprintf("req_local_id=%d editing_id=%d remote_id=%d", req.localID, le.editing.ID, data.ID))
		return false, nil
	}
	if req.rev != le.editRev {
		applog.InfoDetail("Wavelog: ignored stale remote refresh",
			fmt.Sprintf("fetched_rev=%d current_rev=%d local_id=%d", req.rev, le.editRev, le.editing.ID))
		return false, nil // operator typed since the fetch began
	}
	local, err := store.GetQSOByID(le.db, le.editing.ID)
	if err != nil {
		return false, err
	}
	if local.WavelogDirty {
		applog.InfoDetail("Wavelog: ignored remote refresh for pending local changes",
			fmt.Sprintf("local_id=%d remote_id=%d", le.editing.ID, data.ID))
		return false, nil // local changes not yet synced — never overwrite them
	}
	merged := mergeRemoteQSO(local, data)
	if merged.GridSquare != "" && le.logStationGrid != "" {
		merged.Distance = gridDistanceKm(le.logStationGrid, merged.GridSquare)
		merged.Bearing = gridBearingDeg(le.logStationGrid, merged.GridSquare)
	}
	if err := store.UpdateQSO(le.db, merged); err != nil {
		return false, err
	}
	le.editing = merged
	le.fillEditForm(merged)
	le.needsReload = true
	applog.InfoDetail("Wavelog: refreshed QSO from server",
		fmt.Sprintf("local_id=%d remote_id=%d call=%s", merged.ID, data.ID, merged.Call))
	return true, nil
}

// handleQSOSyncCompletion finishes a remote PATCH on the owner loop: it
// releases the per-contact serialization and, when a newer save was queued
// while the PATCH was on the wire, dispatches the follow-up carrying the
// NEWEST revision from the database. Remote updates for one contact are
// therefore applied in save order — an older stalled PATCH can never
// complete after a newer one and overwrite the server with stale values.
//
// The whole chain is bound to the immutable originating context carried by
// the completion (database, remote endpoint, generation, coordinator): the
// follow-up reads and writes ONLY the originating database and PATCHes ONLY
// the originating endpoint. The coordination is keyed by persistent
// logbook/contact identity and remote source on the model-owned coordinator
// — NOT by the disposable editor — so reopening the editor (F8) while a
// PATCH is on the wire preserves the queue and the queued revision is still
// pushed. The context's database lease covers the entire queued operation
// including the interval between workers, and is released here when the
// chain is fully drained.
// Runs globally, even when the editor screen is not visible.
func (m *Model) handleQSOSyncCompletion(em editorMsg) tea.Cmd {
	ctx := em.syncCtx
	if ctx == nil || ctx.coord == nil {
		return nil
	}
	coord := ctx.coord
	delete(coord.inFlight, ctx.key)
	if !coord.queued[ctx.key] {
		// The chain is fully drained — the originating row was
		// acknowledged (or reported) and no further PATCHes are queued.
		// Release the chain database lease.
		ctx.release()
		if ctx.batch != nil {
			m.completeSyncRetryBatch(ctx.batch, em)
		}
		return nil
	}
	delete(coord.queued, ctx.key)
	// Follow-up: re-insert the SAME context as in-flight. Its lease stays
	// held across the worker boundary, keeping the originating database
	// open even if the logbook was switched or the editor recreated
	// meanwhile.
	coord.inFlight[ctx.key] = ctx
	return m.patchLatestRevision(ctx)
}

// patchLatestRevision reads the row fresh from the ORIGINATING database and
// PATCHes its current revision to the ORIGINATING endpoint — the queued save
// stored its newer revision there, so this carries exactly the newest edit.
// Used by both serialized save follow-ups and initial-upload reconciliation.
func (m *Model) patchLatestRevision(ctx *contactSyncContext) tea.Cmd {
	id := ctx.key.localID
	return func() tea.Msg {
		res := syncDirtyRow(ctx.db, ctx.url, ctx.apiKey, id)
		res.gen = ctx.gen
		res.wlSyncFollowUp = true
		res.syncCtx = ctx
		return res
	}
}

// syncDirtyRow pushes a row's CURRENT revision to Wavelog and acknowledges
// the pending flag for exactly that revision. It is the shared per-contact
// sync unit used by serialized save follow-ups, initial-upload
// reconciliation, and the bulk pending-sync retry.
func syncDirtyRow(db *sql.DB, url, key string, id int64) editorMsg {
	row, err := store.GetQSOByID(db, id)
	res := editorMsg{saved: id}
	if err != nil {
		applog.Warn("Wavelog: sync cannot read QSO", "qso_id", id, "error", err)
		res.wlSyncErr = err.Error()
		return res
	}
	res.saveCall = row.Call
	res.saveDate = formatDate(row.QSODate)
	if row.WavelogID <= 0 {
		// The remote copy vanished while the edit was queued — the row
		// is honest again (id already cleared locally).
		res.wlSyncGone = true
		return res
	}
	syncErr := wavelog.UpdateQSO(url, key, row.WavelogID, buildUpdateInput(row))
	if syncErr != nil {
		if apiErr, ok := syncErr.(*wavelog.APIError); ok && apiErr.Code == "not_found" {
			if serr := store.SetWavelogID(db, id, 0); serr == nil {
				res.wlSyncGone = true
				if derr := store.SetWavelogDirty(db, id, false); derr != nil {
					applog.Warn("Wavelog: failed to clear pending sync", "qso_id", id, "error", derr)
				}
			} else {
				res.wlSyncErr = wavelog.FriendlyError(syncErr).Error()
			}
		} else {
			res.wlSyncErr = wavelog.FriendlyError(syncErr).Error()
		}
		return res
	}
	res.wlSyncOK = true
	if derr := store.ClearWavelogDirtyIfRevision(db, id, row.WavelogDirtyRev); derr != nil {
		applog.Warn("Wavelog: failed to clear pending sync", "qso_id", id, "error", derr)
		// The remote copy synced, but the local acknowledgement could
		// not be persisted — report an incomplete synchronization so
		// the row stays pending instead of claiming full success.
		res.wlSyncOK = false
		res.wlSyncIncomplete = true
	}
	return res
}

// queuePendingSyncPatches consumes a bulk pending-sync retry list result:
// EVERY retry PATCH goes through the same per-contact coordinator as save
// chains (keyed by logbook/contact/endpoint), so a retry can never run in
// parallel with a form save or overwrite a newer edit — contacts whose save
// chain is already in flight are covered by it and skipped. Each chain is
// bound to a batch tracker whose summary toast fires when the last chain
// drains.
func (m *Model) queuePendingSyncPatches(em editorMsg) tea.Cmd {
	logbook := em.lbID
	if logbook == "" {
		logbook = m.App.LogbookName
	}
	if m.sync == nil {
		m.sync = &contactSyncCoord{}
	}
	if m.sync.inFlight == nil {
		m.sync.inFlight = make(map[contactSyncKey]*contactSyncContext)
	}
	if m.sync.queued == nil {
		m.sync.queued = make(map[contactSyncKey]bool)
	}
	release := em.wlRetryRelease
	if len(em.wlRetryIDs) == 0 {
		// The list was empty — everything pending was already synchronized
		// before the read. The transferred lease must still be consumed.
		if release != nil {
			release()
		}
		m.toasts.Success("Wavelog: no pending changes")
		return nil
	}
	batch := &syncRetryBatch{remaining: make(map[int64]bool), lbID: logbook}
	var cmds []tea.Cmd
	for i, id := range em.wlRetryIDs {
		syncKey := contactSyncKey{logbook: logbook, localID: id, url: em.wlRetryURL}
		if m.sync.inFlight[syncKey] != nil {
			// A save chain is already pushing this contact — its follow-up
			// carries the newest revision; the retry is covered.
			continue
		}
		var l func()
		if i == 0 {
			l = release
			release = nil
		}
		batch.remaining[id] = true
		cmds = append(cmds, m.queueContactReconcile(em.wlRetryDB, em.wlRetryURL, em.wlRetryKey, logbook, id, l, batch))
	}
	if release != nil {
		release()
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// completeSyncRetryBatch counts a drained retry chain and reports the batch
// summary when the last contact finished. A PATCH the server accepted but
// whose pending flag could not be acknowledged locally counts as unconfirmed
// — the contact stays dirty and retryable, and the summary must never claim
// success ("no pending changes") for it.
func (m *Model) completeSyncRetryBatch(b *syncRetryBatch, em editorMsg) {
	if b == nil || !b.remaining[em.saved] {
		return
	}
	delete(b.remaining, em.saved)
	switch {
	case em.wlSyncOK || em.wlSyncGone:
		b.synced++
	case em.wlSyncErr != "":
		b.failed++
	case em.wlSyncIncomplete:
		b.incomplete++
	}
	if len(b.remaining) > 0 {
		return
	}
	switch {
	case b.failed > 0 && b.incomplete > 0:
		m.toasts.Warn(fmt.Sprintf("Wavelog: pending sync — %d synced, %d failed, %d unconfirmed", b.synced, b.failed, b.incomplete))
	case b.failed > 0:
		m.toasts.Warn(fmt.Sprintf("Wavelog: pending sync — %d synced, %d failed", b.synced, b.failed))
	case b.incomplete > 0:
		m.toasts.Warn(fmt.Sprintf("Wavelog: pending sync — %d synced, %d unconfirmed", b.synced, b.incomplete))
	case b.synced > 0:
		m.toasts.Success(fmt.Sprintf("Wavelog: pending sync — %d contacts synced", b.synced))
	default:
		m.toasts.Success("Wavelog: no pending changes")
	}
	m.needRefresh = true
}

// handleEditorUploadCompletion consumes an editor upload completion
// GLOBALLY: the follow-up chains — the reconciliation PATCH (the row changed
// while the upload was on the wire) and the id-attach retry (the server
// accepted but the id write failed) — are queued against the ORIGINATING
// logbook/database/endpoint carried by the message, independent of the
// visible screen and editor generation. Those gate only UI effects in
// handleLogbookEditorUpdate; a completion that arrives after the operator
// left the editor screen or after the editor was recreated must still
// schedule its PATCH.
//
// The transferred database lease is consumed here exactly once, even when
// screen routing later drops the message: it is handed to the follow-up
// chain (the first chain of a batch gets it; later ones acquire their own
// now that the database is guaranteed open) or released immediately when
// no follow-up is needed.
func (m *Model) handleEditorUploadCompletion(em editorMsg) tea.Cmd {
	logbook := em.lbID
	if logbook == "" {
		logbook = m.App.LogbookName
	}
	release := em.wlUpRelease
	var cmds []tea.Cmd
	if em.wlUpChanged && em.wlUpDB != nil {
		cmds = append(cmds, m.queueContactReconcile(em.wlUpDB, em.wlUpURL, em.wlUpKey, logbook, em.wlQSOID, release, nil))
		release = nil
	}
	if em.wlUpUnresolved && em.wlUpDB != nil {
		cmds = append(cmds, m.retryIDAttachCmd(em.wlUpDB, em.wlUpURL, em.wlUpKey, em.wlUpSID, logbook, em.wlUpSnap, em.wlUpRev, release))
		release = nil
	}
	if len(em.wlReconcileIDs) > 0 && em.wlUpDB != nil {
		for i, id := range em.wlReconcileIDs {
			var l func()
			if i == 0 {
				l = release
				release = nil
			}
			cmds = append(cmds, m.queueContactReconcile(em.wlUpDB, em.wlUpURL, em.wlUpKey, logbook, id, l, nil))
		}
	}
	if release != nil {
		release()
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

// queueContactReconcile queues a PATCH of the row's latest revision after an
// initial upload found the row changed while it was on the wire: the remote
// copy was created from an older snapshot, so the current revision must be
// pushed. It uses the model-owned per-contact coordinator keyed by the
// ORIGINATING logbook/contact/endpoint — the same coordination as save
// chains — so it serializes with any in-flight PATCH and survives editor
// recreation. The row is already durably dirty; the PATCH clears it for its
// exact revision.
//
// release, when non-nil, is the database lease transferred from the upload
// worker whose completion triggered this reconciliation: the chain owns it
// and releases it when drained, so the originating database stays open
// across the whole upload → reconciliation chain with no unprotected
// interval. When the chain is queued behind an in-flight one (which holds
// its own lease) or the request is invalid, the transferred lease is
// released immediately.
//
// batch, when non-nil, binds this chain to a bulk pending-sync retry
// tracker: the chain's drained completion counts toward the batch summary.
func (m *Model) queueContactReconcile(db *sql.DB, url, key, logbook string, localID int64, release func(), batch *syncRetryBatch) tea.Cmd {
	if db == nil || url == "" || key == "" || localID == 0 {
		if release != nil {
			release()
		}
		return nil
	}
	if m.sync == nil {
		m.sync = &contactSyncCoord{}
	}
	if m.sync.inFlight == nil {
		m.sync.inFlight = make(map[contactSyncKey]*contactSyncContext)
	}
	if m.sync.queued == nil {
		m.sync.queued = make(map[contactSyncKey]bool)
	}
	syncKey := contactSyncKey{logbook: logbook, localID: localID, url: url}
	if m.sync.inFlight[syncKey] != nil {
		// A save chain is already running for this contact — its follow-up
		// will read the latest revision; nothing more to queue.
		m.sync.queued[syncKey] = true
		if release != nil {
			release()
		}
		return nil
	}
	chainRelease := release
	if chainRelease == nil {
		chainRelease = func() {}
		if m.App != nil {
			chainRelease = m.App.KeepDBAlive(db)
		}
	}
	ctx := &contactSyncContext{
		key:     syncKey,
		coord:   m.sync,
		db:      db,
		url:     url,
		apiKey:  key,
		gen:     0,
		release: chainRelease,
		batch:   batch,
	}
	m.sync.inFlight[syncKey] = ctx
	return m.patchLatestRevision(ctx)
}

// mergeRemoteQSO overlays the server-side fields onto a local QSO. Local-only
// data (source, contest fields, exchange fields, created_at, wavelog_id) is
// preserved. Every field this function can overwrite is part of the
// field-level synchronization contract and MUST be sent by buildUpdateInput
// — otherwise a successful PATCH would clear the dirty flag while this
// refresh silently restores the server's older value.
func mergeRemoteQSO(local *qso.QSO, d *wavelog.QSOData) *qso.QSO {
	merged := *local
	if call := strings.TrimSpace(d.Call); call != "" {
		merged.Call = call
	}
	if band := strings.ToUpper(strings.TrimSpace(d.Band)); band != "" {
		merged.Band = band
	}
	mode, submode := qso.NormalizeMode(d.Mode, d.Submode)
	merged.Mode = mode
	merged.Submode = submode
	if len(d.QSODate) >= 10 {
		merged.QSODate = strings.ReplaceAll(d.QSODate[:10], "-", "") // ADIF YYYYMMDD
	}
	if len(d.QSODate) >= 19 {
		merged.TimeOn = strings.ReplaceAll(d.QSODate[11:19], ":", "") // HHMMSS
	}
	merged.RSTSent = d.RSTSent
	merged.RSTRcvd = d.RSTRcvd
	merged.GridSquare = strings.TrimSpace(d.Gridsquare)
	merged.Name = d.Name
	merged.QTH = d.QTH
	merged.Comment = d.Comment
	merged.Notes = d.Notes
	merged.TXPower = d.TXPower
	merged.SOTARef = d.SOTARef
	merged.POTARef = d.POTARef
	merged.WWFFRef = d.WWFFRef
	merged.IOTA = d.IOTA
	merged.SIG = d.SIG
	merged.SIGInfo = d.SIGInfo
	// Zones and frequencies need presence awareness: a remote CLEAR (JSON
	// null) must apply — a guarded "only when positive/nonempty" read would
	// keep the stale local value, which the next unrelated save would send
	// back and silently undo the remote clear. An ABSENT field leaves the
	// local value untouched. Literal-constructed QSOData (present == nil,
	// e.g. tests) keeps the value-based fallback.
	if d.HasPresence() {
		if d.Has("cqz") {
			merged.CQZone = ""
			if d.CQZ > 0 {
				merged.CQZone = strconv.Itoa(d.CQZ)
			}
		}
		if d.Has("ituz") {
			merged.ITUZone = ""
			if d.ITUZ > 0 {
				merged.ITUZone = strconv.Itoa(d.ITUZ)
			}
		}
		if d.Has("freq") {
			merged.Freq = 0
			if d.Freq != "" {
				if hz, err := strconv.ParseFloat(d.Freq, 64); err == nil {
					merged.Freq = hz / 1e6
				}
			}
		}
		if d.Has("freq_rx") {
			merged.FreqRx = 0
			if d.FreqRx != "" {
				if hz, err := strconv.ParseFloat(d.FreqRx, 64); err == nil {
					merged.FreqRx = hz / 1e6
				}
			}
		}
	} else {
		if d.CQZ > 0 {
			merged.CQZone = strconv.Itoa(d.CQZ)
		}
		if d.ITUZ > 0 {
			merged.ITUZone = strconv.Itoa(d.ITUZ)
		}
		if d.Freq != "" {
			if hz, err := strconv.ParseFloat(d.Freq, 64); err == nil {
				merged.Freq = hz / 1e6
			}
		}
		if d.FreqRx != "" {
			if hz, err := strconv.ParseFloat(d.FreqRx, 64); err == nil {
				merged.FreqRx = hz / 1e6
			}
		}
	}
	return &merged
}

// strptr returns a pointer to s, so an empty string can be distinguished
// from an untouched field when building a Wavelog PATCH.
func strptr(s string) *string { return &s }

// buildUpdateInput maps a local QSO to the v2 PATCH fields. v2 wants the date
// as YYYY-MM-DD plus time_on as HH:MM:SS (sent together), frequencies as
// string-encoded Hz, and canonical mode names.
//
// Field-level synchronization contract: every field the edit form can change
// AND the remote refresh can overwrite (mergeRemoteQSO) must be sent here.
// A field omitted from the PATCH would be reported as synced (dirty cleared)
// and then silently replaced by the remote refresh's older value — the
// exact zone loss this function must prevent. Every clearable field is
// always present: the edit form backs each one, so an empty value means the
// operator cleared it and the remote copy must be cleared too (the wavelog
// layer sends JSON null for that).
func buildUpdateInput(q *qso.QSO) wavelog.UpdateQSOInput {
	freq := int64(math.Round(q.Freq * 1e6))
	freqRx := int64(math.Round(q.FreqRx * 1e6))
	in := wavelog.UpdateQSOInput{
		Call:       q.Call,
		Band:       q.Band,
		Mode:       q.Mode,
		QSODate:    adifDateToISO(q.QSODate),
		TimeOn:     adifTimeToHMS(q.TimeOn),
		RSTSent:    strptr(q.RSTSent),
		RSTRcvd:    strptr(q.RSTRcvd),
		Gridsquare: strptr(q.GridSquare),
		Name:       strptr(q.Name),
		QTH:        strptr(q.QTH),
		Comment:    strptr(q.Comment),
		Notes:      strptr(q.Notes),
		TXPower:    strptr(q.TXPower),
		SOTARef:    strptr(q.SOTARef),
		POTARef:    strptr(q.POTARef),
		WWFFRef:    strptr(q.WWFFRef),
		IOTA:       strptr(q.IOTA),
		SIG:        strptr(q.SIG),
		SIGInfo:    strptr(q.SIGInfo),
		CQZone:     zonePtr(q.CQZone),
		ITUZone:    zonePtr(q.ITUZone),
		FreqHz:     &freq,
		FreqRxHz:   &freqRx,
	}
	if strings.EqualFold(q.Mode, "MFSK") && q.Submode != "" {
		in.Mode = q.Submode // canonical: MFSK+FT8 → FT8, same as buildCreateQSOInput
	}
	return in
}

// zonePtr converts an editable zone field to a nullable int for the PATCH:
// empty or invalid → 0 (the wavelog layer sends null, clearing the remote
// value), otherwise the parsed number.
func zonePtr(s string) *int {
	v := 0
	if d := qso.StripNonDigits(s); d != "" {
		if n, err := strconv.Atoi(d); err == nil && n > 0 {
			v = n
		}
	}
	return &v
}

// adifTimeToHMS converts a local ADIF time (HHMM or HHMMSS) to the HH:MM:SS
// form required by the v2 API.
func adifTimeToHMS(t string) string {
	digits := qso.StripNonDigits(t)
	for len(digits) < 6 {
		digits += "0"
	}
	if len(digits) > 6 {
		digits = digits[:6]
	}
	return digits[0:2] + ":" + digits[2:4] + ":" + digits[4:6]
}
