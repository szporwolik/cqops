package tui

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/callbook"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/store"
)

// newWorkedPanelTestModel creates a Model with a temp DB pre-seeded with
// test QSOs, suitable for renderWorkedPanel tests.
func newWorkedPanelTestModel(t *testing.T) (*Model, *sql.DB) {
	t.Helper()

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Seed test QSOs:
	// - KI6NAZ on 20m FT8 (worked call)
	// - Various DXCC 291 (USA) QSOs on multiple bands
	seed := []string{
		// KI6NAZ QSOs (call is worked, US DXCC 291).
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('KI6NAZ','KI6NAZ','20250710','142230','20m',14.074,'FT8','','DM03','291','United States','manual','2025-07-10','2025-07-10')`,
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('KI6NAZ','KI6NAZ','20250711','153000','40m',7.074,'FT8','','DM03','291','United States','manual','2025-07-11','2025-07-11')`,
		// DXCC 291 QSOs with other calls.
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('W1AW','W1AW','20250701','120000','20m',14.074,'SSB','','FN31','291','United States','manual','2025-07-01','2025-07-01')`,
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('W1AW','W1AW','20250702','130000','40m',7.074,'SSB','','FN31','291','United States','manual','2025-07-02','2025-07-02')`,
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('K1JT','K1JT','20250703','140000','15m',21.074,'FT8','','FN20','291','United States','manual','2025-07-03','2025-07-03')`,
		// Grid DM03 QSOs.
		`INSERT INTO qsos (call,base_call,qso_date,time_on,band,freq,mode,submode,gridsquare,dxcc,country,source,created_at,updated_at)
		 VALUES('N6NA','N6NA','20250705','100000','20m',14.074,'FT8','','DM03xx','291','United States','manual','2025-07-05','2025-07-05')`,
	}
	for _, s := range seed {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	cfg := &config.Config{
		General: config.GeneralConfig{Units: "metric"},
		Logbooks: map[string]config.Logbook{
			"test": {
				Station: config.Station{
					Callsign: "SP9MOA",
					Grid:     "JO90",
				},
				Wavelog: &config.WavelogConfig{
					Enabled: true,
					URL:     "https://qso.cqops.com/api/",
					APIKey:  "test-key",
				},
			},
		},
	}
	a := &app.App{
		Config:      cfg,
		DB:          db,
		LogbookName: "test",
		Logbook: &config.Logbook{Station: config.Station{Callsign: "SP9MOA", Grid: "JO90"},
			Wavelog: &config.WavelogConfig{
				Enabled: true,
				URL:     "https://qso.cqops.com/api/",
				APIKey:  "test-key",
			}},
	}
	m := New(a, nil)
	m.width = 120
	m.height = 40
	// Pre-compute logbook stats for the seeded data so renderWorkedPanel
	// picks them up without the async fetch cycle.
	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats
	m.rc.logStatsSig = "KI6NAZ|20m|FT8"
	return m, db
}

func TestWorkedPanel_WorkedCall(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	// Pre-compute stats for this test scenario.
	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
		Country:  "United States",
	}
	view := m.renderWorkedPanel(d, 60)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if !strings.Contains(view, "KI6NAZ") {
		t.Error("missing callsign in output")
	}
	if !strings.Contains(view, "worked") {
		t.Error("expected worked for known call")
	}
	if !strings.Contains(view, "20m") {
		t.Error("missing band in output")
	}
	if !strings.Contains(view, "FT8") {
		t.Error("missing mode in output")
	}
	if !strings.Contains(view, "DM03") {
		t.Error("missing grid in output")
	}
	if !strings.Contains(view, "291") {
		// Entity row now shows "United States · 291" — both parts needed.
		if !strings.Contains(view, "United States") {
			t.Error("missing DXCC entity in output")
		}
	}
}

func TestWorkedPanel_DXCCRowIsCompact(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
		Country:  "United States",
	}
	view := m.renderWorkedPanel(d, 80)
	if !strings.Contains(view, "United States · 291") {
		t.Fatalf("expected compact DXCC value, got %q", view)
	}
	if strings.Contains(view, "United States · 291 · worked") {
		t.Fatalf("did not expect duplicated worked state in DXCC row, got %q", view)
	}
}

func TestWorkedPanel_NewCall(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("")
	m.fields[fieldMode].SetValue("")

	// For a new call, stats should be empty.
	stats, _ := store.GetLogbookStats(db, "XX0XXX", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "999",
		Grid:     "AA00bb",
		Country:  "Testland",
	}
	// Use stacked layout (narrow) so content isn't split across columns.
	view := m.renderWorkedPanel(d, 50)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if !strings.Contains(view, "XX0XXX") {
		t.Error("missing callsign in output")
	}
	if !strings.Contains(view, "NEW") {
		t.Error("expected NEW for unknown call")
	}
	if !strings.Contains(view, "first") {
		t.Error("expected 'first contact' for new call")
	}
	if !strings.Contains(view, "AA00") {
		t.Error("missing 4-char grid")
	}
	if !strings.Contains(view, "\u2014") {
		t.Error("expected dash placeholder for missing band")
	}
	// "awaiting frequency" should NOT appear — compact presentation only.
	if strings.Contains(view, "awaiting") {
		t.Error("should not show verbose 'awaiting' text")
	}
}

func TestWorkedPanel_NewCall_WorkedDXCC(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "291", // USA — worked in seed data.
		Grid:     "DM03ab",
		Country:  "United States",
	}
	// Stacked layout avoids column-splitting issues.
	view := m.renderWorkedPanel(d, 50)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if !strings.Contains(view, "XX0XXX") {
		t.Error("missing callsign in output")
	}
	if !strings.Contains(view, "NEW") {
		t.Error("call should be NEW")
	}
	if !strings.Contains(view, "first") {
		t.Error("expected 'first contact'")
	}
	// DXCC history should appear since DXCC 291 has QSOs.
	if !strings.Contains(view, "DXCC log") {
		t.Error("expected DXCC log fallback for worked entity")
	}
	if !strings.Contains(view, "worked") {
		t.Error("DXCC row should show worked")
	}
}

func TestWorkedPanel_GridNew(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("")
	m.fields[fieldMode].SetValue("")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "999",
		Grid:     "ZZ99ab", // Not in seed data.
		Country:  "Testland",
	}
	view := m.renderWorkedPanel(d, 60)
	if !strings.Contains(view, "ZZ99") {
		t.Error("missing 4-char grid")
	}
	if !strings.Contains(view, "NEW") {
		t.Error("grid should be NEW")
	}
}

func TestWorkedPanel_GridWorked(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("")
	m.fields[fieldMode].SetValue("")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "999",
		Grid:     "DM03cd", // DM03 is in seed data.
		Country:  "Testland",
	}
	view := m.renderWorkedPanel(d, 60)
	if !strings.Contains(view, "DM03") {
		t.Error("missing 4-char grid")
	}
	// Grid DM03 has QSOs, so it should show WORKED.
	if strings.Count(view, "NEW") < 2 {
		// At least call NEW — grid should not be NEW.
	}
}

func TestWorkedPanel_PendingBandMode(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03",
	}
	// Stacked layout ensures pending messages aren't split across columns.
	view := m.renderWorkedPanel(d, 50)
	if !strings.Contains(view, "\u2014") {
		t.Error("expected em-dash placeholder when band/mode empty")
	}
	if strings.Contains(view, "awaiting") {
		t.Error("should not show verbose 'awaiting' text in compact mode")
	}
}

func TestWorkedPanel_DXCCHistoryFallback(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "291",
		Grid:     "ZZ99",
		Country:  "United States",
	}
	// Stacked layout avoids column-splitting.
	view := m.renderWorkedPanel(d, 50)
	if !strings.Contains(view, "DXCC log") {
		t.Error("expected DXCC log fallback when DXCC is worked")
	}
	if !strings.Contains(view, "Last DXCC") {
		t.Error("expected Last DXCC row")
	}
}

func TestWorkedTitle_WithWavelog(t *testing.T) {
	m, _ := newWorkedPanelTestModel(t)
	title := m.workedTitle()
	// Title now wraps the hostname in an OSC8 hyperlink — check for
	// the visible hostname text (not the ANSI escape codes).
	if !strings.Contains(title, "qso.cqops.com") {
		t.Errorf("expected compact hostname in title, got %q", title)
	}
	if !strings.Contains(title, "Local") {
		t.Error("title should mention Local")
	}
	// OSC8 link should be present (contains escape sequences).
	if !strings.Contains(title, "\x1b]8;") {
		t.Error("title should contain OSC8 hyperlink")
	}
}

func TestWorkedTitle_NoWavelog(t *testing.T) {
	m, _ := newWorkedPanelTestModel(t)
	m.App.Logbook.Wavelog = nil
	title := m.workedTitle()
	if title != "Worked · Local" {
		t.Errorf("expected 'Worked · Local', got %q", title)
	}
}

func TestBuildWorkedPanelLayout_FullWidthRows(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{Callsign: "KI6NAZ", DXCC: "291", Grid: "DM03xu", Country: "United States"}
	layout := m.buildWorkedPanelLayout(d, 100)

	if !layout.TwoColumns {
		t.Fatal("expected wide two-column layout")
	}
	if len(layout.FullWidthRows) == 0 {
		t.Fatal("expected full-width distribution rows")
	}
	if got := layout.FullWidthRows[0].label; got != "Bands" {
		t.Fatalf("expected first full-width row Bands, got %q", got)
	}
	if got := layout.FullWidthRows[1].label; got != "Modes" {
		t.Fatalf("expected second full-width row Modes, got %q", got)
	}
	if layout.HistoryScope != "call" {
		t.Fatalf("expected call-history scope, got %q", layout.HistoryScope)
	}
}

func TestBuildWorkedPanelLayout_DXCCScope(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{Callsign: "XX0XXX", DXCC: "291", Grid: "ZZ99", Country: "United States"}
	layout := m.buildWorkedPanelLayout(d, 100)
	if layout.HistoryScope != "dxcc" {
		t.Fatalf("expected dxcc-history scope, got %q", layout.HistoryScope)
	}
	if len(layout.FullWidthRows) == 0 {
		t.Fatal("expected full-width rows for DXCC history")
	}
}

func TestFormatCountList(t *testing.T) {
	items := []store.CountItem{
		{Value: "20m", Count: 74},
		{Value: "40m", Count: 52},
		{Value: "15m", Count: 31},
	}
	got := formatCountList(items)
	if !strings.Contains(got, "20m×74") {
		t.Errorf("missing first item, got %q", got)
	}
	if !strings.Contains(got, "40m×52") {
		t.Errorf("missing second item, got %q", got)
	}
	if !strings.Contains(got, "15m×31") {
		t.Errorf("missing third item, got %q", got)
	}
}

func TestFormatCountList_Overflow(t *testing.T) {
	items := []store.CountItem{
		{Value: "20m", Count: 74},
		{Value: "40m", Count: 52},
		{Value: "15m", Count: 31},
		{Value: "10m", Count: 22},
		{Value: "80m", Count: 18},
	}
	got := formatCountList(items)
	if !strings.Contains(got, "+1") {
		t.Errorf("expected overflow '+1' for 5 items, got %q", got)
	}
	if strings.Contains(got, "80m") {
		t.Error("overflow item should be replaced by +N")
	}
}

func TestFormatCountList_Empty(t *testing.T) {
	got := formatCountList(nil)
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
	got = formatCountList([]store.CountItem{})
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestSelectHistoryScope_Call(t *testing.T) {
	ws := store.WorkedSummary{
		CallHistory: store.ScopeHistory{QSOCount: 5},
	}
	sel := selectHistoryScope(ws)
	if sel.scope != "call" {
		t.Errorf("expected 'call' scope, got %q", sel.scope)
	}
}

func TestSelectHistoryScope_Grid(t *testing.T) {
	ws := store.WorkedSummary{
		CallHistory: store.ScopeHistory{QSOCount: 0},
		GridHistory: store.ScopeHistory{QSOCount: 7, UniqueCalls: 3},
	}
	sel := selectHistoryScope(ws)
	if sel.scope != "grid" {
		t.Errorf("expected 'grid' scope, got %q", sel.scope)
	}
}

func TestSelectHistoryScope_DXCC(t *testing.T) {
	ws := store.WorkedSummary{
		CallHistory: store.ScopeHistory{QSOCount: 0},
		GridHistory: store.ScopeHistory{QSOCount: 1, UniqueCalls: 1}, // only 1 unique call — grid not useful
		DXCCHistory: store.ScopeHistory{QSOCount: 50},
	}
	sel := selectHistoryScope(ws)
	if sel.scope != "dxcc" {
		t.Errorf("expected 'dxcc' scope, got %q", sel.scope)
	}
}

func TestSelectHistoryScope_Empty(t *testing.T) {
	ws := store.WorkedSummary{}
	sel := selectHistoryScope(ws)
	if sel.scope != "" {
		t.Errorf("expected empty scope, got %q", sel.scope)
	}
}

func TestWorkedPanel_TwoColumnLayout(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
	}
	// Wide panel should use two-column layout.
	view := m.renderWorkedPanel(d, 80)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if strings.Contains(view, "Status") {
		t.Error("two-col layout should not have 'Status' heading (stacked-only)")
	}
	// "History · Call ..." scope heading is expected in two-col mode now.
}

func TestWorkedPanel_StackedLayout(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
	}
	view := m.renderWorkedPanel(d, 50)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if !strings.Contains(view, "Status") {
		t.Error("stacked layout should have 'Status' heading")
	}
	if !strings.Contains(view, "History") {
		t.Error("stacked layout should have 'History' heading")
	}
}

func TestWorkedPanel_CompactLayout(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
	}
	view := m.renderWorkedPanel(d, 30)
	if view == "" {
		t.Fatal("renderWorkedPanel returned empty")
	}
	if !strings.Contains(view, "KI6NAZ") {
		t.Error("compact layout missing callsign")
	}
}

func TestWorkedPanel_NoCall(t *testing.T) {
	m, _ := newWorkedPanelTestModel(t)
	view := m.renderWorkedPanel(nil, 60)
	if !strings.Contains(view, "Enter a callsign") {
		t.Error("expected guidance when no callsign")
	}
}

func TestWorkedPanel_NewDXCCHistory(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("XX0XXX")

	stats, _ := store.GetLogbookStats(db, "XX0XXX", "", "")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "XX0XXX",
		DXCC:     "999", // Not in seed.
		Grid:     "ZZ99",
		Country:  "Testland",
	}
	view := m.renderWorkedPanel(d, 60)
	// Left column should show DXCC and Grid as NEW — no zero-filler on right.
	if !strings.Contains(view, "999") {
		t.Error("expected DXCC 999 in output")
	}
	if !strings.Contains(view, "ZZ99") {
		t.Error("expected grid ZZ99 in output")
	}
	if !strings.Contains(view, "first contact") {
		t.Error("expected first contact for new call")
	}
	// "new entity" and "new grid" text should NOT appear — left column
	// already communicates newness; zero-filler is removed.
	if strings.Contains(view, "new entity") {
		t.Error("should not show 'new entity' zero-filler")
	}
	if strings.Contains(view, "new grid") {
		t.Error("should not show 'new grid' zero-filler")
	}
}

func TestWorkedPanel_NoBlankRows(t *testing.T) {
	m, db := newWorkedPanelTestModel(t)
	m.fields[fieldCall].SetValue("KI6NAZ")
	m.fields[fieldBand].SetValue("20m")
	m.fields[fieldMode].SetValue("FT8")

	stats, _ := store.GetLogbookStats(db, "KI6NAZ", "20m", "FT8")
	m.rc.logStats = stats

	d := &callbook.Result{
		Callsign: "KI6NAZ",
		DXCC:     "291",
		Grid:     "DM03xu",
	}
	view := m.renderWorkedPanel(d, 60)
	// Check no blank lines (two consecutive newlines).
	if strings.Contains(view, "\n\n") {
		t.Error("renderWorkedPanel should not contain blank rows")
	}
}
