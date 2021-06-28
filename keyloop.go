package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/xyproto/env"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

const (
	defaultUndoSize = 8192 // number of undo actions possible to store in the circular buffer
)

var (
	// Circular undo buffer with room for N actions
	undo = NewUndo(defaultUndoSize)
)

// Loop will set up and run the main loop of the editor
// a *vt100.TTY struct
// a filename to open
// a LineNumber (may be 0 or -1)
// a forceFlag for if the file should be force opened
// If an error and "true" is returned, it is a quit message to the user, and not an error.
// If an error and "false" is returned, it is an error.
func Loop(tty *vt100.TTY, filename string, lineNumber LineNumber, colNumber ColNumber, forceFlag bool, useTheme Theme) (userMessage string, err error) {

	// Create a Canvas for drawing onto the terminal
	vt100.Init()
	c := vt100.NewCanvas()
	c.ShowCursor()

	var (
		statusDuration = 2700 * time.Millisecond

		copyLines         []string  // for the cut/copy/paste functionality
		previousCopyLines []string  // for checking if a paste is the same as last time
		bookmark          *Position // for the bookmark/jump functionality
		statusMode        bool      // if information should be shown at the bottom

		firstPasteAction = true
		firstCopyAction  = true

		lastCopyY  LineIndex = -1 // used for keeping track if ctrl-c is pressed twice on the same line
		lastPasteY LineIndex = -1 // used for keeping track if ctrl-v is pressed twice on the same line
		lastCutY   LineIndex = -1 // used for keeping track if ctrl-x is pressed twice on the same line

		previousKey string // keep track of the previous key press

		lastCommandMenuIndex int // for the command menu

		key string // for the main loop

		jsonFormatToggle bool // for toggling indentation or not when pressing ctrl-w for JSON

		markdownSkipExport = true // for skipping the first ctrl-space keypress
	)

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e, statusMessage, err := NewEditor(tty, c, filename, lineNumber, colNumber, useTheme)
	if err != nil {
		return "", err
	}

	// Find the absolute path to this filename
	absFilename, err := e.AbsFilename()
	if err != nil {
		// This should never happen, just use the given filename
		absFilename = e.filename
	}

	// Prepare a status bar
	status := NewStatusBar(defaultStatusForeground, defaultStatusBackground, defaultStatusErrorForeground, defaultStatusErrorBackground, e, statusDuration)

	// Modify the status bar theme if editing git
	if e.mode == modeGit {
		status.fg = vt100.LightBlue
		status.bg = vt100.BackgroundDefault
	}

	// Use the selected theme
	switch useTheme {
	case redBlackTheme:
		e.setRedBlackTheme()
		e.SetSyntaxHighlight(true)
	case lightTheme:
		e.setLightTheme()
		e.SetSyntaxHighlight(true)
	case defaultTheme:
		fallthrough
	default:
	}

	// Respect the NO_COLOR environment variable
	e.respectNoColorEnvironmentVariable()
	status.respectNoColorEnvironmentVariable()

	// Terminal resize handler
	e.SetUpResizeHandler(c, tty, status)

	// ctrl-c handler
	e.SetUpTerminateHandler(c, tty, status)

	tty.SetTimeout(2 * time.Millisecond)

	previousX := 1
	previousY := 1

	// Create a LockKeeper for keeping track of which files are being edited
	lk := NewLockKeeper(defaultLockFile)

	var (
		canUseLocks   = true
		lockTimestamp time.Time
	)

	// If the lock keeper does not have an overview already, that's fine. Ignore errors from lk.Load().
	if err := lk.Load(); err != nil {
		// Could not load an existing lock overview, this might be the first run? Try saving.
		if err := lk.Save(); err != nil {
			// Could not save a lock overview. Can not use locks.
			canUseLocks = false
		}
	}

	if canUseLocks {
		// Check if the lock should be forced (also force when running git commit, becase it is likely that o was killed in that case)
		if forceFlag || filepath.Base(absFilename) == "COMMIT_EDITMSG" || env.Bool("O_FORCE") {
			// Lock and save, regardless of what the previous status is
			lk.Lock(absFilename)
			// TODO: If the file was already marked as locked, this is not strictly needed? The timestamp might be modified, though.
			lk.Save()
		} else {
			// Lock the current file, if it's not already locked
			if err := lk.Lock(absFilename); err != nil {
				return fmt.Sprintf("Locked by another (possibly dead) instance of this editor.\nTry: o -f %s", filepath.Base(absFilename)), errors.New(absFilename + " is locked")
			}
			// Immediately save the lock file as a signal to other instances of the editor
			lk.Save()
		}
		lockTimestamp = lk.GetTimestamp(absFilename)

		// Set up a catch for panics, so that the current file can be unlocked
		defer func() {
			if x := recover(); x != nil {
				// Unlock and save the lock file
				lk.Unlock(absFilename)
				lk.Save()

				// Save the current file. The assumption is that it's better than not saving, if something crashes.
				// TODO: Save to a crash file, then let the editor discover this when it starts.
				e.Save(c, tty)

				// Output the error message
				quitMessage(tty, fmt.Sprintf("%v", x))
			}
		}()
	}

	// Do a full reset and redraw, but without the statusbar (set to nil)
	e.FullResetRedraw(c, nil, false)

	// Draw the editor lines, respect the offset (true) and redraw (true)
	e.DrawLines(c, true, true)
	e.redraw = false

	// Display the status message
	status.SetMessage(statusMessage)
	status.Show(c, e)

	// Redraw the cursor, if needed
	if e.redrawCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		previousX = x
		previousY = y
		vt100.SetXY(uint(x), uint(y))
		e.redrawCursor = false
	}

	// This is the main loop for the editor
	for !e.quit {

		// Read the next key
		key = tty.String()

		switch key {
		case "c:17": // ctrl-q, quit
			e.quit = true
		case "c:23": // ctrl-w, format (or if in git mode, cycle interactive rebase keywords)
			undo.Snapshot(e)

			// Clear the search term
			e.ClearSearchTerm()

			// Cycle git rebase keywords
			if line := e.CurrentLine(); e.mode == modeGit && hasAnyPrefixWord(line, gitRebasePrefixes) {
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

			if e.mode == modeMarkdown {
				e.ToggleCheckboxCurrentLine()
				break
			}

			e.formatCode(c, tty, status, &jsonFormatToggle)

			// Move the cursor if after the end of the line
			if e.AtOrAfterEndOfLine() {
				e.End(c)
			}
		case "c:6": // ctrl-f, search for a string
			e.SearchMode(c, status, tty, true)
		case "c:0": // ctrl-space, build source code to executable, convert to PDF or write to PNG, depending on the mode

			if e.Empty() {
				// Empty file, nothing to build
				status.ClearAll(c)
				status.SetErrorMessage("Nothing to build")
				status.Show(c, e)
				break
			}

			// Save the current file, but only if it has changed
			if e.changed {
				if err := e.Save(c, tty); err != nil {
					status.ClearAll(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					break
				}
			}

			// Clear the current search term
			e.ClearSearchTerm()

			// Press ctrl-space twice the first time the PDF should be exported to Markdown,
			// to avvoid the first accidental ctrl-space key press.

			// Build or export the current file
			var (
				statusMessage   string
				performedAction bool
				compiled        bool
			)

			if e.mode == modeMarkdown && markdownSkipExport {
				// Do nothing, but don't skip the next one
				markdownSkipExport = false
				// } else if e.mode == modeMarkdown && !markdownSkipExport{
				// statusMessage, performedAction, compiled = e.BuildOrExport(c, status, e.filename)
			} else {
				statusMessage, performedAction, compiled = e.BuildOrExport(c, tty, status, e.filename)
			}

			//logf("status message %s performed action %v compiled %v filename %s\n", statusMessage, performedAction, compiled, e.filename)

			// Could an action be performed for this file extension?
			if !performedAction {
				status.ClearAll(c)
				// Building this file extension is not implemented yet.
				// Just display the current time and word count.
				// TODO: status.ClearAll() should have cleared the status bar first, but this is not always true,
				//       which is why the message is hackily surrounded by spaces. Fix.
				statusMessage := fmt.Sprintf("    %d words, %s    ", e.WordCount(), time.Now().Format("15:04")) // HH:MM
				status.SetMessage(statusMessage)
				status.Show(c, e)
			} else if performedAction && !compiled {
				status.ClearAll(c)
				// Performed an action, but it did not work out
				if statusMessage != "" {
					status.SetErrorMessage(statusMessage)
				} else {
					// This should never happen, failed compilations should return a message
					status.SetErrorMessage("Compilation failed")
				}
				status.ShowNoTimeout(c, e)
			} else if performedAction && compiled {
				// Everything worked out
				if statusMessage != "" {
					// Got a status message (this may not be the case for build/export processes running in the background)
					// NOTE: Do not clear the status message first here!
					status.SetMessage(statusMessage)
					status.ShowNoTimeout(c, e)
				}
			}
		case "c:20": // ctrl-t, render to PDF
			// If in a C++ header file, switch to the corresponding
			// C++ source file, and the other way around.

			// Save the current file, but only if it has changed
			if e.changed {
				if err := e.Save(c, tty); err != nil {
					status.ClearAll(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					break
				}
			}

			e.redrawCursor = true

			// If this is a C++ source file, try finding and opening the corresponding header file
			if hasS([]string{".cpp", ".cc", ".c", ".cxx"}, filepath.Ext(e.filename)) {
				// Check if there is a corresponding header file
				if absFilename, err := e.AbsFilename(); err == nil { // no error
					headerExtensions := []string{".h", ".hpp"}
					if headerFilename, err := ExtFileSearch(absFilename, headerExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
						// Switch to another file (without forcing it)
						e.Switch(c, tty, status, lk, headerFilename, false)
					}
				}
				break
			}

			// If this is a header file, present a menu option for open the corresponding source file
			if hasS([]string{".h", ".hpp"}, filepath.Ext(e.filename)) {
				// Check if there is a corresponding header file
				if absFilename, err := e.AbsFilename(); err == nil { // no error
					sourceExtensions := []string{".c", ".cpp", ".cxx", ".cc"}
					if headerFilename, err := ExtFileSearch(absFilename, sourceExtensions, fileSearchMaxTime); err == nil && headerFilename != "" { // no error
						// Switch to another file (without forcing it)
						e.Switch(c, tty, status, lk, headerFilename, false)
					}
				}
				break
			}

			// Save the current text to .pdf directly (without using pandoc)

			// Write to PDF in a goroutine
			go func() {

				pdfFilename := strings.Replace(filepath.Base(e.filename), ".", "_", -1) + ".pdf"

				// Show a status message while writing
				status.SetMessage("Writing " + pdfFilename + "...")
				status.ShowNoTimeout(c, e)

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
				status.Show(c, e)
			}()
		case "c:28": // ctrl-\, toggle comment for this block
			undo.Snapshot(e)
			e.ToggleCommentBlock(c)
			e.redraw = true
			e.redrawCursor = true
		case "c:15": // ctrl-o, launch the command menu
			status.ClearAll(c)
			undo.Snapshot(e)
			undoBackup := undo
			lastCommandMenuIndex = e.CommandMenu(c, tty, status, undo, lastCommandMenuIndex, forceFlag, lk)
			undo = undoBackup
			if e.AfterEndOfLine() {
				e.End(c)
			}
		case "c:7": // ctrl-g, status mode
			statusMode = !statusMode
			if statusMode {
				status.ShowLineColWordCount(c, e, e.filename)
			} else {
				status.ClearAll(c)
			}
		case "←": // left arrow
			// movement if there is horizontal scrolling
			if e.pos.offsetX > 0 {
				if e.pos.sx > 0 {
					// Move one step left
					if e.TabToTheLeft() {
						e.pos.sx -= e.tabsSpaces.perTab
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
					e.pos.sx -= e.tabsSpaces.perTab
				} else {
					e.pos.sx--
				}
				e.SaveX(true)
			} else if e.DataY() > 0 {
				// no scrolling or movement to the left going on
				e.Up(c, status)
				e.End(c)
				//e.redraw = true
			} // else at the start of the document
			e.redrawCursor = true
			// Workaround for Konsole
			if e.pos.sx <= 2 {
				// Konsole prints "2H" here, but
				// no other terminal emulator does that
				e.redraw = true
			}
		case "→": // right arrow
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
			// Move the screen cursor

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
			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to next match
				wrap := true
				forward := true
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					if wrap {
						status.SetMessage(e.SearchTerm() + " not found")
					} else {
						status.SetMessage(e.SearchTerm() + " not found from here")
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
		case "c:16": // ctrl-p, scroll up or jump to the previous match, using the sticky search term
			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to previous match
				wrap := true
				forward := false
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					if wrap {
						status.SetMessage(e.SearchTerm() + " not found")
					} else {
						status.SetMessage(e.SearchTerm() + " not found from here")
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
			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1
			// Do a full clear and redraw + clear search term
			e.FullResetRedraw(c, status, true)
		case " ": // space
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

			// Modify the paste double-keypress detection to allow for a manual return before pasting the rest
			if lastPasteY != -1 && previousKey != "c:13" {
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
			if e.mode == modeMarkdown || e.mode == modeText || e.mode == modeBlank {
				indent = false
			}

			if trimmedLine == "private:" || trimmedLine == "protected:" || trimmedLine == "public:" {
				// De-indent the current line before moving on to the next
				e.SetCurrentLine(trimmedLine)
				leadingWhitespace = currentLeadingWhitespace
			}

			//onlyOneLine := e.AtFirstLineOfDocument() && e.AtOrAfterLastLineOfDocument()
			//middleOfText := !e.AtOrBeforeStartOfTextLine() && !e.AtOrAfterEndOfLine()

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
				//e.Home()
				if scrollBack {
					e.pos.SetX(c, 0)
				}
			}

			if indent && len(leadingWhitespace) > 0 {
				// If the leading whitespace starts with a tab and ends with a space, remove the final space
				if strings.HasPrefix(leadingWhitespace, "\t") && strings.HasSuffix(leadingWhitespace, " ") {
					leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
					//logf("cleaned leading whitespace: %v\n", []rune(leadingWhitespace))
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
			//e.TrimRight(e.DataY())
			// Just clear the search term, if there is an active search
			if len(e.SearchTerm()) > 0 {
				e.ClearSearchTerm()
				e.redraw = true
				e.redrawCursor = true
				// Don't break, continue to delete to the left after clearing the search
				//break
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
			} else if Spaces(e.mode) && (e.EmptyLine() || e.AtStartOfTheLine()) && len(e.LeadingWhitespace()) >= e.tabsSpaces.perTab {
				// Delete several spaces
				for i := 0; i < e.tabsSpaces.perTab; i++ {
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
		case "c:9": // tab
			y := int(e.DataY())
			r := e.Rune()
			leftRune := e.LeftRune()
			ext := filepath.Ext(e.filename)

			// Tab completion of words for Go
			if word := e.LettersBeforeCursor(); e.mode != modeBlank && leftRune != '.' && !unicode.IsLetter(r) && len(word) > 0 {
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
			} else if word := e.LettersOrDotBeforeCursor(); e.mode != modeBlank && leftRune == '.' && !unicode.IsLetter(r) && len(word) > 0 {
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
			//emptyLine := len(trimmedLine) == 0
			//almostEmptyLine := len(trimmedLine) <= 1

			// Check if a line that is more than just a '{', '(', '[' or ':' ends with one of those
			endsWithSpecial := len(trimmedLine) > 1 && r == '{' || r == '(' || r == '[' || r == ':'

			// Smart indent if:
			// * the rune to the left is not a blank character or the line ends with {, (, [ or :
			// * and also if it the cursor is not to the very left
			// * and also if this is not a text file or a blank file
			if (!unicode.IsSpace(leftRune) || endsWithSpecial) && e.pos.sx > 0 && e.mode != modeBlank {
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
						oneIndentation    string
					)

					// TODO: Don't switch on mode here, but check e.spacesPerTab
					//       and/or introduce a setting just for spaces vs tabs
					switch e.mode {
					case modeShell, modePython, modeCMake, modeConfig:
						// If this is a shell script, use 2 spaces (or however many spaces are defined in e.spacesPerTab)
						oneIndentation = strings.Repeat(" ", e.tabsSpaces.perTab)
					default:
						// For anything else, use real tabs
						oneIndentation = "\t"
					}

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
			switch e.mode {
			case modeShell, modePython, modeCMake, modeConfig:
				for i := 0; i < e.tabsSpaces.perTab; i++ {
					e.InsertRune(c, ' ')
					// Write the spaces that represent the tab to the canvas
					e.WriteTab(c)
					// Move to the next position
					e.Next(c)
				}
			default:
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
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑"
			// If at an empty line, go up one line
			if !justMovedUpOrDown && e.EmptyRightTrimmedLine() && e.SearchTerm() == "" {
				e.Up(c, status)
				//e.GoToStartOfTextLine()
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

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys, or just cut a line with ctrl-x
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑" || previousKey == "c:24"
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
		case "c:30": // ctrl-~, jump to matching parenthesis or curly bracket
			r := e.Rune()

			if e.AfterEndOfLine() {
				e.Prev(c)
				r = e.Rune()
			}

			// Find which opening and closing parenthesis/curly brackets to look for
			opening, closing := rune(0), rune(0)
			switch r {
			case '(', ')':
				opening = '('
				closing = ')'
			case '{', '}':
				opening = '{'
				closing = '}'
			case '[', ']':
				opening = '['
				closing = ']'
			}

			if opening == rune(0) {
				status.Clear(c)
				status.SetMessage("No matching (, ), [, ], { or }")
				status.Show(c, e)
				break
			}

			// Search either forwards or backwards to find a matching rune
			switch r {
			case '(', '{', '[':
				parcount := 0
				for !e.AtOrAfterEndOfDocument() {
					if r := e.Rune(); r == closing {
						if parcount == 1 {
							// FOUND, STOP
							break
						} else {
							parcount--
						}
					} else if r == opening {
						parcount++
					}
					e.Next(c)
				}
			case ')', '}', ']':
				parcount := 0
				for !e.AtStartOfDocument() {
					if r := e.Rune(); r == opening {
						if parcount == 1 {
							// FOUND, STOP
							break
						} else {
							parcount--
						}
					} else if r == closing {
						parcount++
					}
					e.Prev(c)
				}
			}

			e.redrawCursor = true
			e.redraw = true
		case "c:19": // ctrl-s, save
			e.UserSave(c, tty, status)
		case "c:21", "c:26": // ctrl-u or ctrl-z, undo (ctrl-z may background the application)
			// Forget the cut, copy and paste line state
			lastCutY = -1
			lastPasteY = -1
			lastCopyY = -1

			// Try to restore the previous editor state in the undo buffer
			if err := undo.Restore(e); err == nil {
				//c.Draw()
				x := e.pos.ScreenX()
				y := e.pos.ScreenY()
				vt100.SetXY(uint(x), uint(y))
				e.redrawCursor = true
				e.redraw = true
			} else {
				status.SetMessage("Nothing more to undo")
				status.Show(c, e)
			}
		case "c:12": // ctrl-l, go to line number
			status.ClearAll(c)
			status.SetMessage("Go to line number:")
			status.ShowNoTimeout(c, e)
			lns := ""
			cancel := false
			doneCollectingDigits := false
			for !doneCollectingDigits {
				numkey := tty.String()
				switch numkey {
				case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // 0 .. 9
					lns += numkey // string('0' + (numkey - 48))
					status.SetMessage("Go to line number: " + lns)
					status.ShowNoTimeout(c, e)
				case "c:8", "c:127": // ctrl-h or backspace
					if len(lns) > 0 {
						lns = lns[:len(lns)-1]
						status.SetMessage("Go to line number: " + lns)
						status.ShowNoTimeout(c, e)
					}
				case "↑", "↓": // up arrow or down arrow
					fallthrough
				case "c:27", "c:17": // esc or ctrl-q
					cancel = true
					lns = ""
					fallthrough
				case "c:13": // return
					doneCollectingDigits = true
				}
			}
			if !cancel {
				e.ClearSearchTerm()
			}
			status.ClearAll(c)
			if lns == "" && !cancel {
				if e.DataY() > 0 {
					// If not at the top, go to the first line (by line number, not by index)
					e.redraw = e.GoToLineNumber(1, c, status, true)
				} else {
					// Go to the last line (by line number, not by index, e.Len() returns an index which is why there is no -1)
					e.redraw = e.GoToLineNumber(LineNumber(e.Len()), c, status, true)
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
				ClosePortal()

				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1
				// Copy the line internally
				copyLines = []string{line}

				// Copy the line to the clipboard
				err = clipboard.WriteAll(line)
				if err == nil {
					// no issue
				} else if firstCopyAction {
					missingUtility := false

					if env.Has("DISPLAY") { // X11
						if which("xclip") == "" {
							status.SetErrorMessage("The xclip utility is missing!")
							missingUtility = true
						}
					} else {
						if which("wl-copy") == "" {
							status.SetErrorMessage("The wl-copy utility (from wl-clipboard) is missing!")
							missingUtility = true
						}
					}

					// TODO
					_ = missingUtility
				}

				// Delete the line
				e.DeleteLineMoveBookmark(y, bookmark)
			} else { // Multi line cut (add to the clipboard, since it's the second press)
				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1

				// Also close the portal, if any
				ClosePortal()

				s := e.Block(y)
				lines := strings.Split(s, "\n")
				if len(lines) < 1 {
					// Need at least 1 line to be able to cut "the rest" after the first line has been cut
					break
				}
				copyLines = append(copyLines, lines...)
				s = strings.Join(copyLines, "\n")
				// Place the block of text in the clipboard
				_ = clipboard.WriteAll(s)
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

			existingBookmarkAfterThisLine := bookmark != nil && bookmark.LineNumber() > e.LineNumber()
			removedLine := false

			e.DeleteRestOfLine()
			if e.EmptyRightTrimmedLine() {
				// Deleting the rest of the line cleared this line,
				// so just remove it.
				e.DeleteCurrentLineMoveBookmark(bookmark)
				// Then go to the end of the line, if needed
				if e.AfterEndOfLine() {
					e.End(c)
				}
				removedLine = true
			}

			// TODO: Is this one needed/useful?
			vt100.Do("Erase End of Line")

			// Move the bookmark one line up, if a line was removed before that position
			if existingBookmarkAfterThisLine && removedLine {
				bookmark.DecY()
			}

			e.redraw = true
			e.redrawCursor = true
		case "c:3": // ctrl-c, copy the stripped contents of the current line
			y := e.DataY()

			// Forget the cut and paste line state
			lastCutY = -1
			lastPasteY = -1

			// check if this operation is done on the same line as last time
			singleLineCopy := lastCopyY != y
			lastCopyY = y

			// close the portal, if any
			closedPortal := ClosePortal() == nil

			if singleLineCopy { // Single line copy
				status.Clear(c)
				// Pressed for the first time for this line number
				trimmed := strings.TrimSpace(e.Line(y))
				if trimmed != "" {
					// Copy the line to the internal clipboard
					copyLines = []string{trimmed}
					// Copy the line to the clipboard
					s := "Copied 1 line"
					if err := clipboard.WriteAll(strings.Join(copyLines, "\n")); err == nil { // OK
						// The copy operation worked out, using the clipboard
						s += " from the clipboard"
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
					err := clipboard.WriteAll(s)
					if err != nil {
						status.SetMessage(fmt.Sprintf("Copied %d line%s", lineCount, plural))
					} else {
						status.SetMessage(fmt.Sprintf("Copied %d line%s (clipboard)", lineCount, plural))
					}
					status.Show(c, e)
				}
			}
		case "c:22": // ctrl-v, paste

			// Save the file right before pasting, just in case wl-paste stops
			e.UserSave(c, tty, status)

			var (
				gotLineFromPortal bool
				line              string
			)

			if portal, err := LoadPortal(); err == nil { // no error
				line, err = portal.PopLine(false)
				status.Clear(c)
				if err != nil {
					// status.SetErrorMessage("Could not copy text through the portal.")
					status.SetErrorMessage(err.Error())
					ClosePortal()
				} else {
					status.SetMessage(fmt.Sprintf("Using portal at %s\n", portal))
					gotLineFromPortal = true
				}
				status.Show(c, e)
			}
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

			// This may only work for the same user, and not with sudo/su

			// Try fetching the lines from the clipboard first
			s, err := clipboard.ReadAll()
			if err == nil { // no error

				// Make the replacements, then split the text into lines and store it in "copyLines"
				copyLines = strings.Split(opinionatedStringReplacer.Replace(s), "\n")

				// Note that control characters are not replaced, they are just not printed.
			} else if firstPasteAction {
				missingUtility := false

				status.Clear(c)

				if env.Has("DISPLAY") { // X11
					if which("xclip") == "" {
						status.SetErrorMessage("The xclip utility is missing!")
						missingUtility = true
					}
				} else {
					if which("wl-paste") == "" {
						status.SetErrorMessage("The wl-paste utility (from wl-clipboard) is missing!")
						missingUtility = true
					}
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

				if previousKey != "c:13" {
					// Start by pasting (and overwriting) an untrimmed version of this line,
					// if the previous key was not return.
					e.SetLine(y, copyLines[0])
				} else if e.EmptyRightTrimmedLine() {
					skipFirstLineInsert = true
				}

				// The paste the rest of the lines, also untrimmed
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
		case "c:18": // ctrl-r, to open or close a portal

			// Are we in git mode?
			if line := e.CurrentLine(); e.mode == modeGit && hasAnyPrefixWord(line, gitRebasePrefixes) {
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
				ClosePortal()
			} else {
				portal, err := e.NewPortal()
				if err != nil {
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					break
				}
				if err := portal.Save(); err != nil {
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					break
				}
				status.SetMessage("Opening a portal at " + portal.String())
			}
			status.Show(c, e)
		case "c:2": // ctrl-b, bookmark, unbookmark or jump to bookmark
			status.Clear(c)
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
				// Do the redraw manually before showing the status message
				e.DrawLines(c, true, false)
				e.redraw = false
				// Show the status message
				s := "Jumped to bookmark at line " + e.LineNumber().String()
				status.SetMessage(s)
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
			//panic(fmt.Sprintf("PRESSED KEY: %v", []rune(key)))
			if len([]rune(key)) > 0 && unicode.IsLetter([]rune(key)[0]) { // letter

				undo.Snapshot(e)

				// Type the letter that was pressed
				if len([]rune(key)) > 0 {
					// Insert a letter. This is what normally happens.
					wrapped := e.InsertRune(c, []rune(key)[0])
					if !wrapped {
						e.WriteRune(c)
						e.Next(c)
					}
					e.redraw = true
				}
			} else if len([]rune(key)) > 0 && unicode.IsGraphic([]rune(key)[0]) { // any other key that can be drawn
				undo.Snapshot(e)
				e.redraw = true

				// Place *something*
				r := []rune(key)[0]

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

					//noDedent := similarLineBelow

					// Okay, dedent this line by 1 indendation, if possible
					if !noDedent && e.pos.sx > 0 && len(leadingWhitespace) > 0 && noContentHereAlready {
						newLeadingWhitespace := leadingWhitespace
						if strings.HasSuffix(leadingWhitespace, "\t") {
							newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
							e.pos.sx -= e.tabsSpaces.perTab
						} else if strings.HasSuffix(leadingWhitespace, strings.Repeat(" ", e.tabsSpaces.perTab)) {
							newLeadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-e.tabsSpaces.perTab]
							e.pos.sx -= e.tabsSpaces.perTab
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
		previousKey = key
		// Clear status, if needed
		if statusMode && e.redrawCursor {
			status.ClearAll(c)
		}
		// Redraw, if needed
		if e.redraw {
			// Draw the editor lines on the canvas, respecting the offset
			e.DrawLines(c, true, false)
			e.redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		// Drawing status messages should come after redrawing, but before cursor positioning
		if statusMode {
			status.ShowLineColWordCount(c, e, e.filename)
		} else if status.IsError() {
			// Show the status message
			status.Show(c, e)
		}
		// Position the cursor
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		if e.redrawCursor || x != previousX || y != previousY {
			vt100.SetXY(uint(x), uint(y))
			e.redrawCursor = false
		}
		previousX = x
		previousY = y

	} // end of main loop

	if canUseLocks {
		// Start by loading the lock overview, just in case something has happened in the mean time
		lk.Load()

		// Check if the lock is unchanged
		fileLockTimestamp := lk.GetTimestamp(absFilename)
		lockUnchanged := lockTimestamp == fileLockTimestamp

		// TODO: If the stored timestamp is older than uptime, unlock and save the lock overview

		//var notime time.Time

		if !forceFlag || lockUnchanged {
			// If the file has not been locked externally since this instance of the editor was loaded, don't
			// Unlock the current file and save the lock overview. Ignore errors because they are not critical.
			lk.Unlock(absFilename)
			lk.Save()
		}
	}

	// Save the current location in the location history and write it to file
	e.SaveLocation(absFilename, e.locationHistory)

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
	return "", nil
}
