package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xyproto/vt100"
)

// UserSave saves the file and the location history
func (e *Editor) UserSave(c *vt100.Canvas, status *StatusBar) {
	status.ClearAll(c)
	// Save the file
	if err := e.Save(c); err != nil {
		status.SetErrorMessage(err.Error())
		status.Show(c, e)
		return
	}
	// Save the current location in the location history and write it to file
	if absFilename, err := e.AbsFilename(); err == nil { // no error
		e.SaveLocation(absFilename, e.locationHistory)
	}
	// Status message
	status.SetMessage("Saved " + e.filename)
	status.Show(c, e)

	e.pos.offsetX = 0
	c.Draw()
}

// CommandMenu will display a menu with various commands that can be browsed with arrow up and arrow down
// Also returns the selected menu index (can be -1).
func (e *Editor) CommandMenu(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, undo *Undo, lastMenuIndex int, forced bool, lk *LockKeeper) int {

	const insertFilename = "include.txt"

	wrapWidth := e.wrapWidth
	if wrapWidth == 0 {
		wrapWidth = 80
	}

	// Let the menu item for wrapping words suggest the minimum of e.wrapWidth and the terminal width
	if c != nil {
		w := int(c.Width())
		if w < wrapWidth {
			wrapWidth = w - int(0.05*float64(w))
		}
	}

	wrapWhenTypingToggleText := "Enable word wrap when typing"
	if e.wrapWhenTyping {
		wrapWhenTypingToggleText = "Disable word wrap when typing"
	}

	var (
		noColor = os.Getenv("NO_COLOR") != ""

		// These numbers must correspond with actionFunctions!
		actionTitles = map[int]string{
			0: "Save and quit",
			1: wrapWhenTypingToggleText,
			2: "Word wrap at " + strconv.Itoa(wrapWidth),
			3: "Sort the list of strings on the current line",
			4: "Insert \"" + insertFilename + "\" at the current line",
		}
		// These numbers must correspond with actionTitles!
		// Remember to add "undo.Snapshot(e)" in front of function calls that may modify the current file.
		actionFunctions = map[int]func(){
			0: func() { // save and quit
				e.clearOnQuit = true
				e.UserSave(c, status)
				e.quit = true        // indicate that the user wishes to quit
				e.clearOnQuit = true // clear the terminal after quitting
			},
			1: func() { // toggle word wrap when typing
				e.wrapWhenTyping = !e.wrapWhenTyping
				if e.wrapWidth == 0 {
					e.wrapWidth = 79
				}
			},
			2: func() { // word wrap
				// word wrap at the current width - 5, with an allowed overshoot of 5 runes
				tmpWrapAt := e.wrapWidth
				e.wrapWidth = wrapWidth
				if e.WrapAllLinesAt(wrapWidth-5, 5) {
					e.redraw = true
					e.redrawCursor = true
				}
				e.wrapWidth = tmpWrapAt
			},
			3: func() { // sort strings on the current line
				undo.Snapshot(e)
				if err := e.SortStrings(c, status); err != nil {
					status.Clear(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
				}
			},
			4: func() { // insert file
				editedFileDir := filepath.Dir(e.filename)
				if err := e.InsertFile(c, filepath.Join(editedFileDir, insertFilename)); err != nil {
					status.Clear(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
				}
			},
		}
		extraDashes = false
	)

	// If this is a C++ source file, present a menu option for open the corresponding header file
	if hasS([]string{".cpp", ".cc", ".c", ".cxx"}, filepath.Ext(e.filename)) {
		// Check if there is a corresponding header file
		if absFilename, err := e.AbsFilename(); err == nil { // no error
			headerExtensions := []string{".h", ".hpp"}
			if headerFilename, err := ExtFileSearch(absFilename, headerExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
				actionTitles[len(actionTitles)] = "Open " + filepath.Base(headerFilename)
				actionFunctions[len(actionFunctions)] = func() {
					// Switch to another file (without forcing it)
					e.Switch(tty, c, status, lk, headerFilename, false)
				}
			}
		}
	}

	// If this is a header file, present a menu option for open the corresponding source file
	if hasS([]string{".h", ".hpp"}, filepath.Ext(e.filename)) {
		// Check if there is a corresponding header file
		if absFilename, err := e.AbsFilename(); err == nil { // no error
			sourceExtensions := []string{".c", ".cpp", ".cxx", ".cc"}
			if headerFilename, err := ExtFileSearch(absFilename, sourceExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
				actionTitles[len(actionTitles)] = "Open " + filepath.Base(headerFilename)
				actionFunctions[len(actionFunctions)] = func() {
					// Switch to another file (without forcing it)
					e.Switch(tty, c, status, lk, headerFilename, false)
				}
			}
		}
	}

	// Add the syntax highlighting toggle menu item
	if !noColor {
		syntaxToggleText := "Disable syntax highlighting"
		if !e.syntaxHighlight {
			syntaxToggleText = "Enable syntax highlighting"
		}
		actionTitles[len(actionTitles)] = syntaxToggleText
		actionFunctions[len(actionFunctions)] = func() {
			e.ToggleSyntaxHighlight()
		}
	}

	// Add the unlock menu
	// TODO: Detect if the current file is locked first
	if forced {
		actionTitles[len(actionTitles)] = "Unlock if locked"
		actionFunctions[len(actionFunctions)] = func() {
			if absFilename, err := e.AbsFilename(); err == nil { // no issues
				lk.Load()
				lk.Unlock(absFilename)
				lk.Save()
			}
		}
	}

	// Add the portal menu
	if portal, err := LoadPortal(); err == nil { // no problems
		actionTitles[len(actionTitles)] = "Close portal at " + portal.String()
		actionFunctions[len(actionFunctions)] = func() {
			ClosePortal()
		}
	} else {
		// Could not close portal, try opening a new one
		if portal, err := e.NewPortal(); err == nil { // no problems
			actionTitles[len(actionTitles)] = "Open portal at " + portal.String()
			actionFunctions[len(actionFunctions)] = func() {
				portal.Save()
			}
		}
	}

	// Add the option to change the colors, for non-light themes (fg != black)
	if !e.lightTheme && !noColor { // Not a light theme and NO_COLOR is not set

		// Add the "Red/Black theme" menu item text and menu function
		actionTitles[len(actionTitles)] = "Red/black theme"
		actionFunctions[len(actionFunctions)] = func() {
			e.setRedBlackTheme()
			e.SetSyntaxHighlight(true)
			e.FullResetRedraw(c, status, true)
		}

		// Add the "Default theme" menu item text and menu function
		actionTitles[len(actionTitles)] = "Default theme"
		actionFunctions[len(actionFunctions)] = func() {
			e.setDefaultTheme()
			e.SetSyntaxHighlight(true)
			e.FullResetRedraw(c, status, true)
		}

		// Add the Amber, Green and Blue theme options
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
			actionTitles[len(actionTitles)] = colorText[i] + " theme"
			color := color // per-loop copy of the color variable, since it's closed over
			actionFunctions[len(actionFunctions)] = func() {
				e.fg = color
				e.bg = vt100.BackgroundDefault // black background
				e.syntaxHighlight = false
				e.FullResetRedraw(c, status, true)
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
