package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// The QSO form's clock fields hold the canonical digits ("20261028", "0053") —
// what the save path, the dupe check and every export read — and display them
// with separators wherever the column has room. The shape therefore depends only
// on this frame's width, never on when the clock last ticked or on which screen
// the operator came back from.

func TestDisplayFieldValueClockSeparators(t *testing.T) {
	cases := []struct {
		name  string
		field field
		raw   string
		width int
		want  string
	}{
		{"date roomy", fieldDate, "20261028", 10, "2026-10-28"},
		{"date roomy+", fieldDate, "20261028", 24, "2026-10-28"},
		{"date exact digits", fieldDate, "20261028", 8, "20261028"},
		{"date too narrow", fieldDate, "20261028", 7, "202610…"},
		{"date already typed", fieldDate, "2026-10-28", 10, "2026-10-28"},
		{"time hhmm roomy", fieldTime, "0053", 5, "00:53"},
		{"time hhmm narrow", fieldTime, "0053", 4, "0053"},
		{"time hhmmss roomy", fieldTime, "005300", 8, "00:53:00"},
		{"time hhmmss exact", fieldTime, "005300", 6, "005300"},
		{"time hhmmss negative", fieldTime, "005300", 5, "0053…"},
		{"time typed", fieldTime, "00:53", 5, "00:53"},
		{"other fields pass through", fieldBand, "20m", 10, "20m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := displayFieldValue(tc.field, tc.raw, tc.width); got != tc.want {
				t.Fatalf("displayFieldValue(%v, %q, %d) = %q, want %q", tc.field, tc.raw, tc.width, got, tc.want)
			}
		})
	}
}

// TestClockFieldsRenderAtEveryWidth: the form never cuts the date in half and
// never hides the time separator when there is room for it. The value it holds is
// the same at every width.
func TestClockFieldsRenderAtEveryWidth(t *testing.T) {
	for _, w := range []int{75, 80, 84, 85, 88, 100, 120, 160} {
		m := newLayoutTestModel(t, w, 40)
		m.dateTimeAuto = true
		m.autoUpdateDateTime()
		m.rc.formSig = ""
		frame := stripANSI(m.View().Content)

		date := m.fields[fieldDate].Value()
		if len(date) != 8 {
			t.Fatalf("width %d: date value %q is not the canonical 8 digits", w, date)
		}
		time := m.fields[fieldTime].Value()
		if len(time) != 4 {
			t.Fatalf("width %d: time value %q is not the canonical 4 digits", w, time)
		}

		// The date is whole and separated, whole as digits, or — when the column
		// cannot even hold eight cells — an explicit truncation. What it must
		// never be is a separated prefix of itself ("2026-10-"), which is what
		// the old width threshold produced.
		shown := clockRow(frame, "Date UTC")
		separated := date[:4] + "-" + date[4:6] + "-" + date[6:]
		switch {
		case strings.Contains(shown, separated), strings.Contains(shown, date):
			// fine
		case strings.Contains(shown, "\u2026"):
			// no room for eight cells: truncation is honest, a prefix is not
		default:
			t.Errorf("width %d: the date is a broken prefix: %q", w, shown)
		}
		if strings.Contains(shown, "-") && !strings.Contains(shown, separated) {
			t.Errorf("width %d: the date shows separators but is cut: %q", w, shown)
		}

		// From 88 columns the column holds the ten cells the separated date needs
		// (colW 28 minus the 18-cell label prefix); below that it holds the eight
		// digits, and below 84 it cannot hold even those.
		if w >= 88 && !strings.Contains(shown, separated) {
			t.Errorf("width %d: the date fits but is not separated: %q", w, shown)
		}

		// The time keeps its colon everywhere the app runs: five cells fit from
		// 75 columns up.
		if timeShown := clockRow(frame, "Time UTC"); !strings.Contains(timeShown, time[:2]+":"+time[2:]) {
			t.Errorf("width %d: the time has no separator: %q", w, timeShown)
		}
	}
}

// clockRow returns the form row whose label contains label.
func clockRow(frame, label string) string {
	for _, l := range strings.Split(frame, "\n") {
		if strings.Contains(l, label) {
			return l
		}
	}
	return ""
}

// TestClockValueDoesNotDependOnWidth is the reported symptom: going away and
// coming back (or a resize) must not change the value's shape, only how it is
// rendered.
func TestClockValueDoesNotDependOnWidth(t *testing.T) {
	m := newLayoutTestModel(t, 120, 40)
	m.dateTimeAuto = true
	m.autoUpdateDateTime()

	canonical := func(stage string) {
		t.Helper()
		if d := m.fields[fieldDate].Value(); len(d) != 8 || strings.Contains(d, "-") {
			t.Errorf("%s: date value %q, want the canonical 8 digits", stage, d)
		}
		if v := m.fields[fieldTime].Value(); len(v) != 4 || strings.Contains(v, ":") {
			t.Errorf("%s: time value %q, want the canonical 4 digits", stage, v)
		}
	}

	canonical("120 columns")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	m.autoUpdateDateTime()
	canonical("after a narrow resize")
	m.screen = screenPartner
	m.screen = screenQSO
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.autoUpdateDateTime()
	canonical("after the screen round trip")
}

// TestSavedQSOKeepsTheCanonicalDate proves the display separators never reach the
// log: the operator sees 2026-10-28, the row is stored as 20261028.
func TestSavedQSOKeepsTheCanonicalDate(t *testing.T) {
	m := newLifecycleTestModel(t)
	fillMinimalValidQSO(m)
	m.fields[fieldDate].SetValue("2026-06-14")
	m.fields[fieldTime].SetValue("12:00")

	if cmd := m.saveQSO(); cmd != nil {
		cmd()
	} else {
		t.Fatal("saveQSO returned nil")
	}
	saved := latestQSO(t, m)
	if saved.QSODate != "20260614" {
		t.Errorf("stored date = %q, want 20260614", saved.QSODate)
	}
	if saved.TimeOn != "120000" {
		t.Errorf("stored time = %q, want 120000", saved.TimeOn)
	}
}
