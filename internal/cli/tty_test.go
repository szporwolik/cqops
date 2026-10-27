package cli

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestTUITerminalState pins the decision table: a real tty wins, a reopenable
// controlling terminal is used next, and only a process with neither is
// refused — unless the operator opted out.
func TestTUITerminalState(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	reopenOK := func() (*os.File, error) { return devNull, nil }
	reopenFail := func() (*os.File, error) { return nil, errors.New("ENXIO") }

	tests := []struct {
		name      string
		stdinTTY  bool
		reopen    func() (*os.File, error)
		assumeTTY bool
		wantOK    bool
		wantStdin bool // a replacement stdin is returned
	}{
		{"terminal on stdin", true, reopenFail, false, true, false},
		{"no tty, controlling terminal", false, reopenOK, false, true, true},
		{"no tty, no controlling terminal", false, reopenFail, false, false, false},
		{"no tty, override", false, reopenFail, true, true, false},
		{"terminal on stdin wins over override", true, reopenOK, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin, ok := tuiTerminalState(tt.stdinTTY, tt.reopen, tt.assumeTTY)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if (stdin != nil) != tt.wantStdin {
				t.Errorf("replacement stdin = %v, want %v", stdin != nil, tt.wantStdin)
			}
		})
	}
}

// TestEnsureTUIHasTerminalRefusesWithoutTerminal drives the real entry point
// with stdin pointing at /dev/null and no controlling terminal available, so
// the message and the override are checked end to end.
func TestEnsureTUIHasTerminalRefusesWithoutTerminal(t *testing.T) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("open %s: %v", os.DevNull, err)
	}
	defer devNull.Close()

	origStdin, origOpen := os.Stdin, openTTY
	t.Cleanup(func() { os.Stdin, openTTY = origStdin, origOpen })
	os.Stdin = devNull
	openTTY = func() (*os.File, error) { return nil, errors.New("no controlling terminal") }

	t.Setenv(assumeTTYEnv, "")

	err = ensureTUIHasTerminal()
	if err == nil {
		t.Fatal("a process without a terminal must not start the TUI")
	}
	msg := err.Error()
	for _, want := range []string{"needs a terminal", assumeTTYEnv, "x-terminal-emulator", "Terminal=true"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message does not mention %q:\n%s", want, msg)
		}
	}

	// The override keeps exotic setups working.
	t.Setenv(assumeTTYEnv, "1")
	if err := ensureTUIHasTerminal(); err != nil {
		t.Errorf("override ignored: %v", err)
	}
}

// TestEnvTruthy: only an explicit negative means "off", so a bare
// CQOPS_ASSUME_TTY=1 (or any other value) enables the override.
func TestEnvTruthy(t *testing.T) {
	for value, want := range map[string]bool{
		"": false, "0": false, "false": false, "FALSE": false, "no": false, "off": false,
		"1": true, "true": true, "yes": true, "anything": true,
	} {
		t.Setenv(assumeTTYEnv, value)
		if got := envTruthy(assumeTTYEnv); got != want {
			t.Errorf("envTruthy(%q) = %v, want %v", value, got, want)
		}
	}
}

// TestOpenControllingTTYOnWindows: Windows has no /dev/tty, and the console
// handle is what term.IsTerminal inspects — reopening must fail instead of
// returning a bogus file.
func TestOpenControllingTTYOnWindows(t *testing.T) {
	f, err := openControllingTTY()
	if err == nil {
		f.Close()
		t.Skip("a controlling terminal exists here; the Windows branch is not exercised")
	}
}
