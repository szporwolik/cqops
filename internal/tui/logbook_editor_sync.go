package tui

import (
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

// fetchRemoteCopy loads the latest copy of a Wavelog QSO (by remote id) so
// the edit form can show what the server currently holds.
func (le *LogbookEditor) fetchRemoteCopy(remoteID, localID int64) tea.Cmd {
	url, key := le.wlURL, le.wlKey
	return func() tea.Msg {
		data, err := wavelog.GetQSO(url, key, remoteID)
		if err != nil {
			return editorMsg{wlFetchQSOID: localID, wlFetchErr: wavelog.FriendlyError(err).Error()}
		}
		return editorMsg{wlFetchQSOID: localID, wlFetchQSO: data}
	}
}

// ApplyRemoteRefresh merges the freshly fetched remote copy into the QSO being
// edited: the local row is updated, the form refilled and the list reload
// scheduled. No-op when the user has already left the edit form.
func (le *LogbookEditor) ApplyRemoteRefresh(data *wavelog.QSOData) error {
	if le.editing == nil || le.mode != edModeEdit || le.editing.WavelogID != data.ID {
		return nil // user moved on — ignore the late result
	}
	local, err := store.GetQSOByID(le.db, le.editing.ID)
	if err != nil {
		return err
	}
	merged := mergeRemoteQSO(local, data)
	if merged.GridSquare != "" && le.logStationGrid != "" {
		merged.Distance = gridDistanceKm(le.logStationGrid, merged.GridSquare)
		merged.Bearing = gridBearingDeg(le.logStationGrid, merged.GridSquare)
	}
	if err := store.UpdateQSO(le.db, merged); err != nil {
		return err
	}
	le.editing = merged
	le.fillEditForm(merged)
	le.needsReload = true
	applog.InfoDetail("Wavelog: refreshed QSO from server",
		fmt.Sprintf("local_id=%d remote_id=%d call=%s", merged.ID, data.ID, merged.Call))
	return nil
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

// buildUpdateInput maps a local QSO to the v2 PATCH fields. v2 wants the date
// as YYYY-MM-DD plus time_on as HH:MM:SS (sent together), frequencies as
// string-encoded Hz, and canonical mode names.
func buildUpdateInput(q *qso.QSO) wavelog.UpdateQSOInput {
	in := wavelog.UpdateQSOInput{
		Call:       q.Call,
		Band:       q.Band,
		Mode:       q.Mode,
		QSODate:    adifDateToISO(q.QSODate),
		TimeOn:     adifTimeToHMS(q.TimeOn),
		RSTSent:    q.RSTSent,
		RSTRcvd:    q.RSTRcvd,
		Gridsquare: q.GridSquare,
		Name:       q.Name,
		QTH:        q.QTH,
		Comment:    q.Comment,
		Notes:      q.Notes,
		TXPower:    q.TXPower,
		SOTARef:    q.SOTARef,
		POTARef:    q.POTARef,
		WWFFRef:    q.WWFFRef,
		IOTA:       q.IOTA,
		SIG:        q.SIG,
		SIGInfo:    q.SIGInfo,
	}
	if strings.EqualFold(q.Mode, "MFSK") && q.Submode != "" {
		in.Mode = q.Submode // canonical: MFSK+FT8 → FT8, same as buildCreateQSOInput
	}
	if q.Freq > 0 {
		in.FreqHz = int64(math.Round(q.Freq * 1e6))
	}
	if q.FreqRx > 0 {
		in.FreqRxHz = int64(math.Round(q.FreqRx * 1e6))
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
