package tui

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

// focusableRows is implemented by every config menu and form. It describes
// the focusable rows so the shared focus engine (menuFocus) can own
// navigation and the Save & Back button with one set of invariants:
//
//   - exactly one active marker is visible (row or button, never both),
//   - moving focus always blurs every textinput first,
//   - hidden rows never receive focus,
//   - reopening a form always resets to the first row.
//
// Row indices are menu-specific but must be stable and sequential.
type focusableRows interface {
	// rowCount returns the number of row slots (including hidden ones).
	rowCount() int
	// rowVisible reports whether row i is currently rendered/focusable.
	rowVisible(i int) bool
	// blurAll removes textinput focus from every field in the menu.
	blurAll()
	// focusRow activates row i (focuses its textinput if it has one).
	focusRow(i int) tea.Cmd
}

// menuFocus owns the active row (or the Save & Back button) for a config
// menu. row == -1 means the button is focused.
type menuFocus struct {
	row int
	btn saveBackButton
}

// reset returns focus to the first row with the button unfocused.
// Call it whenever the menu (re)opens.
func (f *menuFocus) reset() {
	f.row = 0
	f.btn.Focus = false
}

// focusButton hands focus to the Save & Back button, leaving the list.
func (f *menuFocus) focusButton(m focusableRows) {
	m.blurAll()
	f.row = -1
	f.btn.Focus = true
}

// leaveButtonForward moves focus from the button to the first visible row.
func (f *menuFocus) leaveButtonForward(m focusableRows) tea.Cmd {
	f.btn.Focus = false
	f.row = f.firstVisible(m)
	m.blurAll()
	return m.focusRow(f.row)
}

// leaveButtonBack moves focus from the button to the last visible row.
func (f *menuFocus) leaveButtonBack(m focusableRows) tea.Cmd {
	f.btn.Focus = false
	f.row = f.lastVisible(m)
	m.blurAll()
	return m.focusRow(f.row)
}

// firstVisible returns the lowest visible row index.
func (f *menuFocus) firstVisible(m focusableRows) int {
	for i := 0; i < m.rowCount(); i++ {
		if m.rowVisible(i) {
			return i
		}
	}
	return 0
}

// lastVisible returns the highest visible row index.
func (f *menuFocus) lastVisible(m focusableRows) int {
	for i := m.rowCount() - 1; i >= 0; i-- {
		if m.rowVisible(i) {
			return i
		}
	}
	return 0
}

// onFirst reports whether the first visible row is active.
func (f *menuFocus) onFirst(m focusableRows) bool {
	return f.row == f.firstVisible(m)
}

// onLast reports whether the last visible row is active.
func (f *menuFocus) onLast(m focusableRows) bool {
	return f.row == f.lastVisible(m)
}

// next moves focus to the next visible row, wrapping around.
func (f *menuFocus) next(m focusableRows) tea.Cmd {
	n := m.rowCount()
	if n <= 0 {
		return nil
	}
	for step := 1; step <= n; step++ {
		nxt := (f.row + step) % n
		if m.rowVisible(nxt) {
			f.row = nxt
			m.blurAll()
			return m.focusRow(nxt)
		}
	}
	return nil
}

// prev moves focus to the previous visible row, wrapping around.
func (f *menuFocus) prev(m focusableRows) tea.Cmd {
	n := m.rowCount()
	if n <= 0 {
		return nil
	}
	for step := 1; step <= n; step++ {
		prv := ((f.row-step)%n + n) % n
		if m.rowVisible(prv) {
			f.row = prv
			m.blurAll()
			return m.focusRow(prv)
		}
	}
	return nil
}

// fixFocus advances to the next visible row when the current row has become
// hidden (e.g. after toggling a section off).
func (f *menuFocus) fixFocus(m focusableRows) {
	if m.rowVisible(f.row) {
		return
	}
	_ = f.next(m)
}

// scrollFraction returns 0.0 (top) to 1.0 (bottom) for the current row's
// rank among the visible rows — used to keep the active row visible while
// scrolling a viewport.
func (f *menuFocus) scrollFraction(m focusableRows) float64 {
	visible := 0
	rank := -1
	for i := 0; i < m.rowCount(); i++ {
		if m.rowVisible(i) {
			visible++
		}
		if i == f.row {
			rank = visible
		}
	}
	if visible <= 1 || rank <= 0 {
		return 0
	}
	return float64(rank-1) / float64(visible-1)
}

// scrollToFocusedLine clamps the viewport offset so the line carrying the
// focus marker (the FormPrefixOn "> " prefix) stays inside the visible
// window. Unlike a fractional mapping it tolerates hidden rows, multi-line
// bodies and resize changes. content is the raw styled body passed to
// SetContent.
func scrollToFocusedLine(vp *viewport.Model, content string) {
	marker := S.FormPrefixOn.Render("> ")
	focus := -1
	for i, ln := range strings.Split(content, "\n") {
		if strings.Contains(ln, marker) {
			focus = i
			break
		}
	}
	if focus < 0 {
		return
	}
	visible := vp.VisibleLineCount()
	off := vp.YOffset()
	if focus < off {
		off = focus
	} else if focus >= off+visible {
		off = focus - visible + 1
	}
	if off < 0 {
		off = 0
	}
	vp.SetYOffset(off)
}

// scrollViewportToFraction sets a viewport's Y offset so that the row at the
// given 0..1 fraction stays visible.
func scrollViewportToFraction(vp *viewport.Model, frac float64) {
	total := vp.TotalLineCount()
	visible := vp.VisibleLineCount()
	if total <= visible {
		vp.SetYOffset(0)
		return
	}
	maxOffset := total - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	offset := int(float64(maxOffset) * frac)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	vp.SetYOffset(offset)
}

// onKey handles the navigation keys shared by every config menu:
// Tab/Down moves to the next row, Shift+Tab/Up to the previous one, and the
// Save & Back button is reachable from the last row with Tab and from the
// first row with Shift+Tab. While the button is focused, Space/Enter invoke
// save and every other key is swallowed (Esc falls through to the caller).
//
// It returns handled=true when the key was consumed. The caller handles
// Esc, per-row Space/Enter actions, and its own save command via save.
func (f *menuFocus) onKey(k tea.KeyPressMsg, m focusableRows, save func() tea.Cmd) (bool, tea.Cmd) {
	if f.btn.Focus {
		switch {
		case f.btn.activate(k):
			return true, save()
		case f.btn.next(k):
			return true, f.leaveButtonForward(m)
		case f.btn.prev(k):
			return true, f.leaveButtonBack(m)
		case k.String() == "esc":
			return false, nil // caller handles Esc (back without saving)
		default:
			return true, nil // swallow keys while the button is focused
		}
	}
	if f.btn.next(k) && f.onLast(m) {
		f.focusButton(m)
		return true, nil
	}
	if f.btn.prev(k) && f.onFirst(m) {
		f.focusButton(m)
		return true, nil
	}
	if f.btn.next(k) {
		return true, f.next(m)
	}
	if f.btn.prev(k) {
		return true, f.prev(m)
	}
	return false, nil
}
