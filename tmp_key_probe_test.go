package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestKeyProbe(t *testing.T) {
	t.Logf("ctrl+s text: %q", tea.KeyPressMsg{Text: "\x13"}.String())
	t.Logf("space text: %q", tea.KeyPressMsg{Text: " "}.String())
	t.Logf("space code: %q", tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}.String())
}
