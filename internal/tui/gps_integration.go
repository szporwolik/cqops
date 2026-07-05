package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/gps"
)

// =============================================================================
// GPS integration — serial NMEA receiver, position tracking, status display.
// =============================================================================

// gpsState holds the live GPS integration state.
type gpsState struct {
	client   *gps.Client
	reader   *gps.SerialReader
	online   bool
	hasFix   bool
	lastGrid string
	lastLat  float64
	lastLon  float64
	lastSeen time.Time

	// reconnect tracking
	reconnectUntil  time.Time
	connectFailures int

	// toast state tracking — avoids repeated toasts.
	didToastConnect bool
	didToastFix     bool
	didToastLost    bool

	// original station grid — saved before GPS override is applied,
	// restored when GPS is stopped or GPSGrid is disabled.
	originalStationGrid string
}

// startGPS opens the serial port and starts reading NMEA sentences.
// Returns a tea.Cmd for polling. If the port cannot be opened, it
// schedules a delayed reconnect instead of failing.
func (m *Model) startGPS() tea.Cmd {
	cfg := m.App.Config.Integrations.GPS
	if !cfg.Enabled || cfg.Port == "" {
		return nil
	}
	if cfg.BaudRate == 0 {
		cfg.BaudRate = 115200
	}

	// Clean up any previous client/reader before creating new ones.
	m.stopGPS()

	m.gps.reader = gps.NewSerialReader(gps.SerialConfig{
		Port:     cfg.Port,
		BaudRate: cfg.BaudRate,
		DTR:      cfg.DTR,
		RTS:      cfg.RTS,
	})

	// Synchronously verify the port is reachable before launching the reader.
	if err := m.gps.reader.TryOpen(); err != nil {
		applog.Warn("GPS: cannot open port", "port", cfg.Port, "error", err.Error())
		m.toasts.Error("GPS: cannot open " + cfg.Port)
		m.gps.reader.Close()
		m.gps.reader = nil
		return m.pollGPS()
	}

	m.gps.client = gps.NewClient(m.gps.reader)
	m.gps.client.Start()

	m.gps.online = true
	m.gps.lastSeen = time.Now()
	m.gps.didToastConnect = false
	m.gps.didToastFix = false
	m.gps.didToastLost = false

	applog.Info("GPS: integration started",
		"port", cfg.Port,
		"baud", fmt.Sprintf("%d", cfg.BaudRate),
	)
	m.toasts.Success("GPS: connecting to " + cfg.Port)
	return m.pollGPS()
}

// stopGPS closes the serial port and stops the GPS client.
func (m *Model) stopGPS() {
	m.restoreGPSGridOverride()
	if m.gps.client != nil {
		m.gps.client.Stop()
		m.gps.client = nil
	}
	if m.gps.reader != nil {
		m.gps.reader.Close()
		m.gps.reader = nil
	}
	m.gps.online = false
	m.gps.hasFix = false
	m.gps.lastGrid = ""
	m.gps.didToastConnect = false
	m.gps.didToastFix = false
	m.gps.didToastLost = false
	applog.Info("GPS: integration stopped")
}

// applyGPSGridOverride sets the station grid to the GPS-derived grid
// when GPSGrid is enabled in the logbook config. The original grid is
// saved for later restoration.
func (m *Model) applyGPSGridOverride() {
	if m.App == nil || m.App.Logbook == nil {
		return
	}
	if !m.App.Logbook.Station.GPSGrid || m.gps.lastGrid == "" {
		return
	}
	if m.gps.originalStationGrid != "" {
		return // already applied
	}
	m.gps.originalStationGrid = m.App.Logbook.Station.Grid
	m.App.Logbook.Station.Grid = m.gps.lastGrid
	applog.Info("GPS: grid override applied",
		"original", m.gps.originalStationGrid,
		"gps", m.gps.lastGrid,
	)
}

// restoreGPSGridOverride restores the original station grid when the
// GPS grid was overridden. Safe to call multiple times.
func (m *Model) restoreGPSGridOverride() {
	if m.gps.originalStationGrid == "" || m.App == nil || m.App.Logbook == nil {
		return
	}
	m.App.Logbook.Station.Grid = m.gps.originalStationGrid
	applog.Info("GPS: grid override restored", "grid", m.gps.originalStationGrid)
	m.gps.originalStationGrid = ""
}

// pollGPS returns a tea.Cmd that checks the GPS position and schedules
// the next poll.
func (m *Model) pollGPS() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return gpsTickMsg{}
	})
}

type gpsTickMsg struct{}

// handleGPSTick reads the latest position from the GPS client and
// updates the status bar, toasts, and other state. Handles disconnect
// detection and auto-reconnect.
func (m *Model) handleGPSTick() tea.Cmd {
	// If the GPS is configured but has no active client (e.g. TryOpen
	// failed during start), initiate reconnect without touching the client.
	if m.gps.client == nil {
		if !m.gps.online && m.App.Config.Integrations.GPS.Enabled {
			return m.scheduleOrReconnect()
		}
		return nil
	}

	// Safety: recover from panics in GPS client (e.g. serial port issues).
	defer func() {
		if r := recover(); r != nil {
			applog.Error("GPS: panic in handleGPSTick — resetting", "panic", fmt.Sprintf("%v", r))
			m.stopGPS()
		}
	}()

	pos := m.gps.client.Latest()
	prevOnline := m.gps.online
	prevFix := m.gps.hasFix

	// Check if the reader goroutine is still alive.
	// The reader pointer stays non-nil even after the port dies —
	// use the client's IsRunning to detect a dead loop.
	m.gps.online = m.gps.client.IsRunning()

	// Process position data.
	if m.gps.online && pos.IsValid() {
		m.gps.hasFix = true
		m.gps.lastSeen = time.Now()
		m.gps.lastLat = pos.Lat
		m.gps.lastLon = pos.Lon
		m.gps.connectFailures = 0 // reset on success
		grid := pos.Grid()
		if grid != "" && grid != m.gps.lastGrid {
			m.gps.lastGrid = grid
			applog.Info("GPS: grid updated", "grid", grid,
				"lat", fmt.Sprintf("%.6f", pos.Lat),
				"lon", fmt.Sprintf("%.6f", pos.Lon),
			)
		}
	} else if m.gps.online {
		// Online but no valid fix — normal during acquisition.
		m.gps.hasFix = false
		m.gps.connectFailures = 0
	} else {
		m.gps.hasFix = false
	}

	// Toast on state changes — one-shot per transition.
	if m.gps.online && !m.gps.didToastConnect {
		m.gps.didToastConnect = true
		m.toasts.Success("GPS: connected — " + m.App.Config.Integrations.GPS.Port)
		applog.Info("GPS: connected", "port", m.App.Config.Integrations.GPS.Port)
	}
	if !prevOnline && !m.gps.online {
		// Never managed to connect — just retry silently.
	}
	if prevOnline && !m.gps.online {
		// Connection dropped.
		if !m.gps.didToastLost {
			m.gps.didToastLost = true
			m.toasts.Warn("GPS: disconnected — retrying")
			applog.Warn("GPS: disconnected", "port", m.App.Config.Integrations.GPS.Port)
		}
	}
	if m.gps.hasFix && !m.gps.didToastFix {
		m.gps.didToastFix = true
		m.toasts.Success("GPS: fix acquired — " + m.gps.lastGrid)
		applog.Info("GPS: fix acquired",
			"lat", fmt.Sprintf("%.6f", m.gps.lastLat),
			"lon", fmt.Sprintf("%.6f", m.gps.lastLon),
			"grid", m.gps.lastGrid,
		)
		// Apply GPS grid override when GPSGrid flag is set.
		m.applyGPSGridOverride()
	}
	if prevFix && !m.gps.hasFix {
		m.gps.didToastFix = false // allow re-toast on next fix
		m.toasts.Warn("GPS: fix lost")
		applog.Warn("GPS: fix lost")
	}

	// Reconnect logic — fixed 60s retry interval.
	if !m.gps.online {
		return m.scheduleOrReconnect()
	}

	return m.pollGPS()
}

// scheduleOrReconnect either sets up a reconnect timer or fires a
// reconnect attempt if the timer has expired. Extracted so the nil-
// client path can also trigger reconnects.
func (m *Model) scheduleOrReconnect() tea.Cmd {
	if m.gps.reconnectUntil.IsZero() {
		m.gps.connectFailures++
		delay := gpsReconnectDelay(m.gps.connectFailures)
		m.gps.reconnectUntil = time.Now().Add(delay)
		applog.Info("GPS: reconnect scheduled",
			"delay", delay.String(),
			"failures", fmt.Sprintf("%d", m.gps.connectFailures),
		)
	}
	if time.Now().After(m.gps.reconnectUntil) {
		m.gps.reconnectUntil = time.Time{}
		applog.Info("GPS: attempting reconnect",
			"attempt", fmt.Sprintf("%d", m.gps.connectFailures),
		)
		m.toasts.Info("GPS: reconnecting…")
		return m.startGPS()
	}
	return m.pollGPS()
}

// gpsReconnectDelay returns a fixed 60s retry interval.
// GPS devices are either physically present or not — exponential backoff
// doesn't help, and a one-minute retry catches a replugged device promptly.
func gpsReconnectDelay(failures int) time.Duration {
	return 60 * time.Second
}

// effectiveGrid returns the current effective station grid locator.
// When GPS is enabled, has a fix, and the logbook's GPSGrid flag is set,
// the GPS-derived grid is used. Otherwise the configured station grid
// is returned. This is the single source of truth for the station grid
// used in QSO logging, APRS beacons, dashboard, and distance calculations.
func (m *Model) effectiveGrid() string {
	// GPS override: enabled + has fix + logbook flag set.
	if m.App != nil && m.App.Config != nil &&
		m.App.Config.Integrations.GPS.Enabled && m.gps.hasFix &&
		m.App.Logbook != nil && m.App.Logbook.Station.GPSGrid && m.gps.lastGrid != "" {
		return m.gps.lastGrid
	}
	if m.App == nil || m.App.Logbook == nil {
		return ""
	}
	return strings.TrimSpace(strings.ToUpper(m.App.Logbook.Station.Grid))
}

// isGPSGridActive returns true when the displayed station grid is
// currently derived from GPS (GPS enabled, has fix, GPSGrid flag set).
func (m *Model) isGPSGridActive() bool {
	return m.App != nil && m.App.Config != nil &&
		m.App.Config.Integrations.GPS.Enabled && m.gps.hasFix &&
		m.App.Logbook != nil && m.App.Logbook.Station.GPSGrid && m.gps.lastGrid != ""
}
