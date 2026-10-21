package store

import (
	"database/sql"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/qso"
)

// mustQSO inserts one QSO and fails the test on error.
func mustQSO(t *testing.T, db *sql.DB, q *qso.QSO) int64 {
	t.Helper()
	return mustInsertQSO(t, db, q)
}

func metaOrEmpty(t *testing.T, db *sql.DB, key string) string {
	t.Helper()
	var v string
	err := db.QueryRow(`SELECT value FROM worked_index_meta WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return ""
	}
	if err != nil {
		t.Fatalf("read meta %s: %v", key, err)
	}
	return v
}

// TestWorkedIndexTracksConfirmations: confirmed_count follows inserts, edits
// that clear the confirmation, deletions and a full rebuild.
func TestWorkedIndexTracksConfirmations(t *testing.T) {
	db := newTempDB(t)
	mustQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269",
	})
	confirmed := mustQSO(t, db, &qso.QSO{
		Call: "SP9MOA", QSODate: "20240502", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269", LOTWQSLRcvd: "Y",
	})

	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 1 {
		t.Fatalf("confirmed_count = %d, want 1", got)
	}

	// Clear the confirmation through an ordinary edit: the aggregate must
	// follow the stored row, not the caller's snapshot.
	q, err := GetQSOByID(db, confirmed)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q.LOTWQSLRcvd = "N"
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 0 {
		t.Errorf("confirmed_count = %d after clearing the confirmation, want 0", got)
	}
	// An edit that does not carry confirmation fields must not wipe them.
	q2, err := GetQSOByID(db, confirmed)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q2.LOTWQSLRcvd = "Y"
	if err := UpdateQSO(db, q2); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	q3, err := GetQSOByID(db, confirmed)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q3.Comment = "comment only"
	if err := UpdateQSO(db, q3); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 1 {
		t.Errorf("confirmed_count = %d after a comment-only edit, want 1 (preserved)", got)
	}

	// Incremental and rebuilt aggregates must agree.
	if err := RebuildWorkedIndex(db); err != nil {
		t.Fatalf("RebuildWorkedIndex: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 1 {
		t.Errorf("confirmed_count = %d after rebuild, want 1", got)
	}
	if got := workedCountOrZero(t, db, `SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 2 {
		t.Errorf("qso_count = %d after rebuild, want 2", got)
	}
	if err := DeleteQSO(db, confirmed); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 0 {
		t.Errorf("confirmed_count = %d after deleting the confirmed contact, want 0", got)
	}
	if got := workedCountOrZero(t, db, `SELECT confirmed_count FROM worked_call WHERE base_call='SP9MOA'`); got != 0 {
		t.Errorf("worked_call confirmed_count = %d after delete, want 0", got)
	}
}

// TestWorkedIndexSourceMask: the provenance mask ORs sources in and clears a
// bit again once no contact from that path is left.
func TestWorkedIndexSourceMask(t *testing.T) {
	db := newTempDB(t)
	mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269", Source: "manual"})
	wsjtx := mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240502", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269", Source: "wsjtx"})

	mask := workedCountOrZero(t, db, `SELECT source_mask FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`)
	if mask != SourceMaskManual|SourceMaskWSJTX {
		t.Fatalf("source_mask = %d, want manual|wsjtx = %d", mask, SourceMaskManual|SourceMaskWSJTX)
	}

	if err := DeleteQSO(db, wsjtx); err != nil {
		t.Fatalf("DeleteQSO: %v", err)
	}
	mask = workedCountOrZero(t, db, `SELECT source_mask FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`)
	if mask != SourceMaskManual {
		t.Errorf("source_mask = %d after removing the WSJT-X contact, want manual only", mask)
	}

	// Rebuild must produce the same mask as the incremental path.
	mustQSO(t, db, &qso.QSO{Call: "DL1AA", QSODate: "20240503", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269", Source: "wavelog"})
	if err := RebuildWorkedIndex(db); err != nil {
		t.Fatalf("RebuildWorkedIndex: %v", err)
	}
	mask = workedCountOrZero(t, db, `SELECT source_mask FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`)
	if mask != SourceMaskManual|SourceMaskRemote {
		t.Errorf("source_mask = %d after rebuild, want manual|remote = %d", mask, SourceMaskManual|SourceMaskRemote)
	}
}

// TestWorkedIndexSourceMaskClassification pins the source→bit mapping used by
// both the incremental path and the rebuild SQL.
func TestWorkedIndexSourceMaskClassification(t *testing.T) {
	cases := map[string]int{
		"":            SourceMaskManual,
		"manual":      SourceMaskManual,
		"form":        SourceMaskManual,
		"wsjtx":       SourceMaskWSJTX,
		"WSJT-X":      SourceMaskWSJTX,
		"adif":        SourceMaskADIF,
		"import":      SourceMaskADIF,
		"wavelog":     SourceMaskRemote,
		"kiss":        SourceMaskOther,
		"some-plugin": SourceMaskOther,
	}
	for source, want := range cases {
		if got := sourceMaskBit(source); got != want {
			t.Errorf("sourceMaskBit(%q) = %d, want %d", source, got, want)
		}
	}

	// The SQL classification must agree with the Go mapping.
	db := newTempDB(t)
	for i, source := range []string{"", "manual", "form", "wsjtx", "WSJT-X", "adif", "import", "wavelog", "kiss"} {
		mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "1200" + string(rune('0'+i)),
			Band: "20m", Mode: "FT8", DXCC: "269", Source: source})
	}
	if err := RebuildWorkedIndex(db); err != nil {
		t.Fatalf("RebuildWorkedIndex: %v", err)
	}
	got := workedCountOrZero(t, db, `SELECT source_mask FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`)
	want := SourceMaskManual | SourceMaskWSJTX | SourceMaskADIF | SourceMaskRemote | SourceMaskOther
	if got != want {
		t.Errorf("rebuilt source_mask = %d, want %d (all bits)", got, want)
	}
}

// TestDXCCProvenance: every write path records how the entity was resolved,
// and an edit that does not manage DXCC keeps both the number and its origin.
func TestDXCCProvenance(t *testing.T) {
	db := newTempDB(t)

	// 1. Explicit: an entity supplied with the record.
	explicit := mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "SSB", DXCC: "269"})
	assertDXCCSource(t, db, explicit, "269", qso.DXCCSourceExplicit)

	// 2. Unknown: nothing to resolve yet.
	unresolved := mustQSO(t, db, &qso.QSO{Call: "F5ABC", QSODate: "20240502", TimeOn: "120000",
		Band: "20m", Mode: "SSB"})
	assertDXCCSource(t, db, unresolved, "", qso.DXCCSourceUnknown)

	// 3. Prefix: the CTY backfill resolves it.
	if _, err := BackfillDXCC(db, map[int64]string{unresolved: "227"}); err != nil {
		t.Fatalf("BackfillDXCC: %v", err)
	}
	assertDXCCSource(t, db, unresolved, "227", qso.DXCCSourcePrefix)

	// 4. Callbook: enrichment only fills an unresolved row.
	callbook := mustQSO(t, db, &qso.QSO{Call: "DJ7NT", QSODate: "20240503", TimeOn: "120000",
		Band: "20m", Mode: "FT8"})
	if err := UpdateQSOEnrichment(db, callbook, EnrichmentData{DXCC: "230"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment: %v", err)
	}
	assertDXCCSource(t, db, callbook, "230", qso.DXCCSourceCallbook)
	// A second enrichment must not relabel an already-resolved entity.
	if err := UpdateQSOEnrichment(db, callbook, EnrichmentData{DXCC: "999"}); err != nil {
		t.Fatalf("UpdateQSOEnrichment (second): %v", err)
	}
	assertDXCCSource(t, db, callbook, "230", qso.DXCCSourceCallbook)

	// 5. A comment-only edit keeps the entity and its provenance.
	q, err := GetQSOByID(db, callbook)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q.DXCC, q.DXCCSource = "", ""
	q.Comment = "edited"
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	assertDXCCSource(t, db, callbook, "230", qso.DXCCSourceCallbook)

	// 6. Changing the callsign invalidates the stale entity (the form save
	// path carries no DXCC at all).
	q, err = GetQSOByID(db, callbook)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	q.DXCC, q.DXCCSource = "", ""
	q.Call = "G4ABC"
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	assertDXCCSource(t, db, callbook, "", qso.DXCCSourceUnknown)
}

func assertDXCCSource(t *testing.T, db *sql.DB, id int64, wantDXCC, wantSource string) {
	t.Helper()
	var dxcc, source string
	if err := db.QueryRow(`SELECT COALESCE(dxcc,''), dxcc_source FROM qsos WHERE id = ?`, id).Scan(&dxcc, &source); err != nil {
		t.Fatalf("read dxcc provenance: %v", err)
	}
	if dxcc != wantDXCC || source != wantSource {
		t.Errorf("dxcc=%q source=%q, want dxcc=%q source=%q", dxcc, source, wantDXCC, wantSource)
	}
}

// TestCoverageDrivesVerdicts pins the honesty rules: a positive hit is always
// WORKED, an absence is NEW only when the relevant dimension is complete.
func TestCoverageDrivesVerdicts(t *testing.T) {
	db := newTempDB(t)

	cov, err := Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	// An empty log is complete for local data — a first contact is genuinely
	// NEW. Whether a remote logbook still has to be synchronized is decided by
	// the caller through LastRemoteSync (covered by the panel tests).
	if !cov.CallComplete() || !cov.DXCCComplete() {
		t.Errorf("an empty log should be locally complete: %+v", cov)
	}
	if got := Verdict(false, cov.CallComplete()); got != StateNew {
		t.Errorf("verdict on an empty log = %v, want NEW", got)
	}
	if cov.LastRemoteSync != "" {
		t.Errorf("LastRemoteSync = %q, want empty before synchronization", cov.LastRemoteSync)
	}

	mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269", GridSquare: "JO90"})
	cov, err = Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if !cov.Complete() {
		t.Fatalf("complete log reported incomplete: %+v", cov)
	}
	if got := Verdict(true, cov.Complete()); got != StateWorked {
		t.Errorf("worked verdict = %v, want WORKED", got)
	}
	if got := Verdict(false, cov.Complete()); got != StateNew {
		t.Errorf("absent verdict on a complete log = %v, want NEW", got)
	}

	// An unresolved entity makes DXCC verdicts uncertain, while call/grid
	// answers stay trustworthy.
	mustQSO(t, db, &qso.QSO{Call: "XX1AA", QSODate: "20240502", TimeOn: "120000",
		Band: "20m", Mode: "FT8", GridSquare: "AA00"})
	cov, err = Coverage(db)
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}
	if cov.UnresolvedDXCC != 1 {
		t.Fatalf("UnresolvedDXCC = %d, want 1", cov.UnresolvedDXCC)
	}
	if !cov.CallComplete() || !cov.BandComplete() || !cov.ModeComplete() {
		t.Errorf("call/band/mode coverage should stay complete: %+v", cov)
	}
	if cov.DXCCComplete() {
		t.Error("DXCC coverage must be incomplete with an unresolved entity")
	}
	if got := Verdict(false, cov.DXCCComplete()); got != StateUnknown {
		t.Errorf("absent DXCC verdict = %v, want UNKNOWN", got)
	}

	// A remote sync marker is recorded independently of the aggregates.
	if metaOrEmpty(t, db, metaKeyLastRemoteSync) != "" {
		t.Fatal("sync marker set before MarkRemoteSync")
	}
	if err := MarkRemoteSync(db, time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)); err != nil {
		t.Fatalf("MarkRemoteSync: %v", err)
	}
	if got := metaOrEmpty(t, db, metaKeyLastRemoteSync); got != "2026-01-02T03:04:05Z" {
		t.Errorf("last_remote_sync = %q", got)
	}
}

// TestWorkedPointLookups: the index answers every panel verdict with a
// primary-key seek, including confirmed counts and the four DXCC levels.
func TestWorkedPointLookups(t *testing.T) {
	db := newTempDB(t)
	mustQSO(t, db, &qso.QSO{Call: "SP9MOA/P", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "MFSK", Submode: "FT8", GridSquare: "JO90xx", DXCC: "269",
		QSLRcvd: "Y"})
	mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240502", TimeOn: "130000",
		Band: "40m", Mode: "SSB", GridSquare: "JO90", DXCC: "269"})

	call, err := WorkedCall(db, "SP9MOA")
	if err != nil {
		t.Fatalf("WorkedCall: %v", err)
	}
	if call.Count != 2 || call.ConfirmedCount != 1 {
		t.Errorf("WorkedCall = %+v, want 2 QSOs / 1 confirmed", call)
	}
	if call.FirstUTC != "20240501120000" || call.LastUTC != "20240502130000" {
		t.Errorf("WorkedCall edges = %s..%s", call.FirstUTC, call.LastUTC)
	}

	grid, err := WorkedGrid(db, "JO90")
	if err != nil {
		t.Fatalf("WorkedGrid: %v", err)
	}
	if grid.Count != 2 {
		t.Errorf("WorkedGrid(JO90) = %+v, want 2", grid)
	}

	levels := []struct {
		band, mode string
		want       int
	}{
		{"", "", 2},
		{"20m", "", 1},
		{"40m", "", 1},
		{"", "FT8", 1},
		{"", "SSB", 1},
		{"20m", "FT8", 1},
		{"20m", "SSB", 0},
	}
	for _, lv := range levels {
		w, err := WorkedDXCC(db, "269", lv.band, lv.mode)
		if err != nil {
			t.Fatalf("WorkedDXCC(%q,%q): %v", lv.band, lv.mode, err)
		}
		if w.Count != lv.want {
			t.Errorf("WorkedDXCC(269,%q,%q) = %d, want %d", lv.band, lv.mode, w.Count, lv.want)
		}
	}

	// Absent keys are a clean zero, not an error.
	for _, probe := range []struct {
		name string
		get  func() (WorkedInfo, error)
	}{
		{"call", func() (WorkedInfo, error) { return WorkedCall(db, "N0CALL") }},
		{"grid", func() (WorkedInfo, error) { return WorkedGrid(db, "AA00") }},
		{"dxcc", func() (WorkedInfo, error) { return WorkedDXCC(db, "999", "", "") }},
	} {
		w, err := probe.get()
		if err != nil {
			t.Fatalf("%s probe: %v", probe.name, err)
		}
		if w.Worked() || w.Count != 0 {
			t.Errorf("%s probe = %+v, want an absent aggregate", probe.name, w)
		}
	}
	// Degenerate inputs short-circuit without touching the database.
	if w, err := WorkedDXCC(db, "", "", ""); err != nil || w.Count != 0 {
		t.Errorf("empty dxcc lookup = %+v, %v", w, err)
	}
	if w, err := WorkedGrid(db, "JO"); err != nil || w.Count != 0 {
		t.Errorf("short grid lookup = %+v, %v", w, err)
	}
}

// TestIndexRulesVersionRebuild: a stale (or missing) rules version rebuilds
// the aggregates exactly once; a current version leaves them alone.
func TestIndexRulesVersionRebuild(t *testing.T) {
	db := newTempDB(t)
	mustQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000",
		Band: "20m", Mode: "FT8", DXCC: "269"})

	if stored := metaOrEmpty(t, db, metaKeyRulesVersion); stored != workedIndexRulesVersion {
		t.Fatalf("rules_version = %q after init, want %q", stored, workedIndexRulesVersion)
	}

	// Simulate a rules change: an older version, plus a drifted index.
	if err := SetMetaValue(db, metaKeyRulesVersion, "0"); err != nil {
		t.Fatalf("SetMetaValue: %v", err)
	}
	if _, err := db.Exec(`DELETE FROM worked_dxcc`); err != nil {
		t.Fatalf("drift: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT qso_count FROM worked_dxcc WHERE dxcc='269' AND band='' AND mode=''`); got != 1 {
		t.Errorf("aggregate = %d after a rules-version rebuild, want 1", got)
	}
	if stored := metaOrEmpty(t, db, metaKeyRulesVersion); stored != workedIndexRulesVersion {
		t.Errorf("rules_version = %q after rebuild, want %q", stored, workedIndexRulesVersion)
	}

	// With a current version the migration must NOT rebuild: an emptied
	// aggregate stays empty (proves this is not a rebuild-every-start). The
	// legacy "build a missing index" heuristic keys on worked_call, so that
	// table is left populated and worked_dxcc is the one emptied.
	if _, err := db.Exec(`DELETE FROM worked_dxcc`); err != nil {
		t.Fatalf("drift: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if got := workedCountOrZero(t, db, `SELECT COUNT(*) FROM worked_dxcc`); got != 0 {
		t.Errorf("worked_dxcc rows = %d, want 0 (migration must not rebuild on a current version)", got)
	}
}
