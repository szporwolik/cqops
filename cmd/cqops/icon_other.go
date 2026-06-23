//go:build !windows

package main

// setConsoleIcon is a no-op on non-Windows platforms.
func setConsoleIcon() {}
