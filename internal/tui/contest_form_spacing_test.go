package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// contestFormLines renders the create-contest form with the given prefill
// toggles and returns the view content split into lines.
func contestFormLines(t *testing.T, prefillSent, prefillRcvd bool) []string {
	t.Helper()
	cc := newTestContestChooser(t, map[string]config.Contest{})
	cc.startCreate()
	cc.prefillExchange = prefillSent
	cc.prefillExchangeRcvd = prefillRcvd
	return strings.Split(cc.View().Content, "\n")
}

// nextLineWith finds the first line at or after start containing sub.
func nextLineWith(lines []string, start int, sub string) int {
	for i := start; i < len(lines); i++ {
		if strings.Contains(lines[i], sub) {
			return i
		}
	}
	return -1
}

func TestContestFormSpacing_PrefillsOff(t *testing.T) {
	lines := contestFormLines(t, false, false)
	sent := lineIndex(lines, "Prefill Exchange Sent:")
	rcvd := lineIndex(lines, "Prefill Exchange Rcvd:")
	markers := lineIndex(lines, "Exchange markers")
	if sent < 0 || rcvd < 0 || markers < 0 {
		t.Fatalf("rows missing: sent=%d rcvd=%d markers=%d", sent, rcvd, markers)
	}
	if rcvd != sent+1 {
		t.Errorf("unexpected blank line between Prefill checkboxes (sent=%d, rcvd=%d)", sent, rcvd)
	}
	if markers != rcvd+2 {
		t.Errorf("expected exactly one blank line before the marker section (rcvd=%d, markers=%d)", rcvd, markers)
	}
}

func TestContestFormSpacing_PrefillsOn(t *testing.T) {
	lines := contestFormLines(t, true, true)
	sent := lineIndex(lines, "Prefill Exchange Sent:")
	rcvd := lineIndex(lines, "Prefill Exchange Rcvd:")
	if sent < 0 || rcvd < 0 {
		t.Fatalf("checkbox rows missing: sent=%d rcvd=%d", sent, rcvd)
	}
	sentField := nextLineWith(lines, sent+1, "Exchange Sent:")
	rcvdField := nextLineWith(lines, rcvd+1, "Exchange Rcvd:")
	markers := lineIndex(lines, "Exchange markers")
	if sentField < 0 || rcvdField < 0 || markers < 0 {
		t.Fatalf("rows missing: sentField=%d rcvdField=%d markers=%d", sentField, rcvdField, markers)
	}
	if sentField != sent+2 {
		t.Errorf("expected one blank line between checkbox and its field (sent=%d, sentField=%d)", sent, sentField)
	}
	if rcvd != sentField+1 {
		t.Errorf("unexpected blank line between the sent field and the Rcvd checkbox (sentField=%d, rcvd=%d)", sentField, rcvd)
	}
	if rcvdField != rcvd+2 {
		t.Errorf("expected one blank line between checkbox and its field (rcvd=%d, rcvdField=%d)", rcvd, rcvdField)
	}
	if markers != rcvdField+2 {
		t.Errorf("expected exactly one blank line before the marker section (rcvdField=%d, markers=%d)", rcvdField, markers)
	}
}

func TestContestFormSpacing_OnlySentOn(t *testing.T) {
	lines := contestFormLines(t, true, false)
	sent := lineIndex(lines, "Prefill Exchange Sent:")
	rcvd := lineIndex(lines, "Prefill Exchange Rcvd:")
	markers := lineIndex(lines, "Exchange markers")
	if sent < 0 || rcvd < 0 || markers < 0 {
		t.Fatalf("rows missing: sent=%d rcvd=%d markers=%d", sent, rcvd, markers)
	}
	sentField := nextLineWith(lines, sent+1, "Exchange Sent:")
	if sentField != sent+2 {
		t.Errorf("expected one blank line between checkbox and its field (sent=%d, sentField=%d)", sent, sentField)
	}
	if rcvd != sentField+1 {
		t.Errorf("unexpected blank line after the sent field (sentField=%d, rcvd=%d)", sentField, rcvd)
	}
	if markers != rcvd+2 {
		t.Errorf("expected exactly one blank line before the marker section (rcvd=%d, markers=%d)", rcvd, markers)
	}
}
