package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// =============================================================================
// Wavelog status checks and QSO upload orchestration.
// These methods are pure orchestration — they call the wavelog package
// for network operations and handle the results.
// =============================================================================

// maybeCheckWavelog returns a tea.Cmd to check Wavelog connectivity
// on the first tick and periodically thereafter.
func (m *Model) maybeCheckWavelog() tea.Cmd {
	if !m.App.Config.Wavelog.Enabled {
		m.wlOnline = false
		return nil
	}
	if m.tickCount != 1 && m.tickCount%healthCheckTicks != 0 {
		return nil
	}
	return m.checkWavelogCmd()
}

// checkWavelogCmd returns a tea.Cmd that tests Wavelog server connectivity
// and fetches station profile info.
func (m *Model) checkWavelogCmd() tea.Cmd {
	url := m.App.Config.Wavelog.URL
	key := m.App.Config.Wavelog.APIKey
	stationID := m.App.Config.Wavelog.StationProfileID
	return func() tea.Msg {
		err := wavelog.TestConnection(url, key)
		online := err == nil
		if online && stationID != "" {
			stations, ferr := wavelog.FetchStations(url, key)
			if ferr == nil {
				for _, s := range stations {
					if s.ID == stationID {
						name := fmt.Sprintf("%s / %s", s.Gridsquare, s.Callsign)
						label := s.Name
						applog.InfoDetail("Wavelog: station info updated", fmt.Sprintf("id=%s grid=%s call=%s label=%s", s.ID, s.Gridsquare, s.Callsign, s.Name))
						return wlStatusMsg{online: online, stationName: name, stationLabel: label}
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

// maybeUploadRawADIFToWavelog returns a tea.Cmd that sends raw ADIF (from WSJT-X) to Wavelog.
func (m *Model) maybeUploadRawADIFToWavelog(adifStr string, qID int64, call string) tea.Cmd {
	return m.uploadADIFToWavelog(adifStr, qID, call)
}

// uploadADIFToWavelog returns a tea.Cmd that uploads an ADIF record to Wavelog.
func (m *Model) uploadADIFToWavelog(adifStr string, qID int64, call string) tea.Cmd {
	if !m.App.Config.Wavelog.Enabled || !m.inetOnline || m.App.Config.Wavelog.StationProfileID == "" {
		return nil
	}
	url := m.App.Config.Wavelog.URL
	key := m.App.Config.Wavelog.APIKey
	stationID := m.App.Config.Wavelog.StationProfileID

	return func() tea.Msg {
		applog.InfoDetail("Wavelog: uploading QSO", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		result, err := wavelog.PostQSOWithResult(url, key, stationID, adifStr)
		if err != nil {
			applog.Error("Wavelog: QSO upload failed", "qso_id", qID, "call", call, "error", err)
			return wlUploadResultMsg{qID: qID, call: call, ok: false, err: err}
		}
		if result != nil && result.AllDuplicates {
			applog.InfoDetail("Wavelog: QSO already present (duplicate)", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		}
		applog.InfoDetail("Wavelog: QSO uploaded OK", fmt.Sprintf("qso_id=%d call=%s", qID, call))
		return wlUploadResultMsg{qID: qID, call: call, ok: true}
	}
}

type wlUploadResultMsg struct {
	qID  int64
	call string
	ok   bool
	err  error
}
