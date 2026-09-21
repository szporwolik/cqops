package tui

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Wavelog status checks and QSO upload orchestration.
// These methods are pure orchestration — they call the wavelog package
// for network operations and handle the results.
// =============================================================================

// maybeCheckWavelog returns a tea.Cmd to check Wavelog connectivity
// at startup (tick 1), when the logbook is switched, and periodically
// with exponential-like backoff on failure: 3 quick retries at 1s, then every 60s.
func (m *Model) maybeCheckWavelog() tea.Cmd {
	if m.Offline || !m.inetOnline {
		m.lookup.wlOnline = false
		m.lookup.wlFailCount = 0
		return nil
	}
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || wl.StationProfileID == "" {
		m.lookup.wlOnline = false
		m.lookup.wlFailCount = 0
		return nil
	}
	// Check on startup or when forced (logbook switch).
	if m.tickCount != 1 && !m.lookup.wlForceCheck {
		// Retry with backoff when offline.
		if !m.lookup.wlOnline && m.lookup.wlFailCount > 0 {
			if time.Now().Before(m.lookup.wlNextRetry) {
				return nil
			}
			m.lookup.wlNextRetry = time.Now().Add(retryInterval(m.lookup.wlFailCount))
			return m.checkWavelogCmd()
		}
		return nil
	}
	m.lookup.wlForceCheck = false
	return m.checkWavelogCmd()
}

// retryInterval returns the delay before the next Wavelog retry:
// ≤3 failures: 1 second; >3 failures: 60 seconds.
func retryInterval(failCount int) time.Duration {
	if failCount <= 3 {
		return 1 * time.Second
	}
	return 60 * time.Second
}

// checkWavelogCmd returns a tea.Cmd that tests Wavelog server connectivity
// and fetches station profile info. Legacy v1 keys are detected up front and
// reported with the migration message — no network call needed.
func (m *Model) checkWavelogCmd() tea.Cmd {
	wl := m.App.Logbook.Wavelog
	url := wl.URL
	key := wl.APIKey
	stationID := wl.StationProfileID
	if !wavelog.IsV2Token(key) {
		applog.Warn("Wavelog: legacy v1 API key detected", "url", url)
		return func() tea.Msg {
			return wlStatusMsg{online: false, err: wavelog.V1KeyRequiredMsg}
		}
	}
	return func() tea.Msg {
		err := wavelog.TestConnection(url, key)
		online := err == nil && stationID != ""
		if err == nil && stationID != "" {
			stations, ferr := wavelog.FetchStations(url, key)
			if ferr == nil {
				sidInt, _ := wavelog.ParseStationID(stationID)
				for _, s := range stations {
					if s.ID == strconv.Itoa(sidInt) {
						name := fmt.Sprintf("%s / %s", s.Gridsquare, s.Callsign)
						label := s.Name
						applog.InfoDetail("Wavelog: station info updated", fmt.Sprintf("id=%s grid=%s call=%s label=%s", s.ID, s.Gridsquare, s.Callsign, s.Name))
						return wlStatusMsg{online: true, stationName: name, stationLabel: label}
					}
				}
			}
		}
		statusErr := ""
		if err != nil {
			statusErr = err.Error()
		}
		return wlStatusMsg{online: online, err: statusErr}
	}
}

type wlStatusMsg struct {
	online       bool
	stationName  string
	stationLabel string
	err          string // human-readable reason when offline
}

// maybeUploadToWavelog returns a tea.Cmd that sends a QSO to Wavelog.
func (m *Model) maybeUploadToWavelog(qs *qso.QSO) tea.Cmd {
	return m.uploadQSOToWavelog(qs)
}

// wlOpCtx is an immutable snapshot of the logbook database and Wavelog
// destination, captured when a background command is created. SwitchLogbook
// closes and replaces App.DB, so background commands must only touch this
// snapshot — never the live model fields.
type wlOpCtx struct {
	logbook   string  // logbook ID the QSO belongs to (routing of results)
	db        *sql.DB // captured handle, kept alive via App.KeepDBAlive
	url       string  // Wavelog destination
	key       string
	stationID string
}

// uploadQSOToWavelog returns a tea.Cmd that uploads a single QSO to Wavelog
// via the v2 single-JSON create, so the remote id can be stored locally.
func (m *Model) uploadQSOToWavelog(qs *qso.QSO) tea.Cmd {
	if m.App == nil || m.App.DB == nil {
		return nil
	}
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || !m.inetOnline || wl.StationProfileID == "" {
		return nil
	}

	// Capture the operation context up front: after this point the user may
	// switch logbooks, which closes and replaces App.DB — the command must
	// only touch the captured database and destination.
	ctx := wlOpCtx{
		logbook:   m.App.LogbookName,
		db:        m.App.DB,
		url:       wl.URL,
		key:       wl.APIKey,
		stationID: wl.StationProfileID,
	}
	release := m.App.KeepDBAlive(ctx.db)

	return func() tea.Msg {
		defer release()
		ok, isDup, remoteID, err := postQSOSingle(ctx.url, ctx.key, ctx.stationID, qs, ctx.db)
		return wlUploadResultMsg{qID: qs.ID, call: qs.Call, logbook: ctx.logbook, ok: ok, isDup: isDup, remoteID: remoteID, err: err}
	}
}

// postQSOSingle uploads one QSO via the v2 single-JSON create and stores the
// remote id locally — wavelog_id > 0 is the single source of truth for
// "uploaded". Falls back to the ADIF import path when the JSON create fails
// for any reason, preserving the previous behavior. On duplicates the remote
// id is backfilled via a callsign lookup so the row still counts as uploaded.
func postQSOSingle(url, key, sid string, qs *qso.QSO, db *sql.DB) (ok bool, isDup bool, remoteID int64, err error) {
	if sidInt, perr := wavelog.ParseStationID(sid); perr == nil {
		remoteID, dup, cerr := wavelog.CreateQSO(url, key, buildCreateQSOInput(sidInt, qs))
		if cerr == nil {
			if dup {
				applog.InfoDetail("Wavelog: QSO already present (JSON create)", fmt.Sprintf("qso_id=%d", qs.ID))
				return true, true, backfillRemoteID(url, key, sid, qs, db), nil
			}
			if remoteID > 0 {
				if derr := store.SetWavelogID(db, qs.ID, remoteID); derr != nil {
					applog.Error("Wavelog: failed to store remote id", "qso_id", qs.ID, "error", derr)
				} else {
					applog.InfoDetail("Wavelog: QSO created via v2", fmt.Sprintf("qso_id=%d remote_id=%d", qs.ID, remoteID))
				}
			} else {
				// Server returned no id (bulk-summary shape) — backfill it.
				remoteID = backfillRemoteID(url, key, sid, qs, db)
			}
			return true, false, remoteID, nil
		}
		// Auth errors (v1 key, revoked/expired token) fail identically on the
		// ADIF path — surface the friendly message directly instead of a
		// doomed fallback.
		if apiErr, ok := cerr.(*wavelog.APIError); ok {
			switch apiErr.Code {
			case "unauthorized", "invalid_token", "token_expired":
				return false, false, 0, wavelog.FriendlyError(cerr)
			}
		}
		applog.Warn("Wavelog: JSON create failed, falling back to ADIF", "qso_id", qs.ID, "error", cerr)
	}
	// Fallback: the existing ADIF import path.
	ok, isDup, err = postQSO(url, key, sid, qs.ToADIF(), qs.ID, qs.Call, db)
	if ok {
		remoteID = backfillRemoteID(url, key, sid, qs, db)
	}
	return ok, isDup, remoteID, err
}

// backfillRemoteID learns the remote id of an already-present QSO via the
// callsign-scoped JSON list, so wavelog_id reflects reality even when the
// create/import response carried no id (duplicates, bulk summaries).
func backfillRemoteID(url, key, sid string, qs *qso.QSO, db *sql.DB) int64 {
	mode := qs.Mode
	if strings.EqualFold(mode, "MFSK") && qs.Submode != "" {
		mode = qs.Submode // canonical: MFSK+FT8 → FT8, same as buildCreateQSOInput
	}
	rid, ferr := wavelog.FindQSOID(url, key, sid, qs.Call, adifDateToISO(qs.QSODate), qs.TimeOn, qs.Band, mode)
	if ferr != nil {
		applog.Warn("Wavelog: remote id backfill failed", "qso_id", qs.ID, "call", qs.Call, "error", ferr)
		return 0
	}
	if rid > 0 {
		if serr := store.SetWavelogID(db, qs.ID, rid); serr != nil {
			applog.Error("Wavelog: failed to store remote id", "qso_id", qs.ID, "error", serr)
		} else {
			applog.InfoDetail("Wavelog: remote id backfilled", fmt.Sprintf("qso_id=%d remote_id=%d", qs.ID, rid))
		}
	}
	return rid
}

// buildCreateQSOInput maps a local QSO to the v2 JSON-create fields.
// v2 dates are ISO (YYYY-MM-DD) and frequencies are in Hz.
func buildCreateQSOInput(sidInt int, qs *qso.QSO) wavelog.CreateQSOInput {
	in := wavelog.CreateQSOInput{
		StationProfileID: sidInt,
		Call:             qs.Call,
		Band:             qs.Band,
		Mode:             qs.Mode,
		QSODate:          adifDateToISO(qs.QSODate),
		TimeOn:           qs.TimeOn,
		TimeOff:          qs.TimeOff,
		RSTSent:          qs.RSTSent,
		RSTRcvd:          qs.RSTRcvd,
		Gridsquare:       qs.GridSquare,
		Name:             qs.Name,
		QTH:              qs.QTH,
		Comment:          qs.Comment,
		Notes:            qs.Notes,
	}
	if qs.Freq > 0 {
		in.FreqHz = int64(math.Round(qs.Freq * 1e6))
	}
	if qs.FreqRx > 0 {
		in.FreqRxHz = int64(math.Round(qs.FreqRx * 1e6))
	}
	// ADIF codes digital modes as MFSK+submode; v2 wants the canonical mode.
	if strings.EqualFold(qs.Mode, "MFSK") && qs.Submode != "" {
		in.Mode = qs.Submode
	}
	return in
}

// adifDateToISO converts an ADIF date (YYYYMMDD) to ISO (YYYY-MM-DD).
func adifDateToISO(d string) string {
	if len(d) == 8 {
		if t, err := time.Parse("20060102", d); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return d
}

// postQSO sends ADIF to Wavelog and updates the local QSO status.
// This is the single canonical upload path — all callers (QSO form auto-upload,
// logbook editor single/batch upload) use this function.
// Returns ok=true on success or duplicate, ok=false on failure.
// isDup is true when the QSO was already present on Wavelog.
func postQSO(url, key, sid, adifStr string, qID int64, call string, db *sql.DB) (ok bool, isDup bool, err error) {
	// Strip MY_GRIDSQUARE from the ADIF before uploading. Wavelog always
	// uses the station profile's grid for distance calculation (see Api.php
	// qso function). Sending a different grid can cause QSO rejection.
	adifStr = stripMyGridsquare(adifStr)

	applog.InfoDetail("Wavelog: uploading QSO", fmt.Sprintf("qso_id=%d call=%s", qID, call))
	result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
	if err != nil {
		// If Wavelog rejected the QSO but the error indicates it's a duplicate
		// (e.g. another app pushed the same QSO), treat it as success and
		// learn the remote id from the server.
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate") {
			applog.InfoDetail("Wavelog: QSO already present (duplicate via error)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
			backfillQSOID(url, key, sid, qID, db)
			return true, true, nil
		}
		applog.Error("Wavelog: QSO upload failed", "qso_id", qID, "call", call, "error", err)
		return false, false, err
	}
	if result != nil && result.AllDuplicates {
		applog.InfoDetail("Wavelog: QSO already present (duplicate)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		backfillQSOID(url, key, sid, qID, db)
		return true, true, nil
	}
	// Bulk import summaries carry no id — learn it from the server list.
	backfillQSOID(url, key, sid, qID, db)
	applog.InfoDetail("Wavelog: QSO uploaded OK", fmt.Sprintf("qso_id=%d call=%s", qID, call))
	return true, false, nil
}

// backfillQSOID loads a local QSO and stores its remote id, when the server
// can identify it. Failures are logged, never fatal — the next download
// backfills ids as a safety net.
func backfillQSOID(url, key, sid string, qID int64, db *sql.DB) {
	qs, lerr := store.GetQSOByID(db, qID)
	if lerr != nil || qs == nil {
		return
	}
	rid := backfillRemoteID(url, key, sid, qs, db)
	if rid > 0 {
		applog.InfoDetail("Wavelog: remote id stored", fmt.Sprintf("qso_id=%d remote_id=%d", qID, rid))
	}
}

type wlUploadResultMsg struct {
	qID      int64
	call     string
	logbook  string // logbook ID at command creation; handler drops foreign results
	ok       bool
	isDup    bool
	remoteID int64
	err      error
}

// stripMyGridsquare removes the MY_GRIDSQUARE field from an ADIF string.
// Wavelog always uses the station profile's grid for distance calculation
// (see Api.php qso function) — sending a different grid risks QSO rejection.
// The local SQLite DB retains the actual operating grid.
func stripMyGridsquare(adif string) string {
	// Match <MY_GRIDSQUARE:N>value where N is the field length.
	// Use a simple loop to strip all occurrences.
	for {
		start := 0
		// Find <MY_GRIDSQUARE or <my_gridsquare (case-insensitive)
		lower := strings.ToLower(adif)
		idx := -1
		for _, tag := range []string{"<my_gridsquare:", "<my_gridsquare "} {
			if i := strings.Index(lower[start:], tag); i >= 0 {
				idx = start + i
				break
			}
		}
		if idx < 0 {
			break
		}
		// Find the closing >
		end := strings.IndexByte(adif[idx:], '>')
		if end < 0 {
			break
		}
		end += idx
		// The value after > has N characters where N is the field length.
		// Parse the length from <MY_GRIDSQUARE:N>
		lenStart := strings.IndexByte(adif[idx:], ':') + idx + 1
		if lenStart <= idx {
			break
		}
		lenEnd := end
		if spaceIdx := strings.IndexAny(adif[lenStart:end], " >"); spaceIdx >= 0 {
			lenEnd = lenStart + spaceIdx
		}
		fieldLen := 0
		fmt.Sscanf(adif[lenStart:lenEnd], "%d", &fieldLen)
		// Remove from <MY_GRIDSQUARE...> through the N value chars.
		cutEnd := end + 1 + fieldLen
		if cutEnd > len(adif) {
			cutEnd = len(adif)
		}
		adif = adif[:idx] + adif[cutEnd:]
	}
	return adif
}

// wsjtxEnrichDoneMsg signals that WSJT-X QRZ enrichment has completed
// for an auto-logged QSO. The handler triggers a Recent QSOs refresh so
// the name/QTH/country fields appear immediately.
type wsjtxEnrichDoneMsg struct {
	logbook string // logbook ID at command creation; handler drops foreign results
}
