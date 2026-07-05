package gps

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
)

// GPSDReader connects to a GPSD server and converts TPV position reports
// into NMEA $GPGGA sentences so the existing NMEA parsing pipeline works
// unchanged.
type GPSDReader struct {
	addr    string
	conn    net.Conn
	scanner *bufio.Scanner
	closed  bool
}

// NewGPSDReader creates a reader that connects to the GPSD server at host:port.
// The connection is opened lazily on the first ReadLine call.
func NewGPSDReader(host, port string) *GPSDReader {
	if port == "" {
		port = "2947"
	}
	return &GPSDReader{addr: net.JoinHostPort(host, port)}
}

// TryOpen attempts to connect to the GPSD server synchronously.
// Returns nil on success, or an error describing why the connection failed.
func (r *GPSDReader) TryOpen() error {
	if r.conn != nil {
		return nil
	}
	return r.connect()
}

// ReadLine returns the next NMEA sentence (converted from a GPSD TPV
// report), or an error if the connection is lost.
func (r *GPSDReader) ReadLine() (string, error) {
	if r.scanner == nil {
		if err := r.connect(); err != nil {
			return "", err
		}
	}
	for r.scanner.Scan() {
		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}
		// GPSD sends JSON objects. We only care about TPV (position) reports.
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			continue
		}
		if cls, _ := obj["class"].(string); cls != "TPV" {
			continue
		}
		gga := tpvToGGA(obj)
		if gga == "" {
			continue
		}
		return gga, nil
	}
	err := r.scanner.Err()
	if err != nil {
		applog.Warn("GPS: GPSD read error — reconnecting", "addr", r.addr, "error", err.Error())
	}
	r.close()
	return "", fmt.Errorf("GPSD: connection closed")
}

// Close terminates the GPSD connection.
func (r *GPSDReader) Close() error {
	r.close()
	return nil
}

func (r *GPSDReader) connect() error {
	applog.Debug("GPS: GPSD connecting", "addr", r.addr)
	conn, err := net.DialTimeout("tcp", r.addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("GPSD: cannot connect to %s: %w", r.addr, err)
	}
	r.conn = conn
	// Send WATCH command to enable JSON position reports.
	_, err = fmt.Fprintf(conn, "?WATCH={\"enable\":true,\"json\":true}\n")
	if err != nil {
		conn.Close()
		return fmt.Errorf("GPSD: WATCH command failed: %w", err)
	}
	r.scanner = bufio.NewScanner(conn)
	r.scanner.Split(bufio.ScanLines)
	applog.Info("GPS: GPSD connected", "addr", r.addr)
	return nil
}

func (r *GPSDReader) close() {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
	r.scanner = nil
	r.closed = true
	applog.Debug("GPS: GPSD connection closed", "addr", r.addr)
}

// tpvToGGA converts a GPSD TPV JSON object to an NMEA $GPGGA sentence.
// Returns "" when the TPV lacks a usable position fix.
func tpvToGGA(obj map[string]interface{}) string {
	mode, _ := obj["mode"].(float64)
	if mode < 2 { // 1=no fix, 2=2D, 3=3D
		return ""
	}
	lat, okLat := obj["lat"].(float64)
	lon, okLon := obj["lon"].(float64)
	if !okLat || !okLon || lat == 0 || lon == 0 {
		return ""
	}
	alt := 0.0
	if a, ok := obj["alt"].(float64); ok {
		alt = a
	}
	sats := 0
	// GPSD may report "sats" (satellites used) or it may be absent.
	// Some GPSD versions report it; default to 4 when unknown.
	if v, ok := obj["sat"].(float64); ok {
		sats = int(v)
	} else if v, ok := obj["sats"].(float64); ok {
		sats = int(v)
	}
	if sats <= 0 {
		sats = 4
	}
	quality := 1 // GPS fix
	if mode >= 3 {
		quality = 2 // DGPS / 3D
	}
	// Parse time if present.
	utcTime := ""
	if t, ok := obj["time"].(string); ok && len(t) >= 6 {
		// GPSD time format: "2026-07-05T03:55:18.000Z"
		// Extract HHMMSS.
		if idx := strings.Index(t, "T"); idx >= 0 {
			rest := t[idx+1:]
			rest = strings.TrimSuffix(rest, "Z")
			parts := strings.Split(rest, ":")
			if len(parts) >= 3 {
				sec := parts[2]
				if dotIdx := strings.Index(sec, "."); dotIdx >= 0 {
					sec = sec[:dotIdx]
				}
				utcTime = parts[0] + parts[1] + sec
			}
		}
	}
	if utcTime == "" {
		utcTime = "000000"
	}

	// Convert decimal lat/lon to NMEA DDMM.MMMM format.
	latNS, latDDMM := degToNMEA(lat, true)
	lonEW, lonDDMM := degToNMEA(lon, false)

	// Build $GPGGA sentence (no checksum — our parser ignores it).
	gga := fmt.Sprintf("$GPGGA,%s,%s,%s,%s,%s,%d,%02d,1.0,%.1f,M,0.0,M,,",
		utcTime, latDDMM, latNS, lonDDMM, lonEW, quality, sats, alt)
	return gga
}

// degToNMEA converts decimal degrees to NMEA DDMM.MMMM format and returns
// the hemisphere indicator (N/S or E/W).
func degToNMEA(deg float64, isLat bool) (hemi string, ddmm string) {
	neg := deg < 0
	if neg {
		deg = -deg
	}
	d := int(deg)
	m := (deg - float64(d)) * 60.0
	if isLat {
		hemi = "N"
		if neg {
			hemi = "S"
		}
		return hemi, fmt.Sprintf("%02d%07.4f", d, m)
	}
	hemi = "E"
	if neg {
		hemi = "W"
	}
	return hemi, fmt.Sprintf("%03d%07.4f", d, m)
}

// readLineRaw reads a raw JSON line from GPSD for testing purposes.
func (r *GPSDReader) readLineRaw() (string, error) {
	if r.scanner == nil {
		if err := r.connect(); err != nil {
			return "", err
		}
	}
	if !r.scanner.Scan() {
		err := r.scanner.Err()
		if err != nil {
			return "", err
		}
		return "", io.EOF
	}
	return strings.TrimSpace(r.scanner.Text()), nil
}
