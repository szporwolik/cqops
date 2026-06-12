package log

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/szporwolik/cqops/internal/config"
)

var Logger *slog.Logger
var entries []Entry
var logDir string

const maxStored = 500
const retentionDays = 7

type Entry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"msg"`
	Details string `json:"details,omitempty"`
}

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
	name := "cqops-" + time.Now().Format("2006-01-02") + ".log"
	return os.OpenFile(filepath.Join(logDir, name), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func cleanupOldLogs() {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "cqops-") || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		dateStr := strings.TrimPrefix(e.Name(), "cqops-")
		dateStr = strings.TrimSuffix(dateStr, ".log")
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(logDir, e.Name()))
		}
	}
}

func Append(level, msg, details string) {
	e := Entry{Time: nowStamp(), Level: level, Message: msg, Details: details}
	entries = append(entries, e)
	if len(entries) > maxStored {
		entries = entries[len(entries)-maxStored:]
	}
}

func Entries() []Entry {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Time < entries[j].Time })
	return entries
}

func Debug(msg string, args ...any) {
	Append("DEBUG", msg, "")
	Logger.Debug(msg, args...)
}
func Info(msg string, args ...any) {
	Append("INFO", msg, "")
	Logger.Info(msg, args...)
}
func Warn(msg string, args ...any) {
	Append("WARN", msg, "")
	Logger.Warn(msg, args...)
}
func Error(msg string, args ...any) {
	Append("ERROR", msg, "")
	Logger.Error(msg, args...)
}

func InfoDetail(msg, details string) {
	Append("INFO", msg, details)
	Logger.Info(msg, "details", details)
}

func nowStamp() string {
	return time.Now().Format("15:04:05")
}
