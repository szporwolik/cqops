package wavelog

import (
	"encoding/json"
	"fmt"
	"strings"
)

// apiError is a generic Wavelog API error response.
type apiError struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// extractAPIReason tries to pull the "reason" field from a JSON error body.
func extractAPIReason(bodyStr string) string {
	var ae apiError
	if err := json.Unmarshal([]byte(bodyStr), &ae); err == nil && ae.Reason != "" {
		return ae.Reason
	}
	return ""
}

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

	// HTTP 401 — differentiate between invalid key and station access.
	if strings.Contains(msg, "HTTP 401") {
		if strings.Contains(msg, "Station ID not accessible") {
			return fmt.Errorf("Station profile not accessible — check your Wavelog Station Profile ID")
		}
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

// uploadErrorDetail extracts a user-friendly message from a Wavelog upload
// error response. It parses the structured JSON when available and strips
// HTML tags from the server messages.
func uploadErrorDetail(result *QSOUploadResult, bodyStr string) string {
	// Prefer structured messages from the parsed response.
	if result != nil && len(result.Messages) > 0 {
		var parts []string
		for _, m := range result.Messages {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			// Strip HTML tags.
			m = stripHTML(m)
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			parts = append(parts, m)
		}
		if len(parts) > 0 {
			msg := strings.Join(parts, "; ")
			// Translate known Wavelog error patterns.
			if strings.Contains(msg, "Differing station callsign") {
				return "Station callsign mismatch — check your station profile settings"
			}
			return msg
		}
	}

	// Fallback: try to extract reason from the raw body.
	if reason := extractAPIReason(bodyStr); reason != "" {
		return reason
	}

	// Last resort: return the raw body (truncated).
	if len(bodyStr) > 200 {
		bodyStr = bodyStr[:200] + "…"
	}
	return bodyStr
}

// stripHTML removes simple HTML tags from a string.
func stripHTML(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
