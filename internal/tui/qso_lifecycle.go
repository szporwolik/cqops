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
type qsoRefreshedMsg struct {
	qsos []qso.QSO
	err  error
}

// saveQSO validates, persists, and uploads the current QSO from the form fields.
// It orchestrates validation, DB insert, Wavelog upload, toast feedback,
// and form clearing/retention. This is cross-cutting lifecycle logic, not form-only.
// Called directly by Enter (and Ctrl+S). All pre-save autofill (call commit,
// DXCC, contest exchange, RST, submode, dupe check) runs here so Enter is a
// single action: "log what's in the form".
func (m *Model) saveQSO() tea.Cmd {
	m.commitCall()
	m.dxccAutoFill()
	m.prefillContestExchange()
	m.autoFillRST()
	m.autoFillSSBSubmode()
	m.checkDupe() // ensure dupe state is fresh before saving
	if m.dupe && !m.dupeConfirmed {
		m.dupeConfirmed = true
		m.toasts.Warn("DUPE! " + strings.TrimSpace(m.fields[fieldCall].Value()) + " already logged on this band/mode today — press Enter again to log anyway")
		m.rc.pathSig = "" // invalidate path row cache to show DUPE badge
		return nil
	}
	m.dupeConfirmed = false
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
	// Pad to 6 digits (HHMMSS) if user didn't type seconds — DB expects full time.
	if len(qs.TimeOn) == 4 {
		qs.TimeOn += "00"
	}
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
	qs.SIGInfo = strings.TrimSpace(m.fields[fieldSIGInfo].Value())
	qs.ExchSent = strings.TrimSpace(m.fields[fieldExchSent].Value())
	qs.ExchRcvd = strings.TrimSpace(m.fields[fieldExchRcvd].Value())
	// Derive STX_STRING from EXCH_SENT by stripping the RST prefix;
	// fall back to the full exchange if the RST is not a prefix match.
	qs.STXString = qso.StripRSTPrefix(qs.ExchSent, qs.RSTSent)
	qs.SRXString = qso.StripRSTPrefix(qs.ExchRcvd, qs.RSTRcvd)
	// STX/SRX are the last integer in the exchange (the serial).
	// ParseSerial returns 0 when no integer is found (normal QSOs, non-serial contests).
	qs.STX = qso.ParseSerial(qs.ExchSent)
	qs.SRX = qso.ParseSerial(qs.ExchRcvd)
	station := qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.activeOperatorCallsign(),
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
	applog.Debug("QSO save: operator", "active", m.App.Logbook.ActiveOperator, "callsign", station.Operator)
	// Attach active contest to QSO.
	qs.ContestID = m.App.Logbook.ActiveContest
	// Set the ADIF Contest ID from the active contest config.
	if m.App.Logbook.ActiveContest != "" {
		if ct, ok := m.App.Config.Contests[m.App.Logbook.ActiveContest]; ok {
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
	m.clearFilteredTable()
	m.toasts.Success(fmt.Sprintf("QSO saved: %s", qs.Call))
	m.pushLoggedQSOToDashboard(qs)
	return tea.Batch(m.refreshQSOS(), m.maybeUploadToWavelog(qs))
}

// refreshQSOS reloads the QSO list from the store, updates the RecentQSOs component,
// and re-applies any active filter. Returns a non-nil message to trigger a re-render.
func (m *Model) refreshQSOS() tea.Cmd {
	return func() tea.Msg {
		var qsos []qso.QSO
		var err error
		for attempt := 0; attempt < 2; attempt++ {
			qsos, err = store.ListQSOs(m.App.DB, 500, m.App.Logbook.ActiveContest)
			if err == nil {
				break
			}
			if !strings.Contains(err.Error(), "database is locked") {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		return qsoRefreshedMsg{qsos: qsos, err: err}
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
		// Re-read call at execution time — the field may have changed
		// since the command was created.
		currentCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		if currentCall == "" {
			m.recentQSOs.ClearFilter()
			return nil
		}
		qsos, err := store.SearchQSOsByCall(m.App.DB, currentCall, 200)
		if err != nil {
			return nil
		}
		m.recentQSOs.SetFilterCall(currentCall, qsos)
		return nil
	}
}

// clearFilteredTable clears the RecentQSOs filter, returning to normal mode.
func (m *Model) clearFilteredTable() {
	m.recentQSOs.ClearFilter()
}
