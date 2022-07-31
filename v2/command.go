package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/vt100"
)

// CommandToFunction takes an editor command as a string (with optional arguments) and returns a function that
// takes no arguments and performs the suggested action, like "save". Some functions may take an undo snapshot first.
func (e *Editor) CommandToFunction(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, undo *Undo, args ...string) (func(), error) {
	if len(args) == 0 {
		return nil, errors.New("no command given")
	}

	trimmedCommand := strings.TrimPrefix(strings.TrimSpace(args[0]), ":")

	// Argument checks
	switch trimmedCommand {
	case "insertfile":
		if len(args) != 2 {
			return nil, fmt.Errorf("%s requires a filename as the second argument", trimmedCommand)
		}
	default:
		if len(args) != 1 {
			return nil, fmt.Errorf("%s takes no arguments", args[0])
		}
	}

	// Define args and corresponding functions
	var commandLookup = map[string]func(){
		"help": func() { // display an informative status message
			// TODO: Draw the same type of box that is used in debug mode, listing all possible commands
			status.SetMessageAfterRedraw(":wq, s, save, sq, savequit, q, quit, h, help, sort, v, version")
		},
		"insertdate": func() {
			undo.Snapshot(e)
			// note that if a space is added after the string here, it will be stripped when the command menu disappears
			dateString := time.Now().Format(time.RFC3339)[:10]
			e.InsertString(c, dateString)
			e.addSpace = true
		},
		"insertfile": func() {
			undo.Snapshot(e)
			editedFileDir := filepath.Dir(e.filename)
			if err := e.InsertFile(c, filepath.Join(editedFileDir, strings.TrimSpace(args[1]))); err != nil {
				status.Clear(c)
				status.SetError(err)
				status.Show(c, e)
			}
		},
		"save": func() { // save the current file
			e.UserSave(c, tty, status)
		},
		"savequit": func() { // save and quit
			e.UserSave(c, tty, status)
			e.quit = true
		},
		"savequitclear": func() { // save and quit, then clear the screen
			e.UserSave(c, tty, status)
			e.quit = true
			e.clearOnQuit = true
		},
		"sortblock": func() { // sort the current block of lines, until the next blank line or EOF
			undo.Snapshot(e)
			e.SortBlock(c, status, bookmark)
		},
		"sortstrings": func() { // sort the words on the current line
			undo.Snapshot(e)
			e.SortStrings(c, status)
			e.redraw = true
			e.redrawCursor = true
		},
		"quit": func() { // quit
			e.quit = true
		},
		"version": func() { // display the program name and version as a status message
			status.SetMessageAfterRedraw(versionString)
		},
	}

	// TODO: Also handle the command arguments, command[1:], if given.
	//       For instance, the save commands could take a filename.

	// Helpful command aliases that can also handle some typos and abbreviations
	lookupWord := ""
	switch trimmedCommand {
	case "qs", "byes", "cus", "exitsave", "quitandsave", "quitsave", "qw", "saq", "saveandquit", "saveexit", "saveq", "savequit", "savq", "sq", "wq":
		lookupWord = "savequit"
	case "s", "sa", "sav", "save", "w", "ww":
		lookupWord = "save"
	case "bye", "cu", "ee", "exit", "q", "qq", "qu", "qui", "quit":
		lookupWord = "quit"
	case "h", "he", "hh", "hel", "help":
		lookupWord = "help"
	case "v", "ver", "vv", "version":
		lookupWord = "version"
	case "sb", "so", "sor", "sort":
		lookupWord = "sortblock"
	case "sortstrings", "sortw", "sortwords", "sow", "ss", "sw", "sortfields", "sf":
		lookupWord = "sortstrings"
	default:
		return nil, fmt.Errorf("unknown command: %s", args[0])
	}

	// Return the selected function
	f, ok := commandLookup[lookupWord]
	if !ok {
		return nil, fmt.Errorf("implementation missing for command: %s", args[0])
	}
	return f, nil
}

// RunCommand takes a command string and performs and action (like "save" or "quit")
func (e *Editor) RunCommand(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, undo *Undo, args ...string) error {
	f, err := e.CommandToFunction(c, tty, status, bookmark, undo, args...)
	if err != nil {
		return err
	}
	f()
	return nil
}
