package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/vt100"
)

var splashToggleFilename = filepath.Join(userCacheDir, "o", "splash.txt")

// DisableSplashScreen saves a file to the cache directory so that the splash screen will be disabled the next time the editor starts
func DisableSplashScreen(status *StatusBar) bool {
	// Remove the file, but ignore errors if it was already gone
	_ = os.Remove(splashToggleFilename)

	// Write a new file
	contents := []byte{'0', '\n'} // 1 for enabled, 0 for disabled
	err := os.WriteFile(splashToggleFilename, contents, 0o644)
	if err != nil {
		return false
	}

	// TODO: Add a flag like "--welcome" to be able to re-enable the quick overview at start
	status.SetMessageAfterRedraw("Quick overview at start has been disabled.")
	return true
}

// EnableSplashScreen removes the splash screen config file
func EnableSplashScreen(status *StatusBar) bool {
	// Ignore any errors. If the file is already removed, that is fine too.
	_ = os.Remove(splashToggleFilename)
	if SplashScreenIsDisabled() {
		return false
	}
	status.SetMessageAfterRedraw("Quick overview at start has been enabled.")
	return true
}

// SplashScreenIsDisabled checks if the splash screen config file exists
func SplashScreenIsDisabled() bool {
	// Check if the splash config file exists and contains just "0"
	d, err := os.ReadFile(splashToggleFilename)
	if err != nil || len(d) == 0 {
		// No data means that the splash screen is enabled
		return false
	}
	// If there is data, it must be 0, otherwise the splash screen is enabled
	return strings.TrimSpace(string(d)) == "0"
}

// DrawSplash draws the splash screen + some help for new users
func (e *Editor) DrawSplash(c *vt100.Canvas, repositionCursorAfterDrawing bool) {
	const (
		maxLines = 8
		title    = "Welcome to " + versionString
	)

	var (
		minWidth        = 55
		foregroundColor = e.StatusForeground
		backgroundColor = e.DebugRunningBackground // e.Background
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
	bt.LowerEdge = bt.UpperEdge

	e.DrawBox(bt, c, centerBox)

	e.DrawTitle(bt, c, centerBox, title)

	e.DrawText(bt, c, listBox, lines)

	// Blit
	c.Draw()

	// Reposition the cursor
	if repositionCursorAfterDrawing {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		vt100.SetXY(uint(x), uint(y))
	}
}
