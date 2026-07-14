package version

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Version = "dev"

// Commit is the git commit hash, embedded at compile time via -X ldflags.
var Commit = ""

// BuildDate is embedded at compile time via -X ldflags.
// When empty, the binary's modification time is used as fallback.
var BuildDate = ""

const maxVersionSearchDepth = 5

func Resolved() string {
	if Version != "dev" {
		return Version
	}
	if v, ok := readVersionFile(); ok {
		return v
	}
	return "dev"
}

// ResolvedFull returns version with optional commit hash.
func ResolvedFull() string {
	v := Resolved()
	if Commit != "" {
		v += "-" + Commit
	}
	return v
}

// ResolvedDate returns the embedded build date, or falls back to the binary's
// file modification time formatted as ISO 8601.
func ResolvedDate() string {
	if BuildDate != "" {
		return BuildDate
	}
	exe, err := os.Executable()
	if err != nil {
		return "unknown"
	}
	info, err := os.Stat(exe)
	if err != nil {
		return "unknown"
	}
	return info.ModTime().UTC().Format(time.RFC3339)
}

func readVersionFile() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	dir := filepath.Dir(exe)

	for range maxVersionSearchDepth {
		path := filepath.Join(dir, "VERSION")
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data)), true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", false
}
