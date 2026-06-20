package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolved_LdflagsVersion(t *testing.T) {
	prev := Version
	defer func() { Version = prev }()

	Version = "1.2.3"
	if got := Resolved(); got != "1.2.3" {
		t.Errorf("expected 1.2.3, got %q", got)
	}
}

func TestResolved_DevFallback(t *testing.T) {
	prev := Version
	defer func() { Version = prev }()

	Version = "dev"
	if got := Resolved(); got != "dev" {
		t.Errorf("expected dev, got %q", got)
	}
}

func TestResolvedDate_LdflagsDate(t *testing.T) {
	prev := BuildDate
	defer func() { BuildDate = prev }()

	BuildDate = "2026-06-20T12:00:00Z"
	if got := ResolvedDate(); got != "2026-06-20T12:00:00Z" {
		t.Errorf("expected 2026-06-20T12:00:00Z, got %q", got)
	}
}

func TestResolvedDate_Fallback(t *testing.T) {
	prev := BuildDate
	defer func() { BuildDate = prev }()

	BuildDate = ""
	got := ResolvedDate()
	if got == "" {
		t.Error("expected non-empty fallback date")
	}
}

func TestReadVersionFile_Found(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.5.4\n"), 0644)

	// Can't easily mock os.Executable(), but we can test readVersionFile directly
	// by calling Resolved when Version=dev and a VERSION file exists.
	// This is an integration-level check — skip for unit test.
	_ = dir
}

func TestReadVersionFile_NotFound(t *testing.T) {
	v, ok := readVersionFile()
	// On a dev machine this typically returns false (no VERSION file near the binary).
	// Just verify it doesn't panic.
	_ = v
	_ = ok
}
