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
	"github.com/szporwolik/cqops/internal/qrz"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/rig/flrig"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/version"
)

type field int

const (
	fieldCall field = iota
	fieldBand
	fieldFreq
	fieldMode
	fieldRSTSent
	fieldRSTRcvd
	fieldGrid
	fieldComment
	fieldName
	fieldQTH
	fieldCount
)

var fieldNames = []string{
	"Call:", "Band:", "Freq:", "Mode:", "RST sent:", "RST rcvd:",
	"Grid:", "Comment:", "Name:", "QTH:",
}

type Model struct {
	App          *app.App
	fields       [fieldCount]textinput.Model
	focus        field
	qsos         []qso.QSO
	statusMsg    string
	statusType   string
	err          error
	width        int
	height       int
	quitting     bool
	rigConnected bool
	rigFreq      float64
	rigMode      string
	rigPower     float64
	rigBlink     bool
	showChooser  bool
	chooser      *LogbookChooser
	showRigEdit  bool
	rigChooser   *RigChooser
	showConfig   bool
	configMenu   *ConfigMenu
	flrigClient  *flrig.Flrig
	showAdvanced bool
}

type tickMsg time.Time
type qrzResultMsg struct{ Call string; Data *qrz.CallData }

func New(a *app.App, initialQSOS []qso.QSO) *Model {
	m := &Model{App: a, qsos: initialQSOS}
	for i := field(0); i < fieldCount; i++ {
		ti := textinput.New()
		ti.CharLimit = 40
		switch i {
		case fieldCall: ti.Focus()
		case fieldFreq: ti.CharLimit = 16
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showChooser { _, _ = m.chooser.Update(msg); if m.chooser.done { m.showChooser = false; m.qsos = nil }; return m, nil }
	if m.showRigEdit { _, _ = m.rigChooser.Update(msg); if m.rigChooser.done { m.showRigEdit = false; m.refreshFlrigClient() }; return m, nil }
	if m.showConfig {
		_, _ = m.configMenu.Update(msg)
		if m.configMenu.done { m.showConfig = false; m.App.Config.QRZAPIKey = m.configMenu.apiKey.Value(); config.Save(m.App.ConfigPath, m.App.Config) }
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg: m.width, m.height = msg.Width, msg.Height
	case tickMsg: m.pollFlrig(); return m, tickCmd()
	case qrzResultMsg: m.fillQRZData(msg); return m, nil
	case tea.KeyMsg:
		switch {
		case msg.String() == "shift+tab": wasCall := m.focus == fieldCall; m.prevField(); if wasCall { return m, qrzLookup(m) }
		case msg.Type == tea.KeyUp || msg.String() == "up": wasCall := m.focus == fieldCall; m.prevField(); if wasCall { return m, qrzLookup(m) }
		case msg.Type == tea.KeyDown || msg.String() == "down": wasCall := m.focus == fieldCall; m.nextField(); if wasCall { return m, qrzLookup(m) }
		case msg.String() == "ctrl+q": m.quitting = true; return m, tea.Quit
		case msg.String() == "ctrl+s": return m, m.saveQSO()
		case msg.String() == "ctrl+u": m.clearForm()
		case msg.String() == "ctrl+a": m.showAdvanced = !m.showAdvanced; if !m.showAdvanced && m.focus >= fieldGrid { m.focus = fieldCall; m.fields[m.focus].Focus() }
		case msg.String() == "ctrl+c": m.configMenu = NewConfigMenu(m.App.Config); m.showConfig = true
		case msg.String() == "ctrl+l": m.chooser = NewLogbookChooser(m.App); m.showChooser = true
		case msg.String() == "ctrl+g": m.rigChooser = NewRigChooser(m.App); m.showRigEdit = true
		case msg.String() == "tab": wasCall := m.focus == fieldCall; m.nextField(); if wasCall { return m, qrzLookup(m) }
		case msg.String() == "enter": return m, m.saveQSO()
		default: m.updateFocused(msg)
		}
	}
	return m, nil
}

func qrzLookup(m *Model) tea.Cmd {
	call := strings.ToUpper(strings.TrimSpace(m.fields[fieldCall].Value()))
	key := m.App.Config.QRZAPIKey
	if key == "" || call == "" { return nil }
	return func() tea.Msg { data, _ := qrz.Lookup(key, call); return qrzResultMsg{Call: call, Data: data} }
}

func (m *Model) fillQRZData(msg qrzResultMsg) {
	if msg.Call == "" { return }
	if m.App.Config.QRZAPIKey == "" { m.setStatus("Set QRZ API key in Ctrl+C Settings", "warning"); return }
	d := msg.Data
	if d == nil || d.Callsign == "" { return }
	if d.Name != "" { m.fields[fieldName].SetValue(d.Name) }
	if d.Grid != "" && m.fields[fieldGrid].Value() == "" { m.fields[fieldGrid].SetValue(d.Grid) }
	if d.QTH != "" { m.fields[fieldQTH].SetValue(d.QTH) }
	if (d.Country != "" || d.State != "") && m.fields[fieldComment].Value() == "" {
		m.fields[fieldComment].SetValue(strings.TrimSpace(d.Country + " " + d.State))
	}
	m.setStatus("QRZ: "+d.Callsign+" "+d.Name, "success")
}

func (m *Model) View() string {
	if m.quitting { return "\n" }
	if m.err != nil { return errorStyle.Render(fmt.Sprintf("Error: %v\nPress any key to exit.", m.err)) }
	if m.showChooser { return m.chooser.View() }
	if m.showRigEdit { return m.rigChooser.View() }
	if m.showConfig { return m.configMenu.View() }
	w := m.width; if w < 40 { w = 80 }
	topBar := m.viewTopBar(w)
	form := m.viewForm(w)
	status := m.viewStatus()
	qsoList := m.viewQSOS()
	helpText := "Enter/Ctrl+S Save  Ctrl+U Clear  Ctrl+A Advanced  Ctrl+L Logbooks  Ctrl+G Rig  Ctrl+C Settings  Ctrl+Q Quit"
	if w < 70 { helpText = "Enter=Save | Ctrl+A Adv | Ctrl+L Logs | Ctrl+G Rig | Ctrl+C Set | Ctrl+Q Quit" }
	help := helpStyle.Render(helpText)
	content := lipgloss.JoinVertical(lipgloss.Left, form, status, qsoList)
	body := lipgloss.NewStyle().Width(w).Padding(0, 1).Render(content)
	footer := lipgloss.NewStyle().Width(w).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("241")).Padding(0, 1).Align(lipgloss.Center).Render(help)
	all := lipgloss.JoinVertical(lipgloss.Left, topBar, body, footer)
	if lines := strings.Count(all, "\n") + 1; m.height > 0 && lines < m.height { all += strings.Repeat("\n", m.height-lines) }
	return all
}

func (m *Model) viewTopBar(width int) string {
	s := m.App.Logbook.Station
	leftText := s.Callsign
	if s.Operator != "" && s.Operator != s.Callsign { leftText += " — " + s.Operator }
	if m.App.LogbookName != "" { leftText += " — [" + m.App.LogbookName + "]" }
	if leftText == "" { leftText = "CQOPS" }
	now := time.Now(); utc := now.UTC()
	rightText := fmt.Sprintf("%s local  %s utc", now.Format("15:04:05"), utc.Format("15:04"))
	rigName := s.RigName; if rigName == "" { rigName = "default" }
	if rp, ok := m.App.Config.Rigs[rigName]; ok {
		rigDisplay := rp.Model
		if rp.FlrigEnabled {
			if m.rigConnected { if m.rigBlink { rigDisplay += " *" } else { rigDisplay += "  " } } else { rigDisplay += " !" }
		}
		if rigDisplay != "" { rightText = rigDisplay + "  " + rightText }
	} else {
		for name := range m.App.Config.Rigs { rigName = name; break }
		if rigName != "" {
			lb := m.App.Config.Logbooks[m.App.LogbookName]; lb.Station.RigName = rigName
			lb.Station.Rig, lb.Station.Antenna, lb.Station.Power = m.App.Config.Rigs[rigName].Model, m.App.Config.Rigs[rigName].Antenna, m.App.Config.Rigs[rigName].Power
			m.App.Config.Logbooks[m.App.LogbookName] = lb; m.App.Logbook = &lb
		}
	}
	versionText := "CQOPS v" + version.Resolved(); if version.Resolved() == "dev" { versionText = "CQOPS" }
	innerW, third, rightW := width-2, (width-2)/3, (width-2)-(width-2)/3-(width-2)/3
	_ = innerW
	left, center, right := padRight(leftText, third), padCenter(versionText, third), padLeft(rightText, rightW)
	return lipgloss.NewStyle().Width(width).Background(lipgloss.Color("236")).Foreground(lipgloss.Color("229")).Render(" " + left + center + right + " ")
}

func padRight(s string, w int) string { s = truncate(s, w); for lipgloss.Width(s) < w { s += " " }; return s }
func padCenter(s string, w int) string { s = truncate(s, w); pad := w - lipgloss.Width(s); l, r := pad/2, pad-pad/2; for i := 0; i < l; i++ { s = " " + s }; for i := 0; i < r; i++ { s += " " }; return s }
func padLeft(s string, w int) string { s = truncate(s, w); for lipgloss.Width(s) < w { s = " " + s }; return s }
func truncate(s string, max int) string { if max < 3 { return s }; if lipgloss.Width(s) <= max { return s }; return s[:max-1] + "…" }

func (m *Model) viewForm(width int) string {
	var rows []string; labelW := 11
	for i := field(0); i < fieldGrid; i++ {
		label, value := fmt.Sprintf("%-*s", labelW, fieldNames[i]), m.fields[i].View()
		if int(i) == int(m.focus) { value = cursorStyle.Render(value) }
		rows = append(rows, label+" "+value)
	}
	if m.showAdvanced {
		for i := fieldGrid; i < fieldCount; i++ {
			label, value := fmt.Sprintf("%-*s", labelW, fieldNames[i]), m.fields[i].View()
			if int(i) == int(m.focus) { value = cursorStyle.Render(value) }
			rows = append(rows, label+" "+value)
		}
	}
	sep := strings.Repeat("─", width-2)
	return sep + "\n" + strings.Join(rows, "\n") + "\n" + sep
}

func (m *Model) viewStatus() string {
	if m.statusMsg == "" { return "" }
	switch m.statusType { case "error": return errorStyle.Render(m.statusMsg); case "warning": return warningStyle.Render(m.statusMsg); case "success": return successStyle.Render(m.statusMsg); default: return m.statusMsg }
}

func (m *Model) viewQSOS() string {
	if len(m.qsos) == 0 { return headerStyle.Render(fmt.Sprintf("No QSOs in [%s]", m.App.LogbookName)) }
	var rows []string
	row := fmt.Sprintf("%-5s %-8s %-6s %-7s %-5s %-6s %-4s %-4s %s", "ID", "Date", "Time", "Call", "Band", "Mode", "RSTs", "RSTr", "Comment")
	rows = append(rows, headerStyle.Render(row))
	w := m.width - 4; if w < 1 { w = 60 }
	rows = append(rows, "  "+strings.Repeat("─", w))
	for i, q := range m.qsos {
		band := q.Band; if band == "" { band = fmt.Sprintf("%.3f", q.Freq) }
		r := fmt.Sprintf("%-5d %-8s %-6s %-7s %-5s %-6s %-4s %-4s %s", q.ID, q.QSODate, q.TimeOn, q.Call, band, q.Mode, q.RSTSent, q.RSTRcvd, q.Comment)
		if i%2 == 0 { r = inputStyle.Render(r) }
		rows = append(rows, r)
	}
	return strings.Join(rows, "\n")
}

func (m *Model) nextField() {
	m.fields[m.focus].Blur(); m.focus = (m.focus + 1) % fieldCount
	if !m.showAdvanced && m.focus >= fieldGrid { m.focus = fieldCall }
	m.fields[m.focus].Focus()
}
func (m *Model) prevField() {
	m.fields[m.focus].Blur()
	if m.focus == 0 { m.focus = fieldCount - 1 } else { m.focus-- }
	if !m.showAdvanced && m.focus >= fieldGrid { m.focus = fieldComment - 1 }
	m.fields[m.focus].Focus()
}
func (m *Model) updateFocused(msg tea.KeyMsg) {
	m.fields[m.focus], _ = m.fields[m.focus].Update(msg)
	if m.focus == fieldCall || m.focus == fieldGrid { m.fields[m.focus].SetValue(strings.ToUpper(m.fields[m.focus].Value())) }
}
func (m *Model) clearForm() {
	for i := field(0); i < fieldCount; i++ { m.fields[i].SetValue("") }
	m.focus = fieldCall; m.fields[m.focus].Focus(); m.statusMsg = ""
}
func (m *Model) saveQSO() tea.Cmd {
	qs := qso.NewQSO()
	qs.Call, qs.Band, qs.Freq = strings.ToUpper(m.fields[fieldCall].Value()), strings.ToUpper(m.fields[fieldBand].Value()), parseFloat(m.fields[fieldFreq].Value())
	qs.Mode, qs.RSTSent, qs.RSTRcvd = strings.ToUpper(m.fields[fieldMode].Value()), m.fields[fieldRSTSent].Value(), m.fields[fieldRSTRcvd].Value()
	qs.GridSquare, qs.Comment, qs.Name, qs.QTH = m.fields[fieldGrid].Value(), m.fields[fieldComment].Value(), m.fields[fieldName].Value(), m.fields[fieldQTH].Value()
	station := qso.FillSource{StationCallsign: m.App.Logbook.Station.Callsign, Operator: m.App.Logbook.Station.Operator, MyGridSquare: m.App.Logbook.Station.Grid, MyRig: m.App.Logbook.Station.Rig, MyAntenna: m.App.Logbook.Station.Antenna}
	qso.Fill(qs, nil, station)
	if err := qso.ValidateForSave(qs); err != nil { m.setStatus(err.Error(), "error"); return nil }
	if _, err := store.InsertQSO(m.App.DB, qs); err != nil { m.setStatus(fmt.Sprintf("Save failed: %v", err), "error"); return nil }
	m.clearForm(); m.setStatus(fmt.Sprintf("QSO saved: %s", qs.Call), "success")
	return m.refreshQSOS()
}
func (m *Model) refreshQSOS() tea.Cmd {
	qsos, err := store.ListQSOs(m.App.DB, 30)
	if err != nil { m.setStatus(fmt.Sprintf("Refresh failed: %v", err), "error"); return nil }
	m.qsos = qsos; return nil
}
func (m *Model) setStatus(msg, typ string) { m.statusMsg, m.statusType = msg, typ }
func parseFloat(s string) float64 { var f float64; fmt.Sscanf(s, "%f", &f); return f }
