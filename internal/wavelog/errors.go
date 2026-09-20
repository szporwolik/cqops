package wavelog

import (
	"fmt"
	"strings"
)

// friendlyAPIError translates API v2 error codes into user-facing messages.
func friendlyAPIError(e *APIError) error {
	switch e.Code {
	case "unauthorized", "invalid_token":
		return fmt.Errorf("invalid API token — Wavelog API v2 token (wl2_) required since CQOps 0.11.0")
	case "token_expired":
		return fmt.Errorf("API token expired — create a new token in Wavelog")
	case "insufficient_scope":
		if s := e.RequiredScope(); s != "" {
			return fmt.Errorf("API token lacks scope %s — grant it in Wavelog", s)
		}
		return fmt.Errorf("API token lacks a required scope — check token permissions in Wavelog")
	case "rate_limited":
		if n := e.RetryAfterSeconds(); n > 0 {
			return fmt.Errorf("too many requests — retry in %ds", n)
		}
		return fmt.Errorf("too many requests — slow down")
	case "not_found":
		return fmt.Errorf("not found on the Wavelog server")
	case "conflict":
		return fmt.Errorf("duplicate — the QSO already exists in Wavelog")
	case "validation_error", "invalid_json":
		if e.Message != "" {
			return fmt.Errorf("%s", e.Message)
		}
		return fmt.Errorf("request rejected by Wavelog")
	case "forbidden", "club_access_revoked", "insufficient_club_permission":
		return fmt.Errorf("access denied — check the token permissions")
	}
	return e
}

// FriendlyError translates technical Go/HTTP errors into messages suitable
// for display in toasts and status lines. Use this when returning errors
// from Wavelog API calls so the UI can show them directly.
func FriendlyError(err error) error {
	if err == nil {
		return nil
	}
	if apiErr, ok := err.(*APIError); ok {
		return friendlyAPIError(apiErr)
	}
	msg := err.Error()

	// DNS / host not found — try to extract the hostname.
	if strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "Name or service not known") {
		if idx := strings.Index(msg, "lookup "); idx >= 0 {
			rest := msg[idx+7:]
			if end := strings.IndexAny(rest, " :\n"); end > 0 {
				return fmt.Errorf("cannot reach %s — check the URL", rest[:end])
			}
		}
		return fmt.Errorf("cannot reach server — check the URL")
	}

	// Connection refused
	if strings.Contains(msg, "connection refused") {
		return fmt.Errorf("connection refused — is the server running?")
	}

	// Timeout
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "Timeout") ||
		strings.Contains(msg, "deadline exceeded") {
		return fmt.Errorf("connection timed out — check the URL and try again")
	}

	// HTTP 401 — differentiate between invalid key and station access.
	if strings.Contains(msg, "HTTP 401") {
		if strings.Contains(msg, "Station ID not accessible") {
			return fmt.Errorf("station profile not accessible — check your Wavelog Station Profile ID")
		}
		return fmt.Errorf("invalid API key — check your Wavelog API key")
	}

	// HTTP 403 Forbidden
	if strings.Contains(msg, "HTTP 403") {
		return fmt.Errorf("access denied — check your API key permissions")
	}

	// HTTP 404 Not Found
	if strings.Contains(msg, "HTTP 404") {
		return fmt.Errorf("server not found at this URL — check the address")
	}

	// Other HTTP errors
	if strings.Contains(msg, "HTTP 5") {
		return fmt.Errorf("server error — the service may be down, try again later")
	}
	if strings.Contains(msg, "HTTP 4") {
		return fmt.Errorf("request failed — check the URL and API key")
	}

	// TLS / certificate errors
	if strings.Contains(msg, "x509") || strings.Contains(msg, "tls") ||
		strings.Contains(msg, "certificate") {
		return fmt.Errorf("secure connection failed — check the URL (https vs http)")
	}

	// Connection reset / EOF
	if strings.Contains(msg, "connection reset") || strings.Contains(msg, "EOF") {
		return fmt.Errorf("connection lost — the server may have dropped the connection")
	}

	// Known sentinel messages — pass through as-is.
	if strings.Contains(msg, "URL and API key required") ||
		strings.Contains(msg, "no station profiles") {
		return err
	}

	return err
}
