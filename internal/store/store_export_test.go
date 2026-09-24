package store

import (
	"fmt"
	"strings"
	"testing"

	adif "github.com/farmergreg/adif/v5"
	"github.com/szporwolik/cqops/internal/qso"
)

// TestExportQSOsSnapshot_ADIFFieldFidelity verifies every supported
// persisted field survives the snapshot export and an ADIF encode/parse
// round-trip. Count and ordering checks cannot detect a silently omitted
// projection column — this is the regression test for DXCC being dropped
// from the shared snapshot projection.
func TestExportQSOsSnapshot_ADIFFieldFidelity(t *testing.T) {
	db := newTempDB(t)
	want := &qso.QSO{
		Call: "SP9MOA", QSODate: "20240501", TimeOn: "120030", TimeOff: "120130",
		Band: "20m", Freq: 14.25, FreqRx: 14.2505, Mode: "MFSK", Submode: "FT4",
		RSTSent: "59", RSTRcvd: "57",
		GridSquare: "JO90AA", Name: "Jan", QTH: "Krakow", Country: "Poland",
		Comment: "comment", Notes: "notes", TXPower: "25",
		StationCallsign: "SP9MOA", Operator: "SP9MOA",
		MyGridSquare: "KO00CA", MyRig: "FT-891", MyAntenna: "Dipole",
		MyCQZone: "15", MyITUZone: "28", MyDXCC: "269",
		SOTARef: "SP/TQ-001", POTARef: "SP-1234", WWFFRef: "SPFF-001",
		IOTA: "EU-005", SIG: "SIG", SIGInfo: "info",
		MySOTARef: "SP/TQ-002", MyPOTARef: "SP-5678", MyWWFFRef: "SPFF-002",
		MySIG: "MYSIG", MySIGInfo: "myinfo",
		Distance: 123.4, Bearing: 90,
		CQZone: "15", ITUZone: "28", DXCC: "269",
		STX: 5, SRX: 7, STXString: "ZONE", SRXString: "SERIAL",
		ContestADIFID: "CQ-WW-SSB",
	}
	if _, err := InsertQSO(db, want); err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}

	var exported qso.QSO
	count := 0
	if err := ExportQSOsSnapshot(db, "", false, func(q qso.QSO) error {
		exported = q
		count++
		return nil
	}); err != nil {
		t.Fatalf("ExportQSOsSnapshot: %v", err)
	}
	if count != 1 {
		t.Fatalf("exported %d rows, want 1", count)
	}

	// The snapshot projection itself must carry DXCC (the regression).
	if exported.DXCC != want.DXCC {
		t.Errorf("snapshot DXCC = %q, want %q", exported.DXCC, want.DXCC)
	}

	// Full round-trip through the real ADIF writer and parser.
	sc := adif.NewScanner(strings.NewReader(exported.ToADIF()))
	var rec adif.Record
	for sc.Scan() {
		if sc.IsHeader() {
			continue
		}
		rec = sc.Record()
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan ADIF: %v", err)
	}
	if rec == nil {
		t.Fatal("no ADIF record produced by ToADIF")
	}
	round := qso.ParseADIFRecord(rec, "import")

	// Every string field the ADIF format supports round-tripping.
	strChecks := []struct{ name, want, got string }{
		{"Call", want.Call, round.Call},
		{"QSODate", want.QSODate, round.QSODate},
		{"TimeOn", want.TimeOn, round.TimeOn},
		{"TimeOff", want.TimeOff, round.TimeOff},
		{"Band", want.Band, round.Band},
		{"Mode", want.Mode, round.Mode},
		{"Submode", want.Submode, round.Submode},
		{"RSTSent", want.RSTSent, round.RSTSent},
		{"RSTRcvd", want.RSTRcvd, round.RSTRcvd},
		{"GridSquare", want.GridSquare, round.GridSquare},
		{"Name", want.Name, round.Name},
		{"QTH", want.QTH, round.QTH},
		{"Country", want.Country, round.Country},
		{"Comment", want.Comment, round.Comment},
		{"Notes", want.Notes, round.Notes},
		{"TXPower", want.TXPower, round.TXPower},
		{"StationCallsign", want.StationCallsign, round.StationCallsign},
		{"Operator", want.Operator, round.Operator},
		{"MyGridSquare", want.MyGridSquare, round.MyGridSquare},
		{"MyRig", want.MyRig, round.MyRig},
		{"MyAntenna", want.MyAntenna, round.MyAntenna},
		{"MyCQZone", want.MyCQZone, round.MyCQZone},
		{"MyITUZone", want.MyITUZone, round.MyITUZone},
		{"MyDXCC", want.MyDXCC, round.MyDXCC},
		{"SOTARef", want.SOTARef, round.SOTARef},
		{"POTARef", want.POTARef, round.POTARef},
		{"WWFFRef", want.WWFFRef, round.WWFFRef},
		{"IOTA", want.IOTA, round.IOTA},
		{"SIG", want.SIG, round.SIG},
		{"SIGInfo", want.SIGInfo, round.SIGInfo},
		{"MySOTARef", want.MySOTARef, round.MySOTARef},
		{"MyPOTARef", want.MyPOTARef, round.MyPOTARef},
		{"MyWWFFRef", want.MyWWFFRef, round.MyWWFFRef},
		{"MySIG", want.MySIG, round.MySIG},
		{"MySIGInfo", want.MySIGInfo, round.MySIGInfo},
		{"CQZone", want.CQZone, round.CQZone},
		{"ITUZone", want.ITUZone, round.ITUZone},
		{"DXCC", want.DXCC, round.DXCC},
		{"STXString", want.STXString, round.STXString},
		{"SRXString", want.SRXString, round.SRXString},
		{"ContestADIFID", want.ContestADIFID, round.ContestADIFID},
		{"Source", "import", round.Source},
	}
	for _, c := range strChecks {
		if c.want != c.got {
			t.Errorf("round-trip %s = %q, want %q", c.name, c.got, c.want)
		}
	}

	// Numeric fields.
	numChecks := []struct {
		name      string
		want, got float64
	}{
		{"Freq", want.Freq, round.Freq},
		{"FreqRx", want.FreqRx, round.FreqRx},
		{"Distance", want.Distance, round.Distance},
		{"Bearing", want.Bearing, round.Bearing},
	}
	for _, c := range numChecks {
		if c.want != c.got {
			t.Errorf("round-trip %s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if round.STX != want.STX || round.SRX != want.SRX {
		t.Errorf("round-trip STX/SRX = %d/%d, want %d/%d", round.STX, round.SRX, want.STX, want.SRX)
	}
}

// TestListQSOsPageAfterTx_KeysetPages verifies the keyset cursor pages
// strictly after the previous page in the same order ListQSOsPage uses —
// both descending and ascending.
func TestListQSOsPageAfterTx_KeysetPages(t *testing.T) {
	db := newTempDB(t)
	for i := 1; i <= 5; i++ {
		mustInsertQSO(t, db, &qso.QSO{
			Call: fmt.Sprintf("C%d", i), QSODate: "2024050" + string(rune('0'+i)),
			TimeOn: "120000", Band: "20m", Mode: "SSB",
		})
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback()

	// Descending: ids in date order 5,4,3,2,1.
	var got []int64
	var cursor *qso.QSO
	for {
		rows, err := ListQSOsPageAfterTx(tx, 2, "", false, cursor)
		if err != nil {
			t.Fatalf("page desc: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			got = append(got, r.ID)
		}
		cursor = &rows[len(rows)-1]
		if len(rows) < 2 {
			break
		}
	}
	wantDesc := []int64{5, 4, 3, 2, 1}
	if len(got) != len(wantDesc) {
		t.Fatalf("desc pages = %v, want %v", got, wantDesc)
	}
	for i := range wantDesc {
		if got[i] != wantDesc[i] {
			t.Errorf("desc page[%d] = %d, want %d (full: %v)", i, got[i], wantDesc[i], got)
		}
	}

	// Ascending: ids 1..5.
	got = got[:0]
	cursor = nil
	for {
		rows, err := ListQSOsPageAfterTx(tx, 2, "", true, cursor)
		if err != nil {
			t.Fatalf("page asc: %v", err)
		}
		if len(rows) == 0 {
			break
		}
		for _, r := range rows {
			got = append(got, r.ID)
		}
		cursor = &rows[len(rows)-1]
		if len(rows) < 2 {
			break
		}
	}
	wantAsc := []int64{1, 2, 3, 4, 5}
	for i := range wantAsc {
		if i >= len(got) || got[i] != wantAsc[i] {
			t.Fatalf("asc pages = %v, want %v", got, wantAsc)
		}
	}
}

// TestExportQSOsSnapshot_ConsistentDuringInsert reproduces the export race:
// a QSO inserted while the export is streaming must not duplicate or omit
// any record — the snapshot transaction freezes the view, and the keyset
// cursor never shifts.
func TestExportQSOsSnapshot_ConsistentDuringInsert(t *testing.T) {
	db := newTempDB(t)
	for i := 1; i <= 5; i++ {
		mustInsertQSO(t, db, &qso.QSO{
			Call: fmt.Sprintf("C%d", i), QSODate: "2024050" + string(rune('0'+i)),
			TimeOn: "120000", Band: "20m", Mode: "SSB",
		})
	}

	inserted := false
	seen := map[int64]int{}
	var ids []int64
	err := ExportQSOsSnapshot(db, "", false, func(q qso.QSO) error {
		if !inserted {
			inserted = true
			// Concurrent insert from a separate connection while the
			// snapshot stream is mid-flight (e.g. WSJT-X logging).
			if _, err := InsertQSO(db, &qso.QSO{
				Call: "NEW", QSODate: "20240509", TimeOn: "130000",
				Band: "20m", Mode: "SSB",
			}); err != nil {
				return fmt.Errorf("concurrent insert: %w", err)
			}
		}
		seen[q.ID]++
		ids = append(ids, q.ID)
		return nil
	})
	if err != nil {
		t.Fatalf("ExportQSOsSnapshot: %v", err)
	}

	if len(ids) != 5 {
		t.Fatalf("exported %d rows, want 5 (%v)", len(ids), ids)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("id %d exported %d times, want exactly once", id, n)
		}
	}
	// The snapshot must not contain the concurrently inserted row.
	var newID int64
	if err := db.QueryRow(`SELECT id FROM qsos WHERE call='NEW'`).Scan(&newID); err != nil {
		t.Fatalf("find inserted row: %v", err)
	}
	if seen[newID] != 0 {
		t.Errorf("concurrently inserted row leaked into the snapshot")
	}
	// Order must match the descending export convention.
	if ids[0] != 5 || ids[4] != 1 {
		t.Errorf("export order = %v, want [5 4 3 2 1]", ids)
	}
}
