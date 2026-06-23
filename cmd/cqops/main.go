package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/szporwolik/cqops/internal/cli"
)

func main() {
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
}

// pauseIfTerminal waits for Enter when launched directly (not from an
// existing terminal session). Only active on Windows, where double-clicking
// an .exe opens a new console that closes immediately on exit.
func pauseIfTerminal() {
	if runtime.GOOS != "windows" {
		return
	}
	fmt.Fprint(os.Stderr, "Press Enter to close...")
	// Use a single byte read from stdin; simplest cross-shell approach.
	var buf [1]byte
	os.Stdin.Read(buf[:])
}
