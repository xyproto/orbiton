package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/clip"
	"github.com/xyproto/files"
	"github.com/xyproto/vt"
)

const commandTimeout = 10 * time.Second

// CommandToFunction takes an editor command as a string (with optional arguments) and returns a function that
// takes no arguments and performs the suggested action, like "save". Some functions may take an undo snapshot first.
func (e *Editor) CommandToFunction(c *vt.Canvas, tty *vt.TTY, status *StatusBar, bookmark *Position, undo *Undo, args ...string) (func(), error) {
	if len(args) == 0 {
		return nil, errors.New("no command given")
	}

	trimmedCommand := strings.TrimPrefix(strings.TrimSpace(args[0]), ":")

	if strings.HasPrefix(trimmedCommand, "!") {
		return func() {
			cmd := exec.Command(trimmedCommand[1:])
			if len(args) > 1 {
				cmd.Args = args[1:]
			}

			// Now run the cmd with the current block of lines as input
			stdin, err := cmd.StdinPipe()
			if err != nil {
				status.Clear(c, false)
				status.SetError(err)
				status.Show(c, e)
				return
			}
			go func() {
				defer stdin.Close()
				io.WriteString(stdin, e.Block(e.LineIndex()))
			}()

			// Gather the output in the same way as CombinedOutput and Run
			var buf bytes.Buffer
			cmd.Stdout = &buf
			cmd.Stderr = &buf
			err = cmd.Start()
			if err != nil {
				status.Clear(c, false)
				status.SetError(err)
				status.Show(c, e)
				return
			}

			outputString := ""

			// Create a completion channel, thanks
			// https://medium.com/@vCabbage/go-timeout-commands-with-os-exec-commandcontext-ba0c861ed738
			done := make(chan error)
			go func() { done <- cmd.Wait() }()

			// Start a timer
			timeout := time.After(commandTimeout)

			// Check if the timeout channel or done channel receives something first
			select {
			case <-timeout:
				cmd.Process.Kill()
				status.Clear(c, false)
				status.SetErrorMessage("command timed out")
				status.Show(c, e)
				return
			case err := <-done:
				outputString = buf.String()
				if err != nil {
					status.Clear(c, false)
					status.SetErrorMessage(cmd.String() + ": " + err.Error())
					status.Show(c, e)
					return
				}
			}

			if outputString == "" {
				status.Clear(c, false)
				status.SetErrorMessage("no output")
				status.Show(c, e)
				return
			}

			undo.Snapshot(e)
			e.ReplaceBlock(c, status, bookmark, outputString)
		}, nil
	}

	// Argument checks, remember to use all available aliases
	switch trimmedCommand {
	case "if", "i", "insertfile", "insert", "insertf":
		if len(args) != 2 {
			return nil, fmt.Errorf("%s requires a filename as the second argument", trimmedCommand)
		}
	default:
		if len(args) != 1 {
			return nil, fmt.Errorf("%s takes no arguments", args[0])
		}
	}

	const (
		nothing = iota
		build
		copyall
		copymark
		copy200
		gobacktofunc
		help
		insertdate
		insertfile
		inserttime
		insertdateandtime
		quit
		runmake
		save
		savequit
		savequitclear
		sortblock
		sortstrings
		spellcheck
		splitline
		version
	)

	// Define args and corresponding functions
	commandLookup := map[int]func(){
		build: func() { // build
			if e.Empty() {
				// Empty file, nothing to build
				e.redraw.Store(true)
				status.SetErrorMessageAfterRedraw("Nothing to build")
				return
			}
			// Save the current file, but only if it has changed
			if e.changed.Load() {
				if err := e.Save(c, tty); err != nil {
					status.ClearAll(c, false)
					status.SetError(err)
					status.Show(c, e)
					return
				}
			}
			// Build or export the current file
			outputExecutable, err := e.BuildOrExport(tty, c, status)
			// All clear when it comes to status messages and redrawing
			status.ClearAll(c, false)
			if err != nil {
				status.SetError(err)
				status.ShowNoTimeout(c, e)
				return
			}
			// --- Success ---
			status.SetMessageAfterRedraw("Success, built " + outputExecutable)
		},
		copyall: func() { // copy all contents to the clipboard
			text := e.String()
			if err := clip.WriteAll(text, e.primaryClipboard); err != nil {
				status.Clear(c, false)
				status.SetError(err)
				status.Show(c, e)
			} else {
				numLines := strings.Count(text, "\n") + 1
				plural := "s"
				if numLines == 1 {
					plural = ""
				}
				const fmtMsg = "Copied %d line%s from %s"
				status.SetMessageAfterRedraw(fmt.Sprintf(fmtMsg, numLines, plural, filepath.Base(e.filename)))
			}
		},
		copymark: func() { // copy the text between the bookmark and the current line (inclusive)
			startIndex := e.LineIndex()
			stopIndex := startIndex
			// If no bookmark has been set, just copy the line that the cursor is currently at
			if bookmark != nil {
				stopIndex = bookmark.LineIndex()
			}
			if startIndex > stopIndex {
				startIndex, stopIndex = stopIndex, startIndex
			}
			// from startIndex to stopIndex copy the lines (inclusive)
			var sb strings.Builder
			for lineIndex := startIndex; lineIndex <= stopIndex; lineIndex++ {
				if lineIndex > startIndex {
					sb.WriteRune('\n')
				}
				sb.WriteString(e.Line(lineIndex))
			}
			text := sb.String()
			if err := clip.WriteAll(text, e.primaryClipboard); err != nil {
				status.Clear(c, false)
				status.SetError(err)
				status.Show(c, e)
			} else {
				numLines := strings.Count(text, "\n") + 1
				status.SetMessageAfterRedraw(fmt.Sprintf("Copied %d lines", numLines))
			}
			// move the cursor to stopIndex
			e.redraw.Store(e.GoToLineNumber(LineNumber(stopIndex), c, status, true))
		},
		copy200: func() { // copy 200 lines of text and move the cursor 200 lines ahead
			startIndex := e.LineIndex()
			stopIndex := startIndex + 200
			lastIndex := LineIndex(e.Len()) - 1
			if stopIndex >= lastIndex {
				stopIndex = lastIndex
			}
			// copy lines from from startIndex (inclusive) to stopIndex (exclusive)
			var sb strings.Builder
			for lineIndex := startIndex; lineIndex < stopIndex; lineIndex++ {
				if lineIndex > startIndex {
					sb.WriteRune('\n')
				}
				sb.WriteString(e.Line(lineIndex))
			}
			text := sb.String()
			if err := clip.WriteAll(text, e.primaryClipboard); err != nil {
				status.Clear(c, false)
				status.SetError(err)
				status.Show(c, e)
			} else {
				numLines := strings.Count(text, "\n") + 1
				status.SetMessageAfterRedraw(fmt.Sprintf("Copied %d lines", numLines))
			}
			// move the cursor to stopIndex
			e.redraw.Store(e.GoToLineNumber(LineNumber(stopIndex+1), c, status, true))
		},
		gobacktofunc: func() {
			// A special case, search backwards to the start of the function (or to "main")
			s := e.FuncPrefix()
			if s == "" {
				s = "main"
			}
			const forward = false
			const wrap = true
			e.SetSearchTerm(c, status, s, false) // no timeout
			// Perform the actual search
			if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
				if err == errNoSearchMatch {
					e.ClearSearch()
					e.redraw.Store(true)
					e.redrawCursor.Store(true)
					status.SetErrorMessageAfterRedraw("No function signatures found")
				}
			}
		},
		help: func() { // display an informative status message
			// TODO: Draw the same type of box that is used in debug mode, listing all possible commands
			status.SetMessageAfterRedraw("sq, wq, savequit, s, save, q, quit, h, help, sort, v, version, date, insertfile [filename], build")
		},
		insertdate: func() { // insert the current date
			undo.Snapshot(e)
			// If a space is added after the string here, instead of using e.addSpace,
			// it will be stripped when the command menu disappears.
			dateString := time.Now().Format(time.RFC3339)[:10]
			e.InsertString(c, dateString)
			e.addSpace = true
		},
		insertfile: func() { // insert a file
			undo.Snapshot(e)
			editedFileDir := filepath.Dir(e.filename)
			filename2 := strings.TrimSpace(args[1])              // include.txt
			filename1 := filepath.Join(editedFileDir, filename2) // include.txt in the same dir as the edited file
			// First try inserting include.txt from the same directory as the edited file,
			// then try inserting include.txt from the current directory.
			if err := e.InsertFile(c, filename1); err != nil {
				if err2 := e.InsertFile(c, filename2); err2 != nil {
					e.redraw.Store(true)
					status.SetErrorAfterRedraw(err2)
				}
			}
		},
		inserttime: func() { // insert the current time
			undo.Snapshot(e)
			// If a space is added after the string here, instead of using e.addSpace,
			// it will be stripped when the command menu disappears.
			timeString := time.Now().Format("15:04") // HH:MM
			e.InsertString(c, timeString)
			e.addSpace = true
		},
		insertdateandtime: func() { // insert the current date and time
			undo.Snapshot(e)
			// If a space is added after the string here, instead of using e.addSpace,
			// it will be stripped when the command menu disappears.
			dateString := time.Now().Format(time.RFC3339)[:10]
			timeString := time.Now().Format("15:04") // HH:MM
			e.InsertString(c, dateString+" "+timeString)
			e.addSpace = true
		},
		runmake: func() {
			workDir := filepath.Dir(e.filename)
			found := false
			for _, fn := range []string{"GNUmakefile", "makefile", "Makefile"} {
				if files.Exists(filepath.Join(workDir, fn)) {
					found = true
					break
				}
			}
			if found {
				e.UserSave(c, tty, status)
				quitExecShellCommand(tty, workDir, "make")
			} else {
				status.SetErrorMessageAfterRedraw("no Makefile")
			}
		},
		save: func() { // save the current file
			e.UserSave(c, tty, status)
		},
		savequit: func() { // save and quit
			e.UserSave(c, tty, status)
			e.quit = true
		},
		savequitclear: func() { // save and quit, then clear the screen
			e.UserSave(c, tty, status)
			e.quit = true
			clearOnQuit.Store(true)
		},
		sortblock: func() { // sort the current block of lines, until the next blank line or EOF
			undo.Snapshot(e)
			e.SortBlock(c, status, bookmark)
		},
		sortstrings: func() { // sort the words on the current line
			undo.Snapshot(e)
			e.SortStrings()
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
		},
		spellcheck: func() {
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
			typo, corrected, err := e.SearchForTypo()
			switch {
			case err != nil:
				status.SetErrorAfterRedraw(err)
			case err == errFoundNoTypos || typo == "":
				status.SetMessageAfterRedraw("No typos found")
			case typo != "" && corrected != "":
				status.SetMessageAfterRedraw(typo + " could be " + corrected)
			}
		},
		splitline: func() { // split the current line on space
			undo.Snapshot(e)
			e.SmartSplitLineOnBlanks(c, status, bookmark)
		},
		quit: func() { // quit
			e.quit = true
		},
		version: func() { // display the program name and version as a status message
			status.SetMessageAfterRedraw(versionString)
		},
	}

	// TODO: Also handle the command arguments, command[1:], if given.
	//       For instance, the save commands could take a filename.

	// Helpful command aliases that can also handle some typos and abbreviations
	var functionID int
	switch trimmedCommand {
	case "bye", "cu", "ee", "exit", "q", "qq", "qu", "qui", "quit", "c:17": // ctrl-q
		functionID = quit
	case "build", "b", "bu", "bui":
		functionID = build
	case "copyall", "copya":
		functionID = copyall
	case "copymark", "copym":
		functionID = copymark
	case "copy200":
		functionID = copy200
	case "gobacktofunc":
		functionID = gobacktofunc
	case "h", "he", "hh", "hel", "help":
		functionID = help
	case "if", "i", "insertfile", "insert", "insertf":
		functionID = insertfile
	case "insertdate", "insertd", "id", "date", "d":
		functionID = insertdate
	case "inserttime", "time", "t", "ti", "tim":
		functionID = inserttime
	case "insertdateandtime", "dateandtime", "dt", "dati", "datim":
		functionID = insertdateandtime
	case "make":
		functionID = runmake
	case "qs", "byes", "cus", "exitsave", "quitandsave", "quitsave", "qw", "saq", "saveandquit", "saveexit", "saveq", "savequit", "savq", "sq", "wq", "↑", "c:23": // ctrl-w, if the user keeps holding down ctrl
		functionID = savequit
	case "s", "sa", "sav", "save", "w", "ww", "↓", "c:19": // ctrl-s, if the user keeps holding down ctrl
		functionID = save
	case "sb", "so", "sor", "sort", "sortblock":
		functionID = sortblock
	case "spl", "split", "splitline", "smartsplit":
		functionID = splitline
	case "sp", "spellcheck", "spell", "findtypo":
		functionID = spellcheck
	case "sortstrings", "sortw", "sortwords", "sow", "ss", "sw", "sortfields", "sf":
		functionID = sortstrings
	case "sqc", "savequitclear":
		functionID = savequitclear
	case "v", "ver", "vv", "version":
		functionID = version
	default:
		return nil, fmt.Errorf("unknown command: %s", args[0])
	}

	// Return the selected function
	f, ok := commandLookup[functionID]
	if !ok {
		return nil, fmt.Errorf("implementation missing for command: %s", args[0])
	}
	return f, nil
}

// RunCommand takes a command string and performs and action (like "save" or "quit")
func (e *Editor) RunCommand(c *vt.Canvas, tty *vt.TTY, status *StatusBar, bookmark *Position, undo *Undo, args ...string) error {
	f, err := e.CommandToFunction(c, tty, status, bookmark, undo, args...)
	if err != nil {
		return err
	}
	f()
	return nil
}

// CommandPrompt shows and handles user input that is interpreted as internal commands,
// or external commands if they start with "!"
func (e *Editor) CommandPrompt(c *vt.Canvas, tty *vt.TTY, status *StatusBar, bookmark *Position, undo *Undo) {
	// The spaces are intentional, to stop the shorter strings from always kicking in before
	// the longer ones can be typed.
	quickList := []string{":wq", "wq", "sq", "sqc", ":q", "q", ":w ", "s ", "w ", "d", "b", "↑", "↓", "c:23", "c:19", "c:17"}
	if useASCII {
		for i, entry := range quickList {
			quickList[i] = asciiFallback(entry)
		}
	}
	// TODO: Show a REPL in a nicely drawn box instead of this simple command interface
	//       The REPL can have colors, tab-completion, a command history and single-letter commands
	const tabCommand = "help"
	if commandString, ok := e.UserInput(c, tty, status, "o", "", quickList, true, tabCommand); ok {
		args := strings.Split(strings.TrimSpace(commandString), " ")
		if err := e.RunCommand(c, tty, status, bookmark, undo, args...); err != nil {
			status.SetErrorMessage(err.Error())
		}
		if e.quit {
			// Briefly show the last status message before quitting
			time.Sleep(120 * time.Millisecond)
		}
	} else {
		e.redrawCursor.Store(true)
	}
}
