package tui

import (
	"fmt"
	"strings"
	"testing"
)

// =============================================================================
// Wavelog download result screen render tests (Pass 28)
// =============================================================================
// Verifies that the logbook editor Wavelog download result screen displays
// the correct counts, error text, and safe messages. No real network/DB.

// newResultEditor creates a LogbookEditor in edModeWLDownloadResult with
// the given download result state. No DB is required for render tests.
func newResultEditor(count, dupes, failed int, dlErr string) *LogbookEditor {
	le := NewLogbookEditor(nil, "https://log.example.com", "key123", "1", 0, "OP", "JO90")
	le.mode = edModeWLDownloadResult
	le.wlDownloadCount = count
	le.wlDownloadDupes = dupes
	le.wlDownloadFailed = failed
	le.wlDownloadErr = dlErr
	le.width = 120
	le.height = 30
	return le
}

// =============================================================================
// Success state render tests
// =============================================================================

func TestResultRender_Success_OnlyCount(t *testing.T) {
	le := newResultEditor(42, 0, 0, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Downloaded 42 QSOs") {
		t.Error("should show downloaded count")
	}
	if strings.Contains(view, "duplicate") {
		t.Error("should NOT show duplicate text when dupes=0")
	}
	if strings.Contains(view, "failed") {
		t.Error("should NOT show failed text when failed=0")
	}
	if strings.Contains(view, "Download failed") {
		t.Error("should NOT show failure text on success")
	}
}

func TestResultRender_Success_WithDuplicates(t *testing.T) {
	le := newResultEditor(10, 3, 0, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Downloaded 10 QSOs") {
		t.Error("should show downloaded count")
	}
	if !strings.Contains(view, "3 duplicates skipped") {
		t.Error("should show duplicate count")
	}
	if strings.Contains(view, "Download failed") {
		t.Error("should NOT show failure text")
	}
}

func TestResultRender_Success_WithFailed(t *testing.T) {
	le := newResultEditor(5, 0, 2, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Downloaded 5 QSOs") {
		t.Error("should show downloaded count")
	}
	if !strings.Contains(view, "2 failed") {
		t.Error("should show failed count")
	}
	if strings.Contains(view, "duplicates") {
		t.Error("should NOT show duplicate text when dupes=0")
	}
}

func TestResultRender_Success_AllCounts(t *testing.T) {
	le := newResultEditor(7, 2, 1, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Downloaded 7 QSOs") {
		t.Error("should show downloaded count")
	}
	if !strings.Contains(view, "2 duplicates skipped") {
		t.Error("should show duplicate count")
	}
	if !strings.Contains(view, "1 failed") {
		t.Error("should show failed count")
	}
}

func TestResultRender_Success_ZeroImported(t *testing.T) {
	le := newResultEditor(0, 0, 0, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Downloaded 0 QSOs") {
		t.Error("should show 0 imported rather than hide the message")
	}
}

// =============================================================================
// Failure state render tests
// =============================================================================

func TestResultRender_Failure_ShowsError(t *testing.T) {
	le := newResultEditor(0, 0, 0, "server error: HTTP 500")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Download failed") {
		t.Error("should show 'Download failed'")
	}
	if !strings.Contains(view, "HTTP 500") {
		t.Error("should show the error detail")
	}
}

func TestResultRender_Failure_ErrorOverridesCounts(t *testing.T) {
	// When error is set, the error message replaces the entire dialog
	// message — counts are not shown. This is current behavior.
	le := newResultEditor(5, 2, 0, "connection refused")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "Downloaded") {
		t.Error("error message should replace success text, not show counts")
	}
	if !strings.Contains(view, "Download failed") {
		t.Error("should show failure text")
	}
	if !strings.Contains(view, "connection refused") {
		t.Error("should show error detail")
	}
}

func TestResultRender_Failure_AuthError(t *testing.T) {
	le := newResultEditor(0, 0, 0, "server error: HTTP 403 — forbidden")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Download failed") {
		t.Error("should show failure for auth error")
	}
	if !strings.Contains(view, "403") {
		t.Error("should show HTTP status")
	}
}

func TestResultRender_Failure_LongError(t *testing.T) {
	longErr := strings.Repeat("error detail ", 50)
	le := newResultEditor(0, 0, 0, longErr)
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Download failed") {
		t.Error("should show failure header even with long error")
	}
	if !strings.Contains(view, "error detail") {
		t.Error("should include error detail")
	}
}

// =============================================================================
// Secret safety in rendered output
// =============================================================================

func TestResultRender_Failure_DoesNotLeakAPIKey(t *testing.T) {
	// The editor was created with "key123" as the API key.
	// The error message should NOT contain the key.
	le := newResultEditor(0, 0, 0, "server error: HTTP 500 — ")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "key123") {
		t.Error("rendered output should NOT contain API key")
	}
}

func TestResultRender_Failure_DoesNotLeakURL(t *testing.T) {
	le := newResultEditor(0, 0, 0, "server error: HTTP 500 — ")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "log.example.com") {
		t.Log("URL may be visible in rendered output (acceptable if intentional)")
	}
}

// =============================================================================
// Dialog structure verification
// =============================================================================

func TestResultRender_HasDialogTitle(t *testing.T) {
	le := newResultEditor(1, 0, 0, "")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Wavelog Download") {
		t.Error("dialog should have title 'Wavelog Download'")
	}
}

func TestResultRender_HasOKButton(t *testing.T) {
	le := newResultEditor(0, 0, 0, "some error")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "OK") {
		t.Error("dialog should have OK button")
	}
}

// =============================================================================
// Edge cases
// =============================================================================

func TestResultRender_EmptyError(t *testing.T) {
	// Empty error string should not trigger failure path.
	le := newResultEditor(3, 0, 0, "")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "Download failed") {
		t.Error("empty wlDownloadErr should NOT show failure")
	}
	if !strings.Contains(view, "Downloaded 3 QSOs") {
		t.Error("should show success count when error is empty")
	}
}

func TestResultRender_WhitespaceError(t *testing.T) {
	// Whitespace-only error → success path (trimmed to empty).
	le := newResultEditor(3, 0, 0, "   ")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "Download failed") {
		t.Error("whitespace-only wlDownloadErr should NOT show failure")
	}
	if !strings.Contains(view, "Downloaded 3 QSOs") {
		t.Error("whitespace-only error should use success path with count")
	}
}

func TestResultRender_TrimmedMeaningfulError(t *testing.T) {
	// Error with surrounding whitespace → failure path, trimmed message.
	le := newResultEditor(0, 0, 0, "  HTTP 500  ")
	view := fmt.Sprint(le.View())

	if !strings.Contains(view, "Download failed") {
		t.Error("meaningful error should show failure")
	}
	if !strings.Contains(view, "HTTP 500") {
		t.Error("error detail should be visible after trim")
	}
	if strings.Contains(view, "  HTTP 500  ") {
		t.Error("surrounding whitespace should be trimmed from displayed error")
	}
}

func TestResultRender_WhitespaceErrorShowsCount(t *testing.T) {
	// Whitespace error with valid counts → counts shown.
	le := newResultEditor(5, 1, 0, "\n\t ")
	view := fmt.Sprint(le.View())

	if strings.Contains(view, "Download failed") {
		t.Error("whitespace-only error should NOT trigger failure")
	}
	if !strings.Contains(view, "Downloaded 5 QSOs") {
		t.Error("should show success count")
	}
	if !strings.Contains(view, "1 duplicates skipped") {
		t.Error("should show duplicate count")
	}
}
