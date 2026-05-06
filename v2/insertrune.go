package main

import (
	"github.com/xyproto/vt"
)

// InsertRune will insert a rune at the current data position, with word wrap.
// Returns true if the line was wrapped.
func (e *Editor) InsertRune(c *vt.Canvas, r rune) bool {
	// Re-enable typo highlights when the user starts typing
	e.showTypoHighlights = true
	// Insert a regular space instead of a non-breaking space.
	// Nobody likes non-breaking spaces.
	// Normalise runes that look like ASCII punctuation so the user
	// doesn't accidentally commit them (they're invisible or ambiguous
	// in most fonts). The values used here are Unicode code points -
	// earlier revisions compared against UTF-8 byte sequences like
	// 0xC2A0, which never matched a decoded rune (NBSP is U+00A0,
	// decoded to rune 0x00A0 = 160, not 0xC2A0) so the substitutions
	// silently stopped working.
	switch r {
	case '\u00A0': // non-breaking space → regular space
		r = ' '
	case '\u0308': // combining diaeresis (sticky dead key)
		r = '~'
	case '\u037E': // Greek question mark (looks like ';')
		r = ';'
	case '\u0387': // Greek ano teleia (looks like ';' / '·')
		r = ';'
	case '\u00B7': // middle dot (looks like '·')
		r = '.'
	}

	e.MarkChanged()
	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	// Insert the rune first, then check if the line needs wrapping.
	// Capture line length before insertion so we only auto-fill when this
	// keystroke itself pushes the line past the limit (not for pre-existing
	// long lines that the user is editing).
	lenBefore := len([]rune(e.Line(e.DataY())))
	e.Insert(c, r)

	// Emacs auto-fill style: if this insertion caused the line to exceed
	// wrapWidth, break at the last space before the limit and move the tail
	// to a new line below.
	if e.wrapWhenTyping && e.wrapWidth > 0 && lenBefore <= e.wrapWidth {
		runes := []rune(e.Line(e.DataY()))
		if len(runes) > e.wrapWidth {
			breakAt := -1
			for i := e.wrapWidth - 1; i >= 0; i-- {
				if runes[i] == ' ' {
					breakAt = i
					break
				}
			}
			if breakAt >= 0 {
				// skip any run of spaces at the break point
				rightStart := breakAt + 1
				for rightStart < len(runes) && runes[rightStart] == ' ' {
					rightStart++
				}
				y := e.DataY()
				e.SetLine(y, string(runes[:breakAt]))
				e.InsertLineBelowAt(y)
				e.SetLine(y+1, string(runes[rightStart:]))
				// move cursor to the new line if it was in the wrapped portion
				cursorX := e.pos.sx + e.pos.offsetX
				if cursorX > breakAt {
					newX := max(cursorX-rightStart, 0)
					e.pos.sy++
					e.pos.sx = newX
					e.pos.offsetX = 0
				}
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
		}
	}

	// Scroll right when reaching 95% of the terminal width
	wf := 80.0
	if c != nil {
		wf = float64(c.Width())
	}
	if e.pos.sx > int(wf*0.95) {
		e.pos.offsetX++
		e.pos.sx--
	}
	return false
}
