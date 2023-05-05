package main

import (
	"github.com/xyproto/vt100"
)

// HelpMessage tries to show a friendly help message to the user.
func (e *Editor) HelpMessage(c *vt100.Canvas, status *StatusBar) {
	// TODO: Implement a better help message, that covers more of the screen
	status.SetMessage("Press ctrl-q to quit or ctrl-o to show the menu. Use the --help flag for more help.")
	status.Show(c, e)
}
