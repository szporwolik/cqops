package aprs

// SymbolName returns a human-readable name for a 2-character APRS symbol
// code ("table" + "code", e.g. "/>" for a car). Returns "" for unknown
// or alternate-table symbols — callers fall back to the raw code.
//
// Curated subset of the APRS 1.2 symbol tables covering the most common
// primary-table symbols. Secondary and alternate overlay tables fall
// back to the raw code rather than risk a wrong name.
func SymbolName(code string) string {
	if len(code) < 2 {
		return ""
	}
	table, sym := code[0], code[1]
	if table != '/' && table != '\\' {
		return "" // alternate/overlay tables — not mapped
	}

	// Primary table — common symbols only.
	if table == '/' {
		switch sym {
		case '!':
			return "Police"
		case '#':
			return "Digipeater"
		case '$':
			return "Phone"
		case '%':
			return "DX Cluster"
		case '&':
			return "HF Gateway"
		case '\'':
			return "Small Aircraft"
		case '(':
			return "Satellite"
		case '*':
			return "Snowmobile"
		case '+':
			return "Red Cross"
		case '-':
			return "House"
		case '>':
			return "Car"
		case '@':
			return "Hurricane"
		case 'A':
			return "Aid Station"
		case 'E':
			return "Special Event"
		case 'K':
			return "School"
		case 'O':
			return "Balloon"
		case 'P':
			return "Police"
		case 'R':
			return "RV"
		case 'W':
			return "WX Site"
		case 'X':
			return "Helicopter"
		case 'Y':
			return "Sailboat"
		case '[':
			return "Runner"
		case '^':
			return "Aircraft"
		case '_':
			return "Weather Station"
		case 'a':
			return "Ambulance"
		case 'b':
			return "Bicycle"
		case 'f':
			return "Fire Truck"
		case 'g':
			return "Glider"
		case 'h':
			return "Hospital"
		case 'j':
			return "Jeep"
		case 'k':
			return "Truck"
		case 'n':
			return "Node"
		case 'p':
			return "Dog"
		case 'r':
			return "Repeater"
		case 's':
			return "Ship"
		case 'v':
			return "Van"
		}
		return ""
	}

	// Secondary table — the few well-known entries.
	switch sym {
	case '>':
		return "Car (alternate)"
	case 'O':
		return "Balloon (alternate)"
	}
	return ""
}
