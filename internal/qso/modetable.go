package qso

import "strings"

var modeSubmodes = map[string][]string{
	"AM":           {},
	"ARDOP":        {},
	"ATV":          {},
	"CHIP":         {"CHIP64", "CHIP128"},
	"CLO":          {},
	"CONTESTI":     {},
	"CW":           {"PCW"},
	"DIGITALVOICE": {"C4FM", "DMR", "DSTAR", "FREEDV", "M17"},
	"DOMINO":       {"DOM-M", "DOM4", "DOM5", "DOM8", "DOM11", "DOM16", "DOM22", "DOM44", "DOM88", "DOMINOEX", "DOMINOF"},
	"DYNAMIC":      {"FREEDATA", "VARA HF", "VARA SATELLITE", "VARA FM 1200", "VARA FM 9600"},
	"FAX":          {},
	"FM":           {},
	"FSK441":       {},
	"FSK":          {"SCAMP_FAST", "SCAMP_SLOW", "SCAMP_VSLOW"},
	"FT8":          {}, // top-level mode per ADIF 3.1.7
	"HELL":         {"FMHELL", "FSKH105", "FSKH245", "FSKHELL", "HELL80", "HELLX5", "HELLX9", "HFSK", "PSKHELL", "SLOWHELL"},
	"ISCAT":        {"ISCAT-A", "ISCAT-B"},
	"JT4":          {"JT4A", "JT4B", "JT4C", "JT4D", "JT4E", "JT4F", "JT4G"},
	"JT6M":         {},
	"JT9": {
		"JT9-1", "JT9-2", "JT9-5", "JT9-10", "JT9-30",
		"JT9A", "JT9B", "JT9C", "JT9D", "JT9E", "JT9E FAST",
		"JT9F", "JT9F FAST", "JT9G", "JT9G FAST", "JT9H", "JT9H FAST",
	},
	"JT44":   {},
	"JT65":   {"JT65A", "JT65B", "JT65B2", "JT65C", "JT65C2"},
	"MFSK":   {"FSQCALL", "FST4", "FST4W", "FT2", "FT4", "JS8", "JTMS", "MFSK4", "MFSK8", "MFSK11", "MFSK16", "MFSK22", "MFSK31", "MFSK32", "MFSK64", "MFSK64L", "MFSK128", "MFSK128L", "Q65"},
	"MSK144": {},
	"MTONE":  {"SCAMP_OO", "SCAMP_OO_SLW"},
	"MT63":   {},
	"OFDM":   {"RIBBIT_PIX", "RIBBIT_SMS"},
	"OLIVIA": {"OLIVIA 4/125", "OLIVIA 4/250", "OLIVIA 8/250", "OLIVIA 8/500", "OLIVIA 16/500", "OLIVIA 16/1000", "OLIVIA 32/1000"},
	"OPERA":  {"OPERA-BEACON", "OPERA-QSO"},
	"PAC":    {"PAC2", "PAC3", "PAC4"},
	"PAX":    {"PAX2"},
	"PKT":    {},
	"PSK": {
		"8PSK125", "8PSK125F", "8PSK125FL", "8PSK250", "8PSK250F", "8PSK250FL",
		"8PSK500", "8PSK500F", "8PSK1000", "8PSK1000F", "8PSK1200F", "FSK31",
		"PSK10", "PSK31", "PSK63", "PSK63F", "PSK63RC4", "PSK63RC5", "PSK63RC10",
		"PSK63RC20", "PSK63RC32", "PSK125", "PSK125C12", "PSK125R", "PSK125RC10",
		"PSK125RC12", "PSK125RC16", "PSK125RC4", "PSK125RC5", "PSK250", "PSK250C6",
		"PSK250R", "PSK250RC2", "PSK250RC3", "PSK250RC5", "PSK250RC6", "PSK250RC7",
		"PSK500", "PSK500C2", "PSK500C4", "PSK500R", "PSK500RC2", "PSK500RC3",
		"PSK500RC4", "PSK800C2", "PSK800RC2", "PSK1000", "PSK1000C2", "PSK1000R",
		"PSK1000RC2", "PSKAM10", "PSKAM31", "PSKAM50", "PSKFEC31", "QPSK31",
		"QPSK63", "QPSK125", "QPSK250", "QPSK500", "SIM31",
	},
	"PSK2K":  {},
	"Q15":    {},
	"QRA64":  {"QRA64A", "QRA64B", "QRA64C", "QRA64D", "QRA64E"},
	"ROS":    {"ROS-EME", "ROS-HF", "ROS-MF"},
	"RTTY":   {"ASCI"},
	"RTTYM":  {},
	"SSB":    {"LSB", "USB"},
	"SSTV":   {},
	"T10":    {},
	"THOR":   {"THOR-M", "THOR4", "THOR5", "THOR8", "THOR11", "THOR16", "THOR22", "THOR25X4", "THOR50X1", "THOR50X2", "THOR100"},
	"THRB":   {"THRBX", "THRBX1", "THRBX2", "THRBX4", "THROB1", "THROB2", "THROB4"},
	"TOR":    {"AMTORFEC", "GTOR", "NAVTEX", "SITORB"},
	"V4":     {},
	"VOI":    {},
	"WINMOR": {},
	"WSPR":   {},
}

type modeImport struct {
	mode    string
	submode string
}

var importOnlyModes = map[string]modeImport{
	"AMTORFEC": {"TOR", "AMTORFEC"},
	"ASCI":     {"RTTY", "ASCI"},
	"C4FM":     {"DIGITALVOICE", "C4FM"},
	"CHIP64":   {"CHIP", "CHIP64"},
	"CHIP128":  {"CHIP", "CHIP128"},
	"DOMINOF":  {"DOMINO", "DOMINOF"},
	"DSTAR":    {"DIGITALVOICE", "DSTAR"},
	"FMHELL":   {"HELL", "FMHELL"},
	"FSK31":    {"PSK", "FSK31"},
	"FT4":      {"MFSK", "FT4"},
	"FT2":      {"MFSK", "FT2"},
	"GTOR":     {"TOR", "GTOR"},
	"HELL80":   {"HELL", "HELL80"},
	"HFSK":     {"HELL", "HFSK"},
	"JT4A":     {"JT4", "JT4A"},
	"JT4B":     {"JT4", "JT4B"},
	"JT4C":     {"JT4", "JT4C"},
	"JT4D":     {"JT4", "JT4D"},
	"JT4E":     {"JT4", "JT4E"},
	"JT4F":     {"JT4", "JT4F"},
	"JT4G":     {"JT4", "JT4G"},
	"JT65A":    {"JT65", "JT65A"},
	"JT65B":    {"JT65", "JT65B"},
	"JT65C":    {"JT65", "JT65C"},
	"LSB":      {"SSB", "LSB"},
	"MFSK8":    {"MFSK", "MFSK8"},
	"MFSK16":   {"MFSK", "MFSK16"},
	"PAC2":     {"PAC", "PAC2"},
	"PAC3":     {"PAC", "PAC3"},
	"PAX2":     {"PAX", "PAX2"},
	"PCW":      {"CW", "PCW"},
	"PSK10":    {"PSK", "PSK10"},
	"PSK31":    {"PSK", "PSK31"},
	"PSK63":    {"PSK", "PSK63"},
	"PSK63F":   {"PSK", "PSK63F"},
	"PSK125":   {"PSK", "PSK125"},
	"PSKAM10":  {"PSK", "PSKAM10"},
	"PSKAM31":  {"PSK", "PSKAM31"},
	"PSKAM50":  {"PSK", "PSKAM50"},
	"PSKFEC31": {"PSK", "PSKFEC31"},
	"PSKHELL":  {"HELL", "PSKHELL"},
	"QPSK31":   {"PSK", "QPSK31"},
	"QPSK63":   {"PSK", "QPSK63"},
	"QPSK125":  {"PSK", "QPSK125"},
	"THRBX":    {"THRB", "THRBX"},
	"USB":      {"SSB", "USB"},
}

var rigModeMap = map[string]string{
	"USB":     "SSB",
	"LSB":     "SSB",
	"CW":      "CW",
	"CW-L":    "CW",
	"CW-U":    "CW",
	"CWR":     "CW",
	"RTTY":    "RTTY",
	"RTTYR":   "RTTY",
	"AM":      "AM",
	"FM":      "FM",
	"WFM":     "FM",
	"PKT":     "PKT",
	"PKT-L":   "PKT",
	"PKT-U":   "PKT",
	"PKT-FM":  "PKT",
	"DATA-U":  "DATA-U",
	"DATA-L":  "DATA-L",
	"DATA-FM": "DATA-FM",
}

func IsValidMode(mode string) bool {
	_, ok := modeSubmodes[strings.ToUpper(mode)]
	return ok
}

func IsValidSubmode(mode, submode string) bool {
	if submode == "" {
		return true
	}
	mode = strings.ToUpper(mode)
	if mode == "" {
		return false
	}
	submodes, ok := modeSubmodes[mode]
	if !ok {
		return false
	}
	if len(submodes) == 0 {
		return false
	}
	upper := strings.ToUpper(submode)
	for _, sm := range submodes {
		if strings.EqualFold(sm, submode) || strings.ToUpper(sm) == upper {
			return true
		}
	}
	return false
}

func NormalizeMode(mode, submode string) (string, string) {
	mode = strings.ToUpper(strings.TrimSpace(mode))

	if mapping, ok := importOnlyModes[mode]; ok {
		if strings.TrimSpace(submode) == "" {
			return mapping.mode, mapping.submode
		}
		return mapping.mode, strings.ToUpper(strings.TrimSpace(submode))
	}

	if submode != "" && strings.ToUpper(strings.TrimSpace(submode)) == mode {
		switch mode {
		case "SSB":
			return mode, ""
		case "CW":
			return mode, ""
		case "FM":
			return mode, ""
		case "AM":
			return mode, ""
		default:
			if _, ok := modeSubmodes[mode]; ok {
				return mode, ""
			}
		}
	}

	// ADIF 3.1.7: FT4 and FT2 are MFSK submodes, FT8 is a top-level mode.
	// Import leniently: accept standalone FT4/FT2 as legacy.
	if mode == "FT4" || mode == "FT2" {
		if strings.TrimSpace(submode) == "" {
			return "MFSK", mode
		}
		return "MFSK", strings.ToUpper(strings.TrimSpace(submode))
	}
	// Import leniently: accept MFSK+FT8 (non-standard) → normalize to FT8.
	if mode == "MFSK" && submode == "FT8" {
		return "FT8", ""
	}

	return mode, submode
}

func SubmodesFor(mode string) []string {
	submodes, ok := modeSubmodes[strings.ToUpper(mode)]
	if !ok {
		return nil
	}
	if len(submodes) == 0 {
		return nil
	}
	result := make([]string, len(submodes))
	copy(result, submodes)
	return result
}

func AllModes() []string {
	result := make([]string, 0, len(modeSubmodes))
	for m := range modeSubmodes {
		result = append(result, m)
	}
	return result
}

// CycleModes returns the short list of main modes for quick PgUp/PgDn cycling.
// Users can still type any valid mode manually; external software can set any mode.
var cycleModes = []string{"SSB", "CW", "AM", "FM", "RTTY", "DIGITALVOICE"}

func CycleModes() []string {
	return cycleModes
}

// NormalizeRigMode maps a raw mode string from any rig backend (flrig XML-RPC,
// hamlib rigctld) to a canonical ADIF mode token.  Unknown modes pass through
// unchanged.
func NormalizeRigMode(raw string) string {
	if m, ok := rigModeMap[strings.ToUpper(strings.TrimSpace(raw))]; ok {
		return m
	}
	return strings.ToUpper(strings.TrimSpace(raw))
}
