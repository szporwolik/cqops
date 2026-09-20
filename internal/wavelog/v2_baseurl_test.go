package wavelog

import "testing"

func TestV2BaseURL(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://qso.cqops.com", "https://qso.cqops.com/api/v2"},
		{"https://qso.cqops.com/", "https://qso.cqops.com/api/v2"},
		{"https://qso.cqops.com/api/v2", "https://qso.cqops.com/api/v2"},
		{"https://qso.cqops.com/api/v2/", "https://qso.cqops.com/api/v2"},
		{"https://qso.cqops.com/index.php/api/v2", "https://qso.cqops.com/index.php/api/v2"},
	}
	for _, c := range cases {
		if got := v2BaseURL(c.in); got != c.want {
			t.Errorf("v2BaseURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
