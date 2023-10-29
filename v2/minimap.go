package main

import (
	"github.com/xyproto/minimap"
	"github.com/xyproto/vt100"
)

// DrawMiniMap draws a minimap of the current file contents to the right side of the canvas
func (e *Editor) DrawMiniMap(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	var cw = int(c.Width())
	var ch = int(c.Height())

	const margin = 2

	const width = 20
	var height = ch - (margin * 2)

	// The x and y position for where the minimap should be drawn
	var xpos = cw - (width + margin)
	const ypos = margin

	minimap.DrawMinimap(c, e.String(), xpos, ypos, width, height, e.mode, int(e.LineIndex()), e.CommentColor, e.Background, e.MenuArrowColor, e.Background)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}

}