package main

import (
	"github.com/xyproto/vt100"
)

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct
func (e *Editor) FullResetRedraw(c *vt100.Canvas, status *StatusBar, drawLines, resized bool) {

	savePos := e.pos

	currentLineNumber := e.LineNumber()
	if e.sshMode {
		// TODO: Figure out why this helps doing a full redraw when o is used over ssh
		e.GoToLineNumber(e.LastLineNumber(), c, nil, true)
	}

	if status != nil {
		status.ClearAll(c)
		e.SetSearchTerm(c, status, "")
	}

	vt100.Close()
	vt100.Reset()
	vt100.Clear()
	vt100.Init()

	newC := vt100.NewCanvas()
	newC.ShowCursor()
	w := int(newC.Width())
	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}
	if drawLines {
		if e.sshMode {
			e.DrawLines(c, true, true)
		} else {
			e.DrawLines(c, true, false)
		}
	}
	// Assign the new canvas to the current canvas
	*c = *newC

	// TODO: Find out why the following lines are needed to properly handle the SIGWINCH resize signal

	newC = vt100.NewCanvas()
	newC.ShowCursor()
	w = int(newC.Width())
	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}

	if drawLines {
		e.DrawLines(c, true, true)
	}

	if e.sshMode {
		// TODO: Figure out why this helps doing a full redraw when o is used over ssh
		// Go to the line we were at
		e.redraw = e.ScrollDown(c, nil, e.pos.scrollSpeed)
		e.redraw = e.ScrollUp(c, nil, e.pos.scrollSpeed)
		e.GoToLineNumber(currentLineNumber, c, nil, resized)
		e.redraw = true
		e.redrawCursor = true
	} else {
		e.redraw = false
		e.redrawCursor = false
	}

	e.pos = savePos
}

// RedrawIfNeeded will redraw the text on the canvas if e.redraw is set
func (e *Editor) RedrawIfNeeded(c *vt100.Canvas) {
	if e.redraw {
		respectOffset := true
		redrawCanvas := true
		e.DrawLines(c, respectOffset, redrawCanvas)
		e.redraw = false
	}
}

// RepositionCursor will send the VT100 commands needed to position the cursor
func (e *Editor) RepositionCursor(x, y int) {
	// Redraw the cursor
	vt100.SetXY(uint(x), uint(y))
	e.previousX = x
	e.previousY = y
}

// RepositionCursorIfNeeded will reposition the cursor using VT100 commands, if needed
func (e *Editor) RepositionCursorIfNeeded() {
	// Redraw the cursor, if needed
	x := e.pos.ScreenX()
	y := e.pos.ScreenY()
	if e.redrawCursor || x != e.previousX || y != e.previousY {
		e.RepositionCursor(x, y)
		e.redrawCursor = false
	}
}

// DrawLines will draw a screen full of lines on the given canvas
func (e *Editor) DrawLines(c *vt100.Canvas, respectOffset, redrawCanvas bool) error {
	var err error
	h := int(c.Height())
	if respectOffset {
		offsetY := e.pos.OffsetY()
		err = e.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)
	} else {
		err = e.WriteLines(c, LineIndex(0), LineIndex(h), 0, 0)
	}
	if redrawCanvas {
		c.Redraw()
	} else {
		c.Draw()
	}
	return err
}

// InitialRedraw is called right before the main loop is started
func (e *Editor) InitialRedraw(c *vt100.Canvas, status *StatusBar) {

	// Check if an extra reset is needed
	if e.sshMode {
		drawLines := true
		resized := true
		e.FullResetRedraw(c, status, drawLines, resized)
	} else {
		// Draw the editor lines, respect the offset (true) and redraw (true)
		e.RedrawIfNeeded(c)
	}

	// Display the status message
	if e.statusMode {
		status.ShowLineColWordCount(c, e, e.filename)
	} else if status.IsError() {
		status.Show(c, e)
	}

	e.RepositionCursorIfNeeded()
}

// RedrawAtEndOfKeyLoop is called after each main loop
func (e *Editor) RedrawAtEndOfKeyLoop(c *vt100.Canvas, status *StatusBar, statusMessage string) {

	// Redraw, if needed
	if e.redraw {
		// Draw the editor lines on the canvas, respecting the offset
		e.DrawLines(c, true, true)
		e.redraw = false
	} else if e.Changed() {
		c.Draw()
	}

	// Drawing status messages should come after redrawing, but before cursor positioning
	if e.statusMode {
		status.ShowLineColWordCount(c, e, e.filename)
	} else if status.IsError() {
		// Show the status message
		status.Show(c, e)
	}

	// Position the cursor
	e.RepositionCursorIfNeeded()
}
