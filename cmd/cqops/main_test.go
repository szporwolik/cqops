package main

import (
	"testing"
)

// TestPackageStructure verifies the main package links correctly
// against its dependency (internal/cli). A build failure here means
// the import path or API is broken.
func TestPackageStructure(t *testing.T) {
	// go build already verifies linkage — no explicit check needed.
}

// TestMainFunctionExists is a compile-time check that main exists.
// The actual main() behaviour (os.Exit on error) is tested via
// integration/CLI tests in internal/cli.
func TestMainFunctionExists(t *testing.T) {
	// main is a package-level function; this test just ensures the
	// package compiles and links. If main were missing or had the
	// wrong signature, `go build` would fail before tests run.
}
