package main

import (
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/termtitle"
)

// FilenameOrData represents either a filename, or data read in from stdin
type FilenameOrData struct {
	filename string
	data     []byte
	length   int
	stdin    bool
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
	if fnord.stdin {
		title = "stdin"
		if len(fnord.data) > 512 {
			fields := strings.Fields(string(fnord.data[:512]))
			firstWord := fields[0]
			if len(firstWord) >= 2 && strings.Contains(firstWord, "(") && strings.Contains(firstWord, ")") {
				// Probably a man page, create a nicely formatted lowercase title
				// "LS(1)" becomes "man ls"
				fields = strings.Split(firstWord, "(")
				title = "man " + strings.ToLower(fields[0])
			}
		}
	} else if fnord.filename != "" {
		title = termtitle.GenerateTitle(fnord.filename)
	}
	termtitle.Set(title)
}
