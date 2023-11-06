package main

import (
	"github.com/xyproto/vt100"
)

// Ratio returns where the cursor is in the document, linewise, from 0 to 1
func (e *Editor) Ratio() float64 {
	lineNumber := e.LineNumber()
	allLines := e.Len()
	if allLines <= 0 {
		return 1.0
	}
	return float64(lineNumber) / float64(allLines)
}

// DrawProgress draws a small progress indicator on the right hand side
func (e *Editor) DrawProgress(c *vt100.Canvas) {
	r := e.Ratio()
	w := int(c.Width())
	h := int(c.Height())
	x := w - 1
	y := int(float64(h) * float64(r))
	c.WriteBackground(uint(x), uint(y), e.MenuArrowColor.Background())
	c.Draw()
}
