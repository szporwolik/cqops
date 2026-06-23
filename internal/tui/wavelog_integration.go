package tui

import (
	"database/sql"
	"fmt"
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
// and fetches station profile info.
func (m *Model) checkWavelogCmd() tea.Cmd {
	wl := m.App.Logbook.Wavelog
	url := wl.URL
	key := wl.APIKey
	stationID := wl.StationProfileID
	return func() tea.Msg {
		err := wavelog.TestConnection(url, key)
		online := err == nil && stationID != ""
		if err == nil && stationID != "" {
			stations, ferr := wavelog.FetchStations(url, key)
			if ferr == nil {
				for _, s := range stations {
					if s.ID == stationID {
						name := fmt.Sprintf("%s / %s", s.Gridsquare, s.Callsign)
						label := s.Name
						applog.InfoDetail("Wavelog: station info updated", fmt.Sprintf("id=%s grid=%s call=%s label=%s", s.ID, s.Gridsquare, s.Callsign, s.Name))
						return wlStatusMsg{online: true, stationName: name, stationLabel: label}
					}
				}
			}
		}
		return wlStatusMsg{online: online}
	}
}

type wlStatusMsg struct {
	online       bool
	stationName  string
	stationLabel string
}

// maybeUploadToWavelog returns a tea.Cmd that sends a QSO to Wavelog.
func (m *Model) maybeUploadToWavelog(qs *qso.QSO) tea.Cmd {
	return m.uploadADIFToWavelog(qs.ToADIF(), qs.ID, qs.Call)
}

// uploadADIFToWavelog returns a tea.Cmd that uploads an ADIF record to Wavelog.
func (m *Model) uploadADIFToWavelog(adifStr string, qID int64, call string) tea.Cmd {
	wl := m.App.Logbook.Wavelog
	if wl == nil || !wl.Enabled || !m.inetOnline || wl.StationProfileID == "" {
		return nil
	}
	url := wl.URL
	key := wl.APIKey
	stationID := wl.StationProfileID

	return func() tea.Msg {
		ok, isDup, err := postQSO(url, key, stationID, adifStr, qID, call, m.App.DB)
		return wlUploadResultMsg{qID: qID, call: call, ok: ok, isDup: isDup, err: err}
	}
}

// postQSO sends ADIF to Wavelog and updates the local QSO status.
// This is the single canonical upload path — all callers (QSO form auto-upload,
// logbook editor single/batch upload) use this function.
// Returns ok=true on success or duplicate, ok=false on failure.
// isDup is true when the QSO was already present on Wavelog.
func postQSO(url, key, sid, adifStr string, qID int64, call string, db *sql.DB) (ok bool, isDup bool, err error) {
	applog.InfoDetail("Wavelog: uploading QSO", fmt.Sprintf("qso_id=%d call=%s", qID, call))
	result, err := wavelog.PostQSOWithResult(url, key, sid, adifStr)
	if err != nil {
		// If Wavelog rejected the QSO but the error indicates it's a duplicate
		// (e.g. another app pushed the same QSO), treat it as success.
		errStr := strings.ToLower(err.Error())
		if strings.Contains(errStr, "duplicate") {
			applog.InfoDetail("Wavelog: QSO already present (duplicate via error)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
			if dbErr := store.UpdateWavelogStatus(db, qID, "yes"); dbErr != nil {
				applog.Error("Wavelog: failed to update local status", "qso_id", qID, "error", dbErr)
			}
			return true, true, nil
		}
		applog.Error("Wavelog: QSO upload failed", "qso_id", qID, "call", call, "error", err)
		if dbErr := store.UpdateWavelogStatus(db, qID, "no"); dbErr != nil {
			applog.Error("Wavelog: failed to update local status", "qso_id", qID, "error", dbErr)
		}
		return false, false, err
	}
	if dbErr := store.UpdateWavelogStatus(db, qID, "yes"); dbErr != nil {
		applog.Error("Wavelog: failed to update local status", "qso_id", qID, "error", dbErr)
	}
	if result != nil && result.AllDuplicates {
		applog.InfoDetail("Wavelog: QSO already present (duplicate)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		return true, true, nil
	}
	applog.InfoDetail("Wavelog: QSO uploaded OK", fmt.Sprintf("qso_id=%d call=%s", qID, call))
	return true, false, nil
}

type wlUploadResultMsg struct {
	qID   int64
	call  string
	ok    bool
	isDup bool
	err   error
}

// wsjtxEnrichDoneMsg signals that WSJT-X QRZ enrichment has completed
// for an auto-logged QSO. The handler triggers a Recent QSOs refresh so
// the name/QTH/country fields appear immediately.
type wsjtxEnrichDoneMsg struct{}
