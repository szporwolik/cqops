package tui

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
)

// =============================================================================
// wrapNext
// =============================================================================

func TestWrapNext(t *testing.T) {
	tests := []struct {
		current, count, want int
	}{
		// Normal increment
		{0, 5, 1},
		{1, 5, 2},
		{3, 5, 4},
		// Wrap from last to first
		{4, 5, 0},
		{2, 3, 0},
		// Count of 1 — always wraps to 0
		{0, 1, 0},
		// Large count
		{98, 100, 99},
		{99, 100, 0},
		{0, 100, 1},
	}
	for _, tt := range tests {
		got := wrapNext(tt.current, tt.count)
		if got != tt.want {
			t.Errorf("wrapNext(%d, %d) = %d; want %d", tt.current, tt.count, got, tt.want)
		}
	}
}

// =============================================================================
// wrapPrev
// =============================================================================

func TestWrapPrev(t *testing.T) {
	tests := []struct {
		current, count, want int
	}{
		// Normal decrement
		{1, 5, 0},
		{3, 5, 2},
		{4, 5, 3},
		// Wrap from first to last
		{0, 5, 4},
		{0, 3, 2},
		// Count of 1 — always wraps to 0
		{0, 1, 0},
		// Large count
		{0, 100, 99},
		{50, 100, 49},
	}
	for _, tt := range tests {
		got := wrapPrev(tt.current, tt.count)
		if got != tt.want {
			t.Errorf("wrapPrev(%d, %d) = %d; want %d", tt.current, tt.count, got, tt.want)
		}
	}
}

// =============================================================================
// wrapNext + wrapPrev round-trip
// =============================================================================

func TestWrapNextPrevRoundTrip(t *testing.T) {
	for count := 1; count <= 10; count++ {
		for start := 0; start < count; start++ {
			// Next then prev should return to start
			pos := wrapNext(start, count)
			pos = wrapPrev(pos, count)
			if pos != start {
				t.Errorf("round-trip failed: count=%d start=%d got=%d", count, start, pos)
			}
			// Prev then next should return to start
			pos = wrapPrev(start, count)
			pos = wrapNext(pos, count)
			if pos != start {
				t.Errorf("round-trip failed: count=%d start=%d got=%d", count, start, pos)
			}
		}
	}
}

// =============================================================================
// blurTextinputs
// =============================================================================

func TestBlurTextinputs(t *testing.T) {
	// Create two textinputs, focus one, call blurTextinputs, verify both blurred.
	a := textinput.New()
	b := textinput.New()
	a.Focus()       // a is focused
	b.Blur()        // b is not focused (just to be explicit)

	if !a.Focused() {
		t.Fatal("expected a to be focused before blurTextinputs")
	}

	blurTextinputs(&a, &b)

	if a.Focused() {
		t.Error("expected a to be blurred after blurTextinputs")
	}
	if b.Focused() {
		t.Error("expected b to remain blurred after blurTextinputs")
	}
}

func TestBlurTextinputs_Empty(t *testing.T) {
	// Should not panic with no arguments.
	blurTextinputs()
}

func TestBlurTextinputs_Single(t *testing.T) {
	ti := textinput.New()
	ti.Focus()
	if !ti.Focused() {
		t.Fatal("expected ti to be focused")
	}
	blurTextinputs(&ti)
	if ti.Focused() {
		t.Error("expected ti to be blurred")
	}
}
