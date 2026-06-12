package tui

import (
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type ToastLevel int

const (
	ToastInfo ToastLevel = iota
	ToastSuccess
	ToastWarning
	ToastError
)

type Toast struct {
	Level   ToastLevel
	Message string
	Created time.Time
}

type ToastQueue struct {
	mu       sync.Mutex
	items    []Toast
	OutToast *Toast
}

const toastMaxAge = 3 * time.Second

func NewToastQueue() *ToastQueue {
	return &ToastQueue{}
}

func (tq *ToastQueue) Push(level ToastLevel, msg string) {
	if msg == "" {
		return
	}
	tq.mu.Lock()
	tq.items = append(tq.items, Toast{
		Level:   level,
		Message: msg,
		Created: time.Now(),
	})
	tq.mu.Unlock()
}

func (tq *ToastQueue) Info(msg string)    { tq.Push(ToastInfo, msg) }
func (tq *ToastQueue) Success(msg string) { tq.Push(ToastSuccess, msg) }
func (tq *ToastQueue) Warn(msg string)    { tq.Push(ToastWarning, msg) }
func (tq *ToastQueue) Error(msg string)   { tq.Push(ToastError, msg) }

func (tq *ToastQueue) Expire() {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	cutoff := time.Now().Add(-toastMaxAge)
	n := 0
	for _, t := range tq.items {
		if t.Created.After(cutoff) {
			tq.items[n] = t
			n++
		}
	}
	tq.items = tq.items[:n]
}

func (tq *ToastQueue) Active() []Toast {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	result := make([]Toast, len(tq.items))
	copy(result, tq.items)
	return result
}

func (tq *ToastQueue) PopOutToast() *Toast {
	if tq.OutToast == nil {
		return nil
	}
	t := tq.OutToast
	tq.OutToast = nil
	return t
}

var (
	toastInfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	toastSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("46")).
				Bold(true)

	toastWarningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214")).
				Bold(true)

	toastErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

func toastPrefix(level ToastLevel) string {
	switch level {
	case ToastInfo:
		return toastInfoStyle.Render("i")
	case ToastSuccess:
		return toastSuccessStyle.Render("\u2713")
	case ToastWarning:
		return toastWarningStyle.Render("!")
	case ToastError:
		return toastErrorStyle.Render("\u2717")
	}
	return ""
}

func toastLevelStyle(level ToastLevel) lipgloss.Style {
	switch level {
	case ToastInfo:
		return toastInfoStyle
	case ToastSuccess:
		return toastSuccessStyle
	case ToastWarning:
		return toastWarningStyle
	case ToastError:
		return toastErrorStyle
	}
	return toastInfoStyle
}

func RenderToasts(toasts []Toast, width int) string {
	if len(toasts) == 0 {
		return ""
	}
	var lines []string
	showCount := 2
	if len(toasts) < showCount {
		showCount = len(toasts)
	}
	for i := showCount - 1; i >= 0; i-- {
		t := toasts[len(toasts)-1-i]
		prefix := toastPrefix(t.Level)
		msg := toastLevelStyle(t.Level).Render(t.Message)
		lines = append(lines, lipgloss.NewStyle().Padding(0, 1).Render(prefix+" "+msg))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
