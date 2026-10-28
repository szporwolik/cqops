package tui

import (
	"testing"

	"github.com/NimbleMarkets/ntcharts/v2/picture"
)

// TestKittyPicturesFillTheirBox: ntcharts v2.4+ made FitContain (letterbox) the
// default for Kitty images, while the glyph path and both of CQOps' image
// pipelines draw to fill the box — the map buffer is built at exactly the
// placement rectangle's pixel size, and the partner photo has always filled its
// frame. After the v2.2.0 → v2.8.0 bump an untouched config therefore started
// letterboxing the photo inside its box (a 4:3 photo in a 200×64 cell box
// covered ~42% of the width, the rest background), which reads as a broken
// image. Every picture model states fill explicitly now, so the drawing does not
// depend on which way the library default moves next.
func TestKittyPicturesFillTheirBox(t *testing.T) {
	mr := newMapRenderer()
	if got := mr.kittyPic.Fit(); got != picture.FitFill {
		t.Errorf("partner map fill mode = %v, want FitFill", got)
	}
	if got := mr.pskKittyPic.Fit(); got != picture.FitFill {
		t.Errorf("PSK reporter map fill mode = %v, want FitFill", got)
	}

	m := newLifecycleTestModel(t)
	if got := m.photo.viewer.Fit(); got != picture.FitFill {
		t.Errorf("full-screen photo fill mode = %v, want FitFill", got)
	}
	if got := m.photo.partnerPicViewer.Fit(); got != picture.FitFill {
		t.Errorf("inline partner photo fill mode = %v, want FitFill", got)
	}
}

// TestKittyForceCapableCoversOurTerminals: the capability is declared at startup
// for every terminal CQOps claims to support, so it does not depend on the
// library's 250 ms probe round-trip. WezTerm was in the probe gate but missing
// from the force list, which left it the only supported terminal whose Kitty
// path could stay blocked forever.
func TestKittyForceCapableCoversOurTerminals(t *testing.T) {
	for _, v := range []string{"KITTY_WINDOW_ID", "KITTY_INSTALLATION_DIR", "GHOSTTY_RESOURCES_DIR",
		"WEZTERM_EXECUTABLE", "WEZTERM_PANE", "TERM_PROGRAM", "TERM", "NTCHARTS_KITTY"} {
		t.Setenv(v, "")
	}
	clear := func() {
		for _, v := range []string{"KITTY_WINDOW_ID", "KITTY_INSTALLATION_DIR", "GHOSTTY_RESOURCES_DIR",
			"WEZTERM_EXECUTABLE", "WEZTERM_PANE", "TERM_PROGRAM", "TERM", "NTCHARTS_KITTY"} {
			t.Setenv(v, "")
		}
	}

	cases := []struct {
		name string
		env  map[string]string
		want bool
	}{
		// WezTerm renders images but not the Unicode placeholders every
		// ntcharts picture places them with (wezterm#7807, draft PR #7924):
		// it drew the map as a grid of green U+10EEEE boxes with the image in
		// a stray strip, both before and after the v2.2.0 → v2.8.0 bump.
		{"wezterm binary", map[string]string{"WEZTERM_EXECUTABLE": "/usr/bin/wezterm"}, false},
		{"wezterm pane", map[string]string{"WEZTERM_PANE": "3"}, false},
		{"wezterm TERM_PROGRAM", map[string]string{"TERM_PROGRAM": "WezTerm"}, false},
		{"wezterm overridden", map[string]string{"WEZTERM_EXECUTABLE": "/usr/bin/wezterm", "NTCHARTS_KITTY": "supported"}, true},
		{"kitty", map[string]string{"KITTY_WINDOW_ID": "1"}, true},
		{"kitty terminfo", map[string]string{"TERM": "xterm-kitty"}, true},
		{"ghostty", map[string]string{"GHOSTTY_RESOURCES_DIR": "/usr/share/ghostty"}, true},
		{"ghostty terminfo", map[string]string{"TERM": "xterm-ghostty"}, true},
		{"explicit override", map[string]string{"NTCHARTS_KITTY": "supported"}, true},
		{"plain xterm", map[string]string{"TERM": "xterm-256color"}, false},
		{"iterm", map[string]string{"TERM_PROGRAM": "iTerm.app"}, false},
		{"konsole", map[string]string{"TERM": "xterm-256color", "KONSOLE_VERSION": "240800"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clear()
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if got := kittyForceCapable(); got != tc.want {
				t.Errorf("kittyForceCapable() = %v, want %v for %v", got, tc.want, tc.env)
			}
			// Everything declared capable must also pass the probe gate, or the
			// model would force the capability and then refuse to switch modes.
			if tc.want && !kittyTerminalEnv() {
				t.Errorf("kittyTerminalEnv() = false for %v, but the capability is force-enabled", tc.env)
			}
		})
	}
	clear()
}

// TestKittyBlockedIsReportedOnce: a configured Kitty mode that never engages is
// logged once (with the library's reason), not on every frame the check runs.
func TestKittyBlockedIsReportedOnce(t *testing.T) {
	mr := newMapRenderer()
	if mr.kittyBlockedLogged {
		t.Fatal("a fresh renderer has already reported the blocked state")
	}
	mr.NoteKittyBlocked()
	if !mr.kittyBlockedLogged {
		t.Error("the blocked state was not recorded")
	}
	// A second call is a no-op — the flag is what keeps the tick quiet.
	mr.NoteKittyBlocked()

	// Once Kitty is actually on, there is nothing to report.
	on := newMapRenderer()
	on.kittyOn = true
	on.NoteKittyBlocked()
	if on.kittyBlockedLogged {
		t.Error("an active Kitty map was reported as blocked")
	}
}
