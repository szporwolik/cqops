package config

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// NewID generates a deterministic hex identifier from a seed string.
// The returned ID is 12 hex characters (48 bits), suitable as a map key.
// The same seed always produces the same ID — use a unique seed (callsign,
// name) to avoid collisions; do NOT rely on this for randomness.
func NewID(seed string) string {
	h := sha256.Sum256([]byte("cqops/id/v2:" + seed))
	return fmt.Sprintf("%x", h[:6])
}

// LogbookDisplayName returns the human-readable name for a logbook:
// the station callsign, or "Unnamed" if empty.
func LogbookDisplayName(lb *Logbook) string {
	if lb.Name != "" {
		return lb.Name
	}
	if lb.Station.Callsign != "" {
		return lb.Station.Callsign
	}
	return "Unnamed"
}

// RigDisplayName returns the human-readable name for a rig preset:
// the Name field, or the Model field, or "Unnamed".
func RigDisplayName(rp *RigPreset) string {
	if rp.Name != "" {
		return rp.Name
	}
	if rp.Model != "" {
		return rp.Model
	}
	return "Unnamed"
}

// ContestDisplayName returns the human-readable name for a contest.
func ContestDisplayName(c *Contest) string {
	if c.Name != "" {
		return c.Name
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
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
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
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
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

// SortedOperatorIDs returns operator IDs sorted by callsign.
func SortedOperatorIDs(cfg *Config) []string {
	type pair struct {
		id   string
		name string
	}
	pairs := make([]pair, 0, len(cfg.Operators))
	for id, op := range cfg.Operators {
		pairs = append(pairs, pair{id, OperatorDisplayName(&op)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
	})
	ids := make([]string, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
}

// OperatorDisplayName returns the human-readable name for an operator.
func OperatorDisplayName(op *Operator) string {
	if op.Callsign != "" {
		n := op.Callsign
		if op.Name != "" {
			n += " (" + op.Name + ")"
		}
		return n
	}
	return "Unnamed"
}

// FindOperatorByCallsign returns the first operator whose callsign matches
// (case-insensitive). Returns the ID, operator pointer, and true if found.
func FindOperatorByCallsign(cfg *Config, callsign string) (string, *Operator, bool) {
	for id, op := range cfg.Operators {
		if equalFold(op.Callsign, callsign) {
			opCopy := op
			return id, &opCopy, true
		}
	}
	return "", nil, false
}

// OperatorSlice returns operators ordered by callsign as a slice.
func OperatorSlice(cfg *Config) []Operator {
	ids := SortedOperatorIDs(cfg)
	ops := make([]Operator, 0, len(ids))
	for _, id := range ids {
		ops = append(ops, cfg.Operators[id])
	}
	return ops
}

// Pass empty logbookID to get all contests (backward compatibility).
// Use ActiveContestIDs for cycling — that one excludes not-in-use contests.
func SortedContestIDs(cfg *Config, logbookID string) []string {
	type pair struct {
		id   string
		name string
	}
	pairs := make([]pair, 0, len(cfg.Contests))
	for id, c := range cfg.Contests {
		if logbookID != "" && c.LogbookID != logbookID {
			continue
		}
		pairs = append(pairs, pair{id, ContestDisplayName(&c)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
	})
	ids := make([]string, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
}

// ActiveContestIDs returns contest IDs for the given logbook, sorted by name,
// excluding contests where InUse is explicitly set to false. Used for Ctrl+C cycling.
// Pass empty logbookID to get all contests (backward compatibility).
func ActiveContestIDs(cfg *Config, logbookID string) []string {
	type pair struct {
		id   string
		name string
	}
	pairs := make([]pair, 0, len(cfg.Contests))
	for id, c := range cfg.Contests {
		if c.InUse != nil && !*c.InUse {
			continue
		}
		if logbookID != "" && c.LogbookID != logbookID {
			continue
		}
		pairs = append(pairs, pair{id, ContestDisplayName(&c)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return strings.ToLower(pairs[i].name) < strings.ToLower(pairs[j].name)
	})
	ids := make([]string, len(pairs))
	for i, p := range pairs {
		ids[i] = p.id
	}
	return ids
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

// PopulateIDs ensures every logbook, rig, contest, and operator has its
// ID field set from its map key. Call this after loading a config from YAML.
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
	if cfg.Contests != nil {
		for key, c := range cfg.Contests {
			if c.ID == "" {
				c.ID = key
				cfg.Contests[key] = c
			}
		}
	}
	if cfg.Operators != nil {
		for key, op := range cfg.Operators {
			if op.ID == "" {
				op.ID = key
				cfg.Operators[key] = op
			}
		}
	}
}
