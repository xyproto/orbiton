package main

import (
	"strings"

	"github.com/xyproto/vt100"
)

// SetSearchTerm will set the current search term to highlight
func (e *Editor) SetSearchTerm(s string, c *vt100.Canvas, status *StatusBar) {
	// set the search term
	e.searchTerm = s
	// set the sticky search term (used by ctrl-n, cleared by Esc only)
	e.stickySearchTerm = s
	// Go to the first instance after the current line, if found
	e.lineBeforeSearch = e.DataY()
	for y := e.DataY(); y < e.Len(); y++ {
		if strings.Contains(e.Line(y), s) {
			// Found an instance, scroll there
			// GoTo returns true if the screen should be redrawn
			redraw := e.GoTo(y, c, status)
			if redraw {
				e.Center(c)
			}
			break
		}
	}
	// draw the lines to the canvas
	e.DrawLines(c, true, false)
}

// SearchTerm will return the current search term
func (e *Editor) SearchTerm() string {
	return e.searchTerm
}

// UseStickySearchTerm will use the sticky search term as the current search term,
// which is not cleared by Esc, but by ctrl-p.
func (e *Editor) UseStickySearchTerm() {
	if e.stickySearchTerm != "" {
		e.searchTerm = e.stickySearchTerm
	}
}

// ClearStickySearchTerm will clear the sticky search term, for when ctrl-n is pressed.
func (e *Editor) ClearStickySearchTerm() {
	e.stickySearchTerm = ""
}

// GoToNextMatch will go to the next match, using e.SearchTerm(), if possible.
// The search does not wrap around and is case-sensitive.
// TODO: Add wrap around behavior, toggled with a bool argument.
// TODO: Add case-insensitive behavior, toggled with a bool argument.
func (e *Editor) GoToNextMatch(c *vt100.Canvas, status *StatusBar) {
	s := e.SearchTerm()
	if s != "" {
		// Go to the next line with "s"
		foundY := -1
		foundX := -1
		for y := e.DataY(); y < e.Len(); y++ {
			lineContents := e.Line(y)
			if y == e.DataY() {
				x, err := e.DataX()
				if err != nil {
					continue
				}
				// Search from the next position on this line
				x++
				if x >= len(lineContents) {
					continue
				}
				if strings.Contains(lineContents[x:], s) {
					foundX = x + strings.Index(lineContents[x:], s)
					foundY = y
					break
				}
			} else {
				if strings.Contains(lineContents, s) {
					foundX = strings.Index(lineContents, s)
					foundY = y
					break
				}
			}
		}
		if foundY != -1 {
			e.redraw = e.GoTo(foundY, c, status)
			if foundX != -1 {
				tabs := strings.Count(e.Line(foundY), "\t")
				e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
			}
			e.Center(c)
			e.redraw = true
			e.redrawCursor = e.redraw
		} else {
			e.GoTo(e.lineBeforeSearch, c, status)
			status.SetMessage(s + " not found from here")
			status.Show(c, e)
		}
	}
}

// SearchMode will enter the interactive "search mode" where the user can type in a string and then press return to search
func (e *Editor) SearchMode(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, clear bool) {
	if clear {
		// Clear the previous search
		e.SetSearchTerm("", c, status)
	}
	s := e.SearchTerm()
	status.ClearAll(c)
	if s == "" {
		status.SetMessage("Search:")
	} else {
		status.SetMessage("Search: " + s)
	}
	status.ShowNoTimeout(c, e)
	doneCollectingLetters := false
	for !doneCollectingLetters {
		key := tty.String()
		switch key {
		case "c:127": // backspace
			if len(s) > 0 {
				s = s[:len(s)-1]
				e.SetSearchTerm(s, c, status)
				status.SetMessage("Search: " + s)
				status.ShowNoTimeout(c, e)
			}
		case "c:27", "c:17": // esc or ctrl-q
			s = ""
			e.SetSearchTerm(s, c, status)
			fallthrough
		case "c:13": // return
			doneCollectingLetters = true
		default:
			if key != "" && !strings.HasPrefix(key, "c:") {
				s += key
				e.SetSearchTerm(s, c, status)
				status.SetMessage("Search: " + s)
				status.ShowNoTimeout(c, e)
			}
		}
	}
	status.ClearAll(c)
	e.SetSearchTerm(s, c, status)
	e.GoToNextMatch(c, status)
	e.Center(c)
}
