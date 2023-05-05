package main

import (
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

	// We might be able to go to the definition of word, maybe.
	// word can be a string like "package.DoSomething" at this point.

	// TODO: Go to definition should store the current location in a special kind of bookmark (including filename)
	//       so that another keypress can jump back to where we were.

	//bookmark = e.pos.Copy()
	//s := "Bookmarked line " + e.LineNumber().String()
	//status.SetMessage("  " + s + "  ")

	// TODO: Implement "go to definition"
	status.ClearAll(c)
	status.SetMessage("TO IMPLEMENT: GO TO DEFINITION OF " + word)
	status.Show(c, e)

	return true
}
