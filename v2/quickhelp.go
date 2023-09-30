package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/vt100"
)

var quickHelpToggleFilename = filepath.Join(userCacheDir, "o", "quickhelp.txt")

// DisableQuickHelpScreen saves a file to the cache directory so that the quick help will be disabled the next time the editor starts
func DisableQuickHelpScreen(status *StatusBar) bool {
	// Remove the file, but ignore errors if it was already gone
	_ = os.Remove(quickHelpToggleFilename)

	// Write a new file
	contents := []byte{'0', '\n'} // 1 for enabled, 0 for disabled
	err := os.WriteFile(quickHelpToggleFilename, contents, 0o644)
	if err != nil {
		return false
	}

	// TODO: Add a flag like "--welcome" to be able to re-enable the quick overview at start
	status.SetMessageAfterRedraw("Quick overview at start has been disabled.")
	return true
}

// EnableQuickHelpScreen removes the quick help config file
func EnableQuickHelpScreen(status *StatusBar) bool {
	// Ignore any errors. If the file is already removed, that is fine too.
	_ = os.Remove(quickHelpToggleFilename)
	if QuickHelpScreenIsDisabled() {
		return false
	}
	status.SetMessageAfterRedraw("Quick overview at start has been enabled.")
	return true
}

// QuickHelpScreenIsDisabled checks if the quick help config file exists
func QuickHelpScreenIsDisabled() bool {
	// Check if the quick help config file exists and contains just "0"
	d, err := os.ReadFile(quickHelpToggleFilename)
	if err != nil || len(d) == 0 {
		// No data means that the quick help is enabled
		return false
	}
	// If there is data, it must be 0, otherwise the quick help is enabled
	return strings.TrimSpace(string(d)) == "0"
}

// DrawQuickHelp draws the quick help + some help for new users
func (e *Editor) DrawQuickHelp(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines = 8
		title    = "Quick Overview"
	)

	var (
		minWidth = 55

		foregroundColor = e.Foreground
		backgroundColor = e.Background
		edgeColor       = e.StatusForeground
	)

	// Get the last maxLine lines, and create a string slice
	lines := strings.Split(quickHelpText, "\n")
	if l := len(lines); l > maxLines {
		lines = lines[l-maxLines:]
	}
	for _, line := range lines {
		if len(line) > minWidth {
			minWidth = len(line) + 5
		}
	}

	// First create a box the size of the entire canvas
	canvasBox := NewCanvasBox(c)

	centerBox := NewBox()

	centerBox.UpperRightPlacement(canvasBox, minWidth)
	centerBox.H += 2

	// Then create a list box
	listBox := NewBox()
	listBox.FillWithMargins(centerBox, 2, 2)

	// Get the current theme for the stdout box
	bt := e.NewBoxTheme()
	bt.Foreground = &foregroundColor
	bt.Background = &backgroundColor
	bt.UpperEdge = &edgeColor
	bt.LowerEdge = bt.UpperEdge

	leftoverHeight := (canvasBox.Y + canvasBox.H) - (centerBox.Y + centerBox.H)

	// This is just an attempt at drawing the text, in order to find addedLinesBecauseWordWrap
	if addedLinesBecauseWordWrap := e.DrawText(bt, c, listBox, quickHelpText); leftoverHeight > addedLinesBecauseWordWrap {
		centerBox.H += addedLinesBecauseWordWrap + 2
	}

	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, title)
	e.DrawFooter(bt, c, centerBox, versionString)
	e.DrawText(bt, c, listBox, quickHelpText)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
