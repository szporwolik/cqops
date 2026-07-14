package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/szporwolik/cqops/internal/cli"
)

func main() {
	// Hide the terminal cursor on startup — prevents a double-cursor
	// artifact on Windows conhost where the native block cursor remains
	// visible alongside Bubble Tea's text-input cursor.
	// Restored in the defer below.
	fmt.Print("\033[?25l")
	defer func() {
		fmt.Print("\033[?25h") // restore cursor on exit
	}()

	// Set the console window icon from the embedded resource so
	// Windows Terminal / conhost shows the CQOps icon in the tab.
	setConsoleIcon()

	// Top-level recover: if CQOps panics, show the error and pause so the
	// user can read it before the terminal window closes.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nFATAL: %v\n\n", r)
			pauseIfTerminal()
			os.Exit(1)
		}
	}()

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nCQOps cannot start: %v\n\n", err)
		pauseIfTerminal()
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "CQOps: finished.")
}

// pauseIfTerminal waits for Enter when launched directly (not from an
// existing terminal session). Only active on Windows, where double-clicking
// an .exe opens a new console that closes immediately on exit.
// Times out after 5 seconds to avoid blocking indefinitely on headless systems.
func pauseIfTerminal() {
	if runtime.GOOS != "windows" {
		return
	}
	fmt.Fprint(os.Stderr, "Press Enter to close (5s timeout)...")

	done := make(chan struct{}, 1)
	go func() {
		var buf [1]byte
		os.Stdin.Read(buf[:])
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}
}
