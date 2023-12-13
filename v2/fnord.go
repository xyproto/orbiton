package main

import (
	"strings"

	"github.com/xyproto/env/v2"
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
