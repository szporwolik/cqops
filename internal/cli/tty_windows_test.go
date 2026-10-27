//go:build windows

package cli

import (
	"os"
	"testing"
)

// TestWindowsConsoleSurvivesRedirectedStdin reproduces the regression the
// guard introduced for cmd.exe users: 'cqops < NUL' has a console, so it must
// start. The console input buffer (CONIN$) is reopened and verified instead of
// refusing every Windows process whose stdin is redirected.
//
// Skipped when the test process itself has no console (some CI sandboxes do
// not attach one) — the decision table in tty_test.go still covers the logic
// there.
func TestWindowsConsoleSurvivesRedirectedStdin(t *testing.T) {
	if f, err := openControllingTTY(); err != nil {
		t.Skipf("no console attached to this test process: %v", err)
	} else {
		f.Close()
	}

	nul, err := os.Open(os.DevNull) // "NUL" on Windows
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer nul.Close()

	origStdin, origOpen := os.Stdin, openTTY
	t.Cleanup(func() { os.Stdin, openTTY = origStdin, origOpen })
	os.Stdin = nul
	openTTY = openControllingTTY

	t.Setenv(assumeTTYEnv, "")

	if err := ensureTUIHasTerminal(); err != nil {
		t.Fatalf("a console process with redirected stdin must still start: %v", err)
	}
	// The guard replaced stdin with the console handle, so the TUI has input.
	if os.Stdin == nul {
		t.Error("stdin was not reopened from the console")
	}
	if !isTerminalFile(os.Stdin) {
		t.Error("stdin is not a terminal after reopening")
	}
}
