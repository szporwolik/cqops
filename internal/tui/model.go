package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig/flrig"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type field int

const (
	rigPollInterval     = 15                      // ticks between flrig polls
	healthCheckTicks    = 300                     // ticks between health checks
	flrigStatusTimeout  = 1500 * time.Millisecond // context timeout for flrig status
	flrigDefaultTimeout = 1000                    // default flrig HTTP timeout (ms)
)

const (
	fieldDate field = iota
	fieldTime
	fieldCall
	fieldFreq
	fieldBand
	fieldMode
	fieldSubmode
	fieldRSTSent
	fieldRSTRcvd
	fieldName
	fieldQTH
	fieldGrid
	fieldCountry
	fieldTXPower
	fieldFreqRx
	fieldSOTA
	fieldPOTA
	fieldWWFF
	fieldIOTA
	fieldComment
	fieldCount // sentinel: must be last; equals number of fields above
)

var fieldNames = []string{
	"Date UTC", "Time UTC", "Call", "Frequency", "Band", "Mode", "Submode",
	"RST sent", "RST rcvd", "Name", "QTH", "Grid", "DXCC", "Power W", "Freq RX",
	"SOTA Ref", "POTA Ref", "WWFF Ref", "IOTA", "Comment",
}

type Model struct {
	App               *app.App
	fields            [fieldCount]textinput.Model
	focus             field
	qsos              []qso.QSO
	toasts            *ToastQueue
	err               error
	width             int
	height            int
	quitting          bool
	rigConnected      bool
	rigFreq           float64
	rigMode           string
	rigPower          float64
	rigBlink          bool
	rigSkipTicks      int
	rigPolling        bool
	dateTimeAuto      bool
	tickCount         int
	inetOnline        bool
	wsjtxOnline       bool
	wsjtxStatus       string
	needRefresh       bool
	pendingADIF       string
	pendingStatus     statusPending
	adifMu            sync.Mutex
	showChooser       bool
	chooser           *LogbookChooser
	showRigEdit       bool
	rigChooser        *RigChooser
	showConfig        bool
	configMenu        *GeneralMenu
	showCallbook      bool
	callbookMenu      *CallbookMenu
	showIntegration   bool
	integrationMenu   *IntegrationMenu
	showMainMenu      bool
	showLogView       bool
	logViewer         *LogViewer
	showLogbookEditor bool
	logbookEditor     *LogbookEditor
	mainMenu          *MainMenu
	confirmQuit       bool
	showPartner       bool
	partnerData       *qrz.CallData
	flrigClient       *flrig.Client
	qrzNeed           bool
	qrzCall           string
	qrzLastLook       time.Time
	retainComment     bool
	retainFocused     bool // true when the Retain checkbox has focus (instead of a text field)
	wlOnline          bool
	wlStationName     string // e.g. "JO30oo / DJ7NT"
	wlStationLabel    string // e.g. "Debug location"
}

type tickMsg time.Time
type qrzResultMsg struct {
	Call string
	Data *qrz.CallData
	Err  error
}
type inetResultMsg bool
type statusPending struct {
	call, grid, mode, submode, report string
	freq                              uint64
	hasData                           bool
}

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	m := &Model{App: a, qsos: initialQSOS, toasts: NewToastQueue(), dateTimeAuto: true}
	now := time.Now().UTC()
	for i := field(0); i < fieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 40
		switch i {
		case fieldCall:
			ti.Focus()
		case fieldBand:
			ti.CharLimit = 8
		case fieldFreq, fieldFreqRx:
			ti.CharLimit = 16
		case fieldMode:
			ti.CharLimit = 12
		case fieldSubmode:
			ti.CharLimit = 16
		case fieldDate:
			ti.CharLimit = 10
			ti.SetValue(now.Format("2006-01-02"))
		case fieldTime:
			ti.CharLimit = 8
			ti.SetValue(now.Format("15:04:05"))
		case fieldGrid:
			ti.CharLimit = 8
		case fieldCountry:
			ti.CharLimit = 20
		case fieldName:
			ti.CharLimit = 30
		case fieldQTH:
			ti.CharLimit = 30
		case fieldTXPower:
			ti.CharLimit = 8
		case fieldComment:
			ti.CharLimit = 60
		case fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA:
			ti.CharLimit = 20
		}
		m.fields[i] = ti
	}
	m.focus = fieldCall
	return m
}

func (m *Model) Init() tea.Cmd {
	m.refreshFlrigClient()
	m.App.WSJTX.OnADIF = func(adif string) {
		m.adifMu.Lock()
		m.pendingADIF = adif
		m.adifMu.Unlock()
	}
	m.App.WSJTX.OnStatus = func(call, grid string, freq uint64, mode, submode, report string) {
		m.adifMu.Lock()
		m.pendingStatus = statusPending{call: call, grid: grid, freq: freq, mode: mode, submode: submode, report: report, hasData: true}
		m.adifMu.Unlock()
	}
	applog.Info("WSJT-X: callbacks registered, restarting listener")
	m.App.MaybeRestartWSJTX()
	return tea.Batch(tickCmd(), checkInetCmd())
}
func tickCmd() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}
func (m *Model) qrzLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := qrz.Lookup(m.App.Config.QRZUser, m.App.Config.QRZPass, call)
		return qrzResultMsg{Call: call, Data: data, Err: err}
	}
}

func (m *Model) qrzLookup(call string) tea.Cmd {
	if time.Since(m.qrzLastLook) < 3*time.Second {
		applog.Debug("QRZ: debounced", "callsign", call)
		return nil
	}
	m.qrzLastLook = time.Now()
	applog.Info("QRZ: looking up", "call", call)
	return m.qrzLookupCmd(call)
}

func isLookupKey(key tea.KeyMsg) bool {
	s := key.String()
	return s == "insert" || s == "\x1b[2~" || s == "ctrl+l" ||
		key.Type == tea.KeyInsert
}

func checkInetCmd() tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("https://clients3.google.com/generate_204")
		if err != nil {
			return inetResultMsg(false)
		}
		resp.Body.Close()
		return inetResultMsg(true)
	}
}

func (m *Model) refreshFlrigClient() {
	if m.App == nil || m.App.Logbook == nil {
		return
	}
	if len(m.App.Config.Rigs) == 0 {
		s := m.App.Logbook.Station
		m.App.Config.Rigs = map[string]config.RigPreset{"default": {
			Model: s.Rig, Antenna: s.Antenna, Power: s.Power,
			FlrigEnabled: m.App.Config.Rig.Flrig.Enabled, FlrigHost: "localhost", FlrigPort: "12345",
		}}
	}
	rigName := m.App.Logbook.Station.RigName
	if rigName == "" {
		rigName = "default"
	}
	if rp, ok := m.App.Config.Rigs[rigName]; ok && rp.FlrigEnabled {
		host, port := rp.FlrigHost, rp.FlrigPort
		if host == "" {
			host = "localhost"
		}
		if port == "" {
			port = "12345"
		}
		url := "http://" + host + ":" + port
		applog.InfoDetail("flrig: connecting", fmt.Sprintf("rig=%s host=%s port=%s url=%s", rigName, host, port, url))
		m.flrigClient = flrig.New(url, flrigDefaultTimeout)
	} else {
		if !ok {
			applog.Debug("flrig: rig not found in config", "rigName", rigName)
		} else {
			applog.Debug("flrig: disabled for rig", "rigName", rigName)
		}
		m.flrigClient = nil
	}
}

type flrigResultMsg struct {
	connected bool
	freq      float64
	mode      string
	band      string
	power     float64
	err       string
}

func (m *Model) flrigStatusCmd() tea.Cmd {
	if m.flrigClient == nil {
		return nil
	}
	client := m.flrigClient
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), flrigStatusTimeout)
		defer cancel()
		s, err := client.Status(ctx)
		if err != nil {
			return flrigResultMsg{err: err.Error()}
		}
		return flrigResultMsg{connected: s.Connected, freq: s.FrequencyMHz, mode: s.Mode, band: s.Band, power: s.Power}
	}
}

func (m *Model) pollFlrig() tea.Cmd {
	m.rigSkipTicks++
	if m.rigSkipTicks < rigPollInterval {
		return nil
	}
	m.rigSkipTicks = 0
	if m.flrigClient == nil {
		m.rigConnected = false
		return nil
	}
	if m.rigPolling {
		return nil
	}
	m.rigPolling = true
	m.rigBlink = !m.rigBlink
	return m.flrigStatusCmd()
}

func (m *Model) applyFlrigResult(r flrigResultMsg) {
	m.rigPolling = false
	if r.err != "" {
		m.rigConnected = false
		return
	}
	if !r.connected {
		m.rigConnected = false
		return
	}
	m.rigConnected = true
	m.rigFreq = r.freq
	if !m.wsjtxOnline {
		m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", r.freq))
	}
	if r.mode != "" && !m.wsjtxOnline {
		m.fields[fieldMode].SetValue(r.mode)
	}
	if r.band != "" {
		m.fields[fieldBand].SetValue(r.band)
	}
	if !m.wsjtxOnline {
		m.autoFillSSBSubmode()
	}
	if r.power > 0 {
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", r.power))
	}
}

func (m *Model) autoUpdateDateTime() {
	if !m.dateTimeAuto {
		return
	}
	now := time.Now().UTC()
	m.fields[fieldDate].SetValue(now.Format("2006-01-02"))
	m.fields[fieldTime].SetValue(now.Format("15:04:05"))
}

func (m *Model) maybeCheckInet() tea.Cmd {
	if m.tickCount%healthCheckTicks == 0 {
		return checkInetCmd()
	}
	return nil
}

func (m *Model) maybeCheckWavelog() tea.Cmd {
	if !m.App.Config.Wavelog.Enabled {
		m.wlOnline = false
		return nil
	}
	// Check on first tick and every N ticks thereafter
	if m.tickCount != 1 && m.tickCount%healthCheckTicks != 0 {
		return nil
	}
	return m.checkWavelogCmd()
}

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

// maybeUploadToWavelog returns a tea.Cmd that sends a QSO to Wavelog asynchronously.
func (m *Model) maybeUploadToWavelog(qs *qso.QSO) tea.Cmd {
	return m.uploadADIFToWavelog(qs.ToADIF(), qs.ID, qs.Call)
}

// maybeUploadRawADIFToWavelog returns a tea.Cmd that sends raw ADIF (from WSJT-X) to Wavelog.
func (m *Model) maybeUploadRawADIFToWavelog(adifStr string, qID int64, call string) tea.Cmd {
	return m.uploadADIFToWavelog(adifStr, qID, call)
}

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

func (m *Model) applyWSJTXStatus(call, grid string, freqHz uint64, mode, submode, report string) {
	m.wsjtxOnline = true
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
			applog.InfoDetail("WSJT-X: switching DX call", fmt.Sprintf("%s → %s", prevCall, newCall))
			if m.App.Config.QRZEnabled && m.App.Config.QRZUser != "" {
				applog.Info("QRZ: looking up " + call + "…")
				m.qrzNeed = true
				m.qrzCall = newCall
			}
		}
	}
	if grid != "" {
		m.fields[fieldGrid].SetValue(formatLocator(grid))
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

func (m *Model) logQSOFromADIF(adif string) tea.Cmd {
	qs := parseWSJTXADIF(adif)
	if qs.Call == "" {
		applog.Warn("WSJT-X: logged ADIF has no call, skipping")
		m.toasts.Warn("WSJT-X: ADIF has no call")
		return nil
	}
	qso.ApplyStationDefaults(qs, qso.StationInfo{
		StationCallsign: m.App.Logbook.Station.Callsign,
		Operator:        m.App.Logbook.Station.Operator,
		MyGridSquare:    m.App.Logbook.Station.Grid,
		MyRig:           m.App.Logbook.Station.Rig,
		MyAntenna:       m.App.Logbook.Station.Antenna,
		TXPower:         m.App.Logbook.Station.Power,
		MySOTARef:       m.App.Logbook.Station.SOTARef,
		MyPOTARef:       m.App.Logbook.Station.POTARef,
		MyWWFFRef:       m.App.Logbook.Station.WWFFRef,
	})
	if err := qso.ValidateForSave(qs); err != nil {
		applog.Error("WSJT-X: ADIF validation failed", "error", err.Error())
		m.toasts.Error("WSJT-X: " + err.Error())
		return nil
	}
	id, err := store.InsertQSO(m.App.DB, qs)
	if err != nil {
		applog.Error("WSJT-X: DB insert failed", "error", err.Error())
		m.toasts.Error("WSJT-X: DB save failed")
		return nil
	}
	applog.InfoDetail("WSJT-X: auto-logged QSO", fmt.Sprintf("id=%d call=%s", id, qs.Call))
	m.toasts.Success(fmt.Sprintf("WSJT-X: %s logged", qs.Call))
	m.clearForm()
	m.needRefresh = true
	return m.maybeUploadRawADIFToWavelog(adif, id, qs.Call)
}

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
			fmt.Sscanf(v, "%f", &qs.Freq)
		}
		if v := r[adifield.FREQ_RX]; v != "" {
			fmt.Sscanf(v, "%f", &qs.FreqRx)
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
		break // only process first QSO record
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle WindowSizeMsg first — store dimensions, then let sub-models see it too
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
	}

	if _, ok := msg.(tickMsg); ok {
		m.adifMu.Lock()
		adif := m.pendingADIF
		m.pendingADIF = ""
		m.adifMu.Unlock()
		if adif != "" {
			applog.Info("WSJT-X: processing pending ADIF")
			cmd = tea.Batch(cmd, m.logQSOFromADIF(adif))
		}
		m.adifMu.Lock()
		sp := m.pendingStatus
		m.pendingStatus = statusPending{}
		m.adifMu.Unlock()
		if sp.hasData {
			m.applyWSJTXStatus(sp.call, sp.grid, sp.freq, sp.mode, sp.submode, sp.report)
		}
		m.toasts.Expire()
		m.autoUpdateDateTime()
		m.tickCount++
		cmd = tea.Batch(tickCmd(), m.maybeCheckInet(), m.pollFlrig(), m.maybeCheckWavelog())
	}
	if m.confirmQuit {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "y", "Y":
				return m, tea.Quit
			default:
				m.confirmQuit = false
			}
		}
		return m, cmd
	}
	if ir, ok := msg.(inetResultMsg); ok {
		m.inetOnline = bool(ir)
	}
	if wl, ok := msg.(wlStatusMsg); ok {
		m.wlOnline = wl.online
		if wl.stationName != "" {
			m.wlStationName = wl.stationName
		}
		if wl.stationLabel != "" {
			m.wlStationLabel = wl.stationLabel
		}
	}
	if ur, ok := msg.(wlUploadResultMsg); ok {
		if ur.ok {
			store.UpdateWavelogStatus(m.App.DB, ur.qID, "yes")
			m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", ur.call))
		} else {
			store.UpdateWavelogStatus(m.App.DB, ur.qID, "no")
			m.toasts.Warn(fmt.Sprintf("Wavelog: %s failed", ur.call))
		}
	}
	if fr, ok := msg.(flrigResultMsg); ok {
		m.applyFlrigResult(fr)
		return m, cmd
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "f10":
			applog.Debug("tab: F10 quit requested")
			m.confirmQuit = true
		case "f1":
			applog.Debug("tab: F1 QSO Form")
			m.showChooser = false
			m.showRigEdit = false
			m.showIntegration = false
			m.showConfig = false
			m.showMainMenu = false
			m.showLogView = false
			m.showLogbookEditor = false
			m.showPartner = false
		case "f2":
			// F2 toggle: partner sub-model handles close, QSO form handler handles open+lookup
			if !m.showPartner {
				applog.Debug("tab: F2 Partner Details (clearing views)")
				m.showChooser = false
				m.showRigEdit = false
				m.showIntegration = false
				m.showConfig = false
				m.showCallbook = false
				m.showMainMenu = false
				m.showLogView = false
				m.showLogbookEditor = false
			}
		case "f8":
			if m.showMainMenu {
				applog.Debug("tab: F8 close Config")
				m.showMainMenu = false
			} else {
				applog.Debug("tab: F8 Config")
				m.showChooser = false
				m.showRigEdit = false
				m.showIntegration = false
				m.showConfig = false
				m.showCallbook = false
				m.showLogView = false
				m.showLogbookEditor = false
				m.showPartner = false
				m.mainMenu = NewMainMenu()
				m.showMainMenu = true
			}
		case "f5":
			applog.Debug("tab: F5 Log Editor")
			m.showChooser = false
			m.showRigEdit = false
			m.showIntegration = false
			m.showConfig = false
			m.showCallbook = false
			m.showMainMenu = false
			m.showLogView = false
			m.showPartner = false
			m.showLogbookEditor = true
			m.logbookEditor = NewLogbookEditor(m.App.DB, m.App.Config.Wavelog.URL, m.App.Config.Wavelog.APIKey, m.App.Config.Wavelog.StationProfileID, m.App.Config.Wavelog.StationCallsign, m.App.Logbook.Station.Operator, m.App.Logbook.Station.Grid)
			qsos, _ := store.ListAllQSOs(m.App.DB)
			m.logbookEditor.SetQSOS(qsos)
			return m, cmd
		case "f9":
			applog.Debug("tab: F9 Log Viewer")
			m.showChooser = false
			m.showRigEdit = false
			m.showIntegration = false
			m.showConfig = false
			m.showCallbook = false
			m.showMainMenu = false
			m.showLogbookEditor = false
			m.showPartner = false
			m.logViewer = NewLogViewer(m.App.LogbookName)
			m.showLogView = true
			return m, cmd
		}
		if !m.showChooser && !m.showRigEdit && !m.showIntegration && !m.showConfig && !m.showCallbook && !m.showMainMenu && !m.showLogView && !m.showLogbookEditor && !m.showPartner {
			if key.String() == "delete" || key.Type == tea.KeyDelete {
				m.clearForm()
				return m, nil
			}
			if isLookupKey(key) {
				call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
				if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled {
					return m, m.qrzLookup(call)
				}
			}
		}
	}
	if m.showChooser {
		m.chooser.width = m.width
		m.chooser.height = m.height
		_, chooserCmd := m.chooser.Update(msg)
		cmd = tea.Batch(cmd, chooserCmd)
		if m.chooser.done {
			m.showChooser = false
			m.showMainMenu = true
			m.needRefresh = true
		}
		return m, cmd
	}
	if m.showRigEdit {
		m.rigChooser.width = m.width
		m.rigChooser.height = m.height
		_, rigCmd := m.rigChooser.Update(msg)
		cmd = tea.Batch(cmd, rigCmd)
		if m.rigChooser.done {
			m.showRigEdit = false
			m.showMainMenu = true
			m.refreshFlrigClient()
		}
		return m, cmd
	}
	if m.showConfig {
		m.configMenu.width = m.width
		m.configMenu.height = m.height
		_, configCmd := m.configMenu.Update(msg)
		cmd = tea.Batch(cmd, configCmd)
		if m.configMenu.done {
			m.showConfig = false
			if m.configMenu.goBack {
				m.showMainMenu = true
			}
			if m.configMenu.saved {
				m.App.Config.DistanceUnit = m.configMenu.distanceUnit
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					applog.Info("Settings saved")
				}
				m.showMainMenu = true
			}
		}
		return m, cmd
	}
	if m.showCallbook {
		m.callbookMenu.width = m.width
		m.callbookMenu.height = m.height
		m.callbookMenu.inetOnline = m.inetOnline
		_, callbookCmd := m.callbookMenu.Update(msg)
		if m.callbookMenu.done {
			m.showCallbook = false
			if m.callbookMenu.goBack {
				m.showMainMenu = true
			}
			if m.callbookMenu.saved {
				m.App.Config.QRZUser = m.callbookMenu.user.Value()
				m.App.Config.QRZPass = m.callbookMenu.pass.Value()
				m.App.Config.QRZEnabled = m.callbookMenu.enabled
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					applog.Info("Settings saved")
				}
				m.showMainMenu = true
			}
		}
		return m, tea.Batch(cmd, callbookCmd)
	}
	if m.showIntegration {
		m.integrationMenu.width = m.width
		m.integrationMenu.height = m.height
		_, integrationCmd := m.integrationMenu.Update(msg)
		if m.integrationMenu.done {
			m.showIntegration = false
			if m.integrationMenu.goBack {
				m.showMainMenu = true
			}
			if m.integrationMenu.saved {
				wsjtxE, wsjtxH, wsjtxP, wlE, wlURL, wlKey, wlSta, wlStaCall, _ := m.integrationMenu.Values()
				m.App.Config.WSJTX.Enabled = wsjtxE
				m.App.Config.WSJTX.UDPHost = wsjtxH
				m.App.Config.WSJTX.UDPPort = wsjtxP
				m.App.Config.Wavelog.Enabled = wlE
				m.App.Config.Wavelog.URL = wlURL
				m.App.Config.Wavelog.APIKey = wlKey
				m.App.Config.Wavelog.StationProfileID = wlSta
				m.App.Config.Wavelog.StationCallsign = wlStaCall
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					applog.Info("Integration config saved, restarting services")
					m.App.MaybeRestartWSJTX()
					cmd = tea.Batch(cmd, m.checkWavelogCmd())
				}
				m.showMainMenu = true
			}
		}
		return m, tea.Batch(cmd, integrationCmd)
	}
	if m.showMainMenu {
		m.mainMenu.width = m.width
		m.mainMenu.height = m.height
		_, mainCmd := m.mainMenu.Update(msg)
		cmd = tea.Batch(cmd, mainCmd)
		if m.mainMenu.action != "" {
			action := m.mainMenu.action
			m.mainMenu.action = ""
			m.showMainMenu = false
			switch action {
			case "general":
				m.configMenu = NewGeneralMenu(m.App.Config)
				m.showConfig = true
			case "callbook":
				m.callbookMenu = NewCallbookMenu(m.App.Config)
				m.showCallbook = true
			case "logbook":
				m.chooser = NewLogbookChooser(m.App, m.toasts)
				m.showChooser = true
			case "rig":
				m.rigChooser = NewRigChooser(m.App, m.toasts)
				m.showRigEdit = true
			case "integration":
				m.integrationMenu = NewIntegrationMenu(m.App.Config)
				m.showIntegration = true
			}
		}
		if m.mainMenu.done {
			m.showMainMenu = false
		}
		return m, cmd
	}
	if m.showPartner {
		// partner view uses m.width/m.height directly, no sub-model
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height
			return m, cmd
		case tea.KeyMsg:
			switch msg.String() {
			case "f2":
				// Close partner view and fall through — QSO form handler won't re-open
				// because partnerData is already set, so it just toggles back to form
				m.showPartner = false
				return m, cmd
			case "f1":
				m.showPartner = false
				return m, cmd
			case "f8":
				m.showPartner = false
				m.mainMenu = NewMainMenu()
				m.showMainMenu = true
				return m, cmd
			default:
				return m, cmd
			}
		}
	}
	if m.showLogbookEditor {
		m.logbookEditor.width = m.width
		m.logbookEditor.height = m.height
		_, editorCmd := m.logbookEditor.Update(msg)
		if em, ok := msg.(editorMsg); ok {
			if em.err != nil && em.wlQSOID == 0 {
				m.toasts.Error(em.err.Error())
			}
			if em.deleted != 0 {
				m.toasts.Success(fmt.Sprintf("QSO %d deleted", em.deleted))
			}
			if em.saved != 0 {
				m.toasts.Success(fmt.Sprintf("QSO %d saved", em.saved))
			}
			if em.purged {
				m.toasts.Success("Logbook purged")
			}
			if em.wlQSOID != 0 {
				if em.wlOK {
					m.toasts.Success(fmt.Sprintf("Wavelog: %s sent", em.wlCall))
					m.logbookEditor.UpdateWLStatus(em.wlQSOID, "yes")
					m.logbookEditor.needsReload = true
				} else {
					m.toasts.Warn(fmt.Sprintf("Wavelog: %s failed", em.wlCall))
					m.logbookEditor.UpdateWLStatus(em.wlQSOID, "no")
				}
			}
		}
		if m.logbookEditor.needsReload {
			m.logbookEditor.needsReload = false
			qsos, _ := store.ListAllQSOs(m.App.DB)
			m.logbookEditor.SetQSOS(qsos)
			m.needRefresh = true
		}
		if m.logbookEditor.done {
			m.showLogbookEditor = false
			m.needRefresh = true
		}
		return m, tea.Batch(cmd, editorCmd)
	}
	if m.showLogView {
		m.logViewer.width = m.width
		m.logViewer.height = m.height
		_, logCmd := m.logViewer.Update(msg)
		cmd = tea.Batch(cmd, logCmd)
		if m.logViewer.done {
			m.showLogView = false
		}
		return m, cmd
	}
	switch msg := msg.(type) {
	case qrzResultMsg:
		m.fillQRZData(msg)
		return m, cmd
	case inetResultMsg:
		m.inetOnline = bool(msg)
		return m, nil
	case tea.KeyMsg:
		switch {
		case m.retainFocused:
			switch msg.String() {
			case " ", "enter":
				m.retainComment = !m.retainComment
			case "tab", "down":
				m.nextField()
			case "shift+tab", "up":
				m.prevField()
			case "ctrl+r":
				m.retainComment = !m.retainComment
			}
			return m, cmd
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab:
			m.prevField()
		case msg.Type == tea.KeyUp || msg.String() == "up":
			m.prevField()
		case msg.Type == tea.KeyDown || msg.String() == "down":
			m.nextField()
		case msg.String() == "ctrl+s":
			return m, m.saveQSO()
		case msg.String() == "delete" || msg.Type == tea.KeyDelete:
			m.clearForm()
		case msg.String() == "ctrl+r":
			m.retainComment = !m.retainComment
		case msg.String() == "ctrl+c":
			m.mainMenu = NewMainMenu()
			m.showMainMenu = true
		case msg.String() == "f1":
			m.focusField(fieldCall)
		case msg.String() == "f2":
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" {
				m.showPartner = true
			}
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled && m.partnerData == nil {
				return m, m.qrzLookup(call)
			}
		case msg.String() == "insert" || msg.Type == tea.KeyInsert || msg.String() == "\x1b[2~":
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled {
				return m, m.qrzLookup(call)
			}
		case msg.String() == "tab" || msg.String() == "\t" || msg.Type == tea.KeyTab:
			m.nextField()
		case msg.String() == "enter":
			return m, m.saveQSO()
		case msg.Type == tea.KeyPgUp || msg.String() == "pgup":
			m.cycleFieldUp()
		case msg.Type == tea.KeyPgDown || msg.String() == "pgdown":
			m.cycleFieldDown()
		default:
			m.updateFocused(msg)
		}
	}
	if m.needRefresh {
		m.needRefresh = false
		return m, tea.Batch(cmd, m.refreshQSOS())
	}
	if m.qrzNeed {
		m.qrzNeed = false
		call := m.qrzCall
		if call == "" {
			return m, cmd
		}
		if !m.App.Config.QRZEnabled {
			return m, cmd
		}
		if m.App.Config.QRZUser == "" {
			m.toasts.Warn("QRZ not configured — F8 Config → Callbook / QRZ.com to enable")
			return m, cmd
		}
		return m, tea.Batch(cmd, m.qrzLookup(call))
	}
	return m, cmd
}

func (m *Model) fillQRZData(msg qrzResultMsg) {
	if msg.Call == "" {
		return
	}
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		return
	}
	if !m.App.Config.QRZEnabled || m.App.Config.QRZUser == "" {
		m.toasts.Warn("QRZ not configured")
		return
	}
	if msg.Err != nil {
		m.toasts.Error("QRZ error: " + msg.Err.Error())
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" {
		m.toasts.Warn("QRZ: no data for " + msg.Call)
		return
	}
	m.partnerData = d
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" {
		m.fields[fieldGrid].SetValue(formatLocator(d.Grid))
	}
	if d.QTH != "" {
		m.fields[fieldQTH].SetValue(d.QTH)
	}
	if d.Country != "" && m.fields[fieldCountry].Value() == "" {
		m.fields[fieldCountry].SetValue(d.Country)
	}
	m.autoFillRST()
	m.toasts.Info("QRZ: " + d.Callsign + " " + d.Name)
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err))
	}
	if m.width < 75 || m.height < 24 {
		msg := fmt.Sprintf("Terminal too small: %dx%d (min 75x24)\n\nPress F10 to quit", m.width, m.height)
		if m.confirmQuit {
			msg = "Quit CQOps? (y/N)"
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(msg)
	}
	w := m.width
	if w < 40 {
		w = 80
	}
	header := m.renderHeader(w)

	var content string
	if m.confirmQuit {
		content = titleStyle.Render("Quit CQOps? (y/N)")
	} else if m.showChooser {
		content = m.chooser.View()
	} else if m.showRigEdit {
		content = m.rigChooser.View()
	} else if m.showConfig {
		content = m.configMenu.View()
	} else if m.showCallbook {
		content = m.callbookMenu.View()
	} else if m.showIntegration {
		content = m.integrationMenu.View()
	} else if m.showMainMenu {
		content = m.mainMenu.View()
	} else if m.showLogView {
		content = m.logViewer.View()
	} else if m.showLogbookEditor {
		content = m.logbookEditor.View()
	} else if m.showPartner && (m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != "") {
		content = m.viewPartner()
	} else {
		content = m.buildQSOFormContent(w, header)
	}

	if content == "" {
		return ""
	}

	toastBar := RenderToasts(m.toasts.Active(), w)
	footer := m.viewFooter(w)

	return m.layoutFrame(header, content, toastBar, footer)
}

func (m *Model) buildQSOFormContent(w int, header string) string {
	headerLines := strings.Count(header, "\n") + 1
	const toastReserved = 2
	const footerLines = 1
	maxBodyH := m.height - headerLines - toastReserved - footerLines
	if maxBodyH < 8 {
		maxBodyH = 8
	}

	form := m.viewForm(w)
	distLine := m.formDistanceLine(w)
	formBlock := strings.TrimRight(form, "\n")
	if distLine != "" {
		formBlock = formBlock + "\n" + distLine
	}
	formRendered := lipgloss.NewStyle().Width(w).Padding(0, 1).Render(formBlock)

	// Hide QSO table on very small terminals
	if m.height < 26 {
		return formRendered
	}

	formLines := strings.Count(formRendered, "\n") + 1
	qsoRows := maxBodyH - formLines - 2
	if qsoRows < 0 {
		qsoRows = 0
	}
	qsoList := m.viewQSOS(qsoRows)
	qsoRendered := lipgloss.NewStyle().Width(w).Padding(0, 1).Render(qsoList)

	return formRendered + "\n" + qsoRendered
}

// layoutFrame assembles header, content, toasts, footer into a fixed-height frame.
// Header at top, toasts+footer pinned to bottom, content in between, filler if needed.
func (m *Model) layoutFrame(header, content, toastBar, footer string) string {
	headerLines := strings.Count(header, "\n") + 1
	footerLines := strings.Count(footer, "\n") + 1

	// Build toast block: exactly 2 rows
	const toastReserved = 2
	toastLines := strings.Split(toastBar, "\n")
	if toastBar == "" {
		toastLines = nil
	}
	for len(toastLines) < toastReserved {
		toastLines = append(toastLines, "")
	}
	toastBlock := strings.Join(toastLines, "\n")

	// Trim content: strip trailing newlines, cap line count
	content = strings.TrimRight(content, "\n")
	contentLines := strings.Split(content, "\n")
	if len(contentLines) == 0 {
		contentLines = []string{""}
	}

	// Maximum body space: terminal height minus everything else
	maxBodyH := m.height - headerLines - toastReserved - footerLines
	if maxBodyH < 1 {
		maxBodyH = 1
	}

	// Cap content lines, then fill remaining with blank lines
	if len(contentLines) > maxBodyH {
		contentLines = contentLines[:maxBodyH]
	}
	filler := maxBodyH - len(contentLines)

	var b strings.Builder
	b.WriteString(strings.Join(contentLines, "\n"))
	for i := 0; i < filler; i++ {
		b.WriteString("\n")
	}
	body := b.String()

	return lipgloss.JoinVertical(lipgloss.Left, header, body) + "\n" + toastBlock + "\n" + footer
}

func (m *Model) renderHeader(width int) string {
	s := m.App.Logbook.Station
	now := time.Now()
	utc := now.UTC()

	rigName := s.RigName
	if rigName == "" {
		rigName = "default"
	}
	rigModel := ""
	rigIndicator := ""
	if rp, ok := m.App.Config.Rigs[rigName]; ok {
		rigModel = rp.Model
		if rp.FlrigEnabled {
			if m.rigConnected {
				if m.rigBlink {
					rigIndicator = SuccessStyle.Render(" on")
				} else {
					rigIndicator = "   "
				}
			} else {
				rigIndicator = ErrorStyle.Render(" err")
			}
		}
	}

	locator := formatLocator(s.Grid)
	if locator == "" {
		locator = "----"
	}

	inetVal := ErrorStyle.Render("no ")
	if m.inetOnline {
		inetVal = SuccessStyle.Render("yes")
	}

	left := LabelStyle.Render("My Call ") + ValueStyle.Render(clamp(s.Callsign, 8))
	left += LabelStyle.Render("  Logbook ") + ValueStyle.Render(clamp(m.App.LogbookName, 8))

	center := TitleStyle.UnsetPadding().Render("CQOPS")
	if v := version.Resolved(); v != "dev" {
		center += SubtleStyle.Render(" v" + v)
	}

	right := LabelStyle.Render("Internet ") + inetVal
	if m.App.Config.WSJTX.Enabled {
		wVal := ErrorStyle.Render("err")
		if m.wsjtxOnline {
			wVal = SuccessStyle.Render("on")
		}
		right += LabelStyle.Render("  WSJT-X ") + wVal
	}
	right += LabelStyle.Render("  Rig ") + ValueStyle.Render(rigModel) + rigIndicator
	if m.App.Config.Wavelog.Enabled {
		wlVal := ErrorStyle.Render("err")
		if m.wlOnline {
			wlVal = SuccessStyle.Render("on")
		}
		right += LabelStyle.Render("  Wavelog ") + wlVal
	}
	rightFully := right +
		LabelStyle.Render("  LT ") + ValueStyle.Render(now.Format("15:04")) +
		LabelStyle.Render("  UTC ") + ValueStyle.Render(utc.Format("15:04:05"))
	rightCompact := right +
		LabelStyle.Render("  utc: ") + ValueStyle.Render(utc.Format("15:04:05"))

	leftW := lipgloss.Width(left)

	// Try full right side first, then without local time, then two rows
	rightStr := rightFully
	rightW := lipgloss.Width(rightFully)
	if leftW+rightW+6 > width {
		rightStr = rightCompact
		rightW = lipgloss.Width(rightCompact)
	}
	if leftW+rightW+6 > width {
		// Two-row fallback (no background on header)
		line1 := "  " + left
		for lipgloss.Width(line1) < width {
			line1 += " "
		}

		line2 := rightFully
		for lipgloss.Width(line2) < width-2 {
			line2 = " " + line2
		}
		line2 = "  " + line2
		for lipgloss.Width(line2) < width {
			line2 += " "
		}

		tabLine := m.renderTabLine(width)
		return line1 + "\n" + line2 + "\n" + tabLine
	}

	gap := width - 4 - leftW - rightW
	if gap < 2 {
		gap = 2
	}

	statusLine := "  " + left + strings.Repeat(" ", gap) + rightStr
	for lipgloss.Width(statusLine) < width {
		statusLine += " "
	}

	tabLine := m.renderTabLine(width)
	return statusLine + "\n" + tabLine
}

func (m *Model) renderTabLine(width int) string {
	active := ActiveTabStyle
	inactive := InactiveTabStyle
	disabled := DisabledTabStyle

	hasPartner := m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""

	labels := []struct {
		label      string
		show       bool
		isActive   bool
		canDisable bool
	}{
		{"F1 QSO Form", true, !m.showPartner && !m.showLogbookEditor && !m.showMainMenu && !m.showLogView, false},
		{"F2 Partner Details", true, m.showPartner && hasPartner, !hasPartner},
		{"F5 Log Editor", true, m.showLogbookEditor, false},
		{"F8 Config", true, m.showMainMenu, false},
		{"F9 Logs", true, m.showLogView, false},
	}

	var parts []string
	for _, t := range labels {
		if !t.show {
			continue
		}
		if t.isActive {
			parts = append(parts, active.Render(t.label))
		} else if t.canDisable && !t.isActive {
			parts = append(parts, disabled.Render(t.label))
		} else {
			parts = append(parts, inactive.Render(t.label))
		}
	}

	line := " " + strings.Join(parts, "")
	for lipgloss.Width(line) < width {
		line += " "
	}
	return BarStyle.Render(line)
}

func clamp(s string, w int) string {
	if s == "" {
		return strings.Repeat(" ", w)
	}
	if lipgloss.Width(s) > w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func stripNonDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatDate(adif string) string {
	if len(adif) < 8 {
		return "—"
	}
	return adif[0:4] + "-" + adif[4:6] + "-" + adif[6:8]
}

func formatTime(adif string) string {
	if len(adif) < 6 {
		return "—"
	}
	return adif[0:2] + ":" + adif[2:4] + ":" + adif[4:6]
}

func (m *Model) viewPartner() string {
	d := m.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d.Callsign == "" {
			return ""
		}
	}
	w := m.width
	h := m.height
	bodyW := w - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder

	title := "── Partner: " + d.Callsign + " "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")

	info := m.renderPartnerInfo(d, bodyW)
	b.WriteString(info)

	// Photo: left-aligned, one empty row above, matching colors
	if d.ImageURL != "" {
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render("  Photo "))
		b.WriteString(DimStyle.Render(d.ImageURL))
	}

	dl := m.partnerDistanceLineForm(d)
	if dl != "" {
		b.WriteString("\n\n")
		pathTitle := "── Short Path "
		b.WriteString(section(pathTitle, bodyW))
		b.WriteString("\n  ")
		b.WriteString(inputStyle.Render(dl))
	}

	usedLines := strings.Count(b.String(), "\n") + 1
	availMapH := h - 4 - usedLines - 2
	if availMapH >= 6 {
		b.WriteString("\n\n")
		ownGrid := m.App.Logbook.Station.Grid
		partnerGrid := d.Grid
		partnerLat, partnerLon := 0.0, 0.0
		hasPartnerLoc := false

		if ownGrid == "" {
			b.WriteString(section("── Map ", bodyW))
			b.WriteString("\n  ")
			b.WriteString(DimStyle.Render("Set your grid in station config to enable the map"))
		} else if partnerGrid == "" && d.Lat == "" {
			b.WriteString(section("── Map ", bodyW))
			b.WriteString("\n  ")
			b.WriteString(DimStyle.Render("No partner location — enter a grid or use QRZ lookup"))
		} else {
			ownLat, ownLon := gridToLatLon(ownGrid)
			if partnerGrid != "" {
				partnerLat, partnerLon = gridToLatLon(partnerGrid)
				hasPartnerLoc = partnerLat != 0 || partnerLon != 0
			}
			if !hasPartnerLoc && d.Lat != "" {
				partnerLat = parseCoord(d.Lat)
				partnerLon = parseCoord(d.Lon)
				hasPartnerLoc = partnerLat != 0 || partnerLon != 0
			}
			if hasPartnerLoc || ownLat != 0 || ownLon != 0 {
				mapW := bodyW
				if mapW < 40 {
					mapW = 40
				}
				mapH := availMapH - 1
				if mapH < 4 {
					mapH = 4
				}
				if mapH > 24 {
					mapH = 24
				}
				b.WriteString(section("── Map ", bodyW))
				b.WriteString("\n")
				mapStr := renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, mapH)
				if mapStr != "" {
					b.WriteString(mapStr)
				} else {
					b.WriteString("  ")
					b.WriteString(DimStyle.Render("Map unavailable"))
				}
			} else {
				b.WriteString(section("── Map ", bodyW))
				b.WriteString("\n  ")
				b.WriteString(DimStyle.Render("Could not determine coordinates"))
			}
		}
	}

	return b.String()
}

// partnerDistanceLineForm uses the form's grid field when QRZ data is unavailable.
func (m *Model) partnerDistanceLineForm(d *qrz.CallData) string {
	own := formatLocator(m.App.Logbook.Station.Grid)
	partner := ""
	if d != nil && d.Grid != "" {
		partner = formatLocator(d.Grid)
	}
	if partner == "" {
		partner = formatLocator(strings.TrimSpace(m.fields[fieldGrid].Value()))
	}
	return distanceLine(own, partner, m.App.Config.DistanceUnit)
}

// formPartnerData builds a CallData from the current QSO form fields.
func (m *Model) formPartnerData() *qrz.CallData {
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if call == "" {
		return nil
	}
	return &qrz.CallData{
		Callsign: call,
		Name:     strings.TrimSpace(m.fields[fieldName].Value()),
		Grid:     strings.TrimSpace(m.fields[fieldGrid].Value()),
		QTH:      strings.TrimSpace(m.fields[fieldQTH].Value()),
		Country:  strings.TrimSpace(m.fields[fieldCountry].Value()),
	}
}

func (m *Model) renderPartnerInfo(d *qrz.CallData, maxW int) string {
	type row struct{ label, value string }
	var rows []row
	add := func(label, value string) {
		if value != "" {
			rows = append(rows, row{label, value})
		}
	}
	add("Callsign", d.Callsign)
	add("Name", d.Name)
	add("Grid", d.Grid)
	add("QTH", d.QTH)
	add("Country", d.Country)
	add("State", d.State)
	add("County", d.County)
	add("Zip", d.Zip)
	add("Class", d.Class)
	add("Email", d.Email)
	add("URL", d.URL)
	if d.Lat != "" || d.Lon != "" {
		add("Coordinates", strings.TrimSpace(d.Lat+" "+d.Lon))
	}
	add("DXCC", d.DXCC)
	add("CQ Zone", d.CQZone)
	add("ITU Zone", d.ITUZone)

	if len(rows) == 0 {
		return ""
	}

	labelW := 13
	indentW := 2
	spaceW := 1
	valW := maxW - indentW - labelW - spaceW
	if valW < 8 {
		valW = 8
	}

	lblStyle := LabelStyle
	valStyle := ValueStyle

	var b strings.Builder
	for _, r := range rows {
		v := valStyle.Render(truncate(r.value, valW))
		b.WriteString(lblStyle.Render(fmt.Sprintf("%s%-*s", strings.Repeat(" ", indentW), labelW, r.label)))
		b.WriteString(strings.Repeat(" ", spaceW))
		b.WriteString(v)
		b.WriteString("\n")
	}
	return b.String()
}

func (m *Model) viewFooter(width int) string {
	var text string
	switch {
	case m.showMainMenu:
		text = m.mainMenu.FooterText()
	case m.showConfig:
		text = m.configMenu.FooterText()
	case m.showCallbook:
		text = m.callbookMenu.FooterText()
	case m.showIntegration:
		text = m.integrationMenu.FooterText()
	case m.showChooser:
		text = m.chooser.FooterText()
	case m.showRigEdit:
		text = m.rigChooser.FooterText()
	case m.showLogView:
		text = "↑↓ to scroll  F10 Quit"
	case m.showLogbookEditor:
		text = m.logbookEditor.FooterText()
	case m.showPartner && (m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""):
		text = "F10 Quit"
	default:
		if width < 70 {
			text = "Enter=Save | Del Clear | Ins/Ctrl+L Lookup | PgUp/Dn Cycle | Space=Mark Retain | F10 Quit"
		} else {
			text = "Enter/Ctrl+S Save  Del Clear  Ins/Ctrl+L Lookup  PgUp/Dn Cycle  Space=Mark Retain  F10 Quit"
		}
	}
	ver := ""
	if v := version.Resolved(); v != "dev" {
		ver = "CQOPS v" + v
	}
	helpStr := HelpStyle.Render(text)
	verStr := DimStyle.Render(ver)
	helpW := lipgloss.Width(helpStr)
	verW := lipgloss.Width(verStr)
	innerW := width - 4
	if helpW+verW > innerW {
		line := "  " + helpStr
		for lipgloss.Width(line) < width {
			line += " "
		}
		return BarStyle.Render(line)
	}
	gap := innerW - helpW - verW
	line := "  " + helpStr + strings.Repeat(" ", gap) + verStr + "  "
	for lipgloss.Width(line) < width {
		line += " "
	}
	return BarStyle.Render(line)
}

func truncate(s string, max int) string {
	if max < 3 {
		return s
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func (m *Model) viewForm(width int) string {
	bodyW := width - 2
	if bodyW < 20 {
		bodyW = 20
	}
	dim := DimStyle
	hl := CursorStyle
	choiceFields := map[field]bool{fieldBand: true, fieldMode: true, fieldSubmode: true}

	// Three columns with equal space: left 7, middle 6, right 5 (comment is special, spans cols 1+2)
	leftFields := []field{fieldDate, fieldTime, fieldCall, fieldFreq, fieldBand, fieldMode, fieldSubmode}
	middleFields := []field{fieldRSTSent, fieldRSTRcvd, fieldName, fieldQTH, fieldGrid, fieldCountry}
	rightFields := []field{fieldTXPower, fieldFreqRx, fieldSOTA, fieldPOTA, fieldWWFF, fieldIOTA}

	colW := (bodyW - 4) / 3 // 4 = two 2-char gaps between three columns
	if colW < 20 {
		colW = bodyW // fallback to single column on very narrow terminals
	}

	renderField := func(f field, w int) string {
		label := fieldNames[f]
		raw := strings.TrimSpace(m.fields[f].Value())
		lbl := fmt.Sprintf("%-12s", label)

		choiceIcon := ""
		if choiceFields[f] {
			choiceIcon = dim.Render("▼ ")
		}

		isFocused := int(f) == int(m.focus) && !m.retainFocused
		// Use textinput.View() so the cursor appears on focused fields
		tiView := m.fields[f].View()
		val := choiceIcon
		if isFocused {
			val += tiView
		} else if raw == "" {
			val += dim.Render("—")
		} else {
			val += ValueStyle.Render(raw)
		}

		line := " " + lbl + " " + val
		if isFocused {
			line = hl.Render(" "+lbl) + " " + val
		}
		// Pad to column width
		for lipgloss.Width(line) < w {
			line += " "
		}
		return line
	}

	var b strings.Builder

	// Build right-side info for QSO section header (operator + my grid + my refs)
	s := m.App.Logbook.Station
	var infoParts []string
	if s.Operator != "" {
		infoParts = append(infoParts, "Operator: "+s.Operator)
	}
	if s.Grid != "" {
		infoParts = append(infoParts, "My Grid: "+formatLocator(s.Grid))
	}
	if s.SOTARef != "" {
		infoParts = append(infoParts, "My SOTA: "+s.SOTARef)
	}
	if s.POTARef != "" {
		infoParts = append(infoParts, "My POTA: "+s.POTARef)
	}
	if s.WWFFRef != "" {
		infoParts = append(infoParts, "My WWFF: "+s.WWFFRef)
	}

	// Build header: left title + dashes + right info, dynamically hide info when no space
	title := "── QSO "
	titleW := lipgloss.Width(title)
	infoStr := ""
	if len(infoParts) > 0 {
		// Try fitting all parts, drop from the end if too wide
		for try := len(infoParts); try >= 1; try-- {
			candidate := strings.Join(infoParts[:try], "  ")
			candW := lipgloss.Width(candidate)
			gap := 2
			if titleW+candW+gap <= bodyW || try == 1 {
				infoStr = candidate
				break
			}
		}
	}

	if infoStr != "" {
		infoW := lipgloss.Width(infoStr)
		gap := 2
		dashes := bodyW - titleW - infoW - gap
		if dashes < 1 {
			dashes = 1
		}
		b.WriteString(SectionStyle.Render(title + strings.Repeat("─", dashes) + strings.Repeat(" ", gap) + DimStyle.Render(infoStr)))
	} else {
		b.WriteString(section(title, bodyW))
	}
	b.WriteString("\n")

	rows := len(leftFields)
	if len(middleFields) > rows {
		rows = len(middleFields)
	}
	if len(rightFields) > rows {
		rows = len(rightFields)
	}
	for i := 0; i < rows; i++ {
		left := ""
		if i < len(leftFields) {
			left = renderField(leftFields[i], colW)
		}
		middle := ""
		if i < len(middleFields) {
			middle = renderField(middleFields[i], colW)
		}
		right := ""
		if i < len(rightFields) {
			right = renderField(rightFields[i], colW)
		}
		if colW >= 20 {
			parts := []string{}
			if left != "" {
				parts = append(parts, left)
			}
			if middle != "" {
				parts = append(parts, middle)
			}
			if right != "" {
				parts = append(parts, right)
			}
			b.WriteString(strings.Join(parts, "  "))
			b.WriteString("\n")
		} else {
			if left != "" {
				b.WriteString(left)
				b.WriteString("\n")
			}
			if middle != "" {
				b.WriteString(middle)
				b.WriteString("\n")
			}
			if right != "" {
				b.WriteString(right)
				b.WriteString("\n")
			}
		}
	}

	// Comment row: spans columns 1+2, with Retain checkbox in column 3
	commentW := colW*2 + 2 // two columns + gap
	if commentW < 20 {
		commentW = bodyW
	}
	commentLine := renderField(fieldComment, commentW)

	// Retain checkbox
	retainMark := "[ ]"
	retainLabel := "Retain"
	if m.retainComment {
		retainMark = "[x]"
	}
	if m.retainFocused {
		retainBox := hl.Render(" "+retainMark) + " " + inputStyle.Render(retainLabel)
		for lipgloss.Width(retainBox) < colW {
			retainBox += " "
		}
		if colW >= 20 {
			b.WriteString(commentLine)
			b.WriteString("  ")
			b.WriteString(retainBox)
			b.WriteString("\n")
		} else {
			b.WriteString(commentLine)
			b.WriteString("\n")
			b.WriteString("  ")
			b.WriteString(retainBox)
			b.WriteString("\n")
		}
	} else {
		if m.retainComment {
			retainBox := " " + inputStyle.Render(retainMark) + " " + dim.Render(retainLabel)
			for lipgloss.Width(retainBox) < colW {
				retainBox += " "
			}
			if colW >= 20 {
				b.WriteString(commentLine)
				b.WriteString("  ")
				b.WriteString(retainBox)
				b.WriteString("\n")
			} else {
				b.WriteString(commentLine)
				b.WriteString("\n")
				b.WriteString("  ")
				b.WriteString(retainBox)
				b.WriteString("\n")
			}
		} else {
			retainBox := " " + dim.Render(retainMark) + " " + dim.Render(retainLabel)
			for lipgloss.Width(retainBox) < colW {
				retainBox += " "
			}
			if colW >= 20 {
				b.WriteString(commentLine)
				b.WriteString("  ")
				b.WriteString(retainBox)
				b.WriteString("\n")
			} else {
				b.WriteString(commentLine)
				b.WriteString("\n")
				b.WriteString("  ")
				b.WriteString(retainBox)
				b.WriteString("\n")
			}
		}
	}

	return b.String()
}

type qsoCol struct {
	header   string
	minWidth int
	grow     bool
	value    func(q *qso.QSO) string
}

var qsoAllCols = map[string]qsoCol{
	"Date": {"Date", 10, false, func(q *qso.QSO) string { return formatDate(q.QSODate) }},
	"Time": {"Time", 8, false, func(q *qso.QSO) string { return formatTime(q.TimeOn) }},
	"Call": {"Call", 7, true, func(q *qso.QSO) string { return q.Call }},
	"Band": {"Band", 5, false, func(q *qso.QSO) string {
		b := qso.NormalizeBand(q.Band)
		if b == "" && q.Freq > 0 {
			b = fmt.Sprintf("%.1f", q.Freq)
		}
		return b
	}},
	"Mode":    {"Mode", 5, false, func(q *qso.QSO) string { return q.Mode }},
	"RSTs":    {"RSTs", 4, false, func(q *qso.QSO) string { return q.RSTSent }},
	"RSTr":    {"RSTr", 4, false, func(q *qso.QSO) string { return q.RSTRcvd }},
	"ID":      {"ID", 3, false, func(q *qso.QSO) string { return fmt.Sprintf("%d", q.ID) }},
	"DXCC":    {"DXCC", 6, true, func(q *qso.QSO) string { return q.Country }},
	"Sub":     {"Sub", 4, false, func(q *qso.QSO) string { return q.Submode }},
	"Name":    {"Name", 7, true, func(q *qso.QSO) string { return q.Name }},
	"Grid":    {"Grid", 6, false, func(q *qso.QSO) string { return q.GridSquare }},
	"QTH":     {"QTH", 8, true, func(q *qso.QSO) string { return q.QTH }},
	"Comment": {"Comment", 10, true, func(q *qso.QSO) string { return q.Comment }},
	"Dist": {"Dist", 6, false, func(q *qso.QSO) string {
		if q.Distance > 0 {
			return fmt.Sprintf("%.0f", q.Distance)
		}
		return ""
	}},
}

var qsoColTiers = []struct {
	minW  int
	names []string
}{
	{0, []string{"Date", "Time", "Call", "Mode", "RSTs", "RSTr"}},
	{52, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr"}},
	{65, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "ID", "DXCC"}},
	{85, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "ID", "DXCC", "Sub", "Name"}},
	{105, []string{"Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "ID", "DXCC", "Sub", "Name", "Grid", "QTH", "Comment", "Dist"}},
}

func selectQSOCols(width int) []qsoCol {
	var names []string
	for _, t := range qsoColTiers {
		if width >= t.minW {
			names = t.names
		}
	}
	cols := make([]qsoCol, len(names))
	for i, n := range names {
		cols[i] = qsoAllCols[n]
	}
	if width >= 130 {
		totalMin := 0
		growCount := 0
		for _, c := range cols {
			totalMin += c.minWidth + 1
			if c.grow {
				growCount++
			}
		}
		extra := width - totalMin
		if extra > 0 && growCount > 0 {
			perGrow := extra / growCount
			for i := range cols {
				if cols[i].grow {
					cols[i].minWidth += perGrow
				}
			}
		}
	}
	return cols
}

func (m *Model) viewQSOS(maxRows int) string {
	dim := DimStyle
	bodyW := m.width - 2
	if bodyW < 20 {
		bodyW = 20
	}

	var b strings.Builder
	// Section header with Wavelog station (left) and total count (right)
	var leftInfo string
	if m.App.Config.Wavelog.Enabled && m.wlStationName != "" {
		leftInfo = "Wavelog: " + m.wlStationName
		if m.wlStationLabel != "" {
			leftInfo += " (" + m.wlStationLabel + ")"
		}
	}
	totalStr := fmt.Sprintf("Total: %d", len(m.qsos))
	title := "── Recent QSOs "
	totalW := lipgloss.Width(totalStr)
	titleW := lipgloss.Width(title)
	leftW := 0
	if leftInfo != "" {
		leftW = lipgloss.Width(leftInfo) + 2 // 2 spaces before it
	}
	gap := 2
	dashes := bodyW - titleW - leftW - totalW - gap
	if dashes < 1 {
		dashes = 1
	}
	hdr := SectionStyle.Render(title + strings.Repeat("─", dashes))
	if leftInfo != "" {
		hdr += "  " + DimStyle.Render(leftInfo)
	}
	hdr += strings.Repeat(" ", gap) + DimStyle.Render(totalStr)
	b.WriteString(hdr)
	b.WriteString("\n")

	cols := selectQSOCols(bodyW)

	var headerParts []string
	var fmtParts []string
	for _, c := range cols {
		headerParts = append(headerParts, c.header)
		fmtParts = append(fmtParts, fmt.Sprintf("%%-%ds", c.minWidth))
	}

	headerFmt := strings.Join(fmtParts, " ")
	headerLine := headerStyle.Render(fmt.Sprintf(headerFmt, toAny(headerParts)...))
	b.WriteString(headerLine)
	b.WriteString("\n")

	if len(m.qsos) == 0 {
		for i := 0; i < maxRows; i++ {
			emptyRow := make([]string, len(cols))
			for j := range emptyRow {
				emptyRow[j] = "—"
			}
			b.WriteString(dim.Render(fmt.Sprintf(headerFmt, toAny(emptyRow)...)))
			b.WriteString("\n")
		}
	} else {
		limit := maxRows
		if limit > len(m.qsos) {
			limit = len(m.qsos)
		}
		for i := 0; i < limit; i++ {
			q := m.qsos[i]
			var vals []string
			for _, c := range cols {
				v := c.value(&q)
				if v == "" {
					v = "—"
				}
				v = trunc(v, c.minWidth)
				vals = append(vals, v)
			}
			r := fmt.Sprintf(headerFmt, toAny(vals)...)
			if i%2 == 0 {
				r = inputStyle.Render(r)
			}
			b.WriteString(r)
			b.WriteString("\n")
		}
		for i := limit; i < maxRows; i++ {
			emptyRow := make([]string, len(cols))
			for j := range emptyRow {
				emptyRow[j] = "—"
			}
			b.WriteString(dim.Render(fmt.Sprintf(headerFmt, toAny(emptyRow)...)))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func toAny(ss []string) []any {
	aa := make([]any, len(ss))
	for i, s := range ss {
		aa[i] = s
	}
	return aa
}

func trunc(s string, w int) string {
	if s == "" {
		return ""
	}
	if len(s) > w {
		return s[:w]
	}
	return s
}

func (m *Model) formDistanceLine(width int) string {
	ownGrid := formatLocator(m.App.Logbook.Station.Grid)
	partnerGrid := formatLocator(strings.TrimSpace(m.fields[fieldGrid].Value()))
	if ownGrid == "" {
		return ""
	}
	if partnerGrid == "" {
		return ""
	}
	dl := distanceLine(ownGrid, partnerGrid, m.App.Config.DistanceUnit)
	if dl == "" {
		return ""
	}
	bodyW := width - 2
	if bodyW < 20 {
		bodyW = 20
	}

	// Build rig/antenna info for right side of header
	s := m.App.Logbook.Station
	rigInfo := ""
	if s.Rig != "" {
		rigInfo = "Rig: " + s.Rig
	}
	if s.Antenna != "" {
		if rigInfo != "" {
			rigInfo += "  "
		}
		rigInfo += "Ant: " + s.Antenna
	}

	title := "── Short Path "
	titleW := lipgloss.Width(title)
	if rigInfo != "" {
		rigW := lipgloss.Width(rigInfo)
		gap := 2
		dashes := bodyW - titleW - rigW - gap
		if dashes < 1 {
			dashes = 1
		}
		hdr := SectionStyle.Render(title + strings.Repeat("─", dashes) + strings.Repeat(" ", gap) + DimStyle.Render(rigInfo))
		return hdr + "\n  " + inputStyle.Render(dl)
	}

	hdr := section(title, bodyW)
	return hdr + "\n  " + inputStyle.Render(dl)
}

func (m *Model) focusField(f field) {
	m.retainFocused = false
	m.fields[m.focus].Blur()
	m.focus = f
	m.fields[m.focus].Focus()
}

func (m *Model) nextField() {
	wasCall := m.focus == fieldCall

	if m.retainFocused {
		// Retain checkbox → first field
		m.retainFocused = false
		m.focus = 0
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == fieldComment {
		// Comment is last text field — Tab goes to Retain checkbox
		m.retainFocused = true
	} else {
		m.focus = (m.focus + 1) % fieldCount
		m.fields[m.focus].Focus()
	}
	if wasCall {
		m.qrzNeed = true
		m.qrzCall = strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		m.autoFillRST()
		m.autoFillSSBSubmode()
	}
}
func (m *Model) prevField() {
	wasCall := m.focus == fieldCall

	if m.retainFocused {
		// Retain checkbox → go back to Comment field
		m.retainFocused = false
		m.focus = fieldComment
		m.fields[m.focus].Focus()
		return
	}

	m.fields[m.focus].Blur()
	if m.focus == 0 {
		// First field → go to Retain checkbox (not wrapping to Comment)
		m.retainFocused = true
	} else {
		m.focus--
		m.fields[m.focus].Focus()
	}
	if wasCall {
		m.qrzNeed = true
		m.qrzCall = strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		m.autoFillRST()
		m.autoFillSSBSubmode()
	}
}
func (m *Model) autoFillRST() {
	if m.fields[fieldRSTSent].Value() != "" || m.fields[fieldRSTRcvd].Value() != "" {
		return
	}
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	if mode == "CW" {
		m.fields[fieldRSTSent].SetValue("599")
		m.fields[fieldRSTRcvd].SetValue("599")
	} else {
		m.fields[fieldRSTSent].SetValue("59")
		m.fields[fieldRSTRcvd].SetValue("59")
	}
}

// applyFreqDefaults derives band → mode → submode from the frequency field.
// Called whenever frequency or band changes.
func (m *Model) applyFreqDefaults() {
	freqStr := strings.TrimSpace(m.fields[fieldFreq].Value())
	if freqStr == "" {
		return
	}
	var freq float64
	fmt.Sscanf(freqStr, "%f", &freq)
	if freq <= 0 {
		return
	}

	// Step 1: freq → band
	band := qso.DeriveBand(freq)
	if band != "" {
		m.fields[fieldBand].SetValue(band)
	}

	// Step 2: band → mode
	low, _, _ := qso.BandRange(band)
	if low >= 50 {
		m.fields[fieldMode].SetValue("FM")
	} else {
		m.fields[fieldMode].SetValue("SSB")
	}

	// Step 3: mode + freq → submode
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	switch mode {
	case "SSB":
		if freq < 10.0 {
			m.fields[fieldSubmode].SetValue("LSB")
		} else {
			m.fields[fieldSubmode].SetValue("USB")
		}
	case "FM":
		m.fields[fieldSubmode].SetValue("")
	}
}

func (m *Model) autoFillSSBSubmode() {
	// Keep for backward compat — calls the new pipeline
	m.applyFreqDefaults()
}

func (m *Model) updateFocused(msg tea.KeyMsg) {
	if m.retainFocused {
		return
	}
	prevCall := strings.TrimSpace(m.fields[fieldCall].Value())
	prevVal := m.fields[m.focus].Value()
	prevFreq := m.fields[fieldFreq].Value()
	m.fields[m.focus], _ = m.fields[m.focus].Update(msg)

	// Frequency changed: derive band → mode → submode
	if m.focus == fieldFreq && m.fields[fieldFreq].Value() != prevFreq {
		m.applyFreqDefaults()
	}
	// Band changed: derive mode → submode from the new band
	if m.focus == fieldBand && m.fields[m.focus].Value() != prevVal {
		m.applyFreqDefaults()
	}
	// Mode/submode changed manually: let user override, don't auto-set
	if (m.focus == fieldDate || m.focus == fieldTime) && m.fields[m.focus].Value() != prevVal {
		m.dateTimeAuto = false
	}
	if m.focus == fieldCall {
		m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value()))
	}
	if m.focus == fieldGrid {
		m.fields[m.focus].SetValue(formatLocator(m.fields[m.focus].Value()))
	}
	if m.focus == fieldCall {
		cur := strings.TrimSpace(m.fields[fieldCall].Value())
		if cur != prevCall {
			// Callsign changed — clear stale QRZ data and related form fields
			if m.partnerData != nil && !strings.EqualFold(m.partnerData.Callsign, cur) {
				m.partnerData = nil
				m.showPartner = false
			}
			m.fields[fieldName].SetValue("")
			m.fields[fieldGrid].SetValue("")
			m.fields[fieldQTH].SetValue("")
			m.fields[fieldCountry].SetValue("")
		}
	}
}
func (m *Model) clearForm() {
	// Preserve comment if retain is on
	retainedComment := ""
	if m.retainComment {
		retainedComment = m.fields[fieldComment].Value()
	}

	rig := [5]struct {
		idx   field
		value string
	}{
		{fieldBand, m.fields[fieldBand].Value()},
		{fieldFreq, m.fields[fieldFreq].Value()},
		{fieldMode, m.fields[fieldMode].Value()},
		{fieldSubmode, m.fields[fieldSubmode].Value()},
		{fieldTXPower, m.fields[fieldTXPower].Value()},
	}

	for i := field(0); i < fieldCount; i++ {
		m.fields[i].SetValue("")
		m.fields[i].Blur()
	}
	now := time.Now().UTC()
	m.fields[fieldDate].SetValue(now.Format("2006-01-02"))
	m.fields[fieldTime].SetValue(now.Format("15:04:05"))

	for _, r := range rig {
		if r.value != "" {
			m.fields[r.idx].SetValue(r.value)
		}
	}
	// Restore retained comment
	if retainedComment != "" {
		m.fields[fieldComment].SetValue(retainedComment)
	}
	m.dateTimeAuto = true
	m.retainFocused = false
	m.focus = fieldCall
	m.fields[m.focus].Focus()
	m.partnerData = nil
	m.showPartner = false
}

func (m *Model) cycleFieldUp() {
	switch m.focus {
	case fieldBand:
		m.cycleBand(1)
	case fieldMode:
		m.cycleMode(1)
	case fieldSubmode:
		m.cycleSubmode(1)
	}
}

func (m *Model) cycleFieldDown() {
	switch m.focus {
	case fieldBand:
		m.cycleBand(-1)
	case fieldMode:
		m.cycleMode(-1)
	case fieldSubmode:
		m.cycleSubmode(-1)
	}
}

func (m *Model) cycleBand(dir int) {
	b := strings.ToUpper(strings.TrimSpace(m.fields[fieldBand].Value()))
	b = qso.NormalizeBand(b)
	list := qso.AllBands()
	idx := indexOfStr(list, b)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldBand].SetValue(list[idx])
	m.autoFillSSBSubmode()
}

func (m *Model) cycleMode(dir int) {
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	mode, _ = qso.NormalizeMode(mode, "")
	if !qso.IsValidMode(mode) {
		mode = ""
	}
	list := qso.AllModes()
	idx := indexOfStr(list, mode)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldMode].SetValue(list[idx])
}

func (m *Model) cycleSubmode(dir int) {
	cur := strings.ToUpper(strings.TrimSpace(m.fields[fieldSubmode].Value()))
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	mode, _ = qso.NormalizeMode(mode, "")
	list := qso.SubmodesFor(mode)
	if len(list) == 0 {
		m.fields[fieldSubmode].SetValue("")
		return
	}
	idx := indexOfStr(list, cur)
	idx += dir
	if idx < 0 {
		idx = len(list) - 1
	} else if idx >= len(list) {
		idx = 0
	}
	m.fields[fieldSubmode].SetValue(list[idx])
}

func indexOfStr(list []string, s string) int {
	for i, v := range list {
		if strings.EqualFold(v, s) {
			return i
		}
	}
	return -1
}
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
func (m *Model) refreshQSOS() tea.Cmd {
	qsos, err := store.ListQSOs(m.App.DB, 30)
	if err != nil {
		m.toasts.Error(fmt.Sprintf("Refresh failed: %v", err))
		return nil
	}
	m.qsos = qsos
	return nil
}
