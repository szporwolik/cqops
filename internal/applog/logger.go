package applog

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/szporwolik/cqops/internal/config"
)

// Logger is the application-wide structured logger.
var Logger *slog.Logger

var (
	mu      sync.Mutex
	entries []Entry
	logDir  string
)

const maxStored = 500
const retentionDays = 7

// Entry is a single in-memory log record exposed via the TUI log viewer.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Details string `json:"details,omitempty"`
}

// Init initializes the structured logger, creates the log directory,
// and starts a background goroutine that prunes old log files.
func Init() error {
	var err error
	logDir, err = config.LogDir()
	if err != nil {
		return err
	}
	os.MkdirAll(logDir, 0755)

	f, err := openLogFile()
	if err != nil {
		return err
	}

	Logger = slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(Logger)

	go cleanupOldLogs()
	return nil
}

func openLogFile() (*os.File, error) {
	name := "cqops-" + time.Now().Format("2006-01-02T15-04-05") + ".log"
	return os.OpenFile(filepath.Join(logDir, name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func cleanupOldLogs() {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, "cqops-") || !strings.HasSuffix(n, ".log") {
			continue
		}
		stem := strings.TrimPrefix(n, "cqops-")
		stem = strings.TrimSuffix(stem, ".log")
		t, err := time.Parse("2006-01-02T15-04-05", stem)
		if err != nil {
			t, err = time.Parse("2006-01-02", stem)
			if err != nil {
				continue
			}
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(logDir, e.Name()))
		}
	}
}

// Append stores a log entry in the in-memory ring buffer (for TUI display).
func Append(level, msg, details string) {
	e := Entry{Time: nowStamp(), Level: level, Message: msg, Details: details}
	mu.Lock()
	entries = append(entries, e)
	if len(entries) > maxStored {
		entries = entries[len(entries)-maxStored:]
	}
	mu.Unlock()
}

// Entries returns a sorted copy of all in-memory log entries.
func Entries() []Entry {
	mu.Lock()
	sort.Slice(entries, func(i, j int) bool { return entries[i].Time < entries[j].Time })
	result := make([]Entry, len(entries))
	copy(result, entries)
	mu.Unlock()
	return result
}

// Debug logs a message at DEBUG level.
func Debug(msg string, args ...any) {
	Append("DEBUG", msg, "")
	Logger.Debug(msg, args...)
}

// Info logs a message at INFO level.
func Info(msg string, args ...any) {
	Append("INFO", msg, "")
	Logger.Info(msg, args...)
}

// Warn logs a message at WARN level.
func Warn(msg string, args ...any) {
	Append("WARN", msg, "")
	Logger.Warn(msg, args...)
}

// Error logs a message at ERROR level.
func Error(msg string, args ...any) {
	Append("ERROR", msg, "")
	Logger.Error(msg, args...)
}

// InfoDetail logs an INFO message with an additional detail string shown
// in the TUI log viewer.
func InfoDetail(msg, details string) {
	Append("INFO", msg, details)
	Logger.Info(msg, "details", details)
}

func nowStamp() string {
	return time.Now().Format("15:04:05")
}
