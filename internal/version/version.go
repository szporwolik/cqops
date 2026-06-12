package version

import (
	"os"
	"path/filepath"
	"strings"
)

var Version = "dev"

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
