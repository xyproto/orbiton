package main

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

var errNoSearchMatch = errors.New("no search match")

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
// Returns an error if the search was successful but no match was found.
func (e *Editor) GoToNextMatch(c *vt100.Canvas, status *StatusBar) error {
	s := e.SearchTerm()
	if s == "" {
		return nil
	}

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

	// Check if a match was found
	if foundY == -1 {
		// Not found
		e.GoTo(e.lineBeforeSearch, c, status)
		status.SetMessage(s + " not found from here")
		status.Show(c, e)
		return errNoSearchMatch
	}

	// Go to the found match
	e.redraw = e.GoTo(foundY, c, status)
	if foundX != -1 {
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
	}
	e.Center(c)

	e.redraw = true
	e.redrawCursor = e.redraw
	return nil
}

// SearchMode will enter the interactive "search mode" where the user can type in a string and then press return to search
func (e *Editor) SearchMode(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, clear bool) {
	const searchPrompt = "Search:"
	if clear {
		// Clear the previous search
		e.SetSearchTerm("", c, status)
	}
	s := e.SearchTerm()
	status.ClearAll(c)
	if s == "" {
		status.SetMessage(searchPrompt)
	} else {
		status.SetMessage(searchPrompt + " " + s)
	}
	status.ShowNoTimeout(c, e)
	var (
		key                   string
		doneCollectingLetters bool
	)
	for !doneCollectingLetters {
		key = tty.String()
		switch key {
		case "c:127": // backspace
			if len(s) > 0 {
				s = s[:len(s)-1]
				e.SetSearchTerm(s, c, status)
				status.ClearAll(c)
				status.SetMessage(searchPrompt + " " + s)
				status.ShowNoTimeout(c, e)
			}
		case "c:27", "c:17": // esc or ctrl-q
			s = ""
			e.SetSearchTerm(s, c, status)
			fallthrough
		case "c:13": // return
			doneCollectingLetters = true
		case "↑": // previous in the search history
			// TODO: Browse backwards in the search history and don't fall through
		case "↓": // next in the search history
			// TODO: Browse backwards in the search history and don't fall through
		default:
			if key != "" && !strings.HasPrefix(key, "c:") {
				s += key
				e.SetSearchTerm(s, c, status)
				status.SetMessage(searchPrompt + " " + s)
				status.ShowNoTimeout(c, e)
			}
		}
	}
	status.ClearAll(c)
	e.SetSearchTerm(s, c, status)
	// Perform the actual search
	if err := e.GoToNextMatch(c, status); err == errNoSearchMatch {
		// If no match was found, try again from the top
		e.redraw = e.GoToLineNumber(1, c, status, true)
		_ = e.GoToNextMatch(c, status)

	}
	e.Center(c)
}
