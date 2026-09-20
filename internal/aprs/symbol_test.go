package aprs

import "testing"

func TestSymbolName(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"/>", "Car"},
		{"/-", "House"},
		{"/#", "Digipeater"},
		{"/_", "Weather Station"},
		{"/k", "Truck"},
		{"/j", "Jeep"},
		{"/O", "Balloon"},
		{"/Y", "Sailboat"},
		{"/s", "Ship"},
		{"/b", "Bicycle"},
		{"/[", "Runner"},
		{"/r", "Repeater"},
		{"/P", "Police"},
		{"/h", "Hospital"},
		{"/X", "Helicopter"},
		{"/a", "Ambulance"},
		{"/f", "Fire Truck"},
		{"/^", "Aircraft"},
		{"/g", "Glider"},
		{"/$", "Phone"},
		{"/E", "Special Event"},
		{"/W", "WX Site"},
		{"/!", "Police"},
		{"/v", "Van"},
		{"/n", "Node"},
		{"/R", "RV"},
		{"/K", "School"},
		{"/@", "Hurricane"},
		{"/p", "Dog"},
		{"/A", "Aid Station"},
		{"/+", "Red Cross"},
		{"/&", "HF Gateway"},
		{"/%", "DX Cluster"},
		{"/(", "Satellite"},
		{"/'", "Small Aircraft"},
		{"/*", "Snowmobile"},
	}

	for _, tc := range cases {
		if got := SymbolName(tc.code); got != tc.want {
			t.Errorf("SymbolName(%q) = %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestSymbolName_Fallback(t *testing.T) {
	for _, code := range []string{"", "/", "/Q", "\\q", "3>", "Z>", "/~"} {
		if got := SymbolName(code); got != "" {
			t.Errorf("SymbolName(%q) = %q, want empty fallback", code, got)
		}
	}
}
