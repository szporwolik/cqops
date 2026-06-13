package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

type screenKind int

const (
	screenQSO screenKind = iota
	screenPartner
	screenMainMenu
	screenConfig
	screenCallbook
	screenIntegration
	screenChooser
	screenRigEdit
	screenLogView
	screenLogbookEditor
)

type Model struct {
	App             *app.App
	screen          screenKind
	fields          [fieldCount]textinput.Model
	focus           field
	qsos            []qso.QSO
	toasts          *ToastQueue
	err             error
	width           int
	height          int
	quitting        bool
	rigConnected    bool
	rigFreq         float64
	rigMode         string
	rigPower        float64
	rigBlink        bool
	rigSkipTicks    int
	rigPolling      bool
	dateTimeAuto    bool
	tickCount       int
	inetOnline      bool
	wsjtxOnline     bool
	wsjtxStatus     string
	needRefresh     bool
	pendingADIF     string
	pendingStatus   statusPending
	adifMu          sync.Mutex
	chooser         *LogbookChooser
	rigChooser      *RigChooser
	configMenu      *GeneralMenu
	callbookMenu    *CallbookMenu
	integrationMenu *IntegrationMenu
	mainMenu        *MainMenu
	logViewer       *LogViewer
	logbookEditor   *LogbookEditor
	confirmQuit     bool
	confirm         *DialogModel // active confirmation dialog (quit, etc.)
	partnerData     *qrz.CallData
	flrigClient     *flrig.Client
	qrzNeed         bool
	qrzCall         string
	qrzLastLook     time.Time
	retainComment   bool
	retainFocused   bool // true when the Retain checkbox has focus (instead of a text field)
	wlOnline        bool
	wlStationName   string // e.g. "JO30oo / DJ7NT"
	wlStationLabel  string // e.g. "Debug location"
	keys            KeyMap
	help            help.Model
	vp              viewport.Model
	recentQSOs      *RecentQSOs // read-only Recent QSOs view
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
	m := &Model{App: a, qsos: initialQSOS, toasts: NewToastQueue(), dateTimeAuto: true, width: 80, height: 24}
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
	m.keys = DefaultKeyMap()
	m.help = help.New()
	m.vp = viewport.New(viewport.WithWidth(80), viewport.WithHeight(10))
	m.recentQSOs = NewRecentQSOs(initialQSOS)
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

// hideAllSubmodels returns to the QSO form screen.
func (m *Model) hideAllSubmodels() {
	m.screen = screenQSO
}

// isSubmodelActive returns true when any sub-screen is visible.
func (m *Model) isSubmodelActive() bool {
	return m.screen != screenQSO
}

// saveConfig persists the app configuration and shows a toast.
func (m *Model) saveConfig(msg string) {
	if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
		m.toasts.Error("Settings save failed: " + err.Error())
	} else {
		if msg != "" {
			m.toasts.Success(msg)
		} else {
			m.toasts.Success("Settings saved")
		}
		applog.Info("Settings saved")
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle WindowSizeMsg first — store dimensions, then let sub-models see it too
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		// Viewport: available height = terminal - status(1) - tabs(1) - help(1) - toasts(2) - form(~18)
		vpHeight := m.height - 1 - 1 - 1 - 2 - 18
		if vpHeight < 3 {
			vpHeight = 3
		}
		m.vp.SetWidth(wsm.Width - 2)
		m.vp.SetHeight(vpHeight)
	}

	// Active confirmation dialog — checked first, routes all keys to dialog
	if m.confirm != nil {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			updated, _ := m.confirm.Update(msg)
			*m.confirm = updated.(DialogModel)
			if m.confirm.Done() {
				if m.confirm.Result.Confirmed && m.confirm.Result.Value == "quit" {
					return m, tea.Quit
				}
				m.confirm = nil
			}
		}
		return m, cmd
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
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, m.keys.Quit):
			applog.Debug("tab: F10 quit requested")
			dlg := NewDialog("Quit CQOPS", "Exit the application?",
				Option{Label: "Quit", Value: "quit"},
				Option{Label: "Cancel", Value: "cancel"},
			)
			m.confirm = &dlg
			m.screen = screenQSO
			return m, cmd
		case key.Matches(keyMsg, m.keys.QSOForm):
			applog.Debug("tab: F1 QSO Form")
			m.screen = screenQSO
		case key.Matches(keyMsg, m.keys.Partner):
			if m.screen != screenPartner {
				applog.Debug("tab: F2 Partner Details")
				m.screen = screenPartner
			}
		case key.Matches(keyMsg, m.keys.Config):
			if m.screen == screenMainMenu {
				applog.Debug("tab: F8 close Config")
				m.screen = screenQSO
			} else {
				applog.Debug("tab: F8 Config")
				m.mainMenu = NewMainMenu()
				m.screen = screenMainMenu
			}
		case key.Matches(keyMsg, m.keys.LogEditor):
			applog.Debug("tab: F5 Log Editor")
			m.logbookEditor = NewLogbookEditor(m.App.DB, m.App.Config.Wavelog.URL, m.App.Config.Wavelog.APIKey, m.App.Config.Wavelog.StationProfileID, m.App.Config.Wavelog.StationCallsign, m.App.Logbook.Station.Operator, m.App.Logbook.Station.Grid)
			qsos, _ := store.ListAllQSOs(m.App.DB)
			m.logbookEditor.SetQSOS(qsos)
			m.screen = screenLogbookEditor
			return m, cmd
		case key.Matches(keyMsg, m.keys.Logs):
			applog.Debug("tab: F9 Log Viewer")
			m.logViewer = NewLogViewer(m.App.LogbookName)
			m.screen = screenLogView
			return m, cmd
		}
		if !m.isSubmodelActive() {
			if key.Matches(keyMsg, m.keys.Delete) {
				m.clearForm()
				return m, nil
			}
			if key.Matches(keyMsg, m.keys.Lookup) {
				call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
				if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled {
					return m, m.qrzLookup(call)
				}
			}
		}
	}
	switch m.screen {
	case screenChooser:
		m.chooser.width = m.width
		m.chooser.height = m.height
		_, chooserCmd := m.chooser.Update(msg)
		cmd = tea.Batch(cmd, chooserCmd)
		if m.chooser.done {
			m.screen = screenMainMenu
			m.needRefresh = true
		}
		return m, cmd
	case screenRigEdit:
		m.rigChooser.width = m.width
		m.rigChooser.height = m.height
		_, rigCmd := m.rigChooser.Update(msg)
		cmd = tea.Batch(cmd, rigCmd)
		if m.rigChooser.done {
			m.screen = screenMainMenu
			m.refreshFlrigClient()
		}
		return m, cmd
	case screenConfig:
		m.configMenu.width = m.width
		m.configMenu.height = m.height
		_, configCmd := m.configMenu.Update(msg)
		cmd = tea.Batch(cmd, configCmd)
		if m.configMenu.done {
			m.screen = screenQSO
			if m.configMenu.goBack {
				m.screen = screenMainMenu
			}
			if m.configMenu.saved {
				m.App.Config.DistanceUnit = m.configMenu.distanceUnit
				m.saveConfig("Settings saved")
				m.screen = screenMainMenu
			}
		}
		return m, cmd
	case screenCallbook:
		m.callbookMenu.width = m.width
		m.callbookMenu.height = m.height
		m.callbookMenu.inetOnline = m.inetOnline
		_, callbookCmd := m.callbookMenu.Update(msg)
		if m.callbookMenu.done {
			m.screen = screenQSO
			if m.callbookMenu.goBack {
				m.screen = screenMainMenu
			}
			if m.callbookMenu.saved {
				m.App.Config.QRZUser = m.callbookMenu.user.Value()
				m.App.Config.QRZPass = m.callbookMenu.pass.Value()
				m.App.Config.QRZEnabled = m.callbookMenu.enabled
				m.saveConfig("Settings saved")
				m.screen = screenMainMenu
			}
		}
		return m, tea.Batch(cmd, callbookCmd)
	case screenIntegration:
		m.integrationMenu.width = m.width
		m.integrationMenu.height = m.height
		_, integrationCmd := m.integrationMenu.Update(msg)
		if m.integrationMenu.done {
			m.screen = screenQSO
			if m.integrationMenu.goBack {
				m.screen = screenMainMenu
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
				m.saveConfig("Settings saved")
				applog.Info("Integration config saved, restarting services")
				m.App.MaybeRestartWSJTX()
				cmd = tea.Batch(cmd, m.checkWavelogCmd())
				m.screen = screenMainMenu
			}
		}
		return m, tea.Batch(cmd, integrationCmd)
	case screenMainMenu:
		m.mainMenu.width = m.width
		m.mainMenu.height = m.height
		_, mainCmd := m.mainMenu.Update(msg)
		cmd = tea.Batch(cmd, mainCmd)
		if m.mainMenu.action != "" {
			action := m.mainMenu.action
			m.mainMenu.action = ""
			m.screen = screenQSO
			switch action {
			case "general":
				m.configMenu = NewGeneralMenu(m.App.Config)
				m.screen = screenConfig
			case "callbook":
				m.callbookMenu = NewCallbookMenu(m.App.Config)
				m.screen = screenCallbook
			case "logbook":
				m.chooser = NewLogbookChooser(m.App, m.toasts)
				m.screen = screenChooser
			case "rig":
				m.rigChooser = NewRigChooser(m.App, m.toasts)
				m.screen = screenRigEdit
			case "integration":
				m.integrationMenu = NewIntegrationMenu(m.App.Config)
				m.screen = screenIntegration
			}
		}
		if m.mainMenu.done {
			m.screen = screenQSO
		}
		return m, cmd
	case screenPartner:
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height
			return m, cmd
		case tea.KeyPressMsg:
			switch msg.String() {
			case "f1", "esc":
				m.screen = screenQSO
				return m, cmd
			case "f8":
				m.screen = screenQSO
				m.mainMenu = NewMainMenu()
				m.screen = screenMainMenu
				return m, cmd
			default:
				return m, cmd
			}
		}
	case screenLogbookEditor:
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
			m.screen = screenQSO
			m.needRefresh = true
		}
		return m, tea.Batch(cmd, editorCmd)
	case screenLogView:
		m.logViewer.width = m.width
		m.logViewer.height = m.height
		_, logCmd := m.logViewer.Update(msg)
		cmd = tea.Batch(cmd, logCmd)
		if m.logViewer.done {
			m.screen = screenQSO
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
	case tea.KeyPressMsg:
		switch {
		case m.retainFocused:
			switch msg.String() {
			case "space", "enter":
				m.retainComment = !m.retainComment
			case "tab", "down":
				m.nextField()
			case "shift+tab", "up":
				m.prevField()
			case "ctrl+r":
				m.retainComment = !m.retainComment
			}
			return m, cmd
		case key.Matches(msg, m.keys.PrevField):
			m.prevField()
		case key.Matches(msg, m.keys.NextField):
			m.nextField()
		case key.Matches(msg, m.keys.Save):
			return m, m.saveQSO()
		case key.Matches(msg, m.keys.Delete):
			m.clearForm()
		case key.Matches(msg, m.keys.Retain):
			m.retainComment = !m.retainComment
		case msg.String() == "ctrl+c":
			m.mainMenu = NewMainMenu()
			m.screen = screenMainMenu
		case key.Matches(msg, m.keys.FocusCall):
			m.focusField(fieldCall)
		case key.Matches(msg, m.keys.Partner):
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" {
				m.screen = screenPartner
			}
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled && m.partnerData == nil {
				return m, m.qrzLookup(call)
			}
		case key.Matches(msg, m.keys.Lookup):
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled {
				return m, m.qrzLookup(call)
			}
		case key.Matches(msg, m.keys.Enter):
			return m, m.saveQSO()
		case key.Matches(msg, m.keys.CycleUp):
			m.cycleFieldUp()
		case key.Matches(msg, m.keys.CycleDown):
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

func (m *Model) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}
	if m.err != nil {
		return tea.NewView(errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)))
	}

	// Measure all fixed zones and calculate content area dimensions
	layout := MeasureLayout(m)

	if layout.TerminalW < 75 || layout.TerminalH < 24 {
		msg := fmt.Sprintf("Terminal too small: %dx%d (min 75x24)\n\nPress F10 to quit",
			layout.TerminalW, layout.TerminalH)
		return tea.NewView(lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(msg))
	}

	// 1. Status bar
	statusBar := m.renderStatusBar()

	// 2. Profile/context line — right-aligned above tabs for readability
	profileLine := m.renderProfileBar()

	// 3. Tab bar
	tabBar := m.renderTabBar()

	var body string
	body = m.buildBodyForScreen(layout)
	if body == "" {
		body = DimStyle.Render("\u2014")
	}

	helpBar := m.renderHelpBar()

	// Clip body to ContentH so it never pushes the help bar off-screen.
	// No .Height() — we don't pad; the table fills exact remaining space.
	body = lipgloss.NewStyle().MaxHeight(layout.ContentH).Render(body)

	// Build the main view without toasts or dialogs (those are composited on top)
	mainView := lipgloss.JoinVertical(lipgloss.Left,
		statusBar,
		profileLine,
		tabBar,
		body,
		helpBar,
	)

	// Composite confirm dialog as a centered overlay if active
	if m.confirm != nil {
		mainView = RenderDialogOverlay(mainView, *m.confirm, layout.TerminalW, layout.TerminalH)
	}

	// Composite toasts as a floating overlay in the bottom-right corner.
	// Clip mainView to terminal height first so the compositor canvas
	// matches the visible terminal exactly.
	mainView = lipgloss.NewStyle().MaxHeight(layout.TerminalH).Render(mainView)
	finalView := RenderToastOverlay(mainView, m.toasts.Active(), layout.TerminalW, layout.TerminalH)

	v := tea.NewView(finalView)
	v.AltScreen = true
	v.WindowTitle = m.windowTitle()
	return v
}

func (m *Model) headerView() string {
	s := m.App.Logbook.Station
	utc := time.Now().UTC()

	callsign := s.Callsign
	if callsign == "" {
		callsign = "—"
	}
	logName := m.App.LogbookName
	if logName == "" {
		logName = "—"
	}

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		S.StatusApp.Render(" CQOPS v"+version.Resolved()+" "),
		S.StatusLabel.Render(" Call "),
		S.StatusValue.Render(clamp(callsign, 8)),
		S.StatusLabel.Render(" Log "),
		S.StatusValue.Render(clamp(logName, 10)),
	)

	right := lipgloss.JoinHorizontal(lipgloss.Top,
		statusDotStyled(m.inetOnline, "Net"),
		statusDotStyled(m.wsjtxOnline, "WSJT"),
		statusDotStyled(m.rigConnected, "Rig"),
		statusDotStyled(m.wlOnline, "WL"),
		S.StatusRight.Render(" "),
		S.StatusTime.Render(utc.Format("15:04:05")),
	)

	fillerW := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if fillerW < 1 {
		fillerW = 1
	}

	return left + S.StatusFill.Render(strings.Repeat(" ", fillerW)) + right
}

func statusDotStyled(on bool, label string) string {
	dot := S.StatusDotOff.Render("●")
	if on {
		dot = S.StatusDotOn.Render("●")
	}
	return dot + S.StatusRight.Render(label) + S.StatusRight.Render(" ")
}

func (m *Model) tabView() string {
	hasPartner := m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != ""

	type tab struct {
		label    string
		active   bool
		disabled bool
	}
	tabs := []tab{
		{"F1 QSO Form", m.screen == screenQSO && m.confirm == nil, false},
		{"F2 Partner", m.screen == screenPartner && hasPartner, !hasPartner},
		{"F5 Log Editor", m.screen == screenLogbookEditor, false},
		{"F8 Config", m.screen == screenMainMenu || m.screen == screenConfig || m.screen == screenCallbook || m.screen == screenIntegration || m.screen == screenChooser || m.screen == screenRigEdit, false},
		{"F9 Logs", m.screen == screenLogView, false},
	}

	var parts []string
	for _, t := range tabs {
		s := S.TabInactive
		if t.active {
			s = S.TabActive
		}
		if t.disabled {
			s = S.TabDisabled
		}
		parts = append(parts, s.Render(" "+t.label+" "))
	}

	row := strings.Join(parts, S.TabGap.Render(" "))
	return S.TabBar.Width(m.width).Render(row)
}

func (m *Model) renderProfileLine() string {
	s := m.App.Logbook.Station
	var parts []string
	if s.Operator != "" {
		parts = append(parts, "Op "+s.Operator)
	}
	if s.Rig != "" {
		parts = append(parts, "Rig "+s.Rig)
	}
	if s.Antenna != "" {
		parts = append(parts, "Ant "+s.Antenna)
	}
	if s.Grid != "" {
		parts = append(parts, "Grid "+formatLocator(s.Grid))
	}
	if s.SOTARef != "" {
		parts = append(parts, "SOTA "+s.SOTARef)
	}
	if s.POTARef != "" {
		parts = append(parts, "POTA "+s.POTARef)
	}
	if s.WWFFRef != "" {
		parts = append(parts, "WWFF "+s.WWFFRef)
	}
	if len(parts) == 0 {
		return ""
	}
	return DimStyle.Render("  " + strings.Join(parts, " · "))
}

func (m *Model) helpView() string {
	if m.confirm != nil {
		return HelpStyle.Render("←/→ choose  •  enter confirm  •  esc cancel")
	}
	bindings := m.ActiveBindings()
	if len(bindings) == 0 {
		bindings = []key.Binding{m.keys.Quit}
	}
	helpText := m.help.ShortHelpView(bindings)
	if helpText == "" {
		helpText = m.help.ShortHelpView([]key.Binding{m.keys.Quit})
	}
	return HelpStyle.Render(helpText)
}

// =============================================================================
// Layout wrapper methods — called by layout.MeasureLayout and View().
// These are the canonical entry points for rendering each fixed zone.
// =============================================================================

func (m *Model) renderStatusBar() string { return m.headerView() }
func (m *Model) renderTabBar() string    { return m.tabView() }
func (m *Model) renderHelpBar() string   { return m.helpView() }

func (m *Model) renderProfileBar() string {
	if !m.isSubmodelActive() && m.confirm == nil {
		line := m.renderProfileLine()
		if line == "" {
			return ""
		}
		// Right-align, single line — content beyond width is hidden, not wrapped
		return lipgloss.NewStyle().
			Width(m.width).
			MaxHeight(1).
			Align(lipgloss.Right).
			Render(line)
	}
	return ""
}

func (m *Model) renderToastBar() string {
	return RenderToasts(m.toasts.Active(), m.width)
}

// windowTitle returns the terminal window title for the main TUI.
func (m *Model) windowTitle() string {
	s := m.App.Logbook.Station
	callsign := s.Callsign
	logbook := m.App.LogbookName
	if callsign == "" && logbook == "" {
		return "CQOPS"
	}
	if callsign == "" {
		return "CQOPS — " + logbook
	}
	if logbook == "" {
		return "CQOPS — " + callsign
	}
	return "CQOPS — " + callsign + " @ " + logbook
}

// buildBodyForScreen returns the content string for the active screen,
// using Layout dimensions for proper sizing.
func (m *Model) buildBodyForScreen(l Layout) string {
	switch m.screen {
	case screenQSO:
		return m.buildQSOFormWithLayout(l)
	case screenPartner:
		if m.partnerData != nil || strings.TrimSpace(m.fields[fieldCall].Value()) != "" {
			return m.viewPartner()
		}
	case screenMainMenu:
		return m.mainMenu.View().Content
	case screenConfig:
		return m.configMenu.View().Content
	case screenCallbook:
		return m.callbookMenu.View().Content
	case screenIntegration:
		return m.integrationMenu.View().Content
	case screenChooser:
		return m.chooser.View().Content
	case screenRigEdit:
		return m.rigChooser.View().Content
	case screenLogView:
		return m.logViewer.View().Content
	case screenLogbookEditor:
		return m.logbookEditor.View().Content
	}
	return ""
}

// buildQSOFormWithLayout renders the QSO form, short path info, and recent
// QSOs using layout-derived dimensions. The short path gets its own bordered
// box between the form and the table for visual separation.
func (m *Model) buildQSOFormWithLayout(l Layout) string {
	innerW := l.ContentW - 4 // 2 border + 2 padding (from S.QSOFormBox)
	if innerW < 20 {
		innerW = l.ContentW
	}

	// QSO form in its bordered box
	form := m.viewForm(innerW)
	formBlock := strings.TrimRight(form, "\n")
	formBox := S.QSOFormBox.Width(l.ContentW).Render(formBlock)

	// Short path info (no border — just a clean dim line between form and table)
	spacer := ""
	pathBox := ""
	distLine := m.formDistanceLine(innerW)
	if distLine != "" {
		spacer = "\n"
		pathBox = DimStyle.Width(l.ContentW).Padding(0, 1).Render(distLine)
	}

	formRenderedH := lipgloss.Height(formBox)
	pathRenderedH := lipgloss.Height(pathBox)
	// Give all remaining space to the table — the "\n" separators are
	// included in the measured component heights.
	recentH := l.ContentH - formRenderedH - pathRenderedH
	if recentH < 3 {
		recentH = 3
	}

	m.recentQSOs.SetSize(l.ContentW, recentH)

	return formBox + spacer + pathBox + "\n" + m.recentQSOs.View()
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

	usedLines := lipgloss.Height(b.String())
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
	valW := maxW - indentW - labelW - 1 // 1 for gap
	if valW < 8 {
		valW = 8
	}

	lblStyle := LabelStyle.Width(labelW).Align(lipgloss.Right)
	valStyle := ValueStyle.Width(valW)

	var lines []string
	for _, r := range rows {
		label := lblStyle.Render(r.label)
		value := valStyle.Render(truncate(r.value, valW))
		line := strings.Repeat(" ", indentW) + label + " " + value
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) viewForm(width int) string {
	// width is the exact available space inside the border (already accounts for padding)
	bodyW := width
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
		lbl := S.FormLabel.Align(lipgloss.Left).Render(label)

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
			val += SubtleStyle.Render("\u2014") // visible placeholder
		} else {
			val += ValueStyle.Render(raw)
		}

		line := " " + lbl + " " + val
		if isFocused {
			line = hl.Render(" "+lbl) + " " + val
		}
		// Pad to column width using lipgloss
		return lipgloss.NewStyle().Width(w).Render(line)
	}

	var b strings.Builder

	rows := len(leftFields)
	if len(middleFields) > rows {
		rows = len(middleFields)
	}
	if len(rightFields) > rows {
		rows = len(rightFields)
	}
	for i := 0; i < rows; i++ {
		var cols []string
		if i < len(leftFields) {
			cols = append(cols, renderField(leftFields[i], colW))
		}
		if i < len(middleFields) {
			cols = append(cols, renderField(middleFields[i], colW))
		}
		if i < len(rightFields) {
			cols = append(cols, renderField(rightFields[i], colW))
		}
		if colW >= 20 {
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cols...))
		} else {
			b.WriteString(strings.Join(cols, "\n"))
		}
		b.WriteString("\n")
	}

	// Comment row spans columns 1+2; Retain checkbox in column 3
	commentW := colW*2 + 2
	if commentW < 20 {
		commentW = bodyW
	}
	commentLine := renderField(fieldComment, commentW)

	retainBox := m.renderRetainCheckbox(colW)
	if colW >= 20 {
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, commentLine, retainBox))
	} else {
		b.WriteString(commentLine)
		b.WriteString("\n")
		b.WriteString(retainBox)
	}
	b.WriteString("\n")

	return b.String()
}

func (m *Model) renderRetainCheckbox(colW int) string {
	mark := "[ ]"
	label := "Retain"
	if m.retainComment {
		mark = "[x]"
	}
	if m.retainFocused {
		return lipgloss.NewStyle().Width(colW).Render(
			CursorStyle.Render(" "+mark) + " " + inputStyle.Render(label),
		)
	}
	if m.retainComment {
		return lipgloss.NewStyle().Width(colW).Render(
			" " + inputStyle.Render(mark) + " " + DimStyle.Render(label),
		)
	}
	return lipgloss.NewStyle().Width(colW).Render(
		" " + DimStyle.Render(mark) + " " + DimStyle.Render(label),
	)
}

func (m *Model) formDistanceLine(width int) string {
	ownGrid := formatLocator(m.App.Logbook.Station.Grid)
	partnerGrid := formatLocator(strings.TrimSpace(m.fields[fieldGrid].Value()))

	line := ""
	if ownGrid != "" && partnerGrid != "" {
		line = distanceLine(ownGrid, partnerGrid, m.App.Config.DistanceUnit)
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Foreground(P.Info).
		Render(line)
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

func (m *Model) updateFocused(msg tea.KeyPressMsg) {
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
				m.screen = screenQSO
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
	m.screen = screenQSO
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
