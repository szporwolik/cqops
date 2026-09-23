package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// TestIntegrationMenuSectionsFlush pins the section layout: no blank
// separator lines between DX Cluster / HTTP Server / GPS / APRS / PSK
// Reporter, regardless of which sections are expanded.
func TestIntegrationMenuSectionsFlush(t *testing.T) {
	cfg := config.DefaultConfig()
	im := NewIntegrationMenu(cfg)
	im.dxcEnabled = true // expanded section
	im.aprsEnabled = true
	im.width = 120
	im.height = 60

	lines := strings.Split(im.View().Content, "\n")
	login := lineIndex(lines, "Login:")
	http := lineIndex(lines, "HTTP Server:")
	gps := lineIndex(lines, "GPS Service:")
	aprs := lineIndex(lines, "APRS:")
	testBtn := lineIndex(lines, "[ Test APRS ]")
	psk := lineIndex(lines, "PSK Reporter:")
	if login < 0 || http < 0 || gps < 0 || aprs < 0 || testBtn < 0 || psk < 0 {
		t.Fatalf("rows missing: login=%d http=%d gps=%d aprs=%d test=%d psk=%d",
			login, http, gps, aprs, testBtn, psk)
	}
	if http != login+1 {
		t.Errorf("blank line(s) after expanded DX Cluster (login=%d, http=%d)", login, http)
	}
	if gps != http+1 {
		t.Errorf("blank line(s) after collapsed HTTP Server (http=%d, gps=%d)", http, gps)
	}
	if aprs != gps+1 {
		t.Errorf("blank line(s) after collapsed GPS (gps=%d, aprs=%d)", gps, aprs)
	}
	if psk != testBtn+1 {
		t.Errorf("blank line(s) after expanded APRS section (test=%d, psk=%d)", testBtn, psk)
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
	dxc := lineIndex(lines, "DX Cluster:")
	http := lineIndex(lines, "HTTP Server:")
	gps := lineIndex(lines, "GPS Service:")
	aprs := lineIndex(lines, "APRS:")
	psk := lineIndex(lines, "PSK Reporter:")
	if dxc < 0 || http < 0 || gps < 0 || aprs < 0 || psk < 0 {
		t.Fatalf("rows missing: dxc=%d http=%d gps=%d aprs=%d psk=%d", dxc, http, gps, aprs, psk)
	}
	if http != dxc+1 {
		t.Errorf("blank line(s) between DX Cluster and HTTP Server (dxc=%d, http=%d)", dxc, http)
	}
	if gps != http+1 {
		t.Errorf("blank line(s) between HTTP Server and GPS (http=%d, gps=%d)", http, gps)
	}
	if aprs != gps+1 {
		t.Errorf("blank line(s) between GPS and APRS (gps=%d, aprs=%d)", gps, aprs)
	}
	if psk != aprs+1 {
		t.Errorf("blank line(s) between APRS and PSK Reporter (aprs=%d, psk=%d)", aprs, psk)
	}
}
