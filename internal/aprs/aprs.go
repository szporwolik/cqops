// Package aprs provides APRS-IS connectivity for position beaconing.
// It handles TCP connection to APRS-IS servers, login authentication,
// and position report formatting per the APRS specification.
package aprs

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/version"
)

// DefaultTimeout is the connection + read timeout for APRS-IS operations.
const DefaultTimeout = 8 * time.Second

// TestConnection verifies that the APRS-IS server is reachable and the
// provided callsign/passcode are accepted for login.
//
// server should be in "host:port" format (e.g. "euro.aprs2.net:14580").
// passcode "-1" is the conventional "read-only / no transmit" passcode.
// A valid amateur radio APRS-IS passcode is a 5-digit number derived from
// the callsign; third-party services can compute it.
func TestConnection(server, callsign, passcode string) error {
	applog.Debug("APRS: testing connection", "server", server, "callsign", callsign)

	if server == "" {
		return fmt.Errorf("APRS server address is required")
	}
	if callsign == "" {
		return fmt.Errorf("APRS callsign is required")
	}
	if passcode == "" {
		return fmt.Errorf("APRS passcode is required")
	}

	conn, err := net.DialTimeout("tcp", server, DefaultTimeout)
	if err != nil {
		applog.Error("APRS: connection failed", "server", server, "error", err)
		return fmt.Errorf("cannot reach APRS server %s: %v", server, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(DefaultTimeout)); err != nil {
		return fmt.Errorf("set deadline: %w", err)
	}

	// Send login string per APRS-IS specification.
	// Format: user {call} pass {code} vers {software} {version}
	// The "filter" parameter is optional; we omit it for test-only connections.
	login := fmt.Sprintf("user %s pass %s vers CQOps %s\r\n", callsign, passcode, version.Resolved())
	if _, err := fmt.Fprint(conn, login); err != nil {
		applog.Error("APRS: write login failed", "error", err)
		return fmt.Errorf("send login: %w", err)
	}

	// Read server banner and login response.
	// The server sends one or more #-prefixed lines, then the logresp line.
	scanner := bufio.NewScanner(conn)
	var lastLine string
	for scanner.Scan() {
		line := scanner.Text()
		applog.Debug("APRS: server line", "line", line)
		lastLine = line
		// Look for the logresp line.
		if strings.HasPrefix(line, "# logresp ") {
			if strings.Contains(line, " verified") {
				applog.Info("APRS: login verified", "callsign", callsign, "server", server)
				return nil
			}
			if strings.Contains(line, "unverified") {
				applog.Error("APRS: login unverified", "callsign", callsign, "line", line)
				return fmt.Errorf("APRS login rejected — check callsign and passcode")
			}
		}
	}
	if err := scanner.Err(); err != nil {
		applog.Error("APRS: read response failed", "error", err)
		return fmt.Errorf("read server response: %w", err)
	}

	// Fallback: no explicit logresp line received.
	applog.Error("APRS: unexpected response", "lastLine", lastLine)
	return fmt.Errorf("APRS server did not confirm login (unexpected response)")
}
