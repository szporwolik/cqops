package tui

import (
	"strconv"
	"strings"
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

	// Active snapshot cache — avoids alloc+copy on every frame.
	dirty        bool
	cachedActive []Toast

	// Overlay render cache — avoids rebuilding toast overlay every frame.
	cachedView    string
	cachedViewSig string
}

const toastMaxAge = 5 * time.Second
const toastMaxItems = 20
const toastDedupWindow = 2 * time.Second // suppress identical messages within this window

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
	tq.lastMsg = msg
	tq.lastTime = time.Now()
	tq.items = append(tq.items, Toast{
		Level:   level,
		Message: msg,
		Created: tq.lastTime,
	})
	// Cap queue size to prevent unbounded growth on error loops.
	if len(tq.items) > toastMaxItems {
		tq.items = tq.items[len(tq.items)-toastMaxItems:]
	}
	tq.dirty = true
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
	oldLen := len(tq.items)
	n := 0
	for _, t := range tq.items {
		if t.Created.After(cutoff) {
			tq.items[n] = t
			n++
		}
	}
	tq.items = tq.items[:n]
	if n != oldLen {
		tq.dirty = true
	}
}

func (tq *ToastQueue) Active() []Toast {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	if !tq.dirty && tq.cachedActive != nil {
		return tq.cachedActive
	}
	if len(tq.items) == 0 {
		tq.cachedActive = nil
		tq.dirty = false
		return nil
	}
	result := make([]Toast, len(tq.items))
	copy(result, tq.items)
	tq.cachedActive = result
	tq.dirty = false
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

// RenderOverlay caches and returns the mainView with toasts composited as a
// floating overlay in the bottom-right corner. Cache is for toast content only;
// the final compositing with mainView is always done fresh (mainView changes on
// screen switch, resize, or clock tick).
func (tq *ToastQueue) RenderOverlay(mainView string, viewW, viewH int) string {
	toasts := tq.Active()
	if len(toasts) == 0 {
		return mainView
	}

	// Build a signature from view dims + toast count + last toast message.
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(viewW))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(viewH))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(len(toasts)))
	sb.WriteByte('|')
	sb.WriteString(toasts[len(toasts)-1].Message)
	sig := sb.String()

	// Cached toast content (expensive to rebuild: lines + JoinVertical + layout).
	var toastContent string
	var toastW, toastH int
	if tq.cachedViewSig == sig && tq.cachedView != "" {
		toastContent = tq.cachedView
		toastW = lipgloss.Width(toastContent)
		toastH = lipgloss.Height(toastContent)
	} else {
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
		toastContent = lipgloss.JoinVertical(lipgloss.Right, lines...)
		toastW = lipgloss.Width(toastContent)
		toastH = lipgloss.Height(toastContent)
		tq.cachedView = toastContent
		tq.cachedViewSig = sig
	}

	// Position: bottom-right corner with 2-cell margin from edges.
	// Always re-composite with the current mainView (never cached).
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
