// Package callook implements the callbook.Provider interface for the
// Callook.info free US callsign database (https://callook.info).
//
// Callook.info provides a simple REST API:
//   - No authentication required.
//   - GET https://callook.info/CALLSIGN/json returns JSON.
//   - US callsigns only — returns status "VALID" for valid calls, "INVALID" otherwise.
package callook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/applog"
	"github.com/szporwolik/cqops/internal/callbook"
)

// --- JSON types ---

type callookResponse struct {
	Status  string `json:"status"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Current struct {
		Callsign  string `json:"callsign"`
		OperClass string `json:"operClass"`
	} `json:"current"`
	Address struct {
		Line1 string `json:"line1"`
		Line2 string `json:"line2"`
	} `json:"address"`
	Location struct {
		Latitude   string `json:"latitude"`
		Longitude  string `json:"longitude"`
		Gridsquare string `json:"gridsquare"`
	} `json:"location"`
}

// --- Client ---

// Client implements callbook.Provider for Callook.info.
type Client struct {
	priority int
	httpFn   func(string) ([]byte, error)
}

// NewClient creates a Callook.info client with default priority (40).
func NewClient() *Client {
	return &Client{priority: 40, httpFn: httpGetFn}
}

// NewClientWithPriority creates a Callook.info client with explicit priority.
func NewClientWithPriority(priority int) *Client {
	return &Client{priority: priority, httpFn: httpGetFn}
}

// Name returns the provider identifier.
func (c *Client) Name() string { return "Callook.info" }

// Priority returns the lookup priority (default 40).
func (c *Client) Priority() int { return c.priority }

// Lookup queries Callook.info for a callsign. Returns nil, nil when the
// callsign is not found or not a US callsign.
func (c *Client) Lookup(callsign string) (*callbook.Result, error) {
	if callsign == "" {
		return nil, nil
	}
	d, err := c.lookup(callsign)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, nil
	}

	// Build QTH from address parts.
	qth := strings.TrimSpace(d.Address.Line2)
	if qth == "" {
		qth = strings.TrimSpace(d.Address.Line1)
	}

	return &callbook.Result{
		Callsign: d.Current.Callsign,
		Name:     strings.TrimSpace(d.Name),
		Grid:     strings.TrimSpace(d.Location.Gridsquare),
		QTH:      qth,
		Lat:      normalizeCoord(d.Location.Latitude),
		Lon:      normalizeCoord(d.Location.Longitude),
		Class:    strings.TrimSpace(d.Current.OperClass),
		Provider: "callook",
	}, nil
}

func (c *Client) lookup(callsign string) (*callookResponse, error) {
	url := "https://callook.info/" + strings.ToUpper(callsign) + "/json"
	data, err := c.httpFn(url)
	if err != nil {
		return nil, fmt.Errorf("Callook: %w", err)
	}
	var r callookResponse
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("Callook json: %w", err)
	}
	if r.Status != "VALID" {
		applog.Debug("Callook: not valid", "callsign", callsign, "status", r.Status)
		return nil, nil
	}
	if r.Current.Callsign == "" {
		applog.Debug("Callook: no callsign in response", "callsign", callsign)
		return nil, nil
	}
	applog.InfoDetail("Callook: lookup ok", fmt.Sprintf("%s — %s", r.Current.Callsign, strings.TrimSpace(r.Name)))
	return &r, nil
}

// TestConnection verifies that the Callook.info API is reachable.
func (c *Client) TestConnection() error {
	applog.Debug("Callook: testing connection")
	_, err := c.lookup("W1AW")
	return err
}

// --- HTTP transport and test seam ---

var httpGetFn = defaultHTTPGet

func defaultHTTPGet(rawURL string) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("callsign not found")
	}
	return io.ReadAll(io.LimitReader(resp.Body, 256*1024))
}

// SetHTTPFn sets the HTTP transport function (test seam).
func SetHTTPFn(fn func(string) ([]byte, error)) func() {
	prev := httpGetFn
	httpGetFn = fn
	return func() { httpGetFn = prev }
}

// --- Helpers ---

func normalizeCoord(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Callook returns ~6 decimal places — keep as-is, just trim.
	return raw
}
