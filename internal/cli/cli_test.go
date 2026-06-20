package cli

import (
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// =============================================================================
// formatCLIDate
// =============================================================================

func TestFormatCLIDate(t *testing.T) {
	cases := []struct{ in, want string }{
		{"20260619", "2026-06-19"},
		{"20260101", "2026-01-01"},
		{"20261231", "2026-12-31"},
		{"2026", "--------"},
		{"", "--------"},
		{"2026061", "--------"},
	}
	for _, c := range cases {
		if got := formatCLIDate(c.in); got != c.want {
			t.Errorf("formatCLIDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// =============================================================================
// resolveLogbookArg
// =============================================================================

func TestResolveLogbookArg_ByID(t *testing.T) {
	id := config.NewID("mylog")
	cfg := &config.Config{
		Logbooks: map[string]config.Logbook{
			id: {ID: id, Station: config.Station{Callsign: "SP9ABC"}},
		},
	}

	name, lb, ok := resolveLogbookArg(cfg, id)
	if !ok || name != id || lb.Station.Callsign != "SP9ABC" {
		t.Errorf("got name=%q ok=%v callsign=%q", name, ok, lb.Station.Callsign)
	}
}

func TestResolveLogbookArg_ByCallsign(t *testing.T) {
	id := config.NewID("mylog")
	cfg := &config.Config{
		State: config.StateConfig{ActiveLogbook: id},
		Logbooks: map[string]config.Logbook{
			id: {ID: id, Station: config.Station{Callsign: "SP9ABC"}},
		},
	}

	name, lb, ok := resolveLogbookArg(cfg, "SP9ABC")
	if !ok || lb.Station.Callsign != "SP9ABC" {
		t.Errorf("got name=%q ok=%v callsign=%q", name, ok, lb.Station.Callsign)
	}
}

func TestResolveLogbookArg_NotFound(t *testing.T) {
	cfg := &config.Config{Logbooks: map[string]config.Logbook{}}
	_, _, ok := resolveLogbookArg(cfg, "nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

// =============================================================================
// Command registration (verify no panics)
// =============================================================================

func TestRegisterCommands(t *testing.T) {
	// Must not panic.
	RegisterCommands()
	if rootCmd == nil {
		t.Error("rootCmd is nil")
	}
}

func TestCommandTree(t *testing.T) {
	if rootCmd == nil {
		RegisterCommands()
	}
	found := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		found[c.Name()] = true
	}
	for _, want := range []string{"config", "logbook", "log", "rig", "reset", "version"} {
		if !found[want] {
			t.Errorf("missing subcommand %q", want)
		}
	}
}
