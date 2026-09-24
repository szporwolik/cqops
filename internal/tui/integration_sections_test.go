package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// TestIntegrationMenuSectionsMatchTopPaneOrder pins the section order to the
// top-pane function order: APRS (F3), DXC (F4), PSK (F5), then HTTP Server
// and GPS (no pane).
func TestIntegrationMenuSectionsMatchTopPaneOrder(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.width = 120
	im.height = 60

	lines := strings.Split(im.View().Content, "\n")
	aprs := lineIndex(lines, "APRS:")
	dxc := lineIndex(lines, "DX Cluster:")
	psk := lineIndex(lines, "PSK Reporter:")
	http := lineIndex(lines, "HTTP Server:")
	gps := lineIndex(lines, "GPS Service:")
	if aprs < 0 || dxc < 0 || psk < 0 || http < 0 || gps < 0 {
		t.Fatalf("rows missing: aprs=%d dxc=%d psk=%d http=%d gps=%d", aprs, dxc, psk, http, gps)
	}
	if !(aprs < dxc && dxc < psk && psk < http && http < gps) {
		t.Errorf("section order wrong: APRS=%d DXC=%d PSK=%d HTTP=%d GPS=%d, want APRS < DXC < PSK < HTTP < GPS",
			aprs, dxc, psk, http, gps)
	}
}

// TestIntegrationMenuSectionsFlush pins the section layout: no blank
// separator lines between APRS / DX Cluster / PSK / HTTP / GPS, regardless
// of which sections are expanded.
func TestIntegrationMenuSectionsFlush(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.aprsEnabled = true // expanded section
	im.dxcEnabled = true
	im.width = 120
	im.height = 60

	lines := strings.Split(im.View().Content, "\n")
	testBtn := lineIndex(lines, "[ Test APRS ]")
	dxc := lineIndex(lines, "DX Cluster:")
	login := lineIndex(lines, "Login:")
	psk := lineIndex(lines, "PSK Reporter:")
	http := lineIndex(lines, "HTTP Server:")
	gps := lineIndex(lines, "GPS Service:")
	if testBtn < 0 || dxc < 0 || login < 0 || psk < 0 || http < 0 || gps < 0 {
		t.Fatalf("rows missing: test=%d dxc=%d login=%d psk=%d http=%d gps=%d",
			testBtn, dxc, login, psk, http, gps)
	}
	if dxc != testBtn+1 {
		t.Errorf("blank line(s) after expanded APRS section (test=%d, dxc=%d)", testBtn, dxc)
	}
	if psk != login+1 {
		t.Errorf("blank line(s) after expanded DX Cluster (login=%d, psk=%d)", login, psk)
	}
	if http != psk+1 {
		t.Errorf("blank line(s) after PSK Reporter (psk=%d, http=%d)", psk, http)
	}
	if gps != http+1 {
		t.Errorf("blank line(s) after collapsed HTTP Server (http=%d, gps=%d)", http, gps)
	}
}

// TestIntegrationMenuAllCollapsedFlush pins the collapsed layout: every
// section header follows the previous one directly.
func TestIntegrationMenuAllCollapsedFlush(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.width = 120
	im.height = 60

	lines := strings.Split(im.View().Content, "\n")
	aprs := lineIndex(lines, "APRS:")
	dxc := lineIndex(lines, "DX Cluster:")
	psk := lineIndex(lines, "PSK Reporter:")
	http := lineIndex(lines, "HTTP Server:")
	gps := lineIndex(lines, "GPS Service:")
	if aprs < 0 || dxc < 0 || psk < 0 || http < 0 || gps < 0 {
		t.Fatalf("rows missing: aprs=%d dxc=%d psk=%d http=%d gps=%d", aprs, dxc, psk, http, gps)
	}
	if dxc != aprs+1 {
		t.Errorf("blank line(s) between APRS and DX Cluster (aprs=%d, dxc=%d)", aprs, dxc)
	}
	if psk != dxc+1 {
		t.Errorf("blank line(s) between DX Cluster and PSK Reporter (dxc=%d, psk=%d)", dxc, psk)
	}
	if http != psk+1 {
		t.Errorf("blank line(s) between PSK Reporter and HTTP Server (psk=%d, http=%d)", psk, http)
	}
	if gps != http+1 {
		t.Errorf("blank line(s) between HTTP Server and GPS (http=%d, gps=%d)", http, gps)
	}
}
