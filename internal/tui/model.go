package tui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/log"
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig/flrig"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
)

type field int

const (
	fieldCall field = iota
	fieldRSTSent
	fieldRSTRcvd
	fieldBand
	fieldFreq
	fieldMode
	fieldSubmode
	fieldDate
	fieldTime
	fieldGrid
	fieldCountry
	fieldName
	fieldQTH
	fieldTXPower
	fieldComment
	fieldCount
)

var fieldNames = []string{
	"Call", "RST sent", "RST rcvd", "Band", "Frequency", "Mode", "Submode",
	"Date UTC", "Time UTC",
	"Grid", "DXCC", "Name", "QTH", "Power W", "Comment",
}

type Model struct {
	App          *app.App
	fields       [fieldCount]textinput.Model
	focus        field
	qsos         []qso.QSO
	toasts       *ToastQueue
	err          error
	width        int
	height       int
	quitting     bool
	rigConnected bool
	rigFreq      float64
	rigMode      string
	rigPower     float64
	rigBlink     bool
	dateTimeAuto bool
	tickCount    int
	inetOnline   bool
	showChooser  bool
	chooser      *LogbookChooser
	showRigEdit  bool
	rigChooser   *RigChooser
	showConfig   bool
	configMenu   *GeneralMenu
	showCallbook bool
	callbookMenu *CallbookMenu
	showMainMenu bool
	showLogView  bool
	logViewer    *LogViewer
	mainMenu     *MainMenu
	confirmQuit  bool
	showPartner  bool
	partnerData  *qrz.CallData
	partnerASCII string
	asciiW int
	asciiH int
	resizeSeq   int
	partnerDirty bool
	flrigClient  *flrig.Flrig
	qrzNeed      bool
	qrzCall      string
}

type tickMsg time.Time
type qrzResultMsg struct{ Call string; Data *qrz.CallData; Err error }
type inetResultMsg bool
type resizeSettledMsg struct{ Width, Height, Seq int }

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	m := &Model{App: a, qsos: initialQSOS, toasts: NewToastQueue(), dateTimeAuto: true}
	now := time.Now().UTC()
	for i := field(0); i < fieldCount; i++ {
		ti := textinput.New()
		ti.Prompt = ""
		ti.CharLimit = 40
		switch i {
		case fieldCall: ti.Focus()
		case fieldBand: ti.CharLimit = 8
		case fieldFreq: ti.CharLimit = 16
		case fieldMode: ti.CharLimit = 12
		case fieldSubmode: ti.CharLimit = 16
		case fieldDate: ti.CharLimit = 10; ti.SetValue(now.Format("2006-01-02"))
		case fieldTime: ti.CharLimit = 8; ti.SetValue(now.Format("15:04:05"))
		case fieldGrid: ti.CharLimit = 8
		case fieldCountry: ti.CharLimit = 20
		case fieldName: ti.CharLimit = 30
		case fieldQTH: ti.CharLimit = 30
		case fieldTXPower: ti.CharLimit = 8
		case fieldComment: ti.CharLimit = 60
		}
		m.fields[i] = ti
	}
	m.focus = fieldCall
	return m
}

func (m *Model) Init() tea.Cmd { m.refreshFlrigClient(); return tea.Batch(tickCmd(), checkInetCmd()) }
func tickCmd() tea.Cmd { return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }) }
func (m *Model) qrzLookupCmd(call string) tea.Cmd {
	return func() tea.Msg {
		data, err := qrz.Lookup(m.App.Config.QRZUser, m.App.Config.QRZPass, call)
		return qrzResultMsg{Call: call, Data: data, Err: err}
	}
}

func (m *Model) qrzLookup(call string) tea.Cmd {
	m.toasts.Info("QRZ: looking up " + call + "…")
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

func resizeDebounceCmd(w, h, seq int) tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return resizeSettledMsg{Width: w, Height: h, Seq: seq}
	})
}

func (m *Model) refreshFlrigClient() {
	if len(m.App.Config.Rigs) == 0 {
		s := m.App.Logbook.Station
		m.App.Config.Rigs = map[string]config.RigPreset{"default": {
			Model: s.Rig, Antenna: s.Antenna, Power: s.Power,
			FlrigEnabled: m.App.Config.Rig.Flrig.Enabled, FlrigHost: "localhost", FlrigPort: "12345",
		}}
	}
	rigName := m.App.Logbook.Station.RigName
	if rigName == "" { rigName = "default" }
	if rp, ok := m.App.Config.Rigs[rigName]; ok && rp.FlrigEnabled {
		host, port := rp.FlrigHost, rp.FlrigPort
		if host == "" { host = "localhost" }
		if port == "" { port = "12345" }
		m.flrigClient = flrig.New("http://"+host+":"+port, 1000)
	} else { m.flrigClient = nil }
}

func (m *Model) pollFlrig() {
	m.rigBlink = !m.rigBlink
	if m.flrigClient == nil { m.rigConnected = false; return }
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	status, err := m.flrigClient.Status(ctx)
	if err != nil || !status.Connected { m.rigConnected = false; return }
	m.rigConnected = true
	m.rigFreq = status.FrequencyMHz
	m.fields[fieldFreq].SetValue(fmt.Sprintf("%.6f", status.FrequencyMHz))
	if status.Mode != "" { m.fields[fieldMode].SetValue(status.Mode) }
	if status.Band != "" { m.fields[fieldBand].SetValue(status.Band) }
	m.autoFillSSBSubmode()
	if status.Power > 0 {
		m.rigPower = status.Power
		m.fields[fieldTXPower].SetValue(fmt.Sprintf("%.0f", status.Power))
		rigName := m.App.Logbook.Station.RigName
		if rigName == "" { rigName = "default" }
		if rp, ok := m.App.Config.Rigs[rigName]; ok {
			rp.Power = fmt.Sprintf("%.0f", status.Power)
			m.App.Config.Rigs[rigName] = rp
		}
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
	if m.tickCount%300 == 0 {
		return checkInetCmd()
	}
	return tickCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if _, ok := msg.(tickMsg); ok {
		m.pollFlrig()
		m.toasts.Expire()
		m.autoUpdateDateTime()
		m.tickCount++
		cmd = m.maybeCheckInet()
	}
	if m.confirmQuit {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "y", "Y": return m, tea.Quit
			default: m.confirmQuit = false
			}
		}
		return m, cmd
	}
		if ir, ok := msg.(inetResultMsg); ok {
			m.inetOnline = bool(ir)
		}
		if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "f10": m.confirmQuit = true
		case "f1": m.showChooser = false; m.showRigEdit = false; m.showConfig = false; m.showMainMenu = false; m.showLogView = false; m.showPartner = false
		case "f2": if m.showPartner { m.showPartner = false } else if m.partnerData != nil { m.showPartner = true }
		case "f8": if m.showMainMenu { m.showMainMenu = false } else { m.mainMenu = NewMainMenu(); m.showMainMenu = true }
		case "f9": m.logViewer = NewLogViewer(m.App.LogbookName); m.showLogView = true
		}
		if !m.showChooser && !m.showRigEdit && !m.showConfig && !m.showCallbook && !m.showMainMenu && !m.showLogView && !m.showPartner {
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
		_, _ = m.chooser.Update(msg)
		if m.chooser.done {
			m.showChooser = false
			m.showMainMenu = true
			m.qsos = nil
		}
		return m, cmd
	}
	if m.showRigEdit {
		_, _ = m.rigChooser.Update(msg)
		if m.rigChooser.done {
			m.showRigEdit = false
			m.showMainMenu = true
			m.refreshFlrigClient()
		}
		return m, cmd
	}
	if m.showConfig {
		_, _ = m.configMenu.Update(msg)
		if m.configMenu.done {
			m.showConfig = false
			if m.configMenu.goBack { m.showMainMenu = true }
			if m.configMenu.saved {
				m.App.Config.RenderImages = m.configMenu.renderImages
				m.App.Config.DistanceUnit = m.configMenu.distanceUnit
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					log.Info("Settings saved")
				}
				m.showMainMenu = true
			}
		}
		return m, cmd
	}
	if m.showCallbook {
		_, _ = m.callbookMenu.Update(msg)
		if m.callbookMenu.done {
			m.showCallbook = false
			if m.callbookMenu.goBack { m.showMainMenu = true }
			if m.callbookMenu.saved {
				m.App.Config.QRZUser = m.callbookMenu.user.Value()
				m.App.Config.QRZPass = m.callbookMenu.pass.Value()
				m.App.Config.QRZEnabled = m.callbookMenu.enabled
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					log.Info("Settings saved")
				}
				m.showMainMenu = true
			}
		}
		return m, cmd
	}
	if m.showMainMenu {
		_, _ = m.mainMenu.Update(msg)
		if m.mainMenu.action != "" {
			action := m.mainMenu.action
			m.mainMenu.action = ""
			m.showMainMenu = false
			switch action {
			case "general": m.configMenu = NewGeneralMenu(m.App.Config); m.showConfig = true
			case "callbook": m.callbookMenu = NewCallbookMenu(m.App.Config); m.showCallbook = true
			case "logbook": m.chooser = NewLogbookChooser(m.App, m.toasts); m.showChooser = true
			case "rig": m.rigChooser = NewRigChooser(m.App, m.toasts); m.showRigEdit = true
			}
		}
		if m.mainMenu.done {
			m.showMainMenu = false
		}
		return m, cmd
	}
	if m.showPartner {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height
			m.asciiW = 0
			m.asciiH = 0
			m.partnerDirty = false
			m.resizeSeq++
			seq := m.resizeSeq
			return m, resizeDebounceCmd(msg.Width, msg.Height, seq)
		case qrzResultMsg:
			m.fillQRZData(msg)
			return m, cmd
		case inetResultMsg:
			m.inetOnline = bool(msg)
			return m, cmd
		case resizeSettledMsg:
			if msg.Seq == m.resizeSeq {
				m.partnerDirty = true
			}
			return m, cmd
		case tea.KeyMsg:
			switch {
			case msg.String() == "f8": m.showPartner = false
		}
		return m, cmd
	}
}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.partnerDirty = false
		m.resizeSeq++
		return m, resizeDebounceCmd(msg.Width, msg.Height, m.resizeSeq)
	case qrzResultMsg: m.fillQRZData(msg); return m, cmd
	case inetResultMsg: m.inetOnline = bool(msg); return m, nil
	case resizeSettledMsg:
		if msg.Seq == m.resizeSeq {
			m.partnerDirty = true
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab: m.prevField()
		case msg.Type == tea.KeyUp || msg.String() == "up": m.prevField()
		case msg.Type == tea.KeyDown || msg.String() == "down": m.nextField()
		case msg.String() == "ctrl+s": return m, m.saveQSO()
		case msg.String() == "delete" || msg.Type == tea.KeyDelete: m.clearForm()
		case msg.String() == "ctrl+c": m.mainMenu = NewMainMenu(); m.showMainMenu = true
		case msg.String() == "f1": m.focusField(fieldCall)
		case msg.String() == "f2":
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled && m.partnerData == nil {
				return m, m.qrzLookup(call)
			}
			if m.partnerData != nil {
				m.showPartner = true
			}
		case msg.String() == "insert" || msg.Type == tea.KeyInsert || msg.String() == "\x1b[2~":
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled {
				return m, m.qrzLookup(call)
			}
		case msg.String() == "tab" || msg.String() == "\t" || msg.Type == tea.KeyTab: m.nextField()
		case msg.String() == "enter": return m, m.saveQSO()
		case msg.Type == tea.KeyPgUp || msg.String() == "pgup": m.cycleFieldUp()
		case msg.Type == tea.KeyPgDown || msg.String() == "pgdown": m.cycleFieldDown()
		default: m.updateFocused(msg)
		}
	}
	if m.qrzNeed {
		m.qrzNeed = false
		call := m.qrzCall
		if call == "" { return m, cmd }
		if !m.App.Config.QRZEnabled { return m, cmd }
		if m.App.Config.QRZUser == "" { m.toasts.Warn("QRZ not configured — F8 Config → Callbook / QRZ.com to enable"); return m, cmd }
		return m, tea.Batch(cmd, m.qrzLookup(call))
	}
	return m, cmd
}

func (m *Model) fillQRZData(msg qrzResultMsg) {
	if msg.Call == "" { return }
	if !m.App.Config.QRZEnabled || m.App.Config.QRZUser == "" { m.toasts.Warn("QRZ not configured"); return }
	if msg.Err != nil {
		m.toasts.Error("QRZ error: "+msg.Err.Error())
		return
	}
	d := msg.Data
	if d == nil || d.Callsign == "" { m.toasts.Warn("QRZ: no data for "+msg.Call); return }
	m.partnerData = d
	m.partnerASCII = ""
	m.asciiW = 0
	m.asciiH = 0
	if d.Name != "" { m.fields[fieldName].SetValue(d.Name) }
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" { m.fields[fieldGrid].SetValue(formatLocator(d.Grid)) }
	if d.QTH != "" { m.fields[fieldQTH].SetValue(d.QTH) }
	if d.Country != "" && m.fields[fieldCountry].Value() == "" { m.fields[fieldCountry].SetValue(d.Country) }
	m.toasts.Info("QRZ: "+d.Callsign+" "+d.Name)
}

func (m *Model) View() string {
	if m.quitting { return "" }
	if m.err != nil { return errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)) }
	w := m.width; if w < 40 { w = 80 }
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
	} else if m.showMainMenu {
		content = m.mainMenu.View()
	} else if m.showLogView {
		content = m.logViewer.View()
	} else if m.showPartner && m.partnerData != nil {
		content = m.viewPartner()
	} else {
		form := m.viewForm(w)
		distLine := m.formDistanceLine(w)
		qsoList := m.viewQSOS(m.availableQSORows())
		content = lipgloss.JoinVertical(lipgloss.Left, form, distLine, qsoList)
	}
	body := lipgloss.NewStyle().Width(w).Padding(0, 1).Render(content)
	toastBar := RenderToasts(m.toasts.Active(), w)
	footer := m.viewFooter(w)
	mainBlock := lipgloss.JoinVertical(lipgloss.Left, header, body)
	mainLines := strings.Count(mainBlock, "\n") + 1
	toastLines := strings.Count(toastBar, "\n")
	if toastBar != "" {
		toastLines++
	}
	footerLines := strings.Count(footer, "\n") + 1
	extraLines := toastLines + footerLines
	if m.height > 0 {
		pad := m.height - mainLines - extraLines
		if pad < 0 {
			pad = 0
		}
		mainBlock += strings.Repeat("\n", pad)
	}
	var all string
	if toastBar != "" {
		all = lipgloss.JoinVertical(lipgloss.Left, mainBlock, toastBar, footer)
	} else {
		all = lipgloss.JoinVertical(lipgloss.Left, mainBlock, footer)
	}
	return all
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
					rigIndicator = ansiFg("46", " on")
				} else {
					rigIndicator = "   "
				}
			} else {
				rigIndicator = ansiFg("196", " err")
			}
		}
	}

	locator := formatLocator(s.Grid)
	if locator == "" {
		locator = "----"
	}

	inetVal := ansiFg("196", "No ")
	if m.inetOnline {
		inetVal = ansiFg("46", "Yes")
	}

	left := ansiFg("243", "Call: ") + ansiFg("229", clamp(s.Callsign, 8)) +
		ansiFg("243", "  Op: ") + ansiFg("229", clamp(s.Operator, 8)) +
		ansiFg("243", "  Log: ") + ansiFg("229", clamp(m.App.LogbookName, 8)) +
		ansiFg("243", "  Loc: ") + ansiFg("229", clamp(locator, 6))

	center := ansiBoldFg("86", "CQOPS")
	if v := version.Resolved(); v != "dev" {
		center += ansiFg("245", " v"+v)
	}

	right := ansiFg("243", "Inet: ") + inetVal +
		ansiFg("243", "  Rig: ") + ansiFg("229", rigModel) + rigIndicator +
		ansiFg("243", "  LT: ") + ansiFg("229", now.Format("15:04")) +
		ansiFg("243", "  UTC: ") + ansiFg("229", utc.Format("15:04:05"))

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	totalW := leftW + rightW

	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))

	if totalW+6 > width {
		line1 := "  " + left
		for lipgloss.Width(line1) < width {
			line1 += " "
		}
		line1 = bgStyle.Render(line1)

		line2 := right
		for lipgloss.Width(line2) < width-2 {
			line2 = " " + line2
		}
		line2 = "  " + line2
		for lipgloss.Width(line2) < width {
			line2 += " "
		}
		line2 = bgStyle.Render(line2)

		tabLine := m.renderTabLine(width)
		return line1 + "\n" + line2 + "\n" + tabLine
	}

	gap := width - 4 - totalW
	if gap < 2 {
		gap = 2
	}

	statusLine := "  " + left + strings.Repeat(" ", gap) + right
	for lipgloss.Width(statusLine) < width {
		statusLine += " "
	}
	statusLine = bgStyle.Render(statusLine)

	tabLine := m.renderTabLine(width)
	return statusLine + "\n" + tabLine
}

func ansiFg(color, s string) string {
	return "\x1b[38;5;" + color + "m" + s + "\x1b[39m"
}

func ansiBoldFg(color, s string) string {
	return "\x1b[1m\x1b[38;5;" + color + "m" + s + "\x1b[22m\x1b[39m"
}

func (m *Model) renderTabLine(width int) string {
	active := lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("229")).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("241")).Padding(0, 2)
	disabled := lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("238")).Padding(0, 2)

	qsoLabel := "F1 QSO Form"
	partnerLabel := "F2 Partner Details"
	configLabel := "F8 Config"
	logsLabel := "F9 Logs"

	var qsoTab, partnerTab, configTab, logsTabStr string

	if m.showPartner && m.partnerData != nil {
		qsoTab = inactive.Render(qsoLabel)
		partnerTab = active.Render(partnerLabel)
		configTab = inactive.Render(configLabel)
		logsTabStr = inactive.Render(logsLabel)
	} else if m.partnerData != nil {
		qsoTab = active.Render(qsoLabel)
		partnerTab = inactive.Render(partnerLabel)
		configTab = inactive.Render(configLabel)
		logsTabStr = inactive.Render(logsLabel)
	} else if m.showMainMenu {
		qsoTab = inactive.Render(qsoLabel)
		partnerTab = disabled.Render(partnerLabel)
		configTab = active.Render(configLabel)
		logsTabStr = inactive.Render(logsLabel)
	} else if m.showLogView {
		qsoTab = inactive.Render(qsoLabel)
		partnerTab = disabled.Render(partnerLabel)
		configTab = inactive.Render(configLabel)
		logsTabStr = active.Render(logsLabel)
	} else {
		qsoTab = active.Render(qsoLabel)
		partnerTab = disabled.Render(partnerLabel)
		configTab = inactive.Render(configLabel)
		logsTabStr = inactive.Render(logsLabel)
	}

	line := " " + qsoTab + partnerTab + configTab + logsTabStr
	for lipgloss.Width(line) < width {
		line += " "
	}
	return lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
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
	w := m.width
	h := m.height
	bodyW := w - 2
	if bodyW < 30 {
		bodyW = 30
	}
	headerH := 3
	footerH := 1

	minDetailsW := 44
	prefDetailsW := 60
	minImageW := 22
	gap := 3

	detailsW := bodyW
	imageW := 0

	if bodyW >= prefDetailsW+minImageW+gap {
		detailsW = prefDetailsW
		imageW = bodyW - detailsW - gap
	} else if bodyW >= minDetailsW+minImageW+gap {
		detailsW = bodyW - minImageW - gap
		imageW = minImageW
	}

	showImage := imageW >= minImageW && m.App.Config.RenderImages && d.ImageURL != ""

	var b strings.Builder

	title := "── Partner: " + d.Callsign + " "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
	b.WriteString("\n\n")

	info := m.renderPartnerInfo(d, detailsW)

	if showImage {
		infoLines := strings.Count(info, "\n") + 1
		propH := imageW * 6 / 4
		imgH := infoLines
		if propH > imgH {
			imgH = propH
		}
		if imgH < 8 {
			imgH = 8
		}
		maxAvailH := h - headerH - footerH - 10
		if imgH > maxAvailH {
			imgH = maxAvailH
		}
		if imgH > 22 {
			imgH = 22
		}
		if m.partnerDirty || m.partnerASCII == "" || m.asciiW != imageW || m.asciiH != imgH {
			ascii, _ := downloadAndRenderASCII(d.ImageURL, imageW, imgH)
			m.partnerASCII = ascii
			m.asciiW = imageW
			m.asciiH = imgH
			m.partnerDirty = false
		}
		leftBlock := lipgloss.NewStyle().Width(detailsW).Render(info)
		rightContent := m.partnerASCII
		if rightContent == "" {
			rightContent = lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("(no image)")
		}
		rightBlock := rightContent
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, strings.Repeat(" ", gap), rightBlock))
	} else {
		b.WriteString(info)
	}

	dl := m.partnerDistanceLine(bodyW)
	if dl != "" {
		b.WriteString("\n\n")
		pathTitle := "── Path "
		pathRem := bodyW - lipgloss.Width(pathTitle)
		if pathRem > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(pathTitle + strings.Repeat("─", pathRem)))
		}
		b.WriteString("\n  ")
		b.WriteString(inputStyle.Render(dl))
	}

	usedLines := strings.Count(b.String(), "\n") + 1
	availMapH := h - headerH - footerH - usedLines - 2
	if availMapH >= 6 {
		b.WriteString("\n\n")
		mapTitle := "── Map "
		mapRem := bodyW - lipgloss.Width(mapTitle)
		if mapRem > 0 {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(mapTitle + strings.Repeat("─", mapRem)))
		}
		b.WriteString("\n")
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
		ownGrid := m.App.Logbook.Station.Grid
		partnerGrid := d.Grid
		if ownGrid != "" && partnerGrid != "" {
			ownLat, ownLon := gridToLatLon(ownGrid)
			partnerLat, partnerLon := gridToLatLon(partnerGrid)
			if ownLat != 0 || ownLon != 0 || partnerLat != 0 || partnerLon != 0 {
				mapStr := renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, mapH)
				b.WriteString(mapStr)
			}
		}
	} else {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Render("── Map hidden: terminal too small"))
	}

	return b.String()
}

func (m *Model) partnerDistanceLine(width int) string {
	if m.partnerData == nil {
		return ""
	}
	own := formatLocator(m.App.Logbook.Station.Grid)
	partner := formatLocator(m.partnerData.Grid)
	return distanceLine(own, partner, m.App.Config.DistanceUnit)
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

	lblStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229"))

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
		text = "F10 Quit"
	case m.showConfig:
		text = m.configMenu.FooterText()
	case m.showCallbook:
		text = m.callbookMenu.FooterText()
	case m.showChooser:
		text = m.chooser.FooterText()
	case m.showRigEdit:
		text = m.rigChooser.FooterText()
	case m.showLogView:
		text = "↑↓ to scroll  F10 Quit"
	case m.showPartner && m.partnerData != nil:
			text = "F10 Quit"
	default:
		if width < 70 {
			text = "Enter=Save | Del Clear | Ins/Ctrl+L Lookup | PgUp/Dn Cycle | F10 Quit"
		} else {
			text = "Enter/Ctrl+S Save  Del Clear  Ins/Ctrl+L Lookup  PgUp/Dn Cycle  F10 Quit"
		}
	}
	ver := ""
	if v := version.Resolved(); v != "dev" {
		ver = "CQOPS v" + v
	}
	helpStr := ansiFg("241", text)
	verStr := ansiFg("240", ver)
	helpW := lipgloss.Width(helpStr)
	verW := lipgloss.Width(verStr)
	innerW := width - 4
	if helpW+verW > innerW {
		line := "  " + helpStr
		for lipgloss.Width(line) < width {
			line += " "
		}
		return lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
	}
	gap := innerW - helpW - verW
	line := "  " + helpStr + strings.Repeat(" ", gap) + verStr + "  "
	for lipgloss.Width(line) < width {
		line += " "
	}
	return lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
}

func truncate(s string, max int) string { if max < 3 { return s }; if lipgloss.Width(s) <= max { return s }; return s[:max-1] + "…" }

func (m *Model) viewForm(width int) string {
	bodyW := width - 2
	if bodyW < 20 {
		bodyW = 20
	}
	labelW := 12
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	hl := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	var b strings.Builder

	title := "── QSO "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
	b.WriteString("\n")

	choiceFields := map[field]bool{fieldBand: true, fieldMode: true, fieldSubmode: true}

	for i := field(0); i < fieldCount; i++ {
		label := fieldNames[i]
		raw := strings.TrimSpace(m.fields[i].Value())
		display := raw
		if display == "" {
			display = dim.Render("—")
		}

		hasChoices := choiceFields[i] && raw != ""
		choiceIcon := ""
		if hasChoices {
			choiceIcon = dim.Render("▼ ")
		}

		if int(i) == int(m.focus) {
			b.WriteString(hl.Render(fmt.Sprintf("  %-*s", labelW, label)))
			b.WriteString(" ")
			b.WriteString(inputStyle.Render(choiceIcon + m.fields[i].View()))
		} else {
			b.WriteString(fmt.Sprintf("  %-*s", labelW, label))
			b.WriteString(" ")
			if raw == "" {
				b.WriteString(display)
			} else {
				b.WriteString(inputStyle.Render(choiceIcon + display))
			}
		}
		b.WriteString("\n")
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
	"Date":    {"Date", 10, false, func(q *qso.QSO) string { return formatDate(q.QSODate) }},
	"Time":    {"Time", 8, false, func(q *qso.QSO) string { return formatTime(q.TimeOn) }},
	"Call":    {"Call", 7, true, func(q *qso.QSO) string { return q.Call }},
	"Band":    {"Band", 5, false, func(q *qso.QSO) string { b := qso.NormalizeBand(q.Band); if b == "" && q.Freq > 0 { b = fmt.Sprintf("%.1f", q.Freq) }; return b }},
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
	"Dist":    {"Dist", 6, false, func(q *qso.QSO) string { if q.Distance > 0 { return fmt.Sprintf("%.0f", q.Distance) }; return "" }},
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
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	bodyW := m.width - 2
	if bodyW < 20 {
		bodyW = 20
	}

	var b strings.Builder
	title := "── Recent QSOs "
	rem := bodyW - lipgloss.Width(title)
	if rem > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem)))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title))
	}
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
	title := "── Path "
	rem := bodyW - lipgloss.Width(title)
	hdr := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title)
	if rem > 0 {
		hdr = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + strings.Repeat("─", rem))
	}
	return hdr + "\n  " + inputStyle.Render(dl)
}

func (m *Model) availableQSORows() int {
	if m.height <= 0 { return 5 }
	avail := m.height - 32
	if avail < 1 { avail = 1 }
	return avail
}

func (m *Model) focusField(f field) {
	m.fields[m.focus].Blur()
	m.focus = f
	m.fields[m.focus].Focus()
}

func (m *Model) nextField() {
	wasCall := m.focus == fieldCall
	m.fields[m.focus].Blur(); m.focus = (m.focus + 1) % fieldCount
	m.fields[m.focus].Focus()
	if wasCall {
		m.qrzNeed = true
		m.qrzCall = strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
		m.autoFillRST()
		m.autoFillSSBSubmode()
	}
}
func (m *Model) prevField() {
	wasCall := m.focus == fieldCall
	m.fields[m.focus].Blur()
	if m.focus == 0 { m.focus = fieldCount - 1 } else { m.focus-- }
	m.fields[m.focus].Focus()
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

func (m *Model) autoFillSSBSubmode() {
	mode := strings.ToUpper(strings.TrimSpace(m.fields[fieldMode].Value()))
	if mode != "SSB" {
		return
	}
	if strings.TrimSpace(m.fields[fieldSubmode].Value()) != "" {
		return
	}
	var freq float64
	fmt.Sscanf(m.fields[fieldFreq].Value(), "%f", &freq)
	if freq <= 0 {
		return
	}
	if freq < 10.0 {
		m.fields[fieldSubmode].SetValue("LSB")
	} else {
		m.fields[fieldSubmode].SetValue("USB")
	}
}

func (m *Model) updateFocused(msg tea.KeyMsg) {
	prevCall := strings.TrimSpace(m.fields[fieldCall].Value())
	prevVal := m.fields[m.focus].Value()
	m.fields[m.focus], _ = m.fields[m.focus].Update(msg)
	if (m.focus == fieldDate || m.focus == fieldTime) && m.fields[m.focus].Value() != prevVal {
		m.dateTimeAuto = false
	}
	if m.focus == fieldCall { m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value())) }
	if m.focus == fieldGrid {
		m.fields[m.focus].SetValue(formatLocator(m.fields[m.focus].Value()))
	}
	if m.focus == fieldCall {
		cur := strings.TrimSpace(m.fields[fieldCall].Value())
		if cur != prevCall && m.partnerData != nil && !strings.EqualFold(m.partnerData.Callsign, cur) {
			m.partnerData = nil
			m.partnerASCII = ""
			m.asciiW = 0
			m.asciiH = 0
			m.showPartner = false
		}
	}
}
func (m *Model) clearForm() {
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
	m.dateTimeAuto = true
	m.focus = fieldCall; m.fields[m.focus].Focus()
	m.partnerData = nil
	m.partnerASCII = ""
	m.asciiW = 0
	m.asciiH = 0
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
	fmt.Sscanf(m.fields[fieldFreq].Value(), "%f", &freq)
	qs.Call, qs.Band, qs.Freq = strings.ToUpper(m.fields[fieldCall].Value()), strings.ToUpper(m.fields[fieldBand].Value()), freq
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
	station := qso.StationInfo{StationCallsign: m.App.Logbook.Station.Callsign, Operator: m.App.Logbook.Station.Operator, MyGridSquare: m.App.Logbook.Station.Grid, MyRig: m.App.Logbook.Station.Rig, MyAntenna: m.App.Logbook.Station.Antenna, TXPower: m.App.Logbook.Station.Power}
	if qs.GridSquare != "" && station.MyGridSquare != "" {
		qs.Distance = gridDistanceKm(station.MyGridSquare, qs.GridSquare)
		bearStr := gridBearing(station.MyGridSquare, qs.GridSquare)
		if bearStr != "" {
			fmt.Sscanf(bearStr, "%f", &qs.Bearing)
		}
	}
	qso.ApplyStationDefaults(qs, station)
	if err := qso.ValidateForSave(qs); err != nil { m.toasts.Error(err.Error()); return nil }
	if _, err := store.InsertQSO(m.App.DB, qs); err != nil { m.toasts.Error(fmt.Sprintf("Save failed: %v", err)); return nil }
	m.clearForm(); m.toasts.Success(fmt.Sprintf("QSO saved: %s", qs.Call))
	return m.refreshQSOS()
}
func (m *Model) refreshQSOS() tea.Cmd {
	qsos, err := store.ListQSOs(m.App.DB, 30)
	if err != nil { m.toasts.Error(fmt.Sprintf("Refresh failed: %v", err)); return nil }
	m.qsos = qsos; return nil
}
