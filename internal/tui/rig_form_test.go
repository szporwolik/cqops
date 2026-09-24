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
	// Sub-options sit flush under their selector — no blank line.
	if host != radio+1 {
		t.Errorf("unexpected blank line before Hamlib sub-options (radio=%d, host=%d)", radio, host)
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
	// Sub-options sit flush under their selector — no blank line.
	if host != rotor+1 {
		t.Errorf("unexpected blank line before Hamlib sub-options (rotor=%d, host=%d)", rotor, host)
	}
	// The port row is the last sub-option; the checkbox follows directly.
	if wsjtx != host+2 {
		t.Errorf("unexpected blank line(s) after rotor sub-options (host=%d, wsjtx=%d)", host, wsjtx)
	}
}

func TestRigFormView_ExpandedWsjtxSpacing(t *testing.T) {
	f := NewRigForm("", "", "")
	f.width = 100
	f.WsjtxEnabled = true
	lines := strings.Split(fmt.Sprint(f.View()), "\n")

	wsjtx := lineIndex(lines, "Use WSJT-X:")
	host := lineIndex(lines, "UDP Host:")
	port := lineIndex(lines, "UDP Port:")
	if wsjtx < 0 || host < 0 || port < 0 {
		t.Fatalf("rows missing: wsjtx=%d host=%d port=%d", wsjtx, host, port)
	}
	// UDP fields sit flush under the checkbox.
	if host != wsjtx+1 {
		t.Errorf("unexpected blank line before UDP Host (wsjtx=%d, host=%d)", wsjtx, host)
	}
	if port != host+1 {
		t.Errorf("unexpected blank line between UDP Host and UDP Port (host=%d, port=%d)", host, port)
	}
}
