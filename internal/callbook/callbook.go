// Package callbook provides a neutral abstraction over amateur radio callbook
// lookup providers (QRZ.com, HamQTH, Callook.info, etc.).
//
// Provider implementations live in their own packages (internal/qrzcom, etc.)
// and satisfy the callbook.Provider interface. The TUI layer works exclusively
// with callbook.Result and never imports provider-specific types directly.
//
// Multi-provider lookup works via a Registry: providers are ordered by
// decreasing priority. Lookup starts with the top-priority provider and
// cascades through remaining providers to fill any blank fields.
package callbook

import (
	"fmt"
	"sort"
	"strings"

	"github.com/szporwolik/cqops/internal/applog"
)

// Result is a provider-neutral callbook lookup result.
// It mirrors the fields historically returned by QRZ.com but adds a Provider
// field to identify which callbook source produced the data.
type Result struct {
	Callsign string
	Name     string
	Grid     string
	Country  string
	QTH      string
	State    string
	Zip      string
	County   string
	Class    string
	Email    string
	URL      string
	Lat      string
	Lon      string
	DXCC     string
	CQZone   string
	ITUZone  string
	ImageURL string
	// QSL information.
	LoTW       bool     // LoTW member
	EQSL       bool     // eQSL member
	QSLManager string   // QSL manager callsign
	Provider   string   // e.g. "qrz", "hamqth", "callook" — primary source
	Providers  []string // all providers that contributed data (for UI feedback)
}

// Provider is the interface that every callbook lookup service must satisfy.
type Provider interface {
	// Lookup queries the callbook for a callsign.
	// Returns nil, nil when the callsign is not found (not an error).
	Lookup(callsign string) (*Result, error)

	// TestConnection verifies that the provider can reach its backend.
	TestConnection() error

	// Name returns a human-readable provider identifier (e.g. "QRZ.com").
	Name() string

	// Priority returns the lookup order for this provider. Higher values
	// are tried first. Providers with equal priority are tried in
	// registration order.
	Priority() int
}

// Registry holds an ordered set of callbook providers. Lookup tries each
// provider in decreasing priority order and merges results: the first
// provider to return data sets the baseline; subsequent providers fill in
// any remaining blank fields.
type Registry struct {
	providers []Provider
}

// NewRegistry creates a Registry from a slice of providers. Providers are
// sorted by decreasing priority internally; ties preserve the given order.
// Passing nil or an empty slice returns a registry whose Lookup always
// returns nil, nil.
func NewRegistry(providers []Provider) *Registry {
	if len(providers) == 0 {
		return &Registry{}
	}
	// Sort by priority descending, preserving original order on tie.
	sorted := make([]Provider, len(providers))
	copy(sorted, providers)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority() > sorted[j].Priority()
	})
	return &Registry{providers: sorted}
}

// Len returns the number of registered providers.
func (r *Registry) Len() int { return len(r.providers) }

// Lookup queries all providers in priority order. The first provider that
// returns data sets the baseline; subsequent providers fill in blank fields.
// Returns nil, nil when no provider returns data or when the callsign is empty.
func (r *Registry) Lookup(callsign string) (*Result, error) {
	if callsign == "" || len(r.providers) == 0 {
		return nil, nil
	}

	// Build provider name list for the log header.
	names := make([]string, len(r.providers))
	for i, p := range r.providers {
		names[i] = fmt.Sprintf("%s(p=%d)", p.Name(), p.Priority())
	}
	applog.Debug("Callbook: lookup start", "call", callsign, "providers", strings.Join(names, " → "))

	var merged *Result
	var lastErr error

	for _, p := range r.providers {
		data, err := p.Lookup(callsign)
		if err != nil {
			lastErr = err
			applog.Debug("Callbook: provider error", "provider", p.Name(), "call", callsign, "error", err.Error())
			continue
		}
		if data == nil || data.Callsign == "" {
			applog.Debug("Callbook: provider no data", "provider", p.Name(), "call", callsign)
			continue
		}
		// Record this provider as a contributor.
		data.Providers = append(data.Providers, p.Name())
		applog.Debug("Callbook: provider returned data", "provider", p.Name(), "call", callsign,
			"fields", resultFields(data))

		if merged == nil {
			// First successful provider sets the baseline.
			merged = data
			applog.Debug("Callbook: baseline set", "provider", p.Name(), "fields", resultFields(merged))
		} else {
			// Fill in blanks from this provider without overwriting.
			filled := mergeInto(merged, data)
			if filled != "" {
				applog.Debug("Callbook: merged from provider", "provider", p.Name(),
					"filled", filled, "merged", resultFields(merged))
			} else {
				applog.Debug("Callbook: provider had no new fields", "provider", p.Name(),
					"call", callsign)
			}
		}
	}

	if merged == nil {
		if lastErr != nil {
			applog.Debug("Callbook: all providers failed", "call", callsign, "last_error", lastErr.Error())
		} else {
			applog.Debug("Callbook: no data from any provider", "call", callsign)
		}
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, nil
	}

	applog.Debug("Callbook: lookup complete", "call", callsign,
		"providers", strings.Join(merged.Providers, ","), "result", resultFields(merged))
	return merged, nil
}

// MergeInto fills blank fields in dst with values from src.
// Returns a comma-separated list of field names that were filled.
// Exported for use by the TUI layer (base-call fallback merging).
func MergeInto(dst, src *Result) string {
	return mergeInto(dst, src)
}

// mergeInto fills blank fields in dst with values from src.
// Returns a comma-separated list of field names that were filled.
func mergeInto(dst, src *Result) string {
	// Accumulate contributor names from all providers that supplied data.
	dst.Providers = append(dst.Providers, src.Providers...)
	var filled []string

	if dst.Name == "" && src.Name != "" {
		dst.Name = src.Name
		filled = append(filled, "name")
	}
	if dst.Grid == "" && src.Grid != "" {
		dst.Grid = src.Grid
		filled = append(filled, "grid")
	}
	if dst.Country == "" && src.Country != "" {
		dst.Country = src.Country
		filled = append(filled, "country")
	}
	if dst.QTH == "" && src.QTH != "" {
		dst.QTH = src.QTH
		filled = append(filled, "qth")
	}
	if dst.State == "" && src.State != "" {
		dst.State = src.State
		filled = append(filled, "state")
	}
	if dst.Zip == "" && src.Zip != "" {
		dst.Zip = src.Zip
		filled = append(filled, "zip")
	}
	if dst.County == "" && src.County != "" {
		dst.County = src.County
		filled = append(filled, "county")
	}
	if dst.Class == "" && src.Class != "" {
		dst.Class = src.Class
		filled = append(filled, "class")
	}
	if dst.Email == "" && src.Email != "" {
		dst.Email = src.Email
		filled = append(filled, "email")
	}
	if dst.URL == "" && src.URL != "" {
		dst.URL = src.URL
		filled = append(filled, "url")
	}
	if dst.Lat == "" && src.Lat != "" {
		dst.Lat = src.Lat
		filled = append(filled, "lat")
	}
	if dst.Lon == "" && src.Lon != "" {
		dst.Lon = src.Lon
		filled = append(filled, "lon")
	}
	if dst.DXCC == "" && src.DXCC != "" {
		dst.DXCC = src.DXCC
		filled = append(filled, "dxcc")
	}
	if dst.CQZone == "" && src.CQZone != "" {
		dst.CQZone = src.CQZone
		filled = append(filled, "cq")
	}
	if dst.ITUZone == "" && src.ITUZone != "" {
		dst.ITUZone = src.ITUZone
		filled = append(filled, "itu")
	}
	if dst.ImageURL == "" && src.ImageURL != "" {
		dst.ImageURL = src.ImageURL
		filled = append(filled, "image")
	}
	// Provider stays as the first provider that returned data.

	return strings.Join(filled, ",")
}

// resultFields returns a compact log-friendly summary of the non-empty
// fields in a Result. Used for debug logging to show what data each
// provider contributed and what the final merged result looks like.
func resultFields(r *Result) string {
	var parts []string
	add := func(name, val string) {
		if val != "" {
			// Truncate for readability in logs.
			if len(val) > 40 {
				val = val[:37] + "..."
			}
			parts = append(parts, name+"="+val)
		}
	}
	add("name", r.Name)
	add("grid", r.Grid)
	add("country", r.Country)
	add("qth", r.QTH)
	add("state", r.State)
	add("zip", r.Zip)
	add("county", r.County)
	add("class", r.Class)
	add("email", r.Email)
	add("url", r.URL)
	add("lat", r.Lat)
	add("lon", r.Lon)
	add("dxcc", r.DXCC)
	add("cq", r.CQZone)
	add("itu", r.ITUZone)
	add("image", r.ImageURL)
	if len(parts) == 0 {
		return "(empty)"
	}
	return strings.Join(parts, " ")
}
