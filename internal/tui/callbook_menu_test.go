package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// Regression: several priority fields were written without a trailing newline,
// so the next section header (e.g. "QRZ.com:") was appended to the truncated
// priority line and effectively invisible — the QRZ section could not be seen
// or navigated into.

func TestCallbookMenu_AllSectionHeadersRender(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cfg.Integrations.Callbook.HamQTH.Enabled = true
	cfg.Integrations.Callbook.QRZRu.Enabled = true
	cfg.Logbooks["test"] = config.Logbook{
		Station: config.Station{Callsign: "SP9MOA"},
		Wavelog: &config.WavelogConfig{Enabled: true, URL: "https://log.example.com", APIKey: "wl2_x"},
	}

	cm := NewCallbookMenu(cfg)
	cm.width = 100
	cm.height = 40
	lines := strings.Split(cm.View().Content, "\n")

	headers := []string{"QRZ.com:", "HamQTH:", "Callook.info:", "QRZ.RU:", "Wavelog:"}
	for _, h := range headers {
		found := false
		for _, line := range lines {
			if !strings.Contains(line, h) {
				continue
			}
			found = true
			// The header must own its line — no previous field may have
			// been joined onto it.
			if strings.Contains(line, "Priority:") {
				t.Errorf("%s shares its line with a priority field: %q", h, line)
			}
		}
		if !found {
			t.Errorf("section header %q missing from the rendered menu", h)
		}
	}
}

func TestCallbookMenu_TabReachesQRZFields(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cfg.Integrations.Callbook.QRZRu.Enabled = true
	cm := NewCallbookMenu(cfg)
	cm.width = 100
	cm.height = 40
	cm.fm.reset()

	seen := map[int]bool{}
	for i := 0; i < 24; i++ {
		seen[cm.fm.row] = true
		cm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	for _, row := range []int{cmQRZChk, cmQRZUser, cmQRZPass, cmQRZPriority, cmQRZTest} {
		if !seen[row] {
			t.Errorf("row %d (QRZ section) was never reachable by Tab; seen: %v", row, seen)
		}
	}
	for _, row := range []int{cmQRZRuChk, cmQRZRuUser, cmQRZRuPass, cmQRZRuPriority, cmQRZRuTest} {
		if !seen[row] {
			t.Errorf("row %d (QRZ.RU section) was never reachable by Tab; seen: %v", row, seen)
		}
	}
}

// TestCallbookMenu_SectionOrderMatchesPriority pins the visible order in
// Settings → Callbook to the trust-based priority order, independent of
// which providers are currently enabled.
func TestCallbookMenu_SectionOrderMatchesPriority(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Integrations.Callbook.QRZ.Enabled = true
	cfg.Integrations.Callbook.HamQTH.Enabled = true
	cfg.Integrations.Callbook.QRZRu.Enabled = true
	cfg.Integrations.Callbook.Logbook.Enabled = true

	cm := NewCallbookMenu(cfg)
	cm.width = 100
	cm.height = 50
	content := cm.View().Content

	order := []string{"QRZ.com:", "HamQTH:", "Callook.info:", "QRZ.RU:", "Logbook:"}
	prev := -1
	for _, h := range order {
		p := strings.Index(content, h)
		if p < 0 {
			t.Fatalf("section header %q missing from the menu", h)
		}
		if p < prev {
			t.Errorf("section %q rendered above the previous section — order must match priority", h)
		}
		prev = p
	}
}
