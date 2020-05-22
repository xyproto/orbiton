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
	case "toggledrawmode":
		e.ToggleDrawMode()
		statusMessage := "Text mode"
		if e.DrawMode() {
			statusMessage = "Draw mode"
		}
		status.Clear(c)
		status.SetMessage(statusMessage)
		status.Show(c, e)
	case "sortstrings":
		// sort the list of comma or space separated strings, either quoted with ", with ' or "bare"
		e.SortStrings(c, status)
	}
}

// CommandMenu will display a menu with various commands that can be browsed with arrow up and arrow down
// Also returns the selected menu index (can be -1).
func (e *Editor) CommandMenu(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, undo *Undo, lastMenuIndex int) int {

	drawModeStatus := "on"
	if !e.drawMode {
		drawModeStatus = "off"
	}

	syntaxStatus := "on"
	if !e.syntaxHighlight {
		syntaxStatus = "off"
	}

	var (
		// These numbers must correspond with actionFunctions!
		actionTitles = map[int]string{
			0: "Save and quit",
			1: "Sort the list of strings on the current line",
			2: "Toggle syntax highlighting (currently " + syntaxStatus + ")",
			3: "Toggle draw mode (currently " + drawModeStatus + ")",
			4: "Amber mode",
			5: "Save",
			/*
				6: "Green mode",
				7: "Blue mode",
			*/
		}
		// These numbers must correspond with actionTitles!
		// Remember to add "undo.Snapshot(e)" in front of function calls that may modify the current file.
		actionFunctions = map[int]func(){
			0: func() { e.UserCommand(c, status, "save"); e.UserCommand(c, status, "quit") },
			1: func() {
				undo.Snapshot(e)
				err := e.SortStrings(c, status)
				if err != nil {
					status.Clear(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
				}
			},
			2: func() { e.ToggleSyntaxHighlight() },
			3: func() { e.ToggleDrawMode() },
			4: func() {
				e.fg = vt100.Yellow
				e.SetSyntaxHighlight(false)
				e.bg = vt100.BackgroundDefault
				e.DrawLines(c, true, true)
				c = e.FullResetRedraw(c, status)
			},
			5: func() { e.UserCommand(c, status, "save") },
			/*
				6: func() {
					e.fg = vt100.LightGreen
					e.SetSyntaxHighlight(false)
					e.bg = vt100.BackgroundDefault
					e.DrawLines(c, true, true)
					c = e.FullResetRedraw(c, status)
				},
				7: func() {
					e.fg = vt100.LightBlue
					e.SetSyntaxHighlight(false)
					e.bg = vt100.BackgroundDefault
					e.DrawLines(c, true, true)
					c = e.FullResetRedraw(c, status)
				},
			*/
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
		e.redraw = false
		return selected
	}

	// Perform the selected command (call the function from the functionMap above)
	actionFunctions[selected]()

	// Redraw editor
	e.redraw = true
	return selected
}
