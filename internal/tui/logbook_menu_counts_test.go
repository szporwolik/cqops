package tui

import (
	"fmt"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// The per-logbook QSO counts are read by a background command while the menu's
// own handlers add and delete logbooks — i.e. while Config.Logbooks is being
// written. The worker therefore gets a snapshot taken on the update loop, and
// its result is tagged with the chooser instance that queued it.

// newCountsTestApp builds an app whose logbooks have real, seeded databases at
// temp paths (never the user's data directory) with the given QSO counts.
func newCountsTestApp(t *testing.T, counts map[string]int) *app.App {
	t.Helper()
	dir := t.TempDir()
	logbooks := make(map[string]config.Logbook, len(counts))
	for id, n := range counts {
		dbPath := filepath.Join(dir, id+".db")
		db, err := store.InitDB(dbPath)
		if err != nil {
			t.Fatalf("InitDB(%s): %v", id, err)
		}
		for i := 0; i < n; i++ {
			if _, err := store.InsertQSO(db, &qso.QSO{
				Call:    fmt.Sprintf("T%02d", i),
				QSODate: "20260101",
				TimeOn:  "120000",
				Band:    "20m",
				Mode:    "CW",
			}); err != nil {
				t.Fatalf("InsertQSO(%s): %v", id, err)
			}
		}
		if err := db.Close(); err != nil {
			t.Fatalf("close %s: %v", id, err)
		}
		logbooks[id] = config.Logbook{
			Name:         id,
			DatabasePath: dbPath,
			Station:      config.Station{Callsign: "SP9MOA"},
		}
	}
	return &app.App{
		Config: &config.Config{
			State:    config.StateConfig{ActiveLogbook: "home"},
			Logbooks: logbooks,
		},
		ConfigPath:  filepath.Join(dir, "config.yaml"),
		LogbookName: "home",
	}
}

// TestQSOsCountPassSnapshotsTheLogbookList holds the queued worker (nothing
// runs until the test calls the command), deletes a logbook the way the menu
// does, and then lets the pass finish.
//
// The message must describe the list as it was when the pass was queued — that
// is what proves the worker never reads the live configuration — and applying
// it must not bring the deleted logbook back into the counts.
func TestQSOsCountPassSnapshotsTheLogbookList(t *testing.T) {
	a := newCountsTestApp(t, map[string]int{"home": 3, "portable": 2})
	c := NewLogbookChooser(a, NewToastQueue())
	cmd := c.Init() // queued, deliberately not run yet
	if cmd == nil {
		t.Fatal("Init did not queue the count pass")
	}

	// The operator deletes the inactive logbook while the pass is still queued.
	for i, id := range c.names {
		if id == "portable" {
			c.cursor = i
		}
	}
	c.deleteLogbook()
	if _, ok := a.Config.Logbooks["portable"]; ok {
		t.Fatal("setup: deleteLogbook did not remove the logbook")
	}

	msg, ok := cmd().(qsoCountsMsg)
	if !ok {
		t.Fatalf("count pass returned %T, want qsoCountsMsg", cmd())
	}
	if msg.gen != c.gen {
		t.Errorf("result tagged with generation %d, want the chooser's %d", msg.gen, c.gen)
	}
	if got, want := msg.counts["home"], 3; got != want {
		t.Errorf("snapshot count for home = %d, want %d", got, want)
	}
	if _, ok := msg.counts["portable"]; !ok {
		t.Fatal("the pass read the live configuration instead of its snapshot (portable is missing)")
	}

	// A late result must not restore the deleted row.
	c.Update(msg)
	if _, ok := c.qsoCounts["portable"]; ok {
		t.Errorf("a late count pass restored the deleted logbook: %v", c.qsoCounts)
	}
	if len(c.qsoCounts) != len(c.names) {
		t.Errorf("counts cover %d logbooks, the list shows %d: %v", len(c.qsoCounts), len(c.names), c.qsoCounts)
	}
	if got, want := c.qsoCounts["home"], 3; got != want {
		t.Errorf("home count = %d, want %d", got, want)
	}
}

// TestQSOsCountPassKeepsALogbookCreatedWhileItRan: the pass has no count for a
// logbook created after it was queued, and the zero installed by the create
// path must survive.
func TestQSOsCountPassKeepsALogbookCreatedWhileItRan(t *testing.T) {
	a := newCountsTestApp(t, map[string]int{"home": 3})
	c := NewLogbookChooser(a, NewToastQueue())
	cmd := c.Init()

	// The create path appends the id and installs a zero count.
	c.names = append(c.names, "new")
	c.qsoCounts["new"] = 0

	c.Update(cmd())
	if n, ok := c.qsoCounts["new"]; !ok || n != 0 {
		t.Fatalf("count for the logbook created during the pass = %d (present %v), want 0", n, ok)
	}
	if got, want := c.qsoCounts["home"], 3; got != want {
		t.Errorf("home count = %d, want %d", got, want)
	}
}

// TestQSOsCountPassIgnoresAnotherChooserInstance: closing and reopening the
// menu creates a new chooser while the old pass may still be running. Its
// result belongs to the old instance and must not be installed into the new
// one.
func TestQSOsCountPassIgnoresAnotherChooserInstance(t *testing.T) {
	a := newCountsTestApp(t, map[string]int{"home": 3, "portable": 2})
	first := NewLogbookChooser(a, NewToastQueue())
	cmd := first.Init()

	second := NewLogbookChooser(a, NewToastQueue())
	if second.gen == first.gen {
		t.Fatal("two choosers share a generation")
	}

	msg := cmd()
	second.Update(msg)
	if len(second.qsoCounts) != 0 {
		t.Errorf("a stale pass filled the new chooser's counts: %v", second.qsoCounts)
	}

	first.Update(msg) // the instance that asked for it still accepts it
	if got, want := first.qsoCounts["home"], 3; got != want {
		t.Errorf("the queueing chooser did not receive its own result: %v", first.qsoCounts)
	}
}

// TestQSOsCountPassRunsWhileTheConfigurationChanges runs the real command on a
// separate goroutine while the update loop keeps adding and deleting logbooks.
// The worker owns a private snapshot, so there is no concurrent map access —
// this is the shape that fails under -race when the command walks
// Config.Logbooks itself.
func TestQSOsCountPassRunsWhileTheConfigurationChanges(t *testing.T) {
	a := newCountsTestApp(t, map[string]int{"home": 3, "portable": 2})
	c := NewLogbookChooser(a, NewToastQueue())
	cmd := c.Init()

	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	for i := 0; i < 200; i++ {
		a.Config.Logbooks["scratch"] = config.Logbook{Name: "scratch"}
		delete(a.Config.Logbooks, "scratch")
	}

	msg := (<-done).(qsoCountsMsg)
	c.Update(msg)
	if _, ok := c.qsoCounts["scratch"]; ok {
		t.Errorf("counts mention a logbook that never made it into the list: %v", c.qsoCounts)
	}
	if got, want := c.qsoCounts["home"], 3; got != want {
		t.Errorf("home count = %d, want %d", got, want)
	}
}
