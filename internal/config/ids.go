package config

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

// NewID generates a short unique hex identifier from a seed string.
// The returned ID is 12 hex characters (48 bits), suitable as a map key.
func NewID(seed string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", seed, time.Now().UnixNano())))
	return fmt.Sprintf("%x", h[:6])
}

// LogbookDisplayName returns the human-readable name for a logbook:
// the station callsign, or "Unnamed" if empty.
func LogbookDisplayName(lb *Logbook) string {
	if lb.Station.Callsign != "" {
		return lb.Station.Callsign
	}
	return "Unnamed"
}

// RigDisplayName returns the human-readable name for a rig preset:
// the model name, or "Unnamed" if empty.
func RigDisplayName(rp *RigPreset) string {
	if rp.Model != "" {
		return rp.Model
	}
	return "Unnamed"
}

// SortedLogbookIDs returns logbook IDs sorted by callsign (display name).
func SortedLogbookIDs(cfg *Config) []string {
	type pair struct {
		id   string
		name string
	}
	pairs := make([]pair, 0, len(cfg.Logbooks))
	for id, lb := range cfg.Logbooks {
		pairs = append(pairs, pair{id, LogbookDisplayName(&lb)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].name < pairs[j].name
	})
	ids := make([]string, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
}

// SortedRigIDs returns rig IDs sorted by model (display name).
func SortedRigIDs(cfg *Config) []string {
	type pair struct {
		id   string
		name string
	}
	pairs := make([]pair, 0, len(cfg.Rigs))
	for id, rp := range cfg.Rigs {
		pairs = append(pairs, pair{id, RigDisplayName(&rp)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].name < pairs[j].name
	})
	ids := make([]string, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
}

// FindLogbookByCallsign returns the first logbook whose station callsign
// matches (case-insensitive). Returns the ID, logbook pointer, and true if found.
func FindLogbookByCallsign(cfg *Config, callsign string) (string, *Logbook, bool) {
	for id, lb := range cfg.Logbooks {
		if equalFold(lb.Station.Callsign, callsign) {
			lbCopy := lb
			return id, &lbCopy, true
		}
	}
	return "", nil, false
}

// FindRigByModel returns the first rig whose model name matches
// (case-insensitive). Returns the ID, rig pointer, and true if found.
func FindRigByModel(cfg *Config, model string) (string, *RigPreset, bool) {
	for id, rp := range cfg.Rigs {
		if equalFold(rp.Model, model) {
			rpCopy := rp
			return id, &rpCopy, true
		}
	}
	return "", nil, false
}

// equalFold is a simple case-insensitive string comparison.
func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 32
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 32
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// PopulateIDs ensures every logbook and rig has its ID field set from its
// map key. Call this after loading a config from YAML (where the id field
// is not serialized).
func PopulateIDs(cfg *Config) {
	if cfg.Logbooks != nil {
		for key, lb := range cfg.Logbooks {
			if lb.ID == "" {
				lb.ID = key
				cfg.Logbooks[key] = lb
			}
		}
	}
	if cfg.Rigs != nil {
		for key, rp := range cfg.Rigs {
			if rp.ID == "" {
				rp.ID = key
				cfg.Rigs[key] = rp
			}
		}
	}
}
