package main

import (
	"strings"

	"github.com/xyproto/env"
)

// FilenameOrData represents either a filename, or data read in from stdin
type FilenameOrData struct {
	filename string
	data     []byte
}

// ExpandUser will expand the filename if it starts with "~"
func (fnod *FilenameOrData) ExpandUser() {
	// If the filename starts with "~", then expand it
	if strings.HasPrefix(fnod.filename, "~") {
		fnod.filename = env.ExpandUser(fnod.filename)
	}
}

// Empty checks if data has been loaded
func (fnod *FilenameOrData) Empty() bool {
	return len(fnod.data) == 0
}
