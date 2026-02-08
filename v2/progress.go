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
		bottomLine    = canvasHeight - 1
		y             = bottomLine
	)
	if allLines > 0 {
		y = (canvasHeight * lineNumberTop * (allLines + canvasHeight)) / (allLines * allLines)
		if y >= canvasHeight {
			y = bottomLine
		}
	}
	c.WriteBackground(x, uint(y), e.ProgressIndicatorBackground)
}
