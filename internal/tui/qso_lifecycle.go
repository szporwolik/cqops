package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// qsoRefreshedMsg signals that the QSO list has been reloaded from the store.
type qsoRefreshedMsg struct{}

// saveQSO validates, persists, and uploads the current QSO from the form fields.
// It orchestrates validation, DB insert, Wavelog upload, toast feedback,
// and form clearing/retention. This is cross-cutting lifecycle logic, not form-only.
func (m *Model) saveQSO() tea.Cmd {
	m.autoFillRST()
	m.autoFillSSBSubmode()
	m.checkDupe() // ensure dupe state is fresh before saving
	qs := qso.NewQSO()
	var freq float64
	if _, err := fmt.Sscanf(m.fields[fieldFreq].Value(), "%f", &freq); err != nil {
		freq = 0
	}
	qs.Call, qs.Band, qs.Freq = qso.NormalizeCall(m.fields[fieldCall].Value()), strings.ToUpper(m.fields[fieldBand].Value()), freq
	var freqRx float64
	fmt.Sscanf(m.fields[fieldFreqRx].Value(), "%f", &freqRx)
	qs.FreqRx = freqRx
	qs.Mode, qs.RSTSent, qs.RSTRcvd = strings.ToUpper(m.fields[fieldMode].Value()), m.fields[fieldRSTSent].Value(), m.fields[fieldRSTRcvd].Value()
	qs.Submode = strings.ToUpper(m.fields[fieldSubmode].Value())
	qs.QSODate = stripNonDigits(m.fields[fieldDate].Value())
	if qs.QSODate == "" {
		qs.QSODate = time.Now().UTC().Format("20060102")
	}
	qs.TimeOn = stripNonDigits(m.fields[fieldTime].Value())
	if qs.TimeOn == "" {
		qs.TimeOn = time.Now().UTC().Format("150405")
	}
	qs.GridSquare = formatLocator(m.fields[fieldGrid].Value())
	qs.Comment, qs.Name, qs.QTH, qs.Country = m.fields[fieldComment].Value(), m.fields[fieldName].Value(), m.fields[fieldQTH].Value(), m.fields[fieldCountry].Value()
	qs.TXPower = strings.TrimSpace(m.fields[fieldTXPower].Value())
	qs.SOTARef = strings.TrimSpace(m.fields[fieldSOTA].Value())
	qs.POTARef = strings.TrimSpace(m.fields[fieldPOTA].Value())
	qs.WWFFRef = strings.TrimSpace(m.fields[fieldWWFF].Value())
	qs.IOTA = strings.TrimSpace(m.fields[fieldIOTA].Value())
	qs.SIG = strings.TrimSpace(m.fields[fieldSIG].Value())
	qs.ExchSent = strings.TrimSpace(m.fields[fieldExchSent].Value())
	qs.ExchRcvd = strings.TrimSpace(m.fields[fieldExchRcvd].Value())
	qs.STX = qso.ParseSerial(qs.ExchSent)
	qs.SRX = qso.ParseSerial(qs.ExchRcvd)
	qs.STXString = qs.ExchSent
	qs.SRXString = qs.ExchRcvd
	station := qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Operator,
		MyGridSquare:    m.App.Logbook.Station.Grid,
		MyRig:           m.App.Logbook.Station.RigModel(m.App.Config.Rigs),
		MyAntenna:       m.App.Logbook.Station.RigAntenna(m.App.Config.Rigs),
		TXPower:         m.App.Logbook.Station.RigPower(m.App.Config.Rigs),
		MySOTARef:       m.App.Logbook.Station.SOTARef,
		MyPOTARef:       m.App.Logbook.Station.POTARef,
		MyWWFFRef:       m.App.Logbook.Station.WWFFRef,
		MyCQZone:        qso.ItoaOrEmpty(m.App.Logbook.Station.CQZone),
		MyITUZone:       qso.ItoaOrEmpty(m.App.Logbook.Station.ITUZone),
		MyDXCC:          qso.ItoaOrEmpty(m.App.Logbook.Station.DXCC),
		MySIG:           m.App.Logbook.Station.SIG,
		MySIGInfo:       m.App.Logbook.Station.SIGInfo,
	}
	if qs.GridSquare != "" && station.MyGridSquare != "" {
		qs.Distance = gridDistanceKm(station.MyGridSquare, qs.GridSquare)
		qs.Bearing = gridBearingDeg(station.MyGridSquare, qs.GridSquare)
	}
	// Enrich CQ/ITU zone from DXCC prefix lookup (CTY.DAT).
	if m.App.Config.General.UseCTY && m.App.DXCC != nil {
		if p := m.dxccLookup(qs.Call); p != nil {
			qs.CQZone = fmt.Sprintf("%d", p.CQZone)
			qs.ITUZone = fmt.Sprintf("%d", p.ITUZone)
		}
	}
	qso.ApplyStationDefaults(qs, station)
	// Attach active contest to QSO.
	qs.ContestID = m.App.Config.State.ActiveContest
	// Set the ADIF Contest ID from the active contest config.
	if m.App.Config.State.ActiveContest != "" {
		if ct, ok := m.App.Config.Contests[m.App.Config.State.ActiveContest]; ok {
			qs.ContestADIFID = ct.ContestID
		}
	}
	if err := qso.ValidateForSave(qs); err != nil {
		applog.Warn("QSO validation failed", "error", err.Error())
		m.toasts.Error(err.Error())
		return nil
	}
	if _, err := store.InsertQSO(m.App.DB, qs); err != nil {
		m.toasts.Error(fmt.Sprintf("Save failed: %v", err))
		return nil
	}

	// Increment contest Next QSO seq on successful save.
	if qs.ContestID != "" {
		if ct, ok := m.App.Config.Contests[qs.ContestID]; ok {
			ct.NextQSO++
			m.App.Config.Contests[qs.ContestID] = ct
		}
	}

	// System notification on QSO saved.
	n := m.App.Config.General.Notifications
	if n.Enabled && n.QSO {
		applog.Info("Sending QSO notification", "call", qs.Call, "band", qs.Band, "mode", qs.Mode)
		if err := beeep.Notify("CQOps — QSO Logged", fmt.Sprintf("%s on %s %s", qs.Call, qs.Band, qs.Mode), ""); err != nil {
			applog.Warn("QSO notification failed", "error", err.Error())
		}
	}

	m.clearForm()
	m.toasts.Success(fmt.Sprintf("QSO saved: %s", qs.Call))
	return tea.Batch(m.refreshQSOS(), m.maybeUploadToWavelog(qs))
}

// refreshQSOS reloads the QSO list from the store, updates the RecentQSOs component,
// and re-applies any active filter. Returns a non-nil message to trigger a re-render.
func (m *Model) refreshQSOS() tea.Cmd {
	return func() tea.Msg {
		var qsos []qso.QSO
		var err error
		for attempt := 0; attempt < 3; attempt++ {
			qsos, err = store.ListQSOs(m.App.DB, 500, m.App.Config.State.ActiveContest)
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "database is locked") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if err != nil {
			m.toasts.Error(fmt.Sprintf("Refresh failed: %v", err))
			return qsoRefreshedMsg{} // still return message to trigger re-render
		}
		m.qsos = qsos
		m.recentQSOs.SetQSOS(qsos)
		m.rc.pathSig = ""
		m.rc.logStatsSig = ""

		// Re-apply filter if active.
		if m.recentQSOs.IsFiltered() {
			filtered, err := store.SearchQSOsByCall(m.App.DB, m.recentQSOs.filterCall, 200)
			if err == nil {
				m.recentQSOs.SetFilterCall(m.recentQSOs.filterCall, filtered)
			}
		}
		return qsoRefreshedMsg{}
	}
}

// updateFilteredTable searches the DB for QSOs matching the current callsign
// and applies the filter to the RecentQSOs table. When no call is entered,
// the filter is cleared and the table returns to normal mode.
func (m *Model) updateFilteredTable() tea.Cmd {
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if call == "" {
		m.recentQSOs.ClearFilter()
		return nil
	}
	// Don't re-query if already filtered for the same call and cache is valid.
	if m.recentQSOs.IsFiltered() && m.recentQSOs.filterCall == call && m.recentQSOs.filterCacheID != 0 {
		return nil
	}
	return func() tea.Msg {
		qsos, err := store.SearchQSOsByCall(m.App.DB, call, 200)
		if err != nil {
			return nil
		}
		m.recentQSOs.SetFilterCall(call, qsos)
		return nil
	}
}

// clearFilteredTable clears the RecentQSOs filter, returning to normal mode.
func (m *Model) clearFilteredTable() {
	m.recentQSOs.ClearFilter()
}
