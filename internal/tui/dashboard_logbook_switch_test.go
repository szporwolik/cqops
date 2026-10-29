package tui

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/dashboard"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// TestLogbookSwitchPublishesTheNewTodayList covers what a live dashboard caught:
// switching logbooks replaced the data set without writing a QSO, and the Today
// panel is pushed only when the data is marked dirty — so the switch published
// station/logbook/stats but no `today` event. The browser clears its contacts on
// the logbook event (they belong to the previous logbook), which left the map,
// Top QSOs and the band/mode counts empty until the next save, even though the
// new logbook already had contacts.
func TestLogbookSwitchPublishesTheNewTodayList(t *testing.T) {
	m := dashboardCycleModel(t)
	m.App.ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
	t.Cleanup(m.App.StopAPRSTimer)

	today := time.Now().UTC().Format("20060102")
	insert := func(call string) {
		t.Helper()
		if _, err := store.InsertQSO(m.App.DB, &qso.QSO{
			Call: call, QSODate: today, TimeOn: "120000", Band: "20m", Mode: "CW",
		}); err != nil {
			t.Fatalf("insert %s: %v", call, err)
		}
	}

	insert("DL1AA")
	// The startup push fills the panel for logbook A.
	m.tickCount += 2
	m.pushDashboardState()
	if got := todayCalls(m); len(got) != 1 || got[0] != "DL1AA" {
		t.Fatalf("logbook A shows %v, want [DL1AA]", got)
	}

	// Logbook B already holds its own contact, so switching must publish it.
	m.App.Config.Logbooks["b"] = config.Logbook{
		Station:      config.Station{Callsign: "SP9XYZ", Grid: "JO90AA"},
		DatabasePath: filepath.Join(t.TempDir(), "b.db"),
	}
	if err := m.App.SwitchLogbook("b"); err != nil {
		t.Fatalf("switch to B: %v", err)
	}
	t.Cleanup(func() { m.App.DB.Close() })
	insert("K1JT")

	ch := m.http.client.Hub().Subscribe()
	if ch == nil {
		t.Fatal("dashboard hub is full; cannot subscribe")
	}
	t.Cleanup(func() { m.http.client.Hub().Unsubscribe(ch) })

	m.forcePushDashboardAll()

	published := false
	deadline := time.After(200 * time.Millisecond)
	for !published {
		select {
		case ev := <-ch:
			if ev.Type == string(dashboard.EventToday) {
				published = true
			}
		case <-deadline:
			t.Fatal("the logbook switch published no today event — the dashboard never learns the new logbook's contacts")
		}
	}

	if got := todayCalls(m); len(got) != 1 || got[0] != "K1JT" {
		t.Errorf("the dashboard holds %v after the switch, want [K1JT]", got)
	}
}

// todayCalls reads the contact list the dashboard would serve in /api/snapshot.
func todayCalls(m *Model) []string {
	snap := m.http.client.State().Snapshot()
	calls := make([]string, 0, len(snap.Today))
	for _, v := range snap.Today {
		calls = append(calls, v.Call)
	}
	return calls
}
