package main

import (
	"github.com/xyproto/vt100"
)

// InsertRune will insert a rune at the current data position, with word wrap
// Returns true if the line was wrapped
func (e *Editor) InsertRune(c *vt100.Canvas, r rune) bool {
	// Insert a regular space instead of a nonbreaking space.
	// Nobody likes nonbreaking spaces.
	if r == 0xc2a0 {
		r = ' '
	} else if r == 0xcc88 {
		r = '~'
	}

	// The document will be changed
	e.changed = true

	// Repaint afterwards
	e.redrawCursor = true
	e.redraw = true

	// If wrapWhenTyping is enabled, check if we should wrap to the next line
	if e.wrapWhenTyping && e.wrapWidth > 0 && e.pos.sx >= e.wrapWidth {
		e.InsertLineBelow()
		e.pos.sy++
		y := e.pos.sy
		e.Home()
		e.pos.sx = 0
		if r != ' ' {
			e.Insert(r)
			e.pos.sx = 1
		}
		e.pos.sy = y

		h := 80
		if c != nil {
			h = int(c.Height())
		}
		if e.pos.sy >= (h - 1) {
			e.ScrollDown(c, nil, 1)
		}
		return true
	}

	// A regular insert, no wrapping
	e.Insert(r)

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
