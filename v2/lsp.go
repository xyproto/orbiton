package main

import (
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// TryLSPCompletion attempts to perform LSP-based tab completion for Go files.
// Returns true if completion was performed, false if normal tab behavior should continue.
func (e *Editor) TryLSPCompletion(c *vt.Canvas) bool {
	// Debug: Log the attempt
	logf("LSP: Attempting completion for mode=%s, filename=%s", e.mode.String(), e.filename)

	// Only attempt LSP completion for Go files
	if e.mode != mode.Go {
		logf("LSP: Not a Go file, mode=%s", e.mode.String())
		return false
	}

	// SIMPLE TEST: Just insert some test text to verify tab key integration works
	logf("LSP: Inserting test completion text")
	e.InsertString(c, "TEST_COMPLETION")

	// Mark the editor as changed and redraw
	e.changed.Store(true)
	if c != nil {
		c.Redraw()
	}

	logf("LSP: Test completion inserted successfully")
	return true
}
