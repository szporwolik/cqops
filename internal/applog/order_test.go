package applog

import "testing"

// The stored stamp is a wall clock without a date ("15:04:05"), so ordering by it
// is wrong the moment the clock passes midnight — and sorting the shared ring did
// that in place, which also fed the eviction. Insertion order is the only order
// that is right; the ring is appended to chronologically.

// seedEntries installs a ring in the given order and returns a function that
// restores the previous contents.
func seedEntries(t *testing.T, ring []Entry) {
	t.Helper()
	mu.Lock()
	previous := entries
	entries = ring
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		entries = previous
		mu.Unlock()
	})
}

func messages(es []Entry) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Message
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestEntries_KeepInsertionOrderAcrossMidnight: an entry logged after midnight is
// newer than one logged before it, whatever the clock face says.
func TestEntries_KeepInsertionOrderAcrossMidnight(t *testing.T) {
	seedEntries(t, []Entry{
		{Time: "23:59:59", Level: "INFO", Message: "before midnight"},
		{Time: "00:00:00", Level: "INFO", Message: "after midnight"},
	})
	got := messages(Entries())
	want := []string{"before midnight", "after midnight"}
	if !equalStrings(got, want) {
		t.Errorf("order across midnight = %v, want %v (the newest entry sorts first by clock face)", got, want)
	}
}

// TestEntries_KeepInsertionOrderWithinTheSameSecond: entries sharing a second are
// told apart by the order they arrived in.
func TestEntries_KeepInsertionOrderWithinTheSameSecond(t *testing.T) {
	seedEntries(t, []Entry{
		{Time: "12:00:00", Message: "first"},
		{Time: "12:00:00", Message: "second"},
		{Time: "12:00:00", Message: "third"},
	})
	got := messages(Entries())
	want := []string{"first", "second", "third"}
	if !equalStrings(got, want) {
		t.Errorf("same-second order = %v, want %v", got, want)
	}
}

// TestEntries_DoNotReorderTheRing: reading the log must not touch the shared
// buffer — the ring's front is what Append evicts, so reordering it drops a live
// entry instead of the oldest one.
func TestEntries_DoNotReorderTheRing(t *testing.T) {
	seedEntries(t, []Entry{
		{Time: "23:59:59", Message: "older"},
		{Time: "00:00:00", Message: "newer"},
	})
	Entries()
	EntriesWithRevision()

	mu.Lock()
	defer mu.Unlock()
	if len(entries) != 2 || entries[0].Message != "older" || entries[1].Message != "newer" {
		t.Errorf("the ring was reordered by a read: %v", messages(entries))
	}
}

// TestAppend_OverflowKeepsTheNewestInOrder is the acceptance sequence end to end:
// one more entry than the ring holds, and the survivors are the newest ones, in
// the order they were logged.
func TestAppend_OverflowKeepsTheNewestInOrder(t *testing.T) {
	mu.Lock()
	previous := entries
	entries = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		entries = previous
		mu.Unlock()
	})

	total := maxStored + 10
	for i := 0; i < total; i++ {
		Append("INFO", messageAt(i), "")
	}
	got := Entries()
	if len(got) != maxStored {
		t.Fatalf("ring holds %d entries, want %d", len(got), maxStored)
	}
	for i, e := range got {
		if want := messageAt(total - maxStored + i); e.Message != want {
			t.Fatalf("entry %d is %q, want %q — the ring dropped the wrong end", i, e.Message, want)
		}
	}
}

// TestAppend_OverflowAcrossMidnight uses stamps that wrap: the oldest entries
// (logged before midnight) must be the ones evicted.
func TestAppend_OverflowAcrossMidnight(t *testing.T) {
	mu.Lock()
	previous := entries
	// A full ring of pre-midnight entries, oldest first.
	entries = make([]Entry, 0, maxStored)
	for i := 0; i < maxStored; i++ {
		entries = append(entries, Entry{Time: "23:5" + string(rune('0'+i%10)) + ":59", Message: "before", Level: "INFO"})
	}
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		entries = previous
		mu.Unlock()
	})

	Append("INFO", "after midnight", "")

	got := Entries()
	if len(got) != maxStored {
		t.Fatalf("ring holds %d entries, want %d", len(got), maxStored)
	}
	if last := got[len(got)-1]; last.Message != "after midnight" {
		t.Errorf("the entry logged after midnight is not the newest: %q", last.Message)
	}
	for i := 0; i < len(got)-1; i++ {
		if got[i].Message != "before" {
			t.Errorf("entry %d is %q, want a pre-midnight entry", i, got[i].Message)
		}
	}
}

// messageAt names an entry by its position so eviction order is visible.
func messageAt(i int) string {
	digits := []byte("0123456789")
	return "entry-" + string([]byte{
		digits[(i/100)%10], digits[(i/10)%10], digits[i%10],
	})
}
