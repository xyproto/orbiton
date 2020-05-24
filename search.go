package main

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/xyproto/vt100"
)

var (
	searchHistoryFilename = "~/.cache/o/search.txt" // TODO: Use XDG_CACHE_HOME
	searchHistory         = []string{}
	errNoSearchMatch      = errors.New("no search match")
)

// SetSearchTerm will set the current search term to highlight
func (e *Editor) SetSearchTerm(c *vt100.Canvas, status *StatusBar, s string) {
	// set the search term
	e.searchTerm = s
	// set the sticky search term (used by ctrl-n, cleared by Esc only)
	e.stickySearchTerm = s
	// Go to the first instance after the current line, if found
	e.lineBeforeSearch = e.DataY()
	for y := e.DataY(); y < LineIndex(e.Len()); y++ {
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

// ClearSearchTerm will clear the current search term
func (e *Editor) ClearSearchTerm() {
	e.searchTerm = ""
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

// forwardSearch is a helper function for searching for a string from the given startIndex,
// up to the given stopIndex. -1
// -1 is returned if there are no matches.
// startIndex is expected to be smaller than stopIndex
// x, y is returned.
func (e *Editor) forwardSearch(startIndex, stopIndex LineIndex) (int, LineIndex) {
	var (
		s      string    = e.SearchTerm()
		foundX int       = -1
		foundY LineIndex = -1
	)
	if s == "" {
		// Return -1, -1 if no search term is set
		return foundX, foundY
	}
	currentIndex := e.DataY()
	// Search from the given startIndex up to the given stopIndex
	for y := startIndex; y < stopIndex; y++ {
		lineContents := e.Line(y)
		if y == currentIndex {
			x, err := e.DataX()
			if err != nil {
				continue
			}
			// Search from the next byte (not rune) position on this line
			// TODO: Move forward one rune instead of one byte
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
	return foundX, LineIndex(foundY)
}

// backwardSearch is a helper function for searching for a string from the given startIndex,
// backwards to the given stopIndex. -1, -1 is returned if there are no matches.
// startIndex is expected to be larger than stopIndex
func (e *Editor) backwardSearch(startIndex, stopIndex LineIndex) (int, LineIndex) {
	var (
		s      string    = e.SearchTerm()
		foundX int       = -1
		foundY LineIndex = -1
	)
	if s == "" {
		// Return -1, -1 if no search term is set
		return foundX, foundY
	}
	currentIndex := e.DataY()
	// Search from the given startIndex backwards up to the given stopIndex
	for y := startIndex; y >= stopIndex; y-- {
		lineContents := e.Line(y)
		if y == currentIndex {
			x, err := e.DataX()
			if err != nil {
				continue
			}
			// Search from the next byte (not rune) position on this line
			// TODO: Move forward one rune instead of one byte
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
	return foundX, LineIndex(foundY)
}

// GoToNextMatch will go to the next match, searching for "e.SearchTerm()".
// * The search wraps around if wrap is true.
// * The search is backawards if forward is false.
// * The search is case-sensitive.
// Returns an error if the search was successful but no match was found.
func (e *Editor) GoToNextMatch(c *vt100.Canvas, status *StatusBar, wrap, forward bool) error {
	var (
		foundX int
		foundY LineIndex
		s      = e.SearchTerm()
	)

	// Check if there's something to search for
	if s == "" {
		return nil
	}

	// Search forward or backward
	if forward {
		// Forward search from the current location
		startIndex := e.DataY()
		stopIndex := LineIndex(e.Len())
		foundX, foundY = e.forwardSearch(startIndex, stopIndex)
	} else {
		// Backward search form the current location
		startIndex := e.DataY()
		stopIndex := LineIndex(0)
		foundX, foundY = e.backwardSearch(startIndex, stopIndex)
	}

	if foundY == -1 && wrap {
		if forward {
			// Do a search from the top if a match was not found
			startIndex := LineIndex(0)
			stopIndex := LineIndex(e.Len())
			foundX, foundY = e.forwardSearch(startIndex, stopIndex)
		} else {
			// Do a search from the bottom if a match was not found
			startIndex := LineIndex(e.Len())
			stopIndex := LineIndex(0)
			foundX, foundY = e.backwardSearch(startIndex, stopIndex)
		}
	}

	// Check if a match was found
	if foundY == -1 {
		// Not found
		e.GoTo(e.lineBeforeSearch, c, status)
		return errNoSearchMatch
	}

	// Go to the found match
	e.redraw = e.GoTo(foundY, c, status)
	if foundX != -1 {
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
	}

	// Center and prepare to redraw
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
		e.SetSearchTerm(c, status, "")
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
		initialLocation       = e.DataY().LineNumber()
		pressedReturn         bool
		index                 int = 1
	)
	for !doneCollectingLetters {
		key = tty.String()
		switch key {
		case "c:127": // backspace
			if len(s) > 0 {
				s = s[:len(s)-1]
				e.SetSearchTerm(c, status, s)
				e.GoToLineNumber(initialLocation, c, status, false)

				//status.ClearAll(c)
				status.SetMessage(searchPrompt + " " + s)
				//status.Show(c, e)
				status.ShowNoTimeout(c, e)
			}
		case "c:27", "c:17": // esc or ctrl-q
			s = ""
			e.SetSearchTerm(c, status, s)
			doneCollectingLetters = true
		case "c:13": // return
			pressedReturn = true
			doneCollectingLetters = true
		case "↑": // previous in the search history
			if len(searchHistory) == 0 {
				break
			}
			index--
			if index < 0 {
				// wraparound
				index = len(searchHistory) - 1
			}
			s = searchHistory[index]
			e.SetSearchTerm(c, status, s)

			//status.ClearAll(c)
			status.SetMessage(searchPrompt + " " + s)
			//status.Show(c, e)
			status.ShowNoTimeout(c, e)
		case "↓": // next in the search history
			if len(searchHistory) == 0 {
				break
			}
			index++
			if index >= len(searchHistory) {
				// wraparound
				index = 0
			}
			s = searchHistory[index]
			e.SetSearchTerm(c, status, s)

			status.SetMessage(searchPrompt + " " + s)
			status.ShowNoTimeout(c, e)
		default:
			if key != "" && !strings.HasPrefix(key, "c:") {
				s += key
				e.SetSearchTerm(c, status, s)

				status.SetMessage(searchPrompt + " " + s)
				status.ShowNoTimeout(c, e)

			}
		}
	}
	status.ClearAll(c)
	e.SetSearchTerm(c, status, s)

	if pressedReturn {
		// Return to the first location before performing the actual search
		e.GoToLineNumber(initialLocation, c, status, false)
	}

	if len(strings.TrimSpace(s)) > 0 {
		searchHistory = append(searchHistory, s)
	}

	// Perform the actual search
	wrap := true    // with wraparound
	forward := true // forward search
	if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
		// If no match was found, and return was not pressed, try again from the top
		//e.redraw = e.GoToLineNumber(1, c, status, true)
		//err = e.GoToNextMatch(c, status)
		if err == errNoSearchMatch {
			if wrap {
				status.SetMessage(s + " not found")
			} else {
				status.SetMessage(s + " not found from here")
			}
			status.ShowNoTimeout(c, e)
		}
	}
	e.Center(c)
}

// LoadSearchHistory will load a list of strings from the given filename
func LoadSearchHistory(filename string) ([]string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return []string{}, err
	}
	// This can load empty words, but they should never be stored in the first place
	return strings.Split(string(data), "\n"), nil
}

// SaveSearchHistory will save a list of strings to the given filename
func SaveSearchHistory(filename string, list []string) error {
	if len(list) == 0 {
		return nil
	}

	// First create the folder, if needed, in a best effort attempt
	folderPath := filepath.Dir(filename)
	os.MkdirAll(folderPath, os.ModePerm)

	// Then save the data, with strict permissions
	data := []byte(strings.Join(list, "\n"))
	return ioutil.WriteFile(filename, data, 0600)
}
