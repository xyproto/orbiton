package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/xyproto/clip"
	"github.com/xyproto/digraph"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// For when the user scrolls too far
const (
	endOfFileMessage = "EOF"

	leftArrow  = "←"
	rightArrow = "→"
	upArrow    = "↑"
	downArrow  = "↓"

	// These keys are undocumented features
	pgUpKey = "⇞" // page up
	pgDnKey = "⇟" // page down
	homeKey = "⇱" // home
	endKey  = "⇲" // end
	copyKey = "⎘" // ctrl-insert

	delayUntilSpeedUp = 700 * time.Millisecond
)

var (
	// Create a LockKeeper for keeping track of which files are being edited
	fileLock = NewLockKeeper(defaultLockFile)
	// Remember if locks can be saved and loaded
	canUseLocks atomic.Bool

	// Track if the user is in regular editing mode (not in a menu or special mode)
	notRegularEditingRightNow atomic.Bool
)

// Loop will set up and run the main loop of the editor
// a *vt.TTY struct
// fnord contains either data or a filename to open
// a LineNumber (may be 0 or -1)
// a forceFlag for if the file should be force opened
// If an error and "true" is returned, it is a quit message to the user, and not an error.
// If an error and "false" is returned, it is an error.
func Loop(tty *vt.TTY, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, theme Theme, syntaxHighlight, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp, fmtFlag bool) (userMessage string, stopParent bool, err error) {

	// Create a Canvas for drawing onto the terminal
	vt.Init()
	c := vt.NewCanvas()
	c.ShowCursor()
	vt.EchoOff()

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

		markdownTableEditorCounter int // the number of times the Markdown table editor has been displayed

		highlightTimerCounter atomic.Uint64
		highlightTimerMut     sync.Mutex

		justJumpedToMatchingP bool

		// Keys where one wishes to speed up the actions when they are held down for a while
		heldDownLeftArrowTime  time.Time // used to speed up cursor movement after the left arrow key has been held down for a while
		heldDownRightArrowTime time.Time // used to speed up cursor movement after the right arrow key has been held down for a while
		heldDownCtrlKTime      time.Time // used to speed up line deletion after ctrl-k has been held down for a while
	)

	// TODO: Move this to themes.go
	if nanoMode { // make the status bar stand out
		theme.StatusBackground = theme.DebugInstructionsBackground
		theme.StatusErrorBackground = theme.DebugInstructionsBackground
	}

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e, messageAfterRedraw, displayedImage, err := NewEditor(tty, c, fnord, lineNumber, colNumber, theme, syntaxHighlight, true, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp)
	if err != nil {
		if e != nil {
			return "", false, err
		}
		clearOnQuit.Store(false)
		return "", false, err
	} else if displayedImage {
		// A special case for if an image was displayed instead of a file being opened
		if e != nil {
			return "", false, nil
		}
		clearOnQuit.Store(false)
		return "", false, nil
	}

	// Find the absolute path to this filename
	absFilename := fnord.filename
	if !fnord.stdin {
		if filename, err := e.AbsFilename(); err == nil { // success
			absFilename = filename
		}
	}

	if parentIsMan == nil {
		b := parentProcessIs("man")
		parentIsMan = &b
	}
	if *parentIsMan {
		// The parent process is "man", but NROFF_FILENAME has not been set:
		// This means that ctrl-space has been pressed a second time when editing an Nroff file, so switch back to Nroff mode.
		if env.Has("ORBITON_SPACE") && env.No("NROFF_FILENAME") {
			e.mode = mode.Nroff
		} else {
			e.mode = mode.ManPage
		}
	}

	// Minor adjustments to some modes
	switch e.mode {
	case mode.Email, mode.Git:
		e.StatusForeground = vt.LightBlue
		e.StatusBackground = vt.BackgroundDefault
	case mode.ManPage:
		e.readOnly = true
	}

	// Prepare a status bar
	status := e.NewStatusBar(statusDuration, messageAfterRedraw)

	e.SetTheme(e.Theme)

	// ctrl-c, USR1 and terminal resize handlers
	const onlyClearSignals = false
	e.SetUpSignalHandlers(c, tty, status, onlyClearSignals)

	// Monitor a read-only file?
	if monitorAndReadOnly {
		e.readOnly = true
		if err := e.StartMonitoring(c, tty, status); err != nil {
			quitError(tty, err)
		}
	}

	if e.mode == mode.Log && e.readOnly {
		e.syntaxHighlight = true
	}

	e.previousX = 1
	e.previousY = 1

	tty.SetTimeout(2 * time.Millisecond)

	var lockTimestamp time.Time
	canUseLocks.Store(!fnord.stdin && !monitorAndReadOnly)

	if canUseLocks.Load() {

		go func() {
			// If the lock keeper does not have an overview already, that's fine. Ignore errors from lk.Load().
			if err := fileLock.Load(); err != nil {
				// Could not load an existing lock overview, this might be the first run? Try saving.
				if err := fileLock.Save(); err != nil {
					// Could not save a lock overview. Can not use locks.
					canUseLocks.Store(false)
				}
			}
		}()

		// Check if the lock should be forced (also force when running git commit, because it is likely that o was killed in that case)
		if forceFlag || filepath.Base(absFilename) == "COMMIT_EDITMSG" || env.Bool("O_FORCE") {
			// Lock and save, regardless of what the previous status is
			go func() {
				fileLock.Lock(absFilename)
				// TODO: If the file was already marked as locked, this is not strictly needed? The timestamp might be modified, though.
				fileLock.Save()
			}()
		} else {
			// Lock the current file, if it's not already locked
			if err := fileLock.Lock(absFilename); err != nil {
				return fmt.Sprintf("Locked by another (possibly dead) instance of this editor.\nTry: o -f %s", filepath.Base(absFilename)), false, errors.New(absFilename + " is locked")
			}
			// Save the lock file as a signal to other instances of the editor
			go fileLock.Save()
		}
		lockTimestamp = fileLock.GetTimestamp(absFilename)

		// Set up a catch for panics, so that the current file can be unlocked
		defer func() {
			if x := recover(); x != nil {
				// Unlock and save the lock file
				go func() {
					quitMut.Lock()
					defer quitMut.Unlock()
					fileLock.Unlock(absFilename)
					fileLock.Save()
				}()

				// Save the current file. The assumption is that it's better than not saving, if something crashes.
				// TODO: Save to a crash file, then let the editor discover this when it starts.

				// Create a suitable error message, depending on if the file is saved or not
				msg := fmt.Sprintf("Saved the file first!\n%v", x)
				if err := e.Save(c, tty); err != nil {
					// Output the error message
					msg = fmt.Sprintf("Could not save the file first! %v\n%v", err, x)
				}

				// Output the error message
				quitMessageWithStack(tty, msg)
			}
		}()
	}

	if fmtFlag { // Only try to format the file, save the file and then quit
		if e.mode == mode.Markdown {
			e.GoToStartOfTextLine(c)
			e.FormatAllMarkdownTables()
		}
		e.formatCode(c, tty, status, &jsonFormatToggle) // jsonFormatToggle is for also formatting indentation, or not, when formatting
		if msg := strings.TrimSpace(status.msg); status.isError && msg != "" {
			quitError(tty, errors.New(msg))
		}
		e.UserSave(c, tty, status)
		// Continue to the loop and then quit
		e.quit = true
	}

	// Draw everything once, with slightly different behavior if used over ssh
	e.InitialRedraw(c, status)

	// Request function description at startup if cursor is on a function
	if ollama.Loaded() && ProgrammingLanguage(e.mode) {
		s := e.FindCurrentFunctionName()
		if s != "" {
			// Extract function body
			y := e.DataY()
			funcBody, err := e.FunctionBlock(y)
			if err != nil {
				funcBody = e.Block(y)
			}
			if funcBody != "" {
				e.RequestFunctionDescription(s, funcBody, c)
			}
		}
	}

	// QuickHelp screen + help for new users
	if (!QuickHelpScreenIsDisabled() || e.displayQuickHelp) && !e.noDisplayQuickHelp {
		e.DrawQuickHelp(c, false)
	}

	// Place and enable the cursor
	e.PlaceAndEnableCursor()

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

			if e.nanoMode.Load() { // nano: ctrl-w, search backwards
				const clearPreviousSearch = true
				const searchForward = false
				e.SearchMode(c, status, tty, clearPreviousSearch, searchForward, undo)
				break
			}

			e.quit = true
		case "c:23": // ctrl-w, format or insert template (or if in git mode, cycle interactive rebase keywords)

			if e.nanoMode.Load() { // nano: ctrl-w, search
				const clearPreviousSearch = true
				const searchForward = true
				e.SearchMode(c, status, tty, clearPreviousSearch, searchForward, undo)
				break
			}

			undo.Snapshot(e)

			// Clear the search term
			e.ClearSearch()

			// First check if we are editing Markdown and are in a Markdown table (and that this is not the previous thing that we did)
			if e.mode == mode.Markdown && e.InTable() && !kh.PrevIs("c:23") {
				e.GoToStartOfTextLine(c)
				// Just format the Markdown table
				const justFormat = true
				const displayQuickHelp = false
				e.EditMarkdownTable(tty, c, status, bookmark, justFormat, displayQuickHelp)
				break
			} else if e.mode == mode.Markdown && !kh.PrevIs("c:23") {
				e.GoToStartOfTextLine(c)
				e.FormatAllMarkdownTables()
				break
			}

			// Add a watch
			if e.debugMode { // AddWatch will start a new gdb session if needed
				// Ask the user to type in a watch expression
				if expression, ok := e.UserInput(c, tty, status, "Variable name to watch", "", []string{}, false, ""); ok {
					if _, err := e.AddWatch(expression); err != nil {
						status.ClearAll(c, true)
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
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				break
			}

			if e.Empty() {
				// If the filename is __init__.py, don't insert anything, since this should often stay empty
				if filepath.Base(e.filename) == "__init__.py" {
					break
				}
				// Empty file, nothing to format, insert a program template, if available
				if err := e.InsertTemplateProgram(c); err != nil {
					status.ClearAll(c, true)
					status.SetMessage("nothing to format and no template available")
					status.Show(c, e)
				} else {
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
				}
				break
			}

			status.ClearAll(c, true)
			e.formatCode(c, tty, status, &jsonFormatToggle)

			// Move the cursor if after the end of the line
			if e.AtOrAfterEndOfLine() {
				e.End(c)
			}

			// Keep the message on screen for 1 second, despite e.redraw being set.
			// This is only to have a minimum amount of display time for the message.
			status.HoldMessage(c, 250*time.Millisecond)

		case "c:6": // ctrl-f, search for a string

			if e.nanoMode.Load() { // nano: ctrl-f, cursor forward
				e.CursorForward(c, status)
				break
			}

			// If in Debug mode, let ctrl-f mean "finish"
			if e.debugMode {
				if e.gdb == nil { // success
					status.SetMessageAfterRedraw("Not running")
					break
				}
				status.ClearAll(c, false)
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

			const clearPreviousSearch = true
			const searchForward = true
			e.SearchMode(c, status, tty, clearPreviousSearch, searchForward, undo)

		case "c:0": // ctrl-space, build source code to executable, or export, depending on the mode
			if e.nanoMode.Load() {
				break // do nothing
			}

			switch e.mode {
			case mode.Markdown:
				if e.ToggleCheckboxCurrentLine() { // Toggle checkbox
					undo.Snapshot(e)
					break
				}
			case mode.Config, mode.FSTAB:
				break // do nothing
			case mode.Nroff:
				// Switch to man page mode, just in case the switching over does not work out
				e.mode = mode.ManPage
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				// Save the current file, but only if it has changed
				if e.changed.Load() {
					if err := e.Save(c, tty); err != nil {
						status.ClearAll(c, false)
						status.SetError(err)
						status.Show(c, e)
						break
					}
				}
				// Switch over to the rendered man page output, as produced by the "man" command
				if pwd, err := os.Getwd(); err == nil {
					if absFilename, err := filepath.Abs(e.filename); err == nil { // success
						e.SetUpSignalHandlers(c, tty, status, true) // only clear signals
						var wg sync.WaitGroup
						e.CloseLocksAndLocationHistory(absFilename, lockTimestamp, forceFlag, &wg)
						wg.Wait()
						quitToMan(tty, pwd, absFilename, c.W(), c.H())
					}
				}
			case mode.ManPage:
				// Switch back to Nroff mode, just in case the switching below does not work out (or is activated)
				e.mode = mode.Nroff
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				// Used when switching back and forth between editing and viewing man pages
				if env.Has("NROFF_FILENAME") {
					if pwd, err := os.Getwd(); err == nil {
						e.SetUpSignalHandlers(c, tty, status, true) // only clear signals
						var wg sync.WaitGroup
						e.CloseLocksAndLocationHistory(absFilename, lockTimestamp, forceFlag, &wg)
						wg.Wait()
						quitToNroff(tty, pwd, c.W(), c.H())
					}
				}
			default:
				// Then build, and run if ctrl-space was double-tapped
				e.runAfterBuild = kh.DoubleTapped("c:0")
				// Stop background processes first (if any)
				stopBackgroundProcesses()
				// Then build (and run)
				e.Build(c, status, tty)
				e.redrawCursor.Store(true)
			}

		case "c:20": // ctrl-t

			// for C or C++: jump to header/source, or insert symbol
			// for Agda: insert symbol
			// for the rest: record and play back macros
			// debug mode: next instruction

			// Save the current file, but only if it has changed
			if !e.nanoMode.Load() && e.changed.Load() {
				if err := e.Save(c, tty); err != nil {
					status.ClearAll(c, false)
					status.SetError(err)
					status.Show(c, e)
					break
				}
			}

			if e.nanoMode.Load() {
				e.NanoNextTypo(c, status)
				break
			}

			e.redrawCursor.Store(true)

			// Is there no corresponding header or source file?
			noCorresponding := false
		AGAIN_NO_CORRESPONDING:

			// First check if we are editing Markdown
			if e.mode == mode.Markdown {
				// Try to toggle checkbox first
				if e.ToggleCheckboxCurrentLine() {
					undo.Snapshot(e)
					break
				}
				// If no checkbox, check if we're in a table
				if e.EmptyLine() || e.InTable() { // table editor
					if e.EmptyLine() {
						e.InsertStringAndMove(c, "| | |\n|-|-|\n| | |\n")
						e.Up(c, status)
					}
					undo.Snapshot(e)
					e.GoToStartOfTextLine(c)
					// Edit the Markdown table
					const justFormat = false
					var displayQuickHelp = markdownTableEditorCounter < 1
					e.EditMarkdownTable(tty, c, status, bookmark, justFormat, displayQuickHelp)
					markdownTableEditorCounter++
					// Full redraw
					const drawLines = true
					justMovedUpOrDown := kh.PrevHas("↓", "↑")
					e.FullResetRedraw(c, status, drawLines, justMovedUpOrDown)
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
				}
			} else if !noCorresponding && (e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC) && hasS([]string{".cpp", ".cc", ".c", ".cxx", ".c++", ".m", ".mm", ".M"}, filepath.Ext(e.filename)) { // jump from source to header file
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
				noCorresponding = true
				goto AGAIN_NO_CORRESPONDING
			} else if !noCorresponding && (e.mode == mode.C || e.mode == mode.Cpp || e.mode == mode.ObjC) && hasS([]string{".h", ".hpp", ".h++"}, filepath.Ext(e.filename)) { // jump from header to source file
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
				noCorresponding = true
				goto AGAIN_NO_CORRESPONDING
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
				e.redraw.Store(true)
				selectedX, selectedY, cancel := e.SymbolMenu(tty, status, "Insert symbol", menuChoices, e.MenuTitleColor, e.MenuTextColor, e.MenuArrowColor)
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
				// Full redraw
				const drawLines = true
				justMovedUpOrDown := kh.PrevHas("↓", "↑")
				e.FullResetRedraw(c, status, drawLines, justMovedUpOrDown)
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
			} else if e.macro == nil {
				// Start recording a macro, then stop the recording when ctrl-t is pressed again,
				// then ask for the number of repetitions to play it back when it's pressed after that,
				// then clear the macro when esc is pressed.
				undo.Snapshot(e)
				undo.IgnoreSnapshots(true)
				status.ClearAll(c, false)
				status.SetMessageAfterRedraw("Recording macro")
				e.macro = NewMacro()
				e.macro.Recording = true
				e.playBackMacroCount = 0
			} else if e.macro.Recording { // && e.macro != nil
				e.macro.Recording = false
				undo.IgnoreSnapshots(true)
				e.playBackMacroCount = 0
				status.Clear(c, false)
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
				status.Clear(c, false)
				status.SetMessage("Stopped macro") // stop macro playback
				status.Show(c, e)
				e.playBackMacroCount = 0
				e.macro.Home()
			} else { // && e.macro != nil && e.playBackMacroCount == 0 // start macro playback
				undo.IgnoreSnapshots(false)
				undo.Snapshot(e)
				status.ClearAll(c, false)
				// Play back the macro, once
				e.playBackMacroCount = 1
			}
		case "c:28": // ctrl-\, toggle comment for this block
			undo.Snapshot(e)
			e.ToggleCommentBlock(c)
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
		case "c:15": // ctrl-o, launch the command menu

			if e.nanoMode.Load() { // ctrl-o, save
				// Ask the user which filename to save to
				if newFilename, ok := e.UserInput(c, tty, status, "Save as", e.filename, []string{e.filename}, false, e.filename); ok {
					e.filename = newFilename
					e.Save(c, tty)
					e.Switch(c, tty, status, fileLock, newFilename)
				} else {
					status.Clear(c, false)
					status.SetMessage("Saved nothing")
					status.Show(c, e)
				}
				break
			}

			status.ClearAll(c, false)
			undo.Snapshot(e)
			undoBackup := undo
			selectedIndex, spacePressed := e.CommandMenu(c, tty, status, bookmark, undo, lastCommandMenuIndex, forceFlag, fileLock)
			lastCommandMenuIndex = selectedIndex
			if spacePressed {
				status.Clear(c, false)
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
			}
			undo = undoBackup

		case "c:31": // ctrl-_, jump to a matching parenthesis or enter a digraph

			if e.nanoMode.Load() { // nano: ctrl-/
				// go to line
				e.JumpMode(c, status, tty)
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				break
			}

			// Ask the user to type in a digraph
			const tabInputText = "ae"
			if digraphString, ok := e.UserInput(c, tty, status, "Type in a 2-letter digraph", "", digraph.All(), false, tabInputText); ok {
				if r, ok := digraph.Lookup(digraphString); !ok {
					status.ClearAll(c, true)
					status.SetErrorMessage(fmt.Sprintf("Could not find the %q digraph", digraphString))
					status.ShowNoTimeout(c, e)
				} else {
					undo.Snapshot(e)
					// Insert the found rune
					wrapped := e.InsertRune(c, r)
					if !wrapped {
						e.WriteRune(c)
						// Move to the next position
						e.Next(c)
					}
					e.redraw.Store(true)
				}
			}

		case leftArrow: // left arrow

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith(leftArrow) {
				// TODO: Instead of moving up twice, play back the reverse of the latest keypress history
				e.Up(c, status)
				e.Up(c, status)
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			e.CursorBackward(c, status)

			// Move extra if the key is held down
			if kh.TwoLastAre(leftArrow) && kh.AllWithin(200*time.Millisecond) && kh.LastChanged(200*time.Millisecond) {
				if heldDownLeftArrowTime.IsZero() {
					heldDownLeftArrowTime = time.Now()
				}
				heldDuration := time.Since(heldDownLeftArrowTime)
				steps := int(int64(heldDuration) / int64(delayUntilSpeedUp))
				for i := 1; i < steps; i++ {
					e.CursorBackward(c, status)
				}
			} else {
				heldDownLeftArrowTime = time.Time{}
			}

			if e.highlightCurrentLine || e.highlightCurrentText {
				e.redraw.Store(true)
				e.drawFuncName.Store(true)
			}

		case rightArrow: // right arrow

			// Check if it's a special case
			if kh.SpecialArrowKeypressWith(rightArrow) {
				// TODO: Instead of moving up twice, play back the reverse of the latest keypress history
				e.Up(c, status)
				e.Up(c, status)
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			e.CursorForward(c, status)

			// Move extra if the key is held down
			if kh.TwoLastAre(rightArrow) && kh.AllWithin(200*time.Millisecond) && kh.LastChanged(200*time.Millisecond) {
				if heldDownRightArrowTime.IsZero() {
					heldDownRightArrowTime = time.Now()
				}
				heldDuration := time.Since(heldDownRightArrowTime)
				steps := int(int64(heldDuration) / int64(delayUntilSpeedUp))
				for i := 1; i < steps; i++ {
					e.CursorForward(c, status)
				}
			} else {
				heldDownRightArrowTime = time.Time{}
			}

			if e.highlightCurrentLine || e.highlightCurrentText {
				e.redraw.Store(true)
				e.drawFuncName.Store(true)
			}

		case upArrow: // up arrow
			// Check if it's a special case
			if kh.SpecialArrowKeypressWith(upArrow) {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			e.CursorUpward(c, status)

			// Move extra if the key is held down
			if kh.TwoLastAre(upArrow) && kh.AllWithin(200*time.Millisecond) && kh.LastChanged(200*time.Millisecond) {
				e.CursorUpward(c, status)
			}

			if e.highlightCurrentLine || e.highlightCurrentText {
				e.redraw.Store(true)
				e.drawFuncName.Store(true)
			}
			e.redrawCursor.Store(true)

		case downArrow: // down arrow
			// Check if it's a special case
			if kh.SpecialArrowKeypressWith(downArrow) {
				// Ask the user for a command and run it
				e.CommandPrompt(c, tty, status, bookmark, undo)
				// It's important to reset the key history after hitting this combo
				clearKeyHistory = true
				break
			}

			e.CursorDownward(c, status)

			// Move extra if the key is held down
			if kh.TwoLastAre(downArrow) && kh.AllWithin(200*time.Millisecond) && kh.LastChanged(200*time.Millisecond) {
				e.CursorDownward(c, status)
			}

			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.AfterLineScreenContents() || e.AfterEndOfLine() {
				e.End(c)
				e.redraw.Store(true)
			}
			if e.highlightCurrentLine || e.highlightCurrentText {
				e.redraw.Store(true)
				e.drawFuncName.Store(true)
			}
			e.redrawCursor.Store(true)

		case "c:16": // ctrl-p, scroll up or jump to the previous match, using the sticky search term. In debug mode, change the pane layout.

			if !e.nanoMode.Load() {
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
						status.ClearAll(c, false)
						msg := e.SearchTerm() + " not found"
						if e.spellCheckMode {
							msg = "No typos found"
						}
						if !wrap {
							msg += " from here"
						}
						status.SetMessageAfterRedraw(msg)
						e.spellCheckMode = false
						e.ClearSearch()
					}
				} else {

					// Jump to a matching parenthesis if either an arrow key was last pressed or we just jumped to a matchin parenthesis
					justUsedArrowKeys := kh.PrevHas(downArrow, upArrow, leftArrow, rightArrow)
					if (justUsedArrowKeys || justJumpedToMatchingP) && e.JumpToMatching(c) {
						justJumpedToMatchingP = true
						e.redraw.Store(true)
						e.redrawCursor.Store(true)
						break
					}
					justJumpedToMatchingP = false

					if e.moveLinesMode.Load() && e.AtSecondLineOfDocumentOrLater() {
						// Move the current line up
						line := e.CurrentLine()
						e.DeleteCurrentLineMoveBookmark(bookmark)
						e.Up(c, nil) // no status message if the end of document is reached, there should always be a new line
						e.Home()
						e.InsertLineAbove()
						e.InsertStringAndMove(c, line)
						e.Home()
					} else {
						// Scroll up
						e.redraw.Store(e.ScrollUp(c, status, e.pos.scrollSpeed))
						e.redrawCursor.Store(true)
						if e.AfterLineScreenContents() {
							e.End(c)
						}
					}
				}
				e.drawProgress.Store(true)
				e.drawFuncName.Store(true)
				break
			}

			// nano mode

			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to previous match
				wrap := true
				forward := false
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.ClearAll(c, false)
					msg := e.SearchTerm() + " not found"
					if e.spellCheckMode {
						msg = "No typos found"
					}
					if !wrap {
						msg += " from here"
					}
					status.SetMessageAfterRedraw(msg)
					e.spellCheckMode = false
					e.ClearSearch()
				}
				break
			}

			fallthrough // ctrl-p in nano mode

		case "c:14": // ctrl-n, scroll down or jump to next match, using the sticky search term

			if !e.nanoMode.Load() {

				// If in Debug mode, let ctrl-n mean "next instruction"
				if e.debugMode {
					if e.gdb != nil {
						if !programRunning {
							e.DebugEnd()
							status.SetMessage("Program stopped")
							status.SetMessageAfterRedraw(status.Message())
							e.redraw.Store(true)
							e.redrawCursor.Store(true)
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
						e.redrawCursor.Store(true)
						status.SetMessageAfterRedraw(status.Message())
						break
					} // e.gdb == nil
					// Build or export the current file
					outputExecutable, err := e.BuildOrExport(tty, c, status)
					// All clear when it comes to status messages and redrawing
					status.ClearAll(c, false)
					if err != nil && err != errNoSuitableBuildCommand {
						// Error while building
						status.SetError(err)
						status.ShowNoTimeout(c, e)
						e.debugMode = false
						e.redrawCursor.Store(true)
						e.redraw.Store(true)
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
							e.redrawCursor.Store(true)
							e.redraw.Store(true)
							break
						}
						// Building this file extension is not implemented yet.
						// Just display the current time and word count.
						// TODO: status.ClearAll() should have cleared the status bar first, but this is not always true,
						//       which is why the message is hackily surrounded by spaces. Fix.
						statsMessage := fmt.Sprintf("    %d words, %s    ", e.WordCount(), time.Now().Format("15:04")) // HH:MM
						status.SetMessage(statsMessage)
						status.Show(c, e)
						e.redrawCursor.Store(true)
						break
					}
					// Start debugging
					if err := e.DebugStartSession(c, tty, status, outputExecutable); err != nil {
						status.ClearAll(c, false)
						status.SetError(err)
						status.ShowNoTimeout(c, e)
						e.redrawCursor.Store(true)
					}
					break
				}
				e.UseStickySearchTerm()
				if e.SearchTerm() != "" {
					// Go to next match
					wrap := true
					forward := true
					if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
						status.ClearAll(c, false)
						msg := e.SearchTerm() + " not found"
						if e.spellCheckMode {
							msg = "No typos found"
						}
						if wrap {
							status.SetMessage(msg)
						} else {
							status.SetMessage(msg + " from here")
						}
						status.ShowNoTimeout(c, e)
						e.spellCheckMode = false
						e.ClearSearch()
					}
				} else {

					// Jump to a matching parenthesis if either an arrow key was last pressed or we just jumped to a matching parenthesis
					justUsedArrowKeys := kh.PrevHas(downArrow, upArrow, leftArrow, rightArrow)
					if (justUsedArrowKeys || justJumpedToMatchingP) && e.JumpToMatching(c) {
						justJumpedToMatchingP = true
						e.redraw.Store(true)
						e.redrawCursor.Store(true)
						break
					}
					justJumpedToMatchingP = false

					if e.moveLinesMode.Load() && !e.AtOrAfterLastLineOfDocument() {
						// Move the current line down
						line := e.CurrentLine()
						e.DeleteCurrentLineMoveBookmark(bookmark)
						e.InsertLineBelow()
						e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
						e.Home()
						e.InsertStringAndMove(c, line)
						e.Home()
					} else {
						// Scroll down
						h := int(c.Height())
						redraw := e.ScrollDown(c, status, e.pos.scrollSpeed, h)
						e.redraw.Store(redraw)
						// If redraw is false, the end of file is reached
						if !redraw {
							status.Clear(c, false)
							status.SetMessage(endOfFileMessage)
							status.Show(c, e)
						}
					}

					e.redrawCursor.Store(true)
					if e.AfterLineScreenContents() {
						e.End(c)
					}
				}
				e.drawProgress.Store(true)
				e.drawFuncName.Store(true)
				break
			}

			// nano mode: ctrl-n

			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to next match
				wrap := true
				forward := true
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c, false)
					msg := e.SearchTerm() + " not found"
					if e.spellCheckMode {
						msg = "No typos found"
					}
					if wrap {
						status.SetMessageAfterRedraw(msg)
					} else {
						status.SetMessageAfterRedraw(msg + " from here")
					}
					e.redraw.Store(true)
					e.spellCheckMode = false
					e.ClearSearch()
				}
				break
			}

			fallthrough // nano mode: ctrl-n

		case "c:12": // ctrl-l, go to line number or percentage
			if !e.nanoMode.Load() {
				notRegularEditingRightNow.Store(true)
				e.ClearSearch() // clear the current search first
				switch e.JumpMode(c, status, tty) {
				case showHotkeyOverviewAction:
					const repositionCursorAfterDrawing = true
					e.DrawHotkeyOverview(tty, c, status, repositionCursorAfterDrawing)
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
				case launchTutorialAction:
					const drawLines = true
					e.FullResetRedraw(c, status, drawLines, false)
					LaunchTutorial(tty, c, e, status)
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
				case scrollUpAction:
					e.redraw.Store(e.ScrollUp(c, status, e.pos.scrollSpeed))
					e.redrawCursor.Store(true)
					if e.AfterLineScreenContents() {
						e.End(c)
					}
					e.drawProgress.Store(true)
					e.drawFuncName.Store(false)
				case scrollDownAction:
					canvasHeight := int(c.Height())
					e.redraw.Store(e.ScrollDown(c, status, e.pos.scrollSpeed, canvasHeight))
					e.redrawCursor.Store(true)
					if e.AfterLineScreenContents() {
						e.End(c)
					}
					e.drawProgress.Store(true)
					e.drawFuncName.Store(false)
				case displayQuickHelpAction:
					const repositionCursorAfterDrawing = true
					e.DrawQuickHelp(c, repositionCursorAfterDrawing)
					e.redraw.Store(false)
					e.redrawCursor.Store(false)
				}
				notRegularEditingRightNow.Store(false)
				break
			}
			fallthrough // nano: ctrl-l to refresh
		case "c:27": // esc, clear search term (but not the sticky search term), reset, clean and redraw
			e.blockMode = false
			// If o is used as a man page viewer, exit at the press of esc
			if e.mode == mode.ManPage {
				clearOnQuit.Store(false)
				e.quit = true
				break
			}
			// Exit debug mode, if active
			if e.debugMode {
				e.DebugEnd()
				e.debugMode = false
				status.SetMessageAfterRedraw("Normal mode")
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				break
			}
			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1
			// Stop background processes (like playing music with timidity), if any
			stopBackgroundProcesses()
			// Do a full clear and redraw + clear search term + jump
			const drawLines = true
			e.FullResetRedraw(c, status, drawLines, false)
			notRegularEditingRightNow.Store(false)
			if e.macro != nil || e.playBackMacroCount > 0 {
				// Stop the playback
				e.playBackMacroCount = 0
				// Clear the macro
				e.macro = nil
				// Show a message after the redraw
				status.SetMessageAfterRedraw("Macro cleared")
			}
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
		case " ": // space
			// Scroll down if a man page is being viewed, or if the editor is read-only
			if e.readOnly && !e.blockMode {
				// Try to scroll down a full page
				redraw := e.PgDn(c, status)
				e.redraw.Store(redraw)
				// If e.redraw is false, the end of file is reached
				if !redraw {
					status.Clear(c, false)
					status.SetMessage(endOfFileMessage)
					status.Show(c, e)
				}
				e.redrawCursor.Store(true)
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// Regular behavior, take an undo snapshot and insert a space
			undo.Snapshot(e)

			// De-indent this line by 1 if the line above starts with "case " and this line is only "case" at this time.
			if cLikeSwitch(e.mode) && e.TrimmedLine() == "case" && strings.HasPrefix(e.PrevTrimmedLine(), "case ") {
				oneIndentation := e.indentation.String()
				deIndented := strings.Replace(e.CurrentLine(), oneIndentation, "", 1)
				e.SetCurrentLine(deIndented)
				e.End(c)
			}

			// Place a space
			wrapped := e.InsertRune(c, ' ')
			if !wrapped {
				e.WriteRune(c)
				// Move to the next position
				e.Next(c)
			}
			e.redraw.Store(true)

		case "c:13", "\n": // return

			// Show a "Read only" status message if a man page is being viewed or if the editor is read-only
			// It is an alternative way to quickly check if the file is read-only,
			// and space can still be used for scrolling.
			if e.readOnly {
				status.Clear(c, false)
				status.SetMessage("Read only")
				status.Show(c, e)
				break
			}

			// Modify the paste double-keypress detection to allow for a manual return before pasting the rest
			if lastPasteY != -1 && kh.Prev() != "c:13" {
				lastPasteY++
			}

			undo.Snapshot(e)

			e.ReturnPressed(c, status)

		case "c:8", "c:127": // ctrl-h or backspace

			// Scroll up if a man page is being viewed, or if the editor is read-only
			if e.readOnly && !e.blockMode {
				// Scroll up at double speed
				e.redraw.Store(e.ScrollUp(c, status, e.pos.scrollSpeed*2))
				e.redrawCursor.Store(true)
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// Just clear the search term, if there is an active search
			if len(e.SearchTerm()) > 0 {
				e.ClearSearch()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				// Don't break, continue to delete to the left after clearing the search,
				// since Esc can be used to only clear the search.
				// break
			}

			undo.Snapshot(e)

			e.Backspace(c, bookmark)

			e.redrawCursor.Store(true)
			e.redraw.Store(true)

		case "c:9": // tab or ctrl-i

			if e.spellCheckMode {
				// TODO: Save a "custom words" and "ignored words" list to disk
				if ignoredWord := e.RemoveCurrentWordFromWordList(); ignoredWord != "" {
					typo, corrected := e.NanoNextTypo(c, status)
					msg := "Ignored " + ignoredWord
					if spellChecker != nil && typo != "" {
						msg += ". Found " + typo
						if corrected != "" {
							msg += " which could be " + corrected + "."
						} else {
							msg += "."
						}
					}
					status.SetMessageAfterRedraw(msg)
				}
				break
			}

			if e.debugMode {
				e.debugStepInto = !e.debugStepInto
				break
			}

			y := int(e.DataY())
			r := e.Rune()
			leftRune := e.LeftRune()

			if e.mode == mode.Go && e.syntaxHighlight {
				if e.pos.sx > 0 && (unicode.IsLetter(leftRune) || unicode.IsDigit(leftRune) || leftRune == '_' || leftRune == '.') {
					items, err := e.GetGoCompletions()
					if err == nil && len(items) > 0 {
						choices := make([]string, 0, len(items))
						for _, item := range items {
							label := item.Label
							if item.Detail != "" && len(item.Detail) < 40 {
								label += " • " + item.Detail
							}
							choices = append(choices, label)
						}
						currentWord := e.CurrentWord()
						if choice, _ := e.Menu(status, tty, "Completions", choices, e.Background, e.MenuTitleColor, e.MenuArrowColor, e.MenuTextColor, e.MenuHighlightColor, e.MenuSelectedColor, 0, false); choice >= 0 && choice < len(items) {
							undo.Snapshot(e)

							insertText := items[choice].InsertText
							if insertText == "" {
								insertText = items[choice].Label
							}
							if items[choice].TextEdit != nil && items[choice].TextEdit.NewText != "" {
								insertText = items[choice].TextEdit.NewText
							}
							// Strip function parameters
							if parenIndex := strings.Index(insertText, "("); parenIndex > 0 {
								insertText = insertText[:parenIndex]
							}

							// Calculate how many characters to delete based on textEdit range
							var charsToDelete int
							if items[choice].TextEdit != nil {
								// Use the range from gopls to determine what to replace
								rangeStart := items[choice].TextEdit.Range.Start.Character
								rangeEnd := items[choice].TextEdit.Range.End.Character
								charsToDelete = rangeEnd - rangeStart
							} else if currentWord != "" {
								charsToDelete = len([]rune(currentWord))
							}

							if charsToDelete > 0 {
								for i := 0; i < charsToDelete; i++ {
									e.Prev(c)
								}
								for i := 0; i < charsToDelete; i++ {
									e.Delete(c, false)
								}
							}

							e.InsertString(c, insertText)

							const drawLines = true
							e.FullResetRedraw(c, status, drawLines, false)
							e.redraw.Store(true)
							e.redrawCursor.Store(true)

							status.SetMessage("Completed: " + insertText)
							status.ShowNoTimeout(c, e)
						}
						break
					}
				}
			}

			trimmedLine := e.TrimmedLine()

			endsWithSpecial := len(trimmedLine) > 1 && r == '{' || r == '(' || r == '[' || r == ':'

			// Smart indent if:
			// * the rune to the left is not a blank character or the line ends with {, (, [ or :
			// * and also if it the cursor is not to the very left
			// * and also if this is not a text file or a blank file
			noSmartIndentation := e.NoSmartIndentation()
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
					e.redrawCursor.Store(true)
					e.redraw.Store(true)

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
			e.redrawCursor.Store(true)
			e.redraw.Store(true)

		case pgUpKey: // page up
			h := int(c.H())
			e.redraw.Store(e.ScrollUp(c, status, int(float64(h)*0.9)))
			e.redrawCursor.Store(true)
			if e.AfterLineScreenContents() {
				e.End(c)
			}
			e.drawProgress.Store(true)
			e.drawFuncName.Store(true)

		case pgDnKey: // page down
			h := int(c.H())
			redraw := e.ScrollDown(c, status, int(float64(h)*0.9), h)
			e.redraw.Store(redraw)
			// If redraw is false, the end of file is reached
			if !redraw {
				status.Clear(c, false)
				status.SetMessage(endOfFileMessage)
				status.Show(c, e)
			}
			e.redrawCursor.Store(true)
			if e.AfterLineScreenContents() {
				e.End(c)
			}
			e.drawProgress.Store(true)
			e.drawFuncName.Store(true)

		case "c:25": // ctrl-y

			if e.nanoMode.Load() { // nano: ctrl-y, page up
				h := int(c.H())
				e.redraw.Store(e.ScrollUp(c, status, h))
				e.redrawCursor.Store(true)
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			fallthrough
		case "c:1", homeKey: // ctrl-a, home (or ctrl-y for scrolling up in the st terminal)

			if e.spellCheckMode {
				if addedWord := e.AddCurrentWordToWordList(); addedWord != "" {
					typo, corrected := e.NanoNextTypo(c, status)
					msg := "Added " + addedWord
					if spellChecker != nil && typo != "" {
						msg += ". Found " + typo
						if corrected != "" {
							msg += " which could be " + corrected + "."
						} else {
							msg += "."
						}
					}
					status.SetMessageAfterRedraw(msg)
				}
				break
			}

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys
			justMovedUpOrDown := kh.PrevHas(downArrow, upArrow)
			if e.macro != nil {
				e.Home()
			} else if !justMovedUpOrDown && e.EmptyRightTrimmedLine() && e.SearchTerm() == "" {
				// If at an empty line, go up one line
				e.Up(c, status)
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

			e.redrawCursor.Store(true)
			e.SaveX(true)
		case "c:5", endKey: // ctrl-e, end

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys, or just cut a line with ctrl-x
			justMovedUpOrDown := kh.PrevHas(downArrow, upArrow, "c:24")
			if e.AtEndOfDocument() || e.macro != nil {
				e.End(c)
			} else if !justMovedUpOrDown && e.AfterEndOfLine() && e.SearchTerm() == "" {
				// If we didn't just move here, and are at the end of the line,
				// move down one line and to the end, if not,
				// just move to the end.
				e.Down(c, status)
				e.Home()
			} else {
				e.End(c)
			}
			e.redrawCursor.Store(true)
			e.SaveX(true)
		case "c:4": // ctrl-d, delete
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.Delete(c, e.blockMode)
				e.redraw.Store(true)
			}
			e.redrawCursor.Store(true)
		case "c:29", "c:30": // ctrl-~, insert the current date and time
			if spellCheckFunc, err := e.CommandToFunction(c, tty, status, bookmark, undo, "insertdateandtime"); err == nil { // success
				spellCheckFunc()
			}
		case "c:19": // ctrl-s, save (or step, if in debug mode)
			e.UserSave(c, tty, status)
		case "c:7": // ctrl-g, either go to definition OR jump to matching parent/bracket OR toggle the status bar

			if e.nanoMode.Load() { // nano: ctrl-g, help
				status.ClearAll(c, false)
				const repositionCursorAfterDrawing = false
				e.DrawNanoHelp(c, repositionCursorAfterDrawing)
				e.waitWithRedrawing.Store(true)
				e.redraw.Store(false)
				e.redrawCursor.Store(false)
				messageAfterRedraw = ""
				break
			}

			// If a search is in progress, clear the search first
			if e.searchTerm != "" {
				e.ClearSearch()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
			}

			oldFilename := e.filename
			oldLineIndex := e.LineIndex()

			// Check if we should toggle status bar based on cursor position and file type
			if ProgrammingLanguage(e.mode) && !e.AtOrBeforeStartOfTextScreenLine() {
				if e.JumpToMatching(c) { // jump to matching parenthesis or bracket
					e.redrawCursor.Store(true)
				} else if e.FuncPrefix() != "" && e.GoToDefinition(tty, c, status) { // go to definition
					break
				} else if e.OnIncludeLine() { // go to include
					if includeFilename, jumped := e.GoToInclude(tty, c, status); !jumped {
						status.Clear(c, false)
						status.SetErrorMessage("could not jump to " + includeFilename)
						status.Show(c, e)
						e.redrawCursor.Store(true)
					}
				}
			} else if e.searchTerm != "" && strings.Contains(e.String(), e.searchTerm) {
				// Push a function for how to go back
				backFunctions = append(backFunctions, func() {
					oldFilename := oldFilename
					oldLineIndex := oldLineIndex
					if e.filename != oldFilename {
						// The switch is not strictly needed, since we will probably be in the same file
						e.Switch(c, tty, status, fileLock, oldFilename)
					}
					redraw, _ := e.GoTo(oldLineIndex, c, status)
					e.redraw.Store(redraw)
					e.ClearSearch()
					status.Show(c, e)
				})
			} else if e.JumpToMatching(c) {
				e.redrawCursor.Store(true)
			} else if e.OnIncludeLine() { // Check if we can jump to an #include file, regardless of file mode
				if includeFilename, jumped := e.GoToInclude(tty, c, status); !jumped {
					status.Clear(c, false)
					status.SetErrorMessage("could not jump to " + includeFilename)
					status.Show(c, e)
					e.redrawCursor.Store(true)
				}
			} else if len(backFunctions) > 0 { // Check if we have jumped somewhere and need to jump back
				lastIndex := len(backFunctions) - 1
				// call the function for getting back
				backFunctions[lastIndex]()
				// pop a function from the end of backFunctions
				backFunctions = backFunctions[:lastIndex]
				if len(backFunctions) == 0 {
					// last possibility to jump back
					status.SetMessageAfterRedraw("Loaded " + filepath.Base(e.filename))
				}
			} else {
				// Toggle status bar (for non-programming languages or when at/before start of text), when not on include lines
				status.ClearAll(c, false)
				e.statusMode = !e.statusMode
				if e.statusMode {
					status.ShowFilenameLineColWordCount(c, e)
					e.showColumnLimit = e.wrapWhenTyping
				}
				status.Show(c, e)
			}
			e.redraw.Store(true)

		case "c:21": // ctrl-u to undo

			if e.nanoMode.Load() { // nano: paste after cutting
				e.Paste(c, status, &copyLines, &previousCopyLines, &firstPasteAction, &lastCopyY, &lastPasteY, &lastCutY, kh.PrevIs("c:13"))
				break
			}

			fallthrough // undo behavior
		case "c:26": // ctrl-z to undo (my also background the application, unfortunately)

			// Forget the cut, copy and paste line state
			lastCutY = -1
			lastPasteY = -1
			lastCopyY = -1
			// Try to restore the previous editor state in the undo buffer
			if err := undo.Restore(e); err == nil {
				e.EnableAndPlaceCursor(c)
				e.redrawCursor.Store(true)
				e.redraw.Store(true)
			} else {
				status.SetMessageAfterRedraw("Nothing more to undo")
			}
		case "c:24": // ctrl-x, cut line

			if e.nanoMode.Load() { // nano: ctrl-x, quit

				if e.changed.Load() {
					// Ask the user which filename to save to
					if newFilename, ok := e.UserInput(c, tty, status, "Write to", e.filename, []string{e.filename}, false, e.filename); ok {
						e.filename = newFilename
						e.Save(c, tty)
					} else {
						status.Clear(c, false)
						status.SetMessage("Wrote nothing")
						status.Show(c, e)
					}
				}

				e.quit = true
				break
			}

			// Prepare to cut
			undo.Snapshot(e)

			// First try a single line cut
			if y, multilineCut := e.CutSingleLine(status, bookmark, &lastCutY, &lastCopyY, &lastPasteY, &copyLines, &firstCopyAction); multilineCut { // Multi line cut (add to the clipboard, since it's the second press)
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
				if isDarwin {
					pbcopy(s)
				} else {
					// Place it in the non-primary clipboard
					_ = clip.WriteAll(s, e.primaryClipboard)
				}

				// Delete the corresponding number of lines
				notRegularEditingRightNow.Store(true)
				for range lines {
					e.DeleteLineMoveBookmark(y, bookmark)
				}
				notRegularEditingRightNow.Store(false)

				// No status message is needed for the cut operation, because it's visible that lines are cut
				e.redrawCursor.Store(true)
				e.redraw.Store(true)
			}
			// Go to the end of the current line
			e.End(c)
		case "c:11": // ctrl-k, delete to end of line
			undo.Snapshot(e)
			if e.nanoMode.Load() { // nano: ctrl-k, cut line
				// Prepare to cut
				e.CutSingleLine(status, bookmark, &lastCutY, &lastCopyY, &lastPasteY, &copyLines, &firstCopyAction)
				break
			}

			e.DeleteToEndOfLine(c, status, bookmark, &lastCopyY, &lastPasteY, &lastCutY)

			// Delete extra if the key is held down
			if kh.TwoLastAre("c:11") && kh.AllWithin(200*time.Millisecond) && kh.LastChanged(200*time.Millisecond) {
				if heldDownCtrlKTime.IsZero() {
					heldDownCtrlKTime = time.Now()
				}
				heldDuration := time.Since(heldDownCtrlKTime)
				// 2x slower step up for ctrl-k than for left/right arrow
				steps := int(int64(heldDuration) / int64(delayUntilSpeedUp*2))
				for i := 1; i < steps; i++ {
					e.DeleteToEndOfLine(c, status, bookmark, &lastCopyY, &lastPasteY, &lastCutY)
				}
			} else {
				heldDownCtrlKTime = time.Time{}
			}

		case "c:3", copyKey: // ctrl-c, copy the stripped contents of the current line

			// Stop background processes (like playing music with timidity), if any
			if stopBackgroundProcesses() {
				break // If background processes were stopped, then don't copy text just yet
			}

			if e.nanoMode.Load() { // nano: ctrl-c, report cursor position
				status.ClearAll(c, false)
				status.NanoInfo(c, e)
				break
			}

			// ctrl-c might interrupt the program, but saving at the wrong time might be just as destructive.
			// e.Save(c, tty)

			go func() {

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
					status.Clear(c, false)
					// Pressed for the first time for this line number
					trimmed := strings.TrimSpace(e.Line(y))
					if trimmed != "" {
						// Copy the line to the internal clipboard
						copyLines = []string{trimmed}
						// Copy the line to the clipboard
						s := "Copied 1 line"
						var err error
						if isDarwin {
							err = pbcopy(strings.Join(copyLines, "\n"))
						} else {
							// Place it in the non-primary clipboard
							err = clip.WriteAll(strings.Join(copyLines, "\n"), e.primaryClipboard)
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
					var s string
					if kh.Repeated("c:3", 3) {
						var err error
						s, err = e.FunctionBlock(y)
						if err != nil {
							s = e.Block(y)
						}
					} else {
						s = e.Block(y)
					}
					if s != "" {
						copyLines = strings.Split(s, "\n")
						lineCount := strings.Count(s, "\n")
						// Prepare a status message
						plural := "s"
						if lineCount == 1 {
							plural = ""
						}
						// Place the block of text in the clipboard
						if isDarwin {
							err = pbcopy(s)
						} else {
							// Place it in the non-primary clipboard
							err = clip.WriteAll(s, e.primaryClipboard)
						}
						fmtMsg := "Copied %d line%s from %s"
						if err != nil {
							fmtMsg = "Copied %d line%s from %s to internal buffer"
						}
						status.SetMessage(fmt.Sprintf(fmtMsg, lineCount, plural, filepath.Base(e.filename)))
						status.Show(c, e)
					}
				}

				e.redraw.Store(true)
				e.redrawCursor.Store(true)
			}()

		case "c:22": // ctrl-v, paste

			if e.nanoMode.Load() { // nano: ctrl-v, page down
				h := int(c.H())
				e.redraw.Store(e.ScrollDown(c, status, h, h))
				e.redrawCursor.Store(true)
				if e.AfterLineScreenContents() {
					e.End(c)
				}
				break
			}

			// paste from the portal, clipboard or line buffer. Takes an undo snapshot if text is pasted.
			e.Paste(c, status, &copyLines, &previousCopyLines, &firstPasteAction, &lastCopyY, &lastPasteY, &lastCutY, kh.PrevIs("c:13"))

		case "c:18": // ctrl-r, to open or close a portal. In debug mode, continue running the program.

			if e.nanoMode.Load() { // nano: ctrl-r, insert file
				// Ask the user which filename to insert
				if insertFilename, ok := e.UserInput(c, tty, status, "Insert file", "", []string{e.filename}, false, e.filename); ok {
					err := e.RunCommand(c, tty, status, bookmark, undo, "insertfile", insertFilename)
					if err != nil {
						status.SetError(err)
						status.Show(c, e)
						break
					}
				}

				break
			}

			if e.debugMode {
				e.DebugContinue()
				break
			}

			// Are we in git mode?
			if line := e.CurrentLine(); e.mode == mode.Git && hasAnyPrefixWord(line, gitRebasePrefixes) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetCurrentLine(newLine)
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				break
			}

			// Deal with the portal
			status.ClearAll(c, false)
			if HasPortal() {
				status.SetMessageAfterRedraw("Closing portal")
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
				status.SetMessageAfterRedraw("Opening a portal at " + portal.String())
			}
		case "c:2": // ctrl-b, go back after jumping to a definition, bookmark, unbookmark or jump to bookmark. Toggle breakpoint if in debug mode.

			if e.nanoMode.Load() { // nano: ctrl-b, cursor forward
				e.CursorBackward(c, status)
				break
			}

			// Check if we have jumped to a definition and need to go back
			if len(backFunctions) > 0 {
				lastIndex := len(backFunctions) - 1
				// call the function for getting back
				backFunctions[lastIndex]()
				// pop a function from the end of backFunctions
				backFunctions = backFunctions[:lastIndex]
				if len(backFunctions) == 0 {
					// last possibility to jump back
					status.SetMessageAfterRedraw("Loaded " + filepath.Base(e.filename))
				}
				break
			}

			status.ClearAll(c, false)

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
					e.HideCursorDrawLines(c, true, false, true)
					e.redraw.Store(false)
					// Show the status message
					s := "Jumped to breakpoint at line " + e.LineNumber().String()
					status.SetMessage(s)
				}
			} else {
				status.ClearAll(c, true)
				if bookmark == nil {
					// no bookmark, create a bookmark at the current line
					bookmark = e.pos.Copy()
					// TODO: Modify the statusbar implementation so that extra spaces are not needed here.
					s := "Bookmarked line " + e.LineNumber().String()
					status.SetMessageAfterRedraw("  " + s + "  ")
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
					e.HideCursorDrawLines(c, true, false, true)
					e.redraw.Store(false)
					// Show the status message
					s := "Jumped to bookmark at line " + e.LineNumber().String()
					status.SetMessage(s)
				}
			}
			status.Show(c, e)
			e.redrawCursor.Store(true)
		case "c:10": // ctrl-j, join line
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
				break
			}
			undo.Snapshot(e)
			if e.nanoMode.Load() {
				// Up to 999 times, join the current line with the next, until the next line is empty
				joinCount := 0
				for i := 0; i < 999; i++ {
					if !e.JoinLineWithNext(c, bookmark) {
						break
					}
					joinCount++
				}
				downCounter := 0
				for i := 0; i < joinCount; i++ {
					e.Down(c, status)
					downCounter++
				}
				for i := 0; i < downCounter; i++ {
					e.Up(c, status)
				}
				break
			}

			// The normal join behavior
			e.JoinLineWithNext(c, bookmark)

			// Go to the start of the line when pressing ctrl-j, but only if it is pressed repeatedly
			if kh.Prev() == "c:10" {
				e.GoToStartOfTextLine(c)
			}
		default: // any other key
			keyRunes := []rune(key)
			if len(keyRunes) > 0 && unicode.IsLetter(keyRunes[0]) { // letter
				if keyRunes[0] == 'n' && kh.TwoLastAre("c:14") && kh.PrevWithin(500*time.Millisecond) {
					// Avoid inserting "n" if the user very recently pressed ctrl-n twice
					break
				} else if keyRunes[0] == 'p' && kh.TwoLastAre("c:16") && kh.PrevWithin(500*time.Millisecond) {
					// Avoid inserting "p" if the user very recently pressed ctrl-p twice
					break
				} else if keyRunes[0] == 'l' && kh.TwoLastAre("c:12") && kh.PrevWithin(500*time.Millisecond) {
					// Avoid inserting "l" if the user very recently pressed ctrl-l twice
					break
				} else if keyRunes[0] == 'q' && e.mode == mode.ManPage {
					// If o is used as a man page viewer, exit at the press of "q"
					clearOnQuit.Store(false)
					e.quit = true
					break
				} else if keyRunes[0] == 'q' && !e.nanoMode.Load() && kh.PrevPrev() == "c:27" && kh.Prev() == "," { // <esc> ,q
					// Remove the ","
					e.Backspace(c, bookmark)
					// Quit
					e.quit = true
					break
				} else if keyRunes[0] == 'w' && !e.nanoMode.Load() && kh.PrevPrev() == "c:27" && kh.Prev() == "," { // <esc> ,w
					// Remove the ","
					e.Backspace(c, bookmark)
					// Save the file
					e.UserSave(c, tty, status)
					// Skip the rest
					continue
				}

				undo.Snapshot(e)

				// Type in the letters that were pressed
				for _, r := range keyRunes {
					// Insert a letter. This is what normally happens.
					wrapped := e.InsertRune(c, r)
					if !wrapped {
						e.WriteRune(c)
						e.Next(c)
					}
					e.redraw.Store(true)
				}
			} else if len(keyRunes) > 0 && unicode.IsGraphic(keyRunes[0]) { // any other key that can be drawn
				undo.Snapshot(e)
				e.redraw.Store(true)

				// Place *something*
				r := keyRunes[0]
				switch r {
				case 160:
					// This is a nonbreaking space that may be inserted with altgr+space that is HORRIBLE.
					// Set r to a regular space instead.
					r = ' '
				case '}', ']', ')':
					// "smart dedent"

					// Normally, dedent once, but there are exceptions

					noContentHereAlready := len(e.TrimmedLine()) == 0
					leadingWhitespace := e.LeadingWhitespace()
					nextLineContents := e.Line(e.DataY() + 1)

					currentX := e.pos.sx

					foundCurlyBracketBelow := currentX-1 == strings.Index(nextLineContents, "}")
					foundSquareBracketBelow := currentX-1 == strings.Index(nextLineContents, "]")
					foundParenthesisBelow := currentX-1 == strings.Index(nextLineContents, ")")

					noDedent := foundCurlyBracketBelow || foundSquareBracketBelow || foundParenthesisBelow

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
				if !wrapped {
					// Move to the next position
					e.Next(c)
				}
				e.redrawCursor.Store(true)
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

		// Display the ctrl-o menu if esc was pressed 4 times. Do not react if space is pressed.
		if !e.nanoMode.Load() && kh.Repeated("c:27", 4-1) { // esc pressed 4 times (minus the one that was added just now)
			backFunctions = make([]func(), 0)
			status.ClearAll(c, false)
			undo.Snapshot(e)
			undoBackup := undo
			selectedIndex, _ := e.CommandMenu(c, tty, status, bookmark, undo, lastCommandMenuIndex, forceFlag, fileLock)
			lastCommandMenuIndex = selectedIndex
			undo = undoBackup
			// Reset the key history next iteration
			clearKeyHistory = true
		}

		// Clear status line, if needed
		if (e.statusMode || e.blockMode) && e.redrawCursor.Load() {
			status.ClearAll(c, false)
		}

		const arrowKeyHighlightTime = 1200 * time.Millisecond

		// Draw and/or redraw everything, with slightly different behavior over ssh
		justMovedUpOrDown := kh.PrevIsWithin(arrowKeyHighlightTime, downArrow, upArrow)
		e.RedrawAtEndOfKeyLoop(c, status, justMovedUpOrDown, true)

		notEmptyLine := !e.EmptyLine()

		if notEmptyLine && ProgrammingLanguage(e.mode) {
			e.drawFuncName.Store(true)
			c.HideCursorAndDraw()
		}

		if (e.highlightCurrentLine || e.highlightCurrentText) && !e.statusMode && notEmptyLine && !e.debugMode {
			// When not moving up or down, turn off the text highlight after arrowHighlightTime
			if status.messageAfterRedraw == "" {
				go func() {
					thisID := highlightTimerCounter.Add(1)
					time.Sleep(arrowKeyHighlightTime)
					if thisID < highlightTimerCounter.Load() { // only the freshest ID should be active
						return
					}
					highlightTimerMut.Lock()
					defer highlightTimerMut.Unlock()
					justMovedUpOrDownOrLeftOrRight := kh.PrevIsWithin(arrowKeyHighlightTime, downArrow, upArrow)
					if e.waitWithRedrawing.Load() {
						e.waitWithRedrawing.Store(false)
					} else if !justMovedUpOrDownOrLeftOrRight && !notRegularEditingRightNow.Load() {
						e.redraw.Store(true)
						e.redrawCursor.Store(true)
						e.RedrawAtEndOfKeyLoop(c, status, false, true)
						e.redraw.Store(false)
						e.redrawCursor.Store(false)
					}
				}()
			}
		}

		// Also draw the watches, if debug mode is enabled // and a debug session is in progress
		if e.debugMode {
			const repositionCursor = false
			e.DrawWatches(c, repositionCursor)
			e.DrawRegisters(c, repositionCursor)
			e.DrawGDBOutput(c, repositionCursor)
			e.DrawInstructions(c, repositionCursor)
			e.DrawFlags(c, repositionCursor)
		}

		// Repositions the cursor
		e.EnableAndPlaceCursor(c)

	} // end of main loop

	var closeLocksWaitGroup sync.WaitGroup
	e.CloseLocksAndLocationHistory(absFilename, lockTimestamp, forceFlag, &closeLocksWaitGroup)

	// Clear the colors
	vt.SetNoColor()

	// Quit everything that has to do with the terminal
	if clearOnQuit.Load() {
		vt.Clear()
		vt.Close()
	} else {
		// Clear all status bar messages
		status.ClearAll(c, false)
		// Redraw
		c.Draw()
	}

	// Make sure to enable the cursor again
	vt.ShowCursor(true)

	// Wait for locks to be closed and location history to be written
	closeLocksWaitGroup.Wait()

	// stop background processes, such as "timidity" if music is playing
	stopBackgroundProcesses()

	// All done
	return "", e.stopParentOnQuit, nil
}
