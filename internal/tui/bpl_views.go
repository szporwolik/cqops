package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ftl/hamradio"
	"github.com/ftl/hamradio/bandplan"
	"github.com/szporwolik/cqops/internal/config"
	"github.com/szporwolik/cqops/internal/version"
)

type nonHamProfile struct {
	ID      string // e.g. "CB_CEPT_EU", "PMR446", "FRS_GMRS"
	Label   string // display label
	RangeLo string // e.g. "26.960"
	RangeHi string // e.g. "27.410"
	Mod     string // modulation
	Note    string // extra info
}

// nonHamProfiles returns the non-ham profiles relevant to an IARU region hint.
func nonHamProfiles(region int) []nonHamProfile {
	switch region {
	case 1:
		return []nonHamProfile{
			{"CB_CEPT_EU", "CB (CEPT/EU)", "26.960", "27.410", "FM/AM/SSB", "40 ch, 26.965–27.405 MHz"},
			{"PMR446_ANALOG", "PMR446 Analog", "446.00625", "446.19375", "FM", "16 ch, 12.5 kHz, 500 mW ERP"},
			{"PMR446_DIGITAL", "PMR446 Digital", "446.000", "446.200", "DMR/dPMR", "32×6.25 kHz or 16×12.5 kHz"},
		}
	case 2:
		return []nonHamProfile{
			{"CB_FCC_US", "CB (FCC/US)", "26.965", "27.405", "AM/FM/SSB", "40 ch, FCC CBRS"},
			{"FRS_GMRS", "FRS / GMRS", "462.5500", "467.7250", "FM", "22 ch; FRS licence-free, GMRS licensed"},
		}
	case 3:
		return []nonHamProfile{
			{"CB_HF_AU", "CB HF (AU)", "26.965", "27.405", "AM/SSB", "40 ch, ACMA HF CB"},
			{"CB_UHF_AU", "CB UHF (AU)", "476.425", "477.4125", "FM", "80 ch, Australian UHF CB"},
		}
	default:
		return nil
	}
}

// =============================================================================
// Broadcast emergency/off-grid presets (BRC) — receive-only reference data
// =============================================================================
// These are NOT ham bands. SW schedules are seasonal and should be checked
// against HFCC / broadcaster schedules when possible.

// bcastPreset is an emergency/off-grid broadcast frequency preset.
type bcastPreset struct {
	FreqKHz     int    // frequency in kHz
	Band        string // "LW", "MW", "SW"
	Station     string // station name
	Area        string // target area / use
	Priority    int    // 1 = top emergency/reference, 2 = secondary
	Reliability string // "high", "seasonal", "check_status"
	Note        string // optional extra info
}

// bcastPresets returns broadcast presets for an IARU region hint.
// These are regional emergency/news/time-signal SW/MW/LW stations useful
// for off-grid awareness. Marked BRC (broadcast / receive-only).
func bcastPresets(region int) []bcastPreset {
	switch region {
	case 1:
		return []bcastPreset{
			{225, "LW", "Polskie Radio Jedynka", "PL/Central Europe", 1, "high", "emergency-news fallback"},
			{7220, "SW", "RRI English", "Europe night", 1, "seasonal", ""},
			{9740, "SW", "RRI English", "Europe evening/night", 1, "seasonal", ""},
			{11960, "SW", "RRI English", "Europe morning", 1, "seasonal", ""},
			{15180, "SW", "RRI English", "Europe day/evening", 1, "seasonal", ""},
			{3955, "SW", "BBC World Service", "Europe possible", 2, "seasonal", ""},
			{9690, "SW", "Radio Exterior de España", "Europe/Africa/Atlantic", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Central Europe", 2, "", ""},
			{756, "MW", "Deutschlandfunk", "Central Europe", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe", 2, "", ""},
		}
	case 2:
		return []bcastPreset{
			{2500, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{5000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{10000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{15000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{20000, "SW", "WWV", "NA/global propagation", 1, "high", "time/freq reference"},
			{3330, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{7850, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{14670, "SW", "CHU Canada", "Canada/North America", 1, "high", "time signal"},
			{6180, "SW", "Rádio Nacional da Amazônia", "Brazil/Amazonia", 2, "seasonal", ""},
			{11780, "SW", "Rádio Nacional da Amazônia", "Brazil/Amazonia", 2, "seasonal", ""},
			{11620, "SW", "RRI English", "NA East/West", 2, "seasonal", ""},
			{11900, "SW", "RRI English", "NA East", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe (DX)", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Europe (DX)", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe (DX)", 2, "", ""},
		}
	case 3:
		return []bcastPreset{
			{7390, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", "emergency/news fallback"},
			{11725, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{13755, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{15720, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{17675, "SW", "RNZ Pacific", "Pacific", 1, "seasonal", ""},
			{13580, "SW", "RRI English", "Japan", 2, "seasonal", ""},
			{11650, "SW", "RRI English", "Japan", 2, "seasonal", ""},
			{15410, "SW", "Akashvani / AIR", "South/Central Asia", 2, "seasonal", ""},
			{15280, "SW", "Akashvani / AIR", "Asia", 2, "seasonal", ""},
			{153, "LW", "Radio Romania Actualitati", "Europe (DX)", 2, "", ""},
			{549, "MW", "Deutschlandfunk", "Europe (DX)", 2, "", ""},
			{648, "MW", "BBC World Service", "Europe (DX)", 2, "", ""},
		}
	default:
		return nil
	}
}

// bcastPresetsAll returns all broadcast presets from all regions, deduplicated
// by frequency+band. The Broadcast tab always shows the global list.
func bcastPresetsAll() []bcastPreset {
	seen := make(map[string]bool)
	var all []bcastPreset
	for r := 1; r <= 3; r++ {
		for _, bc := range bcastPresets(r) {
			key := fmt.Sprintf("%s|%d", bc.Band, bc.FreqKHz)
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, bc)
		}
	}
	return all
}

func (m *Model) handleBPLUpdate(msg tea.Msg, cmd tea.Cmd) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.bpl.cachedSig = "" // invalidate cache on resize
		return m, cmd
	case tea.KeyPressMsg:
		k := msg
		switch {
		case k.String() == "f1" || k.String() == "esc":
			m.screen = screenQSO
			return m, cmd
		case k.String() == "ctrl+e":
			cmd = tea.Batch(cmd, m.exportBPL())
			return m, cmd
		case k.String() == "left" || msg.Code == tea.KeyLeft:
			if m.bpl.tab > 0 {
				m.bpl.tab--
			} else {
				m.bpl.tab = bplTabCount - 1
			}
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "right" || msg.Code == tea.KeyRight:
			if m.bpl.tab < bplTabCount-1 {
				m.bpl.tab++
			} else {
				m.bpl.tab = 0
			}
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "tab":
			m.bpl.tab = (m.bpl.tab + 1) % bplTabCount
			m.bpl.cursor = 0
			m.bpl.scroll = 0
			m.bpl.cachedSig = ""
		case k.String() == "up" || msg.Code == tea.KeyUp:
			if m.bpl.cursor > 0 {
				m.bpl.cursor--
				m.bpl.cachedSig = ""
			}
		case k.String() == "down" || msg.Code == tea.KeyDown:
			m.bpl.cursor++
			m.bpl.cachedSig = ""
		case k.String() == "pgup" || msg.Code == tea.KeyPgUp:
			m.bpl.cursor -= 10
			if m.bpl.cursor < 0 {
				m.bpl.cursor = 0
			}
			m.bpl.cachedSig = ""
		case k.String() == "pgdown" || msg.Code == tea.KeyPgDown:
			m.bpl.cursor += 10
			m.bpl.cachedSig = ""
		case k.String() == "home" || msg.Code == tea.KeyHome:
			m.bpl.cursor = 0
			m.bpl.cachedSig = ""
		case k.String() == "end" || msg.Code == tea.KeyEnd:
			m.bpl.cursor = 9999 // clamped later
			m.bpl.cachedSig = ""
		case k.String() == " " || k.String() == "space" || k.String() == "enter":
			if c := m.bplTuneCmd(); c != nil {
				return m, tea.Batch(cmd, c)
			}
		}
	}
	return m, cmd
}

// bplScrollLines returns the current pre-built line list for the active
// bandplan tab, used by the help bar to show scroll position.
func (m *Model) bplScrollLines() []string { return m.bpl.cachedLines }

// renderBPLContent applies scroll, cursor clamping, and cursor highlight
// to a full list of lines, returning the visible window as a string.
func (m *Model) renderBPLContent(lines []string) string {
	maxVisible := contentHeight(m.height) - 5
	if maxVisible < 3 {
		maxVisible = 3
	}

	// Clamp cursor.
	if len(lines) == 0 {
		m.bpl.cursor = 0
		m.bpl.scroll = 0
		return ""
	}
	if m.bpl.cursor >= len(lines) {
		m.bpl.cursor = len(lines) - 1
	}
	if m.bpl.cursor < 0 {
		m.bpl.cursor = 0
	}

	// Keep cursor in view.
	if m.bpl.cursor < m.bpl.scroll {
		m.bpl.scroll = m.bpl.cursor
	}
	if m.bpl.cursor >= m.bpl.scroll+maxVisible {
		m.bpl.scroll = m.bpl.cursor - maxVisible + 1
	}
	if m.bpl.scroll < 0 {
		m.bpl.scroll = 0
	}
	maxScroll := len(lines) - maxVisible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.bpl.scroll > maxScroll {
		m.bpl.scroll = maxScroll
	}

	// Build visible window with cursor highlight.
	end := m.bpl.scroll + maxVisible
	if end > len(lines) {
		end = len(lines)
	}
	moreAbove := m.bpl.scroll > 0

	var b strings.Builder
	if moreAbove {
		b.WriteString(DimStyle.Render("  ▲ more above"))
		b.WriteByte('\n')
	}
	cursorLine := m.bpl.cursor - m.bpl.scroll
	for i := m.bpl.scroll; i < end; i++ {
		if i-m.bpl.scroll == cursorLine {
			b.WriteString(">")
			b.WriteString(lines[i])
		} else {
			b.WriteString(" ")
			b.WriteString(lines[i])
		}
		if i < end-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func (m *Model) viewBPL(l Layout) string {
	w := l.TerminalW
	if w < 40 {
		w = 80
	}
	ch := contentHeight(m.height)
	if ch < 8 {
		ch = 8
	}
	region := 1
	if m.App != nil && m.App.Logbook != nil {
		r := m.App.Logbook.Station.IARURegion
		if r >= 1 && r <= 3 {
			region = r
		}
	}

	// Cache: rebuild only when state changes.
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(m.bpl.tab))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(region))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(w))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(ch))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.bpl.scroll))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.bpl.cursor))
	sb.WriteByte('|')
	sb.WriteString(strconv.Itoa(m.bpl.bandSel))
	sb.WriteByte('|')
	sb.WriteString(m.bpl.search)
	sig := sb.String()
	if m.bpl.cachedSig == sig && m.bpl.cachedView != "" {
		return m.bpl.cachedView
	}

	// Tab bar — use short names on narrow screens.
	names := bplTabNames
	if w < 100 {
		names = bplTabShortNames
	}
	var tabParts []string
	for i, name := range names {
		if i == m.bpl.tab {
			tabParts = append(tabParts, S.TabActive.Render(name))
		} else {
			tabParts = append(tabParts, S.TabInactive.Render(name))
		}
	}
	tabBar := strings.Join(tabParts, " "+S.TabSeparator.Render("│")+" ")
	header := S.Title.Width(w).Render("Bandplan Iaru Region " + fmt.Sprintf("%d", region))

	// Render tab content — each sub-view returns full line list, renderBPLContent handles scroll/cursor.
	// Pre-built line lists are cached per tab+region so switching tabs doesn't rebuild.
	linesKey := fmt.Sprintf("%d|%d|%s", m.bpl.tab, region, m.bpl.search)
	var lines []string
	if m.bpl.cachedLinesKey == linesKey && m.bpl.cachedLines != nil {
		lines = m.bpl.cachedLines
	} else {
		switch m.bpl.tab {
		case bplTabHAM:
			lines = m.viewBPLHAM(region)
		case bplTabVHF:
			lines = m.viewBPLVHF(region)
		case bplTabCB:
			lines = m.viewBPLCB(region)
		case bplTabPMR:
			lines = m.viewBPLPMR(region)
		case bplTabBRC:
			lines = m.viewBPLBRC()
		case bplTabPORT:
			lines = m.viewBPLPORT(region)
		}
		m.bpl.cachedLines = lines
		m.bpl.cachedLinesKey = linesKey
	}
	body := m.renderBPLContent(lines)

	// Content without the disclaimer footer — that's pinned as the last
	// content row by buildBodyForScreen directly above the help bar.
	content := header + "\n " + tabBar + "\n\n" + body

	m.bpl.cachedView = content
	m.bpl.cachedSig = sig
	return content
}

// viewBPLHAM returns lines for the amateur HF band plan (160m–10m).
func (m *Model) viewBPLHAM(region int) []string {
	bp := bplForRegion(region)

	var lines []string
	for _, name := range bandOrder {
		b, ok := bp[name]
		if !ok {
			continue
		}
		// Band summary line — just band name and range.
		fr := strconv.FormatFloat(float64(b.From)/1e6, 'f', 3, 64)
		to := strconv.FormatFloat(float64(b.To)/1e6, 'f', 3, 64)
		summary := fmt.Sprintf("%-5s %s–%s", string(b.Name), fr, to)
		lines = append(lines, summary)

		// Detail rows for this band.
		for _, p := range b.Portions {
			mode := shortModeTag(string(p.Mode))
			bw := ""
			if p.MaxBandwidth > 0 {
				bw = fmt.Sprintf(" BW %.0f Hz", float64(p.MaxBandwidth))
			}
			pfr := strconv.FormatFloat(float64(p.From)/1e6, 'f', 3, 64)
			pto := strconv.FormatFloat(float64(p.To)/1e6, 'f', 3, 64)
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s–%s %s%s", pfr, pto, mode, bw)))
		}
		// Special frequencies under this band.
		if emcom, ok := emcomFreqs[region]; ok {
			if f, ok := emcom[name]; ok {
				lines = append(lines, S.Error.Render(fmt.Sprintf("  %s EMG  emergency — avoid normal QSO", f)))
			}
		}
		if qrps, ok := qrpFreqs[region]; ok {
			if entries, ok := qrps[name]; ok {
				for _, e := range entries {
					lines = append(lines, fmt.Sprintf("  %s QRP %s centre", e.Freq, e.Mode))
				}
			}
		}
		if f, ok := qrsFreqs[name]; ok {
			lines = append(lines, fmt.Sprintf("  %s QRS slow CW centre", f))
		}
		if f, ok := ibpFreqs[name]; ok {
			lines = append(lines, fmt.Sprintf("  %s IBP beacon — avoid TX", f))
		}
		if f, ok := sstvFreqs[region]; ok {
			if freq, ok := f[name]; ok {
				lines = append(lines, fmt.Sprintf("  %s IMG SSTV/image", freq))
			}
		}
		if avoids, ok := dxAvoidFreqs[region]; ok {
			if entries, ok := avoids[name]; ok {
				for _, e := range entries {
					// Skip AVOID if already covered by an EMCOM entry (dedup).
					if e.Name == "EMG" {
						if emcom, ok2 := emcomFreqs[region]; ok2 {
							if _, ok3 := emcom[name]; ok3 {
								continue
							}
						}
					}
					if e.Name == "DX" {
						lines = append(lines, S.Warning.Render(fmt.Sprintf("  %s Avoid - reserved for DX", e.Freq)))
					} else {
						lines = append(lines, S.Error.Render(fmt.Sprintf("  %s AVOID %s", e.Freq, e.Name)))
					}
				}
			}
		}
	}

	return lines
}

// viewBPLVHF renders the VHF/UHF band plan (6m/4m/2m/70cm).
func (m *Model) viewBPLVHF(region int) []string {
	var lines []string

	// 6m and 4m overview from vhfCalling.
	if vhf, ok := vhfCalling[region]; ok {
		for _, v := range vhf {
			if v.Band != "" {
				// Band range header — dimmed, not tunable.
				lines = append(lines, DimStyle.Render(fmt.Sprintf("%s %s–%s", v.Band, v.FromMHz, v.ToMHz)))
			} else {
				// Single frequency — bright, tunable.
				tag := shortModeTag(v.Mode)
				lines = append(lines, fmt.Sprintf("  %s MHz  %s %s", v.Freq, tag, v.Note))
			}
		}
	}

	// 2m detailed.
	if segs, ok := vhf2mSeeds[region]; ok {
		lines = append(lines, "")
		for _, s := range segs {
			if s.Band != "" {
				// Band range header — dimmed.
				lines = append(lines, DimStyle.Render(fmt.Sprintf("%s %s–%s  %s", s.Band, s.FromMHz, s.ToMHz, s.Note)))
			} else if s.ToMHz != "" {
				// Frequency range entry — dimmed, not a single tunable freq.
				freq := s.Freq
				if freq != "" {
					freq = " CoA " + freq
				}
				lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s–%s %s%s  %s", s.FromMHz, s.ToMHz, s.Kind, freq, s.Note)))
			} else {
				// Single frequency — bright, tunable. Use severity for special kinds.
				sty := severityStyle(s.Kind)
				note := s.Note
				if s.Kind == "LRA" {
					note += " (country-specific)"
				}
				lines = append(lines, sty.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, note)))
			}
		}
	}

	// 70cm detailed.
	if segs, ok := vhf70cmSeeds[region]; ok {
		lines = append(lines, "")
		for _, s := range segs {
			if s.Band != "" {
				// Band range header — dimmed.
				lines = append(lines, DimStyle.Render(fmt.Sprintf("%s %s–%s  %s", s.Band, s.FromMHz, s.ToMHz, s.Note)))
			} else if s.ToMHz != "" {
				// Frequency range entry — dimmed, not a single tunable freq.
				freq := s.Freq
				if freq != "" {
					freq = " CoA " + freq
				}
				lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s–%s %s%s  %s", s.FromMHz, s.ToMHz, s.Kind, freq, s.Note)))
			} else {
				// Single frequency — bright, tunable. Use severity for special kinds.
				sty := severityStyle(s.Kind)
				note := s.Note
				if s.Kind == "LRA" {
					note += " (country-specific)"
				}
				lines = append(lines, sty.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, note)))
			}
		}
	}

	// R3 APRS.
	if region == 3 {
		lines = append(lines, "")
		lines = append(lines, "R3 APRS (country-specific):")
		for _, s := range r3APRSKnown {
			lines = append(lines, DimStyle.Render(fmt.Sprintf("  %s MHz  %s %s", s.Freq, s.Kind, s.Note)))
		}
	}
	return lines
}

// viewBPLCB renders CB channels.
func (m *Model) viewBPLCB(region int) []string {
	var lines []string
	profiles := nonHamProfiles(region)
	var cbProfile *nonHamProfile
	for i := range profiles {
		if profiles[i].ID == "CB_CEPT_EU" || profiles[i].ID == "CB_FCC_US" || profiles[i].ID == "CB_HF_AU" {
			cbProfile = &profiles[i]
			break
		}
	}

	if cbProfile != nil {
		lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+cbProfile.Label)
		lines = append(lines, DimStyle.Render(fmt.Sprintf("%s–%s MHz  %s  %s", cbProfile.RangeLo, cbProfile.RangeHi, cbProfile.Mod, cbProfile.Note)))
		lines = append(lines, "")
		// Single-column layout — each channel on its own line for cursor navigation/tuning.
		for _, ch := range cbChannels {
			tag := ch.Tag
			row := fmt.Sprintf("Ch %-2d  %s", ch.Ch, ch.Freq)
			if tag != "" {
				row += "  "
				if tag == "EMG" {
					row += S.Error.Render(tag)
				} else {
					row += tag
				}
			}
			lines = append(lines, row)
		}
	} else {
		// UHF CB for R3.
		for _, p := range profiles {
			if p.ID == "CB_UHF_AU" {
				lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+p.Label)
				lines = append(lines, DimStyle.Render(fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note)))
			}
		}
	}
	return lines
}

// viewBPLPMR renders PMR446 channels and FRS/GMRS.
func (m *Model) viewBPLPMR(region int) []string {
	var lines []string
	profiles := nonHamProfiles(region)

	// No PMR profiles at all — show region-specific note.
	hasPMR := false
	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" || p.ID == "PMR446_DIGITAL" || p.ID == "FRS_GMRS" {
			hasPMR = true
			break
		}
	}
	if !hasPMR {
		if region == 3 {
			lines = append(lines, DimStyle.Render("No Asia-wide PMR446 equivalent. PMR446 exists in some Asian countries,"))
			lines = append(lines, DimStyle.Render("but check the specific country's licence-free radio allocation."))
		} else {
			lines = append(lines, DimStyle.Render("No licence-free radio profiles for this region."))
		}
		return lines
	}

	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" {
			lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+p.Label)
			lines = append(lines, DimStyle.Render(fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note)))
			lines = append(lines, "")
			// Single-column layout — each channel on its own line for cursor navigation/tuning.
			for _, ch := range pmr446Analog {
				lines = append(lines, fmt.Sprintf("Ch %-2d  %s", ch.Ch, ch.Freq))
			}
		}
		if p.ID == "PMR446_DIGITAL" {
			lines = append(lines, "")
			lines = append(lines, DimStyle.Render(p.Label+": "+p.RangeLo+"–"+p.RangeHi+" MHz  "+p.Mod+"  "+p.Note))
		}
	}
	// FRS/GMRS for R2.
	firstFRS := true
	for _, p := range profiles {
		if p.ID == "FRS_GMRS" {
			if firstFRS && len(lines) == 0 {
				// No PMR content — don't add blank separator line.
			} else {
				lines = append(lines, "")
			}
			lines = append(lines, S.Warning.Render("NOT A HAM BAND")+" — "+p.Label)
			lines = append(lines, DimStyle.Render(fmt.Sprintf("%s–%s MHz  %s  %s", p.RangeLo, p.RangeHi, p.Mod, p.Note)))
			for _, ch := range frsGmrsChannels {
				svc := ""
				if ch.FRS && ch.GMRS {
					svc = "F+G"
				} else if ch.FRS {
					svc = "FRS"
				} else if ch.GMRS {
					svc = "GMR"
				}
				row := fmt.Sprintf("Ch %-2d %s  %-3s", ch.Ch, ch.Freq, svc)
				if ch.Tag != "" {
					row += " " + ch.Tag
				}
				if ch.RptIn != "" {
					row += " RPT+" + ch.RptIn
				}
				lines = append(lines, row)
			}
		}
	}
	return lines
}

// bcastBandRange holds a broadcast band frequency range (ITU broadcast bands).
type bcastBandRange struct {
	Label   string
	FromKHz int
	ToKHz   int
	Mod     string
	Note    string
}

// bcBandRanges lists ITU broadcast band ranges, sorted low to high.
var bcBandRanges = []bcastBandRange{
	{"LW BC", 153, 279, "AM", ""},
	{"MW BC", 531, 1602, "AM", "9 kHz"},
	{"MW BC NA", 530, 1700, "AM", "10 kHz"},
	{"120m BC", 2300, 2495, "AM", ""},
	{"90m BC", 3200, 3400, "AM", ""},
	{"75m BC", 3900, 4000, "AM", ""},
	{"60m BC", 4750, 5060, "AM", ""},
	{"49m BC", 5900, 6200, "AM", ""},
	{"41m BC", 7200, 7600, "AM", ""},
	{"31m BC", 9400, 9900, "AM", ""},
	{"25m BC", 11600, 12200, "AM", ""},
	{"22m BC", 13570, 13870, "AM", ""},
	{"19m BC", 15100, 15800, "AM", ""},
	{"16m BC", 17480, 17900, "AM", ""},
	{"15m BC", 18900, 19020, "AM", ""},
	{"13m BC", 21450, 21850, "AM", ""},
	{"11m BC", 25600, 26100, "AM", ""},
}

// bcastPresetsForBRC returns broadcast presets for the BRC tab.
func (m *Model) bcastPresetsForBRC() []bcastPreset {
	if m.App != nil && m.App.Config != nil && len(m.App.Config.BroadcastStations) > 0 {
		var result []bcastPreset
		for _, s := range m.App.Config.BroadcastStations {
			result = append(result, bcastPreset{
				FreqKHz: s.FrequencyKHz,
				Band:    s.BroadcastBand(),
				Station: s.Radio,
				Area:    s.Country,
			})
		}
		sort.Slice(result, func(i, j int) bool { return result[i].FreqKHz < result[j].FreqKHz })
		return result
	}
	return bcastPresetsAll()
}

// viewBPLBRC renders broadcast receive-only presets.
func (m *Model) viewBPLBRC() []string {
	var lines []string
	bcasts := m.bcastPresetsForBRC()
	lines = append(lines, S.Warning.Render("BROADCAST ONLY - receive-only reference"))
	lines = append(lines, DimStyle.Render("SW schedules are seasonal; check HFCC/EiBi for current data."))
	lines = append(lines, "")

	for _, br := range bcBandRanges {
		header := fmt.Sprintf("  %-10s %d\u2013%d kHz", br.Label, br.FromKHz, br.ToKHz)
		if br.Mod != "" {
			header += "       " + br.Mod
		}
		if br.Note != "" {
			header += ", " + br.Note
		}
		lines = append(lines, DimStyle.Render(header))
		for _, bc := range bcasts {
			if bc.FreqKHz >= br.FromKHz && bc.FreqKHz <= br.ToKHz {
				freqMHz := fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0)
				lines = append(lines, fmt.Sprintf("    %-6s %s MHz  %s  %s", bc.Band, freqMHz, bc.Station, DimStyle.Render(bc.Area)))
			}
		}
	}
	return lines
}

// shortModeTag returns a compact 3-letter tag for a mode string.
func shortModeTag(mode string) string {
	switch mode {
	case "CW":
		return "CW"
	case "Digital", "DIGITAL":
		return "DIG"
	case "Phone", "PHONE", "SSB":
		return "PHN"
	default:
		if len(mode) > 3 {
			return mode[:3]
		}
		return mode
	}
}

// bplDefaultSeverity style is the fallback identity style for tunable
// single-frequency entries (pre-allocated to avoid NewStyle() on cache miss).
var bplDefaultSeverity = lipgloss.NewStyle()

// severityStyle returns the appropriate Lip Gloss style for a segment kind.
// Used only for VHF single-frequency (tunable) entries. Range headers use
// DimStyle directly — not this function.
func severityStyle(kind string) lipgloss.Style {
	switch kind {
	case "EMG", "AVOID":
		return S.Error
	case "DX":
		return S.Warning
	case "SAT":
		return S.Warning
	case "RNG":
		return S.Value
	default:
		return bplDefaultSeverity
	}
}

// viewBPLPORT renders the Portable/SOTA/POTA suggested starting areas tab.
// These are NOT official channels — see disclaimer in the tab heading.
func (m *Model) viewBPLPORT(region int) []string {
	presets, ok := portablePresets[region]
	if !ok || len(presets) == 0 {
		return []string{DimStyle.Render("No portable presets for this IARU region")}
	}

	var lines []string
	lines = append(lines, S.Warning.Render("NOT official channels — suggested portable QRP/SOTA/POTA starting areas only"))
	lines = append(lines, S.Warning.Render("Check bandplan + licence rules. Listen first, ask QRL, self-spot exact frequency."))
	lines = append(lines, "")

	for _, p := range presets {
		cw := ""
		if p.CW != "" {
			cw = fmt.Sprintf("CW %s  ", p.CW)
			if p.CWRange != "" {
				cw += fmt.Sprintf("[%s]", p.CWRange)
			}
		}
		ssb := ""
		if p.SSB != "" {
			if cw != "" {
				ssb = "  │  "
			}
			ssb += fmt.Sprintf("SSB %s  ", p.SSB)
			if p.SSBRange != "" {
				ssb += fmt.Sprintf("[%s]", p.SSBRange)
			}
		}
		bandHdr := fmt.Sprintf("%-5s", p.Band)
		if cw != "" || ssb != "" {
			lines = append(lines, bandHdr+cw+ssb)
		}
		if p.CWNote != "" {
			lines = append(lines, DimStyle.Render(fmt.Sprintf("      %s", p.CWNote)))
		}
		if p.SSBNote != "" && p.SSBNote != p.CWNote {
			lines = append(lines, DimStyle.Render(fmt.Sprintf("      %s", p.SSBNote)))
		}
	}
	return lines
}

// bplForRegion returns the bandplan for the given IARU region.
func bplForRegion(r int) bandplan.Bandplan {
	switch r {
	case 2:
		return bandplan.IARURegion2
	case 3:
		return bandplan.IARURegion3
	default:
		return bandplan.IARURegion1
	}
}

// bplExportMsg carries the result of a band plan export.
type bplExportMsg struct {
	path string
	err  error
}

// exportBPL writes the full band plan as a Markdown document to cqops_bandplan.md
// in the CQOPS config directory, overwriting any existing file.
func (m *Model) exportBPL() tea.Cmd {
	return func() tea.Msg {
		region := 1
		if m.App != nil && m.App.Logbook != nil {
			r := m.App.Logbook.Station.IARURegion
			if r >= 1 && r <= 3 {
				region = r
			}
		}
		dir, err := config.ConfigDir()
		if err != nil {
			return bplExportMsg{err: err}
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return bplExportMsg{err: err}
		}
		path := filepath.Join(dir, "cqops_bandplan.md")
		content := m.buildBPLMarkdown(region)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return bplExportMsg{err: err}
		}
		return bplExportMsg{path: path}
	}
}

// buildBPLMarkdown generates a Markdown document with the full band plan.
func (m *Model) buildBPLMarkdown(region int) string {
	var b strings.Builder

	// Main header.
	b.WriteString(fmt.Sprintf("# CQOps - Iaru Region %d - Bandplan\n\n", region))
	b.WriteString("> Band plans are guidance, not a licence. Check national rules. Listen first.\n")
	b.WriteString("> VHF/UHF repeaters, APRS and LoRa often need country/local overrides.\n")
	b.WriteString("> CB/PMR/BRC are non-amateur services; BRC is receive-only.\n\n")

	// HAM HF section.
	b.WriteString("## Ham Radio - HF\n\n")
	b.WriteString("| Band | From MHz | To MHz | Mode | From | To | BW Hz / Note |\n")
	b.WriteString("|------|----------|--------|------|------|----|-------------|\n")
	m.writeBPLMarkdownRows(&b, region)

	// VHF section.
	b.WriteString("\n## Ham Radio - VHF\n\n")
	b.WriteString("| Band | From MHz | To MHz | Mode | From | To | Note |\n")
	b.WriteString("|------|----------|--------|------|------|----|------|\n")
	m.writeVHFMarkdownRows(&b, region)

	// CB section.
	b.WriteString("\n## Citizen Band - CB\n\n")
	m.writeCBMarkdownRows(&b, region)

	// PMR section.
	b.WriteString("\n## Personal Mobile Radio - PMR\n\n")
	m.writePMRMarkdownRows(&b, region)

	// Broadcast section.
	b.WriteString("\n## Broadcast\n\n")
	m.writeBRCMarkdownRows(&b)

	// Portable/SOTA section.
	b.WriteString("\n## Portable SOTA/POTA Starting Areas\n\n")
	b.WriteString("> NOT official channels — suggested starting areas. Check bandplan + licence rules. Listen first, ask QRL, self-spot exact frequency.\n\n")
	m.writePORTMarkdownRows(&b, region)

	// Footer with version and timestamp.
	b.WriteString("\n---\n")
	b.WriteString(fmt.Sprintf("*Generated by CQOps v%s on %s — visit [cqops.com](https://cqops.com)*\n",
		version.Resolved(), time.Now().UTC().Format("2006-01-02")))

	return b.String()
}

// bplFreqStr formats a hamradio.Frequency as MHz with 3 decimal places.
// Uses strconv to avoid fmt.Sprintf allocation on every call.
func bplFreqStr(f hamradio.Frequency) string {
	return strconv.FormatFloat(float64(f)/1e6, 'f', 3, 64)
}

// bplBwStr formats a hamradio.Frequency as Hz with 0 decimal places,
// or returns "" if zero/negative.
func bplBwStr(bw hamradio.Frequency) string {
	if bw <= 0 {
		return ""
	}
	return strconv.FormatFloat(float64(bw), 'f', 0, 64)
}

func (m *Model) writeBPLMarkdownRows(b *strings.Builder, region int) {
	bp := bplForRegion(region)
	for _, name := range bandOrder {
		bd, ok := bp[name]
		if !ok {
			continue
		}
		// Band header row.
		fmt.Fprintf(b, "| **%s** | %s | %s | | | | |\n", string(bd.Name), bplFreqStr(bd.From), bplFreqStr(bd.To))
		for _, p := range bd.Portions {
			fmt.Fprintf(b, "| | | | %s | %s | %s | %s |\n", string(p.Mode), bplFreqStr(p.From), bplFreqStr(p.To), bplBwStr(p.MaxBandwidth))
		}
		if emcom, ok := emcomFreqs[region]; ok {
			if f, ok := emcom[name]; ok {
				fmt.Fprintf(b, "| | | | **EMG** | %s | | emergency — avoid normal QSO |\n", f)
			}
		}
		if qrps, ok := qrpFreqs[region]; ok {
			if entries, ok := qrps[name]; ok {
				for _, e := range entries {
					fmt.Fprintf(b, "| | | | %s | %s | | QRP centre |\n", e.Mode, e.Freq)
				}
			}
		}
		if f, ok := qrsFreqs[name]; ok {
			fmt.Fprintf(b, "| | | | QRS | %s | | slow CW centre |\n", f)
		}
		if f, ok := ibpFreqs[name]; ok {
			fmt.Fprintf(b, "| | | | **IBP** | %s | | beacon — avoid TX |\n", f)
		}
		if sstv, ok := sstvFreqs[region]; ok {
			if f, ok := sstv[name]; ok {
				fmt.Fprintf(b, "| | | | IMG | %s | | SSTV/image |\n", f)
			}
		}
		if avoids, ok := dxAvoidFreqs[region]; ok {
			if entries, ok := avoids[name]; ok {
				for _, e := range entries {
					if e.Name == "EMG" {
						if emcom, ok2 := emcomFreqs[region]; ok2 {
							if _, ok3 := emcom[name]; ok3 {
								continue
							}
						}
					}
					if e.Name == "DX" {
						fmt.Fprintf(b, "| | | | **Avoid** | %s | | reserved for DX |\n", e.Freq)
					} else {
						fmt.Fprintf(b, "| | | | **AVOID** | %s | | %s |\n", e.Freq, e.Name)
					}
				}
			}
		}
	}
}

func (m *Model) writeVHFMarkdownRows(b *strings.Builder, region int) {
	// 6m/4m overview.
	if vhf, ok := vhfCalling[region]; ok {
		for _, v := range vhf {
			if v.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | | | | %s |\n", v.Band, v.FromMHz, v.ToMHz, v.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", v.Mode, v.Freq, v.Note)
			}
		}
	}
	// 2m.
	if segs, ok := vhf2mSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | %s | | | %s |\n", s.Band, s.FromMHz, s.ToMHz, s.Kind, s.Note)
			} else if s.ToMHz != "" {
				fmt.Fprintf(b, "| | %s | %s | %s | %s | | %s |\n", s.FromMHz, s.ToMHz, s.Kind, s.Freq, s.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
			}
		}
	}
	// 70cm.
	if segs, ok := vhf70cmSeeds[region]; ok {
		for _, s := range segs {
			if s.Band != "" {
				fmt.Fprintf(b, "| **%s** | %s | %s | %s | | | %s |\n", s.Band, s.FromMHz, s.ToMHz, s.Kind, s.Note)
			} else if s.ToMHz != "" {
				fmt.Fprintf(b, "| | %s | %s | %s | %s | | %s |\n", s.FromMHz, s.ToMHz, s.Kind, s.Freq, s.Note)
			} else {
				fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
			}
		}
	}
	// R3 APRS.
	if region == 3 {
		for _, s := range r3APRSKnown {
			fmt.Fprintf(b, "| | | | %s | %s | | %s |\n", s.Kind, s.Freq, s.Note)
		}
	}
}

func (m *Model) writeCBMarkdownRows(b *strings.Builder, region int) {
	profiles := nonHamProfiles(region)
	for _, p := range profiles {
		if p.ID == "CB_CEPT_EU" || p.ID == "CB_FCC_US" || p.ID == "CB_HF_AU" {
			b.WriteString(fmt.Sprintf("\n**%s** — %s–%s MHz, %s\n\n", p.Label, p.RangeLo, p.RangeHi, p.Mod))
		}
	}
	b.WriteString("| Ch | Freq MHz | Tag |\n")
	b.WriteString("|---|----------|-----|\n")
	for _, ch := range cbChannels {
		tag := ch.Tag
		if tag == "" {
			tag = "-"
		}
		fmt.Fprintf(b, "| %d | %s | %s |\n", ch.Ch, ch.Freq, tag)
	}
	// UHF CB for R3.
	for _, p := range profiles {
		if p.ID == "CB_UHF_AU" {
			b.WriteString(fmt.Sprintf("\n**%s** — %s–%s MHz, %s, %s\n\n", p.Label, p.RangeLo, p.RangeHi, p.Mod, p.Note))
		}
	}
}

func (m *Model) writePMRMarkdownRows(b *strings.Builder, region int) {
	profiles := nonHamProfiles(region)
	hasPMR := false
	for _, p := range profiles {
		if p.ID == "PMR446_ANALOG" || p.ID == "PMR446_DIGITAL" || p.ID == "FRS_GMRS" {
			hasPMR = true
			break
		}
	}
	if !hasPMR {
		b.WriteString("No licence-free radio profiles for this region.\n\n")
		if region == 3 {
			b.WriteString("PMR446 exists in some Asian countries, but check the specific country's licence-free radio allocation.\n\n")
		}
		return
	}
	for _, p := range profiles {
		switch p.ID {
		case "PMR446_ANALOG":
			b.WriteString(fmt.Sprintf("### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			b.WriteString("| Ch | Freq MHz |\n")
			b.WriteString("|----|----------|\n")
			for _, ch := range pmr446Analog {
				fmt.Fprintf(b, "| %d | %s |\n", ch.Ch, ch.Freq)
			}
		case "PMR446_DIGITAL":
			b.WriteString(fmt.Sprintf("\n### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
		case "FRS_GMRS":
			b.WriteString(fmt.Sprintf("### %s\n\n", p.Label))
			b.WriteString(fmt.Sprintf("%s–%s MHz, %s, %s\n\n", p.RangeLo, p.RangeHi, p.Mod, p.Note))
			b.WriteString("| Ch | Freq MHz | Service | Tag | Repeater |\n")
			b.WriteString("|----|----------|---------|-----|----------|\n")
			for _, ch := range frsGmrsChannels {
				svc := ""
				if ch.FRS && ch.GMRS {
					svc = "FRS+GMRS"
				} else if ch.FRS {
					svc = "FRS"
				} else if ch.GMRS {
					svc = "GMRS"
				}
				tag := ch.Tag
				if tag == "" {
					tag = "-"
				}
				rpt := ch.RptIn
				if rpt == "" {
					rpt = "-"
				}
				fmt.Fprintf(b, "| %d | %s | %s | %s | %s |\n", ch.Ch, ch.Freq, svc, tag, rpt)
			}
		}
	}
}

func (m *Model) writeBRCMarkdownRows(b *strings.Builder) {
	presets := m.bcastPresetsForBRC()

	for _, br := range bcBandRanges {
		var inBand []bcastPreset
		for _, bc := range presets {
			if bc.FreqKHz >= br.FromKHz && bc.FreqKHz <= br.ToKHz {
				inBand = append(inBand, bc)
			}
		}
		if len(inBand) == 0 {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s %d–%d kHz", br.Label, br.FromKHz, br.ToKHz))
		if br.Mod != "" {
			b.WriteString(", ")
			b.WriteString(br.Mod)
		}
		if br.Note != "" {
			b.WriteString(", ")
			b.WriteString(br.Note)
		}
		b.WriteString("\n\n")
		b.WriteString("| Band | Freq MHz | Station | Area |\n")
		b.WriteString("|------|----------|---------|------|\n")
		for _, bc := range inBand {
			freqMHz := fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0)
			area := bc.Area
			if bc.Reliability == "seasonal" {
				area += " [seasonal]"
			}
			fmt.Fprintf(b, "| %s | %s | %s | %s |\n", bc.Band, freqMHz, bc.Station, area)
		}
		b.WriteString("\n")
	}
}

// writePORTMarkdownRows writes the portable/SOTA/POTA presets as a Markdown table.
func (m *Model) writePORTMarkdownRows(b *strings.Builder, region int) {
	presets, ok := portablePresets[region]
	if !ok || len(presets) == 0 {
		return
	}
	b.WriteString("| Band | CW Start | CW Range | SSB Start | SSB Range | Notes |\n")
	b.WriteString("|------|----------|----------|-----------|-----------|-------|\n")
	for _, p := range presets {
		cw := p.CW
		if cw == "" {
			cw = "—"
		}
		cwR := p.CWRange
		if cwR == "" {
			cwR = "—"
		}
		ssb := p.SSB
		if ssb == "" {
			ssb = "—"
		}
		ssbR := p.SSBRange
		if ssbR == "" {
			ssbR = "—"
		}
		note := p.CWNote
		if p.SSBNote != "" && p.SSBNote != p.CWNote {
			if note != "" {
				note += "; "
			}
			note += p.SSBNote
		}
		if note == "" {
			note = "—"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s |\n", p.Band, cw, cwR, ssb, ssbR, note)
	}
	b.WriteString("\n")
}
