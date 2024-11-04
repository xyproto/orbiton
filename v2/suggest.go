package main

import (
	"github.com/xyproto/vt100"
)

// SuggestMode lets the user tab through the suggested words
func (e *Editor) SuggestMode(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	suggestIndex := 0
	s := suggestions[suggestIndex]

	status.ClearAll(c, true)
	status.SetMessage("Suggest: " + s)
	status.ShowNoTimeout(c, e)

	var doneChoosing bool
	for !doneChoosing {
		key := tty.String()
		switch key {
		case "c:9", downArrow, rightArrow: // tab, down arrow or right arrow
			// Cycle suggested words
			suggestIndex++
			if suggestIndex == len(suggestions) {
				suggestIndex = 0
			}
			s = suggestions[suggestIndex]
			status.ClearAll(c, true)
			status.SetMessage("Suggest: " + s)
			status.ShowNoTimeout(c, e)
		case upArrow, leftArrow: // up arrow or left arrow
			// Cycle suggested words (one back)
			suggestIndex--
			if suggestIndex < 0 {
				suggestIndex = len(suggestions) - 1
			}
			s = suggestions[suggestIndex]
			status.ClearAll(c, true)
			status.SetMessage("Suggest: " + s)
			status.ShowNoTimeout(c, e)
		case "c:8", "c:127": // ctrl-h or backspace
			fallthrough
		case "c:27", "c:17": // esc, ctrl-q or backspace
			s = ""
			fallthrough
		case "c:13", "c:32": // return or space
			doneChoosing = true
		}
	}
	status.ClearAll(c, true)
	// The chosen word
	return s
}
