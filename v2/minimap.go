package main

import (
	"github.com/xyproto/minimap"
	"github.com/xyproto/vt100"
)

var miniMapCache string

// DrawMiniMap draws a minimap of the current file contents to the right side of the canvas
func (e *Editor) DrawMiniMap(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	var cw = int(c.Width())
	var ch = int(c.Height())

	const topMargin = 1
	const botMargin = 1
	const rightMargin = 2

	var width = (cw - rightMargin) / 20
	var height = ch - (topMargin + botMargin)

	// The x and y position for where the minimap should be drawn
	var xpos = cw - (width + rightMargin)
	const ypos = topMargin

	//lineIndex := int(e.LineIndex())
	lineIndex := int(LineIndex(e.pos.OffsetY()))

	if miniMapCache == "" {
		miniMapCache = e.String()
	}

	text := e.BoxBackground
	space := e.Background
	highlight := e.NanoHelpBackground

	// TODO: Also pass the canvas height to this function so that the indicator line is drawn all the way down
	minimap.DrawBackgroundMinimap(c, miniMapCache, xpos, ypos, width, height, lineIndex, text, space, highlight)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}

}
