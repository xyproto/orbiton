package main

import "github.com/xyproto/vt100"

func (e *Editor) Backspace(c *vt100.Canvas, bookmark *Position) {
	// Delete the character to the left
	if e.EmptyLine() {
		e.DeleteCurrentLineMoveBookmark(bookmark)
		e.pos.Up()
		e.TrimRight(e.DataY())
		e.End(c)
	} else if e.AtStartOfTheLine() { // at the start of the screen line, the line may be scrolled
		// remove the rest of the current line and move to the last letter of the line above
		// before deleting it
		if e.DataY() > 0 {
			e.pos.Up()
			e.TrimRight(e.DataY())
			e.End(c)
			e.Delete()
		}
	} else if (e.EmptyLine() || e.AtStartOfTheLine()) && e.indentation.Spaces && e.indentation.WSLen(e.LeadingWhitespace()) >= e.indentation.PerTab {
		// Delete several spaces
		for i := 0; i < e.indentation.PerTab; i++ {
			// Move back
			e.Prev(c)
			// Type a blank
			e.SetRune(' ')
			e.WriteRune(c)
			e.Delete()
		}
	} else {
		// Move back
		e.Prev(c)
		// Type a blank
		e.SetRune(' ')
		e.WriteRune(c)
		if !e.AtOrAfterEndOfLine() {
			// Delete the blank
			e.Delete()
			// scroll left instead of moving the cursor left, if possible
			e.pos.mut.Lock()
			if e.pos.offsetX > 0 {
				e.pos.offsetX--
				e.pos.sx++
			}
			e.pos.mut.Unlock()
		}
	}
}
