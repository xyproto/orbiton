package main

import (
	"strings"

	"github.com/xyproto/vt100"
)

// DisableSplashScreen saves a file to the cache directory so that the splash screen will be disabled the next time the editor starts
func DisableSplashScreen(c *vt100.Canvas, e *Editor, status *StatusBar) {
	status.SetMessageAfterRedraw("DISABLE SPLASH SCREEN: NOT YET IMPLEMENTED")
}

// DrawSplash draws the splash screen + some help for new users
func (e *Editor) DrawSplash(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines  = 8
		title     = "Welcome to " + versionString
		oHelpText = `Press ctrl-l and then ? to display the tutorial.        ___
                                                       // \\ ----
Other hotkeys:                                        ||  || ---
  ctrl-l and then ! to disable this help message      \\_// ---
  ctrl-o to display the main menu
  ctrl-s to save
  ctrl-q to quit

Try opening a new main.c file, press ctrl-w and then double ctrl-space.
`
	)

	var (
		minWidth        = 30
		backgroundColor = e.Background // e.DebugInstructionsBackground
	)

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(oHelpText, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
	}
	for _, line := range lines {
		if len(line) > minWidth {
			minWidth = len(line) + 3
		}
	}

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	centerBox := NewBox()

	centerBox.UpperRightPlacement(canvasBox, minWidth)
	centerBox.H++

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Background = &backgroundColor
	bt.UpperEdge = bt.LowerEdge

	e.DrawBox(bt, c, centerBox)

	e.DrawTitle(bt, c, centerBox, title)

	e.DrawList(bt, c, listBox, lines, -1)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
