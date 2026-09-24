package store

import (
	"testing"

	"github.com/szporwolik/cqops/internal/qso"
)

func TestSearchQSOs(t *testing.T) {
	db := newTempDB(t)

	mustInsertQSO(t, db, &qso.QSO{Call: "SP9MOA", Name: "Adam", Country: "Poland", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	mustInsertQSO(t, db, &qso.QSO{Call: "K1ABC", Name: "Bob", Country: "United States", QSODate: "20240502", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "contest-a"})
	mustInsertQSO(t, db, &qso.QSO{Call: "DL2XYZ", Name: "Carla", Country: "Germany", QSODate: "20240503", TimeOn: "120000", Band: "20m", Mode: "SSB", ContestID: "contest-b"})

	tests := []struct {
		name      string
		query     string
		contestID string
		wantCalls []string
	}{
		{"call match", "9mo", "", []string{"SP9MOA"}},
		{"name match", "bob", "", []string{"K1ABC"}},
		{"country match", "germ", "", []string{"DL2XYZ"}},
		{"no match", "zzzz", "", nil},
		{"contest scoped", "9mo", "contest-a", nil},
		{"contest scoped match", "1abc", "contest-a", []string{"K1ABC"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SearchQSOs(db, tt.query, tt.contestID, 100)
			if err != nil {
				t.Fatalf("SearchQSOs: %v", err)
			}
			if len(got) != len(tt.wantCalls) {
				t.Fatalf("got %d results, want %d", len(got), len(tt.wantCalls))
			}
			for i, want := range tt.wantCalls {
				if got[i].Call != want {
					t.Errorf("result %d call = %q, want %q", i, got[i].Call, want)
				}
			}
		})
	}
}

func TestSearchQSOsLimit(t *testing.T) {
	db := newTempDB(t)
	for i := 0; i < 5; i++ {
		mustInsertQSO(t, db, &qso.QSO{Call: "N1TEST", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB"})
	}
	got, err := SearchQSOs(db, "n1te", "", 3)
	if err != nil {
		t.Fatalf("SearchQSOs: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d results, want 3 (limit)", len(got))
	}
}

func TestWavelogIDRoundTrip(t *testing.T) {
	db := newTempDB(t)

	id := mustInsertQSO(t, db, &qso.QSO{Call: "SP9MOA", QSODate: "20240501", TimeOn: "120000", Band: "20m", Mode: "SSB", WavelogID: 42})

	got, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID: %v", err)
	}
	if got.WavelogID != 42 {
		t.Errorf("WavelogID = %d, want 42", got.WavelogID)
	}

	// UpdateQSO must PRESERVE the remote id — a normal field save never
	// writes synchronization metadata (the id is only attached via
	// SetWavelogID*).
	got.Comment = "edited"
	got.WavelogID = 0 // a stale form snapshot must not wipe the id
	if err := UpdateQSO(db, got); err != nil {
		t.Fatalf("UpdateQSO: %v", err)
	}
	again, err := GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID after update: %v", err)
	}
	if again.WavelogID != 42 {
		t.Errorf("WavelogID after update = %d, want 42 (preserved)", again.WavelogID)
	}
	// Changing the remote link goes through SetWavelogID.
	if err := SetWavelogID(db, id, 99); err != nil {
		t.Fatalf("SetWavelogID: %v", err)
	}
	again, err = GetQSOByID(db, id)
	if err != nil {
		t.Fatalf("GetQSOByID after SetWavelogID: %v", err)
	}
	if again.WavelogID != 99 {
		t.Errorf("WavelogID after SetWavelogID = %d, want 99", again.WavelogID)
	}
}
