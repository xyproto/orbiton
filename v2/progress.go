package main

import (
	"github.com/xyproto/vt"
)

// WriteProgress draws a small progress indicator on the right hand side, but does not draw/redraw the canvas
func (e *Editor) WriteProgress(c *vt.Canvas) {
	if useASCII {
		return
	}
	var (
		canvasWidth   = c.Width()
		canvasHeight  = float64(c.Height())
		lineNumberTop = float64(e.LineIndex())
		allLines      = float64(e.Len())
		x             = canvasWidth - 1
		maxY          = canvasHeight - 2 // leave room for the status bar
		y             = maxY
	)
	if allLines > 0 && maxY > 0 {
		y = (maxY * lineNumberTop * (allLines + maxY)) / (allLines * allLines)
		if y > maxY {
			y = maxY
		}
	}
	c.WriteBackground(x, uint(y), e.ProgressIndicatorBackground)
}
