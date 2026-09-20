package aprs

import "testing"

func TestCleanComment(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"73 \u2600\ufe0f GL!", "73 GL!"},
		{"\U0001F600 hello \U0001F60E", "hello"},
		{"snow \u2744 and rain \u2614 today", "snow and rain today"},
		{"\u27a1 145.500", "145.500"},
		{"plain ascii comment", "plain ascii comment"},
		{"", ""},
		{"   ", ""},
		{"a  b\tc", "a b c"},
		// Accented Latin and ordinary punctuation survive.
		{"Krak\u00f3w \u2013 J\u00f3zef, 73!", "Krak\u00f3w \u2013 J\u00f3zef, 73!"},
		{"\U0001F1F5\U0001F1F1 SP9MOA", "SP9MOA"}, // flag emoji pair
	}
	for _, c := range cases {
		if got := CleanComment(c.in); got != c.want {
			t.Errorf("CleanComment(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
