package applog

import (
	"fmt"
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
	mu        sync.Mutex
	entries   []Entry
	logDir    string
	beepFn    func() // called on errors when set (see SetBeepFunc)
	debugMode bool   // when false, Debug/DebugDetail are suppressed
)

const maxStored = 100               // max in-memory log entries for TUI viewer
const maxLogFiles = 20              // keep at most N rotated log files
const retentionDays = 7             // delete log files older than this
const maxLogSize = 10 * 1024 * 1024 // rotate current log file when it exceeds 10 MB

// Entry is a single in-memory log record exposed via the TUI log viewer.
type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Details string `json:"details,omitempty"`
}

// rotateWriter is an io.Writer that automatically rotates to a new file
// when the current one exceeds maxLogSize. Safe for concurrent use.
type rotateWriter struct {
	mu   sync.Mutex
	file *os.File
	dir  string
	size int64
}

func (w *rotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.size >= maxLogSize {
		if err := w.rotate(); err != nil {
			return 0, fmt.Errorf("log rotate: %w", err)
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotateWriter) rotate() error {
	if w.file != nil {
		w.file.Close()
	}
	f, err := openLogFileIn(w.dir)
	if err != nil {
		return err
	}
	w.file = f
	w.size = 0
	return nil
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

	rw := &rotateWriter{dir: logDir}
	if err := rw.rotate(); err != nil {
		return err
	}

	Logger = slog.New(slog.NewTextHandler(rw, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(Logger)

	go cleanupOldLogs()
	return nil
}

func openLogFileIn(dir string) (*os.File, error) {
	name := "cqops-" + time.Now().Format("2006-01-02T15-04-05") + ".log"
	return os.OpenFile(filepath.Join(dir, name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func cleanupOldLogs() {
	// Run once at startup, then periodically
	rotateLogs()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		rotateLogs()
	}
}

// rotateLogs deletes log files older than retentionDays and keeps at most
// maxLogFiles. Files are sorted by name (which includes timestamp), so the
// oldest are deleted first.
func rotateLogs() {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	// Collect log files with their timestamps
	type logFile struct {
		name string
		t    time.Time
	}
	var files []logFile

	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, "cqops-") || !strings.HasSuffix(n, ".log") {
			continue
		}
		stem := strings.TrimPrefix(n, "cqops-")
		stem = strings.TrimSuffix(stem, ".log")
		t, err := time.Parse("2006-01-02T15-04-05", stem)
		if err != nil {
			continue
		}
		files = append(files, logFile{name: n, t: t})
	}

	// Sort oldest first
	sort.Slice(files, func(i, j int) bool { return files[i].t.Before(files[j].t) })

	// Delete files older than retention period
	keepStart := 0
	for keepStart < len(files) && files[keepStart].t.Before(cutoff) {
		os.Remove(filepath.Join(logDir, files[keepStart].name))
		keepStart++
	}

	// If too many files remain, delete oldest beyond maxLogFiles
	if len(files)-keepStart > maxLogFiles {
		for i := keepStart; i < len(files)-maxLogFiles; i++ {
			os.Remove(filepath.Join(logDir, files[i].name))
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

// Debug logs a message at DEBUG level. Safe when Logger is nil (tests).
func Debug(msg string, args ...any) {
	if !debugMode {
		return
	}
	Append("DEBUG", msg, "")
	if Logger != nil {
		Logger.Debug(msg, args...)
	}
}

// Info logs a message at INFO level. Safe when Logger is nil (tests).
func Info(msg string, args ...any) {
	Append("INFO", msg, "")
	if Logger != nil {
		Logger.Info(msg, args...)
	}
}

// Warn logs a message at WARN level. Safe when Logger is nil (tests).
func Warn(msg string, args ...any) {
	Append("WARN", msg, "")
	if Logger != nil {
		Logger.Warn(msg, args...)
	}
}

// Error logs a message at ERROR level. Safe when Logger is nil (tests).
// If SetBeepFunc has been configured, also triggers a system beep.
func Error(msg string, args ...any) {
	Append("ERROR", msg, "")
	if Logger != nil {
		Logger.Error(msg, args...)
	}
	if beepFn != nil {
		beepFn()
	}
}

// InfoDetail logs an INFO message with an additional detail string shown
// in the TUI log viewer. Safe when Logger is nil (tests).
func InfoDetail(msg, details string) {
	Append("INFO", msg, details)
	if Logger != nil {
		Logger.Info(msg, "details", details)
	}
}

// ErrorDetail logs an ERROR message with an additional detail string shown
// in the TUI log viewer. Safe when Logger is nil (tests).
// If SetBeepFunc has been configured, also triggers a system beep.
func ErrorDetail(msg, details string) {
	Append("ERROR", msg, details)
	if Logger != nil {
		Logger.Error(msg, "details", details)
	}
	if beepFn != nil {
		beepFn()
	}
}

// DebugDetail logs a DEBUG message with an additional detail string shown
// in the TUI log viewer. Safe when Logger is nil (tests).
func DebugDetail(msg, details string) {
	if !debugMode {
		return
	}
	Append("DEBUG", msg, details)
	if Logger != nil {
		Logger.Debug(msg, "details", details)
	}
}

func nowStamp() string {
	return time.Now().Format("15:04:05")
}

// SetBeepFunc registers a callback that is invoked on every ERROR-level log.
// Pass nil to disable. Callers should use this to trigger a system beep.
func SetBeepFunc(fn func()) {
	beepFn = fn
}

// SetDebugMode enables or disables DEBUG-level log output.
// Default is false (debug logs suppressed).
func SetDebugMode(on bool) {
	debugMode = on
}
