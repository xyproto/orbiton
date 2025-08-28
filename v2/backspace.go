package main

// Backspace tries to delete characters to the left and move the cursor accordingly. Also supports block mode.
func (e *Editor) Backspace(c *Canvas, bookmark *Position) {
	// doBackspace is defined as a function here in order to enclose the c and bookmark arguments
	doBackspace := func() bool {
		// Delete the character to the left
		if e.EmptyLine() {
			if e.blockMode {
				return false // break
			}
			e.DeleteCurrentLineMoveBookmark(bookmark)
			e.pos.Up()
			e.TrimRight(e.DataY())
			e.End(c)
		} else if e.AtStartOfTheLine() { // at the start of the screen line, the line may be scrolled
			if e.blockMode {
				return false // break
			}
			// remove the rest of the current line and move to the last letter of the line above before deleting it
			if e.DataY() > 0 {
				e.pos.Up()
				e.TrimRight(e.DataY())
				e.End(c)
				e.Delete(c, false)
			}
		} else {
			// move back
			e.Prev(c)
			// type a blank
			e.SetRune(' ')
			e.WriteRune(c)
			if !e.AtOrAfterEndOfLine() {
				// delete the blank
				e.Delete(c, false)
				// scroll left instead of moving the cursor left, if possible
				e.pos.mut.Lock()
				if e.pos.offsetX > 0 {
					e.pos.offsetX--
					e.pos.sx++
				}
				e.pos.mut.Unlock()
			}
		}
		return true // success
	}
	if e.blockMode {
		e.ForEachLineInBlock(c, doBackspace)
	} else {
		doBackspace()
	}
}
