package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/wavelog"
)

type IntegrationMenu struct {
	// WSJT-X
	wsjtxEnabled bool
	host         textinput.Model
	port         textinput.Model

	// Wavelog
	wlEnabled      bool
	wlURL          textinput.Model
	wlAPIKey       textinput.Model
	wlStations     []wavelog.StationProfile
	wlStation      int // index into wlStations, -1 if none
	wlUpdating     bool
	wlTesting      bool
	wlStatus       string
	savedStationID string // previously configured station ID from config

	focus  int
	done   bool
	saved  bool
	goBack bool
	width  int
	height int
}

func NewIntegrationMenu(cfg *config.Config) *IntegrationMenu {
	host := textinput.New()
	host.CharLimit = 40
	host.Placeholder = "127.0.0.1"
	host.SetValue("127.0.0.1")
	if cfg.WSJTX.Enabled && cfg.WSJTX.UDPHost != "" {
		host.SetValue(cfg.WSJTX.UDPHost)
	}

	port := textinput.New()
	port.CharLimit = 6
	port.Placeholder = "2233"
	port.SetValue("2233")
	if cfg.WSJTX.UDPPort > 0 {
		port.SetValue(strconv.Itoa(cfg.WSJTX.UDPPort))
	}
	host.Focus()

	wu := textinput.New()
	wu.CharLimit = 80
	wu.Placeholder = "https://log.example.com"
	wu.SetValue(cfg.Wavelog.URL)

	wk := textinput.New()
	wk.CharLimit = 64
	wk.Placeholder = "Wavelog API key"
	wk.SetValue(cfg.Wavelog.APIKey)

	return &IntegrationMenu{
		wsjtxEnabled:   cfg.WSJTX.Enabled,
		host:           host,
		port:           port,
		wlEnabled:      cfg.Wavelog.Enabled,
		wlURL:          wu,
		wlAPIKey:       wk,
		wlStations:     nil,
		wlStation:      -1,
		savedStationID: cfg.Wavelog.StationProfileID,
		focus:          1,
	}
}

func (im *IntegrationMenu) Init() tea.Cmd { return nil }

type wlUpdateMsg struct {
	stations []wavelog.StationProfile
	err      error
}
type wlTestMsg struct {
	err error
}

func (im *IntegrationMenu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		im.width, im.height = msg.Width, msg.Height

	case wlUpdateMsg:
		im.wlUpdating = false
		if msg.err != nil {
			im.wlStatus = msg.err.Error()
			im.wlStations = nil
			im.wlStation = -1
			applog.Error("Wavelog update failed", "error", msg.err.Error())
		} else {
			im.wlStations = msg.stations
			im.wlStation = 0
			// Try to restore previously saved station
			if im.savedStationID != "" {
				for i, s := range im.wlStations {
					if s.ID == im.savedStationID {
						im.wlStation = i
						break
					}
				}
			}
			im.wlStatus = fmt.Sprintf("OK — %d stations loaded", len(msg.stations))
			applog.InfoDetail("Wavelog stations loaded", fmt.Sprintf("count=%d", len(msg.stations)))
		}

	case wlTestMsg:
		im.wlTesting = false
		if msg.err != nil {
			im.wlStatus = msg.err.Error()
			applog.Error("Wavelog test failed", "error", msg.err.Error())
		} else {
			im.wlStatus = "OK — Wavelog reachable"
			applog.Info("Wavelog test OK")
		}

	case tea.KeyMsg:
		k := msg.String()
		if im.wlUpdating || im.wlTesting {
			return im, nil
		}
		switch k {
		case "esc":
			im.done = true
			im.goBack = true
			return im, nil
		case "ctrl+s", "\x13":
			im.done = true
			im.saved = true
			return im, nil
		case " ", "enter":
			switch im.focus {
			case 0: // WSJTX checkbox
				im.wsjtxEnabled = !im.wsjtxEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			case 3: // Wavelog checkbox
				im.wlEnabled = !im.wlEnabled
				if !im.isPositionVisible(im.focus) {
					im.fixFocus()
				}
				return im, nil
			case 6: // Update button
				im.wlUpdating = true
				im.wlStatus = "Fetching stations…"
				u := strings.TrimRight(strings.TrimSpace(im.wlURL.Value()), "/")
				k := strings.TrimSpace(im.wlAPIKey.Value())
				return im, func() tea.Msg {
					stations, err := wavelog.FetchStations(u, k)
					return wlUpdateMsg{stations: stations, err: err}
				}
			case 7: // Station selector — cycle through
				if len(im.wlStations) > 0 {
					im.wlStation = (im.wlStation + 1) % len(im.wlStations)
				}
				return im, nil
			case 8: // Test button
				im.wlTesting = true
				im.wlStatus = "Testing…"
				u := strings.TrimRight(strings.TrimSpace(im.wlURL.Value()), "/")
				k := strings.TrimSpace(im.wlAPIKey.Value())
				return im, func() tea.Msg {
					if err := wavelog.TestConnection(u, k); err != nil {
						return wlTestMsg{err: err}
					}
					if im.wlStation >= 0 && im.wlStation < len(im.wlStations) {
						sid := im.wlStations[im.wlStation].ID
						if err := wavelog.TestStation(u, k, sid); err != nil {
							return wlTestMsg{err: err}
						}
					}
					return wlTestMsg{}
				}
			default:
				im.next()
			}
		case "tab", "down":
			im.next()
		case "shift+tab", "up":
			im.prev()
		default:
			switch im.focus {
			case 1:
				im.host, _ = im.host.Update(msg)
			case 2:
				im.port, _ = im.port.Update(msg)
			case 4:
				im.wlURL, _ = im.wlURL.Update(msg)
			case 5:
				im.wlAPIKey, _ = im.wlAPIKey.Update(msg)
			}
		}
	}
	return im, nil
}

func (im *IntegrationMenu) next() {
	for {
		im.focus = (im.focus + 1) % 9
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) prev() {
	for {
		if im.focus == 0 {
			im.focus = 8
		} else {
			im.focus--
		}
		if im.isPositionVisible(im.focus) {
			break
		}
	}
	im.blurAll()
	im.focusField()
}

func (im *IntegrationMenu) isPositionVisible(pos int) bool {
	switch pos {
	case 0: // WSJTX checkbox — always visible
		return true
	case 1, 2: // WSJTX host/port — only when enabled
		return im.wsjtxEnabled
	case 3: // Wavelog checkbox — always visible
		return true
	case 4, 5, 6, 8: // Wavelog URL, Key, Update, Test — only when enabled
		return im.wlEnabled
	case 7: // Station selector — only when enabled and stations loaded
		return im.wlEnabled && len(im.wlStations) > 0
	}
	return true
}

// fixFocus moves focus to the next visible position when current becomes hidden.
func (im *IntegrationMenu) fixFocus() {
	if im.isPositionVisible(im.focus) {
		return
	}
	im.next()
}

func (im *IntegrationMenu) blurAll() {
	im.host.Blur()
	im.port.Blur()
	im.wlURL.Blur()
	im.wlAPIKey.Blur()
}
func (im *IntegrationMenu) focusField() {
	switch im.focus {
	case 1:
		im.host.Focus()
	case 2:
		im.port.Focus()
	case 4:
		im.wlURL.Focus()
	case 5:
		im.wlAPIKey.Focus()
	}
}

func (im *IntegrationMenu) FooterText() string {
	return "Ctrl+S to save  Space/Enter to toggle/act  Tab/↓/↑ to navigate  Esc to go back"
}

func (im *IntegrationMenu) View() string {
	if im.done {
		return ""
	}
	bodyW := im.width - 2
	if bodyW < 30 {
		bodyW = 30
	}

	var b strings.Builder
	title := "── Configuration — Integration "
	b.WriteString(section(title, bodyW))
	b.WriteString("\n\n")

	// ── WSJT-X section ──
	checkbox := "[ ]"
	if im.wsjtxEnabled {
		checkbox = "[x]"
	}
	if im.focus == 0 {
		checkbox = cursorStyle.Render(checkbox)
	}
	b.WriteString(formLabelStyle.Render("WSJT-X:"))
	b.WriteString(" ")
	b.WriteString(checkbox)

	if im.wsjtxEnabled {
		b.WriteString("\n\n")
		b.WriteString(im.renderField(1, "UDP Host:", im.host.View()))
		b.WriteString("\n\n")
		b.WriteString(im.renderField(2, "UDP Port:", im.port.View()))
	}

	b.WriteString(fmt.Sprintf("\n\n  %s", SubtleStyle.Render("default: 127.0.0.1:2233")))
	b.WriteString("\n\n")

	// ── Wavelog section ──
	wlCheckbox := "[ ]"
	if im.wlEnabled {
		wlCheckbox = "[x]"
	}
	if im.focus == 3 {
		wlCheckbox = cursorStyle.Render(wlCheckbox)
	}
	b.WriteString(formLabelStyle.Render("Wavelog:"))
	b.WriteString(" ")
	b.WriteString(wlCheckbox)

	if im.wlEnabled {
		b.WriteString("\n\n")
		b.WriteString(im.renderField(4, "API URL:", im.wlURL.View()))
		b.WriteString("\n\n")
		b.WriteString(im.renderField(5, "API Key:", im.wlAPIKey.View()))

		// Update button
		b.WriteString("\n\n")
		updLabel := "[ Update ]"
		if im.focus == 6 {
			updLabel = cursorStyle.Render("[ Update ]")
		} else {
			updLabel = SubtleStyle.Render("[ Update ]")
		}
		b.WriteString("  ")
		b.WriteString(updLabel)
		b.WriteString("  ")
		b.WriteString(SubtleStyle.Render("fetch stations from Wavelog"))

		// Station selector
		if len(im.wlStations) > 0 {
			b.WriteString("\n\n")
			stationLabel := "Station:"
			if im.focus == 7 {
				stationLabel = cursorStyle.Render("Station:")
				b.WriteString("> ")
			} else {
				b.WriteString("  ")
			}
			s := im.wlStations[im.wlStation]
			b.WriteString(formLabelStyle.Render(stationLabel))
			b.WriteString(" ")
			b.WriteString(inputStyle.Render(fmt.Sprintf("%s — %s (%s)", s.ID, s.Callsign, s.Name)))
			if im.focus == 7 {
				b.WriteString(SubtleStyle.Render(" ← Enter to cycle"))
			}
		} else if im.savedStationID != "" {
			b.WriteString("\n\n")
			b.WriteString("  ")
			b.WriteString(SubtleStyle.Render("Station: " + im.savedStationID + " (press Update to refresh)"))
		}

		// Test button
		b.WriteString("\n\n")
		testLabel := "[ Test ]"
		if im.focus == 8 {
			testLabel = cursorStyle.Render("[ Test ]")
		} else {
			testLabel = SubtleStyle.Render("[ Test ]")
		}
		b.WriteString("  ")
		b.WriteString(testLabel)
		b.WriteString("  ")
		b.WriteString(SubtleStyle.Render("verify connection and station"))
	}

	// Status line
	if im.wlStatus != "" {
		b.WriteString("\n\n  ")
		if strings.HasPrefix(im.wlStatus, "OK") {
			b.WriteString(SuccessStyle.Render(im.wlStatus))
		} else if im.wlUpdating || im.wlTesting {
			b.WriteString(SubtleStyle.Render(im.wlStatus))
		} else {
			b.WriteString(ErrorStyle.Render(im.wlStatus))
		}
	}

	return b.String()
}

func (im *IntegrationMenu) renderField(focusIdx int, label, value string) string {
	var line string
	if im.focus == focusIdx {
		line = cursorStyle.Render("> ") + cursorStyle.Render(label) + " " + inputStyle.Render(value)
	} else {
		line = "  " + formLabelStyle.Render(label) + " " + inputStyle.Render(value)
	}
	return line
}

// Values returns all integration config values.
func (im *IntegrationMenu) Values() (wsjtxEnabled bool, wsjtxHost string, wsjtxPort int, wlEnabled bool, wlURL string, wlAPIKey string, wlStationID string, wlStationCall string, wlStationName string) {
	p := 2233
	if v, err := strconv.Atoi(strings.TrimSpace(im.port.Value())); err == nil && v > 0 {
		p = v
	}
	sid := ""
	scall := ""
	sname := ""
	if im.wlStation >= 0 && im.wlStation < len(im.wlStations) {
		s := im.wlStations[im.wlStation]
		sid = s.ID
		scall = s.Callsign
		sname = fmt.Sprintf("%s / %s", s.Gridsquare, s.Callsign)
	} else if im.savedStationID != "" {
		sid = im.savedStationID
	}
	return im.wsjtxEnabled, strings.TrimSpace(im.host.Value()), p,
		im.wlEnabled, strings.TrimSpace(im.wlURL.Value()), strings.TrimSpace(im.wlAPIKey.Value()), sid, scall, sname
}
