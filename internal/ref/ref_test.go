package ref

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// Basic tests
// =============================================================================

func TestOpenClose(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	if n, _ := db.Count(); n != 0 {
		t.Errorf("expected 0 refs, got %d", n)
	}
}

func TestOpen_InvalidPath(t *testing.T) {
	if _, err := Open("/nonexistent/path/should/fail/ref.db"); err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestSchema(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	cols := EnsureColumns(db.UnderlyingDB())
	expected := map[string]bool{"ref_type": true, "ref": true, "name": true, "grid": true, "height": true, "is_group": true, "search": true}
	for _, c := range cols {
		if !expected[c] {
			t.Errorf("unexpected column %q", c)
		}
		delete(expected, c)
	}
	for c := range expected {
		t.Errorf("missing column %q", c)
	}
}

func TestLookup_NotFound(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	if _, ok := db.Lookup(RefSOTA, "XX/XX-999"); ok {
		t.Error("expected not found")
	}
}

func TestSearch_NoResults(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	r, err := db.Search("nonexistent_xyz_123")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(r) != 0 {
		t.Errorf("expected 0, got %d", len(r))
	}
}

// =============================================================================
// SOTA import tests
// =============================================================================

func TestImportSOTA_Valid(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	csv := "title\nSummitCode,x,x,SummitName,AltM,x,x,x,Lon,Lat,x,x\nG/SP-001,x,x,BenNevis,1344,x,x,x,-5.0036,56.7968,x,x\n"
	path := writeTemp(t, "sota.csv", csv)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, err := importSOTA(tx, path)
	if err != nil {
		t.Fatalf("importSOTA: %v", err)
	}
	tx.Commit()
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
	r, ok := db.Lookup(RefSOTA, "G/SP-001")
	if !ok || r.Name != "BenNevis" || r.Height != 1344 || r.Grid == "" {
		t.Errorf("name=%q height=%d grid=%q", r.Name, r.Height, r.Grid)
	}
}

func TestImportSOTA_MissingFile(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	if _, err := importSOTA(tx, "/nonexistent/sota.csv"); err == nil {
		t.Error("expected error")
	}
}

func TestImportSOTA_EmptyFile(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "sota.csv", "")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	if _, err := importSOTA(tx, path); err == nil {
		t.Error("expected error for empty file")
	}
}

func TestImportSOTA_MalformedRows(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "sota.csv", "title\nheader\n,,,,\n")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importSOTA(tx, path)
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestImportSOTA_SkipsBlankRef(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "sota.csv", "title\nhdr,x,x,x,AltM,x,x,x,x,Lon,Lat,x,x\n,Region,x,x,Summit,100,x,x,x,-5.0,56.0,x,x\n")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importSOTA(tx, path)
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

// =============================================================================
// POTA import tests
// =============================================================================

func TestImportPOTA_Valid(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "pota.csv", `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"US-0001","Acadia","1","1","US-ME","44.31","-68.2034","FN54vh"
`)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importPOTA(tx, path)
	tx.Commit()
	if n != 1 {
		t.Errorf("expected 1, got %d", n)
	}
	r, _ := db.Lookup(RefPOTA, "US-0001")
	if r.Grid == "" {
		t.Error("grid empty")
	}
}

func TestImportPOTA_SkipsInactive(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "pota.csv", `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"A-1","Active","1","1","X","44.0","-68.0","FN54"
"A-2","Inactive","0","1","X","44.0","-68.0","FN54"
`)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importPOTA(tx, path)
	if n != 1 {
		t.Errorf("expected 1 active, got %d", n)
	}
}

func TestImportPOTA_GridFallback(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "pota.csv", `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"K-0001","Test","1","1","X","51.5","-0.1",""
`)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importPOTA(tx, path)
	tx.Commit()
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	r, _ := db.Lookup(RefPOTA, "K-0001")
	if !strings.HasPrefix(r.Grid, "IO91") || len(r.Grid) < 4 {
		t.Errorf("expected IO91 prefix from lat/lon, got %q", r.Grid)
	}
}

// =============================================================================
// WWFF import tests
// =============================================================================

func TestImportWWFF_Valid(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "wwff.csv", "reference,status,name,program,dxcc,state,county,continent,iota,iaruLocator,latitude,longitude,x,x\n1SFF-0001,active,Spratly,1SFF,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\n")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importWWFF(tx, path)
	tx.Commit()
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	r, _ := db.Lookup(RefWWFF, "1SFF-0001")
	if r.Name != "Spratly" || r.Grid == "" {
		t.Errorf("name=%q grid=%q", r.Name, r.Grid)
	}
}

func TestImportWWFF_SkipsNonActive(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	path := writeTemp(t, "wwff.csv", "reference,status,name,program,dxcc,state,county,continent,iota,iaruLocator,latitude,longitude,x,x\nA-1,active,One,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\nA-2,deleted,Two,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\nA-3,national,Three,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\n")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importWWFF(tx, path)
	if n != 1 {
		t.Errorf("expected 1 active, got %d", n)
	}
}

// =============================================================================
// IOTA import tests
// =============================================================================

func TestImportIOTA_Valid(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	isl := writeTemp(t, "iota_islands.json", `[{"refno":"AF-001","name":"Agalega Islands"},{"refno":"AF-001","name":"North Island"},{"refno":"AF-002","name":"Amsterdam"}]`)
	grp := writeTemp(t, "iota_groups.json", `[{"refno":"AF-001","name":"Agalega Islands","latitude_max":"-10.0","latitude_min":"-11.0","longitude_max":"57.0","longitude_min":"56.0"},{"refno":"AF-002","name":"Amsterdam & St Paul","latitude_max":"-7.0","latitude_min":"-8.0","longitude_max":"52.0","longitude_min":"51.0"}]`)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, err := importIOTA(tx, isl, grp)
	if err != nil {
		t.Fatalf("importIOTA: %v", err)
	}
	tx.Commit()
	if n != 4 {
		t.Fatalf("expected 4, got %d", n)
	}
	r, _ := db.Lookup(RefIOTA, "AF-001")
	if r.Name != "Agalega Islands" || r.Grid == "" {
		t.Errorf("name=%q grid=%q", r.Name, r.Grid)
	}
}

func TestImportIOTA_InvalidJSON(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	isl := writeTemp(t, "iota_islands.json", "not json")
	grp := writeTemp(t, "iota_groups.json", "[]")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	if _, err := importIOTA(tx, isl, grp); err == nil {
		t.Error("expected parse error")
	}
}

func TestImportIOTA_EmptyArray(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	isl := writeTemp(t, "iota_islands.json", "[]")
	grp := writeTemp(t, "iota_groups.json", "[]")
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importIOTA(tx, isl, grp)
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestImportIOTA_NoCoordinates(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	isl := writeTemp(t, "iota_islands.json", `[{"refno":"XX-001","name":"NoCoords"}]`)
	grp := writeTemp(t, "iota_groups.json", `[{"refno":"XX-001","name":"NoCoords"}]`)
	tx, _ := db.UnderlyingDB().Begin()
	defer tx.Rollback()
	n, _ := importIOTA(tx, isl, grp)
	tx.Commit()
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	r, _ := db.Lookup(RefIOTA, "XX-001")
	if r.Grid != "" {
		t.Errorf("expected empty grid, got %q", r.Grid)
	}
}

// =============================================================================
// Download tests (httptest)
// =============================================================================

func TestDownloadFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	if err := downloadFile(srv.URL, path, func(string) {}); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != "hello" {
		t.Errorf("got %q", string(data))
	}
}

func TestDownloadFile_HTTP500_Retries(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	if err := downloadFile(srv.URL, path, func(string) {}); err == nil {
		t.Error("expected error")
	}
	if attempts < 2 {
		t.Errorf("expected >=2 attempts, got %d", attempts)
	}
}

func TestDownloadFile_EventualSuccess(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	if err := downloadFile(srv.URL, path, func(string) {}); err != nil {
		t.Fatalf("expected success: %v", err)
	}
}

func TestDownloadFile_UsesCache(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("server should not be called (cached)")
	}))
	defer srv.Close()
	dir := t.TempDir()
	path := filepath.Join(dir, "cached.csv")
	os.WriteFile(path, []byte("cached"), 0644)
	if err := downloadFile(srv.URL, path, func(string) {}); err != nil {
		t.Fatalf("downloadFile: %v", err)
	}
}

func TestRebuild_EmptyCacheDir(t *testing.T) {
	db := openDB(t)
	defer db.Close()
	if _, err := db.Rebuild("", func(string) {}); err == nil {
		t.Error("expected error for empty cacheDir")
	}
}

// =============================================================================
// Rebuild integration
// =============================================================================

func TestRebuild_AllSources(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ref.db")
	cache := filepath.Join(dir, "cache")
	os.MkdirAll(cache, 0755)

	writeCSV(t, filepath.Join(cache, sotaFile), "title\nSummitCode,x,x,SummitName,AltM,x,x,x,Lon,Lat,x,x\nG/SP-001,x,x,Summit,100,x,x,x,-5.0,56.0,x,x\n")
	writeCSV(t, filepath.Join(cache, potaFile), `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"US-0001","Park","1","1","X","44.0","-68.0","FN54"
`)
	writeCSV(t, filepath.Join(cache, wwffFile), "reference,status,name,program,dxcc,state,county,continent,iota,iaruLocator,latitude,longitude,x,x\nX-0001,active,Area,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\n")
	writeJSON(t, filepath.Join(cache, iotaFile), []map[string]string{{"refno": "AF-001", "name": "Agalega"}})
	writeJSON(t, filepath.Join(cache, iotaGroupFile), []map[string]string{{"refno": "AF-001", "name": "Agalega Islands", "latitude_max": "-10.0", "latitude_min": "-11.0", "longitude_max": "57.0", "longitude_min": "56.0"}})

	db, _ := Open(dbPath)
	defer db.Close()
	total, _ := db.Rebuild(cache, func(string) {})
	if total != 5 {
		t.Errorf("expected 5, got %d", total)
	}
}

func TestRebuildIfMissing_SkipsWhenPopulated(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ref.db")
	cache := filepath.Join(dir, "cache")
	os.MkdirAll(cache, 0755)

	writeCSV(t, filepath.Join(cache, sotaFile), "title\nSummitCode,x,x,SummitName,AltM,x,x,x,Lon,Lat,x,x\nG/SP-001,x,x,S,100,x,x,x,-5.0,56.0,x,x\n")
	writeCSV(t, filepath.Join(cache, potaFile), `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"US-0001","P","1","1","X","44.0","-68.0","FN54"
`)
	writeCSV(t, filepath.Join(cache, wwffFile), "reference,status,name,program,dxcc,state,county,continent,iota,iaruLocator,latitude,longitude,x,x\nX-0001,active,A,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\n")
	writeJSON(t, filepath.Join(cache, iotaFile), []map[string]string{{"refno": "AF-001", "name": "I"}})
	writeJSON(t, filepath.Join(cache, iotaGroupFile), []map[string]string{{"refno": "AF-001", "name": "Agalega Islands", "latitude_max": "-10.0", "latitude_min": "-11.0", "longitude_max": "57.0", "longitude_min": "56.0"}})

	db, _ := Open(dbPath)
	defer db.Close()
	db.Rebuild(cache, func(string) {})
	if err := db.RebuildIfMissing(cache, func(string) {}); err != nil {
		t.Fatalf("RebuildIfMissing: %v", err)
	}
	if n, _ := db.Count(); n != 5 {
		t.Errorf("expected 5, got %d", n)
	}
}

func TestSearch_AfterRebuild(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(dir, "cache")
	os.MkdirAll(cache, 0755)

	writeCSV(t, filepath.Join(cache, sotaFile), "title\nSummitCode,x,x,SummitName,AltM,x,x,x,Lon,Lat,x,x\nG/SP-001,x,x,BenNevis,1344,x,x,x,-5.0,56.0,x,x\nGM/WS-001,x,x,GoatFell,874,x,x,x,-5.0,55.0,x,x\n")
	writeCSV(t, filepath.Join(cache, potaFile), `"reference","name","active","entityId","locationDesc","latitude","longitude","grid"
"US-0001","Park","1","1","X","44.0","-68.0","FN54"
`)
	writeCSV(t, filepath.Join(cache, wwffFile), "reference,status,name,program,dxcc,state,county,continent,iota,iaruLocator,latitude,longitude,x,x\nX-0001,active,Area,x,x,x,x,x,-,OJ58XO,8.6,111.9,x,x\n")
	writeJSON(t, filepath.Join(cache, iotaFile), []map[string]string{{"refno": "AF-001", "name": "Island"}})
	writeJSON(t, filepath.Join(cache, iotaGroupFile), []map[string]string{{"refno": "AF-001", "name": "Agalega Islands", "latitude_max": "-10.0", "latitude_min": "-11.0", "longitude_max": "57.0", "longitude_min": "56.0"}})

	db, _ := Open(filepath.Join(dir, "ref.db"))
	defer db.Close()
	db.Rebuild(cache, func(string) {})

	r, _ := db.Search("G/SP-001")
	if len(r) != 1 || r[0].Name != "BenNevis" {
		t.Errorf("search ref: %+v", r)
	}
	r, _ = db.Search("nevis")
	if len(r) != 1 {
		t.Errorf("search name: got %d results", len(r))
	}
	r, _ = db.Search("e")
	if len(r) > 500 {
		t.Errorf("limit exceeded: %d", len(r))
	}
}

func TestNormalizeForSearch(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Ćwilin", "cwilin"},
		{"ćwilin", "cwilin"},
		{"Cwilin", "cwilin"},
		{"G/SP-001", "g/sp-001"},
		{"Österreich", "osterreich"},
		{"Åland", "aland"},
		{"hello", "hello"},
		{"", ""},
	}
	for _, tt := range tests {
		got := normalizeForSearch(tt.in)
		if got != tt.want {
			t.Errorf("normalizeForSearch(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// =============================================================================
// Grid calculation
// =============================================================================

func TestGridFromLatLon(t *testing.T) {
	tests := []struct{ lat, lon, wantPrefix string }{
		{"51.5", "-0.1", "IO91"},   // London
		{"40.7", "-74.0", "FN30"},  // New York
		{"-33.9", "151.2", "QF56"}, // Sydney
	}
	for _, tt := range tests {
		got := gridFromLatLon(tt.lat, tt.lon)
		if !strings.HasPrefix(got, tt.wantPrefix) {
			t.Errorf("gridFromLatLon(%q,%q)=%q want prefix %q", tt.lat, tt.lon, got, tt.wantPrefix)
		}
		if len(got) < 4 {
			t.Errorf("gridFromLatLon(%q,%q)=%q too short", tt.lat, tt.lon, got)
		}
	}
	// Invalid inputs
	empty := []struct{ lat, lon string }{{"", "10"}, {"abc", "10"}}
	for _, tt := range empty {
		if got := gridFromLatLon(tt.lat, tt.lon); got != "" {
			t.Errorf("gridFromLatLon(%q,%q)=%q want empty", tt.lat, tt.lon, got)
		}
	}
}

// =============================================================================
// Helpers
// =============================================================================

func openDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ref.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return db
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeCSV(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(content, "\r", "")), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}
