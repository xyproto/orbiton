package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/xyproto/env"
	"github.com/xyproto/guessica"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	lastCommandFile = filepath.Join(userCacheDir, "o", "last_command.sh")
	foundGDB        = which("gdb") != ""
	changedTheme    bool // has the theme been changed manually after the editor was started?
)

// UserSave saves the file and the location history
func (e *Editor) UserSave(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar) {
	// Save the file
	if err := e.Save(c, tty); err != nil {
		status.SetError(err)
		status.Show(c, e)
		return
	}

	// Save the current location in the location history and write it to file
	if absFilename, err := e.AbsFilename(); err == nil { // no error
		e.SaveLocation(absFilename, e.locationHistory)
	}

	// Status message
	status.Clear(c)
	status.SetMessage("Saved " + e.filename)
	status.Show(c, e)
}

// Actions is a list of action titles and a list of action functions.
// The key is an int that is the same for both.
type Actions struct {
	actionTitles    map[int]string
	actionFunctions map[int]func()
}

// NewActions will create a new Actions struct
func NewActions() *Actions {
	var a Actions
	a.actionTitles = make(map[int]string)
	a.actionFunctions = make(map[int]func())
	return &a
}

// NewActions2 will create a new Actions struct, while
// initializing it with the given slices of titles and functions
func NewActions2(actionTitles []string, actionFunctions []func()) (*Actions, error) {
	a := NewActions()
	if len(actionTitles) != len(actionFunctions) {
		return nil, errors.New("length of action titles and action functions differ")
	}
	for i, title := range actionTitles {
		a.actionTitles[i] = title
		a.actionFunctions[i] = actionFunctions[i]
	}
	return a, nil
}

// Add will add an action title and an action function
func (a *Actions) Add(title string, f func()) {
	i := len(a.actionTitles)
	a.actionTitles[i] = title
	a.actionFunctions[i] = f
}

// MenuChoices will return a string that lists the titles of
// the available actions.
func (a *Actions) MenuChoices() []string {
	// Create a list of strings that are menu choices,
	// while also creating a mapping from the menu index to a function.
	menuChoices := make([]string, len(a.actionTitles))
	for i, description := range a.actionTitles {
		menuChoices[i] = fmt.Sprintf("[%d] %s", i, description)
	}
	return menuChoices
}

// Perform will call the given function index
func (a *Actions) Perform(index int) {
	a.actionFunctions[index]()
}

// CommandMenu will display a menu with various commands that can be browsed with arrow up and arrow down.
// Also returns the selected menu index (can be -1), and if a space should be added to the text editor after the return.
func (e *Editor) CommandMenu(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, undo *Undo, lastMenuIndex int, forced bool, lk *LockKeeper) (int, bool) {

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
		extraDashes         bool
		addSpaceAfterReturn bool
	)

	// Add initial menu titles and actions
	// Remember to add "undo.Snapshot(e)" in front of function calls that may modify the current file!
	actions, err := NewActions2(
		[]string{
			"Save and quit",
			wrapWhenTypingToggleText,
			"Word wrap at " + strconv.Itoa(wrapWidth),
			"Sort strings on the current line",
			"Insert \"" + insertFilename + "\" at the current line",
			"Insert the current date", // in the RFC 3339 format
		},
		[]func(){
			func() { // save and quit
				e.clearOnQuit = true
				e.UserSave(c, tty, status)
				e.quit = true        // indicate that the user wishes to quit
				e.clearOnQuit = true // clear the terminal after quitting
			},
			func() { // toggle word wrap when typing
				e.wrapWhenTyping = !e.wrapWhenTyping
				if e.wrapWidth == 0 {
					e.wrapWidth = 79
				}
			},
			func() { // word wrap
				// word wrap at the current width - 5, with an allowed overshoot of 5 runes
				tmpWrapAt := e.wrapWidth
				e.wrapWidth = wrapWidth
				if e.WrapAllLinesAt(wrapWidth-5, 5) {
					e.redraw = true
					e.redrawCursor = true
				}
				e.wrapWidth = tmpWrapAt
			},
			func() { // sort strings on the current line
				undo.Snapshot(e)
				if err := e.SortStrings(c, status); err != nil {
					status.Clear(c)
					status.SetError(err)
					status.Show(c, e)
				}
			},
			func() { // insert file
				editedFileDir := filepath.Dir(e.filename)
				if err := e.InsertFile(c, filepath.Join(editedFileDir, insertFilename)); err != nil {
					status.Clear(c)
					status.SetError(err)
					status.Show(c, e)
				}
			},
			func() { // insert current date
				// note that if a space is added after the string here, it will be stripped when the command menu disappears
				dateString := time.Now().Format(time.RFC3339)[:10]
				e.InsertString(c, dateString)
				addSpaceAfterReturn = true
			},
		},
	)
	if err != nil {
		// If this happens, menu actions and menu functions are not added properly
		// and it should fail hard, so that this can be fixed.
		panic(err)
	}

	// Special menu option for PKGBUILD files
	if strings.HasSuffix(e.filename, "PKGBUILD") {
		actions.Add("Call Guessica", func() {
			status.Clear(c)
			status.SetMessage("Calling Guessica")
			status.Show(c, e)

			// Use the temporary directory defined in TMPDIR, with fallback to /tmp
			tempdir := env.Str("TMPDIR", "/tmp")

			tempFilename := ""

			var (
				f   *os.File
				err error
			)
			if f, err = ioutil.TempFile(tempdir, "__o*"+"guessica"); err == nil {
				// no error, everything is fine
				tempFilename = f.Name()
				// TODO: Implement e.SaveAs
				oldFilename := e.filename
				e.filename = tempFilename
				err = e.Save(c, tty)
				e.filename = oldFilename
			}
			if err != nil {
				status.SetError(err)
				status.Show(c, e)
				return
			}

			if tempFilename == "" {
				status.SetErrorMessage("Could not create a temporary file")
				status.Show(c, e)
				return
			}

			// Show the status message to the user right now
			status.Draw(c, e.pos.offsetY)

			// Call Guessica, which may take a little while
			err = guessica.UpdateFile(tempFilename)

			if err != nil {
				status.SetErrorMessage("Failed to update PKGBUILD: " + err.Error())
				status.Show(c, e)
			} else {
				if _, err := e.Load(c, tty, FilenameOrData{tempFilename, []byte{}}); err != nil {
					status.ClearAll(c)
					status.SetMessage(err.Error())
					status.Show(c, e)
				}
				// Mark the data as changed, despite just having loaded a file
				e.changed = true
				e.redrawCursor = true

			}
		})
	}

	// Copy all the text to the clipboard, if possible
	actions.Add("Copy everything to clipboard", func() { // copy file to clipboard
		// Write all contents to the clipboard
		if err := clipboard.WriteAll(e.String()); err != nil {
			status.Clear(c)
			status.SetError(err)
			status.Show(c, e)
		} else {
			status.ShowAfterRedraw("Copied everything")
		}
	})

	// Debug mode on/off, if gdb is found and the mode is tested
	if foundGDB && e.usingGDBMightWork() {
		if e.debugMode {
			actions.Add("Exit debug mode", func() {
				status.Clear(c)
				status.SetMessage("Debug mode disabled")
				status.Show(c, e)
				e.debugMode = false
				// Also end the gdb session if there is one in progress
				e.DebugEnd()
				status.ShowAfterRedraw("Normal mode")
			})
		} else {
			actions.Add("Debug mode", func() {
				// Save the file when entering debug mode, since gdb may crash for some languages
				// TODO: Identify which languages work poorly together with gdb
				e.UserSave(c, tty, status)

				status.Clear(c)
				status.SetMessage("Debug mode enabled")
				status.Show(c, e)

				e.debugMode = true
			})
		}
	}

	if e.debugMode {
		hasOutputData := len(strings.TrimSpace(gdbOutput.String())) > 0
		if hasOutputData {
			if e.debugHideOutput {
				actions.Add("Show output pane", func() {
					e.debugHideOutput = true
				})
			} else {
				actions.Add("Hide output pane", func() {
					e.debugHideOutput = true
				})
			}
		}
	}

	// Add the syntax highlighting toggle menu item
	if !envNoColor {
		syntaxToggleText := "Disable syntax highlighting"
		if !e.syntaxHighlight {
			syntaxToggleText = "Enable syntax highlighting"
		}
		actions.Add(syntaxToggleText, func() {
			e.ToggleSyntaxHighlight()
		})
	}

	// Delete the rest of the file
	actions.Add("Delete the rest of the file", func() { // copy file to clipboard

		prepareFunction := func() {
			// Prepare to delete all lines from this one and out
			undo.Snapshot(e)
			// Also close the portal, if any
			ClosePortal(e)
			// Mark the file as changed
			e.changed = true
		}

		// Get the current index and remove the rest of the lines
		currentLineIndex := int(e.DataY())

		for y := range e.lines {
			if y >= currentLineIndex {
				// Run the prepareFunction, but only once, if there was changes to be made
				if prepareFunction != nil {
					prepareFunction()
					prepareFunction = nil
				}
				delete(e.lines, y)
			}
		}

		if e.changed {
			e.MakeConsistent()
			e.redraw = true
			e.redrawCursor = true
		}
	})

	// Add the unlock menu item
	if forced {
		// TODO: Detect if file is locked first
		actions.Add("Unlock if locked", func() {
			if absFilename, err := e.AbsFilename(); err == nil { // no issues
				lk.Load()
				lk.Unlock(absFilename)
				lk.Save()
			}
		})

	}

	// Render to PDF using the gofpdf package
	actions.Add("Render to PDF", func() {

		// Write to PDF in a goroutine
		pdfFilename := strings.Replace(filepath.Base(e.filename), ".", "_", -1) + ".pdf"

		// Show a status message while writing
		status.SetMessage("Writing " + pdfFilename + "...")
		status.ShowNoTimeout(c, e)

		statusMessage := ""

		// TODO: Only overwrite if the previous PDF file was also rendered by "o".
		_ = os.Remove(pdfFilename)
		// Write the file
		if err := e.SavePDF(e.filename, pdfFilename); err != nil {
			statusMessage = err.Error()
		} else {
			statusMessage = "Wrote " + pdfFilename
		}
		// Show a status message after writing
		status.ClearAll(c)
		status.SetMessage(statusMessage)
		status.ShowNoTimeout(c, e)
	})

	// Render to PDF using pandoc
	if (e.mode == mode.Markdown || e.mode == mode.Doc) && which("pandoc") != "" {
		actions.Add("Render to PDF using pandoc", func() {
			go func() {
				pandocMutex.Lock()
				// The last argument is if pandoc should run in the background or not
				_, err := e.BuildOrExport(c, tty, status, e.filename, false)
				// Could an action be performed for this file extension?
				if err != nil {
					status.SetError(err)
				}
				status.ShowNoTimeout(c, e)
				pandocMutex.Unlock()
			}()
		})
	}

	if !envNoColor || changedTheme {
		// Add an option for selecting a theme
		actions.Add("Change theme", func() {
			menuChoices := allThemes
			useMenuIndex := 0
			for i, menuChoiceText := range menuChoices {
				if menuChoiceText == e.Theme.Name {
					useMenuIndex = i
				}
			}
			switch e.Menu(status, tty, "Select color theme", menuChoices, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, useMenuIndex, extraDashes) {
			case 0: // default
				envNoColor = false
				e.setDefaultTheme()
				e.syntaxHighlight = true
				changedTheme = true
			case 1: // light background
				envNoColor = false
				e.setLightTheme()
				e.syntaxHighlight = true
				changedTheme = true
			case 2: // red and black
				envNoColor = false
				e.setRedBlackTheme()
				e.syntaxHighlight = true
				changedTheme = true
			case 3: // amber
				envNoColor = false
				e.setAmberTheme()
				e.syntaxHighlight = false
				changedTheme = true
			case 4: // green
				envNoColor = false
				e.setGreenTheme()
				e.syntaxHighlight = false
				changedTheme = true
			case 5: // blue
				envNoColor = false
				e.setBlueTheme()
				e.syntaxHighlight = false
				changedTheme = true
			case 6: // no color
				envNoColor = true
				e.setDefaultTheme()
				changedTheme = true
			default:
				return
			}
			drawLines := true
			resized := false
			e.FullResetRedraw(c, status, drawLines, resized)
		})
	}

	actions.Add("Stop parent and quit without saving", func() {
		e.stopParentOnQuit = true
		e.clearOnQuit = true
		e.quit = true        // indicate that the user wishes to quit
		e.clearOnQuit = true // clear the terminal after quitting
	})

	menuChoices := actions.MenuChoices()

	// Launch a generic menu
	useMenuIndex := 0
	if lastMenuIndex > 0 {
		useMenuIndex = lastMenuIndex
	}

	selected := e.Menu(status, tty, "Menu", menuChoices, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, useMenuIndex, extraDashes)

	// Redraw the editor contents
	//e.DrawLines(c, true, false)

	if selected < 0 {
		// Output the selected item text
		status.SetMessage("No action taken")
		status.Show(c, e)

		// Do not immediately redraw the editor
		e.redraw = false
		return selected, addSpaceAfterReturn
	}

	// Perform the selected action by passing the function index
	actions.Perform(selected)

	// Redraw editor
	e.redraw = true
	e.redrawCursor = true
	return selected, addSpaceAfterReturn
}

// getCommand takes an *exec.Cmd and returns the command
// it represents, but with "/usr/bin/sh -c " trimmed away.
func getCommand(cmd *exec.Cmd) string {
	s := cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
	return strings.TrimPrefix(s, "/usr/bin/sh -c ")
}

// Save the command to a temporary file, given an exec.Cmd struct
func saveCommand(cmd *exec.Cmd) error {

	p := lastCommandFile

	// First create the folder for the lock file overview, if needed
	folderPath := filepath.Dir(p)
	os.MkdirAll(folderPath, os.ModePerm)

	// Prepare the file
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700)
	if err != nil {
		return err
	}
	defer f.Close()

	// Strip the leading /usr/bin/sh -c command, if present
	commandString := getCommand(cmd)

	// Write the contents, ignore the number of written bytes
	_, err = f.WriteString(fmt.Sprintf("#!/bin/sh\n%s\n", commandString))
	return err
}
