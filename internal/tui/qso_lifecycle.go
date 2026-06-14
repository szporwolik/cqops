package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// saveQSO validates, persists, and uploads the current QSO from the form fields.
// It orchestrates validation, DB insert, Wavelog upload, toast feedback,
// and form clearing/retention. This is cross-cutting lifecycle logic, not form-only.
func (m *Model) saveQSO() tea.Cmd {
	m.autoFillRST()
	m.autoFillSSBSubmode()
	qs := qso.NewQSO()
	var freq float64
	if _, err := fmt.Sscanf(m.fields[fieldFreq].Value(), "%f", &freq); err != nil {
		freq = 0
	}
	qs.Call, qs.Band, qs.Freq = strings.ToUpper(m.fields[fieldCall].Value()), strings.ToUpper(m.fields[fieldBand].Value()), freq
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
	station := qso.StationInfo{StationCallsign: m.App.Logbook.Station.Callsign, Operator: m.App.Logbook.Station.Operator, MyGridSquare: m.App.Logbook.Station.Grid, MyRig: m.App.Logbook.Station.Rig, MyAntenna: m.App.Logbook.Station.Antenna, TXPower: m.App.Logbook.Station.Power, MySOTARef: m.App.Logbook.Station.SOTARef, MyPOTARef: m.App.Logbook.Station.POTARef, MyWWFFRef: m.App.Logbook.Station.WWFFRef}
	if qs.GridSquare != "" && station.MyGridSquare != "" {
		qs.Distance = gridDistanceKm(station.MyGridSquare, qs.GridSquare)
		qs.Bearing = gridBearingDeg(station.MyGridSquare, qs.GridSquare)
	}
	qso.ApplyStationDefaults(qs, station)
	if err := qso.ValidateForSave(qs); err != nil {
		m.toasts.Error(err.Error())
		return nil
	}
	if _, err := store.InsertQSO(m.App.DB, qs); err != nil {
		m.toasts.Error(fmt.Sprintf("Save failed: %v", err))
		return nil
	}
	m.clearForm()
	m.toasts.Success(fmt.Sprintf("QSO saved: %s", qs.Call))
	return tea.Batch(m.refreshQSOS(), m.maybeUploadToWavelog(qs))
}

// refreshQSOS reloads the QSO list from the store and updates the RecentQSOs component.
func (m *Model) refreshQSOS() tea.Cmd {
	return func() tea.Msg {
		qsos, err := store.ListQSOs(m.App.DB, 500)
		if err != nil {
			m.toasts.Error(fmt.Sprintf("Refresh failed: %v", err))
			return nil
		}
		m.qsos = qsos
		m.recentQSOs.SetQSOS(qsos)
		return nil
	}
}
