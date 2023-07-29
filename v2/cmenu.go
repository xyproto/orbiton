package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/guessica"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	lastCommandFile      = filepath.Join(userCacheDir, "o", "last_command.sh")
	lastCommandMenuIndex int  // for the command menu
	changedTheme         bool // has the theme been changed manually after the editor was started?
)

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
		e.SaveLocation(absFilename, locationHistory)
	}

	// Status message
	status.Clear(c)
	status.SetMessage("Saved " + e.filename)
	status.Show(c, e)
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

// AddCommand will add a command to the action menu, if it can be looked up by e.CommandToFunction
func (a *Actions) AddCommand(e *Editor, c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, undo *Undo, title string, args ...string) error {
	f, err := e.CommandToFunction(c, tty, status, bookmark, undo, args...)
	if err != nil {
		// panic(err)
		return err
	}
	a.Add(title, f)
	return nil
}

// CommandMenu will display a menu with various commands that can be browsed with arrow up and arrow down.
// Also returns the selected menu index (can be -1), and if a space should be added to the text editor after the return.
// TODO: Figure out why this function needs an undo argument and can't use the regular one
func (e *Editor) CommandMenu(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, undo *Undo, lastMenuIndex int, forced bool, lk *LockKeeper) int {
	const insertFilename = "include.txt"
	const menuTitle = versionString

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

	var (
		extraDashes bool
		actions     = NewActions()
	)

	// TODO: Create a string->[]string map from title to command, then add them
	// TODO: Add the 6 first arguments to a context struct instead
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Save and quit", "savequitclear")
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Sort strings on the current line", "sortwords")
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Sort the current block of lines", "sortblock")
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Insert \""+insertFilename+"\" at the current line", "insertfile", insertFilename)
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Insert the current date", "insertdate") // in the RFC 3339 format
	actions.AddCommand(e, c, tty, status, bookmark, undo, "Insert the current time", "inserttime")

	// Word wrap at a custom width + enable word wrap when typing
	actions.Add("Word wrap at...", func() {
		if wordWrapString, ok := e.UserInput(c, tty, status, fmt.Sprintf("Word wrap at [%d]", wrapWidth), []string{}, false); ok {
			if strings.TrimSpace(wordWrapString) == "" {
				e.WrapNow(wrapWidth)
				e.wrapWhenTyping = true
				status.SetMessageAfterRedraw(fmt.Sprintf("Word wrap at %d", wrapWidth))
			} else {
				if ww, err := strconv.Atoi(wordWrapString); err != nil {
					status.Clear(c)
					status.SetError(err)
					status.Show(c, e)
				} else {
					e.WrapNow(ww)
					e.wrapWhenTyping = true
					status.SetMessageAfterRedraw(fmt.Sprintf("Word wrap at %d", wrapWidth))
				}
			}
		}
	})

	// Enter ChatGPT API key, if it's not already set
	if openAIKeyHolder == nil {
		actions.Add("Enter ChatGPT API key...", func() {
			if enteredAPIKey, ok := e.UserInput(c, tty, status, "API key from https://platform.openai.com/account/api-keys", []string{}, false); ok {
				openAIKeyHolder = NewKeyHolderWithKey(enteredAPIKey)
				// env.Set("CHATGPT_API_KEY", enteredAPIKey)
				status.SetMessageAfterRedraw("Using API key " + enteredAPIKey)
				// Write the OpenAI API Key to a file in the cache directory as well, but ignore errors
				_ = openAIKeyHolder.WriteAPIKey()
			}
		})
	}

	// Build (for use on the terminal, since ctrl-space does not work on iTerm2 + macOS)
	if !env.Bool("OG") && runtime.GOOS == "darwin" {
		actions.Add("Build", func() {
			const andRun = false
			e.Build(c, status, tty, andRun)
		})
		if e.CanRun() {
			actions.Add("Build and run", func() {
				const andRun = true
				e.Build(c, status, tty, andRun)
			})
		}
	}

	// Disable or enable word wrap when typing
	if e.wrapWhenTyping {
		actions.Add("Disable word wrap when typing", func() {
			e.wrapWhenTyping = false
			if e.wrapWidth == 0 {
				e.wrapWidth = wrapWidth
			}
		})
	} else {
		actions.Add("Enable word wrap when typing", func() {
			e.wrapWhenTyping = true
			if e.wrapWidth == 0 {
				e.wrapWidth = wrapWidth
			}
		})
	}

	// Special menu option for PKGBUILD and APKBUILD files
	if strings.HasSuffix(e.filename, "PKGBUILD") || strings.HasSuffix(e.filename, "APKBUILD") {
		actions.Add("Call Guessica", func() {
			status.Clear(c)
			status.SetMessage("Calling Guessica")
			status.Show(c, e)

			tempFilename := ""

			var (
				f   *os.File
				err error
			)
			if f, err = os.CreateTemp(tempDir, "__o*"+"guessica"); err == nil {
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
				if _, err := e.Load(c, tty, FilenameOrData{tempFilename, []byte{}, 0, false}); err != nil {
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

	actions.AddCommand(e, c, tty, status, bookmark, undo, "Copy all text to the clipboard", "copyall")

	// Disable or enable the tag-expanding behavior when typing in HTML or XML
	if e.mode == mode.HTML || e.mode == mode.XML {
		if !e.noExpandTags {
			actions.Add("Disable tag expansion when typing", func() {
				e.noExpandTags = true
			})
		} else {
			actions.Add("Enable tag expansion when typing", func() {
				e.noExpandTags = false
			})
		}
	}

	// Find the path to either "rust-gdb" or "gdb", depending on the mode, then check if it's there
	foundGDB := e.findGDB() != ""

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
				status.SetMessageAfterRedraw("Normal mode")
			})
		} else {
			actions.Add("Debug mode", func() {
				// Save the file when entering debug mode, since gdb may crash for some languages
				// TODO: Identify which languages work poorly together with gdb
				e.UserSave(c, tty, status)
				status.SetMessageAfterRedraw("Debug mode enabled")
				e.debugMode = true
			})
		}
	}

	// Fix as you type mode, on/off
	if openAIKeyHolder != nil { // has AI
		if e.fixAsYouType {
			actions.Add("Fix as you type [turn off]", func() {
				e.fixAsYouType = false
				status.SetMessageAfterRedraw("Fix as you type turned off")
			})
		} else {
			actions.Add("Fix as you type", func() {
				e.fixAsYouType = true
				status.SetMessageAfterRedraw("Fix as you type turned on")
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
			e.ClosePortal()
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
		pdfFilename := strings.ReplaceAll(filepath.Base(e.filename), ".", "_") + ".pdf"

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
			menuChoices := []string{
				"Default",
				"Synthwave      (O_THEME=synthwave)",
				"Red & black    (O_THEME=redblack)",
				"VS             (O_THEME=vs)",
				"Blue Edit      (O_THEME=blueedit)",
				"Amber Mono     (O_THEME=ambermono)",
				"Green Mono     (O_THEME=greenmono)",
				"Blue Mono      (O_THEME=bluemono)",
				"No colors      (NO_COLOR=1)"}
			useMenuIndex := 0
			for i, menuChoiceText := range menuChoices {
				if strings.HasPrefix(e.Theme.Name, menuChoiceText) {
					useMenuIndex = i
				}
			}
			changedTheme = true
			switch e.Menu(status, tty, "Select color theme", menuChoices, e.Background, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, useMenuIndex, extraDashes) {
			case 0: // Default
				envNoColor = false
				e.setDefaultTheme()
				e.syntaxHighlight = true
			case 1: // Synthwave
				envNoColor = false
				e.setSynthwaveTheme()
				e.syntaxHighlight = true
			case 2: // Red & black
				envNoColor = false
				e.setRedBlackTheme()
				e.syntaxHighlight = true
			case 3: // VS
				envNoColor = false
				e.setVSTheme()
				e.syntaxHighlight = true
			case 4: // Blue Edit
				envNoColor = false
				e.setBlueEditTheme()
				e.syntaxHighlight = true
			case 5: // Amber Mono
				envNoColor = false
				e.setAmberTheme()
				e.syntaxHighlight = false
			case 6: // Green Mono
				envNoColor = false
				e.setGreenTheme()
				e.syntaxHighlight = false
			case 7: // Blue Mono
				envNoColor = false
				e.setBlueTheme()
				e.syntaxHighlight = false
			case 8: // No color
				envNoColor = true
				e.setNoColorTheme()
				e.syntaxHighlight = false
			default:
				changedTheme = false
				return
			}
			drawLines := true
			e.FullResetRedraw(c, status, drawLines)
		})
	}

	searchProcessNames := []string{"ag", "find", "rg"}

	// Only show the menu option for killing the parent process if the parent process is "ag", "find" or "rg"
	if firstWordContainsOneOf(parentCommand(), searchProcessNames) {
		actions.Add("Kill parent and quit without saving", func() {
			e.stopParentOnQuit = true
			e.clearOnQuit = true
			e.quit = true        // indicate that the user wishes to quit
			e.clearOnQuit = true // clear the terminal after quitting
		})
	} else {
		actions.Add("Exit without saving", func() {
			e.stopParentOnQuit = false
			e.clearOnQuit = true
			e.quit = true        // indicate that the user wishes to quit
			e.clearOnQuit = true // clear the terminal after quitting
		})
	}

	menuChoices := actions.MenuChoices()

	// Launch a generic menu
	useMenuIndex := 0
	if lastMenuIndex > 0 {
		useMenuIndex = lastMenuIndex
	}

	selected := e.Menu(status, tty, menuTitle, menuChoices, e.Background, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, useMenuIndex, extraDashes)

	// Redraw the editor contents
	// e.DrawLines(c, true, false)

	if selected < 0 {
		// Esc was pressed, or an item was otherwise not selected.
		// Trigger a redraw and return.
		e.redraw = true
		e.redrawCursor = true
		return selected
	}

	// Perform the selected action by passing the function index
	actions.Perform(selected)

	// Adjust the cursor placement
	if e.AfterEndOfLine() {
		e.End(c)
	}

	// Redraw editor
	e.redraw = true
	e.redrawCursor = true

	return selected
}

// getCommand takes an *exec.Cmd and returns the command
// it represents, but with "/usr/bin/sh -c " trimmed away.
func getCommand(cmd *exec.Cmd) string {
	s := cmd.Path + " " + strings.Join(cmd.Args[1:], " ")
	return strings.TrimPrefix(s, "/usr/bin/sh -c ")
}

// Save the command to a temporary file, given an exec.Cmd struct
func saveCommand(cmd *exec.Cmd) error {
	if noWriteToCache {
		return nil
	}

	p := lastCommandFile

	// First create the folder for the lock file overview, if needed
	folderPath := filepath.Dir(p)
	os.MkdirAll(folderPath, os.ModePerm)

	// Prepare the file
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
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
