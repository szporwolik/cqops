package cli

import (
	"errors"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/szporwolik/cqops/internal/applog"
)

// assumeTTYEnv lets an operator skip the terminal check. It exists for the
// setups where stdin is not reported as a tty but drawing still works
// (unusual pty wrappers, remote consoles, screen-scraping front-ends). Not
// the default: the check is here because a launcher that ignores the desktop
// entry's Terminal=true — icewm, i3 and openbox menus, cron, a detached
// shell — started CQOps with no terminal at all, and the app then ran
// headless for a few seconds (opening the logbook, connecting to APRS and
// rig, taking the single-instance lock) before exiting without a word.
const assumeTTYEnv = "CQOPS_ASSUME_TTY"

// openTTY is a seam so tests can simulate a process without a controlling
// terminal. It is /dev/tty on POSIX and always fails on Windows.
var openTTY = openControllingTTY

// ensureTUIHasTerminal makes sure the interactive UI has somewhere to draw.
// Returns a user-facing error when it has not; the caller prints it and the
// process exits before any config, database, rig or APRS work happens.
//
// The order matters: a real tty on stdin is used as is; otherwise the process
// may still have a controlling terminal (stdin redirected inside a terminal),
// in which case stdin is reopened from it — so `cqops < /dev/null` keeps
// working exactly as before. Only a process with no controlling terminal is
// refused, and even then the env override is honoured.
func ensureTUIHasTerminal() error {
	stdin, ok := tuiTerminalState(term.IsTerminal(os.Stdin.Fd()), openTTY, envTruthy(assumeTTYEnv))
	if !ok {
		applog.Warn("TUI refused to start: no terminal attached", "override", assumeTTYEnv)
		return errNoTerminal()
	}
	if stdin != nil {
		os.Stdin = stdin
		applog.Info("TUI: stdin reopened from the controlling terminal")
	}
	return nil
}

// tuiTerminalState is the pure decision behind ensureTUIHasTerminal: ok is
// false only when stdin is not a tty, no controlling terminal can be
// reopened, and the operator did not ask to continue anyway. The returned
// file is the stdin replacement to use (nil keeps the current one).
func tuiTerminalState(stdinIsTTY bool, reopen func() (*os.File, error), assumeTTY bool) (*os.File, bool) {
	if stdinIsTTY {
		return nil, true
	}
	if reopen != nil {
		if f, err := reopen(); err == nil {
			return f, true
		}
	}
	return nil, assumeTTY
}

// openControllingTTY opens the terminal the process is attached to,
// regardless of what stdin currently points at.
func openControllingTTY() (*os.File, error) {
	if runtime.GOOS == "windows" {
		// Windows has no /dev/tty; a console process already has a console
		// handle, which term.IsTerminal reports on stdin.
		return nil, errors.New("no controlling terminal on windows")
	}
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// envTruthy reports whether the variable is set to something other than an
// explicit negative.
func envTruthy(name string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(name))) {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

// errNoTerminal describes the situation in the words the operator needs: what
// happened, why, and the two ways out.
func errNoTerminal() error {
	hint := "  x-terminal-emulator -e cqops      # any terminal emulator works"
	if runtime.GOOS == "windows" {
		hint = "  start cqops from Windows Terminal or Command Prompt"
	}
	return errors.New("CQOps needs a terminal window, and this process has none attached.\n\n" +
		"It was probably started from a desktop or window-manager menu that ignores the\n" +
		"entry's Terminal=true setting (icewm, i3, openbox and friends do), from cron, or\n" +
		"from a detached shell.\n\n" +
		"Start it from a terminal, or wrap it in one:\n\n" +
		hint + "\n\n" +
		"If your terminal really is attached in an unusual way, set " + assumeTTYEnv + "=1\n" +
		"to skip this check.")
}
