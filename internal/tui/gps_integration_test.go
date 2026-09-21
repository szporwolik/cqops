package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/app"
	"github.com/szporwolik/cqops/internal/config"
)

func TestTruncateGrid(t *testing.T) {
	tests := []struct {
		grid string
		prec int
		want string
	}{
		{"JO90aa11xx", 10, "JO90aa11xx"},
		{"JO90aa11xx", 8, "JO90aa11"},
		{"JO90aa11xx", 6, "JO90aa"},
		{"JO90", 6, "JO90"},
		{"JO90aa", 10, "JO90aa"},
	}
	for _, tt := range tests {
		if got := truncateGrid(tt.grid, tt.prec); got != tt.want {
			t.Errorf("truncateGrid(%q,%d) = %q, want %q", tt.grid, tt.prec, got, tt.want)
		}
	}
}

func TestGPSLogPrecision(t *testing.T) {
	// No app — default full precision.
	if got := (&Model{}).gpsLogPrecision(); got != 10 {
		t.Errorf("default precision = %d, want 10", got)
	}

	m := &Model{App: &app.App{Config: &config.Config{
		Integrations: config.IntegrationsConfig{GPS: config.GPSConfig{GridPrecision: 6}},
	}}}
	if got := m.gpsLogPrecision(); got != 6 {
		t.Errorf("configured precision = %d, want 6", got)
	}

	m.App.Config.Integrations.GPS.GridPrecision = 8
	if got := m.gpsLogPrecision(); got != 8 {
		t.Errorf("configured precision = %d, want 8", got)
	}

	// Invalid values fall back to full precision.
	m.App.Config.Integrations.GPS.GridPrecision = 4
	if got := m.gpsLogPrecision(); got != 10 {
		t.Errorf("invalid precision = %d, want 10 fallback", got)
	}
}
