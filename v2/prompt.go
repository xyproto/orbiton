package main

import (
	"fmt"

	"github.com/xyproto/vt100"
)

// CommandPrompt takes a command string and performs and action (like "save" or "quit")
func (e *Editor) CommandPrompt(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, bookmark *Position, command string) error {
	switch command {
	case "wq", "sq", "saveandquit", "savequit", "quitsave", "quitandsave", "qw", "qs":
		e.quit = true
		fallthrough
	case "w", "s", "save", "ww", "ss":
		e.UserSave(c, tty, status)
	case "q", "qq", "quit", "ee", "exit", "bye", "cu":
		e.quit = true
	case "h", "hh", "help":
		// TODO: Draw the same type of box that is used in debug mode, listing all possible commands
		status.SetMessageAfterRedraw("w, save, wq, savequit, q, quit, h, help, v, version")
	case "v", "vv", "version":
		status.SetMessageAfterRedraw(versionString)
	case "sort":
		undo.Snapshot(e)
		// sort all lines, until the next blank line or until the end of the file
		e.SortBlock(c, status, bookmark)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
	return nil
}
