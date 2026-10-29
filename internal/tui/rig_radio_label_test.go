package tui

import (
	"testing"

	"github.com/szporwolik/cqops/internal/config"
)

// TestRigRadioLabel pins the dashboard's Radio row. A live smoke test found
// " (Xiegu DIGI (Xiegu G90))" — a leading space and a pair of parentheses —
// because the preset carried a model but no name.
func TestRigRadioLabel(t *testing.T) {
	cases := []struct {
		name string
		rp   config.RigPreset
		want string
	}{
		{"name and model", config.RigPreset{Name: "Xiegu DIGI", Model: "Xiegu G90"}, "Xiegu DIGI (Xiegu G90)"},
		{"model only", config.RigPreset{Model: "Xiegu G90"}, "Xiegu G90"},
		{"name only", config.RigPreset{Name: "Xiegu DIGI"}, "Xiegu DIGI"},
		{"same name and model", config.RigPreset{Name: "IC-7300", Model: "IC-7300"}, "IC-7300"},
		{"neither", config.RigPreset{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rigRadioLabel(tc.rp); got != tc.want {
				t.Errorf("rigRadioLabel(%+v) = %q, want %q", tc.rp, got, tc.want)
			}
		})
	}
}
