package main

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/xyproto/vt"
)

var redrawMutex sync.Mutex // to avoid an issue where the terminal is resized, signals are flying and the user is hammering the esc button

// bookCatAnimStarted ensures only one cat-animation goroutine runs per
// editor instance even if InitialRedraw is called multiple times (e.g. on
// SIGWINCH).
var bookCatAnimStarted atomic.Bool

// startBookCatAnim launches a background goroutine that redraws the text
// book-mode top bar every 250 ms so the animated cat in drawBookTopBar
// walks back and forth even when the user is idle. The goroutine exits
// once the editor is quitting or book (text) mode has been left.
func (e *Editor) startBookCatAnim(c *vt.Canvas) {
	if c == nil {
		return
	}
	if !bookCatAnimStarted.CompareAndSwap(false, true) {
		return
	}
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for range t.C {
			if e.quit {
				return
			}
			if !e.bookTextMode() {
				// Stay dormant: if book mode is toggled back on
				// later, a fresh InitialRedraw will relaunch us
				// (the CAS above guards the restart).
				bookCatAnimStarted.Store(false)
				return
			}
			if e.bookCatPaused {
				// Cat is sitting still — no repaint needed. The
				// static glyph was drawn by the last keystroke
				// redraw (or the pause-toggle redraw).
				continue
			}
			if notRegularEditingRightNow.Load() {
				// A modal menu (ctrl-o), build dialog, input
				// prompt, etc. owns the screen. Repainting the
				// top bar would corrupt that UI; calling
				// bookTextModePlaceCursor would also unhide the
				// cursor that the menu just hid.
				continue
			}
			redrawMutex.Lock()
			w := int(c.Width())
			fg, _, bg := bookBarPalette(e.bookDarkMode)
			// Only repaint the left-side cat walk strip; the
			// right-side stats must stay stationary between key
			// presses. Reserve 0 cells on the right since we only
			// paint our own region.
			e.drawBookTopBarCat(c, w, fg, bg, 2, 0)
			c.Draw()
			e.bookTextModePlaceCursor(c)
			redrawMutex.Unlock()
		}
	}()
}

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
		if !e.bookMode.Load() {
			e.SetSearchTerm(c, status, "", false)
		}
	}

	vt.CloseKeepContent()
	vt.Reset()
	vt.Clear()
	vt.Init()

	newC := vt.NewCanvas()
	if !e.bookGraphicalMode() {
		newC.ShowCursor()
	}
	vt.EchoOff()

	w := int(newC.Width())

	if !e.bookMode.Load() {
		if (w < e.wrapWidth) || (e.wrapWidth < 80 && w >= 80) {
			e.wrapWidth = w
		}
	}

	// Book mode has its own rendering pipeline (graphical renders an image,
	// text renders styled VT100 output). Skip the regular WriteLines path
	// for both variants so that syntax-highlighted bold/colour attributes
	// don't briefly flash on screen.
	if drawLines && !e.bookMode.Load() {
		e.HideCursorDrawLines(c, true, false, shouldHighlightCurrentLine)
	}

	// Assign the new canvas to the current canvas
	*c = newC.Copy() // Copy makes a copy without copying the mutex

	// TODO: Find out why the following lines are needed to properly handle the SIGWINCH resize signal

	resizeMut.Lock()

	newC = vt.NewCanvas()
	if !e.bookGraphicalMode() {
		newC.ShowCursor()
	}
	vt.EchoOff()
	w = int(newC.Width())

	resizeMut.Unlock()

	if !e.bookMode.Load() {
		if w < e.wrapWidth {
			e.wrapWidth = w
		} else if e.wrapWidth < 80 && w >= 80 {
			e.wrapWidth = w
		}
	}

	if drawLines {
		if e.bookGraphicalMode() {
			// c now has the new terminal dimensions; render the image pipeline.
			e.bookModeRenderAll(c, nil)
		} else if e.bookTextMode() {
			// Text book mode: re-render styled Markdown into the canvas.
			e.bookTextModeRender(c)
			c.HideCursorAndDraw()
		} else {
			e.HideCursorDrawLines(c, true, false, shouldHighlightCurrentLine)
		}
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
	// Book mode has its own cursor semantics — the graphical renderer
	// paints the cursor inside the rendered image, and the text-mode
	// renderer maps document coordinates to canvas rows via
	// bookTextModePlaceCursor (accounting for the top bar row and
	// Markdown prefixes). Calling vt.SetXY with the raw ScreenX/ScreenY
	// here would snap the caret to (0,0) on startup — inside the top
	// bar — before the first keystroke moves it back. Defer to the
	// book-mode path instead.
	if e.bookGraphicalMode() {
		return
	}
	if e.bookTextMode() {
		e.bookTextModePlaceCursor(c)
		return
	}

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

	// Book mode with graphics: render the editing area as an image
	if e.bookGraphicalMode() {
		e.bookModeEnsureCursorVisible(c)
		redrawMutex.Lock()
		e.bookModeRenderAll(c, status)
		redrawMutex.Unlock()
		e.redraw.Store(false)
		return
	}

	// Book mode with text (VT100/xterm): render Markdown via ANSI escape codes
	if e.bookTextMode() {
		e.bookModeEnsureCursorVisible(c)
		e.bookTextModeRender(c)
		if !e.statusMode {
			status.NanoInfo(c, e)
		}
		c.HideCursorAndDraw()
		e.redraw.Store(false)
		// Without this, the terminal cursor stays at row 0 (the top
		// filename bar) until the first key is pressed.
		e.bookTextModePlaceCursor(c)
		// Kick off the walking-cat top-bar animation (idempotent).
		e.startBookCatAnim(c)
		return
	}

	// Check if an extra reset is needed
	shouldHighlightCurrentLine := e.highlightCurrentLine || e.highlightCurrentText

	// Draw the editor lines, respect the offset (true) and redraw (true)
	e.RedrawIfNeeded(c, shouldHighlightCurrentLine)

	// Display the status message
	if e.nanoMode.Load() {
		status.Show(c, e)
	} else if e.bookMode.Load() {
		status.NanoInfo(c, e)
	} else if e.statusMode {
		status.ShowFilenameLineColWordCount(c, e)
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

// updateCanvasLines writes the current editor lines to the canvas buffer, without drawing
func (e *Editor) updateCanvasLines(c *vt.Canvas, shouldHighlightCurrentLine bool) {
	if c == nil {
		return
	}
	h := int(c.Height())
	offsetY := e.pos.OffsetY()
	const hideCursorWhenDrawing = false
	e.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0, shouldHighlightCurrentLine, hideCursorWhenDrawing)
}

// RedrawAtEndOfKeyLoop is called after each main loop
func (e *Editor) RedrawAtEndOfKeyLoop(c *vt.Canvas, status *StatusBar, shouldHighlightCurrentLine, repositionCursor bool) {
	redrawMutex.Lock()
	defer redrawMutex.Unlock()

	// Book mode with graphics: bypass canvas, render image + status bar directly.
	// The cursor is drawn inside the image, so we re-render on every key loop
	// iteration to keep it current regardless of whether content changed.
	if e.bookGraphicalMode() {
		e.bookModeEnsureCursorVisible(c)
		e.bookModeRenderAll(c, status)
		e.redraw.Store(false)
		return
	}

	// Book mode with text (VT100/xterm): re-render Markdown on every key loop
	// iteration so movement and edits are reflected immediately.
	if e.bookTextMode() {
		e.bookModeEnsureCursorVisible(c)
		e.bookTextModeRender(c)
		if !e.statusMode {
			status.NanoInfo(c, e)
		}
		c.HideCursorAndDraw()
		e.redraw.Store(false)
		e.bookTextModePlaceCursor(c)
		return
	}

	redraw := e.redraw.Load()
	overlayRedraw := e.drawProgress.Load() || (e.drawFuncName.Load() && !e.nanoMode.Load())
	didDraw := false

	// Update the canvas buffer with fresh line content if needed.
	// Defer the actual terminal write to the single HideCursorAndDraw below,
	// so the lines and overlays are flushed together in one frame.
	if redraw {
		e.updateCanvasLines(c, shouldHighlightCurrentLine)
		if e.debugMode {
			c.HideCursorAndDraw() // debug panes are drawn separately, so draw now
		}
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
	} else if e.bookMode.Load() {
		status.NanoInfo(c, e)
	} else if e.statusMode {
		status.ShowFilenameLineColWordCount(c, e)
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

	if e.blockMode {
		// In block mode, hide the hardware cursor and rely on virtual cursors
		c.HideCursor()
	} else if repositionCursor {
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
