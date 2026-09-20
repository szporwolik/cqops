package tui

import (
	"strings"
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// fakeAPRSClient implements aprs.Client without any network activity.
type fakeAPRSClient struct{ connected bool }

func (f *fakeAPRSClient) Start()            {}
func (f *fakeAPRSClient) Stop()             {}
func (f *fakeAPRSClient) IsRunning() bool   { return f.connected }
func (f *fakeAPRSClient) IsConnected() bool { return f.connected }

func TestStatusBarAPRSStates(t *testing.T) {
	cases := []struct {
		name          string
		globalEnabled bool
		logbookCfg    *config.APRSConfig // nil = no per-logbook APRS
		connected     bool
		offline       bool
		want          string // rendered style segment that must be present
	}{
		{
			name:          "global disabled shows nothing",
			globalEnabled: false,
			want:          "",
		},
		{
			name:          "global only (no logbook APRS) shows warn APRS",
			globalEnabled: true,
			want:          statusDotWarnStyle.Render("APRS") + " ",
		},
		{
			name:          "logbook APRS disabled shows warn APRS",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: false},
			want:          statusDotWarnStyle.Render("APRS") + " ",
		},
		{
			name:          "TX connected shows on APRS",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: true, SendLocation: true},
			connected:     true,
			want:          statusDotOnStyle.Render("APRS") + " ",
		},
		{
			name:          "TX disconnected shows off APRS",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: true, SendLocation: true},
			want:          statusDotOffStyle.Render("APRS") + " ",
		},
		{
			name:          "TX disconnected offline shows warn APRS",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: true, SendLocation: true},
			offline:       true,
			want:          statusDotWarnStyle.Render("APRS") + " ",
		},
		{
			name:          "RX connected shows warn APRS-RX",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: true, SendLocation: false},
			connected:     true,
			want:          statusDotWarnStyle.Render("APRS-RX") + " ",
		},
		{
			name:          "RX disconnected shows off APRS-RX",
			globalEnabled: true,
			logbookCfg:    &config.APRSConfig{Enabled: true, SendLocation: false},
			want:          statusDotOffStyle.Render("APRS-RX") + " ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.width = 120
			m.height = 30
			m.Offline = tc.offline
			m.App.Config.Integrations.APRS.Enabled = tc.globalEnabled
			m.App.Logbook.APRS = tc.logbookCfg
			if tc.connected {
				m.App.APRSClient = &fakeAPRSClient{connected: true}
			}

			bar := m.headerView()
			if tc.want == "" {
				if strings.Contains(bar, "APRS") {
					t.Errorf("APRS indicator present when disabled:\n%s", bar)
				}
				return
			}
			if !strings.Contains(bar, tc.want) {
				t.Errorf("status bar missing %q\nbar: %q", tc.want, bar)
			}
		})
	}
}
