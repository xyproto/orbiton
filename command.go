package main

import (
	"fmt"
	"path/filepath"

	"github.com/xyproto/vt100"
)

// UserCommand performs an editor command, given an action string, like "save"
func (e *Editor) UserCommand(c *vt100.Canvas, status *StatusBar, action string) {
	switch action {
	case "save":
		status.ClearAll(c)
		// Save the file
		if err := e.Save(!e.DrawMode()); err != nil {
			status.SetMessage(err.Error())
			status.Show(c, e)
		} else {
			// TODO: Go to the end of the document at this point, if needed
			// Lines may be trimmed for whitespace, so move to the end, if needed
			if !e.DrawMode() && e.AfterLineScreenContents() {
				e.End()
			}
			// Save the current location in the location history and write it to file
			absFilename, err := filepath.Abs(e.filename)
			if err == nil { // no error
				e.SaveLocation(absFilename, e.locationHistory)
			}
			// Save the current search history
			SaveSearchHistory(expandUser(searchHistoryFilename), searchHistory)
			// Status message
			status.SetMessage("Saved " + e.filename)
			status.Show(c, e)
			c.Draw()
		}
	case "quit":
		e.quit = true        // indicate that the user wishes to quit
		e.clearOnQuit = true // clear the terminal after quitting
	}
}

// CommandMenu will display a menu with various commands that can be browsed with arrow up and arrow down
// Returns true if the editor should immediately be redrawn.
// Also returns the selected menu index (can be -1).
func (e *Editor) CommandMenu(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, lastMenuIndex int) (bool, int) {

	drawModeStatus := "on"
	if !e.drawMode {
		drawModeStatus = "off"
	}

	var (
		actionTitles = map[int]string{
			0: "Toggle draw mode (currently " + drawModeStatus + ")",
			1: "Save " + e.filename,
			2: "Quit o",
		}
		actionFunctions = map[int]func(){
			0: func() { e.ToggleDrawMode() },
			1: func() { e.UserCommand(c, status, "save") },
			2: func() { e.UserCommand(c, status, "quit") },
		}
		extraDashes = false
		menuChoices = make([]string, len(actionTitles))
	)

	// Create a list of strings that are menu choices,
	// while also creating a mapping from the menu index to a function.
	for i, description := range actionTitles {
		menuChoices[i] = fmt.Sprintf("[%d] %s", i, description)
	}

	// Launch a generic menu
	useMenuIndex := 0
	if lastMenuIndex > 0 {
		useMenuIndex = lastMenuIndex
	}

	selected := e.Menu(status, tty, "Select an action", menuChoices, menuTitleColor, menuArrowColor, menuTextColor, menuHighlightColor, menuSelectedColor, useMenuIndex, extraDashes)

	// Redraw the editor contents
	e.DrawLines(c, true, false)

	if selected < 0 {
		// Output the selected item text
		status.SetMessage("No action taken")
		status.Show(c, e)

		// Do not immediately redraw the editor
		return false, selected
	}

	// Perform the selected command (call the function from the functionMap above)
	actionFunctions[selected]()

	// Redraw editor
	return true, selected
}
