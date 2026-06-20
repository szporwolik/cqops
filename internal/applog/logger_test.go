package applog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// Ring buffer / Append + Entries
// =============================================================================

func TestAppend_StoresEntry(t *testing.T) {
	// Reset global state
	mu.Lock()
	entries = nil
	mu.Unlock()

	Append("INFO", "test message", "detail")
	got := Entries()
	if len(got) != 1 || got[0].Message != "test message" || got[0].Details != "detail" || got[0].Level != "INFO" {
		t.Errorf("got %+v", got)
	}
}

func TestAppend_RingBufferEviction(t *testing.T) {
	mu.Lock()
	entries = nil
	mu.Unlock()

	// Fill beyond maxStored
	for i := 0; i < maxStored+10; i++ {
		Append("INFO", "msg", "")
	}
	got := Entries()
	if len(got) != maxStored {
		t.Errorf("expected %d entries, got %d", maxStored, len(got))
	}
}

func TestEntries_CopySafety(t *testing.T) {
	mu.Lock()
	entries = nil
	mu.Unlock()

	Append("INFO", "orig", "")
	got := Entries()
	got[0].Message = "mutated"

	got2 := Entries()
	if got2[0].Message != "orig" {
		t.Error("Entries() should return a copy, mutation leaked")
	}
}

// =============================================================================
// Nil-safety: log functions must not panic when Logger is nil
// =============================================================================

func TestLogFunctions_NilSafe(t *testing.T) {
	Logger = nil
	// Must not panic.
	Debug("debug", "key", "val")
	Info("info", "key", "val")
	Warn("warn", "key", "val")
	Error("error", "key", "val")
	InfoDetail("info", "details")
	ErrorDetail("error", "details")
	DebugDetail("debug", "details")
}

// =============================================================================
// Beep callback
// =============================================================================

func TestBeepCallback_FiresOnError(t *testing.T) {
	Logger = nil
	called := false
	SetBeepFunc(func() { called = true })
	defer SetBeepFunc(nil)

	Error("test error")
	if !called {
		t.Error("beep not called on Error")
	}

	called = false
	ErrorDetail("test", "detail")
	if !called {
		t.Error("beep not called on ErrorDetail")
	}
}

func TestBeepCallback_NotOnInfo(t *testing.T) {
	Logger = nil
	called := false
	SetBeepFunc(func() { called = true })
	defer SetBeepFunc(nil)

	Info("info")
	if called {
		t.Error("beep should not fire on Info")
	}
	Warn("warn")
	if called {
		t.Error("beep should not fire on Warn")
	}
	Debug("debug")
	if called {
		t.Error("beep should not fire on Debug")
	}
}

// =============================================================================
// rotateLogs
// =============================================================================

func TestRotateLogs_DeletesOldFiles(t *testing.T) {
	dir := t.TempDir()
	logDir = dir

	// Create a file older than retention.
	old := filepath.Join(dir, "cqops-2020-01-01T00-00-00.log")
	os.WriteFile(old, []byte("old"), 0644)
	// Create a recent file.
	recent := filepath.Join(dir, "cqops-"+time.Now().Add(-1*time.Hour).Format("2006-01-02T15-04-05")+".log")
	os.WriteFile(recent, []byte("recent"), 0644)

	rotateLogs()

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old log file should have been deleted")
	}
	if _, err := os.Stat(recent); os.IsNotExist(err) {
		t.Error("recent log file should not have been deleted")
	}
}

func TestRotateLogs_CapsFileCount(t *testing.T) {
	dir := t.TempDir()
	logDir = dir

	// Create maxLogFiles + 5 recent files.
	for i := 0; i < maxLogFiles+5; i++ {
		name := "cqops-" + time.Now().Add(time.Duration(-i)*time.Hour).Format("2006-01-02T15-04-05") + ".log"
		os.WriteFile(filepath.Join(dir, name), []byte("x"), 0644)
	}

	rotateLogs()

	files, _ := os.ReadDir(dir)
	logCount := 0
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "cqops-") && strings.HasSuffix(f.Name(), ".log") {
			logCount++
		}
	}
	if logCount > maxLogFiles {
		t.Errorf("expected <= %d log files, got %d", maxLogFiles, logCount)
	}
}

func TestRotateLogs_IgnoresNonLogs(t *testing.T) {
	dir := t.TempDir()
	logDir = dir

	os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("yaml"), 0644)
	os.WriteFile(filepath.Join(dir, "cqops-"+time.Now().Format("2006-01-02T15-04-05")+".log"), []byte("x"), 0644)

	rotateLogs()

	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); os.IsNotExist(err) {
		t.Error("non-log file should not be touched")
	}
}

// =============================================================================
// nowStamp
// =============================================================================

func TestNowStamp(t *testing.T) {
	s := nowStamp()
	if len(s) != 8 || s[2] != ':' || s[5] != ':' {
		t.Errorf("nowStamp = %q, expected HH:MM:SS", s)
	}
}
