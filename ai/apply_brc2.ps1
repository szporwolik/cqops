$ErrorActionPreference = 'Stop'
$f = (Resolve-Path 'g:\Hack\cqops_app\internal\tui\update_screens.go').Path
$lines = [IO.File]::ReadAllLines($f)

# Find viewBPLBRC boundaries
$brcStart = -1; $brcEnd = -1
for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -eq '// viewBPLBRC renders broadcast receive-only presets.') { $brcStart = $i }
    if ($brcStart -ge 0 -and $lines[$i] -eq '// shortModeTag returns a compact 3-letter tag for a mode string.') { $brcEnd = $i; break }
}
Write-Host "BRC view: lines $brcStart-$brcEnd"

$before = $lines[0..($brcStart-1)]

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

'@ -split "`r`n"

$after = $lines[$brcEnd..($lines.Count - 1)]
$result = $before + $newContent + $after
$utf8 = New-Object System.Text.UTF8Encoding $false
[IO.File]::WriteAllText($f, ($result -join "`r`n") + "`r`n", $utf8)
Write-Host "Done. Lines: $($lines.Count) -> $($result.Count)"
