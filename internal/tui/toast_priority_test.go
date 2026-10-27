package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

// The toast overlay paces notices so a redraw storm cannot stack them, but that
// pacing used to apply to errors as well: an informational toast that had just
// appeared swallowed "QSO save failed" for the next half second. Errors outrank
// notices — they are spaced among themselves instead.

// toastMessages returns the queued messages in order.
func toastMessages(tq *ToastQueue) []string {
	var out []string
	for _, t := range tq.Active() {
		out = append(out, t.Message)
	}
	return out
}

// TestErrorSurvivesAPrecedingInfo is the reported sequence: a status notice, then
// a failure.
func TestErrorSurvivesAPrecedingInfo(t *testing.T) {
	tq := NewToastQueue()
	tq.Info("connected")
	tq.Error("QSO save failed")

	got := tq.Active()
	if len(got) != 2 {
		t.Fatalf("queued toasts = %v, want both the notice and the error", toastMessages(tq))
	}
	if got[1].Level != ToastError || got[1].Message != "QSO save failed" {
		t.Errorf("the error is not queued last (so it is not the one on screen): %v", toastMessages(tq))
	}
}

// TestErrorSurvivesAPrecedingSuccessAndWarning covers the other notice levels:
// none of them may shadow a failure.
func TestErrorSurvivesAPrecedingSuccessAndWarning(t *testing.T) {
	for _, tc := range []struct {
		name string
		push func(tq *ToastQueue)
	}{
		{"success", func(tq *ToastQueue) { tq.Success("logged") }},
		{"warning", func(tq *ToastQueue) { tq.Warn("Wavelog offline") }},
		{"info", func(tq *ToastQueue) { tq.Info("connected") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tq := NewToastQueue()
			tc.push(tq)
			tq.Error("QSO save failed")
			if got := toastMessages(tq); len(got) != 2 || got[1] != "QSO save failed" {
				t.Errorf("a %s toast swallowed the error: %v", tc.name, got)
			}
		})
	}
}

// TestRepeatedErrorsDoNotFloodTheOverlay: the error exemption must not turn an
// error loop into a screenful of toasts.
func TestRepeatedErrorsDoNotFloodTheOverlay(t *testing.T) {
	tq := NewToastQueue()

	// The same failure repeating collapses into one toast.
	for i := 0; i < 50; i++ {
		tq.Error("Wavelog: upload failed")
	}
	if got := len(tq.Active()); got != 1 {
		t.Fatalf("%d toasts from 50 identical errors, want 1: %v", got, toastMessages(tq))
	}

	// Distinct failures are spaced instead of stacked.
	for i := 0; i < 50; i++ {
		tq.Error(fmt.Sprintf("upload failed (%d)", i))
	}
	if got := len(tq.Active()); got > 2 {
		t.Errorf("%d toasts from a burst of distinct errors, want the loop throttled: %v", got, toastMessages(tq))
	}
}

// TestErrorsAreSpacedButNotStarved: after the error spacing has passed, the next
// failure is accepted again — the exemption is a priority, not a mute.
func TestErrorsAreSpacedButNotStarved(t *testing.T) {
	original := toastErrorMinSpacing
	toastErrorMinSpacing = 20 * time.Millisecond
	t.Cleanup(func() { toastErrorMinSpacing = original })

	tq := NewToastQueue()
	tq.Error("first failure")
	tq.Error("second failure")
	if got := len(tq.Active()); got != 1 {
		t.Fatalf("%d toasts within the spacing window, want 1", got)
	}

	time.Sleep(30 * time.Millisecond)
	tq.Error("third failure")
	got := toastMessages(tq)
	if len(got) != 2 || got[1] != "third failure" {
		t.Errorf("the failure after the spacing window was dropped: %v", got)
	}
}

// TestNoticesAreStillPaced: the notice throttle keeps working, so the exemption
// did not widen the doorway for informational spam.
func TestNoticesAreStillPaced(t *testing.T) {
	original := toastMinInterval
	toastMinInterval = time.Hour
	t.Cleanup(func() { toastMinInterval = original })

	tq := NewToastQueue()
	tq.Info("connected")
	tq.Info("internet reachable")
	if got := len(tq.Active()); got != 1 {
		t.Errorf("%d notices inside the pacing window, want 1: %v", got, toastMessages(tq))
	}
	// ...and an error still gets through.
	tq.Error("QSO save failed")
	if got := toastMessages(tq); len(got) != 2 || got[1] != "QSO save failed" {
		t.Errorf("the error was paced like a notice: %v", got)
	}
}

// TestToastOverlayGeometryUnchanged: the fix is a queue policy, not a renderer
// change — an error lands on the same row, with the same bounds, as a notice.
func TestToastOverlayGeometryUnchanged(t *testing.T) {
	const w, h = 70, 20
	frame := strings.Repeat(strings.Repeat("-", w)+"\n", h-1) + strings.Repeat("-", w)

	place := func(level ToastLevel) (line, col, lines, maxW int) {
		tq := NewToastQueue()
		tq.Push(level, "message text")
		out := stripANSI(tq.RenderOverlay(frame, w, h))
		for i, l := range strings.Split(out, "\n") {
			if c := strings.Index(l, "message text"); c >= 0 {
				line, col = i, c
			}
			if lw := lipgloss.Width(l); lw > maxW {
				maxW = lw
			}
		}
		return line, col, len(strings.Split(out, "\n")), maxW
	}

	noticeLine, noticeCol, noticeLines, noticeMaxW := place(ToastInfo)
	errorLine, errorCol, errorLines, errorMaxW := place(ToastError)

	if errorLine != noticeLine || errorCol != noticeCol {
		t.Errorf("the error toast moved: notice at %d,%d, error at %d,%d",
			noticeLine, noticeCol, errorLine, errorCol)
	}
	if errorLines != noticeLines || errorMaxW != noticeMaxW {
		t.Errorf("the overlay bounds changed: notice %d lines/%d wide, error %d lines/%d wide",
			noticeLines, noticeMaxW, errorLines, errorMaxW)
	}
	if errorLines != h || errorMaxW > w {
		t.Errorf("the overlay does not fit the frame: %d lines, %d wide (frame %dx%d)", errorLines, errorMaxW, w, h)
	}
	// Unchanged placement: one row above the bottom edge (y = height - 1 - 1).
	if errorLine != h-2 {
		t.Errorf("a single toast sits on row %d, want row %d (one above the bottom edge)", errorLine, h-2)
	}
}
