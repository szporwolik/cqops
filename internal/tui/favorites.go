package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

// handleFavoriteKey checks for ctrl+digit (recall) and ctrl+shift+digit /
// ctrl+shifted_char (save). Returns true if the key was handled.
func (m *Model) handleFavoriteKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	k := msg.String()
	// ctrl+shift+0 … ctrl+shift+9  → save to slot
	if strings.HasPrefix(k, "ctrl+shift+") && len(k) == 12 {
		d := k[11]
		if d >= '0' && d <= '9' {
			return m.favoriteSave(int(d - '0')), true
		}
	}
	// ctrl+shifted_char — what some terminals send for ctrl+shift+digit.
	if d, ok := shiftedDigitFromCtrlPlus(k); ok {
		return m.favoriteSave(d), true
	}
	// ctrl+0 … ctrl+9  → recall from slot
	if strings.HasPrefix(k, "ctrl+") && len(k) == 6 {
		d := k[5]
		if d >= '0' && d <= '9' {
			return m.favoriteRecall(int(d - '0')), true
		}
	}
	return nil, false
}

// shiftedDigitFromCtrlPlus maps "ctrl+!" → 1, "ctrl+@" → 2, … "ctrl+)" → 0.
// Returns the digit and true if k is one of these patterns.
func shiftedDigitFromCtrlPlus(k string) (int, bool) {
	if !strings.HasPrefix(k, "ctrl+") || len(k) != 6 {
		return 0, false
	}
	switch k[5] {
	case ')':
		return 0, true
	case '!':
		return 1, true
	case '@':
		return 2, true
	case '#':
		return 3, true
	case '$':
		return 4, true
	case '%':
		return 5, true
	case '^':
		return 6, true
	case '&':
		return 7, true
	case '*':
		return 8, true
	case '(':
		return 9, true
	}
	return 0, false
}

// favoriteSave stores the current mode, frequency, submode, and band into
// the given slot (0-9) in the config and persists to disk.
func (m *Model) favoriteSave(slot int) tea.Cmd {
	if m.App == nil || m.App.Config == nil {
		return nil
	}
	if m.App.Config.Favorites == nil {
		m.App.Config.Favorites = make(map[int]config.Favorite)
	}

	freqStr := strings.TrimSpace(m.fields[fieldFreq].Value())
	freq, _ := strconv.ParseFloat(freqStr, 64)

	fav := config.Favorite{
		Mode:    strings.TrimSpace(m.fields[fieldMode].Value()),
		Freq:    freq,
		Submode: strings.TrimSpace(m.fields[fieldSubmode].Value()),
		Band:    strings.TrimSpace(m.fields[fieldBand].Value()),
	}

	m.App.Config.Favorites[slot] = fav
	if err := config.Save(m.App.ConfigPath, m.App.Config); err != nil {
		m.toasts.Warn(fmt.Sprintf("Favorite %d save failed", slot))
		return nil
	}
	m.toasts.Success(fmt.Sprintf("Favorite %d saved: %s %s %s",
		slot, fav.Band, freqStr, fav.Mode))
	return nil
}

// favoriteRecall restores the mode, frequency, submode, and band from the
// given slot (0-9). If the slot is empty, a warning toast is shown.
func (m *Model) favoriteRecall(slot int) tea.Cmd {
	if m.App == nil || m.App.Config == nil {
		return nil
	}
	fav, ok := m.App.Config.Favorites[slot]
	if !ok || (fav.Mode == "" && fav.Freq == 0 && fav.Band == "") {
		m.toasts.Warn(fmt.Sprintf("Favorite %d is empty", slot))
		return nil
	}

	if fav.Mode != "" {
		m.fields[fieldMode].SetValue(fav.Mode)
	}
	if fav.Submode != "" {
		m.fields[fieldSubmode].SetValue(fav.Submode)
	}
	if fav.Freq > 0 {
		// Trim trailing zeros so the display is clean (e.g. 14.25 not 14.250000).
		// This matches the ADIF export formatting in internal/qso/adif.go.
		m.fields[fieldFreq].SetValue(strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", fav.Freq), "0"), "."))
	}
	if fav.Band != "" {
		m.fields[fieldBand].SetValue(fav.Band)
	} else if fav.Freq > 0 {
		// Derive band from frequency if not explicitly stored.
		if b := qso.DeriveBand(fav.Freq); b != "" {
			m.fields[fieldBand].SetValue(b)
		}
	}

	m.rc.formSig = "" // invalidate form render cache
	m.toasts.Success(fmt.Sprintf("Favorite %d recalled: %s %.0f %s",
		slot, fav.Band, fav.Freq, fav.Mode))
	return nil
}
