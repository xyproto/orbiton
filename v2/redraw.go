package main

import (
	"sync"

	"github.com/xyproto/vt"
)

var redrawMutex sync.Mutex // to avoid an issue where the terminal is resized, signals are flying and the user is hammering the esc button

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct
func (e *Editor) FullResetRedraw(c *vt.Canvas, status *StatusBar, drawLines, shouldHighlightCurrentLine bool) {
	if noDrawUntilResize.Load() {
		return
	}

	redrawMutex.Lock()
	defer redrawMutex.Unlock()

	savePos := e.pos

	if status != nil {
		status.ClearAll(c, false)
		e.SetSearchTerm(c, status, "", false)
	}

	vt.CloseKeepContent()
	vt.Reset()
	vt.Clear()
	vt.Init()

	newC := vt.NewCanvas()
	newC.ShowCursor()
	vt.EchoOff()

	w := int(newC.Width())

	if (w < e.wrapWidth) || (e.wrapWidth < 80 && w >= 80) {
		e.wrapWidth = w
	}

	if drawLines {
		e.HideCursorDrawLines(c, true, false, shouldHighlightCurrentLine)
	}

	// Assign the new canvas to the current canvas
	*c = newC.Copy() // Copy makes a copy without copying the mutex

	// TODO: Find out why the following lines are needed to properly handle the SIGWINCH resize signal

	resizeMut.Lock()

	newC = vt.NewCanvas()
	newC.ShowCursor()
	vt.EchoOff()
	w = int(newC.Width())

	resizeMut.Unlock()

	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}

	if drawLines {
		e.HideCursorDrawLines(c, true, false, shouldHighlightCurrentLine)
	}

	e.redraw.Store(false)
	e.redrawCursor.Store(false)

	e.pos = savePos
}

// RedrawIfNeeded will redraw the text on the canvas if e.redraw is set
func (e *Editor) RedrawIfNeeded(c *vt.Canvas, shouldHighlight bool) {
	if e.redraw.Load() {
		const respectOffset = true
		const redrawCanvas = false
		e.HideCursorDrawLines(c, respectOffset, redrawCanvas, shouldHighlight)
		e.redraw.Store(false)
	}
}

// RepositionCursor will send the VT100 commands needed to position the cursor
func (e *Editor) RepositionCursor(x, y uint) {
	vt.SetXY(x, y)

	e.previousX = int(x)
	e.previousY = int(y)
}

// PlaceAndEnableCursor will enable the cursor and then place it
func (e *Editor) PlaceAndEnableCursor(c *vt.Canvas) {
	// Redraw the cursor, if needed
	e.pos.mut.RLock()
	x := uint(e.pos.ScreenX())
	y := uint(e.pos.ScreenY())
	e.pos.mut.RUnlock()

	c.ShowCursor()
	vt.SetXY(x, y)

	e.previousX = int(x)
	e.previousY = int(y)
}

// RepositionCursorIfNeeded will reposition the cursor using VT100 commands, if needed
func (e *Editor) RepositionCursorIfNeeded(c *vt.Canvas) {
	// Redraw the cursor, if needed
	e.pos.mut.RLock()
	x := e.pos.ScreenX()
	y := e.pos.ScreenY()
	e.pos.mut.RUnlock()

	if x != e.previousX || y != e.previousY || e.redrawCursor.Load() {
		c.ShowCursor()
		e.RepositionCursor(uint(x), uint(y))
		e.redrawCursor.Store(false)
	}
}

// HideCursorDrawLines will draw a screen full of lines on the given canvas
func (e *Editor) HideCursorDrawLines(c *vt.Canvas, respectOffset, redrawCanvas, shouldHighlightCurrentLine bool) {
	if c == nil {
		return
	}

	const hideCursorWhenDrawing = true

	// TODO: Use a channel for queuing up calls to the package to avoid race conditions

	h := int(c.Height())
	if respectOffset {
		offsetY := e.pos.OffsetY()
		e.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0, shouldHighlightCurrentLine, hideCursorWhenDrawing)
	} else {
		e.WriteLines(c, LineIndex(0), LineIndex(h), 0, 0, shouldHighlightCurrentLine, hideCursorWhenDrawing)
	}
	if redrawCanvas {
		c.HideCursorAndRedraw()
	} else {
		c.HideCursorAndDraw()
	}
}

// InitialRedraw is called right before the main loop is started
func (e *Editor) InitialRedraw(c *vt.Canvas, status *StatusBar) {
	if c == nil {
		return
	}

	if noDrawUntilResize.Load() {
		return
	}

	// Check if an extra reset is needed
	shouldHighlightCurrentLine := e.highlightCurrentLine || e.highlightCurrentText

	// Draw the editor lines, respect the offset (true) and redraw (true)
	e.RedrawIfNeeded(c, shouldHighlightCurrentLine)

	// Display the status message
	if e.nanoMode.Load() {
		status.Show(c, e)
	} else if e.statusMode {
		status.ShowFilenameLineColWordCount(c, e)
	} else if e.blockMode {
		status.ShowBlockModeStatusLine(c, e)
	} else if status.IsError() {
		status.Show(c, e)
	}

	if msg := status.messageAfterRedraw; len(msg) > 0 {
		status.Clear(c, false)
		status.SetMessage(msg)
		status.messageAfterRedraw = ""
		status.Show(c, e)
	}

	e.WriteCurrentFunctionName(c) // not drawing immediately

	// Draw the function description if function description mode is enabled
	if ollama.Loaded() {
		descriptionPopupDrawn = false
		e.DrawBuildErrorExplanationContinuous(c, false)
		e.DrawFunctionDescriptionContinuous(c, false)
	}

	c.HideCursorAndDraw() // drawing now
}

// RedrawAtEndOfKeyLoop is called after each main loop
func (e *Editor) RedrawAtEndOfKeyLoop(c *vt.Canvas, status *StatusBar, shouldHighlightCurrentLine, repositionCursor bool) {
	redrawMutex.Lock()
	defer redrawMutex.Unlock()

	redrawCanvas := !e.debugMode

	redraw := e.redraw.Load()
	overlayRedraw := e.drawProgress.Load() || (e.drawFuncName.Load() && !e.nanoMode.Load())
	didDraw := false

	// Redraw, if needed
	if redraw {
		// Draw the editor lines on the canvas, respecting the offset
		e.HideCursorDrawLines(c, true, redrawCanvas, shouldHighlightCurrentLine)
	}

	if redraw || e.Changed() || overlayRedraw {
		// Draw the scroll progress indicator block on the right side
		if e.drawProgress.Load() {
			e.WriteProgress(c) // not drawing immediately
			e.drawProgress.Store(false)
		}

		// Draw the function name if drawFuncName is set and Nano mode is not enabled.
		// Also redraw while Ollama is thinking, so the upper-right indicator is not lost on redraw.
		if (e.drawFuncName.Load() || functionDescriptionThinking || hasBuildErrorExplanationThinking()) && !e.nanoMode.Load() {
			e.WriteCurrentFunctionName(c) // not drawing immediately
			e.drawFuncName.Store(false)
		}

		// Draw the function description if function description mode is enabled
		if ollama.Loaded() {
			descriptionPopupDrawn = false
			e.DrawBuildErrorExplanationContinuous(c, false)
			e.DrawFunctionDescriptionContinuous(c, false)
		}

		c.HideCursorAndDraw() // drawing now
		didDraw = true
		e.redraw.Store(false) // mark as redrawn
	}

	// Drawing status messages should come after redrawing, but before cursor positioning
	if e.nanoMode.Load() {
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
		status.Clear(c, false)
		status.SetMessage(msg)
		status.messageAfterRedraw = ""
		status.Show(c, e)
	}

	if repositionCursor {
		if didDraw {
			e.EnableAndPlaceCursor(c)
		} else {
			e.RepositionCursorIfNeeded(c)
		}
	} else {
		c.ShowCursor()
		e.RepositionCursorIfNeeded(c)
	}
}
