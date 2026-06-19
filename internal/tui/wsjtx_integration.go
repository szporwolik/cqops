package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	adif "github.com/farmergreg/adif/v5"
	"github.com/gen2brain/beeep"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// =============================================================================
// WSJT-X integration — status updates and automatic QSO logging from ADIF.
// =============================================================================

// applyWSJTXStatus applies a WSJT-X status update to the QSO form fields.
func (m *Model) applyWSJTXStatus(call, grid string, freqHz uint64, mode, submode, report, txMessage string, transmitting bool) {
	m.wsjtx.online = true
	prevTx := m.wsjtx.tx
	prevMsg := m.wsjtx.txMsg
	m.wsjtx.txMsg = txMessage
	m.wsjtx.lastSeen = time.Now()
	m.wsjtx.tx = transmitting
	// Only invalidate the status bar cache when the visible TX state changes.
	if prevTx != transmitting || prevMsg != txMessage {
		m.rc.status = ""
	}
	if call != "" {
		prevCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		newCall := strings.ToUpper(call)
		if prevCall != newCall {
			m.fields[fieldCall].SetValue(call)
			m.fields[fieldCountry].SetValue("")
			m.fields[fieldName].SetValue("")
			m.fields[fieldQTH].SetValue("")
			m.fields[fieldGrid].SetValue("")
			m.lookup.partnerData = nil
			m.lookup.wlPrivateData = nil
			m.lookup.wlLookupDone = false
			m.rc.logStatsSig = ""
			m.invalidatePartnerMapCache()
			applog.InfoDetail("WSJT-X: switching DX call", fmt.Sprintf("%s \u2192 %s", prevCall, newCall))
			if m.App.Config.QRZ.Enabled && m.App.Config.QRZ.User != "" {
				applog.Info("QRZ: looking up " + call + "\u2026")
				m.lookup.qrzNeed = true
				m.lookup.qrzCall = newCall
			}
			if m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled {
				m.lookup.wlNeed = true
				m.lookup.wlCall = newCall
			}
		}
	}
	if grid != "" {
		formatted := formatLocator(grid)
		current := strings.ToUpper(strings.TrimSpace(m.fields[fieldGrid].Value()))
		// QRZ may have already filled a more precise grid (e.g. JN54ks vs
		// WSJT-X's JN54). Only overwrite if the current grid is empty or
		// if the WSJT-X grid is not a prefix of the current one (different
		// location altogether).
		if current == "" || !strings.HasPrefix(current, strings.ToUpper(formatted)) {
			m.fields[fieldGrid].SetValue(formatted)
			m.rc.pathGrid = strings.ToUpper(formatted)
		}
	}
	if freqHz > 0 {
		freqMHz := float64(freqHz) / 1_000_000.0
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", freqMHz))
		if band := qso.DeriveBand(freqMHz); band != "" {
			m.fields[fieldBand].SetValue(band)
		}
	}
	if mode != "" {
		mode, submode = qso.NormalizeMode(mode, submode)
		m.fields[fieldMode].SetValue(mode)
	}
	if submode != "" {
		m.fields[fieldSubmode].SetValue(submode)
	}
	if report != "" {
		m.fields[fieldRSTSent].SetValue(report)
		m.fields[fieldRSTRcvd].SetValue(report)
	}
	m.autoFillRST()
	m.wsjtx.status = mode
	if submode != "" {
		m.wsjtx.status = submode
	}
}

// logQSOFromADIF validates, persists, and uploads a QSO from raw WSJT-X ADIF data.
// logQSOFromADIF parses a WSJT-X ADIF record, inserts it into the database,
// and returns a command to upload it to Wavelog. Returns (cmd, true) on success,
// (nil, false) if the ADIF should be skipped permanently (invalid/duplicate),
// or (nil, true) if the insert failed and should be retried.
func (m *Model) logQSOFromADIF(adif string) (tea.Cmd, bool) {
	qs := parseWSJTXADIF(adif)
	if qs.Call == "" {
		applog.Warn("WSJT-X: logged ADIF has no call, skipping")
		m.toasts.Warn("WSJT-X: ADIF has no call")
		return nil, false // skip permanently
	}
	qs.Source = "wsjtx"
	qs.WavelogUploaded = "no"
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Operator,
		MyGridSquare:    m.App.Logbook.Station.Grid,
		MyRig:           m.App.Logbook.Station.RigModel(m.App.Config.Rigs),
		MyAntenna:       m.App.Logbook.Station.RigAntenna(m.App.Config.Rigs),
		TXPower:         txPowerForWSJTX(m),
		MySOTARef:       m.App.Logbook.Station.SOTARef,
		MyPOTARef:       m.App.Logbook.Station.POTARef,
		MyWWFFRef:       m.App.Logbook.Station.WWFFRef,
	})

	// Enrich QSO: compute distance/bearing from grid squares.
	myGrid := m.App.Logbook.Station.Grid
	if qs.GridSquare != "" && myGrid != "" {
		qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
		qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
	}

	if err := qso.ValidateForSave(qs); err != nil {
		applog.Error("WSJT-X: ADIF validation failed", "error", err.Error())
		m.toasts.Error("WSJT-X: " + err.Error())
		return nil, false // skip permanently
	}

	// Duplicate detection — WSJT-X may re-send the same QSO ADIF.
	if existingID := store.FindQSOByKey(m.App.DB, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
		applog.Info("WSJT-X: duplicate ADIF skipped", "call", qs.Call, "band", qs.Band, "existing_id", existingID)
		return nil, false // skip permanently
	}

	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		applog.Error("WSJT-X: DB insert failed — will retry", "error", err.Error(), "call", qs.Call)
		m.toasts.Error("WSJT-X: DB save failed — retrying")
		return nil, true // retry on next tick
	}
	applog.InfoDetail("WSJT-X: auto-logged QSO", fmt.Sprintf("id=%d call=%s", id, qs.Call))
	m.toasts.Success(fmt.Sprintf("WSJT-X: %s logged", qs.Call))

	n := m.App.Config.General.Notifications
	if n.Enabled && n.QSO {
		applog.Info("Sending WSJT-X QSO notification", "call", qs.Call, "band", qs.Band, "mode", qs.Mode)
		if err := beeep.Notify("CQOps — QSO Logged", fmt.Sprintf("%s on %s %s", qs.Call, qs.Band, qs.Mode), ""); err != nil {
			applog.Warn("QSO notification failed", "error", err.Error())
		}
	}

	m.clearForm()
	m.needRefresh = true

	// Only refresh the QSO table immediately if the user is on the QSO form.
	// On other screens the refresh is deferred via needRefresh → handlePendingRequests.
	var cmds []tea.Cmd
	if m.screen == screenQSO {
		cmds = append(cmds, m.refreshQSOS())
	}
	cmds = append(cmds, m.wsjtxEnrichAndUploadCmd(id, qs.Call))
	return tea.Batch(cmds...), false
}

// wsjtxEnrichAndUploadCmd returns a command that enriches a WSJT-X auto-logged
// QSO via QRZ (if configured) and then uploads the enriched QSO to Wavelog.
// This ensures the QSO on Wavelog contains QRZ-derived fields (Name, QTH,
// Country, GridSquare) rather than the raw WSJT-X ADIF.
// Returns nil when neither QRZ enrichment nor Wavelog upload is possible.
func (m *Model) wsjtxEnrichAndUploadCmd(qsoID int64, call string) tea.Cmd {
	if call == "" {
		return nil
	}
	qrzenabled := m.App.Config.QRZ.Enabled && m.App.Config.QRZ.User != ""
	wl := m.App.Logbook.Wavelog
	wlenabled := wl != nil && wl.Enabled && wl.StationProfileID != ""
	if !qrzenabled && !wlenabled {
		return nil // nothing to do
	}
	return func() tea.Msg {
		// Step 1: enrich via QRZ (best-effort).
		if qrzenabled {
			data, err := qrzLookupFunc(m.App.Config.QRZ.User, m.App.Config.QRZ.Pass, call)
			if err != nil {
				applog.Warn("WSJT-X: QRZ enrichment failed", "call", call, "error", err)
			} else {
				store.UpdateQSOEnrichment(m.App.DB, qsoID, store.EnrichmentData{
					Name:       data.Name,
					QTH:        data.QTH,
					Country:    data.Country,
					GridSquare: data.Grid,
					CQZone:     data.CQZone,
					ITUZone:    data.ITUZone,
				})
				applog.Info("WSJT-X: QRZ enrichment applied", "call", call, "qso_id", qsoID)
			}
		}

		// Step 1b: enrich CQ/ITU zone from DXCC if not already filled by QRZ.
		if m.App.Config.General.UseCTY && m.App.DXCC != nil {
			qs, _ := store.GetQSOByID(m.App.DB, qsoID)
			if qs != nil && (qs.CQZone == "" || qs.ITUZone == "") {
				if p := m.dxccLookup(call); p != nil {
					cqz := fmt.Sprintf("%d", p.CQZone)
					ituz := fmt.Sprintf("%d", p.ITUZone)
					store.UpdateQSOEnrichment(m.App.DB, qsoID, store.EnrichmentData{
						CQZone:  cqz,
						ITUZone: ituz,
					})
					applog.Debug("DXCC: filled CQ/ITU zone from prefix", "call", call, "cqz", cqz, "ituz", ituz)
				}
			}
		}

		// Step 2: load the enriched QSO from DB.
		qs, err := store.GetQSOByID(m.App.DB, qsoID)
		if err != nil {
			applog.Error("WSJT-X: cannot load QSO for Wavelog upload", "qso_id", qsoID, "error", err)
			return nil
		}

		// Step 2b: recompute distance/bearing after enrichment. WSJT-X may
		// not include a grid, or the enriched grid may be more precise.
		if myGrid := m.App.Logbook.Station.Grid; myGrid != "" && qs.GridSquare != "" {
			qs.Distance = gridDistanceKm(myGrid, qs.GridSquare)
			qs.Bearing = gridBearingDeg(myGrid, qs.GridSquare)
			m.App.DB.Exec(`UPDATE qsos SET distance=?, bearing=? WHERE id=?`,
				qs.Distance, qs.Bearing, qsoID)
		}

		// Step 3: upload the enriched QSO's ADIF to Wavelog.
		if !wlenabled || !m.inetOnline {
			return wsjtxEnrichDoneMsg{}
		}
		adifStr := qs.ToADIF()
		ok, isDup, uploadErr := postQSO(wl.URL, wl.APIKey, wl.StationProfileID, adifStr, qsoID, call, m.App.DB)
		return wlUploadResultMsg{qID: qsoID, call: call, ok: ok, isDup: isDup, err: uploadErr}
	}
}

// parseWSJTXADIF parses a single QSO record from a WSJT-X ADIF string.
// Delegates to the shared qso.ParseADIFRecord for field extraction.
func parseWSJTXADIF(adifStr string) *qso.QSO {
	adifStr = strings.TrimSpace(adifStr)

	s := adif.NewScanner(strings.NewReader(adifStr))
	for s.Scan() {
		if s.IsHeader() {
			continue
		}
		qs := qso.ParseADIFRecord(s.Record(), "wsjtx")

		// WSJT-X specific post-processing: normalize mode/submode and derive band.
		qs.Mode, qs.Submode = qso.NormalizeMode(qs.Mode, qs.Submode)
		if qs.Band == "" && qs.Freq > 0 {
			qs.Band = qso.DeriveBand(qs.Freq)
		}
		return qs
	}
	if err := s.Err(); err != nil {
		applog.Warn("WSJT-X: ADIF scanner error", "error", err)
	}
	return qso.NewQSO()
}

// txPowerForWSJTX returns the TX power to use when auto-logging a WSJT-X QSO.
// Prefers the current form field value (populated by flrig or manual entry);
// falls back to the station's rig preset power.
func txPowerForWSJTX(m *Model) string {
	if fp := strings.TrimSpace(m.fields[fieldTXPower].Value()); fp != "" {
		return fp
	}
	return m.App.Logbook.Station.RigPower(m.App.Config.Rigs)
}
