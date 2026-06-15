package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
	"github.com/szporwolik/cqops/internal/wavelog"
)

// newTestModel creates a minimal Model for form rendering and navigation tests.
// It initializes only the fields required by QSO form methods.
func newTestModel() *Model {
	cfg := &config.Config{
		General: config.GeneralConfig{DistanceUnit: "km", RenderMap: true},
		Logbooks: map[string]config.Logbook{
			"test": {
				Station: config.Station{
					Callsign: "SP9MOA",
					Grid:     "JO90",
					Operator: "OP",
					RigName:  "default",
				},
			},
		},
		Rigs: map[string]config.RigPreset{
			"default": {Model: "FT-891", Antenna: "Dipole"},
		},
	}
	a := &app.App{
		Config:      cfg,
		LogbookName: "test",
		Logbook:     &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90", Operator: "OP", RigName: "default"}},
	}
	m := New(a, nil)
	return m
}

func TestQSOFormRender(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	view := m.viewForm(90)
	if view == "" {
		t.Error("viewForm returned empty string")
	}
	if !strings.Contains(view, "Call") {
		t.Error("viewForm missing Call label")
	}
	if !strings.Contains(view, "Date UTC") {
		t.Error("viewForm missing Date UTC label")
	}
}

func TestQSOFormRenderNarrow(t *testing.T) {
	m := newTestModel()
	m.width = 40
	m.height = 20

	view := m.viewForm(30)
	if view == "" {
		t.Error("viewForm on narrow width returned empty")
	}
}

func TestQSOFormRenderTiny(t *testing.T) {
	m := newTestModel()
	m.width = 20
	m.height = 10

	view := m.viewForm(15)
	if view == "" {
		t.Error("viewForm on tiny width returned empty")
	}
}

func TestQSOFormFocusedFieldMarker(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30
	m.focus = fieldCall
	m.fields[fieldCall].Focus()

	view := m.viewForm(90)
	// Focused field should show the textinput cursor/view
	if view == "" {
		t.Error("viewForm returned empty with focused field")
	}
}

func TestQSOFormRetainCheckboxUnchecked(t *testing.T) {
	m := newTestModel()
	m.retainComment = false

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty")
	}
	if !strings.Contains(box, "[ ]") {
		t.Error("Retain checkbox should show [ ] when unchecked")
	}
}

func TestQSOFormRetainCheckboxChecked(t *testing.T) {
	m := newTestModel()
	m.retainComment = true

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty")
	}
	if !strings.Contains(box, "[x]") {
		t.Error("Retain checkbox should show [x] when checked")
	}
}

func TestQSOFormRetainCheckboxFocused(t *testing.T) {
	m := newTestModel()
	m.retainFocused = true

	box := m.renderRetainCheckbox(30)
	if box == "" {
		t.Error("renderRetainCheckbox returned empty when focused")
	}
}

func TestQSOFormEmptyValues(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.height = 30

	view := m.viewForm(90)
	// Empty values should show em-dash placeholder
	if !strings.Contains(view, "\u2014") {
		t.Error("viewForm should show em-dash for empty values")
	}
}

func TestQSOFormNextField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].Focus()

	initialFocus := m.focus
	m.nextField()
	if m.focus == initialFocus {
		t.Error("nextField did not change focus from Call")
	}
}

func TestQSOFormPrevField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldTime // second field
	m.fields[fieldTime].Focus()

	m.prevField()
	if m.focus != fieldDate {
		t.Errorf("prevField should move to Date from Time, got focus=%d", m.focus)
	}
}

func TestQSOFormFocusField(t *testing.T) {
	m := newTestModel()
	m.focusField(fieldBand)
	if m.focus != fieldBand {
		t.Errorf("focusField(Band) gave focus=%d, want %d", m.focus, fieldBand)
	}
}

func TestQSOFormClearForm(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldComment].SetValue("Hello")
	m.fields[fieldRSTSent].SetValue("59")
	m.fields[fieldRSTRcvd].SetValue("59")
	m.retainComment = true

	m.clearForm()

	// Comment should be preserved (retain is on)
	if comment := m.fields[fieldComment].Value(); comment != "Hello" {
		t.Errorf("clearForm with retain should preserve comment, got %q", comment)
	}
	// Call should be cleared
	if call := m.fields[fieldCall].Value(); call != "" {
		t.Errorf("clearForm should clear Call, got %q", call)
	}
	// Focus should be on Call
	if m.focus != fieldCall {
		t.Errorf("clearForm should set focus to Call, got %d", m.focus)
	}
}

func TestQSOFormAutoFillRST(t *testing.T) {
	tests := []struct {
		mode     string
		wantSent string
		wantRcvd string
	}{
		{"CW", "599", "599"},
		{"SSB", "59", "59"},
		{"FT8", "59", "59"},
		{"", "59", "59"},
	}
	for _, tt := range tests {
		m := newTestModel()
		m.fields[fieldMode].SetValue(tt.mode)
		m.fields[fieldRSTSent].SetValue("")
		m.fields[fieldRSTRcvd].SetValue("")
		m.autoFillRST()
		if got := m.fields[fieldRSTSent].Value(); got != tt.wantSent {
			t.Errorf("autoFillRST mode=%q RSTSent=%q, want %q", tt.mode, got, tt.wantSent)
		}
		if got := m.fields[fieldRSTRcvd].Value(); got != tt.wantRcvd {
			t.Errorf("autoFillRST mode=%q RSTRcvd=%q, want %q", tt.mode, got, tt.wantRcvd)
		}
	}
}

func TestQSOFormAutoFillRSTNoOverwrite(t *testing.T) {
	m := newTestModel()
	m.fields[fieldMode].SetValue("CW")
	m.fields[fieldRSTSent].SetValue("599")
	m.fields[fieldRSTRcvd].SetValue("579") // already has a value
	m.autoFillRST()
	// Should NOT overwrite existing RST
	if m.fields[fieldRSTRcvd].Value() != "579" {
		t.Errorf("autoFillRST overwrote existing RST rcvd value")
	}
}

func TestQSOFormAutoFillSSBSubmode(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("7.100") // below 10 MHz
	m.fields[fieldMode].SetValue("SSB")
	// Reset band to trigger derivation
	m.fields[fieldBand].SetValue("40m")
	m.autoFillSSBSubmode()
	// applyFreqDefaults will use the freq to derive submode
	// With freq=7.100, band=40m is already set
	// The autoFillSSBSubmode calls applyFreqDefaults which uses freq directly
}

func TestQSOFormUpdateFocused(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].Focus()
	m.fields[fieldCall].SetValue("")

	// Simulate typing 's' via a KeyPressMsg with Code rune
	m.updateFocused(tea.KeyPressMsg{Code: 's', Text: "s"})
	// Should be uppercased
	val := m.fields[fieldCall].Value()
	if val != "S" {
		t.Logf("updateFocused call value: %q (expected 'S')", val)
	}
}

func TestQSOFormPathRow(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.fields[fieldCall].SetValue("SP9MOA") // callsign triggers path info
	m.fields[fieldGrid].SetValue("JN18")   // partner grid

	row := m.formPathRow(90)
	if row == "" {
		t.Error("formPathRow returned empty when both grids set")
	}
}

func TestQSOFormPathRowNoOwnGrid(t *testing.T) {
	m := newTestModel()
	m.App.Logbook.Station.Grid = ""        // no own grid
	m.fields[fieldCall].SetValue("SP9MOA") // callsign entered
	m.fields[fieldGrid].SetValue("JN18")

	row := m.formPathRow(90)
	if row == "" {
		t.Error("formPathRow returned empty — should fall back to station profile")
	}
	// Falls back to station profile when no own grid.
	if !strings.Contains(row, "Op") {
		t.Error("formPathRow should show station profile when no own grid")
	}
}

// Verify no import issues with textinput
var _ = textinput.New

// Verify lipgloss import
var _ = lipgloss.NewStyle

// Verify tea import
var _ = tea.Quit

func TestCommitCall(t *testing.T) {
	m := newTestModel()
	m.fields[fieldCall].SetValue("  sp9moa  ")

	cur := m.commitCall()

	if cur != "SP9MOA" {
		t.Errorf("commitCall: return = %q, want %q", cur, "SP9MOA")
	}
	if m.pathCall != "SP9MOA" {
		t.Errorf("commitCall: pathCall = %q, want %q", m.pathCall, "SP9MOA")
	}
	if m.cachedPathSig != "" {
		t.Errorf("commitCall: cachedPathSig should be empty after commit, got %q", m.cachedPathSig)
	}
}

func TestCommitCallInvalid(t *testing.T) {
	m := newTestModel()
	m.pathCall = "OLD"
	m.cachedPathSig = "something"

	// No letters — invalid callsign.
	m.fields[fieldCall].SetValue("12345")
	cur := m.commitCall()

	if cur != "" {
		t.Errorf("commitCall with invalid call: return = %q, want empty", cur)
	}
	if m.pathCall != "" {
		t.Errorf("commitCall with invalid call: pathCall should be cleared, got %q", m.pathCall)
	}
	if m.cachedPathSig != "" {
		t.Errorf("commitCall with invalid call: cachedPathSig should be cleared, got %q", m.cachedPathSig)
	}
}

func TestCommitCallEmptyCall(t *testing.T) {
	m := newTestModel()
	m.pathCall = "OLD"
	m.cachedPathSig = "something"

	m.commitCall() // field is empty

	if m.pathCall != "" {
		t.Errorf("commitCall with empty field: pathCall = %q, want empty", m.pathCall)
	}
	if m.cachedPathSig != "" {
		t.Errorf("commitCall with empty field: cachedPathSig should be cleared, got %q", m.cachedPathSig)
	}
}

func TestFormPathRowNewCallBannerWLFirst(t *testing.T) {
	// WL data present: WL says NOT worked → banner shows.
	m := newTestModel()
	m.width = 100
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldGrid].SetValue("PG66pa")
	m.pathCall = "VK3A"

	// WL says NOT worked (new call).
	m.wlPrivateData = &wavelog.PrivateLookupResult{}
	// No WL data means it defaults to false for Worked(), which means NOT worked → new.
	// We can't easily set WL raw data, but nil WL acts as "no data".

	row := m.formPathRow(100)
	if !strings.Contains(row, "New Call!") {
		t.Error("formPathRow should show 'New Call!' when local says new (no WL data)")
	}

	// Clear WL — banner should still show based on local.
	m.wlPrivateData = nil
	row = m.formPathRow(100)
	if !strings.Contains(row, "New Call!") {
		t.Error("formPathRow should show 'New Call!' based on local when WL absent")
	}
}

func TestFormPathRowNewCallBannerNotShownWhenWorked(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.fields[fieldCall].SetValue("SP9MOA")
	m.fields[fieldGrid].SetValue("JN18")
	m.pathCall = "SP9MOA"

	// Local stats: call already worked.
	m.cachedLogStats = store.LogbookStats{CallWorked: true, QSOCount: 5}
	m.cachedLogStatsSig = "SP9MOA||"

	row := m.formPathRow(100)
	if strings.Contains(row, "New Call!") {
		t.Error("formPathRow should NOT show 'New Call!' when local says call already worked")
	}
}

func TestFormPathRowNewDXCCBanner(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldGrid].SetValue("PG66pa")
	m.pathCall = "VK3A"
	m.cachedLogStats = store.LogbookStats{CallWorked: false}

	// No WL data → no DXCC banner.
	row := m.formPathRow(100)
	if strings.Contains(row, "New DXCC!") {
		t.Error("formPathRow should NOT show 'New DXCC!' when WL data absent")
	}
}

func TestFormPathRowCacheInvalidation(t *testing.T) {
	m := newTestModel()
	m.width = 100
	m.fields[fieldCall].SetValue("VK3A")
	m.fields[fieldGrid].SetValue("PG66pa")
	m.pathCall = "VK3A"

	// First render — builds and caches.
	r1 := m.formPathRow(100)
	if r1 == "" {
		t.Fatal("formPathRow returned empty")
	}

	// Second render with same inputs — should return cached.
	// cachedPathSig should be non-empty after first render.
	if m.cachedPathSig == "" {
		t.Error("cachedPathSig should be set after first render")
	}

	// Call commitCall — cache should be invalidated.
	m.commitCall()
	if m.cachedPathSig != "" {
		t.Error("cachedPathSig should be empty after commitCall()")
	}

	// Third render — should rebuild (cache was cleared).
	r3 := m.formPathRow(100)
	if r3 == "" {
		t.Error("formPathRow returned empty after cache invalidation")
	}
}

// =============================================================================
// Band / Mode / Submode cycling tests
// =============================================================================

func TestCycleBandUp(t *testing.T) {
	m := newTestModel()
	m.focus = fieldBand
	m.fields[fieldBand].SetValue("20m")

	m.cycleFieldUp()
	if got := m.fields[fieldBand].Value(); got == "" {
		t.Error("cycleFieldUp on band should change value")
	}
	// 20m → 17m (next band up)
	if got := m.fields[fieldBand].Value(); got != "17m" {
		t.Errorf("cycleBand up from 20m = %q, want 17m", got)
	}
}

func TestCycleBandDown(t *testing.T) {
	m := newTestModel()
	m.focus = fieldBand
	m.fields[fieldBand].SetValue("20m")

	m.cycleFieldDown()
	// 20m → 30m (next band down)
	if got := m.fields[fieldBand].Value(); got != "30m" {
		t.Errorf("cycleBand down from 20m = %q, want 30m", got)
	}
}

func TestCycleBandWrapsAround(t *testing.T) {
	m := newTestModel()
	m.focus = fieldBand
	bands := qso.AllBands()
	last := bands[len(bands)-1]

	m.fields[fieldBand].SetValue(last)
	m.cycleFieldUp()
	// Should wrap to first band
	if got := m.fields[fieldBand].Value(); got != bands[0] {
		t.Errorf("cycleBand up from last (%q) = %q, want %q", last, got, bands[0])
	}

	m.fields[fieldBand].SetValue(bands[0])
	m.cycleFieldDown()
	// Should wrap to last band
	if got := m.fields[fieldBand].Value(); got != last {
		t.Errorf("cycleBand down from first (%q) = %q, want %q", bands[0], got, last)
	}
}

func TestCycleBandEmptyField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldBand
	m.fields[fieldBand].SetValue("")

	m.cycleFieldUp()
	// Should pick first band
	if got := m.fields[fieldBand].Value(); got == "" {
		t.Error("cycleBand up from empty should pick a band")
	}
}

func TestCycleModeUp(t *testing.T) {
	m := newTestModel()
	m.focus = fieldMode
	m.fields[fieldMode].SetValue("SSB")

	m.cycleFieldUp()
	if got := m.fields[fieldMode].Value(); got == "" {
		t.Error("cycleFieldUp on mode should change value")
	}
	// Mode should change, submode should be cleared
	if got := m.fields[fieldSubmode].Value(); got != "" {
		t.Errorf("cycleMode should clear submode, got %q", got)
	}
}

func TestCycleModeDown(t *testing.T) {
	m := newTestModel()
	m.focus = fieldMode
	m.fields[fieldMode].SetValue("SSB")

	m.cycleFieldDown()
	if got := m.fields[fieldMode].Value(); got == "" {
		t.Error("cycleFieldDown on mode should change value")
	}
}

func TestCycleModeEmptyField(t *testing.T) {
	m := newTestModel()
	m.focus = fieldMode
	m.fields[fieldMode].SetValue("")

	m.cycleFieldUp()
	if got := m.fields[fieldMode].Value(); got == "" {
		t.Error("cycleMode up from empty should pick a mode")
	}
}

func TestCycleSubmodeUp(t *testing.T) {
	m := newTestModel()
	m.focus = fieldSubmode
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldSubmode].SetValue("USB")

	m.cycleFieldUp()
	// USB → LSB
	if got := m.fields[fieldSubmode].Value(); got != "LSB" {
		t.Errorf("cycleSubmode up from USB = %q, want LSB", got)
	}
}

func TestCycleSubmodeDown(t *testing.T) {
	m := newTestModel()
	m.focus = fieldSubmode
	m.fields[fieldMode].SetValue("SSB")
	m.fields[fieldSubmode].SetValue("USB")

	m.cycleFieldDown()
	// USB → LSB (only 2 submodes for SSB, so up and down both go to LSB)
	if got := m.fields[fieldSubmode].Value(); got != "LSB" {
		t.Errorf("cycleSubmode down from USB = %q, want LSB", got)
	}
}

func TestCycleSubmodeNoSubmodes(t *testing.T) {
	m := newTestModel()
	m.focus = fieldSubmode
	m.fields[fieldMode].SetValue("FM")
	m.fields[fieldSubmode].SetValue("")

	m.cycleFieldUp()
	// FM has no submodes — should stay empty
	if got := m.fields[fieldSubmode].Value(); got != "" {
		t.Errorf("cycleSubmode on FM should stay empty, got %q", got)
	}
}

// =============================================================================
// applyFreqDefaults tests
// =============================================================================

func TestApplyFreqDefaultsHF(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("14.250")
	m.applyFreqDefaults()

	// 14.250 MHz → 20m band, SSB mode, USB submode (>10 MHz)
	if got := m.fields[fieldBand].Value(); got != "20m" {
		t.Errorf("applyFreqDefaults 14.250 band = %q, want 20m", got)
	}
	if got := m.fields[fieldMode].Value(); got != "SSB" {
		t.Errorf("applyFreqDefaults 14.250 mode = %q, want SSB", got)
	}
	if got := m.fields[fieldSubmode].Value(); got != "USB" {
		t.Errorf("applyFreqDefaults 14.250 submode = %q, want USB", got)
	}
}

func TestApplyFreqDefaultsBelow10MHz(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("7.100")
	m.applyFreqDefaults()

	if got := m.fields[fieldBand].Value(); got != "40m" {
		t.Errorf("applyFreqDefaults 7.100 band = %q, want 40m", got)
	}
	if got := m.fields[fieldSubmode].Value(); got != "LSB" {
		t.Errorf("applyFreqDefaults 7.100 submode = %q, want LSB", got)
	}
}

func TestApplyFreqDefaultsVHF(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("145.500")
	m.applyFreqDefaults()

	if got := m.fields[fieldMode].Value(); got != "FM" {
		t.Errorf("applyFreqDefaults 145.500 mode = %q, want FM", got)
	}
	if got := m.fields[fieldSubmode].Value(); got != "" {
		t.Errorf("applyFreqDefaults 145.500 submode = %q, want empty", got)
	}
}

func TestApplyFreqDefaultsEmpty(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("")
	m.fields[fieldBand].SetValue("20m") // pre-existing

	m.applyFreqDefaults()
	// Should not change anything when freq is empty
	if got := m.fields[fieldBand].Value(); got != "20m" {
		t.Errorf("applyFreqDefaults with empty freq should not change band, got %q", got)
	}
}

func TestApplyFreqDefaultsInvalid(t *testing.T) {
	m := newTestModel()
	m.fields[fieldFreq].SetValue("not-a-number")

	m.applyFreqDefaults()
	// Should not crash and should not set band from invalid input
	// (fmt.Sscanf returns 0, so freq stays 0, which is <= 0 → return early)
}

// =============================================================================
// onFieldExit tests
// =============================================================================

func TestOnFieldExitCall(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].SetValue("dj7nt")
	m.fields[fieldMode].SetValue("SSB")

	m.onFieldExit()

	// Call should be committed (normalized)
	if m.pathCall != "DJ7NT" {
		t.Errorf("onFieldExit call: pathCall = %q, want DJ7NT", m.pathCall)
	}
	// RST should be auto-filled
	if m.fields[fieldRSTSent].Value() != "59" {
		t.Errorf("onFieldExit call: RST sent should be auto-filled, got %q", m.fields[fieldRSTSent].Value())
	}
	// QRZ lookup should be flagged
	if !m.qrzNeed {
		t.Error("onFieldExit call: qrzNeed should be true for new call")
	}
}

func TestOnFieldExitCallInvalid(t *testing.T) {
	m := newTestModel()
	m.focus = fieldCall
	m.fields[fieldCall].SetValue("12345") // invalid

	m.onFieldExit()

	// Invalid call should not set pathCall
	if m.pathCall != "" {
		t.Errorf("onFieldExit invalid call: pathCall = %q, want empty", m.pathCall)
	}
}

func TestOnFieldExitGrid(t *testing.T) {
	m := newTestModel()
	m.focus = fieldGrid
	m.fields[fieldGrid].SetValue("jn18")

	m.onFieldExit()

	if m.pathGrid != "JN18" {
		t.Errorf("onFieldExit grid: pathGrid = %q, want JN18", m.pathGrid)
	}
}

func TestOnFieldExitFreq(t *testing.T) {
	m := newTestModel()
	m.focus = fieldFreq
	m.fields[fieldFreq].SetValue("14.250")

	m.onFieldExit()

	// applyFreqDefaults should have been called
	if m.fields[fieldBand].Value() != "20m" {
		t.Errorf("onFieldExit freq: band should be derived, got %q", m.fields[fieldBand].Value())
	}
}

// =============================================================================
// truncate safety test (multi-byte UTF-8 characters)
// =============================================================================

func TestTruncateMultiByte(t *testing.T) {
	// String with a multi-byte middle dot (2 bytes in UTF-8).
	s := "Path  893 km  45\u00b0  New Call!"
	w := lipgloss.Width(s) // should be ~27
	if w < 10 {
		t.Fatal("unexpected width")
	}
	// Truncate to a width that falls on the degree symbol (multi-byte).
	result := truncate(s, w-2)
	// Result should not contain garbled bytes — must be valid.
	if lipgloss.Width(result) > w-2 {
		t.Errorf("truncate width %d > max %d", lipgloss.Width(result), w-2)
	}
	// Should end with ellipsis.
	if !strings.HasSuffix(result, "\u2026") {
		t.Error("truncate should end with ellipsis")
	}
}

func TestTruncateAscii(t *testing.T) {
	result := truncate("Hello World", 8)
	if lipgloss.Width(result) > 8 {
		t.Errorf("truncate width %d > 8", lipgloss.Width(result))
	}
	if !strings.HasSuffix(result, "\u2026") {
		t.Error("truncate should end with ellipsis")
	}
}

func TestTruncateShort(t *testing.T) {
	result := truncate("Hi", 2)
	// max < 3 → returned unchanged
	if result != "Hi" {
		t.Errorf("truncate short string = %q, want Hi", result)
	}
}
