package main

import (
	"github.com/xyproto/vt"
)

// InsertRune will insert a rune at the current data position, with word wrap
// Returns true if the line was wrapped
func (e *Editor) InsertRune(c *vt.Canvas, r rune) bool {
	// Re-enable typo highlights when the user starts typing
	e.showTypoHighlights = true
	// Insert a regular space instead of a non-breaking space.
	// Nobody likes non-breaking spaces.
	// Normalise runes that look like ASCII punctuation so the user
	// doesn't accidentally commit them (they're invisible or ambiguous
	// in most fonts). The values used here are Unicode code points —
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

	// The document will be changed
	e.MarkChanged()

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

	// If wrapWhenTyping is enabled, check if we should wrap to the next line.
	// Never wrap immediately before a character that must follow its preceding
	// word (apostrophe, closing punctuation, etc.) — e.g. typing 's should
	// not break the line before the apostrophe.
	if e.wrapWhenTyping && e.wrapWidth > 0 && e.pos.sx >= limit && !noLineStartRune(r) {

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
