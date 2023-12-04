package main

import (
	"github.com/xyproto/vt100"
)

// DrawProgress draws a small progress indicator on the right hand side
func (e *Editor) DrawProgress(c *vt100.Canvas) {
	var (
		canvasWidth   = c.Width()
		canvasHeight  = float64(c.Height())
		lineNumberTop = float64(e.LineIndex())
		allLines      = float64(e.Len())
		x             = canvasWidth - 1
		y             = canvasHeight - 1
	)
	if allLines > 0 {
		y = (canvasHeight * lineNumberTop * (allLines + canvasHeight)) / (allLines * allLines)
	}
	if x >= canvasWidth {
		x = canvasWidth - 1
	}
	if y >= canvasHeight {
		y = canvasHeight - 1
	}
	c.WriteBackground(x, uint(y), e.MenuArrowColor.Background())
	c.Draw()
}
