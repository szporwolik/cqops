package tui

import (
	"fmt"
	"strings"
	"testing"
)

// lineIndex returns the first view line containing sub, or -1.
func lineIndex(lines []string, sub string) int {
	for i, l := range lines {
		if strings.Contains(l, sub) {
			return i
		}
	}
	return -1
}

func TestRigFormView_NoBlankLinesBetweenRows(t *testing.T) {
	f := NewRigForm("", "", "")
	f.width = 100
	lines := strings.Split(fmt.Sprint(f.View()), "\n")

	radio := lineIndex(lines, "Radio control:")
	rotor := lineIndex(lines, "Rotator control:")
	wsjtx := lineIndex(lines, "Use WSJT-X:")
	if radio < 0 || rotor < 0 || wsjtx < 0 {
		t.Fatalf("section rows missing: radio=%d rotor=%d wsjtx=%d", radio, rotor, wsjtx)
	}
	if rotor != radio+1 {
		t.Errorf("unexpected blank line(s) between Radio control and Rotator control (radio=%d, rotor=%d)", radio, rotor)
	}
	if wsjtx != rotor+1 {
		t.Errorf("unexpected blank line(s) between Rotator control and Use WSJT-X (rotor=%d, wsjtx=%d)", rotor, wsjtx)
	}
}

func TestRigFormView_ExpandedBackendSpacing(t *testing.T) {
	f := NewRigForm("", "", "")
	f.width = 100
	f.BackendIdx = 1 // Hamlib — sub-options appear
	lines := strings.Split(fmt.Sprint(f.View()), "\n")

	radio := lineIndex(lines, "Radio control:")
	host := lineIndex(lines, "Hamlib host:")
	poll := lineIndex(lines, "Poll (s):")
	rotor := lineIndex(lines, "Rotator control:")
	if radio < 0 || host < 0 || poll < 0 || rotor < 0 {
		t.Fatalf("rows missing: radio=%d host=%d poll=%d rotor=%d", radio, host, poll, rotor)
	}
	// One separator blank line between the selector and its sub-options.
	if host != radio+2 {
		t.Errorf("expected one blank line before Hamlib sub-options (radio=%d, host=%d)", radio, host)
	}
	// No blank line between the last sub-option and the next section.
	if rotor != poll+1 {
		t.Errorf("unexpected blank line(s) after Poll row (poll=%d, rotor=%d)", poll, rotor)
	}
}

func TestRigFormView_ExpandedRotorSpacing(t *testing.T) {
	f := NewRigForm("", "", "")
	f.width = 100
	f.RotorIdx = 1 // Hamlib — sub-options appear
	lines := strings.Split(fmt.Sprint(f.View()), "\n")

	rotor := lineIndex(lines, "Rotator control:")
	host := lineIndex(lines, "Hamlib host:")
	wsjtx := lineIndex(lines, "Use WSJT-X:")
	if rotor < 0 || host < 0 || wsjtx < 0 {
		t.Fatalf("rows missing: rotor=%d host=%d wsjtx=%d", rotor, host, wsjtx)
	}
	// One separator blank line between the selector and its sub-options.
	if host != rotor+2 {
		t.Errorf("expected one blank line before Hamlib sub-options (rotor=%d, host=%d)", rotor, host)
	}
	// The port row is the last sub-option; the checkbox follows directly.
	if wsjtx != host+2 {
		t.Errorf("unexpected blank line(s) after rotor sub-options (host=%d, wsjtx=%d)", host, wsjtx)
	}
}
