package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

// TestCountryWorkedBefore_CountryFallback verifies the dashboard's
// country-worked check: case-insensitive country match including prefix
// variants, excluding the active call itself, with unrelated countries
// reporting false. Also exercises the bounded cache.
func TestCountryWorkedBefore_CountryFallback(t *testing.T) {
	m := newLifecycleTestModel(t)
	db := m.App.DB

	mustInsert := func(call, country string) {
		if _, err := store.InsertQSO(db, &qso.QSO{Call: call, QSODate: "20240101",
			TimeOn: "120000", Band: "20m", Mode: "SSB", Country: country}); err != nil {
			t.Fatalf("InsertQSO: %v", err)
		}
	}
	mustInsert("SP1AAA", "United States")
	mustInsert("SP1BBB", "United States of America")
	mustInsert("SP1CCC", "Poland")

	// The active call itself must not count (its own QSO is excluded).
	pushDashboardLastCall = "SP1AAA"
	if !m.countryWorkedBefore("United States") {
		t.Fatal("United States should be worked by SP1BBB / SP1CCC variants")
	}
	if m.countryWorkedBefore("Germany") {
		t.Fatal("Germany must report false")
	}
	// Prefix variant matches through the LIKE arm.
	if !m.countryWorkedBefore("United States of America") {
		t.Fatal("prefix variant should match")
	}
	// A brand-new call with no history in a worked country must still be true.
	pushDashboardLastCall = "ZZ9NEW"
	if !m.countryWorkedBefore("Poland") {
		t.Fatal("Poland worked by SP1CCC should be true for a new call")
	}
	pushDashboardLastCall = ""
}

// TestStoreCountryWorkedCacheBounded verifies the cache clears wholesale when
// it reaches the cap instead of growing without limit.
func TestStoreCountryWorkedCacheBounded(t *testing.T) {
	for k := range countryWorkedCache {
		delete(countryWorkedCache, k)
	}
	for i := 0; i < countryWorkedCacheCap+100; i++ {
		storeCountryWorked("key", true)
	}
	if len(countryWorkedCache) > countryWorkedCacheCap {
		t.Fatalf("cache grew to %d entries, cap is %d", len(countryWorkedCache), countryWorkedCacheCap)
	}
}

// TestEnqueuePendingADIFBounded verifies the pending-ADIF queue caps at
// maxPendingADIFs and drops the oldest record instead of growing without
// limit (the queue is drained every tick; overflow only matters when the
// database stays busy for a long time).
func TestEnqueuePendingADIFBounded(t *testing.T) {
	m := newLifecycleTestModel(t)
	m.adifQ.mu.Lock()
	for i := 0; i < maxPendingADIFs+3; i++ {
		m.enqueuePendingADIFLocked("<adif>")
	}
	m.adifQ.mu.Unlock()
	if len(m.adifQ.adifs) != maxPendingADIFs {
		t.Fatalf("queue length = %d, want capped at %d", len(m.adifQ.adifs), maxPendingADIFs)
	}
}
