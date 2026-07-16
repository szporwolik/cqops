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
	mu       sync.Mutex
	items    []Toast
	lastMsg  string
	lastTime time.Time
	lastPush time.Time // global rate limiter: min interval between any toasts
}

const toastMaxAge = 5 * time.Second
const toastMaxItems = 20
const toastDedupWindow = 2 * time.Second        // suppress identical messages within this window
const toastMinInterval = 500 * time.Millisecond // rate limit: at most one toast per half second

func NewToastQueue() *ToastQueue {
	return &ToastQueue{}
}

func (tq *ToastQueue) Push(level ToastLevel, msg string) {
	if msg == "" {
		return
	}
	tq.mu.Lock()
	// Dedup: suppress identical messages within the dedup window.
	if msg == tq.lastMsg && time.Since(tq.lastTime) < toastDedupWindow {
		tq.mu.Unlock()
		return
	}
	// Rate limit: at most one toast per minimum interval.
	if time.Since(tq.lastPush) < toastMinInterval {
		tq.mu.Unlock()
		return
	}
	tq.lastMsg = msg
	tq.lastTime = time.Now()
	tq.lastPush = tq.lastTime
	tq.items = append(tq.items, Toast{
		Level:   level,
		Message: msg,
		Created: tq.lastTime,
	})
	// Cap queue size to prevent unbounded growth on error loops.
	if len(tq.items) > toastMaxItems {
		tq.items = tq.items[len(tq.items)-toastMaxItems:]
	}
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
	if len(tq.items) == 0 {
		return nil
	}
	result := make([]Toast, len(tq.items))
	copy(result, tq.items)
	return result
}

// toastPrefix returns a non-emoji UTF-8 symbol for each toast level so the
// toast type is distinguishable even on black-and-white terminals or systems
// that replace emoji with image glyphs.
func toastPrefix(level ToastLevel) string {
	switch level {
	case ToastInfo:
		return S.ToastInfo.Render("●")
	case ToastSuccess:
		return S.ToastSuccess.Render("✓")
	case ToastWarning:
		return S.ToastWarning.Render("▲")
	case ToastError:
		return S.ToastError.Render("✗")
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
		lines = append(lines, prefix+" "+msg)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// RenderOverlay composites toast messages as a floating overlay in the
// bottom-right corner of mainView.  Always computed fresh — toasts change
// every few seconds and the overhead is negligible.
func (tq *ToastQueue) RenderOverlay(mainView string, viewW, viewH int) string {
	toasts := tq.Active()
	if len(toasts) == 0 {
		return mainView
	}

	var lines []string
	showCount := 6
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

	x := viewW - toastW - 2
	if x < 0 {
		x = 0
	}
	y := viewH - toastH - 1
	if y < 0 {
		y = 0
	}

	base := lipgloss.NewLayer(mainView)
	toastLayer := lipgloss.NewLayer(toastContent).
		X(x).
		Y(y).
		Z(1)

	return lipgloss.NewCompositor(base, toastLayer).Render()
}
