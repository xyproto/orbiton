package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xyproto/vt100"
)

// CommandToFunction takes an editor command as a string (with optional arguments) and returns a function that
// takes no arguments and performs the suggested action, like "save". Some functions may take an undo snapshot first.
func (e *Editor) CommandToFunction(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, commands ...string) (func(), error) {
	var commandLookup = map[string]func(){
		"savequit": func() {
			e.UserSave(c, tty, status)
			e.quit = true
		},
		"save": func() {
			e.UserSave(c, tty, status)
		},
		"quit": func() {
			e.quit = true
		},
		"help": func() {
			// TODO: Draw the same type of box that is used in debug mode, listing all possible commands
			status.SetMessageAfterRedraw(":wq, s, save, sq, savequit, q, quit, h, help, sort, v, version")
		},
		"sortblock": func() {
			undo.Snapshot(e)
			// sort all lines, until the next blank line or until the end of the file
			e.SortBlock(c, status, bookmark)
		},
		"version": func() {
			status.SetMessageAfterRedraw(versionString)
		},
	}

	if len(commands) == 0 {
		return nil, errors.New("no command given")
	}

	trimmedCommand := strings.TrimPrefix(strings.TrimSpace(commands[0]), ":")

	// TODO: Also handle the command arguments, command[1:], if given

	// Command aliases
	lookupWord := ""
	switch trimmedCommand {
	case "wq", "sq", "saveandquit", "savequit", "quitsave", "quitandsave", "qw", "qs":
		lookupWord = "savequit"
	case "w", "s", "save", "ww", "ss":
		lookupWord = "save"
	case "q", "qq", "quit", "ee", "exit", "bye", "cu":
		lookupWord = "quit"
	case "h", "hh", "help":
		lookupWord = "help"
	case "v", "vv", "version":
		lookupWord = "version"
	case "sort":
		lookupWord = "sortblock"
	default:
		return nil, fmt.Errorf("unknown command: %s", commands[0])
	}

	// Return the selected function
	f, ok := commandLookup[lookupWord]
	if !ok {
		return nil, fmt.Errorf("implementation missing for command: %s", commands[0])
	}
	return f, nil
}

// RunCommand takes a command string and performs and action (like "save" or "quit")
func (e *Editor) RunCommand(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, command string) error {
	f, err := e.CommandToFunction(c, tty, status, bookmark, command)
	if err != nil {
		return err
	}
	f()
	return nil
}
