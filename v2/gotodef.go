package main

import (
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

// CanGoToDefinition checks if the "go to definition" feature has been implemented for this file mode yet
func (e *Editor) CanGoToDefinition() bool {
	// TODO: Add all modes that makes sense
	return e.mode == mode.Go
}

// GoToDefinition tries to find the definition of the given string, saves the current location and jumps to the location of the definition
func (e *Editor) GoToDefinition(c *vt100.Canvas, status *StatusBar, word string) {
	// TODO: Go to definition should store the current location in a special kind of bookmark (including filename)
	//       so that another keypress can jump back to where we were.

	//bookmark = e.pos.Copy()
	//s := "Bookmarked line " + e.LineNumber().String()
	//status.SetMessage("  " + s + "  ")

	// TODO: Implement "go to definition"
	status.SetMessage("TO IMPLEMENT: GO TO DEFINITION OF " + word)
	status.Show(c, e)
}
