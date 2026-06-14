package wavelog

import (
	"fmt"
	"strings"
)

// FriendlyError translates technical Go/HTTP errors into messages suitable
// for display in toasts and status lines. Use this when returning errors
// from Wavelog API calls so the UI can show them directly.
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()

	// DNS / host not found — try to extract the hostname.
	if strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "Name or service not known") {
		if idx := strings.Index(msg, "lookup "); idx >= 0 {
			rest := msg[idx+7:]
			if end := strings.IndexAny(rest, " :\n"); end > 0 {
				return fmt.Errorf("Cannot reach %s — check the URL", rest[:end])
			}
		}
		return fmt.Errorf("Cannot reach server — check the URL")
	}

	// Connection refused
	if strings.Contains(msg, "connection refused") {
		return fmt.Errorf("Connection refused — is the server running?")
	}

	// Timeout
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") ||
		strings.Contains(msg, "deadline exceeded") {
		return fmt.Errorf("Connection timed out — check the URL and try again")
	}

	// HTTP 401 Unauthorized
	if strings.Contains(msg, "HTTP 401") {
		return fmt.Errorf("Invalid API key — check your Wavelog API key")
	}

	// HTTP 403 Forbidden
	if strings.Contains(msg, "HTTP 403") {
		return fmt.Errorf("Access denied — check your API key permissions")
	}

	// HTTP 404 Not Found
	if strings.Contains(msg, "HTTP 404") {
		return fmt.Errorf("Server not found at this URL — check the address")
	}

	// Other HTTP errors
	if strings.Contains(msg, "HTTP 5") {
		return fmt.Errorf("Server error — the service may be down, try again later")
	}
	if strings.Contains(msg, "HTTP 4") {
		return fmt.Errorf("Request failed — check the URL and API key")
	}

	// TLS / certificate errors
	if strings.Contains(msg, "x509") || strings.Contains(msg, "tls") ||
		strings.Contains(msg, "certificate") {
		return fmt.Errorf("Secure connection failed — check the URL (https vs http)")
	}

	// Connection reset / EOF
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "EOF") {
		return fmt.Errorf("Connection lost — the server may have dropped the connection")
	}

	// Known sentinel messages — pass through as-is.
	if strings.Contains(msg, "URL and API key required") ||
		strings.Contains(msg, "no station profiles") {
		return err
	}

	return err
}
