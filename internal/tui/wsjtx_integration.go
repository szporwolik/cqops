package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
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
	m.wsjtxOnline = true
	prevTx := m.wsjtxTx
	prevMsg := m.wsjtxTxMsg
	m.wsjtxTxMsg = txMessage
	m.wsjtxLastSeen = time.Now()
	m.wsjtxTx = transmitting
	// Only invalidate the status bar cache when the visible TX state changes.
	if prevTx != transmitting || prevMsg != txMessage {
		m.cachedStatus = ""
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
			m.partnerData = nil
			m.wlPrivateData = nil
			m.wlLookupDone = false
			m.cachedLogStatsSig = ""
			m.invalidatePartnerMapCache()
			applog.InfoDetail("WSJT-X: switching DX call", fmt.Sprintf("%s \u2192 %s", prevCall, newCall))
			if m.App.Config.QRZ.Enabled && m.App.Config.QRZ.User != "" {
				applog.Info("QRZ: looking up " + call + "\u2026")
				m.qrzNeed = true
				m.qrzCall = newCall
			}
			if m.App.Logbook.Wavelog != nil && m.App.Logbook.Wavelog.Enabled {
				m.wlNeed = true
				m.wlCall = newCall
			}
		}
	}
	if grid != "" {
		m.fields[fieldGrid].SetValue(formatLocator(grid))
		m.pathGrid = strings.ToUpper(formatLocator(grid))
	}
	if freqHz > 0 {
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", float64(freqHz)/1_000_000.0))
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
	m.wsjtxStatus = mode
	if submode != "" {
		m.wsjtxStatus = submode
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
		TXPower:         m.App.Logbook.Station.RigPower(m.App.Config.Rigs),
		MySOTARef:       m.App.Logbook.Station.SOTARef,
		MyPOTARef:       m.App.Logbook.Station.POTARef,
		MyWWFFRef:       m.App.Logbook.Station.WWFFRef,
	})
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
	cmds = append(cmds, m.maybeUploadRawADIFToWavelog(adif, id, qs.Call))
	return tea.Batch(cmds...), false
}

// parseWSJTXADIF parses a single QSO record from an ADIF string.
func parseWSJTXADIF(adifStr string) *qso.QSO {
	qs := qso.NewQSO()
	adifStr = strings.TrimSpace(adifStr)

	s := adif.NewScanner(strings.NewReader(adifStr))
	for s.Scan() {
		if s.IsHeader() {
			continue
		}
		r := s.Record()
		if v := r[adifield.CALL]; v != "" {
			qs.Call = strings.ToUpper(v)
		}
		if v := r[adifield.GRIDSQUARE]; v != "" {
			qs.GridSquare = formatLocator(v)
		}
		if v := r[adifield.MODE]; v != "" {
			qs.Mode = strings.ToUpper(v)
		}
		if v := r[adifield.SUBMODE]; v != "" {
			qs.Submode = strings.ToUpper(v)
		}
		if v := r[adifield.RST_SENT]; v != "" {
			qs.RSTSent = v
		}
		if v := r[adifield.RST_RCVD]; v != "" {
			qs.RSTRcvd = v
		}
		if v := r[adifield.QSO_DATE]; v != "" {
			qs.QSODate = stripNonDigits(v)
		}
		if v := r[adifield.TIME_ON]; v != "" {
			qs.TimeOn = stripNonDigits(v)
		}
		if v := r[adifield.TIME_OFF]; v != "" {
			qs.TimeOff = stripNonDigits(v)
		}
		if v := r[adifield.BAND]; v != "" {
			qs.Band = qso.NormalizeBand(v)
		}
		if v := r[adifield.FREQ]; v != "" {
			if _, err := fmt.Sscanf(v, "%f", &qs.Freq); err != nil {
				applog.Warn("WSJT-X: bad ADIF frequency", "freq", v, "error", err)
			}
		}
		if v := r[adifield.FREQ_RX]; v != "" {
			if _, err := fmt.Sscanf(v, "%f", &qs.FreqRx); err != nil {
				applog.Warn("WSJT-X: bad ADIF frequency_rx", "freq", v, "error", err)
			}
		}
		if v := r[adifield.STATION_CALLSIGN]; v != "" {
			qs.StationCallsign = strings.ToUpper(v)
		}
		if v := r[adifield.MY_GRIDSQUARE]; v != "" {
			qs.MyGridSquare = formatLocator(v)
		}
		if v := r[adifield.OPERATOR]; v != "" {
			qs.Operator = strings.ToUpper(v)
		}
		if v := r[adifield.COMMENT]; v != "" {
			qs.Comment = v
		}
		if v := r[adifield.NAME]; v != "" {
			qs.Name = v
		}
		if v := r[adifield.QTH]; v != "" {
			qs.QTH = v
		}
		if v := r[adifield.COUNTRY]; v != "" {
			qs.Country = v
		}
		if v := r[adifield.DXCC]; v != "" && qs.Country == "" {
			qs.Country = v
		}
		if v := r[adifield.TX_PWR]; v != "" {
			qs.TXPower = v
		}
		if v := r[adifield.SOTA_REF]; v != "" {
			qs.SOTARef = v
		}
		if v := r[adifield.POTA_REF]; v != "" {
			qs.POTARef = v
		}
		if v := r[adifield.WWFF_REF]; v != "" {
			qs.WWFFRef = v
		}
		if v := r[adifield.IOTA]; v != "" {
			qs.IOTA = v
		}
		if v := r[adifield.MY_SOTA_REF]; v != "" {
			qs.MySOTARef = v
		}
		if v := r[adifield.MY_POTA_REF]; v != "" {
			qs.MyPOTARef = v
		}
		if v := r[adifield.MY_WWFF_REF]; v != "" {
			qs.MyWWFFRef = v
		}
		break
	}
	if err := s.Err(); err != nil {
		applog.Warn("WSJT-X: ADIF scanner error", "error", err)
	}

	qs.Mode, qs.Submode = qso.NormalizeMode(qs.Mode, qs.Submode)
	if qs.Band == "" && qs.Freq > 0 {
		qs.Band = qso.DeriveBand(qs.Freq)
	}
	return qs
}
