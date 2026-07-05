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
	"github.com/szporwolik/cqops/internal/qrz"
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
	httpAddr     textinput.Model
	httpPort     textinput.Model
	httpHeader1  textinput.Model
	httpHeader2  textinput.Model
	httpClubLogo textinput.Model
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

	focus  int
	done   bool
	saved  bool
	goBack bool
	width  int
	height int

	// saveError is set when Ctrl+S is blocked by validation.
	// The parent reads it to show a toast, then clears it.
	SaveError string

	// Viewport for scrolling form content on small terminals.
	vp              viewport.Model
	lastBodyContent string
}

const (
	imDXCChk      = 0
	imDXCHost     = 1
	imDXCPort     = 2
	imDXCLogin    = 3
	imQRZChk      = 4
	imQRZUser     = 5
	imQRZPass     = 6
	imQRZTest     = 7
	imHTTPChk     = 8
	imHTTPAddr    = 9
	imHTTPPort    = 10
	imHTTPHdr1    = 11
	imHTTPHdr2    = 12
	imHTTPLogo    = 13
	imHTTPEvt     = 14
	imGPSChk      = 15
	imGPSSvc      = 16 // service type: None / Serial / GPSD
	imGPSGridPrec = 17 // grid precision: 10 / 8 / 6
	imGPSPort     = 18 // serial port
	imGPSBaud     = 19 // baud rate
	imGPSDTR      = 20 // DTR
	imGPSRTS      = 21 // RTS
	imGPSDHost    = 22 // GPSD host
	imGPSDPort    = 23 // GPSD port
	imGPSTest     = 24 // test button
	imMax         = 25
)

type callbookTestMsg struct {
	ok  bool
	err error
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

func prevGPSCycleInt(current int, opts []int) int {
	for i := len(opts) - 1; i >= 0; i-- {
		if opts[i] == current && i > 0 {
			return opts[i-1]
		}
	}
	return opts[len(opts)-1]
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
	qrzUser.SetValue(cfg.Integrations.QRZ.User)

	qrzPass := newTextinput()
	qrzPass.CharLimit = 40
	qrzPass.SetWidth(28)
	qrzPass.Placeholder = "QRZ.com password"
	qrzPass.EchoMode = textinput.EchoPassword
	qrzPass.EchoCharacter = '*'
	qrzPass.SetValue(cfg.Integrations.QRZ.Pass)

	httpAddr := newTextinput()
	httpAddr.CharLimit = 40
	httpAddr.SetWidth(28)
	httpAddr.Placeholder = "0.0.0.0"
	if cfg.Integrations.HTTPServer.Address != "" {
		httpAddr.SetValue(cfg.Integrations.HTTPServer.Address)
	} else {
		httpAddr.SetValue("0.0.0.0")
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

	httpEvtStart := newTextinput()
	httpEvtStart.CharLimit = 10
	httpEvtStart.SetWidth(28)
	httpEvtStart.Placeholder = "YYYY-MM-DD (optional)"
	if cfg.Integrations.HTTPServer.EventStart != "" {
		httpEvtStart.SetValue(cfg.Integrations.HTTPServer.EventStart)
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

	return &IntegrationMenu{
		dxcEnabled:       cfg.Integrations.DXC.Enabled,
		dxcHost:          dxcHost,
		dxcPort:          dxcPort,
		dxcLogin:         dxcLogin,
		qrzEnabled:       cfg.Integrations.QRZ.Enabled,
		qrzUser:          qrzUser,
		qrzPass:          qrzPass,
		httpEnabled:      cfg.Integrations.HTTPServer.Enabled,
		httpAddr:         httpAddr,
		httpPort:         httpPort,
		httpHeader1:      httpHeader1,
		httpHeader2:      httpHeader2,
		httpClubLogo:     httpClubLogo,
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
			// Validate QRZ fields when QRZ is enabled.
			if im.qrzEnabled {
				if strings.TrimSpace(im.qrzUser.Value()) == "" {
					im.SaveError = "QRZ username is required when QRZ is enabled"
					return im, nil
				}
				if im.qrzPass.Value() == "" {
					im.SaveError = "QRZ password is required when QRZ is enabled"
					return im, nil
				}
			}
			// Validate HTTP server fields when HTTP server is enabled.
			if im.httpEnabled {
				if strings.TrimSpace(im.httpAddr.Value()) == "" {
					im.SaveError = "HTTP server address is required when HTTP server is enabled"
					return im, nil
				}
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
			case imQRZChk:
				im.qrzEnabled = !im.qrzEnabled
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
			case imGPSChk:
				im.gpsEnabled = !im.gpsEnabled
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
			case imHTTPAddr:
				im.httpAddr, _ = im.httpAddr.Update(msg)
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
			if im.focus == imQRZTest {
				if !im.inetOnline {
					im.qrzTestResult = "No internet connection"
					return im, nil
				}
				user := strings.TrimSpace(im.qrzUser.Value())
				pass := im.qrzPass.Value()
				if user == "" || pass == "" {
					im.qrzTestResult = "Username and password required"
					return im, nil
				}
				im.qrzTesting = true
				im.qrzTestResult = "Testing..."
				return im, func() tea.Msg {
					data, err := qrz.Lookup(user, pass, "SP9MOA")
					return callbookTestMsg{ok: err == nil && data != nil, err: err}
				}
			}
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
			im.next()
			im.autoScrollViewport()
		case "pgup":
			if im.focus == imGPSBaud {
				im.gpsBaudRate = nextGPSCycle(im.gpsBaudRate)
				return im, nil
			}
			im.vp, _ = im.vp.Update(msg)
		case "pgdown":
			if im.focus == imGPSBaud {
				im.gpsBaudRate = prevGPSCycle(im.gpsBaudRate)
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
	case imQRZUser:
		im.qrzUser, _ = im.qrzUser.Update(msg)
	case imQRZPass:
		im.qrzPass, _ = im.qrzPass.Update(msg)
	case imHTTPAddr:
		im.httpAddr, _ = im.httpAddr.Update(msg)
	case imHTTPPort:
		im.httpPort, _ = im.httpPort.Update(msg)
	case imHTTPHdr1:
		im.httpHeader1, _ = im.httpHeader1.Update(msg)
	case imHTTPHdr2:
		im.httpHeader2, _ = im.httpHeader2.Update(msg)
	case imHTTPLogo:
		im.httpClubLogo, _ = im.httpClubLogo.Update(msg)
	case imHTTPEvt:
		im.httpEvtStart, _ = im.httpEvtStart.Update(msg)
	case imGPSPort:
		im.gpsPort, _ = im.gpsPort.Update(msg)
	case imGPSDHost:
		im.gpsdHost, _ = im.gpsdHost.Update(msg)
	case imGPSDPort:
		im.gpsdPort, _ = im.gpsdPort.Update(msg)
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
	case imDXCChk, imQRZChk, imHTTPChk, imGPSChk:
		return true
	case imDXCHost, imDXCPort, imDXCLogin:
		return im.dxcEnabled
	case imQRZUser, imQRZPass, imQRZTest:
		return im.qrzEnabled
	case imHTTPAddr, imHTTPPort, imHTTPHdr1, imHTTPHdr2, imHTTPLogo, imHTTPEvt:
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
	blurTextinputs(&im.dxcHost, &im.dxcPort, &im.dxcLogin, &im.qrzUser, &im.qrzPass, &im.httpAddr, &im.httpPort, &im.httpHeader1, &im.httpHeader2, &im.httpClubLogo, &im.httpEvtStart, &im.gpsPort, &im.gpsdHost, &im.gpsdPort)
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case imDXCHost:
		im.dxcHost.Focus()
	case imDXCPort:
		im.dxcPort.Focus()
	case imDXCLogin:
		im.dxcLogin.Focus()
	case imQRZUser:
		im.qrzUser.Focus()
	case imQRZPass:
		im.qrzPass.Focus()
	case imHTTPAddr:
		im.httpAddr.Focus()
	case imHTTPPort:
		im.httpPort.Focus()
	case imHTTPHdr1:
		im.httpHeader1.Focus()
	case imHTTPHdr2:
		im.httpHeader2.Focus()
	case imHTTPLogo:
		im.httpClubLogo.Focus()
	case imHTTPEvt:
		im.httpEvtStart.Focus()
	case imGPSPort:
		im.gpsPort.Focus()
	case imGPSDHost:
		im.gpsdHost.Focus()
	case imGPSDPort:
		im.gpsdPort.Focus()
	}
}

// scrollFraction returns 0.0 (top) to 1.0 (bottom) indicating the
// relative position of the currently focused field. Used to auto-scroll
// the viewport so the active field stays visible on small terminals.
func (im *IntegrationMenu) scrollFraction() float64 {
	n := float64(im.focus)
	m := float64(imMax - 1)
	if m <= 0 {
		return 0
	}
	return n / m
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

	// --- QRZ section ---
	qrzCheckbox := "[ ]"
	if im.qrzEnabled {
		qrzCheckbox = "[x]"
	}
	qrzPrefix := "  "
	qrzLabel := S.FormLabelWide.Align(lipgloss.Left).Render("QRZ.com:")
	if im.focus == imQRZChk {
		qrzPrefix = S.FormPrefixOn.Render("> ")
		qrzLabel = S.FormFocusedWide.Align(lipgloss.Left).Render("QRZ.com:")
		qrzCheckbox = CursorStyle.Render(qrzCheckbox) + " " + DimStyle.Render("(Space)")
	}
	b.WriteString(padOrTrunc(
		lipgloss.JoinHorizontal(lipgloss.Center, qrzPrefix, qrzLabel, " ", qrzCheckbox),
		lineW))

	if im.qrzEnabled {
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imQRZUser, "  Username:", &im.qrzUser, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imQRZPass, "  Password:", &im.qrzPass, true), lineW))

		// Test button
		b.WriteString("\n")
		btnText := "[ Test Connection ]"
		var btnLine string
		if !im.inetOnline {
			btnLine = "    " + DimStyle.Render(btnText) + " " + DimStyle.Render("(offline)")
		} else if im.focus == imQRZTest {
			btnLine = S.FormPrefixOn.Render("> ") + CursorStyle.Render("  "+btnText)
		} else {
			btnLine = "    " + InputStyle.Render(btnText)
		}
		b.WriteString(padOrTrunc(btnLine, lineW))

		if im.qrzTestResult != "" {
			b.WriteString("\n    ")
			if im.qrzTesting {
				b.WriteString(DimStyle.Render(im.qrzTestResult))
			} else if strings.HasPrefix(im.qrzTestResult, "OK") {
				b.WriteString(SuccessStyle.Render(im.qrzTestResult))
			} else {
				b.WriteString(ErrorStyle.Render(im.qrzTestResult))
			}
		}
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
		b.WriteString(padOrTrunc(im.renderField(imHTTPAddr, "  Address:", &im.httpAddr, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPPort, "  Port:", &im.httpPort, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr1, "  Header 1 (opt):", &im.httpHeader1, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPHdr2, "  Header 2 (opt):", &im.httpHeader2, false), lineW))
		b.WriteString("\n")
		b.WriteString(padOrTrunc(im.renderField(imHTTPLogo, "  Logo URL (opt):", &im.httpClubLogo, false), lineW))
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

	// Build raw form body — header is rendered separately above the viewport.
	bodyStr := b.String()

	// Wrap in viewport for scrolling on small terminals.
	boxW := w
	if boxW > partnerMapMaxW {
		boxW = partnerMapMaxW
	}
	vpW := boxW - 4 // account for menu box border + padding
	if vpW < 20 {
		vpW = 20
	}
	contentH := contentHeight(h)
	if contentH < 8 {
		contentH = 8
	}
	// Reserve one line for the scroll indicator inside the box.
	vpH := contentH - 6 // header(1) + box border/padding(4) + hint(1)
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
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, box))
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

// Values returns DXC, QRZ, and HTTP server config values.
func (im *IntegrationMenu) Values() (dxcEnabled bool, dxcHost, dxcPort, dxcLogin string, qrzEnabled bool, qrzUser, qrzPass string, httpEnabled bool, httpAddr, httpPort, httpHdr1, httpHdr2, httpLogo, httpEvtStart string) {
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
		strings.TrimSpace(im.httpHeader1.Value()),
		strings.TrimSpace(im.httpHeader2.Value()),
		strings.TrimSpace(im.httpClubLogo.Value()),
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
