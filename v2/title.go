package main

import (
	"path/filepath"
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/termtitle"
)

// SetTitle sets an appropriate terminal emulator title, unless NO_COLOR is set
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
	termtitle.Set(sanitizeFilename(title))
}

// NoTitle will remove the filename title by setting the shell name as the title,
// if NO_COLOR is not set and the terminal emulator supports it.
func NoTitle() {
	if envNoColor {
		return
	}
	shellName := filepath.Base(env.Str("SHELL", "/bin/sh"))
	termtitle.Set(shellName)
}
