package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/files"
)

// DisableQuickHelpScreen saves a file to the cache directory so that the quick help will be disabled the next time the editor starts
func DisableQuickHelpScreen(status *StatusBar) bool {
	// Remove the file, but ignore errors if it was already gone
	_ = os.Remove(quickHelpToggleFilename)

	folderPath := filepath.Dir(quickHelpToggleFilename)

	// Try to (re)create the cache/o directory, but ignore errors
	_ = os.MkdirAll(folderPath, 0o755)

	// Write a new file
	contents := []byte{'0', '\n'} // 1 for enabled, 0 for disabled
	err := os.WriteFile(quickHelpToggleFilename, contents, 0o644)
	if err != nil {
		return false
	}

	if status != nil {
		status.SetMessageAfterRedraw("Quick overview at start has been disabled.")
	}

	return true
}

// EnableQuickHelpScreen removes the quick help config file
func EnableQuickHelpScreen(status *StatusBar) bool {
	// Ignore any errors. If the file is already removed, that is fine too.
	_ = os.Remove(quickHelpToggleFilename)

	folderPath := filepath.Dir(quickHelpToggleFilename)

	// Try to (re)create the cache/o directory, but ignore errors
	_ = os.MkdirAll(folderPath, 0o755)

	if QuickHelpScreenIsDisabled() {
		return false
	}
	status.SetMessageAfterRedraw("Quick overview at start has been enabled.")
	return true
}

// QuickHelpScreenIsDisabled checks if the quick help config file exists
func QuickHelpScreenIsDisabled() bool {
	return isAndroid || files.Exists(quickHelpToggleFilename)
}

// DrawQuickHelp draws the quick help + some help for new users
func (e *Editor) DrawQuickHelp(c *Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines = 8
		title    = "Quick Overview"
	)

	var (
		minWidth = 55

		foregroundColor = e.Foreground
		backgroundColor = e.Background
		edgeColor       = e.BoxUpperEdge
	)

	if QuickHelpScreenIsDisabled() {
		quickHelpText = strings.ReplaceAll(quickHelpText, "Disable this overview", "Enable this overview ")
	} else {
		quickHelpText = strings.ReplaceAll(quickHelpText, "Enable this overview ", "Disable this overview")
	}

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
	centerBox.X -= 10
	centerBox.W += 5

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
	const dryRun = true
	if addedLinesBecauseWordWrap := e.DrawText(bt, c, listBox, quickHelpText, dryRun); leftoverHeight > addedLinesBecauseWordWrap {
		centerBox.H += addedLinesBecauseWordWrap + 2
	}

	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, "=[ Quick Help ]=", false)
	e.DrawText(bt, c, listBox, quickHelpText, false)

	// Blit
	c.HideCursorAndDraw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}
