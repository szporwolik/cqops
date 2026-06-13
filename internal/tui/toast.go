package tui

import (
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/applog"
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
	mu    sync.Mutex
	items []Toast
}

const toastMaxAge = 5 * time.Second

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

	// Also log every toast
	switch level {
	case ToastInfo, ToastSuccess:
		applog.Info("toast: " + msg)
	case ToastWarning:
		applog.Warn("toast: " + msg)
	case ToastError:
		applog.Error("toast: " + msg)
	}
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

func toastPrefix(level ToastLevel) string {
	switch level {
	case ToastInfo:
		return S.ToastInfo.Render("i")
	case ToastSuccess:
		return S.ToastSuccess.Render("OK")
	case ToastWarning:
		return S.ToastWarning.Render("!")
	case ToastError:
		return S.ToastError.Render("ERR")
	}
	return ""
}

func toastLevelStyle(level ToastLevel) lipgloss.Style {
	switch level {
	case ToastInfo:
		return S.ToastInfo
	case ToastSuccess:
		return S.ToastSuccess
	case ToastWarning:
		return S.ToastWarning
	case ToastError:
		return S.ToastError
	}
	return S.ToastInfo
}

func RenderToasts(toasts []Toast, width int) string {
	if len(toasts) == 0 {
		return ""
	}
	var lines []string
	showCount := 5
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

// RenderToastOverlay returns the full view with toasts composited as a
// floating overlay in the bottom-right corner, per lipgloss compositing.
// If there are no active toasts, mainView is returned unchanged.
func RenderToastOverlay(mainView string, toasts []Toast, viewW, viewH int) string {
	if len(toasts) == 0 {
		return mainView
	}

	// Build the toast content
	var lines []string
	showCount := 5
	if len(toasts) < showCount {
		showCount = len(toasts)
	}
	for i := showCount - 1; i >= 0; i-- {
		t := toasts[len(toasts)-1-i]
		prefix := toastPrefix(t.Level)
		msg := toastLevelStyle(t.Level).Render(t.Message)
		lines = append(lines, prefix+" "+msg)
	}
	toastContent := lipgloss.JoinVertical(lipgloss.Right, lines...)

	toastW := lipgloss.Width(toastContent)
	toastH := lipgloss.Height(toastContent)

	// Position: bottom-right corner with 2-cell margin from edges
	x := viewW - toastW - 2
	if x < 0 {
		x = 0
	}
	y := viewH - toastH - 1
	if y < 0 {
		y = 0
	}

	// Composite: main content as base layer, toasts floating on top
	base := lipgloss.NewLayer(mainView)
	toastLayer := lipgloss.NewLayer(toastContent).
		X(x).
		Y(y).
		Z(1) // above base

	return lipgloss.NewCompositor(base, toastLayer).Render()
}
