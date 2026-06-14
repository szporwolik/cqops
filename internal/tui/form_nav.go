package tui

import "charm.land/bubbles/v2/textinput"

// =============================================================================
// Shared form/menu navigation helpers
// =============================================================================
//
// These tiny helpers eliminate duplicated index-wrapping and textinput-blur
// patterns across the config menus and forms. They are intentionally simple:
// no generics needed (the typed-int callers accept a brief cast), no new
// abstractions, no behavioral changes.
//
// Keep screen-specific logic (skip rules, retain-focused, auto-fill, state
// resets) local to each component. These helpers only cover the mechanically
// repeated parts.

// wrapNext returns (current+1) % count — the standard forward-wrapping index.
func wrapNext(current, count int) int {
	return (current + 1) % count
}

// wrapPrev returns (current-1+count) % count — the standard backward-wrapping
// index. This replaces the common manual if-at-zero-else-decrement pattern.
func wrapPrev(current, count int) int {
	return (current - 1 + count) % count
}

// blurTextinputs calls Blur() on each supplied textinput pointer. Use this
// instead of repeating .Blur() for every field in a form's blurAll method.
func blurTextinputs(tis ...*textinput.Model) {
	for _, ti := range tis {
		ti.Blur()
	}
}
