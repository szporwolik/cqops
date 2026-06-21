$ErrorActionPreference = 'Stop'
$f = (Resolve-Path 'g:\Hack\cqops_app\internal\config\defaults.go').Path
$lines = [IO.File]::ReadAllLines($f)

# Find the DefaultBroadcastStations function boundaries
$fnStart = -1; $fnEnd = -1
for ($i = 0; $i -lt $lines.Count; $i++) {
    if ($lines[$i] -match '^func DefaultBroadcastStations') { $fnStart = $i }
    if ($fnStart -ge 0 -and $lines[$i] -match '^\}$' -and $i -gt $fnStart + 1) { $fnEnd = $i; break }
}
Write-Host "Function at lines $fnStart-$fnEnd"

$newStations = @'
func DefaultBroadcastStations() []BroadcastStation {
	return []BroadcastStation{
		// LW BC 153-279 kHz
		{Radio: "Radio Antena Satelor", Country: "Romania", FrequencyKHz: 153},
		{Radio: "Mongolian National Radio 1", Country: "Mongolia", FrequencyKHz: 164},
		{Radio: "Médi 1", Country: "Morocco", FrequencyKHz: 171},
		{Radio: "BBC Radio 4 LW", Country: "United Kingdom", FrequencyKHz: 198},
		{Radio: "Mongolian National Radio 1", Country: "Mongolia", FrequencyKHz: 209},
		{Radio: "Polskie Radio Jedynka", Country: "Poland", FrequencyKHz: 225},
		{Radio: "Mongolian National Radio 1", Country: "Mongolia", FrequencyKHz: 227},
		{Radio: "Radio Algérie Chaîne 3", Country: "Algeria", FrequencyKHz: 252},

		// MW BC 531-1602 kHz
		{Radio: "Radio Antena Satelor", Country: "Romania", FrequencyKHz: 531},
		{Radio: "Kossuth Rádió", Country: "Hungary", FrequencyKHz: 540},
		{Radio: "Radio România Actualităţi", Country: "Romania", FrequencyKHz: 558},
		{Radio: "BNR Horizont", Country: "Bulgaria", FrequencyKHz: 576},
		{Radio: "CyBC Radio 3", Country: "Cyprus", FrequencyKHz: 603},
		{Radio: "Radio România Actualităţi", Country: "Romania", FrequencyKHz: 612},
		{Radio: "Radio Caroline", Country: "United Kingdom", FrequencyKHz: 648},
		{Radio: "BBC Radio 5 Live", Country: "United Kingdom", FrequencyKHz: 693},
		{Radio: "BBC Radio Scotland", Country: "United Kingdom", FrequencyKHz: 810},
		{Radio: "Radio România Actualităţi", Country: "Romania", FrequencyKHz: 855},

		// MW BC NA 530-1700 kHz
		{Radio: "WSM", Country: "United States", FrequencyKHz: 650},
		{Radio: "WLW", Country: "United States", FrequencyKHz: 700},
		{Radio: "WOR", Country: "United States", FrequencyKHz: 710},
		{Radio: "CFZM", Country: "Canada", FrequencyKHz: 740},
		{Radio: "WSB", Country: "United States", FrequencyKHz: 750},
		{Radio: "WABC", Country: "United States", FrequencyKHz: 770},
		{Radio: "WCBS", Country: "United States", FrequencyKHz: 880},
		{Radio: "WINS", Country: "United States", FrequencyKHz: 1010},
		{Radio: "WBZ", Country: "United States", FrequencyKHz: 1030},
		{Radio: "KFAQ", Country: "United States", FrequencyKHz: 1170},

		// 120m BC 2300-2495 kHz
		{Radio: "Domestic Shortwave Australia", Country: "Australia", FrequencyKHz: 2310},
		{Radio: "Domestic Shortwave Australia", Country: "Australia", FrequencyKHz: 2325},
		{Radio: "Domestic Shortwave Australia", Country: "Australia", FrequencyKHz: 2485},

		// 90m BC 3200-3400 kHz
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 3255},
		{Radio: "Voice of Indonesia", Country: "Indonesia", FrequencyKHz: 3325},
		{Radio: "Radio Sonder Grense", Country: "South Africa", FrequencyKHz: 3325},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 3340},

		// 75m BC 3900-4000 kHz
		{Radio: "Shortwave Gold", Country: "Germany", FrequencyKHz: 3950},
		{Radio: "KBS World Radio", Country: "South Korea", FrequencyKHz: 3955},
		{Radio: "Radio France Internationale", Country: "France", FrequencyKHz: 3965},
		{Radio: "Shortwave Gold", Country: "Germany", FrequencyKHz: 3975},
		{Radio: "HCJB Deutschland", Country: "Germany", FrequencyKHz: 3995},

		// 60m BC 4750-5060 kHz
		{Radio: "China National Radio 1", Country: "China", FrequencyKHz: 4750},
		{Radio: "Radio Tarma", Country: "Peru", FrequencyKHz: 4775},
		{Radio: "Xizang PBS", Country: "China", FrequencyKHz: 4820},
		{Radio: "WWCR", Country: "United States", FrequencyKHz: 4840},
		{Radio: "Voice of America", Country: "United States", FrequencyKHz: 4930},
		{Radio: "Rádio Brasil Central", Country: "Brazil", FrequencyKHz: 4985},
		{Radio: "Radio Rebelde", Country: "Cuba", FrequencyKHz: 5025},
		{Radio: "Beibu Bay Radio", Country: "China", FrequencyKHz: 5050},

		// 49m BC 5900-6200 kHz
		{Radio: "Radio Taiwan International", Country: "Taiwan", FrequencyKHz: 5900},
		{Radio: "Voice of Turkey", Country: "Türkiye", FrequencyKHz: 5960},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 5965},
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 6040},
		{Radio: "CFRX Toronto", Country: "Canada", FrequencyKHz: 6070},
		{Radio: "Voice of America", Country: "United States", FrequencyKHz: 6080},
		{Radio: "Voice of Korea", Country: "North Korea", FrequencyKHz: 6170},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 6195},

		// 41m BC 7200-7600 kHz
		{Radio: "Voice of Nigeria", Country: "Nigeria", FrequencyKHz: 7255},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 7285},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 7325},
		{Radio: "Vatican Radio", Country: "Vatican City", FrequencyKHz: 7360},
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 7420},
		{Radio: "WRMI", Country: "United States", FrequencyKHz: 7455},
		{Radio: "KTWR Trans World Radio", Country: "Guam", FrequencyKHz: 7500},

		// 31m BC 9400-9900 kHz
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 9410},
		{Radio: "Voice of Korea", Country: "North Korea", FrequencyKHz: 9435},
		{Radio: "Radio Taiwan International", Country: "Taiwan", FrequencyKHz: 9545},
		{Radio: "KBS World Radio", Country: "South Korea", FrequencyKHz: 9570},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 9690},
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 9700},
		{Radio: "Radio New Zealand Pacific", Country: "New Zealand", FrequencyKHz: 9700},
		{Radio: "Vatican Radio", Country: "Vatican City", FrequencyKHz: 9705},
		{Radio: "Radio Habana Cuba", Country: "Cuba", FrequencyKHz: 9710},
		{Radio: "Voice of Turkey", Country: "Türkiye", FrequencyKHz: 9875},

		// 25m BC 11600-12200 kHz
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 11650},
		{Radio: "Radio New Zealand Pacific", Country: "New Zealand", FrequencyKHz: 11725},
		{Radio: "KBS World Radio", Country: "South Korea", FrequencyKHz: 11810},
		{Radio: "AWR", Country: "International", FrequencyKHz: 11880},
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 11925},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 12030},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 12095},
		{Radio: "Voice of Korea", Country: "North Korea", FrequencyKHz: 12120},

		// 22m BC 13570-13870 kHz
		{Radio: "Radio Free Asia", Country: "United States", FrequencyKHz: 13580},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 13610},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 13635},
		{Radio: "Voice of Turkey", Country: "Türkiye", FrequencyKHz: 13670},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 13760},
		{Radio: "Voice of Korea", Country: "North Korea", FrequencyKHz: 13760},
		{Radio: "Ifrikya FM", Country: "Algeria", FrequencyKHz: 13790},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 13790},

		// 19m BC 15100-15800 kHz
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 15110},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 15135},
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 15240},
		{Radio: "Voice of America", Country: "United States", FrequencyKHz: 15280},
		{Radio: "Radio Kuwait", Country: "Kuwait", FrequencyKHz: 15530},
		{Radio: "Voice of America", Country: "United States", FrequencyKHz: 15580},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 15680},
		{Radio: "WRMI", Country: "United States", FrequencyKHz: 15770},

		// 16m BC 17480-17900 kHz
		{Radio: "Radio Romania International", Country: "Romania", FrequencyKHz: 17530},
		{Radio: "Voice of America", Country: "United States", FrequencyKHz: 17565},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 17590},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 17675},
		{Radio: "Radio New Zealand Pacific", Country: "New Zealand", FrequencyKHz: 17740},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 17755},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 17790},

		// 15m BC 18900-19020 kHz
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 18910},
		{Radio: "Radio Free Asia", Country: "United States", FrequencyKHz: 18930},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 18970},

		// 13m BC 21450-21850 kHz
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 21525},
		{Radio: "Radio Free Asia", Country: "United States", FrequencyKHz: 21555},
		{Radio: "Radio Exterior de España", Country: "Spain", FrequencyKHz: 21670},
		{Radio: "China Radio International", Country: "China", FrequencyKHz: 21695},
		{Radio: "Radio Saudi", Country: "Saudi Arabia", FrequencyKHz: 21735},
		{Radio: "Radio France Internationale", Country: "France", FrequencyKHz: 21780},

		// 11m BC 25600-26100 kHz
		{Radio: "Radio France Internationale", Country: "France", FrequencyKHz: 25710},
		{Radio: "World Music Radio", Country: "Denmark", FrequencyKHz: 25800},
		{Radio: "BBC World Service", Country: "United Kingdom", FrequencyKHz: 25900},
	}
}
'@ -split "`r`n"

$result = $lines[0..($fnStart-1)] + $newStations + $lines[($fnEnd+1)..($lines.Count-1)]
$utf8 = New-Object System.Text.UTF8Encoding $false
[IO.File]::WriteAllText($f, ($result -join "`r`n"), $utf8)
Write-Host "Done. Lines: $($lines.Count) -> $($result.Count)"
