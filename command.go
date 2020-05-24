package main

import (
	"fmt"
	"path/filepath"
	"strings"

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

	syntaxToggleText := "Disable syntax highlighting"
	if !e.syntaxHighlight {
		syntaxToggleText = "Enable syntax highlighting"
	}

	var (
		// These numbers must correspond with actionFunctions!
		actionTitles = map[int]string{
			0: "Save and quit",
			1: "Sort the list of strings on the current line",
			2: "Amber text",
			3: "Green text",
			4: "Blue text",
			5: syntaxToggleText,
		}
		// These numbers must correspond with actionTitles!
		// Remember to add "undo.Snapshot(e)" in front of function calls that may modify the current file.
		actionFunctions = map[int]func(){
			//0: func() { e.UserCommand(c, status, "save") },
			//0: func() { e.ToggleDrawMode() },
			0: func() { // save and quit
				e.UserCommand(c, status, "save")
				e.UserCommand(c, status, "quit")
			},
			1: func() { // sort strings on the current line
				undo.Snapshot(e)
				err := e.SortStrings(c, status)
				if err != nil {
					status.Clear(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					return // from anonymous function
				}
			},
			2: func() { // amber text
				// Clear and redraw, with syntax highlighting
				vt100.Clear()
				e.SetSyntaxHighlight(true)
				e.DrawLines(c, true, true)
				// Set the color and redraw, without syntax highlighting
				e.fg = vt100.Yellow
				e.SetSyntaxHighlight(false)
				e.DrawLines(c, true, true)
			},
			3: func() { // green text
				// Clear and redraw, with syntax highlighting
				vt100.Clear()
				e.SetSyntaxHighlight(true)
				e.DrawLines(c, true, true)
				// Set the color and redraw, without syntax highlighting
				e.fg = vt100.LightGreen
				e.SetSyntaxHighlight(false)
				e.DrawLines(c, true, true)
			},
			4: func() { // blue text
				// Clear and redraw, with syntax highlighting
				vt100.Clear()
				e.SetSyntaxHighlight(true)
				e.DrawLines(c, true, true)
				// Set the color and redraw, without syntax highlighting
				e.fg = vt100.LightBlue
				e.SetSyntaxHighlight(false)
				e.DrawLines(c, true, true)
			},
			5: func() { // toggle syntax highlighting
				e.ToggleSyntaxHighlight()
			},
		}
		extraDashes = false
		menuChoices = make([]string, len(actionTitles))
	)

	// Add an action for updating the source= line if this is a PKGBUILD file
	if filepath.Base(e.filename) == "PKGBUILD" {
		actionTitles[len(actionTitles)-1] = "Update PKGBUILD"
		actionFunctions[len(actionFunctions)-1] = func() { // update the source= line

			status.SetMessage("Finding new version and commit hash...")
			status.ShowNoTimeout(c, e)

			undo.Snapshot(e)
			pkgverString, sourceString, err := GuessSourceString(e.String())
			if err != nil {
				status.Clear(c)
				status.SetErrorMessage(err.Error())
				status.Show(c, e)
				return // from anonymous function
			}

			for i, runeLine := range e.lines {
				line := string(runeLine)
				if strings.HasPrefix(line, "source=") {
					e.lines[i] = []rune(sourceString)
				} else if strings.HasPrefix(line, "pkgver=") {
					e.lines[i] = []rune(pkgverString)
				}
			}
		}
	}

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
	//e.DrawLines(c, true, false)

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
	e.redrawCursor = true
	return selected
}
