package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
)

// The saved Wavelog API key is already stored encrypted in the secrets store;
// on a shared club station the form must additionally never show it: the field
// stays empty with a placeholder, typing replaces the key, leaving it empty
// keeps the stored one.

func TestStationForm_SharedClubKeyHiddenOnLoad(t *testing.T) {
	f := NewStationForm("", "", "")
	f.SetWavelogValues(&config.WavelogConfig{
		Enabled:    true,
		URL:        "https://log.example.com",
		APIKey:     "wl2_secretkey",
		SharedClub: true,
	})

	if got := f.WlKey.Value(); got != "" {
		t.Errorf("WlKey.Value() = %q, want empty (hidden)", got)
	}
	if f.wlStoredKey != "wl2_secretkey" {
		t.Errorf("wlStoredKey = %q, want the stored key", f.wlStoredKey)
	}
	if f.WlKey.Placeholder == "" || strings.Contains(f.WlKey.Placeholder, "wl2_secretkey") {
		t.Errorf("placeholder = %q, want a non-revealing hint", f.WlKey.Placeholder)
	}

	// Values() must hand the stored key back so saving does not wipe it.
	_, _, _, _, _, _, _, _, _, wlKey, _, _, _, _, _, _, _, _, _ := f.Values()
	if wlKey != "wl2_secretkey" {
		t.Errorf("Values() key = %q, want the stored key", wlKey)
	}
}

func TestStationForm_SharedClubKeyReplacedByTyping(t *testing.T) {
	f := NewStationForm("", "", "")
	f.SetWavelogValues(&config.WavelogConfig{
		Enabled:    true,
		APIKey:     "wl2_oldkey",
		SharedClub: true,
	})

	f.WlKey.SetValue("wl2_newkey")
	_, _, _, _, _, _, _, _, _, wlKey, _, _, _, _, _, _, _, _, _ := f.Values()
	if wlKey != "wl2_newkey" {
		t.Errorf("Values() key = %q, want the typed replacement", wlKey)
	}
}

func TestStationForm_SharedClubKeyNeverUnmasks(t *testing.T) {
	f := NewStationForm("", "", "")
	f.SetWavelogValues(&config.WavelogConfig{
		Enabled:    true,
		APIKey:     "wl2_secretkey",
		SharedClub: true,
	})

	f.WlKey.Focus()
	f.unmaskSecretsOnFocus()
	if f.WlKey.EchoMode != textinput.EchoPassword {
		t.Errorf("EchoMode = %v, want EchoPassword (never reveal on shared club)", f.WlKey.EchoMode)
	}

	// Private mode keeps the existing reveal-on-focus behavior.
	p := NewStationForm("", "", "")
	p.SetWavelogValues(&config.WavelogConfig{Enabled: true, APIKey: "wl2_other"})
	p.WlKey.Focus()
	p.unmaskSecretsOnFocus()
	if p.WlKey.EchoMode != textinput.EchoNormal {
		t.Errorf("private EchoMode = %v, want EchoNormal when focused", p.WlKey.EchoMode)
	}
}

func TestStationForm_SharedClubToggleOffRevealsKeyAgain(t *testing.T) {
	f := NewStationForm("", "", "")
	f.SetWavelogValues(&config.WavelogConfig{
		Enabled:    true,
		APIKey:     "wl2_secretkey",
		SharedClub: true,
	})

	// Toggle the checkbox off (space handler path).
	f.wlSharedCbFocus = true
	f.HandleKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})

	if f.WlSharedClub {
		t.Fatal("shared club flag should be off")
	}
	if got := f.WlKey.Value(); got != "wl2_secretkey" {
		t.Errorf("WlKey.Value() = %q, want the stored key restored", got)
	}
	if f.wlStoredKey != "" {
		t.Errorf("wlStoredKey = %q, want empty after restore", f.wlStoredKey)
	}
}

func TestStationForm_SharedClubWithoutKeyStaysEmpty(t *testing.T) {
	f := NewStationForm("", "", "")
	f.SetWavelogValues(&config.WavelogConfig{Enabled: true, SharedClub: true})

	_, _, _, _, _, _, _, _, _, wlKey, _, _, _, _, _, _, _, _, _ := f.Values()
	if wlKey != "" {
		t.Errorf("Values() key = %q, want empty (nothing stored)", wlKey)
	}
}
