// Package gps provides a GPS receiver client that reads NMEA sentences
// from a serial port and exposes the current position, fix status, and
// Maidenhead grid locator.
package gps

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// Fix quality values from GGA sentence.
const (
	FixNone     = 0
	FixGPS      = 1
	FixDGPS     = 2
	FixPPS      = 3
	FixRTK      = 4
	FixFloatRTK = 5
)

// Position holds a decoded GPS position with fix status.
type Position struct {
	Lat        float64   // decimal degrees, positive north
	Lon        float64   // decimal degrees, positive east
	Altitude   float64   // metres above geoid
	Fix        int       // 0 = none, 1 = GPS, 2 = DGPS
	Satellites int       // number of satellites in use
	Time       time.Time // UTC time from GPS
	UpdatedAt  time.Time // local time of last update
}

// IsValid returns true when the position has a usable fix.
func (p Position) IsValid() bool {
	return p.Fix >= FixGPS && p.Lat != 0 && p.Lon != 0
}

// Grid returns the Maidenhead grid locator (10-char, ~1.25″×0.625″ ≈ ~25×19 m accuracy)
// for this position. Returns empty string when the position is invalid.
func (p Position) Grid() string {
	if !p.IsValid() {
		return ""
	}
	return latLonToGrid(p.Lat, p.Lon)
}

// Client reads NMEA sentences from a serial port Reader.
// It is safe for concurrent use: ReadPosition and Latest are
// goroutine-safe.
type Client struct {
	mu         sync.RWMutex
	latest     Position
	reader     NMEAReader
	stopCh     chan struct{}
	doneCh     chan struct{}
	debugCount int // throttles debug logging
}

// NMEAReader abstracts the source of NMEA data (serial port, GPSD, file, etc.).
type NMEAReader interface {
	ReadLine() (string, error)
	Close() error
	TryOpen() error // synchronous pre-flight check
}

// NewClient creates a new GPS client that reads from r.
func NewClient(r NMEAReader) *Client {
	return &Client{
		reader: r,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start begins reading NMEA sentences in a background goroutine.
func (c *Client) Start() {
	go c.loop()
}

func (c *Client) loop() {
	defer close(c.doneCh)
	defer func() {
		if r := recover(); r != nil {
			applog.Error("GPS: panic in read loop — exiting", "panic", fmt.Sprintf("%v", r))
		}
	}()
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		line, err := c.reader.ReadLine()
		if err != nil {
			// Reader closed or error — exit loop gracefully.
			// The TUI integration will detect this and reconnect.
			applog.Debug("GPS: read loop exited", "error", err.Error())
			return
		}
		c.parseNMEA(line)
	}
}

// Stop signals the read loop to stop and closes the reader.
func (c *Client) Stop() {
	select {
	case <-c.stopCh:
		return // already stopped
	default:
		close(c.stopCh)
	}
	c.reader.Close()
	// Wait for the loop to finish.
	select {
	case <-c.doneCh:
	case <-time.After(2 * time.Second):
	}
}

// IsRunning returns true when the background read loop is still active.
func (c *Client) IsRunning() bool {
	select {
	case <-c.doneCh:
		return false
	default:
		return true
	}
}

// Latest returns the most recent valid position, or a zero Position if
// no fix has been acquired since Start.
func (c *Client) Latest() Position {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latest
}

// parseNMEA decodes one NMEA sentence and updates c.latest when a
// position fix is available.
func (c *Client) parseNMEA(line string) {
	line = strings.TrimSpace(line)
	if len(line) < 6 || line[0] != '$' {
		return
	}

	// Strip optional two-char talker prefix ($GPGGA → $GGA, $GNGGA → $GGA).
	// NMEA talker is present when the 4th character ($TT...) is a letter.
	// Bare sentences like $GGA have a comma at position 3.
	rest := line[1:]
	if len(line) > 4 && line[3] >= 'A' && line[3] <= 'Z' {
		rest = rest[2:] // strip talker
	}

	fields := strings.Split(rest, ",")
	if len(fields) < 2 {
		return
	}
	sentenceType := fields[0]

	switch sentenceType {
	case "GGA":
		c.debugCount++
		if c.debugCount%30 == 1 {
			applog.Debug("GPS: NMEA GGA", "raw", line, "count", fmt.Sprintf("%d", c.debugCount))
		}
		c.parseGGA(fields)
	case "RMC":
		c.parseRMC(fields)
	default:
		// Other sentences (GSA, GSV, ZDA, VTG) are silently skipped.
	}
}

func (c *Client) parseGGA(fields []string) {
	if len(fields) < 10 {
		return
	}
	// $GPGGA,hhmmss.ss,llll.ll,a,yyyyy.yy,a,q,ss,h.h,H,H.h,H,...
	// fields[1]=time, [2]=lat, [3]=N/S, [4]=lon, [5]=E/W, [6]=quality,
	// [7]=satellites, [8]=hdop, [9]=altitude, [10]=altUnit
	quality := parseIntField(fields[6])
	if quality < FixGPS {
		return
	}
	lat := parseLatLon(fields[2], fields[3], true)
	lon := parseLatLon(fields[4], fields[5], false)
	if lat == 0 && lon == 0 {
		return
	}
	alt := parseFloatField(fields[9])
	sats := parseIntField(fields[7])

	c.mu.Lock()
	c.latest.Lat = lat
	c.latest.Lon = lon
	c.latest.Altitude = alt
	c.latest.Fix = quality
	c.latest.Satellites = sats
	c.latest.Time = parseTimeField(fields[1])
	c.latest.UpdatedAt = time.Now()
	c.mu.Unlock()
}

func (c *Client) parseRMC(fields []string) {
	if len(fields) < 10 {
		return
	}
	// $GPRMC,hhmmss.ss,A,llll.ll,a,yyyyy.yy,a,spd,cog,date,,,mode
	status := strings.TrimSpace(fields[2])
	if status != "A" {
		// Void fix — keep existing position but mark as no fix.
		// RMC status='V' is common during initial acquisition.
		return
	}
	// RMC has a valid fix — ensure fix quality is at least GPS.
	c.mu.Lock()
	if c.latest.Fix < FixGPS {
		c.latest.Fix = FixGPS
	}
	c.mu.Unlock()
}

// parseIntField parses a string field to int, returning 0 on failure.
func parseIntField(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

// parseFloatField parses a string field to float64, returning 0 on failure.
func parseFloatField(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// parseLatLon converts a DDMM.MMMM hemisphere pair to decimal degrees.
func parseLatLon(val, hemi string, isLat bool) float64 {
	val = strings.TrimSpace(val)
	hemi = strings.TrimSpace(hemi)
	if val == "" || hemi == "" {
		return 0
	}
	deg := 0
	min := 0.0
	if isLat {
		fmt.Sscanf(val, "%2d%f", &deg, &min)
	} else {
		fmt.Sscanf(val, "%3d%f", &deg, &min)
	}
	dec := float64(deg) + min/60.0
	if hemi == "S" || hemi == "W" {
		dec = -dec
	}
	return dec
}

// parseTimeField parses a UTC HHMMSS.SS field into a time.Time.
// The date is taken from the current day — GPS date comes from RMC/ZDA.
func parseTimeField(s string) time.Time {
	s = strings.TrimSpace(s)
	if len(s) < 6 {
		return time.Time{}
	}
	h, m, sec := 0, 0, 0.0
	fmt.Sscanf(s, "%2d%2d%f", &h, &m, &sec)
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(),
		h, m, int(sec), int((sec-float64(int(sec)))*1e9), time.UTC)
}

// latLonToGrid converts decimal lat/lon to a 10-character Maidenhead grid locator.
// Accuracy: ~25 m — appropriate for GPS-derived positions.
func latLonToGrid(lat, lon float64) string {
	// Normalise longitude to 0–360°, then offset to Maidenhead basis (180°).
	lon += 180
	lat += 90

	// Prevent edge cases exactly at poles / anti-meridian.
	if lat <= 0 {
		lat = 0.0001
	}
	if lat >= 180 {
		lat = 179.9999
	}
	if lon <= 0 {
		lon = 0.0001
	}
	if lon >= 360 {
		lon = 359.9999
	}

	// Field: 20° lon × 10° lat → AA–RR
	fieldLon := int(lon) / 20
	fieldLat := int(lat) / 10
	grid := string(rune('A'+fieldLon)) + string(rune('A'+fieldLat))

	// Square: 2° lon × 1° lat → 00–99
	lonRem := lon - float64(fieldLon*20)
	latRem := lat - float64(fieldLat*10)
	squareLon := int(lonRem) / 2
	squareLat := int(latRem) / 1
	grid += string(rune('0'+squareLon)) + string(rune('0'+squareLat))

	// Sub-square: 5′ lon × 2.5′ lat → aa–xx (24×24 divisions of the square).
	// Lon: 2° / 24 = 5′ → multiply by 12.  Lat: 1° / 24 = 2.5′ → multiply by 24.
	lonSub := lonRem - float64(squareLon*2)
	latSub := latRem - float64(squareLat*1)
	subLon := int(lonSub * 12)
	subLat := int(latSub * 24)
	grid += string(rune('a'+subLon)) + string(rune('a'+subLat))

	// Extended square: 0.5′ lon × 0.25′ lat → 00–99 (10×10).
	lonExt := (lonSub - float64(subLon)/12) * 12
	latExt := (latSub - float64(subLat)/24) * 24
	extLon := int(lonExt * 10)
	extLat := int(latExt * 10)
	grid += string(rune('0'+extLon)) + string(rune('0'+extLat))

	// Extended sub-square: 1.25″ lon × 0.625″ lat → aa–xx (24×24).
	lonExt2 := (lonExt - float64(extLon)/10) * 10
	latExt2 := (latExt - float64(extLat)/10) * 10
	ext2Lon := int(lonExt2 * 24)
	ext2Lat := int(latExt2 * 24)
	grid += string(rune('a'+ext2Lon)) + string(rune('a'+ext2Lat))

	return strings.ToUpper(grid)
}
