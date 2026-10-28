package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

// =============================================================================
// Layout stability gates
// =============================================================================
//
// The frames below are captured from the real View(), never from a stub, so a
// change to any zone breaks them. What is compared is the STRUCTURE of a frame —
// where each panel starts, how many rows it occupies, where the focus is and
// where the cursor sits — not its content: a lookup that returns, an error, a
// logbook switch, a dialog or a resize may change what is written inside the
// zones, but it must not move them.
//
// Width is now part of the contract: every zone fits the terminal and the frame
// itself must not exceed TermW. lipgloss pads every rendered line to the widest
// one, so a single zone laid out one cell too wide made the whole frame wider
// than the terminal — that used to happen here (the recent-QSOs table rendered
// 84 cells in a 73-cell slot, the tab bar two cells too wide) and is pinned by
// assertFrameSane and by table_width_test.go.

// panelNames are the zones a QSO screen frame is made of, top to bottom.
var panelNames = []string{"status", "tabs", "profile", "form", "table", "help"}

// layoutFrame is the measurable structure of one rendered frame.
type layoutFrame struct {
	TermW int // size the model was rendered at
	TermH int

	Lines   int // rendered lines (must equal TermH)
	FrameW  int // widest rendered line
	StatusW int // width of the status bar (the only zone laid out to TermW)
	FormW   int // width of the form box border

	Panels map[string]int // panel name -> first line index, -1 when absent
	Cols   map[string]int // panel name -> left column (panels are left-aligned)
	Rows   map[string]int // zone -> height in rows

	Focus      int    // focused field index
	FocusLabel string // label the frame marks with "> "
	FocusLine  int    // line holding that marker
	FocusCol   int    // column of the marker
	Cursor     int    // textinput cursor position inside the focused field
	ListCursor int    // cursor of the list on screen (-1 when the screen has none)
}

// newLayoutTestModel returns a QSO screen with controlled data: three contacts,
// a fixed clock (no auto date/time) and no integrations, so a frame depends only
// on the layout.
func newLayoutTestModel(t *testing.T, w, h int) *Model {
	t.Helper()
	m := newLifecycleTestModel(t)
	m.dateTimeAuto = false
	m.fields[fieldDate].SetValue("2026-10-27")
	m.fields[fieldTime].SetValue("12:00")
	m.recentQSOs.SetQSOS([]qso.QSO{
		{ID: 1, Call: "SP9MOA", Band: "20m", Mode: "SSB", QSODate: "2026-10-27", TimeOn: "115500", RSTSent: "59", RSTRcvd: "59"},
		{ID: 2, Call: "DL1AA", Band: "40m", Mode: "CW", QSODate: "2026-10-27", TimeOn: "114500", RSTSent: "599", RSTRcvd: "599"},
		{ID: 3, Call: "K1ABC", Band: "15m", Mode: "FT8", QSODate: "2026-10-27", TimeOn: "113000", RSTSent: "-10", RSTRcvd: "-12"},
	})
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

// resize re-renders the model at a new terminal size the way the runtime does.
func resize(m *Model, w, h int) { m.Update(tea.WindowSizeMsg{Width: w, Height: h}) }

// captureLayout renders the model and measures the frame.
func captureLayout(t *testing.T, m *Model) layoutFrame {
	t.Helper()
	f := layoutFrame{
		TermW:      m.width,
		TermH:      m.height,
		StatusW:    lipgloss.Width(stripANSI(m.renderStatusBar())),
		Panels:     make(map[string]int, len(panelNames)),
		Cols:       make(map[string]int, len(panelNames)),
		Rows:       make(map[string]int, len(panelNames)),
		Focus:      int(m.focus),
		Cursor:     m.fields[m.focus].Position(),
		ListCursor: -1,
	}
	if m.screen == screenMainMenu && m.ui.mainMenu != nil {
		f.ListCursor = m.ui.mainMenu.cursor
	}
	for _, p := range panelNames {
		f.Panels[p] = -1
		f.Cols[p] = -1
	}
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	f.Lines = len(lines)
	for _, l := range lines {
		if w := lipgloss.Width(l); w > f.FrameW {
			f.FrameW = w
		}
	}

	// The status bar is the top row; the "terminal too small" warning is not one.
	if len(lines) > 0 && strings.Contains(lines[0], " CQOps ") {
		f.Panels["status"] = 0
		f.Cols["status"] = 0
	}
	f.Panels["tabs"] = firstLine(lines, 0, func(l string) bool { return strings.Contains(l, "F1 QSO") })
	if f.Panels["tabs"] >= 0 {
		f.Cols["tabs"] = leadColumn(lines[f.Panels["tabs"]])
	}
	// The form box is the first continuous border, not the tab bar's segments.
	f.Panels["form"] = firstLine(lines, 0, func(l string) bool {
		t := strings.TrimSpace(l)
		return strings.HasPrefix(t, "\u256d") && !strings.Contains(l, "\u256e\u256d")
	})
	if form := f.Panels["form"]; form >= 0 {
		f.Cols["form"] = strings.Index(lines[form], "\u256d")
		f.FormW = lipgloss.Width(strings.TrimRight(lines[form], " "))
		if form > 0 && strings.TrimSpace(lines[form-1]) != "" {
			f.Panels["profile"] = form - 1
			f.Cols["profile"] = leadColumn(lines[form-1])
		}
	}
	formBottom := -1
	if form := f.Panels["form"]; form >= 0 {
		formBottom = firstLine(lines, form, func(l string) bool {
			return strings.HasPrefix(strings.TrimSpace(l), "\u2570")
		})
	}
	// The recent-QSOs table is the LAST header carrying the RST pair: the
	// per-callsign table above it has the same header when it is shown.
	f.Panels["table"] = -1
	for i, l := range lines {
		if strings.Contains(l, "RSTs") && strings.Contains(l, "RSTr") {
			f.Panels["table"] = i
		}
	}
	f.Panels["help"] = lastLine(lines, func(l string) bool { return strings.Contains(l, "Help") })
	if f.Panels["help"] >= 0 {
		f.Cols["help"] = strings.Index(lines[f.Panels["help"]], "Help")
	}
	if f.Panels["table"] >= 0 {
		f.Cols["table"] = leadColumn(lines[f.Panels["table"]])
	}

	if f.Panels["form"] >= 0 && formBottom >= f.Panels["form"] {
		f.Rows["form"] = formBottom - f.Panels["form"] + 1
	}
	if f.Panels["table"] >= 0 && f.Panels["help"] > f.Panels["table"] {
		f.Rows["table"] = f.Panels["help"] - f.Panels["table"]
	}
	if f.Panels["tabs"] >= 0 && f.Panels["help"] > f.Panels["tabs"] {
		f.Rows["content"] = f.Panels["help"] - f.Panels["tabs"] - 1
	}

	// Focus marker: the form renders the focused field with a "> " prefix.
	f.FocusLine, f.FocusCol = -1, -1
	if form := f.Panels["form"]; form >= 0 && formBottom >= form {
		for i := form; i <= formBottom && i < len(lines); i++ {
			if col := strings.Index(lines[i], "> "); col >= 0 {
				f.FocusLine = i
				f.FocusCol = col
				rest := lines[i][col+2:]
				f.FocusLabel = strings.TrimSpace(strings.SplitN(strings.ReplaceAll(rest, "  ", "\x00"), "\x00", 2)[0])
				break
			}
		}
	}
	return f
}

// leadColumn is the column of the first non-space cell of a line.
func leadColumn(l string) int {
	return len(l) - len(strings.TrimLeft(l, " "))
}

func firstLine(lines []string, from int, pred func(string) bool) int {
	for i := from; i < len(lines); i++ {
		if pred(lines[i]) {
			return i
		}
	}
	return -1
}

func lastLine(lines []string, pred func(string) bool) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if pred(lines[i]) {
			return i
		}
	}
	return -1
}

// String renders a frame for a failure message.
func (f layoutFrame) String() string {
	panels := make([]string, 0, len(panelNames))
	for _, p := range panelNames {
		panels = append(panels, fmt.Sprintf("%s=%d@%d", p, f.Panels[p], f.Cols[p]))
	}
	rows := make([]string, 0, len(f.Rows))
	for k := range f.Rows {
		rows = append(rows, fmt.Sprintf("%s:%d", k, f.Rows[k]))
	}
	sort.Strings(rows)
	return fmt.Sprintf("%dx%d lines=%d frameW=%d statusW=%d [%s] rows{%s} focus=%d/%q@%d,%d cursor=%d listCursor=%d",
		f.TermW, f.TermH, f.Lines, f.FrameW, f.StatusW, strings.Join(panels, " "),
		strings.Join(rows, " "), f.Focus, f.FocusLabel, f.FocusLine, f.FocusCol, f.Cursor, f.ListCursor)
}

// assertFrameSane checks the invariants every frame must keep on its own.
func assertFrameSane(t *testing.T, stage string, f layoutFrame) {
	t.Helper()
	if f.Lines != f.TermH {
		t.Errorf("%s: frame is %d lines tall, want %d (the help bar must sit on the last row)", stage, f.Lines, f.TermH)
	}
	// Every zone is laid out to the terminal, and lipgloss pads every line to the
	// widest one: a frame that is one cell wider than the terminal wraps or clips
	// in a real terminal, so this is a hard limit, not a preference.
	if f.FrameW > f.TermW {
		t.Errorf("%s: frame is %d cells wide in a %d-cell terminal", stage, f.FrameW, f.TermW)
	}
	// The status bar is laid out to exactly the terminal width.
	if f.Panels["status"] >= 0 && f.StatusW != f.TermW {
		t.Errorf("%s: the status bar is %d cells wide in a %d-cell terminal", stage, f.StatusW, f.TermW)
	}
	// The profile line is right-aligned on purpose; everything else in the
	// left column starts at the edge.
	for _, p := range []string{"status", "tabs", "form"} {
		if f.Panels[p] >= 0 && f.Cols[p] != 0 {
			t.Errorf("%s: panel %q is not left-aligned (column %d)", stage, p, f.Cols[p])
		}
	}
	prev := -1
	for _, p := range panelNames {
		idx := f.Panels[p]
		if idx < 0 {
			continue
		}
		if idx <= prev {
			t.Errorf("%s: %s at line %d is not below the panel above it (line %d): %s", stage, p, idx, prev, f)
		}
		prev = idx
	}
	if f.Panels["help"] != f.Lines-1 {
		t.Errorf("%s: the help bar is on line %d of %d: %s", stage, f.Panels["help"], f.Lines-1, f)
	}
}

// assertSameZones compares the listed panels and their row counts between two
// frames.
func assertSameZones(t *testing.T, stage string, want, got layoutFrame, zones ...string) {
	t.Helper()
	for _, z := range zones {
		if want.Panels[z] != got.Panels[z] {
			t.Errorf("%s: panel %q moved from line %d to %d", stage, z, want.Panels[z], got.Panels[z])
		}
		// The profile line is right-aligned, so its column follows the frame
		// width instead of the left edge; only its row is meaningful.
		if z != "profile" && want.Cols[z] != got.Cols[z] {
			t.Errorf("%s: panel %q moved from column %d to %d", stage, z, want.Cols[z], got.Cols[z])
		}
	}
	for _, z := range zones {
		if w, ok := want.Rows[z]; ok {
			if g := got.Rows[z]; g != w {
				t.Errorf("%s: zone %q is %d rows high, want %d", stage, z, g, w)
			}
		}
	}
}

// clockPattern matches the status bar's local+UTC clock, the only part of a
// frame that changes on its own.
var clockPattern = regexp.MustCompile(`\d{2}:\d{2}L \d{4}Z`)

// normalizedFrame returns the rendered frame with the clock neutralised, so two
// frames can be compared for exact identity.
func normalizedFrame(m *Model) string {
	var out []string
	for _, l := range strings.Split(stripANSI(m.View().Content), "\n") {
		out = append(out, strings.TrimRight(clockPattern.ReplaceAllString(l, "<clock>"), " "))
	}
	return strings.Join(out, "\n")
}

// expireToasts back-dates queued toasts so Expire() removes them without the
// test sleeping for the five-second toast lifetime.
func expireToasts(m *Model) {
	m.toasts.mu.Lock()
	for i := range m.toasts.items {
		m.toasts.items[i].Created = time.Now().Add(-time.Minute)
	}
	m.toasts.mu.Unlock()
	m.toasts.Expire()
}

// =============================================================================
// Sequences
// =============================================================================

// TestLayoutStableAcrossLookupStates walks loading → result → error on the QSO
// screen. The zones above the recent table must not move: a call badge, a panel
// of past contacts or an error toast may change what is inside them, never
// where they are.
func TestLayoutStableAcrossLookupStates(t *testing.T) {
	m := newLayoutTestModel(t, 120, 40)
	idle := captureLayout(t, m)
	assertFrameSane(t, "idle", idle)
	idleText := normalizedFrame(m)
	if idle.Panels["form"] < 0 || idle.Panels["table"] < 0 {
		t.Fatalf("idle frame is missing a panel: %s", idle)
	}

	// Loading: a call is typed and its lookups are on the wire.
	m.fields[fieldCall].SetValue("DL1AA")
	m.lookup.wlInFlight = true
	m.fields[fieldCall].Focus()
	loading := captureLayout(t, m)
	assertFrameSane(t, "loading", loading)
	assertSameZones(t, "loading", idle, loading, "status", "tabs", "profile", "form", "table", "help")
	if loading.FocusLabel != "Call" {
		t.Errorf("loading: focus marker is on %q, want Call", loading.FocusLabel)
	}

	// Result: the call's earlier contacts arrive as a panel of their own above
	// the recent table. That is the only thing allowed to move.
	m.lookup.wlInFlight = false
	m.callRecentQSOs.SetFilterCall("DL1AA", []qso.QSO{
		{ID: 10, Call: "DL1AA", Band: "40m", Mode: "CW", QSODate: "2026-05-01", TimeOn: "101500", RSTSent: "599", RSTRcvd: "599"},
		{ID: 11, Call: "DL1AA", Band: "20m", Mode: "SSB", QSODate: "2026-04-01", TimeOn: "181500", RSTSent: "59", RSTRcvd: "59"},
	})
	m.needRefresh = true
	result := captureLayout(t, m)
	assertFrameSane(t, "result", result)
	assertSameZones(t, "result", idle, result, "status", "tabs", "profile", "form", "help")
	if result.Panels["table"] <= idle.Panels["table"] {
		t.Errorf("result: the per-callsign panel did not push the recent table down (%d -> %d)",
			idle.Panels["table"], result.Panels["table"])
	}
	if result.Panels["form"] != idle.Panels["form"] {
		t.Errorf("result: the form moved from line %d to %d when a call panel appeared",
			idle.Panels["form"], result.Panels["form"])
	}

	// Error: the lookup fails, a toast reports it, the data disappears again.
	m.callRecentQSOs.ClearFilter()
	m.toasts.Error("Wavelog: lookup failed — timeout")
	failed := captureLayout(t, m)
	assertFrameSane(t, "error", failed)
	assertSameZones(t, "error", idle, failed, "status", "tabs", "profile", "form", "table", "help")
	if failed.Lines != idle.Lines {
		t.Errorf("error: the toast changed the frame height (%d -> %d)", idle.Lines, failed.Lines)
	}

	// Once the toast expires and the call is cleared the frame is exactly the
	// one from before the sequence started.
	expireToasts(m)
	m.fields[fieldCall].SetValue("")
	if got := normalizedFrame(m); got != idleText {
		t.Errorf("error: the frame did not return to its idle shape after the toast expired\ngot:\n%s\nwant:\n%s", got, idleText)
	}
}

// TestLayoutStableAcrossLogbookSwitch: switching logbooks replaces the data set,
// not the layout. Panels, focus, the textinput cursor and the table cursor all
// have to survive it.
func TestLayoutStableAcrossLogbookSwitch(t *testing.T) {
	m := newLayoutTestModel(t, 110, 34)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldCall].Focus()
	m.fields[fieldCall].SetCursor(3)

	before := captureLayout(t, m)
	assertFrameSane(t, "before", before)
	if before.Cursor != 3 {
		t.Fatalf("setup: cursor=%d, want 3", before.Cursor)
	}

	// A second logbook on its own database file, inside the test directory.
	m.App.Config.Logbooks["b"] = config.Logbook{
		Station:      config.Station{Callsign: "SP9MOA/P", Grid: "JO90", RigName: "default"},
		Wavelog:      &config.WavelogConfig{},
		DatabasePath: t.TempDir() + "/second.db",
	}
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("SwitchLogbook: %v", err)
	}
	m.handleLogbookSwitched()
	m.recentQSOs.SetQSOS(nil) // the new logbook has its own contacts

	after := captureLayout(t, m)
	assertFrameSane(t, "after switch", after)
	assertSameZones(t, "after switch", before, after, "status", "tabs", "profile", "form", "table", "help")
	if after.Focus != before.Focus {
		t.Errorf("after switch: focus moved from field %d to %d", before.Focus, after.Focus)
	}
	if after.FocusLine != before.FocusLine || after.FocusCol != before.FocusCol {
		t.Errorf("after switch: the focus marker moved from %d,%d to %d,%d",
			before.FocusLine, before.FocusCol, after.FocusLine, after.FocusCol)
	}
	if after.Cursor != before.Cursor {
		t.Errorf("after switch: the textinput cursor moved from %d to %d", before.Cursor, after.Cursor)
	}
	if after.FocusLabel != before.FocusLabel {
		t.Errorf("after switch: the focus marker labels %q, want %q", after.FocusLabel, before.FocusLabel)
	}
	if after.Lines != before.Lines || after.FrameW != before.FrameW {
		t.Errorf("after switch: frame is %d lines / %d wide, want %d / %d",
			after.Lines, after.FrameW, before.Lines, before.FrameW)
	}
}

// TestLayoutStableAcrossDialogOpenClose: a modal is an overlay. It must not
// resize the frame, it must be centred, and closing it must restore exactly the
// frame the operator was looking at.
func TestLayoutStableAcrossDialogOpenClose(t *testing.T) {
	m := newLayoutTestModel(t, 100, 30)
	idle := captureLayout(t, m)
	assertFrameSane(t, "idle", idle)
	before := normalizedFrame(m)

	dlg := NewDialog("Quit CQOps?", "Log 3 contacts before leaving.",
		Option{Label: "Yes", Value: "quit"}, Option{Label: "No", Value: ""})
	m.confirm = &dlg
	open := captureLayout(t, m)
	if open.Lines != idle.Lines {
		t.Errorf("dialog: the overlay changed the frame height (%d -> %d)", idle.Lines, open.Lines)
	}
	// The dialog is centred: its box starts around the middle of the frame.
	lines := strings.Split(stripANSI(m.View().Content), "\n")
	top := firstLine(lines, 0, func(l string) bool { return strings.Contains(l, "Quit CQOps?") })
	if top < 0 {
		t.Fatalf("dialog: title is not on screen:\n%s", strings.Join(lines, "\n"))
	}
	boxTop := firstLine(lines, 0, func(l string) bool {
		return strings.HasPrefix(strings.TrimSpace(l), "\u256d")
	})
	if boxTop < 0 || top < boxTop {
		t.Fatalf("dialog: no box around the dialog title")
	}
	runes := []rune(strings.TrimRight(lines[boxTop], " "))
	left, right := -1, -1
	for i, r := range runes {
		if r == '\u256d' && left < 0 {
			left = i
		}
		if r == '\u256e' {
			right = i
		}
	}
	if left < 0 || right <= left {
		t.Fatalf("dialog: the box border is not readable: %q", lines[boxTop])
	}
	boxW := right - left + 1
	if want := (open.TermW - boxW) / 2; left-want > 2 || want-left > 2 {
		t.Errorf("dialog: box left edge at %d is not centred (want ~%d for a %d-cell box in %d cells)",
			left, want, boxW, open.TermW)
	}
	if vert := boxTop - open.TermH/2; vert > 6 || vert < -6 {
		t.Errorf("dialog: box top at %d is not vertically centred in %d rows", boxTop, open.TermH)
	}

	// Closing restores the frame exactly.
	m.confirm = nil
	if after := normalizedFrame(m); after != before {
		t.Errorf("dialog: closing did not restore the original frame\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

// TestLayoutStableAcrossResizeRoundTrip: resizing and coming back must produce
// the frame that was there before, including focus and cursors.
func TestLayoutStableAcrossResizeRoundTrip(t *testing.T) {
	m := newLayoutTestModel(t, 120, 40)
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldCall].Focus()
	m.fields[fieldCall].SetCursor(2)

	first := captureLayout(t, m)
	assertFrameSane(t, "120x40", first)
	original := normalizedFrame(m)

	for _, size := range [][2]int{{100, 30}, {80, 24}, {160, 50}, {75, 24}, {74, 40}, {120, 40}} {
		resize(m, size[0], size[1])
		f := captureLayout(t, m)
		switch {
		case size[0] < 75 || size[1] < 24:
			// Below the minimum the app says so instead of drawing a broken
			// layout — no panels at all.
			if f.Panels["form"] != -1 || f.Panels["table"] != -1 {
				t.Errorf("%dx%d: panels are drawn below the minimum size: %s", size[0], size[1], f)
			}
			if !strings.Contains(normalizedFrame(m), "Terminal too small") {
				t.Errorf("%dx%d: no size warning", size[0], size[1])
			}
		default:
			assertFrameSane(t, fmt.Sprintf("%dx%d", size[0], size[1]), f)
			if f.Focus != first.Focus || f.Cursor != first.Cursor {
				t.Errorf("%dx%d: focus/cursor changed across the resize: focus %d->%d cursor %d->%d",
					size[0], size[1], first.Focus, f.Focus, first.Cursor, f.Cursor)
			}
		}
	}

	if after := normalizedFrame(m); after != original {
		t.Errorf("the frame did not return to its original shape after a resize round trip\nbefore:\n%s\nafter:\n%s",
			original, after)
	}
}

// TestLayoutAroundMinimumSizeThreshold checks the sizes immediately around the
// 75x24 gate: just below it the app refuses to draw, just above it every zone
// fits exactly.
func TestLayoutAroundMinimumSizeThreshold(t *testing.T) {
	for _, tc := range []struct {
		w, h int
		full bool
	}{
		{74, 24, false},
		{75, 23, false},
		{75, 24, true},
		{76, 25, true},
		{80, 24, true},
	} {
		m := newLayoutTestModel(t, tc.w, tc.h)
		f := captureLayout(t, m)
		if !tc.full {
			if f.Panels["form"] != -1 || f.Panels["status"] != -1 {
				t.Errorf("%dx%d: a full layout was drawn below the minimum: %s", tc.w, tc.h, f)
			}
			if !strings.Contains(stripANSI(m.View().Content), "Terminal too small") {
				t.Errorf("%dx%d: the size warning is missing", tc.w, tc.h)
			}
			continue
		}
		assertFrameSane(t, fmt.Sprintf("%dx%d", tc.w, tc.h), f)
		// At the minimum the form must fit without being clipped and the table
		// must still be there.
		if f.Rows["form"] < 5 {
			t.Errorf("%dx%d: the form is only %d rows high", tc.w, tc.h, f.Rows["form"])
		}
		if f.Rows["table"] < 1 {
			t.Errorf("%dx%d: the recent-QSO table is gone: %s", tc.w, tc.h, f)
		}
	}
}

// TestLayoutRefusesATerminalItCannotFit: a terminal below the minimum gets the
// refusal and nothing else — including the sizes too small to be promoted.
//
// MeasureLayout used to replace anything under 40x10 with 80x24, so the check
// in View saw 80x24 and drew the whole screen: a 39-column terminal got a
// 79-cell frame, and an 8-row one got a 24-line frame. Both now refuse, and the
// refusal never exceeds the terminal it is drawn in.
func TestLayoutRefusesATerminalItCannotFit(t *testing.T) {
	for _, size := range [][2]int{{39, 24}, {20, 10}, {30, 40}, {76, 9}, {100, 8}} {
		m := newLayoutTestModel(t, size[0], size[1])
		f := captureLayout(t, m)
		for _, p := range []string{"form", "status", "tabs", "table", "help"} {
			if f.Panels[p] != -1 {
				t.Errorf("%dx%d: panel %q is drawn below the minimum size: %s", size[0], size[1], p, f)
			}
		}
		if f.Lines != 5 {
			t.Errorf("%dx%d: the refusal is %d lines, want 5: %s", size[0], size[1], f.Lines, f)
		}
		if f.FrameW > size[0] {
			t.Errorf("%dx%d: the refusal is %d cells wide and does not fit the terminal", size[0], size[1], f.FrameW)
		}
	}
}

// TestLayoutRefusalNamesTheRealSize: the operator is told the size the terminal
// actually has, not the one the layout would have used.
func TestLayoutRefusalNamesTheRealSize(t *testing.T) {
	m := newLayoutTestModel(t, 39, 24)
	out := stripANSI(m.View().Content)
	if !strings.Contains(out, "39x24 (min 75x24)") {
		t.Errorf("the refusal does not name the real size and the minimum:\n%s", out)
	}
	if !strings.Contains(out, "Terminal too small") {
		t.Errorf("the refusal does not say what is wrong:\n%s", out)
	}

	// The wizard has its own guard and must say the same thing.
	w := newTestWizard(t, "SP9MOA", "JO90AA")
	w.width, w.height = 39, 24
	if got := stripANSI(w.View().Content); !strings.Contains(got, "39x24 (min 75x24)") {
		t.Errorf("the wizard's refusal does not name the real terminal size:\n%s", got)
	}
}

// TestLayoutAroundFormWidthCap: the form box stops growing at partnerMapMaxW-4
// of content, so the chrome stays left-aligned and stable beyond it. Sizes just
// below, at and above the cap must all keep the same zone structure.
func TestLayoutAroundFormWidthCap(t *testing.T) {
	var widths []int
	for _, w := range []int{partnerMapMaxW - 2, partnerMapMaxW - 1, partnerMapMaxW, partnerMapMaxW + 2} {
		if w >= 75 {
			widths = append(widths, w)
		}
	}
	base := captureLayout(t, newLayoutTestModel(t, widths[0], 40))
	capped := 0
	for _, w := range widths {
		f := captureLayout(t, newLayoutTestModel(t, w, 40))
		assertFrameSane(t, fmt.Sprintf("width %d", w), f)
		assertSameZones(t, fmt.Sprintf("width %d", w), base, f, "status", "tabs", "profile", "form", "table", "help")
		if f.Panels["form"] != base.Panels["form"] || f.Rows["form"] != base.Rows["form"] {
			t.Errorf("width %d: the form box changed shape at the width cap: %s", w, f)
		}
		if w >= partnerMapMaxW {
			if capped == 0 {
				capped = f.FormW
			} else if f.FormW != capped {
				t.Errorf("width %d: the form box is %d cells wide, want %d (capped)", w, f.FormW, capped)
			}
		}
	}
	if capped == 0 || capped > partnerMapMaxW+2 {
		t.Errorf("the form box grew to %d cells past the %d-cell cap", capped, partnerMapMaxW)
	}
}

// TestLayoutClockFieldsShowSeparators: the date and time fields display their
// separators wherever the column has room for them, and hold the same canonical
// digits at every width. The layout must not move either way — the old gate
// pinned a format threshold at 85 columns, which the form never actually had
// room for (the value was cut to "2026-10-").
func TestLayoutClockFieldsShowSeparators(t *testing.T) {
	build := func(w int) *Model {
		m := newLayoutTestModel(t, w, 40)
		m.dateTimeAuto = true
		m.autoUpdateDateTime()
		m.rc.formSig = "" // the form cache skips the clock while auto-updating
		return m
	}
	narrow, roomy := build(84), build(100)

	fn := captureLayout(t, narrow)
	fr := captureLayout(t, roomy)
	assertFrameSane(t, "width 84", fn)
	assertFrameSane(t, "width 100", fr)

	// The value never depends on the width — only its rendering does.
	raw := narrow.fields[fieldDate].Value()
	if len(raw) != 8 {
		t.Fatalf("date value %q is not the canonical 8 digits", raw)
	}
	if got := roomy.fields[fieldDate].Value(); got != raw {
		t.Errorf("the date value changed with the terminal width: %q vs %q", got, raw)
	}

	// Narrow column: the whole date as digits (never cut in half), and the time
	// keeps its colon — 5 cells fit everywhere.
	narrowFrame := stripANSI(narrow.View().Content)
	if !strings.Contains(narrowFrame, raw) {
		t.Errorf("width 84: the date is neither whole nor separated: %q", clockRow(narrowFrame, "Date UTC"))
	}
	hhmm := narrow.fields[fieldTime].Value()
	if !strings.Contains(narrowFrame, hhmm[:2]+":"+hhmm[2:]) {
		t.Errorf("width 84: the time has no separator (want %q): %q", hhmm[:2]+":"+hhmm[2:], clockRow(narrowFrame, "Time UTC"))
	}

	// Roomy column: both with separators.
	roomyFrame := stripANSI(roomy.View().Content)
	wantDate := raw[:4] + "-" + raw[4:6] + "-" + raw[6:]
	if !strings.Contains(roomyFrame, wantDate) {
		t.Errorf("width 100: the date is not separated (want %q): %q", wantDate, clockRow(roomyFrame, "Date UTC"))
	}

	// Content changes, layout does not: a hand-typed separated date renders in
	// the same place as the clock's digits.
	typed := build(100)
	typed.fields[fieldDate].SetValue(wantDate)
	typed.dateTimeAuto = false
	typed.rc.formSig = ""
	assertSameZones(t, "width 100 (typed date)", fr, captureLayout(t, typed), "status", "tabs", "profile", "form", "table", "help")
}

// TestLayoutAroundTableColumnTiers: the recent-QSOs table changes its columns at
// fixed widths. The table must stay in the same place and keep the same height
// across every threshold — only its columns may change.
func TestLayoutAroundTableColumnTiers(t *testing.T) {
	var widths []int
	for i := range qsoColTiers {
		tier := qsoColTiers[i]
		if tier.maxW < 76 { // below the minimum terminal the app refuses to draw
			continue
		}
		widths = append(widths, tier.maxW+1, tier.maxW+2) // bodyW = w-2
	}
	if len(widths) == 0 {
		t.Fatal("no table tier width to probe")
	}
	base := captureLayout(t, newLayoutTestModel(t, widths[0], 40))
	for _, w := range widths {
		f := captureLayout(t, newLayoutTestModel(t, w, 40))
		assertFrameSane(t, fmt.Sprintf("width %d", w), f)
		assertSameZones(t, fmt.Sprintf("width %d", w), base, f, "status", "tabs", "profile", "form", "table", "help")
	}
}
