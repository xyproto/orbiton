package main

import (
	"strings"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/termtitle"
	"github.com/xyproto/vt"
)

// SetTitle sets an appropriate terminal emulator title, unless NO_COLOR is set
// or the terminal does not support it
func (fnord *FilenameOrData) SetTitle() {
	if envNoColor || !vt.XtermLike() {
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
// or just "-", if NO_COLOR is not set and the terminal emulator supports it.
func NoTitle() {
	if envNoColor || !vt.XtermLike() {
		return
	}
	if shell := env.Str("SHELL"); shell != "" {
		termtitle.Set(shell)
	} else {
		termtitle.Set("-")
	}
}
