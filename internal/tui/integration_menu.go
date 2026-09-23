package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/gps"
	"go.bug.st/serial"
)

// integrationRows is the row style shared by the integration menu's rows.
var integrationRows = rowStyle{label: S.FormLabelWide, focused: S.FormFocusedWide}

type IntegrationMenu struct {
	// DXC
	dxcEnabled bool
	dxcHost    textinput.Model
	dxcPort    textinput.Model
	dxcLogin   textinput.Model

	inetOnline bool

	// HTTP Server
	httpEnabled  bool
	httpTheme    int // 0=Bright, 1=Dark, 2=Orchid(YL), 3=HighVis
	httpAddrIdx  int // 0=localhost, 1=0.0.0.0
	httpAddr     textinput.Model
	httpPort     textinput.Model
	httpHeader1  textinput.Model
	httpHeader2  textinput.Model
	httpClubLogo textinput.Model
	httpQRLink   textinput.Model
	httpEvtStart textinput.Model
	httpTLS      bool // serve HTTPS; empty cert/key = auto self-signed
	httpTLSCert  textinput.Model
	httpTLSKey   textinput.Model

	// GPS
	gpsEnabled       bool
	gpsService       int // 0=None, 1=Serial, 2=GPSD
	gpsGridPrecision int // 6, 8, or 10
	gpsPort          textinput.Model
	gpsBaudRate      int
	gpsDTR           bool
	gpsRTS           bool
	gpsdHost         textinput.Model
	gpsdPort         textinput.Model
	gpsTesting       bool
	gpsNeedsPoll     bool // set when test passes — main model picks it up

	// APRS
	aprsEnabled  bool
	aprsService  int // 0=APRS-IS, 1=KISS, 2=KISS Server
	aprsServer   textinput.Model
	aprsKISSHost textinput.Model
	aprsKISSPort textinput.Model
	aprsPort     textinput.Model
	aprsBaudRate int
	aprsDataBits int // 8, 7, 6, 5
	aprsParity   int // 0=None, 1=Odd, 2=Even, 3=Mark, 4=Space
	aprsStopBits int // 0=1, 1=1.5, 2=2
	aprsDTR      bool
	aprsRTS      bool
	aprsTesting  bool
	aprsOnline   bool // true when APRS client is connected (KISS or APRS-IS)

	// PSK Reporter — simple enable toggle, no fields.
	pskEnabled bool

	// aprsToast/gpsToast are set by the APRS/GPS test handlers; the parent
	// reads them, shows a toast, then clears. All test feedback goes through
	// toasts — no inline result lines.
	aprsToast string
	gpsToast  string

	fm         menuFocus
	done       bool
	saved      bool
	goBack     bool
	goCallbook bool
	width      int
	height     int

	// saveError is set when Ctrl+S is blocked by validation.
	// The parent reads it to show a toast, then clears it.
	SaveError string

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastBodyContent string
}

const (
	imDXCChk       = 0
	imDXCHost      = 1
	imDXCPort      = 2
	imDXCLogin     = 3
	imQRZChk       = 4
	imQRZUser      = 5
	imQRZPass      = 6
	imQRZTest      = 7
	imHTTPChk      = 8
	imHTTPAddr     = 9
	imHTTPPort     = 10
	imHTTPTheme    = 11
	imHTTPTLS      = 12
	imHTTPTLSCert  = 13
	imHTTPTLSKey   = 14
	imHTTPHdr1     = 15
	imHTTPHdr2     = 16
	imHTTPLogo     = 17
	imHTTPQRLink   = 18
	imHTTPEvt      = 19
	imGPSChk       = 20
	imGPSSvc       = 21 // service type: None / Serial / GPSD
	imGPSGridPrec  = 22 // grid precision: 10 / 8 / 6
	imGPSPort      = 23 // serial port
	imGPSBaud      = 24 // baud rate
	imGPSDTR       = 25 // DTR
	imGPSRTS       = 26 // RTS
	imGPSDHost     = 27 // GPSD host
	imGPSDPort     = 28 // GPSD port
	imGPSTest      = 29 // test button
	imAPRSChk      = 30
	imAPRSSvc      = 31 // service type: APRS-IS / KISS / KISS Server
	imAPRSServer   = 32 // APRS-IS server host:port
	imAPRSKISSHost = 33 // KISS Server TCP host
	imAPRSKISSPort = 34 // KISS Server TCP port
	imAPRSPort     = 35 // KISS serial port
	imAPRSBaud     = 36 // KISS baud rate
	imAPRSData     = 37 // KISS data bits
	imAPRSParity   = 38 // KISS parity
	imAPRSStop     = 39 // KISS stop bits
	imAPRSDTR      = 40 // KISS DTR
	imAPRSRTS      = 41 // KISS RTS
	imAPRSTest     = 42 // test button
	imPSKChk       = 43
	imMax          = 44
)

type callbookTestMsg struct {
	ok       bool
	err      error
	provider string // "qrz" or "hamqth"
}

type gpsTestMsg struct {
	ok  bool
	err error
}

// gpsServiceOptions maps service index to label.
var gpsServiceOptions = []struct {
	label string
}{
	{"Serial"},
	{"GPSD"},
}

// aprsServiceOptions maps APRS service index to label.
var aprsServiceOptions = []struct {
	label string
}{
	{"APRS-IS"},
	{"KISS"},
	{"KISS Server"},
}

// dataBitsOptions lists available serial data bit widths for cycling.
var dataBitsOptions = []int{8, 7, 6, 5}

// parityOptions maps parity index to label.
var parityOptions = []struct {
	label string
}{
	{"None"},
	{"Odd"},
	{"Even"},
	{"Mark"},
	{"Space"},
}

// stopBitsOptions maps stop bits index to label.
var stopBitsOptions = []struct {
	label string
}{
	{"1"},
	{"1.5"},
	{"2"},
}

// firstLogbookCallsign returns the first non-empty station callsign from any
// logbook (deterministic, sorted by logbook ID). Used to prefill convenience
// fields like the DX Cluster login — the operator can always change it.
func firstLogbookCallsign(cfg *config.Config) string {
	if cfg == nil || len(cfg.Logbooks) == 0 {
		return ""
	}
	ids := make([]string, 0, len(cfg.Logbooks))
	for id := range cfg.Logbooks {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if cs := strings.TrimSpace(cfg.Logbooks[id].Station.Callsign); cs != "" {
			return cs
		}
	}
	return ""
}

// gpsPrecisionOptions lists available grid precision levels for cycling.
var gpsPrecisionOptions = []int{10, 8, 6}

func nextGPSCycleInt(current int, opts []int) int {
	for i, v := range opts {
		if v == current && i+1 < len(opts) {
			return opts[i+1]
		}
	}
	return opts[0]
}

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
	dxcHost := newTextinput()
	dxcHost.CharLimit = 60
	dxcHost.SetWidth(28)
	dxcHost.Placeholder = "dxspots.com"
	if cfg.Integrations.DXC.Host != "" {
		dxcHost.SetValue(cfg.Integrations.DXC.Host)
	} else {
		dxcHost.SetValue("dxspots.com")
	}

	dxcPort := newTextinput()
	dxcPort.CharLimit = 6
	dxcPort.SetWidth(28)
	dxcPort.Placeholder = "7300"
	if cfg.Integrations.DXC.Port != "" {
		dxcPort.SetValue(cfg.Integrations.DXC.Port)
	} else {
		dxcPort.SetValue("7300")
	}

	dxcLogin := newTextinput()
	dxcLogin.CharLimit = 20
	dxcLogin.SetWidth(28)
	dxcLogin.Placeholder = "callsign"
	if cfg.Integrations.DXC.Login != "" {
		dxcLogin.SetValue(cfg.Integrations.DXC.Login)
	} else if cs := firstLogbookCallsign(cfg); cs != "" {
		// Convenience prefill — the operator can change it.
		dxcLogin.SetValue(cs)
	}

	httpAddr := newTextinput()
	httpAddr.CharLimit = 40
	httpAddr.SetWidth(28)
	httpAddr.Placeholder = "0.0.0.0"
	// Map existing config to index: localhost/127.0.0.1 → 0, 0.0.0.0 → 1.
	httpAddrIdx := 0
	addr := cfg.Integrations.HTTPServer.Address
	switch {
	case addr == "" || addr == "localhost" || addr == "127.0.0.1":
		httpAddrIdx = 0
		httpAddr.SetValue("localhost")
	case addr == "0.0.0.0":
		httpAddrIdx = 1
		httpAddr.SetValue("0.0.0.0")
	default:
		httpAddrIdx = 1 // treat unknown as network
		httpAddr.SetValue(addr)
	}

	httpPort := newTextinput()
	httpPort.CharLimit = 6
	httpPort.SetWidth(28)
	httpPort.Placeholder = "8073"
	if cfg.Integrations.HTTPServer.Port != "" {
		httpPort.SetValue(cfg.Integrations.HTTPServer.Port)
	} else {
		httpPort.SetValue("8073")
	}

	httpHeader1 := newTextinput()
	httpHeader1.CharLimit = 60
	httpHeader1.SetWidth(28)
	httpHeader1.Placeholder = "e.g. Club Name"
	if cfg.Integrations.HTTPServer.Header1 != "" {
		httpHeader1.SetValue(cfg.Integrations.HTTPServer.Header1)
	}

	httpHeader2 := newTextinput()
	httpHeader2.CharLimit = 60
	httpHeader2.SetWidth(28)
	httpHeader2.Placeholder = "e.g. Field Day 2026"
	if cfg.Integrations.HTTPServer.Header2 != "" {
		httpHeader2.SetValue(cfg.Integrations.HTTPServer.Header2)
	}

	httpClubLogo := newTextinput()
	httpClubLogo.CharLimit = 200
	httpClubLogo.SetWidth(28)
	httpClubLogo.Placeholder = "https://... (URL only)"
	if cfg.Integrations.HTTPServer.ClubLogo != "" {
		httpClubLogo.SetValue(cfg.Integrations.HTTPServer.ClubLogo)
	}

	httpQRLink := newTextinput()
	httpQRLink.CharLimit = 70
	httpQRLink.SetWidth(28)
	httpQRLink.Placeholder = "https://cqops.com (default)"
	if cfg.Integrations.HTTPServer.QRLink != "" {
		httpQRLink.SetValue(cfg.Integrations.HTTPServer.QRLink)
	}

	httpEvtStart := newTextinput()
	httpEvtStart.CharLimit = 10
	httpEvtStart.SetWidth(28)
	httpEvtStart.Placeholder = "YYYY-MM-DD (optional)"
	if cfg.Integrations.HTTPServer.EventStart != "" {
		httpEvtStart.SetValue(cfg.Integrations.HTTPServer.EventStart)
	}

	httpTLSCert := newTextinput()
	httpTLSCert.CharLimit = 200
	httpTLSCert.SetWidth(28)
	httpTLSCert.Placeholder = "auto (self-signed)"
	if cfg.Integrations.HTTPServer.TLSCert != "" {
		httpTLSCert.SetValue(cfg.Integrations.HTTPServer.TLSCert)
	}

	httpTLSKey := newTextinput()
	httpTLSKey.CharLimit = 200
	httpTLSKey.SetWidth(28)
	httpTLSKey.Placeholder = "auto (self-signed)"
	if cfg.Integrations.HTTPServer.TLSKey != "" {
		httpTLSKey.SetValue(cfg.Integrations.HTTPServer.TLSKey)
	}

	httpTheme := 0 // Bright
	switch cfg.Integrations.HTTPServer.Theme {
	case "dark":
		httpTheme = 1
	case "yl":
		httpTheme = 2
	case "hivis":
		httpTheme = 3
	}

	// GPS
	gpsSvc := 0
	switch cfg.Integrations.GPS.Service {
	case "gpsd":
		gpsSvc = 1
	default:
		gpsSvc = 0 // serial (or empty → default to serial)
	}
	gridPrec := cfg.Integrations.GPS.GridPrecision
	if gridPrec != 6 && gridPrec != 8 {
		gridPrec = 10
	}
	gpsPort := newTextinput()
	gpsPort.CharLimit = 40
	gpsPort.SetWidth(28)
	gpsPort.Placeholder = "COM6 or /dev/ttyUSB0"
	if cfg.Integrations.GPS.Port != "" {
		gpsPort.SetValue(cfg.Integrations.GPS.Port)
	}
	gpsBaud := cfg.Integrations.GPS.BaudRate
	if gpsBaud == 0 {
		gpsBaud = 115200
	}
	gpsdHost := newTextinput()
	gpsdHost.CharLimit = 40
	gpsdHost.SetWidth(28)
	gpsdHost.Placeholder = "127.0.0.1"
	if cfg.Integrations.GPS.GPSDHost != "" {
		gpsdHost.SetValue(cfg.Integrations.GPS.GPSDHost)
	} else {
		gpsdHost.SetValue("127.0.0.1")
	}
	gpsdPort := newTextinput()
	gpsdPort.CharLimit = 6
	gpsdPort.SetWidth(28)
	gpsdPort.Placeholder = "2947"
	if cfg.Integrations.GPS.GPSDPort != "" {
		gpsdPort.SetValue(cfg.Integrations.GPS.GPSDPort)
	} else {
		gpsdPort.SetValue("2947")
	}

	// APRS
	aprsSvc := 0
	switch cfg.Integrations.APRS.Service {
	case "kiss":
		aprsSvc = 1
	case "kiss_server":
		aprsSvc = 2
	default:
		aprsSvc = 0 // aprs_is (or empty → default to APRS-IS)
	}
	aprsServer := newTextinput()
	aprsServer.CharLimit = 60
	aprsServer.SetWidth(28)
	aprsServer.Placeholder = "euro.aprs2.net:14580"
	if cfg.Integrations.APRS.Server != "" {
		aprsServer.SetValue(cfg.Integrations.APRS.Server)
	} else {
		aprsServer.SetValue("euro.aprs2.net:14580")
	}
	aprsKISSHost := newTextinput()
	aprsKISSHost.CharLimit = 40
	aprsKISSHost.SetWidth(28)
	aprsKISSHost.Placeholder = "127.0.0.1"
	if cfg.Integrations.APRS.KISSServerHost != "" {
		aprsKISSHost.SetValue(cfg.Integrations.APRS.KISSServerHost)
	} else {
		aprsKISSHost.SetValue("127.0.0.1")
	}
	aprsKISSPort := newTextinput()
	aprsKISSPort.CharLimit = 6
	aprsKISSPort.SetWidth(28)
	aprsKISSPort.Placeholder = "8001"
	if cfg.Integrations.APRS.KISSServerPort != "" {
		aprsKISSPort.SetValue(cfg.Integrations.APRS.KISSServerPort)
	} else {
		aprsKISSPort.SetValue("8001")
	}
	aprsPort := newTextinput()
	aprsPort.CharLimit = 40
	aprsPort.SetWidth(28)
	aprsPort.Placeholder = "COM6 or /dev/ttyUSB0"
	if cfg.Integrations.APRS.Port != "" {
		aprsPort.SetValue(cfg.Integrations.APRS.Port)
	}
	aprsBaud := cfg.Integrations.APRS.BaudRate
	if aprsBaud == 0 {
		aprsBaud = 9600 // KISS default; GPS uses 115200
	}
	aprsDTR := cfg.Integrations.APRS.DTR
	if !aprsDTR && cfg.Integrations.APRS.BaudRate == 0 {
		// First time — apply KISS-typical default (DTR powers most hardware TNCs).
		aprsDTR = true
	}
	aprsData := cfg.Integrations.APRS.DataBits
	if aprsData < 5 || aprsData > 8 {
		aprsData = 8
	}
	aprsPar := 0
	switch cfg.Integrations.APRS.Parity {
	case "odd":
		aprsPar = 1
	case "even":
		aprsPar = 2
	case "mark":
		aprsPar = 3
	case "space":
		aprsPar = 4
	}
	aprsStop := 0
	switch cfg.Integrations.APRS.StopBits {
	case "1.5":
		aprsStop = 1
	case "2":
		aprsStop = 2
	}

	return &IntegrationMenu{
		dxcEnabled:       cfg.Integrations.DXC.Enabled,
		dxcHost:          dxcHost,
		dxcPort:          dxcPort,
		dxcLogin:         dxcLogin,
		httpEnabled:      cfg.Integrations.HTTPServer.Enabled,
		httpTheme:        httpTheme,
		httpAddrIdx:      httpAddrIdx,
		httpAddr:         httpAddr,
		httpPort:         httpPort,
		httpHeader1:      httpHeader1,
		httpHeader2:      httpHeader2,
		httpClubLogo:     httpClubLogo,
		httpQRLink:       httpQRLink,
		httpEvtStart:     httpEvtStart,
		httpTLS:          cfg.Integrations.HTTPServer.TLSEnabled,
		httpTLSCert:      httpTLSCert,
		httpTLSKey:       httpTLSKey,
		gpsEnabled:       cfg.Integrations.GPS.Enabled,
		gpsService:       gpsSvc,
		gpsGridPrecision: gridPrec,
		gpsPort:          gpsPort,
		gpsBaudRate:      gpsBaud,
		gpsDTR:           cfg.Integrations.GPS.DTR,
		gpsRTS:           cfg.Integrations.GPS.RTS,
		gpsdHost:         gpsdHost,
		gpsdPort:         gpsdPort,
		aprsEnabled:      cfg.Integrations.APRS.Enabled,
		pskEnabled:       cfg.Integrations.PSK.Enabled,
		aprsService:      aprsSvc,
		aprsServer:       aprsServer,
		aprsKISSHost:     aprsKISSHost,
		aprsKISSPort:     aprsKISSPort,
		aprsPort:         aprsPort,
		aprsBaudRate:     aprsBaud,
		aprsDataBits:     aprsData,
		aprsParity:       aprsPar,
		aprsStopBits:     aprsStop,
		aprsDTR:          aprsDTR,
		aprsRTS:          cfg.Integrations.APRS.RTS,
	}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height

	case gpsTestMsg:
		im.gpsTesting = false
		if msg.err != nil {
			im.gpsToast = "GPS: " + friendlyGPSError(msg.err)
			applog.Warn("GPS test failed", "error", msg.err.Error())
		} else if msg.ok {
			im.gpsToast = "GPS: connection verified"
			im.gpsNeedsPoll = true // signal main model to refresh status bar
			applog.Info("GPS test OK")
		} else {
			im.gpsToast = "GPS: no data received"
		}

	case aprsTestMsg:
		im.aprsTesting = false
		if msg.err != nil {
			im.aprsToast = "APRS: " + msg.err.Error()
			applog.Warn("APRS test failed", "error", msg.err.Error())
		} else {
			im.aprsToast = "APRS: connection verified"
			applog.Info("APRS test OK")
		}

	case tea.KeyPressMsg:
		k := msg.String()
		// Shared navigation: Tab/Down, Shift+Tab/Up, and the Save & Back
		// button (Space/Enter saves through the same validation as Ctrl+S).
		if handled, cmd := im.fm.onKey(msg, im, func() tea.Cmd { return im.trySave() }); handled {
			return im, cmd
		}
		switch k {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case " ", "space":
			// Space triggers Test buttons in parallel with Enter.
			if im.fm.row == imGPSTest || im.fm.row == imAPRSTest {
				m2, c := im.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
				return m2.(*IntegrationMenu), c
			}
			switch im.fm.row {
			case imDXCChk:
				im.dxcEnabled = !im.dxcEnabled
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imHTTPChk:
				im.httpEnabled = !im.httpEnabled
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imHTTPAddr:
				im.httpAddrIdx = (im.httpAddrIdx + 1) % 2
				if im.httpAddrIdx == 0 {
					im.httpAddr.SetValue("localhost")
				} else {
					im.httpAddr.SetValue("0.0.0.0")
				}
				return im, nil
			case imHTTPTheme:
				im.httpTheme = (im.httpTheme + 1) % 4
				return im, nil
			case imHTTPTLS:
				im.httpTLS = !im.httpTLS
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imGPSChk:
				im.gpsEnabled = !im.gpsEnabled
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imAPRSChk:
				im.aprsEnabled = !im.aprsEnabled
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imPSKChk:
				im.pskEnabled = !im.pskEnabled
				return im, nil
			case imGPSDTR:
				im.gpsDTR = !im.gpsDTR
				return im, nil
			case imGPSRTS:
				im.gpsRTS = !im.gpsRTS
				return im, nil
			case imGPSBaud:
				im.gpsBaudRate = nextGPSCycle(im.gpsBaudRate)
				return im, nil
			case imGPSSvc:
				im.gpsService = (im.gpsService + 1) % len(gpsServiceOptions)
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imGPSGridPrec:
				im.gpsGridPrecision = nextGPSCycleInt(im.gpsGridPrecision, gpsPrecisionOptions)
				return im, nil
			case imAPRSSvc:
				im.aprsService = (im.aprsService + 1) % len(aprsServiceOptions)
				if !im.isPositionVisible(im.fm.row) {
					im.fm.fixFocus(im)
				}
				scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
				return im, nil
			case imAPRSBaud:
				im.aprsBaudRate = nextGPSCycle(im.aprsBaudRate)
				return im, nil
			case imAPRSData:
				im.aprsDataBits = nextGPSCycleInt(im.aprsDataBits, dataBitsOptions)
				return im, nil
			case imAPRSParity:
				im.aprsParity = (im.aprsParity + 1) % len(parityOptions)
				return im, nil
			case imAPRSStop:
				im.aprsStopBits = (im.aprsStopBits + 1) % len(stopBitsOptions)
				return im, nil
			case imAPRSDTR:
				im.aprsDTR = !im.aprsDTR
				return im, nil
			case imAPRSRTS:
				im.aprsRTS = !im.aprsRTS
				return im, nil
			}
			// Fall through to text input for editable fields.
			switch im.fm.row {
			case imDXCHost:
				im.dxcHost, _ = im.dxcHost.Update(msg)
			case imDXCPort:
				im.dxcPort, _ = im.dxcPort.Update(msg)
			case imDXCLogin:
				im.dxcLogin, _ = im.dxcLogin.Update(msg)
			case imHTTPPort:
				im.httpPort, _ = im.httpPort.Update(msg)
			case imHTTPHdr1:
				im.httpHeader1, _ = im.httpHeader1.Update(msg)
			case imHTTPHdr2:
				im.httpHeader2, _ = im.httpHeader2.Update(msg)
			case imHTTPLogo:
				im.httpClubLogo, _ = im.httpClubLogo.Update(msg)
			case imHTTPTLSCert:
				im.httpTLSCert, _ = im.httpTLSCert.Update(msg)
			case imHTTPTLSKey:
				im.httpTLSKey, _ = im.httpTLSKey.Update(msg)
			case imGPSPort:
				im.gpsPort, _ = im.gpsPort.Update(msg)
			}
		case "tab", "down":
			_ = im.fm.next(im)
			scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
		case "shift+tab", "up":
			_ = im.fm.prev(im)
			scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
		case "enter":
			if im.fm.row == imGPSTest {
				switch im.gpsService {
				case 0: // Serial
					port := strings.TrimSpace(im.gpsPort.Value())
					baud := im.gpsBaudRate
					dtr := im.gpsDTR
					rts := im.gpsRTS
					if port == "" || baud == 0 {
						im.gpsToast = "GPS: port and baud rate required"
						return im, nil
					}
					im.gpsTesting = true
					return im, func() tea.Msg {
						err := testGPSConnection(port, baud, dtr, rts)
						return gpsTestMsg{ok: err == nil, err: err}
					}
				case 1: // GPSD
					host := strings.TrimSpace(im.gpsdHost.Value())
					port := strings.TrimSpace(im.gpsdPort.Value())
					if host == "" {
						im.gpsToast = "GPS: GPSD host required"
						return im, nil
					}
					if port == "" {
						port = "2947"
					}
					im.gpsTesting = true
					return im, func() tea.Msg {
						err := testGPSDConnection(host, port)
						return gpsTestMsg{ok: err == nil, err: err}
					}
				}
			}
			if im.fm.row == imAPRSTest {
				switch im.aprsService {
				case 0: // APRS-IS
					srv := strings.TrimSpace(im.aprsServer.Value())
					if !im.inetOnline {
						im.aprsToast = "APRS: no internet connection"
						return im, nil
					}
					if srv == "" {
						im.aprsToast = "APRS: server is required"
						return im, nil
					}
					im.aprsTesting = true
					return im, func() tea.Msg {
						conn, err := net.DialTimeout("tcp", srv, 5*time.Second)
						if err != nil {
							return aprsTestMsg{err: fmt.Errorf("cannot reach %s: %v", srv, err)}
						}
						conn.Close()
						return aprsTestMsg{}
					}
				case 1: // KISS
					// If the KISS client is already running, the port is open —
					// no need to try opening it again (which would fail with "port busy").
					if im.aprsOnline {
						im.aprsToast = "APRS: connection verified"
						return im, nil
					}
					prt := strings.TrimSpace(im.aprsPort.Value())
					baud := im.aprsBaudRate
					if prt == "" || baud == 0 {
						im.aprsToast = "APRS: port and baud rate required"
						return im, nil
					}
					par := intToParity(im.aprsParity)
					stop := intToStopBits(im.aprsStopBits)
					im.aprsTesting = true
					return im, func() tea.Msg {
						err := testKISSPort(prt, baud, im.aprsDataBits, par, stop, im.aprsDTR, im.aprsRTS)
						return aprsTestMsg{err: err}
					}
				case 2: // KISS Server
					host := strings.TrimSpace(im.aprsKISSHost.Value())
					port := strings.TrimSpace(im.aprsKISSPort.Value())
					if host == "" {
						im.aprsToast = "APRS: host is required"
						return im, nil
					}
					if port == "" {
						port = "8001"
					}
					addr := net.JoinHostPort(host, port)
					im.aprsTesting = true
					return im, func() tea.Msg {
						conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
						if err != nil {
							return aprsTestMsg{err: fmt.Errorf("cannot reach %s: %v", addr, err)}
						}
						conn.Close()
						return aprsTestMsg{}
					}
				}
			}
			im.fm.next(im)
			scrollViewportToFraction(&im.vp, im.fm.scrollFraction(im))
		case "pgup":
			if im.fm.row == imGPSBaud {
				im.gpsBaudRate = nextGPSCycle(im.gpsBaudRate)
				return im, nil
			}
			if im.fm.row == imAPRSBaud {
				im.aprsBaudRate = nextGPSCycle(im.aprsBaudRate)
				return im, nil
			}
			im.vp, _ = im.vp.Update(msg)
		case "pgdown":
			if im.fm.row == imGPSBaud {
				im.gpsBaudRate = prevGPSCycle(im.gpsBaudRate)
				return im, nil
			}
			if im.fm.row == imAPRSBaud {
				im.aprsBaudRate = prevGPSCycle(im.aprsBaudRate)
				return im, nil
			}
			im.vp, _ = im.vp.Update(msg)
		default:
			im.forwardToFocused(msg)
			// Forward to viewport for manual scroll (home/end).
			im.vp, _ = im.vp.Update(msg)
		}
	// Forward paste and other non-key messages to the focused textinput.
	default:
		im.forwardToFocused(msg)
	}
	return im, nil
}

func (im *IntegrationMenu) forwardToFocused(msg tea.Msg) {
	switch im.fm.row {
	case imDXCHost:
		im.dxcHost, _ = im.dxcHost.Update(msg)
	case imDXCPort:
		im.dxcPort, _ = im.dxcPort.Update(msg)
	case imDXCLogin:
		im.dxcLogin, _ = im.dxcLogin.Update(msg)
	case imHTTPPort:
		im.httpPort, _ = im.httpPort.Update(msg)
	case imHTTPHdr1:
		im.httpHeader1, _ = im.httpHeader1.Update(msg)
	case imHTTPHdr2:
		im.httpHeader2, _ = im.httpHeader2.Update(msg)
	case imHTTPLogo:
		im.httpClubLogo, _ = im.httpClubLogo.Update(msg)
	case imHTTPQRLink:
		im.httpQRLink, _ = im.httpQRLink.Update(msg)
	case imHTTPEvt:
		im.httpEvtStart, _ = im.httpEvtStart.Update(msg)
	case imHTTPTLSCert:
		im.httpTLSCert, _ = im.httpTLSCert.Update(msg)
	case imHTTPTLSKey:
		im.httpTLSKey, _ = im.httpTLSKey.Update(msg)
	case imGPSPort:
		im.gpsPort, _ = im.gpsPort.Update(msg)
	case imGPSDHost:
		im.gpsdHost, _ = im.gpsdHost.Update(msg)
	case imGPSDPort:
		im.gpsdPort, _ = im.gpsdPort.Update(msg)
	case imAPRSServer:
		im.aprsServer, _ = im.aprsServer.Update(msg)
	case imAPRSKISSHost:
		im.aprsKISSHost, _ = im.aprsKISSHost.Update(msg)
	case imAPRSKISSPort:
		im.aprsKISSPort, _ = im.aprsKISSPort.Update(msg)
	case imAPRSPort:
		im.aprsPort, _ = im.aprsPort.Update(msg)
	}
}

func (im *IntegrationMenu) isPositionVisible(pos int) bool {
	switch pos {
	case imDXCChk, imHTTPChk, imGPSChk:
		return true
	case imDXCHost, imDXCPort, imDXCLogin:
		return im.dxcEnabled
	// QRZ positions are now dead — callbook is a top-level config menu.
	case imQRZChk, imQRZUser, imQRZPass, imQRZTest:
		return false
	case imHTTPAddr, imHTTPPort, imHTTPTheme, imHTTPHdr1, imHTTPHdr2, imHTTPLogo, imHTTPQRLink, imHTTPEvt:
		return im.httpEnabled
	case imHTTPTLS:
		return im.httpEnabled
	case imHTTPTLSCert, imHTTPTLSKey:
		return im.httpEnabled && im.httpTLS
	// GPS fields visibility depends on enabled + service type.
	case imGPSSvc, imGPSGridPrec:
		return im.gpsEnabled
	case imGPSPort, imGPSBaud, imGPSDTR, imGPSRTS:
		return im.gpsEnabled && im.gpsService == 0 // serial
	case imGPSDHost, imGPSDPort:
		return im.gpsEnabled && im.gpsService == 1 // GPSD
	case imGPSTest:
		return im.gpsEnabled // all services
	case imAPRSChk:
		return true // APRS checkbox always reachable
	case imPSKChk:
		return true // PSK checkbox always reachable
	case imAPRSSvc:
		return im.aprsEnabled
	case imAPRSServer:
		return im.aprsEnabled && im.aprsService == 0 // APRS-IS only
	case imAPRSKISSHost, imAPRSKISSPort:
		return im.aprsEnabled && im.aprsService == 2 // KISS Server only
	case imAPRSPort, imAPRSBaud, imAPRSData, imAPRSParity, imAPRSStop, imAPRSDTR, imAPRSRTS:
		return im.aprsEnabled && im.aprsService == 1 // KISS serial only
	case imAPRSTest:
		return im.aprsEnabled // all services
	}
	return true
}

// focusableRows implementation.
func (im *IntegrationMenu) rowCount() int         { return imMax }
func (im *IntegrationMenu) rowVisible(i int) bool { return im.isPositionVisible(i) }

func (im *IntegrationMenu) blurAll() {
	blurTextinputs(&im.dxcHost, &im.dxcPort, &im.dxcLogin,
		&im.httpAddr, &im.httpPort, &im.httpHeader1, &im.httpHeader2, &im.httpClubLogo, &im.httpQRLink, &im.httpEvtStart, &im.httpTLSCert, &im.httpTLSKey,
		&im.gpsPort, &im.gpsdHost, &im.gpsdPort,
		&im.aprsServer, &im.aprsKISSHost, &im.aprsKISSPort, &im.aprsPort)
}
func (im *IntegrationMenu) focusRow(i int) tea.Cmd {
	switch i {
	case imDXCHost:
		im.dxcHost.Focus()
	case imDXCPort:
		im.dxcPort.Focus()
	case imDXCLogin:
		im.dxcLogin.Focus()
	case imHTTPPort:
		im.httpPort.Focus()
	case imHTTPHdr1:
		im.httpHeader1.Focus()
	case imHTTPHdr2:
		im.httpHeader2.Focus()
	case imHTTPLogo:
		im.httpClubLogo.Focus()
	case imHTTPQRLink:
		im.httpQRLink.Focus()
	case imHTTPEvt:
		im.httpEvtStart.Focus()
	case imHTTPTLSCert:
		im.httpTLSCert.Focus()
	case imHTTPTLSKey:
		im.httpTLSKey.Focus()
	case imGPSPort:
		im.gpsPort.Focus()
	case imGPSDHost:
		im.gpsdHost.Focus()
	case imGPSDPort:
		im.gpsdPort.Focus()
	case imAPRSServer:
		im.aprsServer.Focus()
	case imAPRSKISSHost:
		im.aprsKISSHost.Focus()
	case imAPRSKISSPort:
		im.aprsKISSPort.Focus()
	case imAPRSPort:
		im.aprsPort.Focus()
	}
	return nil
}

// trySave validates the enabled integrations and closes the menu. Used by
// both Ctrl+S and the Save & Back button.
func (im *IntegrationMenu) trySave() tea.Cmd {
	// Validate DXC fields when DXC is enabled.
	if im.dxcEnabled {
		if strings.TrimSpace(im.dxcHost.Value()) == "" {
			im.SaveError = "DXC: host (server) is required"
			return nil
		}
		if strings.TrimSpace(im.dxcPort.Value()) == "" {
			im.SaveError = "DXC: port is required"
			return nil
		}
		if strings.TrimSpace(im.dxcLogin.Value()) == "" {
			im.SaveError = "DXC: login (callsign) is required"
			return nil
		}
	}
	// Validate HTTP server fields when HTTP server is enabled.
	if im.httpEnabled {
		if strings.TrimSpace(im.httpPort.Value()) == "" {
			im.SaveError = "HTTP server: port is required"
			return nil
		}
		// Validate Event Start format if entered.
		if es := strings.TrimSpace(im.httpEvtStart.Value()); es != "" {
			if _, err := time.Parse("2006-01-02", es); err != nil {
				im.SaveError = "HTTP server: Event Start must be YYYY-MM-DD or empty"
				return nil
			}
		}
		// TLS certificate and key must be configured together.
		cert := strings.TrimSpace(im.httpTLSCert.Value())
		key := strings.TrimSpace(im.httpTLSKey.Value())
		if (cert == "") != (key == "") {
			im.SaveError = "HTTP server: TLS certificate and key paths must be set together (or both empty for auto self-signed)"
			return nil
		}
	}
	// Validate GPS fields when GPS is enabled.
	if im.gpsEnabled {
		switch im.gpsService {
		case 0: // Serial
			if strings.TrimSpace(im.gpsPort.Value()) == "" {
				im.SaveError = "GPS: serial port is required"
				return nil
			}
		case 1: // GPSD
			if strings.TrimSpace(im.gpsdHost.Value()) == "" {
				im.SaveError = "GPS: GPSD host is required"
				return nil
			}
		}
	}
	// Validate APRS fields when APRS is enabled.
	if im.aprsEnabled {
		switch im.aprsService {
		case 0: // APRS-IS
			if strings.TrimSpace(im.aprsServer.Value()) == "" {
				im.SaveError = "APRS: server is required"
				return nil
			}
		case 1: // KISS
			if strings.TrimSpace(im.aprsPort.Value()) == "" {
				im.SaveError = "APRS: KISS port is required"
				return nil
			}
		}
	}
	im.done = true
	im.saved = true
	return nil
}

func (im *IntegrationMenu) View() tea.View {
	if im.done {
		return tea.NewView("")
	}
	w := im.width
	if w < 40 {
		w = 80
	}
	h := im.height
	if h < 10 {
		h = 24
	}

	var b strings.Builder

	// Truncation width for form lines — must match viewport content width
	// (boxW - 4 for menuBoxStyle border + padding).
	lineW := w - 2 - 4
	if lineW < 36 {
		lineW = 36
	}
	if lineW > partnerMapMaxW-4 {
		lineW = partnerMapMaxW - 4
	}

	// --- Info box (same pattern as other config menus) ---
	infoMaxW := lineW - 4
	if infoMaxW < 30 {
		infoMaxW = 30
	}
	infoText := "Integrations are the core of CQOps functionality. " +
		"Enable or disable each service as needed — but use " +
		"caution: some integrations (DX Cluster, APRS, GPS) " +
		"can create additional CPU or network load, especially " +
		"on low-end hardware or field setups."
	infoBox(&b, infoText, infoMaxW)

	// --- DXC section ---
	dxcCheckbox := "[ ]"
	if im.dxcEnabled {
		dxcCheckbox = "[x]"
	}
	dxcPrefix := "  "
	dxcLabel := S.FormLabelWide.Align(lipgloss.Left).Render("DX Cluster:")
	if im.fm.row == imDXCChk {
		dxcPrefix = S.FormPrefixOn.Render("> ")
		dxcLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("DX Cluster:")
		dxcCheckbox = CursorStyle.Render(dxcCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, dxcPrefix, dxcLabel, " ", dxcCheckbox),
		lineW))

	if im.dxcEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCHost, "  Host:", &im.dxcHost, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCPort, "  Port:", &im.dxcPort, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imDXCLogin, "  Login:", &im.dxcLogin, false), lineW))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// --- HTTP Server section ---
	httpCheckbox := "[ ]"
	if im.httpEnabled {
		httpCheckbox = "[x]"
	}
	httpPrefix := "  "
	httpLabel := S.FormLabelWide.Align(lipgloss.Left).Render("HTTP Server:")
	if im.fm.row == imHTTPChk {
		httpPrefix = S.FormPrefixOn.Render("> ")
		httpLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("HTTP Server:")
		httpCheckbox = CursorStyle.Render(httpCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, httpPrefix, httpLabel, " ", httpCheckbox),
		lineW))

	if im.httpEnabled {
		b.WriteString("\n")
		// Access: cycle between "This PC only" / "Local network".
		addrLabels := []string{"This PC only", "Local network"}
		addrVal := ValueStyle.Render(addrLabels[im.httpAddrIdx])
		addrLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Access:")
		if im.fm.row == imHTTPAddr {
			addrLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Access:")
			addrVal = CursorStyle.Render(addrLabels[im.httpAddrIdx]) + " " + DimStyle.Render("(Space)")
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, "  ", addrLabel, " ", addrVal),
			lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPPort, "  Port:", &im.httpPort, false), lineW))
		b.WriteString("\n")
		themeNames := []string{"Bright", "Dark", "Orchid", "HighVis"}
		valueRow(&b, lineW, im.fm.row == imHTTPTheme, "  Theme:", themeNames[im.httpTheme], "", integrationRows)
		tlsCheckbox := "[ ]"
		if im.httpTLS {
			tlsCheckbox = "[x]"
		}
		tlsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  HTTPS (TLS):")
		if im.fm.row == imHTTPTLS {
			tlsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  HTTPS (TLS):")
			tlsCheckbox = CursorStyle.Render(tlsCheckbox) + " " + DimStyle.Render("(Space)")
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, "  ", tlsLabel, " ", tlsCheckbox),
			lineW))
		b.WriteString("\n")
		if im.httpTLS {
			b.WriteString(padOrTrunc(im.renderField(imHTTPTLSCert, "  TLS Cert (opt):", &im.httpTLSCert, false), lineW))
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imHTTPTLSKey, "  TLS Key (opt):", &im.httpTLSKey, false), lineW))
			b.WriteString("\n")
		}
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr1, "  Header 1 (opt):", &im.httpHeader1, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr2, "  Header 2 (opt):", &im.httpHeader2, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPLogo, "  Logo URL (opt):", &im.httpClubLogo, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPQRLink, "  QR Link (opt):", &im.httpQRLink, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPEvt, "  Event Start (opt):", &im.httpEvtStart, false), lineW))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
	}

	// --- GPS section ---
	gpsCheckbox := "[ ]"
	if im.gpsEnabled {
		gpsCheckbox = "[x]"
	}
	gpsPrefix := "  "
	gpsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("GPS Service:")
	if im.fm.row == imGPSChk {
		gpsPrefix = S.FormPrefixOn.Render("> ")
		gpsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("GPS Service:")
		gpsCheckbox = CursorStyle.Render(gpsCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, gpsPrefix, gpsLabel, " ", gpsCheckbox),
		lineW))

	if im.gpsEnabled {
		// Service type — PgUp/PgDn to cycle.
		b.WriteString("\n")
		svcPrefix := "  "
		svcLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Service:")
		svcVal := gpsServiceOptions[im.gpsService].label
		if im.fm.row == imGPSSvc {
			svcPrefix = S.FormPrefixOn.Render("> ")
			svcLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Service:")
			svcVal = CursorStyle.Render(svcVal) + " " + DimStyle.Render("(Space)")
		} else {
			svcVal = ValueStyle.Render(svcVal)
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, svcPrefix, svcLabel, " ", svcVal),
			lineW))

		// Grid precision — PgUp/PgDn to cycle.
		b.WriteString("\n")
		precPrefix := "  "
		precLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Grid precision:")
		precVal := fmt.Sprintf("%d chars", im.gpsGridPrecision)
		if im.fm.row == imGPSGridPrec {
			precPrefix = S.FormPrefixOn.Render("> ")
			precLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Grid precision:")
			precVal = CursorStyle.Render(precVal) + " " + DimStyle.Render("(Space)")
		} else {
			precVal = ValueStyle.Render(precVal)
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, precPrefix, precLabel, " ", precVal),
			lineW))

		// Serial-specific fields.
		if im.gpsService == 0 {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imGPSPort, "  Port:", &im.gpsPort, false), lineW))
			b.WriteString("\n")
			// Baud rate with PgUp/PgDn cycling.
			baudPrefix := "  "
			baudLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Baud:")
			baudVal := fmt.Sprintf("%d", im.gpsBaudRate)
			if im.fm.row == imGPSBaud {
				baudPrefix = S.FormPrefixOn.Render("> ")
				baudLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Baud:")
				baudVal = CursorStyle.Render(baudVal) + " " + DimStyle.Render("(Space)")
			} else {
				baudVal = ValueStyle.Render(baudVal)
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, baudPrefix, baudLabel, " ", baudVal),
				lineW))
			b.WriteString("\n")
			// DTR checkbox.
			dtrCb := "[ ]"
			if im.gpsDTR {
				dtrCb = "[x]"
			}
			dtrPrefix := "  "
			dtrLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  DTR:")
			if im.fm.row == imGPSDTR {
				dtrPrefix = S.FormPrefixOn.Render("> ")
				dtrLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  DTR:")
				dtrCb = CursorStyle.Render(dtrCb) + " " + DimStyle.Render("(Space)")
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, dtrPrefix, dtrLabel, " ", dtrCb),
				lineW))
			b.WriteString("\n")
			// RTS checkbox.
			rtsCb := "[ ]"
			if im.gpsRTS {
				rtsCb = "[x]"
			}
			rtsPrefix := "  "
			rtsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  RTS:")
			if im.fm.row == imGPSRTS {
				rtsPrefix = S.FormPrefixOn.Render("> ")
				rtsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  RTS:")
				rtsCb = CursorStyle.Render(rtsCb) + " " + DimStyle.Render("(Space)")
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, rtsPrefix, rtsLabel, " ", rtsCb),
				lineW))
		}

		// GPSD-specific fields.
		if im.gpsService == 1 {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imGPSDHost, "  Host:", &im.gpsdHost, false), lineW))
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imGPSDPort, "  Port:", &im.gpsdPort, false), lineW))
		}

		// Test button — always available when GPS is enabled.
		b.WriteString("\n")
		btnText := "[ Test GPS ]"
		if im.gpsTesting {
			b.WriteString(padOrTrunc("    "+DimStyle.Render(btnText)+" "+DimStyle.Render("..."), lineW))
		} else {
			buttonRow(&b, lineW, im.fm.row == imGPSTest, btnText)
		}
	} else {
		b.WriteString("\n")
	}

	// --- APRS section ---
	aprsCheckbox := "[ ]"
	if im.aprsEnabled {
		aprsCheckbox = "[x]"
	}
	aprsPrefix := "  "
	aprsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("APRS:")
	if im.fm.row == imAPRSChk {
		aprsPrefix = S.FormPrefixOn.Render("> ")
		aprsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("APRS:")
		aprsCheckbox = CursorStyle.Render(aprsCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, aprsPrefix, aprsLabel, " ", aprsCheckbox),
		lineW))

	if im.aprsEnabled {
		// Service type — Space to cycle.
		b.WriteString("\n")
		svcPrefix := "  "
		svcLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Service:")
		svcVal := aprsServiceOptions[im.aprsService].label
		if im.fm.row == imAPRSSvc {
			svcPrefix = S.FormPrefixOn.Render("> ")
			svcLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Service:")
			svcVal = CursorStyle.Render(svcVal) + " " + DimStyle.Render("(Space)")
		} else {
			svcVal = ValueStyle.Render(svcVal)
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, svcPrefix, svcLabel, " ", svcVal),
			lineW))

		// APRS-IS — server host:port.
		if im.aprsService == 0 {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imAPRSServer, "  Server:", &im.aprsServer, false), lineW))
		}

		// KISS Server — separate host and port fields.
		if im.aprsService == 2 {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imAPRSKISSHost, "  Host:", &im.aprsKISSHost, false), lineW))
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imAPRSKISSPort, "  Port:", &im.aprsKISSPort, false), lineW))
		}

		// KISS specific fields.
		if im.aprsService == 1 {
			b.WriteString("\n")
			b.WriteString(padOrTrunc(im.renderField(imAPRSPort, "  Port:", &im.aprsPort, false), lineW))
			b.WriteString("\n")
			baudPrefix := "  "
			baudLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Baud:")
			baudVal := fmt.Sprintf("%d", im.aprsBaudRate)
			if im.fm.row == imAPRSBaud {
				baudPrefix = S.FormPrefixOn.Render("> ")
				baudLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Baud:")
				baudVal = CursorStyle.Render(baudVal) + " " + DimStyle.Render("(Space)")
			} else {
				baudVal = ValueStyle.Render(baudVal)
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, baudPrefix, baudLabel, " ", baudVal),
				lineW))
			// Data bits — Space to cycle.
			b.WriteString("\n")
			dataPrefix := "  "
			dataLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Data bits:")
			dataVal := fmt.Sprintf("%d", im.aprsDataBits)
			if im.fm.row == imAPRSData {
				dataPrefix = S.FormPrefixOn.Render("> ")
				dataLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Data bits:")
				dataVal = CursorStyle.Render(dataVal) + " " + DimStyle.Render("(Space)")
			} else {
				dataVal = ValueStyle.Render(dataVal)
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, dataPrefix, dataLabel, " ", dataVal),
				lineW))
			// Parity — Space to cycle.
			b.WriteString("\n")
			parPrefix := "  "
			parLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Parity:")
			parVal := parityOptions[im.aprsParity].label
			if im.fm.row == imAPRSParity {
				parPrefix = S.FormPrefixOn.Render("> ")
				parLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Parity:")
				parVal = CursorStyle.Render(parVal) + " " + DimStyle.Render("(Space)")
			} else {
				parVal = ValueStyle.Render(parVal)
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, parPrefix, parLabel, " ", parVal),
				lineW))
			// Stop bits — Space to cycle.
			b.WriteString("\n")
			stopPrefix := "  "
			stopLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  Stop bits:")
			stopVal := stopBitsOptions[im.aprsStopBits].label
			if im.fm.row == imAPRSStop {
				stopPrefix = S.FormPrefixOn.Render("> ")
				stopLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Stop bits:")
				stopVal = CursorStyle.Render(stopVal) + " " + DimStyle.Render("(Space)")
			} else {
				stopVal = ValueStyle.Render(stopVal)
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, stopPrefix, stopLabel, " ", stopVal),
				lineW))
			// DTR checkbox.
			b.WriteString("\n")
			dtrCb := "[ ]"
			if im.aprsDTR {
				dtrCb = "[x]"
			}
			dtrPrefix := "  "
			dtrLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  DTR:")
			if im.fm.row == imAPRSDTR {
				dtrPrefix = S.FormPrefixOn.Render("> ")
				dtrLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  DTR:")
				dtrCb = CursorStyle.Render(dtrCb) + " " + DimStyle.Render("(Space)")
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, dtrPrefix, dtrLabel, " ", dtrCb),
				lineW))
			// RTS checkbox.
			b.WriteString("\n")
			rtsCb := "[ ]"
			if im.aprsRTS {
				rtsCb = "[x]"
			}
			rtsPrefix := "  "
			rtsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("  RTS:")
			if im.fm.row == imAPRSRTS {
				rtsPrefix = S.FormPrefixOn.Render("> ")
				rtsLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  RTS:")
				rtsCb = CursorStyle.Render(rtsCb) + " " + DimStyle.Render("(Space)")
			}
			b.WriteString(padOrTrunc(
				lipgloss.JoinHorizontal(lipgloss.Center, rtsPrefix, rtsLabel, " ", rtsCb),
				lineW))
		}

		// Test button — always available when APRS is enabled.
		b.WriteString("\n")
		btnText := "[ Test APRS ]"
		if im.aprsTesting {
			b.WriteString(padOrTrunc("    "+DimStyle.Render(btnText)+" "+DimStyle.Render("..."), lineW))
		} else {
			buttonRow(&b, lineW, im.fm.row == imAPRSTest, btnText)
		}
	} else {
		b.WriteString("\n")
	}

	// --- PSK Reporter section ---
	pskCb := "[ ]"
	if im.pskEnabled {
		pskCb = "[x]"
	}
	pskPrefix := "  "
	pskLabel := S.FormLabelWide.Align(lipgloss.Left).Render("PSK Reporter:")
	if im.fm.row == imPSKChk {
		pskPrefix = S.FormPrefixOn.Render("> ")
		pskLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("PSK Reporter:")
		pskCb = CursorStyle.Render(pskCb) + " " + DimStyle.Render("(Space)")
		pskCb += " " + DimStyle.Render("F5 panel, off by default")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, pskPrefix, pskLabel, " ", pskCb),
		lineW))

	// Build raw form body — header is rendered separately above the viewport.
	b.WriteString("\n")
	b.WriteString(im.fm.btn.line("Save & Back", lineW))
	bodyStr := b.String()

	// Wrap in viewport for scrolling on small terminals.
	boxW := w
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // account for menu box left+right padding
	if vpW < 20 {
		vpW = 20
	}
	contentH := contentHeight(h)
	if contentH < 8 {
		contentH = 8
	}
	// Overhead: header(1) + blank row(1) + scroll hint(1) = 3 lines.
	vpH := contentH - 3
	if vpH < 4 {
		vpH = 4
	}
	im.vp.SetWidth(vpW)
	im.vp.SetHeight(vpH)
	if im.vp.TotalLineCount() == 0 || bodyStr != im.lastBodyContent {
		im.vp.SetContent(bodyStr)
		im.lastBodyContent = bodyStr
	}
	// Keep the focus marker inside the visible window — a fractional scroll
	// mapping let the cursor leave the screen on small terminals.
	scrollToFocusedLine(&im.vp, bodyStr)

	header := S.Title.Width(boxW).Render("Configuration \u2014 Integrations")
	vpContent := im.vp.View()
	hint := scrollHint(im.vp)
	hintLine := DimStyle.Width(vpW).Render(hint)
	if hintLine == "" {
		hintLine = strings.Repeat(" ", vpW)
	}
	vpContent = lipgloss.JoinVertical(lipgloss.Left, vpContent, hintLine)
	box := menuBoxStyle.Width(boxW).Render(vpContent)
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, "", box))
}

// renderField renders a labelled textinput line with cursor indicator.
// When masked is true, the value is shown as asterisks when not focused.
func (im *IntegrationMenu) renderField(focusIdx int, label string, ti *textinput.Model, masked bool) string {
	raw := strings.TrimSpace(ti.Value())
	var val string
	if im.fm.row == focusIdx {
		val = ti.View()
	} else if raw == "" {
		val = DimStyle.Render("\u2014")
	} else if masked {
		val = ValueStyle.Render(strings.Repeat("*", len(raw)))
	} else {
		val = ValueStyle.Render(raw)
	}
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render(label)
	if im.fm.row == focusIdx {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

// Values returns DXC and HTTP server config values.
func (im *IntegrationMenu) Values() (dxcEnabled bool, dxcHost, dxcPort, dxcLogin string, httpEnabled bool, httpAddr, httpPort, httpTheme string, httpHdr1, httpHdr2, httpLogo, httpQRLink, httpEvtStart string, httpTLS bool, httpTLSCert, httpTLSKey string) {
	return im.dxcEnabled,
		strings.TrimSpace(im.dxcHost.Value()),
		strings.TrimSpace(im.dxcPort.Value()),
		strings.TrimSpace(im.dxcLogin.Value()),
		im.httpEnabled,
		strings.TrimSpace(im.httpAddr.Value()),
		strings.TrimSpace(im.httpPort.Value()),
		func() string {
			switch im.httpTheme {
			case 1:
				return "dark"
			case 2:
				return "yl"
			case 3:
				return "hivis"
			default:
				return "bright"
			}
		}(),
		strings.TrimSpace(im.httpHeader1.Value()),
		strings.TrimSpace(im.httpHeader2.Value()),
		strings.TrimSpace(im.httpClubLogo.Value()),
		strings.TrimSpace(im.httpQRLink.Value()),
		strings.TrimSpace(im.httpEvtStart.Value()),
		im.httpTLS,
		strings.TrimSpace(im.httpTLSCert.Value()),
		strings.TrimSpace(im.httpTLSKey.Value())
}

func (im *IntegrationMenu) gpsServiceName() string {
	switch im.gpsService {
	case 1:
		return "gpsd"
	default:
		return "serial"
	}
}

func (im *IntegrationMenu) aprsServiceName() string {
	switch im.aprsService {
	case 1:
		return "kiss"
	case 2:
		return "kiss_server"
	default:
		return "aprs_is"
	}
}

func (im *IntegrationMenu) aprsParityName() string {
	switch im.aprsParity {
	case 1:
		return "odd"
	case 2:
		return "even"
	case 3:
		return "mark"
	case 4:
		return "space"
	default:
		return "none"
	}
}

func (im *IntegrationMenu) aprsStopBitsName() string {
	switch im.aprsStopBits {
	case 1:
		return "1.5"
	case 2:
		return "2"
	default:
		return "1"
	}
}

// intToParity converts the parity index (0-4) to a serial.Parity value.
func intToParity(idx int) serial.Parity {
	switch idx {
	case 1:
		return serial.OddParity
	case 2:
		return serial.EvenParity
	case 3:
		return serial.MarkParity
	case 4:
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

// intToStopBits converts the stop bits index (0-2) to a serial.StopBits value.
func intToStopBits(idx int) serial.StopBits {
	switch idx {
	case 1:
		return serial.OnePointFiveStopBits
	case 2:
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

// parityFromString converts a config parity string to a serial.Parity value.
func parityFromString(s string) serial.Parity {
	switch s {
	case "odd":
		return serial.OddParity
	case "even":
		return serial.EvenParity
	case "mark":
		return serial.MarkParity
	case "space":
		return serial.SpaceParity
	default:
		return serial.NoParity
	}
}

// stopBitsFromString converts a config stop bits string to a serial.StopBits value.
func stopBitsFromString(s string) serial.StopBits {
	switch s {
	case "1.5":
		return serial.OnePointFiveStopBits
	case "2":
		return serial.TwoStopBits
	default:
		return serial.OneStopBit
	}
}

// friendlyQRZError wraps raw network errors from QRZ lookups into
// user-readable messages.
func friendlyQRZError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.Contains(msg, "QRZ:") {
		return msg
	}
	if strings.Contains(msg, "no such host") {
		return "Cannot reach QRZ.com - check your internet connection"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") {
		return "QRZ.com timed out - try again later"
	}
	if strings.Contains(msg, "connection refused") {
		return "Cannot connect to QRZ.com - try again later"
	}
	return "QRZ lookup failed - " + msg
}

// friendlyHTestError shortens HamQTH test errors for display.
func friendlyHTestError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// Clean HamQTH-prefixed errors from our client.
	if strings.HasPrefix(msg, "HamQTH:") {
		return msg
	}

	// Hide raw XML/HTTP errors from the user.
	if strings.Contains(msg, "expected element type") || strings.Contains(msg, "cannot unmarshal") {
		return "HamQTH: unexpected server response — try again later"
	}
	if strings.Contains(msg, "no such host") {
		return "Cannot reach HamQTH.com — check your internet connection"
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") {
		return "HamQTH timed out — try again later"
	}
	if strings.Contains(msg, "connection refused") {
		return "Cannot connect to HamQTH — try again later"
	}
	return "HamQTH lookup failed — " + msg
}

// friendlyGPSError shortens verbose Go network errors for display in
// the one-line test result field.
func friendlyGPSError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	// Strip the verbose ": dial tcp ..." suffix.
	if idx := strings.Index(msg, ": dial tcp"); idx >= 0 {
		msg = msg[:idx]
	}
	// Strip "GPSD: " or "GPS: " prefixes.
	msg = strings.TrimPrefix(msg, "GPSD: ")
	msg = strings.TrimPrefix(msg, "GPS: ")
	return msg
}

// gpsBaudRates lists common GPS baud rates for PgUp/PgDn cycling.
var gpsBaudRates = []int{4800, 9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600}

func nextGPSCycle(current int) int {
	for i, r := range gpsBaudRates {
		if r == current && i+1 < len(gpsBaudRates) {
			return gpsBaudRates[i+1]
		}
	}
	return gpsBaudRates[0]
}

func prevGPSCycle(current int) int {
	for i := len(gpsBaudRates) - 1; i >= 0; i-- {
		if gpsBaudRates[i] == current && i > 0 {
			return gpsBaudRates[i-1]
		}
	}
	return gpsBaudRates[len(gpsBaudRates)-1]
}

// testGPSConnection tries to open the given serial port, read one NMEA
// line, and close it. Used by the [ Test GPS ] button in the integration menu.
func testGPSConnection(port string, baud int, dtr, rts bool) error {
	cfg := gps.SerialConfig{Port: port, BaudRate: baud, DTR: dtr, RTS: rts}
	r := gps.NewSerialReader(cfg)
	defer r.Close()

	// Try reading up to 5 lines — GPS at 1Hz should produce GGA within 5s.
	for i := 0; i < 5; i++ {
		line, err := r.ReadLine()
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "$GPGGA") || strings.HasPrefix(line, "$GNGGA") {
			applog.Debug("GPS test: NMEA received", "line", line)
			return nil
		}
	}
	return fmt.Errorf("no NMEA GGA sentence received in 5 seconds")
}

// testGPSDConnection tries to connect to a GPSD server, send a WATCH
// command, and read a TPV position report. Used by the [ Test GPS ]
// button for GPSD service.
func testGPSDConnection(host, port string) error {
	if host == "" {
		return fmt.Errorf("GPSD host is required")
	}
	if port == "" {
		port = "2947"
	}
	addr := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("cannot connect to %s: %w", addr, err)
	}
	defer conn.Close()

	// Read deadline: prevent indefinite hang when the server accepts
	// the connection but never sends data. 10 seconds total is plenty
	// for a GPSD server to respond with a TPV.
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Send WATCH command.
	_, err = fmt.Fprintf(conn, "?WATCH={\"enable\":true,\"json\":true}\n")
	if err != nil {
		return fmt.Errorf("WATCH command failed: %w", err)
	}

	// Read up to 20 lines looking for a TPV with a valid fix.
	scanner := bufio.NewScanner(conn)
	for i := 0; i < 20 && scanner.Scan(); i++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if cls, _ := obj["class"].(string); cls != "TPV" {
			continue
		}
		mode, _ := obj["mode"].(float64)
		if mode < 2 {
			continue
		}
		lat, _ := obj["lat"].(float64)
		lon, _ := obj["lon"].(float64)
		if lat == 0 && lon == 0 {
			continue
		}
		applog.Info("GPSD test: TPV received", "mode", fmt.Sprintf("%.0f", mode))
		applog.Debug("GPSD test: TPV details",
			"lat", fmt.Sprintf("%.6f", lat),
			"lon", fmt.Sprintf("%.6f", lon),
		)
		return nil
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read error: %w", err)
	}
	return fmt.Errorf("no TPV position received from GPSD — check antenna")
}

// testKISSPort opens a serial port, optionally toggles DTR to wake the TNC,
// and verifies the port is readable/writable within 4 seconds.
// Does NOT require incoming RF frames — a successful port open + write probe
// is sufficient to confirm the TNC is alive.
// Used by the [ Test APRS ] button when KISS service is selected.
func testKISSPort(port string, baud, dataBits int, parity serial.Parity, stopBits serial.StopBits, dtr, rts bool) error {
	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: dataBits,
		Parity:   parity,
		StopBits: stopBits,
	}
	p, err := serial.Open(port, mode)
	if err != nil {
		return fmt.Errorf("cannot open %s: %v", port, err)
	}
	defer p.Close()

	if rts {
		if err := p.SetRTS(true); err != nil {
			return fmt.Errorf("RTS failed on %s: %v", port, err)
		}
	}

	// DTR toggle: many hardware TNCs use DTR for power/reset.
	// Toggling DTR off→on triggers a reset, causing the TNC to output a
	// boot message or start decoding RF — giving us data to confirm it's alive.
	if dtr {
		_ = p.SetDTR(false)
		time.Sleep(300 * time.Millisecond)
		if err := p.SetDTR(true); err != nil {
			return fmt.Errorf("DTR failed on %s: %v", port, err)
		}
		// Give the TNC a moment to boot.
		time.Sleep(500 * time.Millisecond)
	}

	// Send an empty KISS frame to probe the TNC.
	// FEND (0xC0) + data command (0x00) + FEND (0xC0).
	kissPing := []byte{0xC0, 0x00, 0xC0}
	if _, err := p.Write(kissPing); err != nil {
		return fmt.Errorf("TNC write failed on %s: %v", port, err)
	}

	// Read whatever the TNC sends back (boot message, KISS frames, etc.).
	// 3-second window is enough for a boot message or nearby RF frames.
	if err := p.SetReadTimeout(3 * time.Second); err != nil {
		return nil // write succeeded, TNC is alive
	}

	buf := make([]byte, 4096)
	n, _ := p.Read(buf)
	if n > 0 {
		applog.Info("KISS test: TNC responsive",
			"port", port,
			"baud", fmt.Sprintf("%d", baud),
			"bytes", fmt.Sprintf("%d", n),
		)
		return nil
	}

	// No data received, but port opened and write succeeded — TNC is alive.
	applog.Info("KISS test: port OK (no data received, but TNC accepts writes)",
		"port", port,
		"baud", fmt.Sprintf("%d", baud),
	)
	return nil
}
