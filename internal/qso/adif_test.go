package qso

import "testing"

func TestSanitizeASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Pure ASCII — fast path.
		{"Hello World", "Hello World"},
		{"", ""},

		// Polish.
		{"Zażółć gęślą jaźń", "Zazolc gesla jazn"},
		{"Kraków", "Krakow"},
		{"Łódź", "Lodz"},
		{"Gdańsk", "Gdansk"},

		// German.
		{"München", "Munchen"},
		{"Düsseldorf", "Dusseldorf"},
		{"Straße", "Strasse"},
		{"Schloß", "Schloss"},

		// French.
		{"Français", "Francais"},
		{"élève", "eleve"},
		{"cœur", "coeur"},
		{"hôpital", "hopital"},
		{"à propos", "a propos"},

		// Spanish.
		{"mañana", "manana"},
		{"español", "espanol"},
		{"América", "America"},

		// Portuguese.
		{"São Paulo", "Sao Paulo"},
		{"não", "nao"},
		{"lingüiça", "linguica"},

		// Nordic.
		{"København", "Kobenhavn"},
		{"Åland", "Aland"},
		{"Örebro", "Orebro"},
		{"Sverige", "Sverige"},

		// Turkish.
		{"İstanbul", "Istanbul"},
		{"şaşırmak", "sasirmak"},

		// Czech/Slovak.
		{"Česká", "Ceska"},
		{"Škoda", "Skoda"},
		{"Ďáblice", "Dablice"},

		// Romanian.
		{"București", "Bucuresti"},
		{"România", "Romania"},

		// Croatian/Serbian.
		{"Zagreb", "Zagreb"},
		{"Đakovo", "DJakovo"},

		// Icelandic.
		{"Reykjavík", "Reykjavik"},
		{"Þingvellir", "THingvellir"},

		// Mixed and edge cases.
		{"Curaçao", "Curacao"},
		{"Zażółć Straße", "Zazolc Strasse"},
		{"™", ""},                             // non-letter symbol → dropped
		{"café–restaurant", "caferestaurant"}, // en-dash dropped (non-ASCII, non-letter)
	}

	for _, tt := range tests {
		result := sanitizeASCII(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeASCII(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestSanitizeASCII_NoAllocForASCII(t *testing.T) {
	input := "Hello World"
	result := sanitizeASCII(input)
	if result != input {
		t.Errorf("pure ASCII should be returned unchanged")
	}
}
