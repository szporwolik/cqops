package tui

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/szporwolik/cqops/internal/ctybig"
)

// =============================================================================
// Reference-data refresh tests — bounded HTTP, capped expansion, message
// results. No real network.
// =============================================================================

// makeBigCTYZip builds an in-memory zip containing cty.csv with the given
// content.
func makeBigCTYZip(t *testing.T, csv string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("cty.csv")
	if err != nil {
		t.Fatalf("zip create: %v", err)
	}
	if _, err := w.Write([]byte(csv)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestDownloadBigCTY_HonorsContext(t *testing.T) {
	release := make(chan struct{})
	defer close(release)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
			w.Write([]byte("never"))
		case <-time.After(2 * time.Second):
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := downloadBigCTY(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected error for a stalled download")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("download took %v, want a prompt context-cancelled failure", elapsed)
	}
}

func TestDownloadBigCTY_CapsExpandedCSV(t *testing.T) {
	old := maxBigCTYCSVBytes
	maxBigCTYCSVBytes = 1024
	t.Cleanup(func() { maxBigCTYCSVBytes = old })

	// 2KB of incompressible data declares a >cap expanded size.
	csv := strings.Repeat("A", 2048)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(makeBigCTYZip(t, csv))
	}))
	defer srv.Close()

	_, err := downloadBigCTY(context.Background(), srv.URL)
	if err == nil || !strings.Contains(err.Error(), "exceeding the 1024 byte cap") {
		t.Fatalf("err = %v, want expanded-size cap rejection", err)
	}
}

func TestDownloadBigCTY_ExtractsCSV(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(makeBigCTYZip(t, "SP,Poland,269,EU,15,28,52.0,19.0,-1,SP1;\n"))
	}))
	defer srv.Close()

	bf, err := downloadBigCTY(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("downloadBigCTY: %v", err)
	}
	if !strings.Contains(string(bf.ctyCSV), "Poland") {
		t.Errorf("ctyCSV = %q, want extracted CSV", bf.ctyCSV)
	}
}

func TestFindBigCTYURL_StallingServerReturnsEmpty(t *testing.T) {
	oldPage := bigCTYCatalogPage
	oldClient := refHTTPClient
	bigCTYCatalogPage = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	})).URL
	refHTTPClient = &http.Client{Timeout: 100 * time.Millisecond}
	t.Cleanup(func() {
		bigCTYCatalogPage = oldPage
		refHTTPClient = oldClient
	})

	start := time.Now()
	if url := findBigCTYURL(context.Background()); url != "" {
		t.Errorf("url = %q, want empty on stalled catalog", url)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("catalog fetch took %v, want a prompt timeout", elapsed)
	}
}

func TestRunRefDataRefresh_DownloadsAndReturnsData(t *testing.T) {
	oldPage := bigCTYCatalogPage
	oldRe := bigCTYZipRe
	t.Cleanup(func() {
		bigCTYCatalogPage = oldPage
		bigCTYZipRe = oldRe
	})

	var zipURL string
	zipBytes := makeBigCTYZip(t, "SP,Poland,269,EU,15,28,52.0,19.0,-1,SP1;\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/catalog":
			fmt.Fprintf(w, `<a href="%s">bigcty-20260922.zip</a>`, zipURL)
		case "/bigcty.zip":
			w.Write(zipBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	zipURL = srv.URL + "/bigcty.zip"
	bigCTYCatalogPage = srv.URL + "/catalog"
	bigCTYZipRe = regexp.MustCompile(regexp.QuoteMeta(srv.URL) + `/bigcty\.zip`)

	cacheDir := t.TempDir()
	out := runRefDataRefresh(cacheDir, true, false, false, false)
	if out.bigCTY == nil {
		t.Fatal("bigCTY should be loaded after refresh")
	}
	if out.bigCTY.Find("SP1ABC") == nil {
		t.Error("loaded Big CTY should resolve SP1ABC")
	}

	// The CSV must be cached for the next startup.
	if _, err := os.Stat(filepath.Join(cacheDir, "cty.csv")); err != nil {
		t.Errorf("cty.csv not cached: %v", err)
	}
}

func TestRunRefDataRefresh_LoadsCachedCSV(t *testing.T) {
	cacheDir := t.TempDir()
	csv := "SP,Poland,269,EU,15,28,52.0,19.0,-1,SP1;\n"
	if err := os.WriteFile(filepath.Join(cacheDir, "cty.csv"), []byte(csv), 0o644); err != nil {
		t.Fatal(err)
	}
	// Point the catalog at a stalling server: the cache must win without a
	// hang when the download is impossible.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()
	oldPage := bigCTYCatalogPage
	oldClient := refHTTPClient
	bigCTYCatalogPage = srv.URL
	refHTTPClient = &http.Client{Timeout: 100 * time.Millisecond}
	t.Cleanup(func() {
		bigCTYCatalogPage = oldPage
		refHTTPClient = oldClient
	})

	out := runRefDataRefresh(cacheDir, true, false, false, false)
	if out.bigCTY == nil {
		t.Fatal("cached CSV should be loaded even when the catalog stalls")
	}
	if out.bigCTY.Find("SP1ABC") == nil {
		t.Error("cached Big CTY should resolve SP1ABC")
	}
}

// TestRefDataMsgInstall verifies the main-loop handler installs the worker's
// result and never touches App from the worker side.
func TestRefDataMsgInstall(t *testing.T) {
	db, err := ctybig.ParseCSV(strings.NewReader("SP,Poland,269,EU,15,28,52.0,19.0,-1,SP1;\n"))
	if err != nil {
		t.Fatalf("ParseCSV: %v", err)
	}

	m := newLifecycleTestModel(t)
	if m.App == nil {
		t.Fatal("App is nil")
	}
	m.App.Config.General.UseCTY = true

	handled, cmd := m.handleAsyncMessages(refDataMsg{bigCTY: db})
	if !handled {
		t.Error("refDataMsg should be consumed")
	}
	if m.App.BigCTY != db {
		t.Error("Big CTY should be installed on App by the handler")
	}
	_ = cmd
}
