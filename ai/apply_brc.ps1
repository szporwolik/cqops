$ErrorActionPreference = 'Stop'
$f = (Resolve-Path 'g:\Hack\cqops_app\internal\tui\update_screens.go').Path
$lines = [IO.File]::ReadAllLines($f)

# Build the replacement content:
# 1. Lines 0 to 1717 (everything before viewBPLBRC comment)
# 2. New bcastBandRange struct + bcBandRanges data + bcastPresetsForBRC method
# 3. New viewBPLBRC function
# 4. Lines 1749 onwards (shortModeTag and everything after)

$before = $lines[0..1717]

$newContent = @'
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
// Uses config-loaded stations if available; falls back to built-in presets.
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
func (m *Model) viewBPLBRC(region int) []string {
	var lines []string
	bcasts := m.bcastPresetsForBRC()
	if len(bcasts) == 0 {
		return []string{DimStyle.Render("  No broadcast presets for this region.")}
	}
	lines = append(lines, S.Warning.Render("BROADCAST ONLY - receive-only reference"))
	lines = append(lines, DimStyle.Render("SW schedules are seasonal; check HFCC for current data."))
	lines = append(lines, "")

	bandIdx := 0
	for _, bc := range bcasts {
		for bandIdx -lt len(bcBandRanges) -and bcBandRanges[bandIdx].FromKHz -le bc.FreqKHz {
			br := bcBandRanges[bandIdx]
			bandIdx++
			header := fmt.Sprintf("  %-10s %d-%d kHz", br.Label, br.FromKHz, br.ToKHz)
			if br.Mod -ne "" {
				header += "       " + br.Mod
			}
			if br.Note -ne "" {
				header += ", " + br.Note
			}
			lines = append(lines, DimStyle.Render(header))
		}
		freqMHz := fmt.Sprintf("%.3f", float64(bc.FreqKHz)/1000.0)
		lines = append(lines, fmt.Sprintf("    %-6s %s MHz  %s  %s", bc.Band, freqMHz, bc.Station, DimStyle.Render(bc.Area)))
	}
	return lines
}

'@ -split "`r`n"

$after = $lines[1749..($lines.Count - 1)]

$result = $before + $newContent + $after
[IO.File]::WriteAllLines($f, $result)
Write-Host "Done. Lines: $($lines.Count) -> $($result.Count)"
