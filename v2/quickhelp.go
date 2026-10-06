package main

import (
	"os"
	"path/filepath"

	"github.com/xyproto/files"
	"github.com/xyproto/vt"
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

const (
	quickHelpMenuNoAction = iota
	quickHelpMenuDisableAction
	quickHelpMenuCommandMenuAction
	quickHelpMenuHotkeysAction
	quickHelpMenuTutorialAction
	quickHelpMenuSaveAndQuitAction
)

// QuickHelpAtStart checks if the quick help is displayed at start, which also makes ctrl-t display help
func (e *Editor) QuickHelpAtStart() bool {
	return (!QuickHelpScreenIsDisabled() || e.displayQuickHelp) && !e.noDisplayQuickHelp
}

// DrawQuickHelp draws a welcome message for new users
func (e *Editor) DrawQuickHelp(c *vt.Canvas, repositionCursorAfterDrawing bool) {
	const titleString = "Orbiton"
	var (
		foregroundColor = e.Foreground
		backgroundColor = e.Background
		edgeColor       = e.BoxUpperEdge
		canvasBox       = NewCanvasBox(c)
		art             = welcomeArt
		artWidth        = uint(0)
		bottomText = versionString + ". " + welcomeText
	)
	for _, line := range art {
		artWidth = max(artWidth, ulen([]rune(line)), ulen(titleString))
	}
	width := max(ulen([]rune(bottomText)), artWidth)
	boxW := int(width) + 6
	boxH := len(art) + 6
	if boxH > canvasBox.H-2 {
		art = nil
		boxH = 6
	}
	if boxW > canvasBox.W {
		boxW = canvasBox.W
	}

	centerBox := NewBox()
	centerBox.W = boxW
	centerBox.H = boxH
	centerBox.X = max(canvasBox.W-boxW-4, 0)
	centerBox.Y = max(canvasBox.H/10, 1)

	bt := e.NewBoxTheme()
	bt.Foreground = &foregroundColor
	bt.Background = &backgroundColor
	bt.UpperEdge = &edgeColor
	bt.LowerEdge = bt.UpperEdge

	e.DrawBox(bt, c, centerBox)
	e.DrawTitle(bt, c, centerBox, "=[ "+titleString+" ]=", false)

	x := uint(centerBox.X + 3)
	y := uint(centerBox.Y + 2)
	for _, line := range art {
		c.Write(x+(width-artWidth)/2, y, edgeColor, backgroundColor, line)
		y++
	}
	if len(art) > 0 {
		y++
	}
	y++

	versionColor := edgeColor // e.MenuArrowColor
	welcomeColor := edgeColor // foregroundcolor

	c.Write(x, y, versionColor, backgroundColor, versionString + ".")
	c.Write(x+ulen(versionString)+2, y, welcomeColor, backgroundColor, welcomeText)

	c.HideCursorAndDraw()

	if repositionCursorAfterDrawing {
		e.EnableAndPlaceCursor(c)
	}
}

// QuickHelpMenu displays a quick introduction, the most used keybindings and a menu.
// Returns one of the quickHelpMenu*Action constants.
func (e *Editor) QuickHelpMenu(status *StatusBar, tty *vt.TTY) int {
	choices := []string{
		"Launch the ctrl-o menu",
		"Overview of keybindings",
		"Tutorial for editing",
		"Disable quick help at start",
		"Save and quit",
	}
	actions := []int{
		quickHelpMenuCommandMenuAction,
		quickHelpMenuHotkeysAction,
		quickHelpMenuTutorialAction,
		quickHelpMenuDisableAction,
		quickHelpMenuSaveAndQuitAction,
	}
	const extraDashes = false
	selected, _ := e.Menu(status, tty, quickHelpIntroText, choices, e.Background, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, 0, extraDashes)
	if selected < 0 || selected >= len(actions) {
		return quickHelpMenuNoAction
	}
	return actions[selected]
}
