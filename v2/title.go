package main

import (
	"path/filepath"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/termtitle"
)

// NoTitle will remove the filename title by setting the shell name as the title,
// if NO_COLOR is not set and the terminal emulator supports it.
func NoTitle() {
	if envNoColor {
		return
	}
	shellName := filepath.Base(env.Str("SHELL", "/bin/sh"))
	termtitle.Set(shellName)
}
