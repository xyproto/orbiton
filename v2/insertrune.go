package main

import (
	"strings"

	"github.com/xyproto/vt"
	"github.com/xyproto/wordwrap"
)

// reFlowForward merges overspill text into line lineIdx and cascades re-wrapping
// downward through the paragraph. A blank line or end-of-document acts as a
// paragraph boundary: a new line is inserted for the overspill rather than
// merging with the existing next line.
func (e *Editor) reFlowForward(overspill string, lineIdx LineIndex, wrapLimit int) {
	if strings.TrimSpace(overspill) == "" {
		return
	}
	if int(lineIdx) >= e.Len() {
		e.InsertLineBelowAt(lineIdx - 1)
		e.SetLine(lineIdx, overspill)
		return
	}
	existing := e.Line(lineIdx)
	if strings.TrimSpace(existing) == "" {
		// Paragraph boundary -- insert before the blank line.
		e.InsertLineBelowAt(lineIdx - 1)
		e.SetLine(lineIdx, overspill)
		return
	}
	// Same paragraph: prepend overspill to existing line and re-wrap downward.
	merged := overspill + " " + existing
	result := wordwrap.WrapLine(merged, wrapLimit, 0)
	if result.Wrapped {
		e.SetLine(lineIdx, result.Left)
		e.reFlowForward(result.Right, lineIdx+1, wrapLimit)
	} else {
		e.SetLine(lineIdx, merged)
	}
}

// InsertRune will insert a rune at the current data position, with wrap when typing.
// Returns true if the cursor was repositioned to a new (wrapped) line.
// When true the caller must NOT advance the cursor; when false the caller
// should call Next as usual.
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
	case '\u00A0': // non-breaking space -> regular space
		r = ' '
	case '\u0308': // combining diaeresis (sticky dead key)
		r = '~'
	case '\u037E': // Greek question mark (looks like ';')
		r = ';'
	case '\u0387': // Greek ano teleia (looks like ';' / '.')
		r = ';'
	case '\u00B7': // middle dot (looks like '.')
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
	// the typing wrap limit, break at the last space before the limit and
	// move the tail to a new line below.
	typingLimit := e.wrapLimitWhenTyping
	if typingLimit <= 0 {
		typingLimit = e.softWrapLimit
	}
	if e.wrapWhenTyping && typingLimit > 0 && lenBefore <= typingLimit {
		line := e.Line(e.DataY())
		result := wordwrap.WrapLine(line, typingLimit, 0)
		if result.Wrapped {
			y := e.DataY()
			e.SetLine(y, result.Left)
			e.reFlowForward(result.Right, y+1, typingLimit)
			// cursorX is the position where the rune was inserted
			// (Insert does not advance the cursor). The post-insertion
			// cursor should be one past that.
			cursorX := e.pos.sx + e.pos.offsetX
			h := 80
			if c != nil {
				h = int(c.Height())
			}
			if cursorX >= result.BreakAt {
				// cursor is in the wrapped portion -- reposition it
				// onto the new line, one past the inserted rune
				newX := max(cursorX+1-result.RightStart, 0)
				e.pos.sy++
				e.pos.sx = newX
				e.pos.offsetX = 0
				if e.pos.sy >= (h - 1) {
					e.ScrollDown(c, nil, 1, h)
				}
				e.Center(c)
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return true
			}
			// cursor is before the break -- the line split happened
			// behind the cursor. Let the caller advance normally.
			e.Center(c)
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
			return false
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
