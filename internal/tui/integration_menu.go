package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
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

type IntegrationMenu struct {
	// DXC
	dxcEnabled bool
	dxcHost    textinput.Model
	dxcPort    textinput.Model
	dxcLogin   textinput.Model

	// QRZ
	qrzEnabled    bool
	qrzUser       textinput.Model
	qrzPass       textinput.Model
	qrzTesting    bool
	qrzTestResult string
	inetOnline    bool

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
	gpsTestResult    string

	// APRS
	aprsEnabled    bool
	aprsService    int // 0=APRS-IS, 1=KISS, 2=KISS Server
	aprsServer     textinput.Model
	aprsKISSHost   textinput.Model
	aprsKISSPort   textinput.Model
	aprsPort       textinput.Model
	aprsBaudRate   int
	aprsDataBits   int // 8, 7, 6, 5
	aprsParity     int // 0=None, 1=Odd, 2=Even, 3=Mark, 4=Space
	aprsStopBits   int // 0=1, 1=1.5, 2=2
	aprsDTR        bool
	aprsRTS        bool
	aprsTesting    bool
	aprsTestResult string
	aprsOnline     bool // true when APRS client is connected (KISS or APRS-IS)

	// aprsToast is set by APRS test handler; parent reads and shows toast, then clears.
	aprsToast string

	focus      int
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
	imHTTPHdr1     = 12
	imHTTPHdr2     = 13
	imHTTPLogo     = 14
	imHTTPQRLink   = 15
	imHTTPEvt      = 16
	imGPSChk       = 17
	imGPSSvc       = 18 // service type: None / Serial / GPSD
	imGPSGridPrec  = 19 // grid precision: 10 / 8 / 6
	imGPSPort      = 20 // serial port
	imGPSBaud      = 21 // baud rate
	imGPSDTR       = 22 // DTR
	imGPSRTS       = 23 // RTS
	imGPSDHost     = 24 // GPSD host
	imGPSDPort     = 25 // GPSD port
	imGPSTest      = 26 // test button
	imAPRSChk      = 27
	imAPRSSvc      = 28 // service type: APRS-IS / KISS / KISS Server
	imAPRSServer   = 29 // APRS-IS server host:port
	imAPRSKISSHost = 30 // KISS Server TCP host
	imAPRSKISSPort = 31 // KISS Server TCP port
	imAPRSPort     = 32 // KISS serial port
	imAPRSBaud     = 33 // KISS baud rate
	imAPRSData     = 34 // KISS data bits
	imAPRSParity   = 35 // KISS parity
	imAPRSStop     = 36 // KISS stop bits
	imAPRSDTR      = 37 // KISS DTR
	imAPRSRTS      = 38 // KISS RTS
	imAPRSTest     = 39 // test button
	imMax          = 40
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
	}

	qrzUser := newTextinput()
	qrzUser.CharLimit = 30
	qrzUser.SetWidth(28)
	qrzUser.Placeholder = "QRZ.com username"
	qrzUser.SetValue(cfg.Integrations.Callbook.QRZ.User)

	qrzPass := newTextinput()
	qrzPass.CharLimit = 40
	qrzPass.SetWidth(28)
	qrzPass.Placeholder = "QRZ.com password"
	qrzPass.EchoMode = textinput.EchoPassword
	qrzPass.EchoCharacter = '*'
	qrzPass.SetValue(cfg.Integrations.Callbook.QRZ.Pass)

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
		qrzEnabled:       cfg.Integrations.Callbook.QRZ.Enabled,
		qrzUser:          qrzUser,
		qrzPass:          qrzPass,
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
		focus:            0,
	}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height

	case callbookTestMsg:
		im.qrzTesting = false
		if msg.err != nil {
			im.qrzTestResult = friendlyQRZError(msg.err)
			applog.Error("QRZ test failed", "error", msg.err.Error())
		} else if msg.ok {
			im.qrzTestResult = "OK - QRZ.com connected"
			applog.Info("QRZ test OK")
		} else {
			im.qrzTestResult = "No data returned"
			applog.Warn("QRZ test: no data returned")
		}

	case gpsTestMsg:
		im.gpsTesting = false
		if msg.err != nil {
			im.gpsTestResult = "Failed — " + friendlyGPSError(msg.err)
			applog.Warn("GPS test failed", "error", msg.err.Error())
		} else if msg.ok {
			im.gpsTestResult = "OK — GPS responding"
			applog.Info("GPS test OK")
		} else {
			im.gpsTestResult = "No data received"
		}

	case aprsTestMsg:
		im.aprsTesting = false
		if msg.err != nil {
			im.aprsTestResult = "Failed — " + msg.err.Error()
			im.aprsToast = "APRS: " + msg.err.Error()
			applog.Warn("APRS test failed", "error", msg.err.Error())
		} else {
			im.aprsTestResult = "OK — connection working"
			im.aprsToast = "APRS: connection verified"
			applog.Info("APRS test OK")
		}

	case tea.KeyPressMsg:
		k := msg.String()
		if im.qrzTesting {
			return im, nil
		}
		switch k {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case "ctrl+s", "\x13":
			// Validate DXC fields when DXC is enabled.
			if im.dxcEnabled {
				if strings.TrimSpace(im.dxcHost.Value()) == "" {
					im.SaveError = "DXC host (server) is required when DXC is enabled"
					return im, nil
				}
				if strings.TrimSpace(im.dxcPort.Value()) == "" {
					im.SaveError = "DXC port is required when DXC is enabled"
					return im, nil
				}
				if strings.TrimSpace(im.dxcLogin.Value()) == "" {
					im.SaveError = "DXC login (callsign) is required when DXC is enabled"
					return im, nil
				}
			}
			// Validate HTTP server fields when HTTP server is enabled.
			if im.httpEnabled {
				if strings.TrimSpace(im.httpPort.Value()) == "" {
					im.SaveError = "HTTP server port is required when HTTP server is enabled"
					return im, nil
				}
				// Validate Event Start format if entered.
				if es := strings.TrimSpace(im.httpEvtStart.Value()); es != "" {
					if _, err := time.Parse("2006-01-02", es); err != nil {
						im.SaveError = "Event Start must be YYYY-MM-DD or empty"
						return im, nil
					}
				}
			}
			// Validate GPS fields when GPS is enabled.
			if im.gpsEnabled {
				switch im.gpsService {
				case 0: // Serial
					if strings.TrimSpace(im.gpsPort.Value()) == "" {
						im.SaveError = "GPS serial port is required"
						return im, nil
					}
				case 1: // GPSD
					if strings.TrimSpace(im.gpsdHost.Value()) == "" {
						im.SaveError = "GPSD host is required"
						return im, nil
					}
				}
			}
			// Validate APRS fields when APRS is enabled.
			if im.aprsEnabled {
				switch im.aprsService {
				case 0: // APRS-IS
					if strings.TrimSpace(im.aprsServer.Value()) == "" {
						im.SaveError = "APRS server is required"
						return im, nil
					}
				case 1: // KISS
					if strings.TrimSpace(im.aprsPort.Value()) == "" {
						im.SaveError = "APRS KISS port is required"
						return im, nil
					}
				}
			}
			im.done = true
			im.saved = true
			return im, nil
		case " ", "space":
			switch im.focus {
			case imDXCChk:
				im.dxcEnabled = !im.dxcEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
				return im, nil
			case imHTTPChk:
				im.httpEnabled = !im.httpEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
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
			case imGPSChk:
				im.gpsEnabled = !im.gpsEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
				return im, nil
			case imAPRSChk:
				im.aprsEnabled = !im.aprsEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
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
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
				return im, nil
			case imGPSGridPrec:
				im.gpsGridPrecision = nextGPSCycleInt(im.gpsGridPrecision, gpsPrecisionOptions)
				return im, nil
			case imAPRSSvc:
				im.aprsService = (im.aprsService + 1) % len(aprsServiceOptions)
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				im.autoScrollViewport()
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
			switch im.focus {
			case imDXCHost:
				im.dxcHost, _ = im.dxcHost.Update(msg)
			case imDXCPort:
				im.dxcPort, _ = im.dxcPort.Update(msg)
			case imDXCLogin:
				im.dxcLogin, _ = im.dxcLogin.Update(msg)
			case imQRZUser:
				im.qrzUser, _ = im.qrzUser.Update(msg)
			case imQRZPass:
				im.qrzPass, _ = im.qrzPass.Update(msg)
			case imHTTPPort:
				im.httpPort, _ = im.httpPort.Update(msg)
			case imHTTPHdr1:
				im.httpHeader1, _ = im.httpHeader1.Update(msg)
			case imHTTPHdr2:
				im.httpHeader2, _ = im.httpHeader2.Update(msg)
			case imHTTPLogo:
				im.httpClubLogo, _ = im.httpClubLogo.Update(msg)
			case imGPSPort:
				im.gpsPort, _ = im.gpsPort.Update(msg)
			}
		case "tab", "down":
			im.next()
			im.autoScrollViewport()
		case "shift+tab", "up":
			im.prev()
			im.autoScrollViewport()
		case "enter":
			if im.focus == imGPSTest {
				switch im.gpsService {
				case 0: // Serial
					port := strings.TrimSpace(im.gpsPort.Value())
					baud := im.gpsBaudRate
					dtr := im.gpsDTR
					rts := im.gpsRTS
					if port == "" || baud == 0 {
						im.gpsTestResult = "Port and baud rate required"
						return im, nil
					}
					im.gpsTesting = true
					im.gpsTestResult = "Testing..."
					return im, func() tea.Msg {
						err := testGPSConnection(port, baud, dtr, rts)
						return gpsTestMsg{ok: err == nil, err: err}
					}
				case 1: // GPSD
					host := strings.TrimSpace(im.gpsdHost.Value())
					port := strings.TrimSpace(im.gpsdPort.Value())
					if host == "" {
						im.gpsTestResult = "GPSD host required"
						return im, nil
					}
					if port == "" {
						port = "2947"
					}
					im.gpsTesting = true
					im.gpsTestResult = "Testing..."
					return im, func() tea.Msg {
						err := testGPSDConnection(host, port)
						return gpsTestMsg{ok: err == nil, err: err}
					}
				}
			}
			if im.focus == imAPRSTest {
				switch im.aprsService {
				case 0: // APRS-IS
					srv := strings.TrimSpace(im.aprsServer.Value())
					if srv == "" {
						im.aprsTestResult = "Server is required"
						return im, nil
					}
					im.aprsTesting = true
					im.aprsTestResult = "Testing..."
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
						im.aprsTestResult = "OK — connection working"
						return im, nil
					}
					prt := strings.TrimSpace(im.aprsPort.Value())
					baud := im.aprsBaudRate
					if prt == "" || baud == 0 {
						im.aprsTestResult = "Port and baud rate required"
						return im, nil
					}
					par := intToParity(im.aprsParity)
					stop := intToStopBits(im.aprsStopBits)
					im.aprsTesting = true
					im.aprsTestResult = "Testing..."
					return im, func() tea.Msg {
						err := testKISSPort(prt, baud, im.aprsDataBits, par, stop, im.aprsDTR, im.aprsRTS)
						return aprsTestMsg{err: err}
					}
				case 2: // KISS Server
					host := strings.TrimSpace(im.aprsKISSHost.Value())
					port := strings.TrimSpace(im.aprsKISSPort.Value())
					if host == "" {
						im.aprsTestResult = "Host is required"
						return im, nil
					}
					if port == "" {
						port = "8001"
					}
					addr := net.JoinHostPort(host, port)
					im.aprsTesting = true
					im.aprsTestResult = "Testing..."
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
			im.next()
			im.autoScrollViewport()
		case "pgup":
			if im.focus == imGPSBaud {
				im.gpsBaudRate = nextGPSCycle(im.gpsBaudRate)
				return im, nil
			}
			if im.focus == imAPRSBaud {
				im.aprsBaudRate = nextGPSCycle(im.aprsBaudRate)
				return im, nil
			}
			im.vp, _ = im.vp.Update(msg)
		case "pgdown":
			if im.focus == imGPSBaud {
				im.gpsBaudRate = prevGPSCycle(im.gpsBaudRate)
				return im, nil
			}
			if im.focus == imAPRSBaud {
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
	switch im.focus {
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

func (im *IntegrationMenu) next() {
	for {
		im.focus = wrapNext(im.focus, imMax)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) prev() {
	for {
		im.focus = wrapPrev(im.focus, imMax)
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
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

func (im *IntegrationMenu) fixFocus() {
	if im.isPositionVisible(im.focus) {
		return
	}
	im.next()
}

func (im *IntegrationMenu) blurAll() {
	blurTextinputs(&im.dxcHost, &im.dxcPort, &im.dxcLogin, &im.httpPort, &im.httpHeader1, &im.httpHeader2, &im.httpClubLogo, &im.httpEvtStart, &im.gpsPort, &im.gpsdHost, &im.gpsdPort, &im.aprsServer, &im.aprsKISSHost, &im.aprsKISSPort, &im.aprsPort)
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
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
}

// scrollFraction returns 0.0 (top) to 1.0 (bottom) indicating the
// relative position of the currently focused field. Used to auto-scroll
// the viewport so the active field stays visible on small terminals.
// scrollFraction returns 0.0 (top) to 1.0 (bottom) indicating the relative
// position of the currently focused field among all currently visible focus
// positions. This adapts to collapsed sections (e.g. disabled DXC hides its
// sub-fields) so the viewport scrolls accurately regardless of which
// integrations are enabled.
func (im *IntegrationMenu) scrollFraction() float64 {
	visible := 0
	rank := -1
	for i := 0; i < imMax; i++ {
		if im.isPositionVisible(i) {
			visible++
		}
		if i == im.focus {
			rank = visible
		}
	}
	if visible <= 1 || rank <= 0 {
		return 0
	}
	return float64(rank-1) / float64(visible-1)
}

// autoScrollViewport adjusts the viewport Y offset to keep the focused
// field visible.
func (im *IntegrationMenu) autoScrollViewport() {
	total := im.vp.TotalLineCount()
	visible := im.vp.VisibleLineCount()
	if total <= visible {
		im.vp.SetYOffset(0)
		return
	}
	frac := im.scrollFraction()
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := int(float64(maxOffset) * frac)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	im.vp.SetYOffset(offset)
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
	infoLines := wrapLines(infoText, infoMaxW)
	var infoContent strings.Builder
	for i, line := range infoLines {
		infoContent.WriteString(DimStyle.Render(line))
		if i < len(infoLines)-1 {
			infoContent.WriteString("\n")
		}
	}
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(P.Border)
	infoBox := boxStyle.Render(infoContent.String())
	b.WriteString(infoBox)
	b.WriteString("\n")

	// --- DXC section ---
	dxcCheckbox := "[ ]"
	if im.dxcEnabled {
		dxcCheckbox = "[x]"
	}
	dxcPrefix := "  "
	dxcLabel := S.FormLabelWide.Align(lipgloss.Left).Render("DX Cluster:")
	if im.focus == imDXCChk {
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
	}

	b.WriteString("\n")
	b.WriteString(padOrTrunc("", lineW))
	b.WriteString("\n")

	// --- HTTP Server section ---
	httpCheckbox := "[ ]"
	if im.httpEnabled {
		httpCheckbox = "[x]"
	}
	httpPrefix := "  "
	httpLabel := S.FormLabelWide.Align(lipgloss.Left).Render("HTTP Server:")
	if im.focus == imHTTPChk {
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
		if im.focus == imHTTPAddr {
			addrLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("  Access:")
			addrVal = CursorStyle.Render(addrLabels[im.httpAddrIdx]) + " " + DimStyle.Render("(Space)")
		}
		b.WriteString(padOrTrunc(
			lipgloss.JoinHorizontal(lipgloss.Center, "  ", addrLabel, " ", addrVal),
			lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPPort, "  Port:", &im.httpPort, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderTheme(), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr1, "  Header 1 (opt):", &im.httpHeader1, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr2, "  Header 2 (opt):", &im.httpHeader2, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPLogo, "  Logo URL (opt):", &im.httpClubLogo, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPQRLink, "  QR Link (opt):", &im.httpQRLink, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPEvt, "  Event Start (opt):", &im.httpEvtStart, false), lineW))
	}

	b.WriteString("\n")
	b.WriteString(padOrTrunc("", lineW))
	b.WriteString("\n")

	// --- GPS section ---
	gpsCheckbox := "[ ]"
	if im.gpsEnabled {
		gpsCheckbox = "[x]"
	}
	gpsPrefix := "  "
	gpsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("GPS Service:")
	if im.focus == imGPSChk {
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
		if im.focus == imGPSSvc {
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
		if im.focus == imGPSGridPrec {
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
			if im.focus == imGPSBaud {
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
			if im.focus == imGPSDTR {
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
			if im.focus == imGPSRTS {
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
		var btnLine string
		if im.gpsTesting {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("...")
		} else if im.focus == imGPSTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, lineW))

		if im.gpsTestResult != "" {
			b.WriteString("\n")
			b.WriteString(padOrTrunc("    "+im.gpsTestResultStyled(), lineW))
		}
	}

	b.WriteString("\n")
	b.WriteString(padOrTrunc("", lineW))
	b.WriteString("\n")

	// --- APRS section ---
	aprsCheckbox := "[ ]"
	if im.aprsEnabled {
		aprsCheckbox = "[x]"
	}
	aprsPrefix := "  "
	aprsLabel := S.FormLabelWide.Align(lipgloss.Left).Render("APRS:")
	if im.focus == imAPRSChk {
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
		if im.focus == imAPRSSvc {
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
			if im.focus == imAPRSBaud {
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
			if im.focus == imAPRSData {
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
			if im.focus == imAPRSParity {
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
			if im.focus == imAPRSStop {
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
			if im.focus == imAPRSDTR {
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
			if im.focus == imAPRSRTS {
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
		var btnLine string
		if im.aprsTesting {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("...")
		} else if im.focus == imAPRSTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, lineW))
	}

	// Build raw form body — header is rendered separately above the viewport.
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
		im.autoScrollViewport()
	}
	if im.vp.PastBottom() {
		im.autoScrollViewport()
	}

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
	if im.focus == focusIdx {
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
	if im.focus == focusIdx {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render(label)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

func (im *IntegrationMenu) renderTheme() string {
	prefix := "  "
	lbl := S.FormLabelWide.Align(lipgloss.Left).Render("  Theme:")
	if im.focus == imHTTPTheme {
		prefix = S.FormPrefixOn.Render("> ")
		lbl = S.FormFocusedWide.Align(lipgloss.Left).Render("  Theme:")
	}
	themeNames := []string{"Bright", "Dark", "Orchid", "HighVis"}
	val := ValueStyle.Render(themeNames[im.httpTheme])
	if im.focus == imHTTPTheme {
		val = CursorStyle.Render(val) + " " + DimStyle.Render("(Space)")
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, prefix, lbl, " ", val)
}

// Values returns DXC, QRZ, and HTTP server config values.
func (im *IntegrationMenu) Values() (dxcEnabled bool, dxcHost, dxcPort, dxcLogin string, qrzEnabled bool, qrzUser, qrzPass string, httpEnabled bool, httpAddr, httpPort, httpTheme string, httpHdr1, httpHdr2, httpLogo, httpQRLink, httpEvtStart string) {
	return im.dxcEnabled,
		strings.TrimSpace(im.dxcHost.Value()),
		strings.TrimSpace(im.dxcPort.Value()),
		strings.TrimSpace(im.dxcLogin.Value()),
		im.qrzEnabled,
		strings.TrimSpace(im.qrzUser.Value()),
		im.qrzPass.Value(),
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
		strings.TrimSpace(im.httpEvtStart.Value())
}

// gpsTestResultStyled returns the GPS test result with appropriate styling.
func (im *IntegrationMenu) gpsTestResultStyled() string {
	if im.gpsTesting {
		return DimStyle.Render(im.gpsTestResult)
	}
	if strings.HasPrefix(im.gpsTestResult, "OK") {
		return SuccessStyle.Render(im.gpsTestResult)
	}
	return ErrorStyle.Render(im.gpsTestResult)
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
		applog.Info("GPSD test: TPV received",
			"lat", fmt.Sprintf("%.6f", lat),
			"lon", fmt.Sprintf("%.6f", lon),
			"mode", fmt.Sprintf("%.0f", mode),
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
