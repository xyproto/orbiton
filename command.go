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
		if err := e.Save(c); err != nil {
			status.SetMessage(err.Error())
			status.Show(c, e)
			break
		}
		// Save the current location in the location history and write it to file
		absFilename, err := filepath.Abs(e.filename)
		if err == nil { // no error
			e.SaveLocation(absFilename, e.locationHistory)
		}
		// Status message
		status.SetMessage("Saved " + e.filename)
		status.Show(c, e)

		e.pos.offsetX = 0
		c.Draw()
	case "quit":
		e.quit = true        // indicate that the user wishes to quit
		e.clearOnQuit = true // clear the terminal after quitting
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
			2: syntaxToggleText,
		}
		// These numbers must correspond with actionTitles!
		// Remember to add "undo.Snapshot(e)" in front of function calls that may modify the current file.
		actionFunctions = map[int]func(){
			//0: func() { e.UserCommand(c, status, "save") },
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
					//return // from anonymous function
				}
			},
			2: func() { // toggle syntax highlighting
				e.ToggleSyntaxHighlight()
			},
		}
		extraDashes = false
	)

	// Add the option to change the colors, for non-light themes (fg != black)
	if !e.lightTheme { // Not a light theme
		// TODO: Use a fixed order instead of a random order
		colors := []vt100.AttributeColor{
			vt100.Yellow,
			vt100.LightGreen,
			vt100.LightBlue,
		}
		colorText := []string{
			"Amber",
			"Green",
			"Blue",
		}
		// Add menu items and menu functions for changing the text color
		// while also turning off syntax highlighting.
		for i, color := range colors {
			actionTitles[len(actionTitles)] = colorText[i] + " text"
			color := color // per-loop copy of the color variable, since it's closed over
			actionFunctions[len(actionFunctions)] = func() {
				e.fg = color
				e.syntaxHighlight = false
				c = e.FullResetRedraw(c, status)
				e.DrawLines(c, true, false)
				e.redraw = true
				e.redrawCursor = true
			}
		}
	}

	// Add an action for updating the source= line if this is a PKGBUILD file
	if filepath.Base(e.filename) == "PKGBUILD" {
		actionTitles[len(actionTitles)] = "Update PKGBUILD"
		actionFunctions[len(actionFunctions)] = func() { // update the source= line

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
	menuChoices := make([]string, len(actionTitles))
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
