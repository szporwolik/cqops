package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adif "github.com/farmergreg/adif/v5"
	"github.com/farmergreg/spec/v6/adifield"
	"github.com/szporwolik/cqops/internal/qso"
	"github.com/szporwolik/cqops/internal/store"
)

func writeTempADIF(t *testing.T, adifStr string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "download.adif")
	if err := os.WriteFile(path, []byte(adifStr), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func importADIFFromPath(t *testing.T, dbPath, adifPath string) (inserted, dupes, failed int) {
	t.Helper()

	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()

	f, err := os.Open(adifPath)
	if err != nil {
		t.Fatalf("Open ADIF: %v", err)
	}
	defer f.Close()

	scanner := adif.NewScanner(f)
	for scanner.Scan() {
		if scanner.IsHeader() {
			continue
		}
		r := scanner.Record()
		qs := qso.NewQSO()
		if v := r[adifield.CALL]; v != "" {
			qs.Call = strings.ToUpper(v)
		}
		if v := r[adifield.BAND]; v != "" {
			qs.Band = qso.NormalizeBand(v)
		}
		if v := r[adifield.MODE]; v != "" {
			qs.Mode = strings.ToUpper(v)
		}
		if v := r[adifield.SUBMODE]; v != "" {
			qs.Submode = strings.ToUpper(v)
		}
		if v := r[adifield.QSO_DATE]; v != "" {
			qs.QSODate = v
		}
		if v := r[adifield.TIME_ON]; v != "" {
			qs.TimeOn = v
		}
		if v := r[adifield.FREQ]; v != "" {
			fmt.Sscanf(v, "%f", &qs.Freq)
		}
		if v := r[adifield.RST_SENT]; v != "" {
			qs.RSTSent = v
		}
		if v := r[adifield.RST_RCVD]; v != "" {
			qs.RSTRcvd = v
		}
		if v := r[adifield.GRIDSQUARE]; v != "" {
			qs.GridSquare = v
		}
		if v := r[adifield.NAME]; v != "" {
			qs.Name = v
		}
		if v := r[adifield.QTH]; v != "" {
			qs.QTH = v
		}
		if v := r[adifield.COMMENT]; v != "" {
			qs.Comment = v
		}
		qs.Source = "wavelog"
		qs.WavelogUploaded = "yes"

		if err := qso.ValidateImportRecord(qs); err != nil {
			failed++
			continue
		}

		if existingID := store.FindQSOByKey(db, qs.Call, qs.Band, qs.Mode, qs.QSODate, qs.TimeOn); existingID != 0 {
			dupes++
			continue
		}

		_, err := store.InsertQSO(db, qs)
		if err != nil {
			failed++
		} else {
			inserted++
		}
	}
	return
}

func TestImportADIF_SingleQSO(t *testing.T) {
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <FREQ:7>14.2500 <MODE:3>SSB
<QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59
<GRIDSQUARE:4>JO90 <NAME:4>John <QTH:6>Krakow <COMMENT:6>73 GL! <EOR>`

	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, dupes, failed := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1", inserted)
	}
	if dupes != 0 || failed != 0 {
		t.Errorf("dupes=%d failed=%d, want 0/0", dupes, failed)
	}

	db, err := store.InitDB(dbPath)
	if err != nil {
		t.Fatalf("re-open DB: %v", err)
	}
	defer db.Close()
	qsos, err := store.ListQSOs(db, 1)
	if err != nil {
		t.Fatalf("ListQSOs: %v", err)
	}
	if len(qsos) != 1 {
		t.Fatalf("expected 1 QSO, got %d", len(qsos))
	}
	q := qsos[0]
	if q.Call != "SP9MOA" {
		t.Errorf("Call = %q", q.Call)
	}
	if q.Band != "20m" {
		t.Errorf("Band = %q", q.Band)
	}
	if q.Mode != "SSB" {
		t.Errorf("Mode = %q", q.Mode)
	}
	if q.GridSquare != "JO90" {
		t.Errorf("GridSquare = %q", q.GridSquare)
	}
	if q.Name != "John" {
		t.Errorf("Name = %q", q.Name)
	}
	if q.Source != "wavelog" {
		t.Errorf("Source = %q, want wavelog", q.Source)
	}
	if q.WavelogUploaded != "yes" {
		t.Errorf("WavelogUploaded = %q, want yes", q.WavelogUploaded)
	}
}

func TestImportADIF_MultipleQSOs(t *testing.T) {
	adifStr := `<CALL:6>SP9AAA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>
<CALL:6>SP9BBB <BAND:3>40m <MODE:2>CW <QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>599 <RST_RCVD:3>579 <EOR>
<CALL:6>SP9CCC <BAND:3>15m <MODE:3>FT8 <FREQ:8>21.074550 <QSO_DATE:8>20260618 <TIME_ON:6>140000 <RST_SENT:3>-05 <RST_RCVD:3>+02 <EOR>`

	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, dupes, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 3 {
		t.Errorf("inserted = %d, want 3", inserted)
	}
	if dupes != 0 {
		t.Errorf("dupes = %d, want 0", dupes)
	}
}

func TestImportADIF_DuplicateDetection(t *testing.T) {
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`

	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, dupes, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 1 || dupes != 0 {
		t.Fatalf("first import: inserted=%d dupes=%d", inserted, dupes)
	}

	inserted2, dupes2, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted2 != 0 {
		t.Errorf("second import: inserted=%d, want 0 (duplicate)", inserted2)
	}
	if dupes2 != 1 {
		t.Errorf("second import: dupes=%d, want 1", dupes2)
	}
}

func TestImportADIF_OneDuplicateOneNew(t *testing.T) {
	adifStr := `<CALL:6>SP9AAA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>
<CALL:6>SP9BBB <BAND:3>40m <MODE:2>CW <QSO_DATE:8>20260618 <TIME_ON:6>130000 <RST_SENT:3>599 <RST_RCVD:3>579 <EOR>`

	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, _, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 2 {
		t.Fatalf("first import: inserted=%d, want 2", inserted)
	}

	adifStr2 := `<CALL:6>SP9AAA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>
<CALL:6>SP9CCC <BAND:3>15m <MODE:3>FT8 <QSO_DATE:8>20260618 <TIME_ON:6>140000 <RST_SENT:3>-05 <RST_RCVD:3>+02 <EOR>`

	adifPath2 := writeTempADIF(t, adifStr2)
	inserted2, dupes2, _ := importADIFFromPath(t, dbPath, adifPath2)
	if inserted2 != 1 {
		t.Errorf("inserted = %d, want 1 (new QSO)", inserted2)
	}
	if dupes2 != 1 {
		t.Errorf("dupes = %d, want 1 (SP9AAA duplicate)", dupes2)
	}
}

func TestImportADIF_EmptyFile(t *testing.T) {
	adifPath := writeTempADIF(t, "")
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, _, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0", inserted)
	}
}

func TestImportADIF_MissingCallsign(t *testing.T) {
	adifStr := `<BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <EOR>`
	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, _, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0 (no call → skipped)", inserted)
	}
}

func TestImportADIF_MalformedNoPanic(t *testing.T) {
	adifStr := `garbage not valid adif data <EOR>`
	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Malformed ADIF with no valid CALL/MODE → rejected by ValidateImportRecord.
	// Should not panic.
	inserted, _, failed := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 0 {
		t.Errorf("inserted = %d, want 0 (malformed ADIF should not insert)", inserted)
	}
	_ = failed // may be 0 or non-zero depending on scanner behavior
}

func TestImportADIF_InvalidFrequency(t *testing.T) {
	// Non-numeric frequency with a valid band: record is accepted (freq stays 0.0,
	// band is provided). The validator does not reject records with zero freq when
	// a valid band is present.
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <FREQ:5>hello <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`
	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, _, failed := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1 (valid band + invalid freq → accepted, freq=0.0)", inserted)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
}

func TestImportADIF_InvalidGrid(t *testing.T) {
	// Invalid grid is now cleared by ValidateImportRecord instead of stored as-is.
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <GRIDSQUARE:4>XXXX <EOR>`
	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, _, failed := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1 (invalid grid is cleared, QSO still accepted)", inserted)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}

	// Verify the grid was cleared.
	db, _ := store.InitDB(dbPath)
	defer db.Close()
	qsos, _ := store.ListQSOs(db, 1)
	if len(qsos) != 1 {
		t.Fatal("no QSO found")
	}
	if qsos[0].GridSquare != "" {
		t.Errorf("GridSquare = %q, want empty (invalid grid cleared)", qsos[0].GridSquare)
	}
}

func TestImportADIF_WavelogStatusSet(t *testing.T) {
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`
	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	importADIFFromPath(t, dbPath, adifPath)

	db, _ := store.InitDB(dbPath)
	defer db.Close()
	qsos, _ := store.ListQSOs(db, 1)
	if len(qsos) != 1 {
		t.Fatal("no QSO found")
	}
	if qsos[0].WavelogUploaded != "yes" {
		t.Errorf("WavelogUploaded = %q, want yes", qsos[0].WavelogUploaded)
	}
	if qsos[0].Source != "wavelog" {
		t.Errorf("Source = %q, want wavelog", qsos[0].Source)
	}
}

func TestImportADIF_RepeatedSameQSOInFile(t *testing.T) {
	adifStr := `<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>
<CALL:6>SP9MOA <BAND:3>20m <MODE:3>SSB <QSO_DATE:8>20260618 <TIME_ON:6>120000 <RST_SENT:2>59 <RST_RCVD:2>59 <EOR>`

	adifPath := writeTempADIF(t, adifStr)
	dbPath := filepath.Join(t.TempDir(), "test.db")

	inserted, dupes, _ := importADIFFromPath(t, dbPath, adifPath)
	if inserted != 1 {
		t.Errorf("inserted = %d, want 1", inserted)
	}
	if dupes != 1 {
		t.Errorf("dupes = %d, want 1 (second occurrence is duplicate)", dupes)
	}
}
