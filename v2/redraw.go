package main

import (
	"sync"

	"github.com/xyproto/vt100"
)

var redrawMutex sync.Mutex // to avoid an issue where the terminal is resized, signals are flying and the user is hammering the esc button

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct
func (e *Editor) FullResetRedraw(c *vt100.Canvas, status *StatusBar, drawLines, shouldHighlight bool) {
	redrawMutex.Lock()
	defer redrawMutex.Unlock()

	savePos := e.pos

	if status != nil {
		status.ClearAll(c)
		e.SetSearchTerm(c, status, "", false)
	}

	vt100.Close()
	vt100.Reset()
	vt100.Clear()
	vt100.Init()

	newC := vt100.NewCanvas()
	newC.ShowCursor()
	vt100.EchoOff()

	w := int(newC.Width())

	if (w < e.wrapWidth) || (e.wrapWidth < 80 && w >= 80) {
		e.wrapWidth = w
	}

	if drawLines {
		e.DrawLines(c, true, e.sshMode, shouldHighlight)
	}

	// Assign the new canvas to the current canvas
	// All mutexes are unlocked at this point for the copying not to be worrysome.
	*c = *newC

	// TODO: Find out why the following lines are needed to properly handle the SIGWINCH resize signal

	resizeMut.Lock()

	newC = vt100.NewCanvas()
	newC.ShowCursor()
	vt100.EchoOff()
	w = int(newC.Width())

	resizeMut.Unlock()

	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}

	if drawLines {
		e.DrawLines(c, true, e.sshMode, shouldHighlight)
	}

	if e.sshMode {
		// TODO: Figure out why this helps doing a full redraw when o is used over ssh
		// Go to the line we were at
		e.ScrollUp(c, nil, e.pos.scrollSpeed)
		e.DrawLines(c, true, true, shouldHighlight)
		e.ScrollDown(c, nil, e.pos.scrollSpeed)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	} else {
		e.redraw.Store(false)
		e.redrawCursor.Store(false)
	}

	e.pos = savePos
}

// RedrawIfNeeded will redraw the text on the canvas if e.redraw is set
func (e *Editor) RedrawIfNeeded(c *vt100.Canvas, shouldHighlight bool) {
	if e.redraw.Load() {
		respectOffset := true
		redrawCanvas := e.sshMode
		e.DrawLines(c, respectOffset, redrawCanvas, shouldHighlight)
		e.redraw.Store(false)
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
	e.pos.mut.RLock()
	x := e.pos.ScreenX()
	y := e.pos.ScreenY()
	e.pos.mut.RUnlock()

	if x != e.previousX || y != e.previousY || e.redrawCursor.Load() {
		e.RepositionCursor(x, y)
		e.redrawCursor.Store(false)
	}
}

// DrawLines will draw a screen full of lines on the given canvas
func (e *Editor) DrawLines(c *vt100.Canvas, respectOffset, redrawCanvas, shouldHighlight bool) {
	h := int(c.Height())
	if respectOffset {
		offsetY := e.pos.OffsetY()
		e.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0, shouldHighlight)
	} else {
		e.WriteLines(c, LineIndex(0), LineIndex(h), 0, 0, shouldHighlight)
	}
	if redrawCanvas {
		c.HideCursorAndRedraw()
	} else {
		c.HideCursorAndDraw()
	}
	e.EnableAndPlaceCursor(c)
}

// InitialRedraw is called right before the main loop is started
func (e *Editor) InitialRedraw(c *vt100.Canvas, status *StatusBar) {
	// Check if an extra reset is needed
	if e.sshMode {
		drawLines := true
		e.FullResetRedraw(c, status, drawLines, false)
	} else {
		// Draw the editor lines, respect the offset (true) and redraw (true)
		e.RedrawIfNeeded(c, false)
	}

	// Display the status message
	if e.nanoMode {
		status.Show(c, e)
	} else if e.statusMode {
		status.ShowFilenameLineColWordCount(c, e)
	} else if e.blockMode {
		status.ShowBlockModeStatusLine(c, e)
	} else if status.IsError() {
		status.Show(c, e)
	}

	if msg := status.messageAfterRedraw; len(msg) > 0 {
		status.Clear(c)
		status.SetMessage(msg)
		status.messageAfterRedraw = ""
		status.Show(c, e)
	}

	e.RepositionCursorIfNeeded()
}

// RedrawAtEndOfKeyLoop is called after each main loop
func (e *Editor) RedrawAtEndOfKeyLoop(c *vt100.Canvas, status *StatusBar, shouldHighlight bool) {
	redrawCanvas := !e.debugMode

	// Redraw, if needed
	if e.redraw.Load() {
		// Draw the editor lines on the canvas, respecting the offset
		e.DrawLines(c, true, redrawCanvas, shouldHighlight)
		e.redraw.Store(false)

		if e.drawProgress.Load() {
			e.DrawProgress(c)
			e.drawProgress.Store(false)
		}
	} else if e.Changed() {
		c.Draw()

		if e.drawProgress.Load() {
			e.DrawProgress(c)
			e.drawProgress.Store(false)
		}
	}

	// Drawing status messages should come after redrawing, but before cursor positioning
	if e.nanoMode {
		status.Show(c, e)
	} else if e.statusMode {
		status.ShowFilenameLineColWordCount(c, e)
	} else if e.blockMode {
		status.ShowBlockModeStatusLine(c, e)
	} else if status.IsError() {
		// Show the status message, if *statusMessage is not set
		if status.messageAfterRedraw == "" {
			status.Show(c, e)
		}
	}

	if msg := status.messageAfterRedraw; len(msg) > 0 {
		status.Clear(c)
		status.SetMessage(msg)
		status.messageAfterRedraw = ""
		status.Show(c, e)
	}

	e.RepositionCursorIfNeeded()
}
