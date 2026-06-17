package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/qso"
)

func TestRecentQSOsEmpty(t *testing.T) {
	r := NewRecentQSOs(nil)
	r.SetSize(80, 10)

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs View() returned empty string for nil QSOs")
	}
}

func TestRecentQSOsWithData(t *testing.T) {
	qsos := []qso.QSO{
		{
			Call:       "SP9MOA",
			QSODate:    "20260614",
			TimeOn:     "120000",
			Band:       "20m",
			Mode:       "SSB",
			RSTSent:    "59",
			RSTRcvd:    "59",
			Name:       "John",
			QTH:        "Krakow",
			GridSquare: "JO90",
			Country:    "Poland",
		},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(80, 10)

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs View() returned empty")
	}
	if !strings.Contains(view, "SP9MOA") {
		t.Error("RecentQSOs render missing call SP9MOA")
	}
}

func TestRecentQSOsLongValuesNoWrap(t *testing.T) {
	longCall := strings.Repeat("A", 40)
	longName := strings.Repeat("B", 40)
	longComment := strings.Repeat("C", 100)
	qsos := []qso.QSO{
		{
			Call:    longCall,
			QSODate: "20260614",
			TimeOn:  "120000",
			Band:    "20m",
			Mode:    "SSB",
			Name:    longName,
			Comment: longComment,
		},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(80, 10)

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs with long values returned empty")
	}
	// The full long call should NOT appear literally (must be truncated)
	if strings.Contains(view, longCall) {
		t.Error("RecentQSOs did not truncate long call value")
	}
}

func TestRecentQSOsNarrowWidth(t *testing.T) {
	qsos := []qso.QSO{
		{Call: "SP9MOA", Band: "20m", Mode: "SSB"},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(30, 10) // narrow width

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs with narrow width returned empty")
	}
	// Should not panic; just verify non-empty output
}

func TestRecentQSOsTinyWidth(t *testing.T) {
	qsos := []qso.QSO{
		{Call: "SP9MOA", Band: "20m", Mode: "SSB"},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(5, 10) // extremely narrow

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs with tiny width returned empty")
	}
}

func TestRecentQSOsZeroHeight(t *testing.T) {
	qsos := []qso.QSO{
		{Call: "SP9MOA", Band: "20m", Mode: "SSB"},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(80, 0) // zero height

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs with zero height returned empty")
	}
}

func TestRecentQSOsNegativeHeight(t *testing.T) {
	qsos := []qso.QSO{
		{Call: "SP9MOA", Band: "20m", Mode: "SSB"},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(80, -5) // negative height

	view := r.View()
	if view == "" {
		t.Error("RecentQSOs with negative height returned empty")
	}
}

func TestRecentQSOsWidthNotExceeded(t *testing.T) {
	qsos := []qso.QSO{
		{Call: "SP9MOA", Band: "20m", Mode: "SSB"},
	}
	r := NewRecentQSOs(qsos)
	r.SetSize(40, 10)

	view := r.View()
	w := lipgloss.Width(view)
	if w > 45 {
		t.Errorf("RecentQSOs width %d > 45 (should respect narrow constraints)", w)
	}
}
