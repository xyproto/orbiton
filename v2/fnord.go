package main

import (
	"path/filepath"
	"strings"

	"github.com/xyproto/env"
	"github.com/xyproto/termtitle"
)

// FilenameOrData represents either a filename, or data read in from stdin
type FilenameOrData struct {
	filename string
	data     []byte
	length   uint64
}

// ExpandUser will expand the filename if it starts with "~"
// fnord is short for "filename or data"
func (fnord *FilenameOrData) ExpandUser() {
	// If the filename starts with "~", then expand it
	if strings.HasPrefix(fnord.filename, "~") {
		fnord.filename = env.ExpandUser(fnord.filename)
	}
}

// Empty checks if data has been loaded
func (fnord *FilenameOrData) Empty() bool {
	return fnord.length == 0
}

// String returns the contents as a string
func (fnord *FilenameOrData) String() string {
	return string(fnord.data)
}

// SetTitle sets an approperiate terminal emulator title, unless NO_COLOR is set
func (fnord *FilenameOrData) SetTitle() {
	if envNoColor {
		return
	}
	title := "?"
	if fnord.length > 0 {
		title = "stdin"
	} else if fnord.filename != "" {
		title = fnord.filename
	}
	termtitle.MustSet(termtitle.GenerateTitle(title))
}

// NoTitle will remove the filename title by setting the shell name as the title,
// if NO_COLOR is not set and the terminal emulator supports it.
func NoTitle() {
	if envNoColor {
		return
	}
	shellName := filepath.Base(env.Str("SHELL", "/bin/sh"))
	termtitle.MustSet(shellName)
}
