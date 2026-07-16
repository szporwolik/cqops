package qso

import "strings"

// ParsedCallsign holds a decomposed callsign with operating context.
type ParsedCallsign struct {
	Raw       string // original callsign as entered
	Canonical string // uppercase, trimmed

	BaseCall          string   // e.g. SP9SPM
	OperatingPrefix   string   // e.g. 9A, EA8, DL (foreign prefix)
	OperatingSuffixes []string // e.g. [P], [M], [QRP]

	IsPortable           bool
	IsMobile             bool
	IsMaritimeMobile     bool
	IsAeronauticalMobile bool
	IsQRP                bool

	// Filled by DXCC resolution (TUI layer).
	OperatingDXCC      int
	OperatingEntity    string
	OperatingContinent string
	OperatingCQZone    int
	OperatingITUZone   int
}

// portableSuffixes maps known portable/mobile suffixes to their flags.
var portableSuffixes = map[string]struct {
	isPortable, isMobile, isMM, isAM, isQRP bool
}{
	"P":   {isPortable: true},
	"M":   {isMobile: true},
	"MM":  {isMM: true},
	"AM":  {isAM: true},
	"QRP": {isQRP: true},
}

// ParseCallsign decomposes a callsign string into its components.
// It does NOT resolve DXCC entities — that is done by the TUI layer
// using CTY.DAT prefix data via EnrichParsedCall.
func ParseCallsign(raw string) ParsedCallsign {
	pc := ParsedCallsign{Raw: raw}
	pc.Canonical = strings.ToUpper(strings.TrimSpace(raw))
	if pc.Canonical == "" {
		return pc
	}

	// Use the existing DeriveBaseCall which understands standard call patterns
	// (e.g. SP9SPM, K1ABC) and correctly distinguishes them from prefixes
	// that also contain digits (e.g. EA8, KH6, 9A).
	pc.BaseCall = DeriveBaseCall(pc.Canonical)
	if pc.BaseCall == "" {
		pc.BaseCall = pc.Canonical
	}

	// Find the base call position in the slash-separated parts.
	parts := strings.Split(pc.Canonical, "/")
	baseIdx := -1
	for i, p := range parts {
		if strings.EqualFold(p, pc.BaseCall) {
			baseIdx = i
			break
		}
	}

	if baseIdx < 0 {
		return pc
	}

	// Parts before the base call are operating prefixes.
	for i := 0; i < baseIdx; i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		if pc.OperatingPrefix == "" {
			pc.OperatingPrefix = p
		} else {
			pc.OperatingPrefix += "/" + p
		}
	}

	// Parts after the base call are suffixes.
	for i := baseIdx + 1; i < len(parts); i++ {
		p := parts[i]
		if p == "" {
			continue
		}
		if flags, ok := portableSuffixes[p]; ok {
			pc.OperatingSuffixes = append(pc.OperatingSuffixes, p)
			pc.IsPortable = pc.IsPortable || flags.isPortable
			pc.IsMobile = pc.IsMobile || flags.isMobile
			pc.IsMaritimeMobile = pc.IsMaritimeMobile || flags.isMM
			pc.IsAeronauticalMobile = pc.IsAeronauticalMobile || flags.isAM
			pc.IsQRP = pc.IsQRP || flags.isQRP
		} else {
			pc.OperatingSuffixes = append(pc.OperatingSuffixes, p)
		}
	}

	return pc
}

// HasForeignPrefix returns true when the callsign includes a recognized
// foreign operating prefix. It does not require numeric DXCC resolution to
// identify the prefix form.
func (pc ParsedCallsign) HasForeignPrefix() bool {
	return pc.OperatingPrefix != ""
}
