package aprs

import "testing"

func TestPacketType(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		// Mic-E (both data type indicators).
		{"SP9ABC>APRS,WIDE1-1:`&Kl n>/\"=!}Test", "E"},
		{"SP9ABC>APRS,WIDE1-1:'&Kl n>/\"=!}Test", "E"},
		// Uncompressed positions.
		{"SP9ABC>APRS,TCPIP*:!5023.45N/02012.34E>Test", "P"},
		{"SP9ABC>APRS,TCPIP*:=5023.45N/02012.34E>Test", "P"},
		{"SP9ABC>APRS,TCPIP*:@091425z5023.45N/02012.34E>Test", "P"},
		// Compressed positions (symbol-table character after the indicator).
		{"SP9ABC>APRS,TCPIP*:!/5L!!<*e7>Test", "p"},
		{"SP9ABC>APRS,TCPIP*://5L!!<*e7>Test", "p"},
		// Object / Item / Weather / Status / NMEA.
		{";OBJNAME*091425z5023.45N/02012.34E>Test", "O"},
		{")ITEM!5023.45N/02012.34E>Test", "I"},
		{"SP9ABC>APRS,TCPIP*:_091425z5023.45N/02012.34E>Test", "W"},
		{"SP9ABC>APRS,TCPIP*:>Status text", "S"},
		{"SP9ABC>APRS,TCPIP*:$GPRMC,091425,A,5023.45,N,02012.34,E", "G"},
		// Unknown / malformed.
		{"SP9ABC>APRS,TCPIP*:XYZ", "?"},
		{"", ""},
		{"no-colon-packet", "?"},
	}
	for _, tc := range cases {
		if got := PacketType(tc.raw); got != tc.want {
			t.Errorf("PacketType(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
