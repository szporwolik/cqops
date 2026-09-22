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
// Runs globally, even when the editor screen is not visible.
func (m *Model) handleQSOSyncCompletion(em editorMsg) tea.Cmd {
	le := m.ui.logbookEditor
	if le == nil {
		return nil
	}
	// A completion from a replaced editor (logbook switched) must not touch
	// the new editor's serialization state.
	if em.gen != 0 && em.gen != le.gen {
		return nil
	}
	delete(le.syncInFlight, em.saved)
	if !le.syncQueued[em.saved] {
		return nil
	}
	delete(le.syncQueued, em.saved)
	le.syncInFlight[em.saved] = true

	db := m.App.DB
	url, key := le.wlURL, le.wlKey
	gen := le.gen
	id := em.saved
	// The follow-up writes locally (ack persistence, gone handling), so it
	// leases the database for its whole lifetime: a logbook switch retires
	// the database instead of closing it under the worker.
	release := le.dbLease(db)
	return func() tea.Msg {
		defer release()
		// Read the row fresh — the queued save already stored its newer
		// revision locally, so this PATCH carries exactly the newest edit.
		row, err := store.GetQSOByID(db, id)
		res := editorMsg{saved: id, gen: gen, wlSyncFollowUp: true}
		if err != nil {
			applog.Warn("Wavelog: follow-up sync cannot read QSO", "qso_id", id, "error", err)
			res.wlSyncErr = err.Error()
			return res
		}
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
}

// mergeRemoteQSO overlays the server-side fields onto a local QSO. Local-only
// data (source, contest fields, created_at, wavelog_id) is preserved.
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
	return &merged
}

// strptr returns a pointer to s, so an empty string can be distinguished
// from an untouched field when building a Wavelog PATCH.
func strptr(s string) *string { return &s }

// buildUpdateInput maps a local QSO to the v2 PATCH fields. v2 wants the date
// as YYYY-MM-DD plus time_on as HH:MM:SS (sent together), frequencies as
// string-encoded Hz, and canonical mode names.
//
// Every clearable field is always present: the edit form backs each one, so
// an empty value means the operator cleared it and the remote copy must be
// cleared too (the wavelog layer sends JSON null for that).
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
		FreqHz:     &freq,
		FreqRxHz:   &freqRx,
	}
	if strings.EqualFold(q.Mode, "MFSK") && q.Submode != "" {
		in.Mode = q.Submode // canonical: MFSK+FT8 → FT8, same as buildCreateQSOInput
	}
	return in
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
