package main

import (
	"github.com/xyproto/vt"
)

// InsertRune will insert a rune at the current data position, with word wrap
// Returns true if the line was wrapped
func (e *Editor) InsertRune(c *vt.Canvas, r rune) bool {
	// Insert a regular space instead of a non-breaking space.
	// Nobody likes non-breaking spaces.

	if r == 0xc2a0 { // non-breaking space
		r = ' '
	} else if r == 0xcc88 { // annoying tilde
		r = '~'
	} else if r == 0xcdbe { // greek question mark
		r = ';'
	}

	// The document will be changed
	e.changed.Store(true)

	// Repaint afterwards
	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	// Wrap a bit before the limit if the inserted rune is a space?
	limit := e.wrapWidth
	if r == ' ' {
		// This is how long a word can be before being broken,
		// and it "eats from" the right margin, so it needs to be balanced.
		// TODO: Use an established method of word wrapping / breaking lines.
		limit -= 5
	}

	// If wrapWhenTyping is enabled, check if we should wrap to the next line
	if e.wrapWhenTyping && e.wrapWidth > 0 && e.pos.sx >= limit {

		e.InsertLineBelow()
		e.pos.sy++
		y := e.pos.sy
		e.Home()
		e.pos.sx = 0
		if r != ' ' {
			e.Insert(c, r)
			e.pos.sx = 1
		}
		e.pos.sy = y

		h := 80
		if c != nil {
			h = int(c.Height())
		}
		if e.pos.sy >= (h - 1) {
			e.ScrollDown(c, nil, 1, h)
		}

		e.Center(c)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)

		return true
	}

	// A regular insert, no wrapping
	e.Insert(c, r)

	// Scroll right when reaching 95% of the terminal width
	wf := 80.0
	if c != nil {
		wf = float64(c.Width())
	}
	if e.pos.sx > int(wf*0.95) {
		// scroll
		e.pos.offsetX++
		e.pos.sx--
	}
	return false
}
