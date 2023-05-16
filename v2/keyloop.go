package main

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/iferr"
	"github.com/xyproto/mode"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// Create a LockKeeper for keeping track of which files are being edited
var fileLock = NewLockKeeper(defaultLockFile)

// The maximum amount of times to display the help text when ctrl-g is pressed
const maxHelpMessages = 3

// Loop will set up and run the main loop of the editor
// a *vt100.TTY struct
// fnord contains either data or a filename to open
// a LineNumber (may be 0 or -1)
// a forceFlag for if the file should be force opened
// If an error and "true" is returned, it is a quit message to the user, and not an error.
// If an error and "false" is returned, it is an error.
func Loop(tty *vt100.TTY, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, theme Theme, syntaxHighlight bool) (userMessage string, stopParent bool, err error) {

	// Create a Canvas for drawing onto the terminal
	vt100.Init()
	c := vt100.NewCanvas()
	c.ShowCursor()
	vt100.EchoOff()

	var (
		statusDuration = 2700 * time.Millisecond

		copyLines         []string  // for the cut/copy/paste functionality
		previousCopyLines []string  // for checking if a paste is the same as last time
		bookmark          *Position // for the bookmark/jump functionality

		firstPasteAction = true
		firstCopyAction  = true

		lastCopyY  LineIndex = -1 // used for keeping track if ctrl-c has been pressed twice on the same line
		lastPasteY LineIndex = -1 // used for keeping track if ctrl-v has been pressed twice on the same line
		lastCutY   LineIndex = -1 // used for keeping track if ctrl-x has been pressed twice on the same line

		clearKeyHistory  bool              // for clearing the last pressed key, for exiting modes that also reads keys
		kh               = NewKeyHistory() // keep track of the previous key presses
		key              string            // for the main loop
		jsonFormatToggle bool              // for toggling indentation or not when pressing ctrl-w for JSON

		helpCounter int // the number of times the help text has been displayed
		jumpMode    bool
	)

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e, messageAfterRedraw, displayedImage, err := NewEditor(tty, c, fnord, lineNumber, colNumber, theme, syntaxHighlight, true)
	if err != nil {
		return "", false, err
	} else if displayedImage {
		// A special case for if an image was displayed instead of a file being opened
		return "", false, nil
	}

	// Find the absolute path to this filename
	absFilename := fnord.filename
	if !fnord.stdin {
		if filename, err := e.AbsFilename(); err == nil { // success
			absFilename = filename
		}
	}

	// Minor adjustments to some modes
	switch e.mode {
	case mode.Email, mode.Git:
		e.StatusForeground = vt100.LightBlue
		e.StatusBackground = vt100.BackgroundDefault
	case mode.ManPage:
		e.readOnly = true
	}

	// Prepare a status bar
	status := NewStatusBar(e.StatusForeground, e.StatusBackground, e.StatusErrorForeground, e.StatusErrorBackground, e, statusDuration, messageAfterRedraw)

	e.SetTheme(e.Theme)

	// ctrl-c, USR1 and terminal resize handlers
	e.SetUpSignalHandlers(c, tty, status)

	e.previousX = 1
	e.previousY = 1

	tty.SetTimeout(2 * time.Millisecond)

	var (
		canUseLocks   = !fnord.stdin
		lockTimestamp time.Time
	)

	// If the lock keeper does not have an overview already, that's fine. Ignore errors from lk.Load().
	if err := fileLock.Load(); err != nil {
		// Could not load an existing lock overview, this might be the first run? Try saving.
		if err := fileLock.Save(); err != nil {
			// Could not save a lock overview. Can not use locks.
			canUseLocks = false
		}
	}

	if canUseLocks {
		// Check if the lock should be forced (also force when running git commit, because it is likely that o was killed in that case)
		if forceFlag || filepath.Base(absFilename) == "COMMIT_EDITMSG" || env.Bool("O_FORCE") {
			// Lock and save, regardless of what the previous status is
			fileLock.Lock(absFilename)
			// TODO: If the file was already marked as locked, this is not strictly needed? The timestamp might be modified, though.
			fileLock.Save()
		} else {
			// Lock the current file, if it's not already locked
			if err := fileLock.Lock(absFilename); err != nil {
				return fmt.Sprintf("Locked by another (possibly dead) instance of this editor.\nTry: o -f %s", filepath.Base(absFilename)), false, errors.New(absFilename + " is locked")
			}
			// Immediately save the lock file as a signal to other instances of the editor
			fileLock.Save()
		}
		lockTimestamp = fileLock.GetTimestamp(absFilename)

		// Set up a catch for panics, so that the current file can be unlocked
		defer func() {
			if x := recover(); x != nil {
				// Unlock and save the lock file
				fileLock.Unlock(absFilename)
				fileLock.Save()

				// Save the current file. The assumption is that it's better than not saving, if something crashes.
				// TODO: Save to a crash file, then let the editor discover this when it starts.
				e.Save(c, tty)

				// Output the error message
				quitMessageWithStack(tty, fmt.Sprintf("%v", x))
			}
		}()
	}

	// Draw everything once, with slightly different behavior if used over ssh
	e.InitialRedraw(c, status)

	// This is the main loop for the editor
	for !e.quit {

		if e.macro == nil || (e.playBackMacroCount == 0 && !e.macro.Recording) {
			// Read the next key in the regular way
			key = tty.String()
			undo.IgnoreSnapshots(false)
		} else {
			if e.macro.Recording {
				undo.IgnoreSnapshots(true)
				// Read and record the next key
				key = tty.String()
				if key != "c:20" { // ctrl-t
					// But never record the macro toggle button
					e.macro.Add(key)
				}
			} else if e.playBackMacroCount > 0 {
				undo.IgnoreSnapshots(true)
				key = e.macro.Next()
				if key == "" || key == "c:20" { // ctrl-t
					e.macro.Home()
					e.playBackMacroCount--
					// No more macro keys. Read the next key.
					key = tty.String()
				}
			}
		}

		switch key {
		case "c:17": // ctrl-q, quit
			e.quit = true
		case "c:23": // ctrl-w, format or insert template (or if in git mode, cycle interactive rebase keywords)

			undo.Snapshot(e)

			// Clear the search term
			e.ClearSearchTerm()

			// Add a watch
			if e.debugMode { // AddWatch will start a new gdb session if needed
				// Ask the user to type in a watch expression
				if expression, ok := e.UserInput(c, tty, status, "Variable name to watch", []string{}, false); ok {
					if _, err := e.AddWatch(expression); err != nil {
						status.ClearAll(c)
						status.SetError(err)
						status.ShowNoTimeout(c, e)
						break
					}
				}
				break
			}

			// Cycle git rebase keywords
			if line := e.CurrentLine(); e.mode == mode.Git && hasAnyPrefixWord(line, gitRebasePrefixes) {
				newLine := nextGitRebaseKeyword(line)
				e.SetCurrentLine(newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			if e.Empty() {
				// Empty file, nothing to format, insert a program template, if available
				if err := e.InsertTemplateProgram(c); err != nil {
					status.ClearAll(c)
					status.SetMessage("nothing to format and no template available")
					status.Show(c, e)
				} else {
					e.redraw = true
					e.redrawCursor = true
				}
				break
			}

			if e.mode == mode.Markdown {
				e.ToggleCheckboxCurrentLine()
				break
			}

			status.ClearAll(c)
			e.formatCode(c, tty, status, &jsonFormatToggle)

			// Move the cursor if after the end of the line
			if e.AtOrAfterEndOfLine() {
				e.End(c)
			}

			// Keep the message on screen for 1 second, despite e.redraw being set.
			// This is only to have a minimum amount of display time for the message.
			status.HoldMessage(c, 250*time.Millisecond)

		case "c:6": // ctrl-f, search for a string

			// If in Debug mode, let ctrl-f mean "finish"
			if e.debugMode {
				if e.gdb == nil { // success
					status.SetMessageAfterRedraw("Not running")
					break
				}
				status.ClearAll(c)
				if err := e.DebugFinish(); err != nil {
					e.DebugEnd()
					status.SetMessage(err.Error())
					e.GoToEnd(c, nil)
				} else {
					status.SetMessage("Finish")
				}
				status.SetMessageAfterRedraw(status.Message())
				break
			}

			e.SearchMode(c, status, tty, true, undo)

		case "c:0": // ctrl-space, build source code to executable, or export, depending on the mode
			// Then build, but don't run
			const andRun = false
			e.Build(c, status, tty, andRun)

		case "c:20": // ctrl-t
			// for C or C++: jump to header/source, or insert symbol
			// for Agda: insert symbol
			// for the rest: record and play back macros
			// debug mode: next insTruction

			// Save the current file, but only if it has changed
			if e.changed {
				if err := e.Save(c, tty); err != nil {
					status.ClearAll(c)
					status.SetError(err)
					status.Show(c, e)
					break
				}
			}

			e.redrawCursor = true

			if (e.mode == mode.C || e.mode == mode.Cpp) && hasS([]string{".cpp", ".cc", ".c", ".cxx", ".c++"}, filepath.Ext(e.filename)) { // jump from source to header file
				// If this is a C++ source file, try finding and opening the corresponding header file
				// Check if there is a corresponding header file
				if absFilename, err := e.AbsFilename(); err == nil { // no error
					headerExtensions := []string{".h", ".hpp", ".h++"}
					if headerFilename, err := ExtFileSearch(absFilename, headerExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
						// Switch to another file (without forcing it)
						e.Switch(c, tty, status, fileLock, headerFilename)
						break
					}
				}
				status.ClearAll(c)
				status.SetErrorMessage("No corresponding header file")
				status.Show(c, e)
			} else if (e.mode == mode.C || e.mode == mode.Cpp) && hasS([]string{".h", ".hpp", ".h++"}, filepath.Ext(e.filename)) { // jump from header to source file
				// If this is a header file, present a menu option for open the corresponding source file
				// Check if there is a corresponding header file
				if absFilename, err := e.AbsFilename(); err == nil { // no error
					sourceExtensions := []string{".c", ".cpp", ".cxx", ".cc", ".c++"}
					if headerFilename, err := ExtFileSearch(absFilename, sourceExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
						// Switch to another file (without forcing it)
						e.Switch(c, tty, status, fileLock, headerFilename)
						break
					}
				}
				status.ClearAll(c)
				status.SetErrorMessage("No corresponding source file")
				status.Show(c, e)
			} else if e.mode == mode.Agda || e.mode == mode.Ivy { // insert symbol
				var (
					menuChoices    [][]string
					selectedSymbol string
				)
				if e.mode == mode.Agda {
					menuChoices = agdaSymbols
					selectedSymbol = "¤"
				} else if e.mode == mode.Ivy {
					menuChoices = ivySymbols
					selectedSymbol = "×"
				}
				e.redraw = true
				selectedX, selectedY, cancel := e.SymbolMenu(status, tty, "Insert symbol", menuChoices, e.MenuTitleColor, e.MenuTextColor, e.MenuArrowColor)
				if !cancel {
					undo.Snapshot(e)
					if selectedY < len(menuChoices) {
						row := menuChoices[selectedY]
						if selectedX < len(row) {
							selectedSymbol = menuChoices[selectedY][selectedX]
						}
					}
					e.InsertString(c, selectedSymbol)
				}
			} else if e.macro == nil {
				// Start recording a macro, then stop the recording when ctrl-t is pressed again,
				// then ask for the number of repetitions to play it back when it's pressed after that,
				// then clear the macro when esc is pressed.
				undo.Snapshot(e)
				undo.IgnoreSnapshots(true)
				status.Clear(c)
				status.SetMessage("Recording macro")
				status.Show(c, e)
				e.macro = NewMacro()
				e.macro.Recording = true
				e.playBackMacroCount = 0
			} else if e.macro.Recording { // && e.macro != nil
				e.macro.Recording = false
				undo.IgnoreSnapshots(true)
				e.playBackMacroCount = 0
				status.Clear(c)
				if macroLen := e.macro.Len(); macroLen == 0 {
					status.SetMessage("Stopped recording")
					e.macro = nil
				} else if macroLen < 10 {
					status.SetMessage("Recorded " + strings.Join(e.macro.KeyPresses, " "))
				} else {
					status.SetMessage(fmt.Sprintf("Recorded %d steps", macroLen))
				}
				status.Show(c, e)
			} else if e.playBackMacroCount > 0 {
				undo.IgnoreSnapshots(false)
				status.Clear(c)
				status.SetMessage("Stopped macro") // stop macro playback
				status.Show(c, e)
				e.playBackMacroCount = 0
				e.macro.Home()
			} else { // && e.macro != nil && e.playBackMacroCount == 0 // start macro playback
				undo.IgnoreSnapshots(false)
				undo.Snapshot(e)
				status.ClearAll(c)
				// Play back the macro, once
				e.playBackMacroCount = 1
			}
		case "c:28": // ctrl-\, toggle comment for this block
			undo.Snapshot(e)
			e.ToggleCommentBlock(c)
			e.redraw = true
			e.redrawCursor = true
		case "c:15": // ctrl-o, launch the command menu
			status.ClearAll(c)
			undo.Snapshot(e)
			undoBackup := undo
			lastCommandMenuIndex = e.CommandMenu(c, tty, status, bookmark, undo, lastCommandMenuIndex, forceFlag, fileLock)
			undo = undoBackup
		case "c:31": // ctrl-_, status mode
			status.ClearAll(c)
			e.statusMode = !e.statusMode
			if e.statusMode {
				status.ShowLineColWordCount(c, e, e.filename)
			}
		case "←": // left arrow

			// Don't move if ChatGPT is currently generating tokens that are being inserted
			if e.generatingTokens {
				break
			}

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith("←") {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			// movement if there is horizontal scrolling
			if e.pos.offsetX > 0 {
				if e.pos.sx > 0 {
					// Move one step left
					if e.TabToTheLeft() {
						e.pos.sx -= e.indentation.PerTab
					} else {
						e.pos.sx--
					}
				} else {
					// Scroll one step left
					e.pos.offsetX--
					e.redraw = true
				}
				e.SaveX(true)
			} else if e.pos.sx > 0 {
				// no horizontal scrolling going on
				// Move one step left
				if e.TabToTheLeft() {
					e.pos.sx -= e.indentation.PerTab
				} else {
					e.pos.sx--
				}
				e.SaveX(true)
			} else if e.DataY() > 0 {
				// no scrolling or movement to the left going on
				e.Up(c, status)
				e.End(c)
				// e.redraw = true
			} // else at the start of the document
			e.redrawCursor = true
			// Workaround for Konsole
			if e.pos.sx <= 2 {
				// Konsole prints "2H" here, but
				// no other terminal emulator does that
				e.redraw = true
			}
		case "→": // right arrow

			// Don't move if ChatGPT is currently generating tokens that are being inserted
			if e.generatingTokens {
				break
			}

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith("→") {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			// If on the last line or before, go to the next character
			if e.DataY() <= LineIndex(e.Len()) {
				e.Next(c)
			}
			if e.AfterScreenWidth(c) {
				e.pos.offsetX++
				e.redraw = true
				e.pos.sx--
				if e.pos.sx < 0 {
					e.pos.sx = 0
				}
				if e.AfterEndOfLine() {
					e.Down(c, status)
				}
			} else if e.AfterEndOfLine() {
				e.End(c)
			}
			e.SaveX(true)
			e.redrawCursor = true
		case "↑": // up arrow

			// Don't move if ChatGPT is currently generating tokens that are being inserted
			if e.generatingTokens {
				break
			}

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith("↑") {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			// TODO: Stay at the same X offset when moving up in the document?
			if e.pos.offsetX > 0 {
				e.pos.offsetX = 0
			}

			if e.DataY() > 0 {
				// Move the position up in the current screen
				if e.UpEnd(c) != nil {
					// If below the top, scroll the contents up
					if e.DataY() > 0 {
						e.redraw = e.ScrollUp(c, status, 1)
						e.pos.Down(c)
						e.UpEnd(c)
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineScreenContents() {
					e.End(c)
				}
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.AfterLineScreenContents() {
				e.End(c)

				// Then, if the rune to the left is '}', move one step to the left
				if r := e.LeftRune(); r == '}' {
					e.Prev(c)
				}
			}
			e.redrawCursor = true
		case "↓": // down arrow

			// Don't move if ChatGPT is currently generating tokens that are being inserted
			if e.generatingTokens {
				break
			}

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith("↓") {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			// TODO: Stay at the same X offset when moving down in the document?
			if e.pos.offsetX > 0 {
				e.pos.offsetX = 0
			}

			if e.DataY() < LineIndex(e.Len()) {
				// Move the position down in the current screen
				if e.DownEnd(c) != nil {
					// If at the bottom, don't move down, but scroll the contents
					// Output a helpful message
					if !e.AfterEndOfDocument() {
						e.redraw = e.ScrollDown(c, status, 1)
						e.pos.Up()
						e.DownEnd(c)
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineScreenContents() {
					e.End(c)

					// Then, if the rune to the left is '}', move one step to the left
					if r := e.LeftRune(); r == '}' {
						e.Prev(c)
					}
				}
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.AfterLineScreenContents() {
				e.End(c)
			}
			e.redrawCursor = true

		case "c:14": // ctrl-n, scroll down or jump to next match, using the sticky search term

			// First check if we can jump to the matching paren or bracket instead
			// also check that the last keypress was not ctrl-n, to make scrolling continous.
			if e.OnParenOrBracket() && (jumpMode || !kh.PrevIs("c:14")) {
				// Don't count successful jumps as ctrl-n scrolling
				clearKeyHistory = true
				if !kh.PrevIs("c:14") {
					jumpMode = false
				}
				if e.JumpToMatching(c) {
					jumpMode = true
					break
				}
			}

			// If in Debug mode, let ctrl-n mean "next instruction"
			if e.debugMode {
				if e.gdb != nil {
					if !programRunning {
						e.DebugEnd()
						status.SetMessage("Program stopped")
						status.SetMessageAfterRedraw(status.Message())
						e.redraw = true
						e.redrawCursor = true
						break
					}
					if err := e.DebugNextInstruction(); err != nil {
						if errorMessage := err.Error(); strings.Contains(errorMessage, "is not being run") {
							e.DebugEnd()
							status.SetMessage("Could not start GDB")
						} else if err == errProgramStopped {
							e.DebugEnd()
							status.SetMessage("Program stopped, could not step")
						} else { // got an unrecognized error
							e.DebugEnd()
							status.SetMessage(errorMessage)
						}
					} else {
						if !programRunning {
							e.DebugEnd()
							status.SetMessage("Program stopped when stepping") // Next instruction
						} else {
							// Don't show a status message per instruction/step when pressing ctrl-n
							break
						}
					}
					e.redrawCursor = true
					status.SetMessageAfterRedraw(status.Message())
					break
				} // e.gdb == nil
				// Build or export the current file
				// The last argument is if the command should run in the background or not
				outputExecutable, err := e.BuildOrExport(c, tty, status, e.filename, e.mode == mode.Markdown)
				// All clear when it comes to status messages and redrawing
				status.ClearAll(c)
				if err != nil && err != errNoSuitableBuildCommand {
					// Error while building
					status.SetError(err)
					status.ShowNoTimeout(c, e)
					e.debugMode = false
					e.redrawCursor = true
					e.redraw = true
					break
				}
				// Was no suitable compilation or export command found?
				if err == errNoSuitableBuildCommand {
					// status.ClearAll(c)
					if e.debugMode {
						// Both in debug mode and can not find a command to build this file with.
						status.SetError(err)
						status.ShowNoTimeout(c, e)
						e.debugMode = false
						e.redrawCursor = true
						e.redraw = true
						break
					}
					// Building this file extension is not implemented yet.
					// Just display the current time and word count.
					// TODO: status.ClearAll() should have cleared the status bar first, but this is not always true,
					//       which is why the message is hackily surrounded by spaces. Fix.
					statsMessage := fmt.Sprintf("    %d words, %s    ", e.WordCount(), time.Now().Format("15:04")) // HH:MM
					status.SetMessage(statsMessage)
					status.Show(c, e)
					e.redrawCursor = true
					break
				}
				// Start debugging
				if err := e.DebugStartSession(c, tty, status, outputExecutable); err != nil {
					status.ClearAll(c)
					status.SetError(err)
					status.ShowNoTimeout(c, e)
					e.redrawCursor = true
				}
				break
			}

			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to next match
				wrap := true
				forward := true
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					msg := e.SearchTerm() + " not found"
					if wrap {
						status.SetMessage(msg)
					} else {
						status.SetMessage(msg + " from here")
					}
					status.Show(c, e)
				}
			} else {

				// Scroll down
				e.redraw = e.ScrollDown(c, status, e.pos.scrollSpeed)
				// If e.redraw is false, the end of file is reached
				if !e.redraw {
					status.Clear(c)
					status.SetMessage("EOF")
					status.Show(c, e)
				}
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}

			}
		case "c:16": // ctrl-p, scroll up or jump to the previous match, using the sticky search term. In debug mode, change the pane layout.

			// First check if we can jump to the matching paren or bracket instead
			// also check that the last keypress was not ctrl-n, to make scrolling continous.
			if e.OnParenOrBracket() && (jumpMode || !kh.PrevIs("c:16")) {
				// Don't count successful jumps as ctrl-p scrolling
				clearKeyHistory = true
				if !kh.PrevIs("c:16") {
					jumpMode = false
				}
				if e.JumpToMatching(c) {
					jumpMode = true
					break
				}
			}

			if e.debugMode {
				// e.showRegisters has three states, 0 (SmallRegisterWindow), 1 (LargeRegisterWindow) and 2 (NoRegisterWindow)
				e.debugShowRegisters++
				if e.debugShowRegisters > noRegisterWindow {
					e.debugShowRegisters = smallRegisterWindow
				}
				break
			}

			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to previous match
				wrap := true
				forward := false
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					msg := e.SearchTerm() + " not found"
					if wrap {
						status.SetMessage(msg)
					} else {
						status.SetMessage(msg + " from here")
					}
					status.Show(c, e)
				}
			} else {
				e.redraw = e.ScrollUp(c, status, e.pos.scrollSpeed)
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}

			}
			// Additional way to clear the sticky search term, like with Esc
		case "c:27": // esc, clear search term (but not the sticky search term), reset, clean and redraw
			// If o is used as a man page viewer, exit at the press of esc
			if e.mode == mode.ManPage {
				e.clearOnQuit = false
				e.quit = true
				break
			}
			// Exit debug mode, if active
			if e.debugMode {
				e.DebugEnd()
				e.debugMode = false
				status.SetMessageAfterRedraw("Normal mode")
				break
			}
			// Stop the call to ChatGPT, if it is running
			e.generatingTokens = false
			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1
			// Do a full clear and redraw + clear search term + jump
			drawLines := true
			e.FullResetRedraw(c, status, drawLines)
			if e.macro != nil || e.playBackMacroCount > 0 {
				// Stop the playback
				e.playBackMacroCount = 0
				// Clear the macro
				e.macro = nil
				// Show a message after the redraw
				status.SetMessageAfterRedraw("Macro cleared")
				break
			}
			e.redraw = true
			e.redrawCursor = true
		case " ": // space

			// Scroll down if a man page is being viewed, or if the editor is read-only
			if e.readOnly {
				// Scroll down at double scroll speed
				e.redraw = e.ScrollDown(c, status, e.pos.scrollSpeed*2)
				// If e.redraw is false, the end of file is reached
				if !e.redraw {
					status.Clear(c)
					status.SetMessage("EOF")
					status.Show(c, e)
				}
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// Regular behavior, take an undo snapshot and insert a space
			undo.Snapshot(e)
			// Place a space
			wrapped := e.InsertRune(c, ' ')
			if !wrapped {
				e.WriteRune(c)
				// Move to the next position
				e.Next(c)
			}
			e.redraw = true
		case "c:13": // return

			// Scroll down if a man page is being viewed, or if the editor is read-only
			if e.readOnly {
				// Scroll down at double scroll speed
				e.redraw = e.ScrollDown(c, status, e.pos.scrollSpeed*2)
				// If e.redraw is false, the end of file is reached
				if !e.redraw {
					status.Clear(c)
					status.SetMessage("EOF")
					status.Show(c, e)
				}
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// Regular behavior

			// Modify the paste double-keypress detection to allow for a manual return before pasting the rest
			if lastPasteY != -1 && kh.Prev() != "c:13" {
				lastPasteY++
			}

			undo.Snapshot(e)

			var (
				lineContents             = e.CurrentLine()
				trimmedLine              = strings.TrimSpace(lineContents)
				currentLeadingWhitespace = e.LeadingWhitespace()

				// Grab the leading whitespace from the current line, and indent depending on the end of trimmedLine
				leadingWhitespace = e.smartIndentation(currentLeadingWhitespace, trimmedLine, false) // the last parameter is "also dedent"

				noHome = false
				indent = true
			)

			// TODO: add and use something like "e.shouldAutoIndent" for these file types
			if e.mode == mode.Markdown || e.mode == mode.Text || e.mode == mode.Blank {
				indent = false
			}

			triggerWordsForAI := []string{"Generate", "generate", "Write", "write", "!"}
			shouldUseAI := false

			if e.AtOrAfterEndOfLine() && e.NextLineIsBlank() {
				for _, triggerWord := range triggerWordsForAI {
					if e.mode == mode.Markdown && triggerWord == "!" {
						continue
					}
					if strings.HasPrefix(trimmedLine, e.SingleLineCommentMarker()+" "+triggerWord+" ") {
						shouldUseAI = true
						break
					} else if strings.HasPrefix(trimmedLine, e.SingleLineCommentMarker()+triggerWord+" ") {
						shouldUseAI = true
						break
					} else if e.mode != mode.Markdown && e.SingleLineCommentMarker() != "!" && strings.HasPrefix(trimmedLine, "!") {
						shouldUseAI = true
						break
					}
				}
			}
			alreadyUsedAI := false
		RETURN_PRESSED_AI_DONE:

			if trimmedLine == "private:" || trimmedLine == "protected:" || trimmedLine == "public:" {
				// De-indent the current line before moving on to the next
				e.SetCurrentLine(trimmedLine)
				leadingWhitespace = currentLeadingWhitespace
			} else if e.fixAsYouType && openAIKey != "" && !alreadyUsedAI {
				// Fix the code and grammar of the written line, using AI
				e.FixCodeOrText(c, status)
				alreadyUsedAI = true
				goto RETURN_PRESSED_AI_DONE
			} else if shouldUseAI && openAIKey != "" {
				// Generate code or text, using AI
				e.GenerateCodeOrText(c, status, bookmark)
				break
			} else if e.mode == mode.Arduino || e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.Shader || e.mode == mode.Zig || e.mode == mode.Java || e.mode == mode.JavaScript || e.mode == mode.Kotlin || e.mode == mode.TypeScript || e.mode == mode.D || e.mode == mode.Hare || e.mode == mode.Jakt {
				// Add missing parenthesis for "if ... {", "} else if", "} elif", "for", "while" and "when" for C-like languages
				for _, kw := range []string{"for", "foreach", "foreach_reverse", "if", "switch", "when", "while", "while let", "} else if", "} elif"} {
					if strings.HasPrefix(trimmedLine, kw+" ") && !strings.HasPrefix(trimmedLine, kw+" (") {
						if strings.HasSuffix(trimmedLine, " {") {
							// Add ( and ), keep the final "{"
							e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[len(kw)+1:len(trimmedLine)-2] + ") {")
							e.pos.sx += 2
						} else if !strings.HasSuffix(trimmedLine, ")") {
							// Add ( and ), there is no final "{"
							e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[len(kw)+1:] + ")")
							e.pos.sx += 2
							indent = true
							leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
						}
					}
				}
			} else if (e.mode == mode.Go || e.mode == mode.Odin) && trimmedLine == "iferr" {
				oneIndentation := e.indentation.String()
				// default "if err != nil" block if iferr.IfErr can not find a more suitable one
				ifErrBlock := "if err != nil {\n" + oneIndentation + "return nil, err\n" + "}\n"
				// search backwards for "func ", return the full contents, the resulting line index and if it was found
				contents, functionLineIndex, found := e.ContentsAndReverseSearchPrefix("func ")
				if found {
					// count the bytes from the start to the end of the "func " line, since this is what iferr.IfErr uses
					byteCount := 0
					for i := LineIndex(0); i <= functionLineIndex; i++ {
						byteCount += len(e.Line(i))
					}
					// fetch a suitable "if err != nil" block for the current function signature
					if generatedIfErrBlock, err := iferr.IfErr([]byte(contents), byteCount); err != nil {
						logf("could not generate iferrblock: %s\n", err)
					} else {
						ifErrBlock = generatedIfErrBlock
					}
				}
				// insert the block of text
				for i, line := range strings.Split(strings.TrimSpace(ifErrBlock), "\n") {
					if i != 0 {
						e.InsertLineBelow()
						e.pos.sy++
					}
					e.SetCurrentLine(currentLeadingWhitespace + line)
				}
				e.End(c)
			} else if (e.mode == mode.XML || e.mode == mode.HTML) && !e.noExpandTags && trimmedLine != "" && !strings.Contains(trimmedLine, "<") && !strings.Contains(trimmedLine, ">") && strings.ToLower(string(trimmedLine[0])) == string(trimmedLine[0]) {
				// Words one a line without < or >? Expand into <tag asdf> above and </tag> below.
				words := strings.Fields(trimmedLine)
				tagName := words[0] // must be at least one word
				// the second word after the tag name needs to be ie. x=42 or href=...,
				// and the tag name must only contain letters a-z A-Z
				if (len(words) == 1 || strings.Contains(words[1], "=")) && onlyAZaz(tagName) {
					above := "<" + trimmedLine + ">"
					if tagName == "img" && !strings.Contains(trimmedLine, "alt=") && strings.Contains(trimmedLine, "src=") {
						// Pick out the image URI from the "src=" declaration
						imageURI := ""
						for _, word := range strings.Fields(trimmedLine) {
							if strings.HasPrefix(word, "src=") {
								imageURI = strings.SplitN(word, "=", 2)[1]
								imageURI = strings.TrimPrefix(imageURI, "\"")
								imageURI = strings.TrimSuffix(imageURI, "\"")
								imageURI = strings.TrimPrefix(imageURI, "'")
								imageURI = strings.TrimSuffix(imageURI, "'")
								break
							}
						}
						// If we got something that looks like and image URI, use the description before "." and capitalize it,
						// then use that as the default "alt=" declaration.
						if strings.Contains(imageURI, ".") {
							imageName := capitalizeWords(strings.TrimSuffix(imageURI, filepath.Ext(imageURI)))
							above = "<" + trimmedLine + " alt=\"" + imageName + "\">"
						}
					}
					// Now replace the current line
					e.SetCurrentLine(currentLeadingWhitespace + above)
					e.End(c)
					// And insert a line below
					e.InsertLineBelow()
					// Then if it's not an img tag, insert the closing tag below the current line
					if tagName != "img" {
						e.pos.sy++
						below := "</" + tagName + ">"
						e.SetCurrentLine(currentLeadingWhitespace + below)
						e.pos.sy--
						e.pos.sx += 2
						indent = true
						leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
					}
				}
			}

			// onlyOneLine := e.AtFirstLineOfDocument() && e.AtOrAfterLastLineOfDocument()
			// middleOfText := !e.AtOrBeforeStartOfTextLine() && !e.AtOrAfterEndOfLine()

			scrollBack := false

			// TODO: Collect the criteria that trigger the same behavior

			switch {
			case e.AtOrAfterLastLineOfDocument() && (e.AtStartOfTheLine() || e.AtOrBeforeStartOfTextScreenLine()):
				e.InsertLineAbove()
				noHome = true
			case e.AtOrAfterEndOfDocument() && !e.AtStartOfTheLine() && !e.AtOrAfterEndOfLine():
				e.InsertStringAndMove(c, "")
				e.InsertLineBelow()
				scrollBack = true
			case e.AfterEndOfLine():
				e.InsertLineBelow()
				scrollBack = true
			case !e.AtFirstLineOfDocument() && e.AtOrAfterLastLineOfDocument() && (e.AtStartOfTheLine() || e.AtOrAfterEndOfLine()):
				e.InsertStringAndMove(c, "")
				scrollBack = true
			case e.AtStartOfTheLine():
				e.InsertLineAbove()
				noHome = true
			default:
				// Split the current line in two
				if !e.SplitLine() {
					e.InsertLineBelow()
				}
				scrollBack = true
				// Indent the next line if at the end, not else
				if !e.AfterEndOfLine() {
					indent = false
				}
			}
			e.MakeConsistent()

			h := int(c.Height())
			if e.pos.sy > (h - 1) {
				e.pos.Down(c)
				e.redraw = e.ScrollDown(c, status, 1)
				e.redrawCursor = true
			} else if e.pos.sy == (h - 1) {
				e.redraw = e.ScrollDown(c, status, 1)
				e.redrawCursor = true
			} else {
				e.pos.Down(c)
			}

			if !noHome {
				e.pos.sx = 0
				// e.Home()
				if scrollBack {
					e.pos.SetX(c, 0)
				}
			}

			if indent && len(leadingWhitespace) > 0 {
				// If the leading whitespace starts with a tab and ends with a space, remove the final space
				if strings.HasPrefix(leadingWhitespace, "\t") && strings.HasSuffix(leadingWhitespace, " ") {
					leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
					// logf("cleaned leading whitespace: %v\n", []rune(leadingWhitespace))
				}
				if !noHome {
					// Insert the same leading whitespace for the new line
					e.SetCurrentLine(leadingWhitespace + e.LineContentsFromCursorPosition())
					// Then move to the start of the text
					e.GoToStartOfTextLine(c)
				}
			}

			e.SaveX(true)
			e.redraw = true
			e.redrawCursor = true
		case "c:8", "c:127": // ctrl-h or backspace

			// Scroll up if a man page is being viewed, or if the editor is read-only
			if e.readOnly {
				// Scroll up at double speed
				e.redraw = e.ScrollUp(c, status, e.pos.scrollSpeed*2)
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// Just clear the search term, if there is an active search
			if len(e.SearchTerm()) > 0 {
				e.ClearSearchTerm()
				e.redraw = true
				e.redrawCursor = true
				// Don't break, continue to delete to the left after clearing the search,
				// since Esc can be used to only clear the search.
				// break
			}

			undo.Snapshot(e)
			// Delete the character to the left
			if e.EmptyLine() {
				e.DeleteCurrentLineMoveBookmark(bookmark)
				e.pos.Up()
				e.TrimRight(e.DataY())
				e.End(c)
			} else if e.AtStartOfTheLine() { // at the start of the screen line, the line may be scrolled
				// remove the rest of the current line and move to the last letter of the line above
				// before deleting it
				if e.DataY() > 0 {
					e.pos.Up()
					e.TrimRight(e.DataY())
					e.End(c)
					e.Delete()
				}
			} else if e.indentation.Spaces && (e.EmptyLine() || e.AtStartOfTheLine()) && e.indentation.WSLen(e.LeadingWhitespace()) >= e.indentation.PerTab {
				// Delete several spaces
				for i := 0; i < e.indentation.PerTab; i++ {
					// Move back
					e.Prev(c)
					// Type a blank
					e.SetRune(' ')
					e.WriteRune(c)
					e.Delete()
				}
			} else {
				// Move back
				e.Prev(c)
				// Type a blank
				e.SetRune(' ')
				e.WriteRune(c)
				if !e.AtOrAfterEndOfLine() {
					// Delete the blank
					e.Delete()
				}
			}
			e.redrawCursor = true
			e.redraw = true
		case "c:9": // tab or ctrl-i

			if e.debugMode {
				e.debugStepInto = !e.debugStepInto
				break
			}

			y := int(e.DataY())
			r := e.Rune()
			leftRune := e.LeftRune()
			ext := filepath.Ext(e.filename)

			// Tab completion of words for Go
			if word := e.LettersBeforeCursor(); e.mode != mode.Blank && e.mode != mode.GoAssembly && e.mode != mode.Assembly && leftRune != '.' && !unicode.IsLetter(r) && len(word) > 0 {
				found := false
				expandedWord := ""
				for kw := range syntax.Keywords {
					if len(kw) < 3 {
						// skip too short suggestions
						continue
					}
					if strings.HasPrefix(kw, word) {
						if !found || (len(kw) < len(expandedWord)) && (len(expandedWord) > 0) {
							expandedWord = kw
							found = true
						}
					}
				}

				// Found a suitable keyword to expand to? Insert the rest of the string.
				if found {
					toInsert := strings.TrimPrefix(expandedWord, word)
					undo.Snapshot(e)
					e.redrawCursor = true
					e.redraw = true
					// Insert the part of expandedWord that comes after the current word
					e.InsertStringAndMove(c, toInsert)
					break
				}

				// Tab completion after a '.'
			} else if word := e.LettersOrDotBeforeCursor(); e.mode != mode.Blank && e.mode != mode.GoAssembly && e.mode != mode.Assembly && leftRune == '.' && !unicode.IsLetter(r) && len(word) > 0 {
				// Now the preceding word before the "." has been found

				// Trim the trailing ".", if needed
				word = strings.TrimSuffix(strings.TrimSpace(word), ".")

				// Grep all files in this directory with the same extension as the currently edited file
				// for what could follow the word and a "."
				suggestions := corpus(word, "*"+ext)

				// Choose a suggestion (tab cycles to the next suggestion)
				chosen := e.SuggestMode(c, status, tty, suggestions)
				e.redrawCursor = true
				e.redraw = true

				if chosen != "" {
					undo.Snapshot(e)
					// Insert the chosen word
					e.InsertStringAndMove(c, chosen)
					break
				}

			}

			// Enable auto indent if the extension is not "" and either:
			// * The mode is set to Go and the position is not at the very start of the line (empty or not)
			// * Syntax highlighting is enabled and the cursor is not at the start of the line (or before)
			trimmedLine := e.TrimmedLine()

			// Check if a line that is more than just a '{', '(', '[' or ':' ends with one of those
			endsWithSpecial := len(trimmedLine) > 1 && r == '{' || r == '(' || r == '[' || r == ':'

			// Smart indent if:
			// * the rune to the left is not a blank character or the line ends with {, (, [ or :
			// * and also if it the cursor is not to the very left
			// * and also if this is not a text file or a blank file
			noSmartIndentation := e.mode == mode.GoAssembly || e.mode == mode.Perl || e.mode == mode.Assembly || e.mode == mode.OCaml || e.mode == mode.StandardML || e.mode == mode.Blank
			if (!unicode.IsSpace(leftRune) || endsWithSpecial) && e.pos.sx > 0 && !noSmartIndentation {
				lineAbove := 1
				if strings.TrimSpace(e.Line(LineIndex(y-lineAbove))) == "" {
					// The line above is empty, use the indentation before the line above that
					lineAbove--
				}
				indexAbove := LineIndex(y - lineAbove)
				// If we have a line (one or two lines above) as a reference point for the indentation
				if strings.TrimSpace(e.Line(indexAbove)) != "" {

					// Move the current indentation to the same as the line above
					undo.Snapshot(e)

					var (
						spaceAbove        = e.LeadingWhitespaceAt(indexAbove)
						strippedLineAbove = e.StripSingleLineComment(strings.TrimSpace(e.Line(indexAbove)))
						newLeadingSpace   string
					)

					oneIndentation := e.indentation.String()

					// Smart-ish indentation
					if !strings.HasPrefix(strippedLineAbove, "switch ") && (strings.HasPrefix(strippedLineAbove, "case ")) ||
						strings.HasSuffix(strippedLineAbove, "{") || strings.HasSuffix(strippedLineAbove, "[") ||
						strings.HasSuffix(strippedLineAbove, "(") || strings.HasSuffix(strippedLineAbove, ":") ||
						strings.HasSuffix(strippedLineAbove, " \\") ||
						strings.HasPrefix(strippedLineAbove, "if ") {
						// Use one more indentation than the line above
						newLeadingSpace = spaceAbove + oneIndentation
					} else if ((len(spaceAbove) - len(oneIndentation)) > 0) && strings.HasSuffix(trimmedLine, "}") {
						// Use one less indentation than the line above
						newLeadingSpace = spaceAbove[:len(spaceAbove)-len(oneIndentation)]
					} else {
						// Use the same indentation as the line above
						newLeadingSpace = spaceAbove
					}

					e.SetCurrentLine(newLeadingSpace + trimmedLine)
					if e.AtOrAfterEndOfLine() {
						e.End(c)
					}
					e.redrawCursor = true
					e.redraw = true

					// job done
					break

				}
			}

			undo.Snapshot(e)
			if e.indentation.Spaces {
				for i := 0; i < e.indentation.PerTab; i++ {
					e.InsertRune(c, ' ')
					// Write the spaces that represent the tab to the canvas
					e.WriteTab(c)
					// Move to the next position
					e.Next(c)
				}
			} else {
				// Insert a tab character to the file
				e.InsertRune(c, '\t')
				// Write the spaces that represent the tab to the canvas
				e.WriteTab(c)
				// Move to the next position
				e.Next(c)
			}

			// Prepare to redraw
			e.redrawCursor = true
			e.redraw = true
		case "c:1", "c:25": // ctrl-a, home (or ctrl-y for scrolling up in the st terminal)
			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys
			justMovedUpOrDown := kh.PrevIs("↓") || kh.PrevIs("↑")
			// If at an empty line, go up one line
			if !justMovedUpOrDown && e.EmptyRightTrimmedLine() && e.SearchTerm() == "" {
				e.Up(c, status)
				// e.GoToStartOfTextLine()
				e.End(c)
			} else if x, err := e.DataX(); err == nil && x == 0 && !justMovedUpOrDown && e.SearchTerm() == "" {
				// If at the start of the line,
				// go to the end of the previous line
				e.Up(c, status)
				e.End(c)
			} else if e.AtStartOfTextScreenLine() {
				// If at the start of the text for this scroll position, go to the start of the line
				e.Home()
			} else {
				// If none of the above, go to the start of the text
				e.GoToStartOfTextLine(c)
			}

			e.redrawCursor = true
			e.SaveX(true)
		case "c:5": // ctrl-e, end

			// First check if we are editing Markdown and are in a Markdown table
			if e.mode == mode.Markdown {
				if strings.Count(e.CurrentLine(), "|") >= 2 {
					e.GoToTableTop(c, status)
					//e.EditMarkdownTable()
					break
				}
			}

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys, or just cut a line with ctrl-x
			justMovedUpOrDown := kh.PrevIs("↓") || kh.PrevIs("↑") || kh.PrevIs("c:24")
			if e.AtEndOfDocument() {
				e.End(c)
				break
			}
			// If we didn't just move here, and are at the end of the line,
			// move down one line and to the end, if not,
			// just move to the end.
			if !justMovedUpOrDown && e.AfterEndOfLine() && e.SearchTerm() == "" {
				e.Down(c, status)
				e.Home()
			} else {
				e.End(c)
			}

			e.redrawCursor = true
			e.SaveX(true)
		case "c:4": // ctrl-d, delete
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.Delete()
				e.redraw = true
			}
			e.redrawCursor = true

		case "c:29", "c:30": // ctrl-~, jump to matching parenthesis or curly bracket
			if e.JumpToMatching(c) {
				break
			}
			status.Clear(c)
			status.SetMessage("No matching (, ), [, ], { or }")
			status.Show(c, e)
		case "c:19": // ctrl-s, save (or step, if in debug mode)
			e.UserSave(c, tty, status)
		case "c:7": // ctrl-g, display some help (3 times), then either go to definition OR toggle the status bar

			// If a search is in progress, clear the search
			if e.searchTerm != "" {
				e.ClearSearchTerm()
				e.redraw = true
				e.redrawCursor = true

				// TODO: Also jump to where the cursor was before "go to definition"?

				break
			}

			// TODO: Make this block of code less if-else-y, and fewer levels deep

			canGoToDefinition := e.FuncPrefix() != ""
			if canGoToDefinition {
				if !e.GoToDefinition(c, status) {
					if helpCounter < maxHelpMessages {
						helpCounter++
						e.HelpMessage(c, status)
					} else {
						// If the definition cold not be found,
						// or if max help messages has been reached:
						// toggle the status line at the bottom.
						status.ClearAll(c)
						e.statusMode = !e.statusMode
						if e.statusMode {
							status.ShowLineColWordCount(c, e, e.filename)
						}
					}
				} else {
					// Don't show the help message any more after a successful jump to definition
					helpCounter += maxHelpMessages
				}
			} else {
				if helpCounter < maxHelpMessages {
					helpCounter++
					e.HelpMessage(c, status)
				} else {
					// Toggle the status line at the bottom
					status.ClearAll(c)
					e.statusMode = !e.statusMode
					if e.statusMode {
						status.ShowLineColWordCount(c, e, e.filename)
					}
				}
			}
		case "c:21", "c:26": // ctrl-u or ctrl-z (ctrl-z may background the application)
			// Forget the cut, copy and paste line state
			lastCutY = -1
			lastPasteY = -1
			lastCopyY = -1

			// Try to restore the previous editor state in the undo buffer
			if err := undo.Restore(e); err == nil {
				// c.Draw()
				x := e.pos.ScreenX()
				y := e.pos.ScreenY()
				vt100.SetXY(uint(x), uint(y))
				e.redrawCursor = true
				e.redraw = true
			} else {
				status.SetMessage("Nothing more to undo")
				status.Show(c, e)
			}
		case "c:12": // ctrl-l, go to line number or percentage
			status.ClearAll(c)
			status.SetMessage("Go to line number or percentage:")
			status.ShowNoTimeout(c, e)
			lns := ""
			cancel := false
			doneCollectingDigits := false
			goToEnd := false
			goToTop := false
			goToCenter := false
			for !doneCollectingDigits {
				numkey := tty.String()
				switch numkey {
				case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "%", ".", ",": // 0..9 + %,.
					lns += numkey // string('0' + (numkey - 48))
					status.SetMessage("Go to line number or percentage: " + lns)
					status.ShowNoTimeout(c, e)
				case "c:8", "c:127": // ctrl-h or backspace
					if len(lns) > 0 {
						lns = lns[:len(lns)-1]
						status.SetMessage("Go to line number or percentage: " + lns)
						status.ShowNoTimeout(c, e)
					}
				case "b", "t": // top of file
					doneCollectingDigits = true
					goToTop = true
				case "e": // end of file
					doneCollectingDigits = true
					goToEnd = true
				case "c", "m": // center of file
					doneCollectingDigits = true
					goToCenter = true
				case "↑", "↓": // up arrow or down arrow
					fallthrough // cancel
				case "c:27", "c:17": // esc or ctrl-q
					cancel = true
					lns = ""
					fallthrough // done
				case "c:13": // return
					doneCollectingDigits = true
				}
			}
			if !cancel {
				e.ClearSearchTerm()
			}
			status.ClearAll(c)
			if goToTop {
				e.GoToTop(c, status)
			} else if goToCenter {
				// Go to the center line
				e.GoToMiddle(c, status)
			} else if goToEnd {
				e.GoToEnd(c, status)
			} else if lns == "" && !cancel {
				if e.DataY() > 0 {
					// If not already at the top, go there
					e.GoToTop(c, status)
				} else {
					// Go to the last line
					e.GoToEnd(c, status)
				}
			} else if strings.HasSuffix(lns, "%") {
				// Go to the specified percentage
				if percentageInt, err := strconv.Atoi(lns[:len(lns)-1]); err == nil { // no error {
					lineIndex := int(math.Round(float64(e.Len()) * float64(percentageInt) * 0.01))
					e.redraw = e.GoToLineNumber(LineNumber(lineIndex), c, status, true)
				}
			} else if strings.Count(lns, ".") == 1 || strings.Count(lns, ",") == 1 {
				if percentageFloat, err := strconv.ParseFloat(strings.ReplaceAll(lns, ",", "."), 64); err == nil { // no error
					lineIndex := int(math.Round(float64(e.Len()) * percentageFloat))
					e.redraw = e.GoToLineNumber(LineNumber(lineIndex), c, status, true)
				}
			} else {
				// Go to the specified line
				if ln, err := strconv.Atoi(lns); err == nil { // no error
					e.redraw = e.GoToLineNumber(LineNumber(ln), c, status, true)
				}
			}
			e.redrawCursor = true
		case "c:24": // ctrl-x, cut line
			y := e.DataY()
			line := e.Line(y)
			// Prepare to cut
			undo.Snapshot(e)
			// Now check if there is anything to cut
			if len(strings.TrimSpace(line)) == 0 {
				// Nothing to cut, just remove the current line
				e.Home()
				e.DeleteCurrentLineMoveBookmark(bookmark)
				// Check if ctrl-x was pressed once or twice, for this line
			} else if lastCutY != y { // Single line cut
				// Also close the portal, if any
				e.ClosePortal()

				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1
				// Copy the line internally
				copyLines = []string{line}

				var err error
				if runtime.GOOS == "darwin" {
					// Copy the line to the clipboard
					err = pbcopy(line)
				} else {
					// Copy the line to the clipboard
					err = clipboard.WriteAll(line)
				}
				if err != nil && firstCopyAction {
					if env.Has("WAYLAND_DISPLAY") && which("wl-copy") == "" { // Wayland
						status.SetErrorMessage("The wl-copy utility (from wl-clipboard) is missing!")
					} else if env.Has("DISPLAY") && which("xclip") == "" {
						status.SetErrorMessage("The xclip utility is missing!")
					} else if runtime.GOOS == "darwin" && which("pbcopy") == "" { // pbcopy is missing, on macOS
						status.SetErrorMessage("The pbcopy utility is missing!")
					}
				}

				// Delete the line
				e.DeleteLineMoveBookmark(y, bookmark)
			} else { // Multi line cut (add to the clipboard, since it's the second press)
				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1

				// Also close the portal, if any
				e.ClosePortal()

				s := e.Block(y)
				lines := strings.Split(s, "\n")
				if len(lines) == 0 {
					// Need at least 1 line to be able to cut "the rest" after the first line has been cut
					break
				}
				copyLines = append(copyLines, lines...)
				s = strings.Join(copyLines, "\n")

				// Place the block of text in the clipboard
				if runtime.GOOS == "darwin" {
					pbcopy(s)
				} else {
					_ = clipboard.WriteAll(s)
				}

				// Delete the corresponding number of lines
				for range lines {
					e.DeleteLineMoveBookmark(y, bookmark)
				}
			}
			// Go to the end of the current line
			e.End(c)
			// No status message is needed for the cut operation, because it's visible that lines are cut
			e.redrawCursor = true
			e.redraw = true
		case "c:11": // ctrl-k, delete to end of line
			if e.Empty() {
				status.SetMessage("Empty file")
				status.Show(c, e)
				break
			}

			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1

			undo.Snapshot(e)

			e.DeleteRestOfLine()
			if e.EmptyRightTrimmedLine() {
				// Deleting the rest of the line cleared this line,
				// so just remove it.
				e.DeleteCurrentLineMoveBookmark(bookmark)
				// Then go to the end of the line, if needed
				if e.AfterEndOfLine() {
					e.End(c)
				}
			}

			// TODO: Is this one needed/useful?
			vt100.Do("Erase End of Line")

			e.redraw = true
			e.redrawCursor = true
		case "c:3": // ctrl-c, copy the stripped contents of the current line

			// ctrl-c might interrupt the program, but saving at the wrong time might be just as destructive.
			// e.Save(c, tty)

			y := e.DataY()

			// Forget the cut and paste line state
			lastCutY = -1
			lastPasteY = -1

			// check if this operation is done on the same line as last time
			singleLineCopy := lastCopyY != y
			lastCopyY = y

			// close the portal, if any
			closedPortal := e.ClosePortal() == nil

			if singleLineCopy { // Single line copy
				status.Clear(c)
				// Pressed for the first time for this line number
				trimmed := strings.TrimSpace(e.Line(y))
				if trimmed != "" {
					// Copy the line to the internal clipboard
					copyLines = []string{trimmed}
					// Copy the line to the clipboard
					s := "Copied 1 line"
					var err error
					if runtime.GOOS == "darwin" {
						err = pbcopy(strings.Join(copyLines, "\n"))
					} else {
						err = clipboard.WriteAll(strings.Join(copyLines, "\n"))
					}
					if err == nil { // OK
						// The copy operation worked out, using the clipboard
						s += " to the clipboard"
					}
					// The portal was closed?
					if closedPortal {
						s += " and closed the portal"
					}
					status.SetMessage(s)
					status.Show(c, e)
					// Go to the end of the line, for easy line duplication with ctrl-c, enter, ctrl-v,
					// but only if the copied line is shorter than the terminal width.
					if uint(len(trimmed)) < c.Width() {
						e.End(c)
					}
				}
			} else { // Multi line copy
				// Pressed multiple times for this line number, copy the block of text starting from this line
				s := e.Block(y)
				if s != "" {
					copyLines = strings.Split(s, "\n")
					// Prepare a status message
					plural := ""
					lineCount := strings.Count(s, "\n")
					if lineCount > 1 {
						plural = "s"
					}
					// Place the block of text in the clipboard
					if runtime.GOOS == "darwin" {
						err = pbcopy(s)
					} else {
						err = clipboard.WriteAll(s)
					}
					if err != nil {
						status.SetMessage(fmt.Sprintf("Copied %d line%s", lineCount, plural))
					} else {
						status.SetMessage(fmt.Sprintf("Copied %d line%s (clipboard)", lineCount, plural))
					}
					status.Show(c, e)
				}
			}
		case "c:22": // ctrl-v, paste
			if portal, err := LoadPortal(); err == nil { // no error
				var gotLineFromPortal bool
				line, err := portal.PopLine(e, false) // pop the line, but don't remove it from the source file
				status.Clear(c)
				if err != nil {
					// status.SetErrorMessage("Could not copy text through the portal.")
					status.SetError(err)
					e.ClosePortal()
				} else {
					status.SetMessage(fmt.Sprintf("Using portal at %s\n", portal))
					gotLineFromPortal = true
				}
				status.Show(c, e)

				if gotLineFromPortal {

					undo.Snapshot(e)

					if e.EmptyRightTrimmedLine() {
						// If the line is empty, replace with the string from the portal
						e.SetCurrentLine(line)
					} else {
						// If the line is not empty, insert the trimmed string
						e.InsertStringAndMove(c, strings.TrimSpace(line))
					}

					e.InsertLineBelow()
					e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line

					e.redraw = true

					break
				} // errors with loading a portal are ignored
			}

			// This may only work for the same user, and not with sudo/su

			// Try fetching the lines from the clipboard first
			var s string

			var err error
			if runtime.GOOS == "darwin" {
				s, err = pbpaste()
			} else {
				// Read the clipboard, for other platforms
				s, err = clipboard.ReadAll()
				if err == nil && strings.TrimSpace(s) == "" {
					s, err = getOtherClipboardContents()
				}
			}

			if err == nil { // no error

				// Make the replacements, then split the text into lines and store it in "copyLines"
				copyLines = strings.Split(opinionatedStringReplacer.Replace(s), "\n")

				// Note that control characters are not replaced, they are just not printed.
			} else if firstPasteAction {
				missingUtility := false

				status.Clear(c)

				if env.Has("WAYLAND_DISPLAY") && which("wl-paste") == "" { // Wayland + wl-paste not found
					status.SetErrorMessage("The wl-paste utility (from wl-clipboard) is missing!")
					missingUtility = true
				} else if env.Has("DISPLAY") && which("xclip") == "" { // X + xclip not found
					status.SetErrorMessage("The xclip utility is missing!")
					missingUtility = true
				} else if runtime.GOOS == "darwin" && which("pbpaste") == "" { // pbcopy is missing, on macOS
					status.SetErrorMessage("The pbpaste utility is missing!")
					missingUtility = true
				}

				if missingUtility && firstPasteAction {
					firstPasteAction = false
					status.Show(c, e)
					break // Break instead of pasting from the internal buffer, but only the first time
				}
			} else {
				status.Clear(c)
				e.redrawCursor = true
			}

			// Now check if there is anything to paste
			if len(copyLines) == 0 {
				break
			}

			// Now save the contents to "previousCopyLines" and check if they are the same first
			if !equalStringSlices(copyLines, previousCopyLines) {
				// Start with single-line paste if the contents are new
				lastPasteY = -1
			}
			previousCopyLines = copyLines

			// Prepare to paste
			undo.Snapshot(e)
			y := e.DataY()

			// Forget the cut and copy line state
			lastCutY = -1
			lastCopyY = -1

			// Redraw after pasting
			e.redraw = true

			if lastPasteY != y { // Single line paste
				lastPasteY = y
				// Pressed for the first time for this line number, paste only one line

				// copyLines[0] is the line to be pasted, and it exists

				if e.EmptyRightTrimmedLine() {
					// If the line is empty, use the existing indentation before pasting
					e.SetLine(y, e.LeadingWhitespace()+strings.TrimSpace(copyLines[0]))
				} else {
					// If the line is not empty, insert the trimmed string
					e.InsertStringAndMove(c, strings.TrimSpace(copyLines[0]))
				}

			} else { // Multi line paste (the rest of the lines)
				// Pressed the second time for this line number, paste multiple lines without trimming
				var (
					// copyLines contains the lines to be pasted, and they are > 1
					// the first line is skipped since that was already pasted when ctrl-v was pressed the first time
					lastIndex = len(copyLines[1:]) - 1

					// If the first line has been pasted, and return has been pressed, paste the rest of the lines differently
					skipFirstLineInsert bool
				)

				if !kh.PrevIs("c:13") {
					// Start by pasting (and overwriting) an untrimmed version of this line,
					// if the previous key was not return.
					e.SetLine(y, copyLines[0])
				} else if e.EmptyRightTrimmedLine() {
					skipFirstLineInsert = true
				}

				// Then paste the rest of the lines, also untrimmed
				for i, line := range copyLines[1:] {
					if i == lastIndex && len(strings.TrimSpace(line)) == 0 {
						// If the last line is blank, skip it
						break
					}
					if skipFirstLineInsert {
						skipFirstLineInsert = false
					} else {
						e.InsertLineBelow()
						e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
					}
					e.InsertStringAndMove(c, line)
				}
			}
			// Prepare to redraw the text
			e.redrawCursor = true
			e.redraw = true
		case "c:18": // ctrl-r, to open or close a portal. In debug mode, continue running the program.

			if e.debugMode {
				e.DebugContinue()
				break
			}

			// Are we in git mode?
			if line := e.CurrentLine(); e.mode == mode.Git && hasAnyPrefixWord(line, gitRebasePrefixes) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetCurrentLine(newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			// Deal with the portal
			status.Clear(c)
			if HasPortal() {
				status.SetMessage("Closing portal")
				e.ClosePortal()
			} else {
				portal, err := e.NewPortal()
				if err != nil {
					status.SetError(err)
					status.Show(c, e)
					break
				}
				// Portals in the same file is a special case, since lines may move around when pasting
				if portal.SameFile(e) {
					e.sameFilePortal = portal
				}
				if err := portal.Save(); err != nil {
					status.SetError(err)
					status.Show(c, e)
					break
				}
				status.SetMessage("Opening a portal at " + portal.String())
			}
			status.Show(c, e)
		case "c:2": // ctrl-b, bookmark, unbookmark or jump to bookmark, toggle breakpoint if in debug mode
			status.Clear(c)
			if e.debugMode {
				if e.breakpoint == nil {
					e.breakpoint = e.pos.Copy()
					_, err := e.DebugActivateBreakpoint(filepath.Base(e.filename))
					if err != nil {
						status.SetError(err)
						break
					}
					s := "Placed breakpoint at line " + e.LineNumber().String()
					status.SetMessage("  " + s + "  ")
				} else if e.breakpoint.LineNumber() == e.LineNumber() {
					// setting a breakpoint at the same line twice: remove the breakpoint
					s := "Removed breakpoint at line " + e.breakpoint.LineNumber().String()
					status.SetMessage(s)
					e.breakpoint = nil
				} else {
					undo.Snapshot(e)
					// Go to the breakpoint position
					e.GoToPosition(c, status, *e.breakpoint)
					// TODO: Just use status.SetMessageAfterRedraw instead?
					// Do the redraw manually before showing the status message
					e.DrawLines(c, true, false)
					e.redraw = false
					// Show the status message
					s := "Jumped to breakpoint at line " + e.LineNumber().String()
					status.SetMessage(s)
				}
			} else {
				if bookmark == nil {
					// no bookmark, create a bookmark at the current line
					bookmark = e.pos.Copy()
					// TODO: Modify the statusbar implementation so that extra spaces are not needed here.
					s := "Bookmarked line " + e.LineNumber().String()
					status.SetMessage("  " + s + "  ")
				} else if bookmark.LineNumber() == e.LineNumber() {
					// bookmarking the same line twice: remove the bookmark
					s := "Removed bookmark for line " + bookmark.LineNumber().String()
					status.SetMessage(s)
					bookmark = nil
				} else {
					undo.Snapshot(e)
					// Go to the saved bookmark position
					e.GoToPosition(c, status, *bookmark)
					// TODO: Just use status.SetMessageAfterRedraw instead?
					// Do the redraw manually before showing the status message
					e.DrawLines(c, true, false)
					e.redraw = false
					// Show the status message
					s := "Jumped to bookmark at line " + e.LineNumber().String()
					status.SetMessage(s)
				}
			}
			status.Show(c, e)
			e.redrawCursor = true
		case "c:10": // ctrl-j, join line
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				undo.Snapshot(e)

				nextLineIndex := e.DataY() + 1
				if e.EmptyRightTrimmedLineBelow() {
					// Just delete the line below if it's empty
					e.DeleteLineMoveBookmark(nextLineIndex, bookmark)
				} else {
					// Join the line below with this line. Also add a space in between.
					e.TrimLeft(nextLineIndex) // this is unproblematic, even at the end of the document
					e.End(c)
					e.InsertRune(c, ' ')
					e.WriteRune(c)
					e.Next(c)
					e.Delete()
				}

				e.redraw = true
			}
			e.redrawCursor = true
		default: // any other key
			keyRunes := []rune(key)
			// panic(fmt.Sprintf("PRESSED KEY: %v", []rune(key)))
			if len(keyRunes) > 0 && unicode.IsLetter(keyRunes[0]) { // letter

				undo.Snapshot(e)

				if e.mode == mode.Go { // TODO: And e.onlyValidCode
					if e.Empty() {
						r := keyRunes[0]
						// Only "/" or "p" is allowed
						if r != 'p' && r != '/' {
							status.Clear(c)
							status.SetMessage("Not valid Go: " + string(r))
							status.Show(c, e)
							break
						}
					}
				}

				// Type in the letters that were pressed
				for _, r := range keyRunes {
					// Insert a letter. This is what normally happens.
					wrapped := e.InsertRune(c, r)
					if !wrapped {
						e.WriteRune(c)
						e.Next(c)
					}
					e.redraw = true
				}
			} else if len(keyRunes) > 0 && unicode.IsGraphic(keyRunes[0]) { // any other key that can be drawn
				undo.Snapshot(e)
				e.redraw = true

				// Place *something*
				r := keyRunes[0]

				if r == 160 {
					// This is a nonbreaking space that may be inserted with altgr+space that is HORRIBLE.
					// Set r to a regular space instead.
					r = ' '
				}

				// "smart dedent"
				if r == '}' || r == ']' || r == ')' {

					// Normally, dedent once, but there are exceptions

					noContentHereAlready := len(e.TrimmedLine()) == 0
					leadingWhitespace := e.LeadingWhitespace()
					nextLineContents := e.Line(e.DataY() + 1)

					currentX := e.pos.sx

					foundCurlyBracketBelow := currentX-1 == strings.Index(nextLineContents, "}")
					foundSquareBracketBelow := currentX-1 == strings.Index(nextLineContents, "]")
					foundParenthesisBelow := currentX-1 == strings.Index(nextLineContents, ")")

					noDedent := foundCurlyBracketBelow || foundSquareBracketBelow || foundParenthesisBelow

					// noDedent := similarLineBelow

					// Okay, dedent this line by 1 indentation, if possible
					if !noDedent && e.pos.sx > 0 && len(leadingWhitespace) > 0 && noContentHereAlready {
						newLeadingWhitespace := leadingWhitespace
						if strings.HasSuffix(leadingWhitespace, "\t") {
							newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
							e.pos.sx -= e.indentation.PerTab
						} else if strings.HasSuffix(leadingWhitespace, strings.Repeat(" ", e.indentation.PerTab)) {
							newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-e.indentation.PerTab]
							e.pos.sx -= e.indentation.PerTab
						}
						e.SetCurrentLine(newLeadingWhitespace)
					}
				}

				wrapped := e.InsertRune(c, r)
				e.WriteRune(c)
				if !wrapped && len(string(r)) > 0 {
					// Move to the next position
					e.Next(c)
				}
				e.redrawCursor = true
			}
		}
		if e.addSpace {
			e.InsertString(c, " ")
			e.addSpace = false
		}

		// Clear the key history, if needed
		if clearKeyHistory {
			kh.Clear()
			clearKeyHistory = false
		} else {
			kh.Push(key)
		}

		// Display the ctrl-o menu if esc was already pressed 2 times,
		// (and this is the third keypress)
		if kh.Repeated("c:27", 2) {
			status.ClearAll(c)
			undo.Snapshot(e)
			undoBackup := undo
			lastCommandMenuIndex = e.CommandMenu(c, tty, status, bookmark, undo, lastCommandMenuIndex, forceFlag, fileLock)
			undo = undoBackup
			// And reset the esc counter
			clearKeyHistory = true
		}

		// Clear status, if needed
		if e.statusMode && e.redrawCursor {
			status.ClearAll(c)
		}

		// Draw and/or redraw everything, with slightly different behavior over ssh
		e.RedrawAtEndOfKeyLoop(c, status)

		// Also draw the watches, if debug mode is enabled // and a debug session is in progress
		if e.debugMode {
			e.DrawWatches(c, false)      // don't reposition cursor
			e.DrawRegisters(c, false)    // don't reposition cursor
			e.DrawGDBOutput(c, false)    // don't reposition cursor
			e.DrawInstructions(c, false) // don't reposition cursor
			e.DrawFlags(c, true)         // also reposition cursor
		}

	} // end of main loop

	if canUseLocks {
		// Start by loading the lock overview, just in case something has happened in the mean time
		fileLock.Load()

		// Check if the lock is unchanged
		fileLockTimestamp := fileLock.GetTimestamp(absFilename)
		lockUnchanged := lockTimestamp == fileLockTimestamp

		// TODO: If the stored timestamp is older than uptime, unlock and save the lock overview

		// var notime time.Time

		if !forceFlag || lockUnchanged {
			// If the file has not been locked externally since this instance of the editor was loaded, don't
			// Unlock the current file and save the lock overview. Ignore errors because they are not critical.
			fileLock.Unlock(absFilename)
			fileLock.Save()
		}
	}

	// Save the current location in the location history and write it to file
	e.SaveLocation(absFilename, locationHistory)

	// Clear all status bar messages
	status.ClearAll(c)

	// Quit everything that has to do with the terminal
	if e.clearOnQuit {
		vt100.Clear()
		vt100.Close()
	} else {
		c.Draw()
		fmt.Println()
	}

	// All done
	return "", e.stopParentOnQuit, nil
}
