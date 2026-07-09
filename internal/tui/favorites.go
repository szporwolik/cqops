package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/qso"
)

// favoriteKeySlot returns a slot number (0-2) for known favorite keys.
// Returns -1 if the key is not a favorite shortcut.
func favoriteKeySlot(code rune) int {
	switch code {
	case tea.KeyInsert:
		return 0
	case tea.KeyHome:
		return 1
	case tea.KeyPgUp:
		return 2
	}
	return -1
}

// favoriteKeyLabel returns a human label for a favorite slot.
func favoriteKeyLabel(slot int) string {
	switch slot {
	case 0:
		return "Ins"
	case 1:
		return "Home"
	case 2:
		return "PgUp"
	}
	return "?"
}

// handleFavoriteKey checks for Alt+Ins/Home/PgUp (recall) and
// Alt+Shift+Ins/Home/PgUp (save).  Alt avoids conflicts with standard
// terminal editing keys and works reliably across all terminal types.
func (m *Model) handleFavoriteKey(msg tea.KeyPressMsg) (tea.Cmd, bool) {
	if msg.Mod&tea.ModAlt == 0 {
		return nil, false
	}
	slot := favoriteKeySlot(msg.Code)
	if slot < 0 {
		return nil, false
	}
	if msg.Mod&tea.ModShift != 0 {
		return m.favoriteSave(slot), true
	}
	return m.favoriteRecall(slot), true
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
		m.toasts.Warn(fmt.Sprintf("Favorite %s save failed", favoriteKeyLabel(slot)))
		return nil
	}
	m.toasts.Success(fmt.Sprintf("Favorite %s saved: %s %s %s",
		favoriteKeyLabel(slot), fav.Band, freqStr, fav.Mode))
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
		m.toasts.Warn(fmt.Sprintf("Favorite %s is empty", favoriteKeyLabel(slot)))
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
	m.toasts.Success(fmt.Sprintf("Favorite %s recalled: %s %g %s",
		favoriteKeyLabel(slot), fav.Band, fav.Freq, fav.Mode))
	return nil
}
