package dashboard

import (
	"strings"
	"testing"
)

// TestSolarModulePlaceholderAndScale pins the two defects fixed in the Solar
// Activity module:
//
//  1. A missing SFI (the hamqsl feed omits <solarflux> between daily updates)
//     must render as a muted dash. The old helper scored 0 as "0 <= 100" and
//     painted the dash green, i.e. it claimed perfect conditions while showing
//     no value at all.
//  2. SFI is the opposite of the geomagnetic indices — higher is better. The
//     old "<=100 green, >150 red" scale painted excellent propagation red.
//
// The markers below are the load-bearing pieces of that behaviour; renaming
// them is fine as long as an equivalent assertion replaces this test.
func TestSolarModulePlaceholderAndScale(t *testing.T) {
	js, err := staticFS.ReadFile("static/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	css, err := staticFS.ReadFile("static/style.css")
	if err != nil {
		t.Fatalf("read style.css: %v", err)
	}
	appJS := string(js)
	styleCSS := string(css)

	if !strings.Contains(appJS, `class="xmod-na"`) {
		t.Error("a missing solar value must render the muted xmod-na placeholder")
	}
	if strings.Contains(appJS, "cond(sf,100,150)") {
		t.Error("the inverted SFI scale is back (<=100 green, >150 red)")
	}
	if !strings.Contains(appJS, "function sfiCls(") {
		t.Error("SFI scoring helper missing")
	}

	for _, cls := range []string{".xmod-ok{", ".xmod-warn{", ".xmod-bad{", ".xmod-na{"} {
		if !strings.Contains(styleCSS, cls) {
			t.Errorf("style.css is missing %s", cls)
		}
	}
}
