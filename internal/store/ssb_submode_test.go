package store

import (
	"path/filepath"
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

// An SSB contact stored without a submode must come back with its sideband —
// the ADIF/spot/edit paths all funnel through insertQSO and the update
// statement, so a blank sideband can no longer reach the table.
func TestSSBSubmodeIsFilledOnWrite(t *testing.T) {
	db, err := InitDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	cases := []struct {
		call, freq, want string
		f                float64
	}{
		{"DL1ABC", "7.1968", "LSB", 7.1968},
		{"DL2ABC", "14.2500", "USB", 14.25},
	}
	for _, tc := range cases {
		q := &qso.QSO{Call: tc.call, QSODate: "20261025", TimeOn: "193000", Band: "40m", Mode: "SSB"}
		q.Freq = tc.f
		id, err := InsertQSO(db, q)
		if err != nil {
			t.Fatalf("InsertQSO %s: %v", tc.call, err)
		}
		stored, err := GetQSOByID(db, id)
		if err != nil {
			t.Fatalf("GetQSOByID: %v", err)
		}
		if stored.Submode != tc.want {
			t.Errorf("%s: stored submode = %q, want %q", tc.call, stored.Submode, tc.want)
		}
	}

	// An edit that loses the submode puts it back instead of blanking the row.
	q := &qso.QSO{Call: "DL3ABC", QSODate: "20261025", TimeOn: "194000", Band: "20m", Mode: "SSB", Freq: 14.2}
	id, err := InsertQSO(db, q)
	if err != nil {
		t.Fatalf("InsertQSO: %v", err)
	}
	q.Submode = ""
	if err := UpdateQSO(db, q); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	stored, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if stored.Submode != "USB" {
		t.Errorf("after edit stored submode = %q, want USB", stored.Submode)
	}
}
