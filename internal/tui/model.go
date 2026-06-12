package tui

import (
	"context"
	"fmt"
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
	fieldDate
	fieldTime
	fieldGrid
	fieldCountry
	fieldName
	fieldCity
	fieldComment
	fieldCount
)

var fieldNames = []string{
	"Call:", "RST sent:", "RST rcvd:", "Band:", "Freq:", "Mode:",
	"Date (UTC):", "Time (UTC):",
	"Grid:", "Country:", "Name:", "City:", "Comment:",
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
	flrigClient  *flrig.Flrig
	qrzNeed      bool
	qrzCall      string
}

type tickMsg time.Time
type qrzResultMsg struct{ Call string; Data *qrz.CallData; Err error }

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	m := &Model{App: a, qsos: initialQSOS, toasts: NewToastQueue(), dateTimeAuto: true}
	now := time.Now().UTC()
	for i := field(0); i < fieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 40
		switch i {
		case fieldCall: ti.Focus()
		case fieldFreq: ti.CharLimit = 16
		case fieldDate: ti.CharLimit = 8; ti.SetValue(now.Format("20060102"))
		case fieldTime: ti.CharLimit = 6; ti.SetValue(now.Format("150405"))
		}
		m.fields[i] = ti
	}
	m.focus = fieldCall
	return m
}

func (m *Model) Init() tea.Cmd { m.refreshFlrigClient(); return tickCmd() }
func tickCmd() tea.Cmd { return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) }) }

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
	if status.Power > 0 {
		m.rigPower = status.Power
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
	m.fields[fieldDate].SetValue(now.Format("20060102"))
	m.fields[fieldTime].SetValue(now.Format("150405"))
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
		if m.confirmQuit {
			if key, ok := msg.(tea.KeyMsg); ok {
				switch key.String() {
				case "y", "Y": return m, tea.Quit
				default: m.confirmQuit = false
				}
			}
			return m, nil
		}
		if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "f10": m.confirmQuit = true
		case "f1": m.showChooser = false; m.showRigEdit = false; m.showConfig = false; m.showMainMenu = false; m.showLogView = false; m.showPartner = false
		case "f2": if m.showPartner { m.showPartner = false } else if m.partnerData != nil { m.showPartner = true }
		case "f8": m.mainMenu = NewMainMenu(); m.showMainMenu = true
		case "f9": m.logViewer = NewLogViewer(); m.showLogView = true
		}
		if !m.showChooser && !m.showRigEdit && !m.showConfig && !m.showCallbook && !m.showMainMenu && !m.showLogView && !m.showPartner {
			if key.String() == "delete" || key.Type == tea.KeyDelete {
				m.clearForm()
				return m, nil
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
		return m, nil
	}
	if m.showRigEdit {
		_, _ = m.rigChooser.Update(msg)
		if m.rigChooser.done {
			m.showRigEdit = false
			m.showMainMenu = true
			m.refreshFlrigClient()
		}
		return m, nil
	}
	if m.showConfig {
		_, _ = m.configMenu.Update(msg)
		if m.configMenu.done {
			m.showConfig = false
			if m.configMenu.goBack { m.showMainMenu = true }
			if m.configMenu.saved {
				m.App.Config.RenderImages = m.configMenu.renderImages
				if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
					m.toasts.Error("Settings save failed: " + err.Error())
				} else {
					m.toasts.Success("Settings saved")
					log.Info("Settings saved")
				}
				m.showMainMenu = true
			}
		}
		return m, nil
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
		return m, nil
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
		return m, nil
	}
	if m.showPartner {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.width, m.height = msg.Width, msg.Height
			m.asciiW = 0
			m.asciiH = 0
		case tickMsg:
			m.pollFlrig()
			m.toasts.Expire()
			m.autoUpdateDateTime()
			return m, tickCmd()
		case qrzResultMsg:
			m.fillQRZData(msg)
			return m, nil
		case tea.KeyMsg:
			switch {
			case msg.String() == "f8": m.showPartner = false
		}
		return m, nil
	}
}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: m.width, m.height = msg.Width, msg.Height
	case tickMsg: m.pollFlrig(); m.toasts.Expire(); m.autoUpdateDateTime(); return m, tickCmd()
	case qrzResultMsg: m.fillQRZData(msg); return m, nil
	case tea.KeyMsg:
		switch {
		case msg.String() == "shift+tab" || msg.Type == tea.KeyShiftTab: m.prevField()
		case msg.Type == tea.KeyUp || msg.String() == "up": m.prevField()
		case msg.Type == tea.KeyDown || msg.String() == "down": m.nextField()
		case msg.String() == "ctrl+s": return m, m.saveQSO()
		case msg.String() == "delete" || msg.Type == tea.KeyDelete: m.clearForm()
		case msg.String() == "ctrl+c": m.mainMenu = NewMainMenu(); m.showMainMenu = true
		case msg.String() == "f1":
		case msg.String() == "f2":
			call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
			if call != "" && m.App.Config.QRZUser != "" && m.App.Config.QRZEnabled && m.partnerData == nil {
				return m, func() tea.Msg {
					data, err := qrz.Lookup(m.App.Config.QRZUser, m.App.Config.QRZPass, call)
					return qrzResultMsg{Call: call, Data: data, Err: err}
				}
			}
			if m.partnerData != nil {
				m.showPartner = true
			}
		case msg.String() == "tab" || msg.String() == "\t" || msg.Type == tea.KeyTab: m.nextField()
		case msg.String() == "enter": return m, m.saveQSO()
		default: m.updateFocused(msg)
		}
	}
	if m.qrzNeed {
		m.qrzNeed = false
		call := m.qrzCall
		if call == "" { return m, nil }
		if !m.App.Config.QRZEnabled { return m, nil }
		if m.App.Config.QRZUser == "" { m.toasts.Warn("QRZ not configured — F8 Config → Callbook / QRZ.com to enable"); return m, nil }
		return m, func() tea.Msg { data, err := qrz.Lookup(m.App.Config.QRZUser, m.App.Config.QRZPass, call); return qrzResultMsg{Call: call, Data: data, Err: err} }
	}
	return m, nil
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
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" { m.fields[fieldGrid].SetValue(d.Grid) }
	if d.QTH != "" { m.fields[fieldCity].SetValue(d.QTH) }
	if d.Country != "" && m.fields[fieldCountry].Value() == "" { m.fields[fieldCountry].SetValue(d.Country) }
	m.toasts.Info("QRZ: "+d.Callsign+" "+d.Name)
}

func (m *Model) View() string {
	if m.quitting { return "" }
	if m.err != nil { return errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)) }
	w := m.width; if w < 40 { w = 80 }
	topBar := m.viewTopBar(w)
	tabBar := m.viewTabBar(w)
	var content string
	if m.showChooser {
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
	} else if m.confirmQuit {
		content = titleStyle.Render("Quit CQOps? (y/N)")
	} else {
		form := m.viewForm(w)
		qsoList := m.viewQSOS(m.availableQSORows())
		content = lipgloss.JoinVertical(lipgloss.Left, form, qsoList)
	}
	body := lipgloss.NewStyle().Width(w).Padding(0, 1).Render(content)
	toastBar := RenderToasts(m.toasts.Active(), w)
	footer := m.viewFooter(w)
	mainBlock := lipgloss.JoinVertical(lipgloss.Left, topBar, tabBar, "", body)
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

func (m *Model) viewTopBar(width int) string {
	s := m.App.Logbook.Station
	now := time.Now(); utc := now.UTC()

	rigDisplay := ""
	rigName := s.RigName; if rigName == "" { rigName = "default" }
	if rp, ok := m.App.Config.Rigs[rigName]; ok {
		rigDisplay = rp.Model
		if rp.FlrigEnabled {
			if m.rigConnected { if m.rigBlink { rigDisplay += " *" } else { rigDisplay += "  " } } else { rigDisplay += " !" }
		}
	}

	versionText := "CQOPS v" + version.Resolved(); if version.Resolved() == "dev" { versionText = "CQOPS" }

	innerW := width - 4
	third := innerW / 3
	rightW := innerW - third - third

	leftRaw := fmt.Sprintf("Call:%-7s Op:%-7s Log:%-8s", s.Callsign, s.Operator, m.App.LogbookName)
	rightRaw := fmt.Sprintf("Rig:%-7s LT:%-6s UTC:%-9s", rigDisplay, now.Format("15:04"), utc.Format("15:04:05"))
	leftRaw = truncate(leftRaw, third)
	rightRaw = truncate(rightRaw, rightW)
	for lipgloss.Width(leftRaw) < third { leftRaw += " " }
	for lipgloss.Width(rightRaw) < rightW { rightRaw += " " }

	line := leftRaw + padCenter(versionText, third) + rightRaw
	return lipgloss.NewStyle().Width(width).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("229")).Render("  " + line + "  ")
}

func (m *Model) viewTabBar(width int) string {
	active := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("229")).
		Bold(true).
		Padding(0, 2)
	inactive := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 2)
	disabled := lipgloss.NewStyle().
		Foreground(lipgloss.Color("238")).
		Padding(0, 2)
	qsoTab := "F1 QSO Form"
	partnerTab := "F2 Partner Details"
	logsTab := "F9 Logs"
	logsStyle := inactive
	if m.showLogView { logsStyle = active }
	if m.showPartner && m.partnerData != nil {
		qsoTab, partnerTab = inactive.Render(qsoTab), active.Render(partnerTab)
	} else if m.partnerData != nil {
		qsoTab, partnerTab = active.Render(qsoTab), inactive.Render(partnerTab)
	} else if !m.showLogView {
		partnerTab = disabled.Render(partnerTab)
		qsoTab = active.Render(qsoTab)
	} else {
		qsoTab = inactive.Render(qsoTab)
		partnerTab = disabled.Render(partnerTab)
	}
	bar := lipgloss.NewStyle().Width(width).Background(lipgloss.Color("236")).Render(" " + qsoTab + partnerTab + logsStyle.Render(logsTab))
	return bar
}

func (m *Model) viewPartner() string {
	d := m.partnerData
	availW := m.width - 6
	if availW < 50 {
		availW = 50
	}
	infoW := 34
	asciiW := availW - infoW - 2
	if asciiW < 20 {
		asciiW = 0
		infoW = availW
	}
	availH := m.height - 9
	if availH < 6 {
		availH = 6
	}
	if m.App.Config.RenderImages && asciiW > 0 && d.ImageURL != "" && availH >= 8 {
		if m.partnerASCII == "" || m.asciiW != asciiW || m.asciiH != availH {
			ascii, err := downloadAndRenderASCII(d.ImageURL, asciiW, availH)
			if err == nil && ascii != "" {
				m.partnerASCII = ascii
				m.asciiW = asciiW
				m.asciiH = availH
			} else {
				m.partnerASCII = ""
				m.asciiW = 0
				m.asciiH = 0
			}
		}
	} else {
		m.partnerASCII = ""
	}
	return m.viewPartnerData(infoW)
}

func (m *Model) viewPartnerData(infoW int) string {
	d := m.partnerData
	var b strings.Builder
	b.WriteString(titleStyle.Render("Partner Details — " + d.Callsign))
	b.WriteString("\n\n")
	info := m.renderPartnerInfo(d, infoW)
	if m.partnerASCII == "" {
		b.WriteString(info)
	} else {
		leftCol := lipgloss.NewStyle().Width(infoW).Render(info)
		rightCol := lipgloss.NewStyle().Render(m.partnerASCII)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, lipgloss.NewStyle().Width(2).Render(""), rightCol))
	}
	distanceLine := m.partnerDistanceLine(m.width - 2)
	if distanceLine != "" {
		b.WriteString("\n\n")
		b.WriteString(lipgloss.NewStyle().Width(m.width - 2).Align(lipgloss.Center).Render(distanceLine))
		b.WriteString("\n")
	}
	mapStr := ""
	if m.App.Config.RenderImages {
		mapStr = m.renderWorldMap()
	}
	if mapStr != "" {
		b.WriteString("\n")
		b.WriteString(mapStr)
	}
	return b.String()
}

func (m *Model) renderWorldMap() string {
	ownGrid := m.App.Logbook.Station.Grid
	if ownGrid == "" || m.partnerData == nil || m.partnerData.Grid == "" {
		return ""
	}
	ownLat, ownLon := gridToLatLon(ownGrid)
	if ownLat == 0 && ownLon == 0 {
		return ""
	}
	partnerLat, partnerLon := gridToLatLon(m.partnerData.Grid)
	if partnerLat == 0 && partnerLon == 0 {
		return ""
	}
	mapW := m.width - 4
	if mapW < 40 {
		mapW = 40
	}
	infoLines := m.partnerInfoLineCount()
	colH := infoLines
	if m.partnerASCII != "" {
		asciiH := strings.Count(m.partnerASCII, "\n")
		if asciiH > colH {
			colH = asciiH
		}
	}
	used := colH + 12
	mapH := m.height - used
	if mapH < 6 {
		mapH = 6
	}
	return renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, mapH)
}

func (m *Model) partnerInfoLineCount() int {
	if m.partnerData == nil {
		return 0
	}
	d := m.partnerData
	count := 0
	for _, v := range []string{d.Name, d.Grid, d.QTH, d.Country, d.State, d.Zip, d.County, d.Class, d.Email, d.URL, d.DXCC, d.CQZone, d.ITUZone} {
		if v != "" {
			count++
		}
	}
	if d.Lat != "" || d.Lon != "" {
		count++
	}
	return count
}

func (m *Model) partnerDistanceLine(width int) string {
	if m.partnerData == nil {
		return ""
	}
	ownGrid := m.App.Logbook.Station.Grid
	partnerGrid := m.partnerData.Grid
	if ownGrid == "" || partnerGrid == "" {
		return ""
	}
	dist := gridDistance(ownGrid, partnerGrid)
	bear := gridBearing(ownGrid, partnerGrid)
	if dist == "" || bear == "" {
		return ""
	}
	return fmt.Sprintf("%s · %s  from %s to %s", dist, bear, ownGrid, partnerGrid)
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
	add("County", d.County)
	add("Class", d.Class)
	add("Email", d.Email)
	add("URL", d.URL)
	if d.Lat != "" || d.Lon != "" {
		coord := strings.TrimSpace(d.Lat + " " + d.Lon)
		add("Coordinates", coord)
	}
	add("DXCC", d.DXCC)
	add("CQ Zone", d.CQZone)
	add("ITU Zone", d.ITUZone)

	if len(rows) == 0 {
		return ""
	}
	labelW := 12
	valW := maxW - labelW
	if valW < 8 {
		valW = 8
	}
	labelSty := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	blankLabel := strings.Repeat(" ", labelW)
	var b strings.Builder
	for _, r := range rows {
		label := labelSty.Render(fmt.Sprintf("%-*s", labelW, r.label+":"))
		lines := wrapString(r.value, valW)
		for i, line := range lines {
			if i == 0 {
				b.WriteString(label)
			} else {
				b.WriteString(blankLabel)
			}
			b.WriteString(inputStyle.Render(line))
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func wrapString(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var lines []string
	runes := []rune(s)
	for len(runes) > width {
		lines = append(lines, string(runes[:width]))
		runes = runes[width:]
	}
	if len(runes) > 0 {
		lines = append(lines, string(runes))
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	return lines
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
	case m.showChooser:
		text = m.chooser.FooterText()
	case m.showRigEdit:
		text = m.rigChooser.FooterText()
	case m.showLogView:
		text = m.logViewer.FooterText()
	case m.showPartner && m.partnerData != nil:
			text = "F8 Config  F10 Quit"
	default:
		if width < 70 {
			text = "Enter=Save | Del Clear | F8 Config | F10 Quit"
		} else {
			text = "Enter/Ctrl+S Save  Del Clear  F8 Config  F10 Quit"
		}
	}
	return lipgloss.NewStyle().Width(width).
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("241")).
		Padding(0, 1).
		Align(lipgloss.Center).
		Render(text)
}

func padRight(s string, w int) string { s = truncate(s, w); for lipgloss.Width(s) < w { s += " " }; return s }
func padCenter(s string, w int) string { s = truncate(s, w); pad := w - lipgloss.Width(s); l, r := pad/2, pad-pad/2; for i := 0; i < l; i++ { s = " " + s }; for i := 0; i < r; i++ { s += " " }; return s }
func padLeft(s string, w int) string { s = truncate(s, w); for lipgloss.Width(s) < w { s = " " + s }; return s }
func truncate(s string, max int) string { if max < 3 { return s }; if lipgloss.Width(s) <= max { return s }; return s[:max-1] + "…" }

func (m *Model) viewForm(width int) string {
	var rows []string; labelW := 11
	for i := field(0); i < fieldCount; i++ {
		label, value := fmt.Sprintf("%-*s", labelW, fieldNames[i]), m.fields[i].View()
		if int(i) == int(m.focus) { value = cursorStyle.Render(value) }
		rows = append(rows, label+" "+value)
	}
	sepW := width - 2; if sepW > 100 { sepW = 100 }
	sep := strings.Repeat("─", sepW)
	return sep + "\n" + strings.Join(rows, "\n") + "\n" + sep
}

func (m *Model) availableQSORows() int {
	if m.height <= 0 { return 5 }
	avail := m.height - 28
	if avail < 1 { avail = 1 }
	return avail
}

func (m *Model) viewQSOS(maxRows int) string {
	var rows []string
	row := fmt.Sprintf("%-5s %-8s %-6s %-7s %-5s %-6s %-4s %-4s %-7s %s", "ID", "Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "Country", "Comment")
	rows = append(rows, headerStyle.Render(row))
	w := m.width - 4; if w < 1 { w = 60 }; if w > 100 { w = 100 }
	rows = append(rows, "  "+strings.Repeat("─", w))
	if len(m.qsos) == 0 {
		for i := 0; i < maxRows; i++ {
			rows = append(rows, fmt.Sprintf(" ---   ----     ----   ---    ---   ---   ---  ---  -------  -------"))
		}
	} else {
		limit := maxRows
		if limit > len(m.qsos) {
			limit = len(m.qsos)
		}
		for i := 0; i < limit; i++ {
			q := m.qsos[i]
			band := q.Band; if band == "" { band = fmt.Sprintf("%.3f", q.Freq) }
			r := fmt.Sprintf("%-5d %-8s %-6s %-7s %-5s %-6s %-4s %-4s %-7s %s", q.ID, q.QSODate, q.TimeOn, q.Call, band, q.Mode, q.RSTSent, q.RSTRcvd, q.Country, q.Comment)
			if i%2 == 0 { r = inputStyle.Render(r) }
			rows = append(rows, r)
		}
		for i := limit; i < maxRows; i++ {
			rows = append(rows, fmt.Sprintf(" ---   ----     ----   ---    ---   ---   ---  ---  -------  -------"))
		}
	}
	return strings.Join(rows, "\n")
}

func (m *Model) nextField() {
	wasCall := m.focus == fieldCall
	m.fields[m.focus].Blur(); m.focus = (m.focus + 1) % fieldCount
	m.fields[m.focus].Focus()
	if wasCall { m.qrzNeed = true; m.qrzCall = strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value())) }
}
func (m *Model) prevField() {
	wasCall := m.focus == fieldCall
	m.fields[m.focus].Blur()
	if m.focus == 0 { m.focus = fieldCount - 1 } else { m.focus-- }
	m.fields[m.focus].Focus()
	if wasCall { m.qrzNeed = true; m.qrzCall = strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value())) }
}
func (m *Model) updateFocused(msg tea.KeyMsg) {
	prevCall := strings.TrimSpace(m.fields[fieldCall].Value())
	prevVal := m.fields[m.focus].Value()
	m.fields[m.focus], _ = m.fields[m.focus].Update(msg)
	if (m.focus == fieldDate || m.focus == fieldTime) && m.fields[m.focus].Value() != prevVal {
		m.dateTimeAuto = false
	}
	if m.focus == fieldCall || m.focus == fieldGrid { m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value())) }
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
	for i := field(0); i < fieldCount; i++ {
		m.fields[i].SetValue("")
		m.fields[i].Blur()
	}
	now := time.Now().UTC()
	m.fields[fieldDate].SetValue(now.Format("20060102"))
	m.fields[fieldTime].SetValue(now.Format("150405"))
	m.dateTimeAuto = true
	m.focus = fieldCall; m.fields[m.focus].Focus()
	m.partnerData = nil
	m.partnerASCII = ""
	m.asciiW = 0
	m.asciiH = 0
	m.showPartner = false
}
func (m *Model) saveQSO() tea.Cmd {
	qs := qso.NewQSO()
	var freq float64
	fmt.Sscanf(m.fields[fieldFreq].Value(), "%f", &freq)
	qs.Call, qs.Band, qs.Freq = strings.ToUpper(m.fields[fieldCall].Value()), strings.ToUpper(m.fields[fieldBand].Value()), freq
	qs.Mode, qs.RSTSent, qs.RSTRcvd = strings.ToUpper(m.fields[fieldMode].Value()), m.fields[fieldRSTSent].Value(), m.fields[fieldRSTRcvd].Value()
	qs.QSODate = strings.TrimSpace(m.fields[fieldDate].Value())
	if qs.QSODate == "" {
		qs.QSODate = time.Now().UTC().Format("20060102")
	}
	qs.TimeOn = strings.TrimSpace(m.fields[fieldTime].Value())
	if qs.TimeOn == "" {
		qs.TimeOn = time.Now().UTC().Format("150405")
	}
	qs.GridSquare, qs.Comment, qs.Name, qs.QTH, qs.Country = m.fields[fieldGrid].Value(), m.fields[fieldComment].Value(), m.fields[fieldName].Value(), m.fields[fieldCity].Value(), m.fields[fieldCountry].Value()
	station := qso.StationInfo{StationCallsign: m.App.Logbook.Station.Callsign, Operator: m.App.Logbook.Station.Operator, MyGridSquare: m.App.Logbook.Station.Grid, MyRig: m.App.Logbook.Station.Rig, MyAntenna: m.App.Logbook.Station.Antenna}
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
