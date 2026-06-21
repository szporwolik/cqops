package cli

import (
	"testing"
)

// =============================================================================
// Command registration (verify no panics)
// =============================================================================

func TestCommandTree(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd is nil")
	}
	found := map[string]bool{}
	for _, c := range rootCmd.Commands() {
		found[c.Name()] = true
	}
	// Only "version" subcommand remains.
	if !found["version"] {
		t.Error("missing subcommand 'version'")
	}
	// Verify no legacy subcommands leaked.
	for _, legacy := range []string{"config", "logbook", "log", "rig", "reset"} {
		if found[legacy] {
			t.Errorf("legacy subcommand %q should not exist", legacy)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	if versionCmd == nil {
		t.Error("versionCmd is nil")
	}
	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want 'version'", versionCmd.Use)
	}
}

func TestRootCommand(t *testing.T) {
	if rootCmd == nil {
		t.Error("rootCmd is nil")
	}
	if rootCmd.Use != "cqops" {
		t.Errorf("rootCmd.Use = %q, want 'cqops'", rootCmd.Use)
	}
}
