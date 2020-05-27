package main

import (
	"github.com/xyproto/vt100"
)

// InsertRune will insert a rune at the current data position, with word wrap
func (e *Editor) InsertRune(c *vt100.Canvas, r rune) {
	// Insert a regular space instead of a nonbreaking space.
	// Nobody likes nonbreaking spaces.
	if r == 0xc2a0 {
		r = ' '
	}

	// The document will be changed
	e.changed = true

	// --- Repaint afterwards ---
	e.redrawCursor = true
	e.redraw = true

	// Disable word wrap completely, for now.
	// TODO: Rewrite the InsertRune function
	e.Insert(r)

	wf := float64(c.Width())
	// Scroll right when reaching 95% of the terminal width
	if e.pos.sx > int(wf*0.95) {
		// scroll
		e.pos.offsetX++
		e.pos.sx--
	}
}
