package main

import (
	"strings"

	"github.com/xyproto/vt100"
)

// SetSearchTerm will set the current search term to highlight
func (e *Editor) SetSearchTerm(s string, c *vt100.Canvas, status *StatusBar) {
	// set the search term
	e.searchTerm = s
	// Go to the first instance after the current line, if found
	e.lineBeforeSearch = e.DataY()
	for y := e.DataY(); y < e.Len(); y++ {
		if strings.Contains(e.Line(y), s) {
			// Found an instance, scroll there
			// GoTo returns true if the screen should be redrawn
			e.GoTo(y, c, status)
			break
		}
	}
	// draw the lines to the canvas
	e.DrawLines(c, true, false)
}

// SearchTerm will return the current search term
func (e *Editor) SearchTerm() string {
	return e.searchTerm
}
