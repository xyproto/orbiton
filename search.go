package main

import (
	"github.com/xyproto/vt100"
)

// SetSearchTerm will set the current search term to highlight
func (e *Editor) SetSearchTerm(s string, c *vt100.Canvas) {
	// set the search term
	e.searchTerm = s
	// redraw all characters
	h := int(c.Height())
	e.WriteLines(c, e.pos.Offset(), h+e.pos.Offset(), 0, 0)
	c.Draw()
}

// SearchTerm will return the current search term
func (e *Editor) SearchTerm() string {
	return e.searchTerm
}
