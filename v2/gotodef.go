package main

import (
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// canGoToDefinition checks if the "go to definition" feature has been implemented for this file mode yet
func (e *Editor) canGoToDefinition() bool {
	// TODO: Add all modes that makes sense
	return e.mode == mode.Go
}

// GoToDefinition tries to find the definition of the given string, saves the current location and jumps to the location of the definition.
// Returns true if it was possible to go to the definition.
func (e *Editor) GoToDefinition(c *vt100.Canvas, status *StatusBar) bool {
	// Can this language / editor mode support this?
	if !e.canGoToDefinition() {
		return false
	}
	// Do we have a word under the cursor? No need to trim it at this point.
	word := e.WordAtCursor()
	if word == "" {
		return false
	}
	// Is the word not a language keyword?
	for kw := range syntax.Keywords {
		if kw == word {
			// Don't go to the definition of keywords
			return false
		}
	}

	// word can be a string like "package.DoSomething" at this point.

	// TODO:
	// * Implement "go to definition"
	// * Go to definition should store the current location in a special kind of bookmark (including filename)
	//   so that another keypress can jump back to where we were.
	// * Implement a special kind of bookmark which also supports storing the filename.

	//bookmark = e.pos.Copy()
	//s := "Bookmarked line " + e.LineNumber().String()
	//status.SetMessage("  " + s + "  ")

	status.ClearAll(c)

	s := "func " + word

	// Go to defintion, but only of functions defined within the same Go file, for now
	e.SetSearchTerm(c, status, s)

	// Backward search form the current location
	startIndex := e.DataY()
	stopIndex := LineIndex(0)
	foundX, foundY := e.backwardSearch(startIndex, stopIndex)

	if foundY == -1 {
		status.SetMessage("Could not find " + s)
		status.Show(c, e)
		return false
	}

	// Go to the found match
	e.redraw, _ = e.GoTo(foundY, c, status)
	if foundX != -1 {
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
		e.HorizontalScrollIfNeeded(c)
	}

	// Center and prepare to redraw
	e.Center(c)
	e.redraw = true
	e.redrawCursor = e.redraw

	//status.SetMessage("Jumped to " + s)
	//status.Show(c, e)
	return true
}
