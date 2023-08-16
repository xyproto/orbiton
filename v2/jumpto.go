package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

// PositionIndex has a ColIndex and LineIndex
type PositionIndex struct {
	X ColIndex
	Y LineIndex
}

var jumpLetters map[rune]PositionIndex

// RegisterJumpLetter will register a jump-letter together with a location that is visible on screen
func (e *Editor) RegisterJumpLetter(r rune, x ColIndex, y LineIndex) bool {
	const skipThese = "0123456789%.,btecm" // used by the ctrl-l functionality for other things
	if strings.ContainsRune(skipThese, r) || unicode.IsSymbol(r) {
		return false
	}
	if jumpLetters == nil {
		jumpLetters = make(map[rune]PositionIndex)
	}
	jumpLetters[r] = PositionIndex{x, y}
	return true
}

// HasJumpLetter checks if this jump letter has been registered yet
func (e *Editor) HasJumpLetter(r rune) bool {
	if jumpLetters == nil {
		return false
	}
	_, found := jumpLetters[r]
	return found
}

// GetJumpX returns the X position for the given jump letter, or -1 if not found
func (e *Editor) GetJumpX(r rune) ColIndex {
	if jumpLetters == nil {
		return -1
	}
	xy, found := jumpLetters[r]
	if !found {
		return -1
	}
	return xy.X
}

// GetJumpY returns the Y position for the given jump letter, or -1 if not found
func (e *Editor) GetJumpY(r rune) LineIndex {
	if jumpLetters == nil {
		return -1
	}
	xy, found := jumpLetters[r]
	if !found {
		return -1
	}
	return xy.Y
}

// ClearJumpLetters clears all jump letters (typically after the ctrl-l screen is done)
func (e *Editor) ClearJumpLetters() {
	jumpLetters = nil
}

// GoTo will go to a given line index, counting from 0
// status is used for clearing status bar messages and can be nil
// Returns true if the editor should be redrawn
// The second returned bool is if the end has been reached
func (e *Editor) GoTo(dataY LineIndex, c *vt100.Canvas, status *StatusBar) (bool, bool) {
	if dataY == e.DataY() {
		// Already at the correct line, but still trigger a redraw
		return true, false
	}
	reachedTheEnd := false
	// Out of bounds checking for y
	if dataY < 0 {
		dataY = 0
	} else if dataY >= LineIndex(e.Len()) {
		dataY = LineIndex(e.Len() - 1)
		reachedTheEnd = true
	}

	h := 25
	if c != nil {
		// Get the current terminal height
		h = int(c.Height())
	}

	// Is the place we want to go within the current scroll window?
	topY := LineIndex(e.pos.offsetY)
	botY := LineIndex(e.pos.offsetY + h)

	if dataY >= topY && dataY < botY {
		// No scrolling is needed, just move the screen y position
		e.pos.sy = int(dataY) - e.pos.offsetY
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if int(dataY) < h {
		// No scrolling is needed, just move the screen y position
		e.pos.offsetY = 0
		e.pos.sy = int(dataY)
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if reachedTheEnd {
		// To the end of the text
		e.pos.offsetY = e.Len() - h
		e.pos.sy = h - 1
	} else {
		prevY := e.pos.sy
		// Scrolling is needed
		e.pos.sy = 0
		e.pos.offsetY = int(dataY)
		lessJumpY := prevY
		lessJumpOffset := int(dataY) - prevY
		if (lessJumpY + lessJumpOffset) < e.Len() {
			e.pos.sy = lessJumpY
			e.pos.offsetY = lessJumpOffset
		}
	}

	// The Y scrolling is done, move the X position according to the contents of the line
	e.pos.SetX(c, int(e.FirstScreenPosition(e.DataY())))

	// Clear all status messages
	if status != nil {
		status.ClearAll(c)
	}

	// Trigger cursor redraw
	e.redrawCursor = true

	// Should also redraw the text, and has the end been reached?
	return true, reachedTheEnd
}

// GoToLineNumber will go to a given line number, but counting from 1, not from 0!
func (e *Editor) GoToLineNumber(lineNumber LineNumber, c *vt100.Canvas, status *StatusBar, center bool) bool {
	if lineNumber < 1 {
		lineNumber = 1
	}
	redraw, _ := e.GoTo(lineNumber.LineIndex(), c, status)
	if redraw && center {
		e.Center(c)
	}
	return redraw
}

// GoToLineNumberAndCol will go to a given line number (counting from 1) and column number (counting from 1)
func (e *Editor) GoToLineNumberAndCol(lineNumber LineNumber, colNumber ColNumber, c *vt100.Canvas, status *StatusBar, center, handleTabExpansion bool) bool {
	if colNumber < 1 {
		colNumber = 1
	}
	if lineNumber < 1 {
		lineNumber = 1
	}
	xIndex := colNumber.ColIndex()
	yIndex := lineNumber.LineIndex()
	// Go to the correct line
	redraw, _ := e.GoTo(yIndex, c, status)
	// Go to the correct column as well
	if handleTabExpansion {
		tabs := strings.Count(e.Line(yIndex), "\t")
		newScreenX := int(xIndex) + (tabs * (e.indentation.PerTab - 1))
		if e.pos.sx != newScreenX {
			redraw = true
		}
		e.pos.sx = newScreenX
	} else {
		if e.pos.sx != int(xIndex) {
			redraw = true
		}
		e.pos.sx = int(xIndex)
	}
	if redraw && center {
		e.Center(c)
	}
	return redraw

}

// GoToLineIndexAndColIndex will go to a given line index (counting from 0) and column index (counting from 0)
func (e *Editor) GoToLineIndexAndColIndex(yIndex LineIndex, xIndex ColIndex, c *vt100.Canvas, status *StatusBar, center, handleTabExpansion bool) bool {
	if xIndex < 0 {
		xIndex = 0
	}
	if yIndex < 0 {
		yIndex = 0
	}
	// Go to the correct line
	redraw, _ := e.GoTo(yIndex, c, status)
	// Go to the correct column as well
	if handleTabExpansion {
		tabs := strings.Count(e.Line(yIndex), "\t")
		newScreenX := int(xIndex) + (tabs * (e.indentation.PerTab - 1))
		if e.pos.sx != newScreenX {
			redraw = true
		}
		e.pos.sx = newScreenX
	} else {
		if e.pos.sx != int(xIndex) {
			redraw = true
		}
		e.pos.sx = int(xIndex)
	}
	if redraw && center {
		e.Center(c)
	}
	return redraw
}
