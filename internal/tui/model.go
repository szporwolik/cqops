package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/textinput"
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
	confirm         *DialogModel // active confirmation dialog (quit, etc.)
	partnerData     *qrz.CallData
	wlPrivateData   *wavelog.PrivateLookupResult // Wavelog callsign lookup
	wlLastBand      string                       // band used in last WL query
	wlLastMode      string                       // mode used in last WL query
	flrigClient     *flrig.Client
	qrzNeed         bool
	qrzCall         string
	qrzLastLook     time.Time
	qrzLastCall     string // last looked-up callsign
	wlNeed          bool   // re-trigger WL lookup after band/mode change
	wlCall          string // callsign for pending WL lookup
	wlLastLook      time.Time
	wlLastCall      string // last looked-up callsign
	retainComment   bool
	retainFocused   bool // true when the Retain checkbox has focus (instead of a text field)
	wlOnline        bool
	wlStationName   string // e.g. "JO30oo / DJ7NT"
	wlStationLabel  string // e.g. "Debug location"
	keys            KeyMap
	help            help.Model
	recentQSOs      *RecentQSOs // read-only Recent QSOs view

	// Partner/map rendering cache — avoids expensive ASCII map generation on every View().
	partnerMapCache    string
	partnerMapCacheSig string
}

type tickMsg time.Time
type qrzResultMsg struct {
	Call string
	Data *qrz.CallData
	Err  error
}
type wlResultMsg struct {
	Call string
	Data *wavelog.PrivateLookupResult
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
	if call == "" {
		return nil
	}
	// Skip if same callsign was already looked up recently.
	if time.Since(m.qrzLastLook) < 3*time.Second && strings.EqualFold(call, m.qrzLastCall) {
		return nil
	}
	m.qrzLastLook = time.Now()
	m.qrzLastCall = call
	applog.Info("QRZ: looking up", "call", call)
	return m.qrzLookupCmd(call)
}

func (m *Model) wlLookupCmd(call, band, mode string) tea.Cmd {
	return func() tea.Msg {
		data, err := wavelog.PrivateLookup(
			m.App.Config.Wavelog.URL,
			m.App.Config.Wavelog.APIKey,
			call, band, mode,
		)
		return wlResultMsg{Call: call, Data: data, Err: err}
	}
}

func (m *Model) wlLookup(call string) tea.Cmd {
	if call == "" {
		return nil
	}
	if !m.App.Config.Wavelog.Enabled || m.App.Config.Wavelog.URL == "" || m.App.Config.Wavelog.APIKey == "" {
		return nil
	}
	if !m.inetOnline {
		return nil
	}
	band := strings.TrimSpace(m.fields[fieldBand].Value())
	mode := strings.TrimSpace(m.fields[fieldMode].Value())
	// Skip if same call+band+mode was already queried recently.
	if time.Since(m.wlLastLook) < 5*time.Second &&
		strings.EqualFold(call, m.wlLastCall) &&
		band == m.wlLastBand && mode == m.wlLastMode {
		return nil
	}
	m.wlLastLook = time.Now()
	m.wlLastCall = call
	m.wlLastBand = band
	m.wlLastMode = mode
	applog.Info("Wavelog: looking up", "call", call)
	return m.wlLookupCmd(call, band, mode)
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
			m.wlPrivateData = nil
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

// partnerMapCacheKey computes a cache key from all inputs that affect
// the partner/map rendered output. When this key matches, the cached
// map output can be reused without expensive ASCII generation.
func (m *Model) partnerMapCacheKey() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("w%d|h%d|", m.width, m.height))
	b.WriteString(fmt.Sprintf("own:%s|", m.App.Logbook.Station.Grid))
	if m.partnerData != nil {
		b.WriteString(fmt.Sprintf("p:%s|g:%s|lat:%s|lon:%s|",
			m.partnerData.Callsign,
			m.partnerData.Grid,
			m.partnerData.Lat,
			m.partnerData.Lon))
	}
	return b.String()
}

// invalidatePartnerMapCache clears the partner map cache.
func (m *Model) invalidatePartnerMapCache() {
	m.partnerMapCache = ""
	m.partnerMapCacheSig = ""
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// WindowSizeMsg — store dimensions first; invalidate map cache on resize
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsm.Width
		m.height = wsm.Height
		m.invalidatePartnerMapCache()
	}

	// Active confirmation dialog — highest priority, blocks everything else
	if m.confirm != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
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

	// Tick processing
	if _, ok := msg.(tickMsg); ok {
		cmd = m.handleTick(cmd)
	}

	// Async result messages (internet, Wavelog, flrig)
	if m.handleAsyncMessages(msg) {
		if _, ok := msg.(flrigResultMsg); ok {
			return m, cmd
		}
	}

	// Global function keys (F1-F10, etc.)
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if handledCmd, handled := m.handleGlobalKeys(keyMsg); handled {
			if handledCmd != nil {
				cmd = tea.Batch(cmd, handledCmd)
			}
			return m, cmd
		}
	}

	// Screen-specific routing
	switch m.screen {
	case screenChooser:
		return m.handleChooserUpdate(msg, cmd)
	case screenRigEdit:
		return m.handleRigEditUpdate(msg, cmd)
	case screenConfig:
		return m.handleConfigUpdate(msg, cmd)
	case screenCallbook:
		return m.handleCallbookUpdate(msg, cmd)
	case screenIntegration:
		return m.handleIntegrationUpdate(msg, cmd)
	case screenMainMenu:
		return m.handleMainMenuUpdate(msg, cmd)
	case screenPartner:
		return m.handlePartnerUpdate(msg, cmd)
	case screenLogbookEditor:
		return m.handleLogbookEditorUpdate(msg, cmd)
	case screenLogView:
		return m.handleLogViewUpdate(msg, cmd)
	}

	// QSO form result messages (QRZ, Wavelog lookups)
	switch r := msg.(type) {
	case qrzResultMsg:
		m.fillQRZData(r)
		return m, cmd
	case wlResultMsg:
		m.fillWLData(r)
		return m, cmd
	}

	// QSO form key handling
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if formCmd, handled := m.handleFormKey(keyMsg); handled {
			if formCmd != nil {
				cmd = tea.Batch(cmd, formCmd)
			}
			return m, cmd
		}
	}

	// Deferred pending requests
	if pendingCmd, handled := m.handlePendingRequests(cmd); handled {
		return m, pendingCmd
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
	m.invalidatePartnerMapCache() // new partner data means map must re-render
	if d.Name != "" {
		m.fields[fieldName].SetValue(d.Name)
	}
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" {
		m.fields[fieldGrid].SetValue(formatLocator(d.Grid))
		applog.Debug("QRZ: filled partner grid", "grid", d.Grid)
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

func (m *Model) fillWLData(msg wlResultMsg) {
	if msg.Call == "" {
		return
	}
	formCall := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	if formCall != "" && formCall != strings.ToUpper(msg.Call) {
		return
	}
	if msg.Err != nil {
		applog.Warn("Wavelog: lookup error", "call", msg.Call, "error", msg.Err)
		m.toasts.Warn("Wavelog: " + msg.Err.Error())
		return
	}
	if msg.Data == nil {
		return
	}
	applog.InfoDetail("Wavelog: lookup OK", fmt.Sprintf("call=%s worked=%v confirmed=%v", msg.Call, msg.Data.Worked(), msg.Data.DXCCConfirmed()))
	m.wlPrivateData = msg.Data
	name := ""
	if msg.Data.Name() != "" {
		name = " " + msg.Data.Name()
	}
	m.toasts.Info("Wavelog: " + msg.Call + name)
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

	// Build the main view without toasts or dialogs (those are composited on top).
	// Only include non-empty rows to avoid blank lines from empty zones.
	var mainParts []string
	addRow := func(s string) {
		if s != "" {
			mainParts = append(mainParts, s)
		}
	}
	addRow(statusBar)
	addRow(profileLine)
	addRow(tabBar)
	addRow(body)
	addRow(helpBar)
	mainView := lipgloss.JoinVertical(lipgloss.Left, mainParts...)

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
	w := l.TerminalW // full width, matching the status bar
	innerW := w - 4  // 2 border + 2 padding (from S.QSOFormBox)
	if innerW < 20 {
		innerW = w
	}

	// QSO form in its bordered box
	form := m.viewForm(innerW)
	formBlock := strings.TrimRight(form, "\n")
	formBox := S.QSOFormBox.Width(w).Render(formBlock)

	// Path row in a clean bordered box (always one row — no layout shift).
	pathContent := m.formPathRow(innerW)
	pathBox := S.MapBox.Width(w).Render(pathContent)

	formRenderedH := lipgloss.Height(formBox)
	pathRenderedH := lipgloss.Height(pathBox)
	recentH := l.ContentH - formRenderedH - pathRenderedH
	if recentH < 5 {
		recentH = 5
	}

	// Table fits inside a bordered box — account for 2 rows (top/bottom border).
	tableW := w - 2
	tableH := recentH - 2
	if tableH < 3 {
		tableH = 3
	}
	m.recentQSOs.SetSize(tableW, tableH)

	return formBox + "\n" + pathBox + "\n" + S.RecentQSOsBox.Width(w).Render(m.recentQSOs.View())
}

func (m *Model) viewPartner() string {
	d := m.partnerData
	if d == nil {
		d = m.formPartnerData()
		if d.Callsign == "" {
			return ""
		}
	}
	bodyW := m.width // full terminal width
	if bodyW < 30 {
		bodyW = 30
	}

	// Two-column layout: info box (left half) + placeholder (right half)
	halfW := (bodyW - 1) / 2 // 1-cell gap between columns
	if halfW < 20 {
		halfW = bodyW // fallback to single column on narrow terminals
	}

	info := m.renderPartnerInfo(d, halfW)
	infoBox := S.QSOFormBox.Width(halfW).Render(info)
	infoH := lipgloss.Height(infoBox)

	// Wavelog info (right column) — same label/value style as partner info.
	wlContent := m.renderWLInfo(halfW)
	wlBox := S.QSOFormBox.Width(halfW).Height(infoH).Render(wlContent)

	// Side-by-side columns
	gap := lipgloss.NewStyle().Width(1).Render("")
	var topRow string
	if halfW >= 20 {
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, infoBox, gap, wlBox)
	} else {
		topRow = lipgloss.JoinVertical(lipgloss.Left, infoBox, wlBox)
	}

	// Map section — full width below the columns. The box is always shown.
	mapW := bodyW - 2 // inside border (MapBox has no h-padding, just borders)

	// Available height for the map: terminal minus fixed rows minus top info.
	topH := lipgloss.Height(topRow)
	mapAvailH := contentHeight(m.height) - topH
	if mapAvailH < 3 {
		mapAvailH = 3
	}

	// Use cached map content when the cache signature matches.
	cacheKey := m.partnerMapCacheKey()
	var mapInner string
	if m.partnerMapCacheSig == cacheKey && m.partnerMapCache != "" {
		mapInner = m.partnerMapCache
	} else {
		ownGrid := m.App.Logbook.Station.Grid
		partnerGrid := d.Grid

		switch {
		case ownGrid == "":
			mapInner = DimStyle.Render("Set your grid in station config to enable the map")
		case partnerGrid == "" && d.Lat == "":
			mapInner = DimStyle.Render("No partner location — enter a grid or use QRZ lookup")
		default:
			if mapAvailH >= NativeMapHeight+5 && mapW >= NativeMapWidth {
				ownLat, ownLon := gridToLatLon(ownGrid)
				partnerLat, partnerLon := 0.0, 0.0

				if partnerGrid != "" {
					partnerLat, partnerLon = gridToLatLon(partnerGrid)
				} else if d.Lat != "" {
					partnerLat = parseCoord(d.Lat)
					partnerLon = parseCoord(d.Lon)
				}

				mapStr := renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, NativeMapHeight)
				if mapStr != "" {
					mapInner = mapStr
				} else {
					mapInner = DimStyle.Render("Terminal too small for map")
				}
			} else {
				mapInner = DimStyle.Render("Terminal too small for map")
			}
		}
		m.partnerMapCacheSig = cacheKey
		m.partnerMapCache = mapInner
	}

	mapBox := S.MapBox.Width(bodyW).Render(mapInner)

	// Filler pushes the help bar toward the bottom without overshooting.
	mapBoxH := lipgloss.Height(mapBox)
	fillerH := contentHeight(m.height) - topH - mapBoxH
	if fillerH < 0 {
		fillerH = 0
	}
	filler := lipgloss.NewStyle().Height(fillerH).Render("")
	return lipgloss.JoinVertical(lipgloss.Left, topRow, mapBox, filler)
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
	// Photo as an OSC-8 hyperlink — Ctrl+click opens the image in the browser.
	if d.ImageURL != "" {
		add("Photo", osc8Link(d.ImageURL, "CLICK"))
	}

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
		value := r.value
		// Don't truncate or restyle OSC-8 links — they embed ANSI sequences.
		if r.label == "Photo" {
			value = S.Info.Width(valW).Align(lipgloss.Left).Render(value)
		} else {
			value = valStyle.Render(truncate(r.value, valW))
		}
		indent := lipgloss.NewStyle().Width(indentW).Render("")
		gap := lipgloss.NewStyle().Width(1).Render("")
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, indent, label, gap, value))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m *Model) renderWLInfo(maxW int) string {
	d := m.wlPrivateData
	if d == nil {
		return lipgloss.NewStyle().
			Width(maxW - 4).
			Align(lipgloss.Center).
			Foreground(P.TextMuted).
			Render("WL lookup pending…")
	}

	type row struct{ label, value string }
	var rows []row
	add := func(label string, value bool, yes, no string) {
		if yes == "" {
			yes = "yes"
		}
		if no == "" {
			no = "—"
		}
		v := no
		if value {
			v = yes
		}
		rows = append(rows, row{label, v})
	}

	hasBand := m.wlLastBand != ""
	hasMode := m.wlLastMode != ""

	add("Call worked", d.Worked(), "Y", "N")
	add("Call on band", hasBand && d.WorkedBand(), "Y", tern(hasBand, "N", "?"))
	add("Call on mode", hasBand && hasMode && d.WorkedBandMode(), "Y", tern(hasBand && hasMode, "N", "?"))
	add("LoTW member", d.LoTW(), "Y", "N")
	add("DXCC confirmed", d.DXCCConfirmed(), "Y", "N")
	add("DXCC on band", hasBand && d.ConfirmedBand(), "Y", tern(hasBand, "N", "?"))
	add("DXCC on mode", hasBand && hasMode && d.ConfirmedBandMode(), "Y", tern(hasBand && hasMode, "N", "?"))

	labelW := 15
	indentW := 1
	valW := maxW - indentW - labelW - 1
	if valW < 3 {
		valW = 3
	}

	lblStyle := LabelStyle.Width(labelW).Align(lipgloss.Right)
	yesStyle := S.Success.Width(valW)
	noStyle := S.Error.Width(valW)
	qStyle := S.Warning.Width(valW)

	var lines []string
	for _, r := range rows {
		label := lblStyle.Render(r.label)
		indent := lipgloss.NewStyle().Width(indentW).Render("")
		gap := lipgloss.NewStyle().Width(1).Render("")
		val := r.value
		switch val {
		case "Y":
			val = yesStyle.Render(val)
		case "?":
			val = qStyle.Render(val)
		case "N":
			val = noStyle.Render(val)
		}
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, indent, label, gap, val))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
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
