package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/xyproto/clip"
	"github.com/xyproto/vt"
)

var errNoSearchMatch = errors.New("no search match")

// SetSearchTerm will set the current search term. This initializes a new search.
func (e *Editor) SetSearchTerm(c *vt.Canvas, status *StatusBar, s string, spellCheckMode bool) bool {
	foundMatch := false
	// set the search term
	e.searchTerm = s
	// set the sticky search term (used by ctrl-n, cleared by Esc only)
	e.stickySearchTerm = s
	// set spellcheck mode
	e.spellCheckMode = spellCheckMode
	// Go to the first instance after the current line, if found
	e.lineBeforeSearch = e.DataY()
	for y := e.DataY(); y < LineIndex(e.Len()); y++ {
		if strings.Contains(e.Line(y), s) {
			// Found an instance, scroll there
			// GoTo returns true if the screen should be redrawn
			redraw, _ := e.GoTo(y, c, status)
			if redraw {
				foundMatch = true
				e.Center(c)
			}
			break
		}
	}
	// draw the lines to the canvas
	e.HideCursorDrawLines(c, true, false, false)
	return foundMatch
}

// SetSearchTermWithTimeout will set the current search term. This initializes a new search.
func (e *Editor) SetSearchTermWithTimeout(c *vt.Canvas, status *StatusBar, s string, spellCheckMode bool, timeout time.Duration) bool {
	// set the search term
	e.searchTerm = s
	// set the sticky search term (used by ctrl-n, cleared by Esc only)
	e.stickySearchTerm = s
	// set spellcheck mode
	e.spellCheckMode = spellCheckMode
	// Go to the first instance after the current line, if found
	e.lineBeforeSearch = e.DataY()

	// create a channel to signal when a match is found
	matchFound := make(chan bool)

	foundMatch := LineIndex(-1)
	var foundMutex sync.RWMutex

	// run the search in a separate goroutine
	go func() {
		for y := e.DataY(); y < LineIndex(e.Len()); y++ {
			if strings.Contains(e.Line(y), s) {
				matchFound <- true
				foundMutex.Lock()
				foundMatch = y
				foundMutex.Unlock()
				return
			}
		}
		matchFound <- false
	}()

	// wait for either a match to be found or timeout
	select {
	case <-matchFound:
	case <-time.After(timeout):
	}

	foundMutex.RLock()
	defer foundMutex.RUnlock()

	if foundMatch != -1 {
		// Found an instance, scroll there
		// GoTo returns true if the screen should be redrawn
		redraw, _ := e.GoTo(foundMatch, c, status)
		if redraw {
			e.Center(c)
			// Draw the lines to the canvas
			e.HideCursorDrawLines(c, true, false, false)
			e.redraw.Store(true)
		}
		return true
	}

	return false
}

// SearchTerm will return the current search term
func (e *Editor) SearchTerm() string {
	return e.searchTerm
}

// ClearSearch will clear the current search term
func (e *Editor) ClearSearch() {
	e.searchTerm = ""
}

// UseStickySearchTerm will use the sticky search term as the current search term,
// which is not cleared by Esc, but by ctrl-p.
func (e *Editor) UseStickySearchTerm() {
	if e.stickySearchTerm != "" {
		e.searchTerm = e.stickySearchTerm
	}
}

// forwardSearch is a helper function for searching for a string from the given startIndex,
// up to the given stopIndex. -1
// -1 is returned if there are no matches.
// startIndex is expected to be smaller than stopIndex
// x, y is returned.
func (e *Editor) forwardSearch(startIndex, stopIndex LineIndex) (int, LineIndex) {
	var (
		s      = e.SearchTerm()
		foundX = -1
		foundY = LineIndex(-1)
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
		s      = e.SearchTerm()
		foundX = -1
		foundY = LineIndex(-1)
	)
	if len(s) == 0 {
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
func (e *Editor) GoToNextMatch(c *vt.Canvas, status *StatusBar, wrap, forward bool) error {
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
	redraw, _ := e.GoTo(foundY, c, status)
	e.redraw.Store(redraw)
	if foundX != -1 {
		tabs := strings.Count(e.Line(foundY), "\t")
		e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
		e.HorizontalScrollIfNeeded(c)
	}

	// Center and prepare to redraw
	e.Center(c)
	e.redraw.Store(true)
	e.redrawCursor.Store(redraw)

	return nil
}

// SearchMode will enter the interactive "search mode" where the user can type in a string and then press return to search
func (e *Editor) SearchMode(c *vt.Canvas, status *StatusBar, tty *vt.TTY, clearSearch, searchForward bool, undo *Undo) {
	notRegularEditingRightNow.Store(true)
	defer func() {
		notRegularEditingRightNow.Store(false)
	}()

	// Attempt to load the search history. Ignores errors, but does not try to load it twice if it fails.
	if searchHistory.Len() == 0 && !searchHistory.FailedToLoad() {
		searchHistory = LoadSearchHistory()
	}
	// Attempt to load the replace history. Ignores errors, but does not try to load it twice if it fails.
	if replaceHistory.Len() == 0 && !replaceHistory.FailedToLoad() {
		replaceHistory = LoadReplaceHistory()
	}

	var (
		previousSearch      string
		key                 string
		initialLocation     = e.DataY().LineNumber()
		searchHistoryIndex  int
		replaceHistoryIndex int
		replaceMode         bool
		timeout             = 500 * time.Millisecond
	)

	searchPrompt := "Search:"
	if !searchForward {
		searchPrompt = "Search backwards:"
	}

AGAIN:
	doneCollectingLetters := false
	pressedReturn := false
	pressedTab := false
	if clearSearch {
		// Clear the previous search
		e.SetSearchTerm(c, status, "", false) // no timeout
	}
	s := e.SearchTerm()
	status.ClearAll(c, true)
	if s == "" {
		status.SetMessage(searchPrompt)
	} else {
		status.SetMessage(searchPrompt + " " + s)
	}
	status.ShowNoTimeout(c, e)
	for !doneCollectingLetters {
		if e.macro == nil || (e.playBackMacroCount == 0 && !e.macro.Recording) {
			// Read the next key in the regular way
			key = tty.StringRaw()
		} else {
			if e.macro.Recording {
				// Read and record the next key
				key = tty.StringRaw()
				if key != "c:20" { // ctrl-t
					// But never record the macro toggle button
					e.macro.Add(key)
				}
			} else if e.playBackMacroCount > 0 {
				key = e.macro.Next()
				if key == "" || key == "c:20" { // ctrl-t
					e.macro.Home()
					e.playBackMacroCount--
					// No more macro keys. Read the next key.
					key = tty.StringRaw()
				}
			}
		}
		switch key {
		case "c:8", "c:127": // ctrl-h or backspace
			if len(s) > 0 {
				s = s[:len(s)-1]
				if previousSearch == "" {
					e.SetSearchTermWithTimeout(c, status, s, false, timeout)
				}
				e.GoToLineNumber(initialLocation, c, status, false)
				status.SetMessage(searchPrompt + " " + s)
				status.ShowNoTimeout(c, e)
			}
		case "c:3", "c:6", "c:17", "c:27", "c:24": // ctrl-c, ctrl-f, ctrl-q, esc or ctrl-x
			s = ""
			if previousSearch == "" {
				e.SetSearchTermWithTimeout(c, status, s, false, timeout)
			}
			doneCollectingLetters = true
		case "c:9": // tab
			// collect letters again, this time for the replace term
			pressedTab = true
			doneCollectingLetters = true
		case "c:13": // return
			pressedReturn = true
			doneCollectingLetters = true
		case "c:22": // ctrl-v, paste the last line in the clipboard
			// Read the clipboard
			clipboardString, err := clip.ReadAll(false) // non-primary clipboard
			if err == nil && strings.TrimSpace(s) == "" {
				clipboardString, err = clip.ReadAll(true) // try the primary clipboard
			}
			if err == nil { // success
				if strings.Contains(clipboardString, "\n") {
					lines := strings.Split(clipboardString, "\n")
					for i := len(lines) - 1; i >= 0; i-- {
						trimmedLine := strings.TrimSpace(lines[i])
						if trimmedLine != "" {
							s = trimmedLine
							status.SetMessage(searchPrompt + " " + s)
							status.ShowNoTimeout(c, e)
							break
						}
					}
				} else if trimmedLine := strings.TrimSpace(clipboardString); trimmedLine != "" {
					s = trimmedLine
					status.SetMessage(searchPrompt + " " + s)
					status.ShowNoTimeout(c, e)
				}
			}
		case upArrow, leftArrow: // arrow up, arrown left : use the previous search or replacement string from the search or replacement history
			if (!replaceMode && searchHistory.Empty()) || (replaceMode && replaceHistory.Empty()) {
				break
			}
			const newestFirst = false
			if !replaceMode {
				searchHistoryIndex--
				if searchHistoryIndex < 0 {
					// wraparound
					searchHistoryIndex = searchHistory.Len() - 1
				}
				s = searchHistory.GetIndex(searchHistoryIndex, newestFirst)
			} else {
				replaceHistoryIndex--
				if replaceHistoryIndex < 0 {
					// wraparound
					replaceHistoryIndex = replaceHistory.Len() - 1
				}
				s = replaceHistory.GetIndex(replaceHistoryIndex, newestFirst)
			}
			if previousSearch == "" {
				e.SetSearchTermWithTimeout(c, status, s, false, timeout)
			}
			status.ClearAll(c, true)
			status.SetMessage(searchPrompt + " " + s)
			status.ShowNoTimeout(c, e)
		case downArrow, rightArrow: // arrow down, arrow right : use the next search or replacement string from the search or replacement history
			if (!replaceMode && searchHistory.Empty()) || (replaceMode && replaceHistory.Empty()) {
				break
			}
			const newestFirst = false
			if !replaceMode {
				searchHistoryIndex++
				if searchHistoryIndex >= searchHistory.Len() {
					// wraparound
					searchHistoryIndex = 0
				}
				s = searchHistory.GetIndex(searchHistoryIndex, newestFirst)
			} else {
				replaceHistoryIndex++
				if replaceHistoryIndex >= replaceHistory.Len() {
					// wraparound
					replaceHistoryIndex = 0
				}
				s = replaceHistory.GetIndex(replaceHistoryIndex, newestFirst)
			}
			if previousSearch == "" {
				e.SetSearchTermWithTimeout(c, status, s, false, timeout)
			}
			status.ClearAll(c, true)
			status.SetMessage(searchPrompt + " " + s)
			status.ShowNoTimeout(c, e)
		default:
			if key != "" && !strings.HasPrefix(key, "c:") {
				s += key
				if previousSearch == "" {
					e.SetSearchTermWithTimeout(c, status, s, false, timeout)
				}
				status.SetMessage(searchPrompt + " " + s)
				status.ShowNoTimeout(c, e)
			}
		}
	}
	status.ClearAll(c, false)
	// Search settings
	forward := searchForward // forward search
	wrap := true             // with wraparound
	foundNoTypos := false
	spellCheckMode := false
	if s == "" && !replaceMode {
		// No search string entered, and not in replace mode, use the current word, if available
		s = e.CurrentWord()
	} else if s == "t" {
		// A special case, search forward for typos
		spellCheckMode = true
		foundNoTypos = false
		typo, corrected, err := e.SearchForTypo()
		if err != nil {
			return
		}
		if err == errFoundNoTypos || typo == "" {
			foundNoTypos = true
			status.ClearAll(c, true)
			status.SetMessageAfterRedraw("No typos found")
			e.redraw.Store(true)
			e.spellCheckMode = false
			e.ClearSearch()
			return
		}
		if typo != "" && corrected != "" {
			status.ClearAll(c, true)
			e.redraw.Store(true)
			status.SetMessageAfterRedraw(typo + " could be " + corrected)
		}
		s = typo
		forward = true
	}
	if pressedTab && previousSearch == "" { // search text -> tab
		// got the search text, now gather the replace text
		previousSearch = e.searchTerm
		searchPrompt = "Replace with:"
		replaceMode = true
		goto AGAIN
	} else if pressedTab && previousSearch != "" { // search text -> tab -> replace text- > tab
		undo.Snapshot(e)
		// replace once
		searchFor := previousSearch
		replaceWith := s
		replaced := strings.Replace(e.String(), searchFor, replaceWith, 1)
		e.LoadBytes([]byte(replaced))
		if replaceWith == "" {
			status.SetMessageAfterRedraw("Removed " + searchFor + ", once")
		} else {
			status.SetMessageAfterRedraw("Replaced " + searchFor + " with " + replaceWith + ", once")
		}
		// Save "searchFor" to the search history, if we are on a fast enough system
		if trimmedSearchString := strings.TrimSpace(searchFor); trimmedSearchString != "" && !e.slowLoad {
			searchHistory.AddAndSave(trimmedSearchString)
		}
		// Save "replaceWidth" to the replace history, if we are on a fast enough system
		if trimmedReplaceString := strings.TrimSpace(replaceWith); trimmedReplaceString != "" && !e.slowLoad {
			replaceHistory.AddAndSave(trimmedReplaceString)
		}
		// Set up a redraw and return
		e.redraw.Store(true)
		return
	} else if pressedReturn && previousSearch != "" { // search text -> tab -> replace text -> return
		undo.Snapshot(e)
		// replace all
		searchForBytes := []byte(previousSearch)
		replaceWithBytes := []byte(s)
		// check if we're searching and replacing an unicode character, like "U+0047" or "u+0000"
		if r, err := runeFromUBytes(searchForBytes); err == nil { // success
			searchForBytes = []byte(string(r))
		}
		if r, err := runeFromUBytes(replaceWithBytes); err == nil { // success
			replaceWithBytes = []byte(string(r))
		}
		// perform the replacements, and count the number of instances
		allBytes := []byte(e.String())
		instanceCount := bytes.Count(allBytes, searchForBytes)
		allReplaced := bytes.ReplaceAll(allBytes, searchForBytes, replaceWithBytes)
		// replace the contents
		e.LoadBytes(allReplaced)
		// build a status message
		extraS := ""
		if instanceCount != 1 {
			extraS = "s"
		}
		if s == "" {
			status.messageAfterRedraw = fmt.Sprintf("Removed %d instance%s of %s", instanceCount, extraS, previousSearch)
		} else {
			status.messageAfterRedraw = fmt.Sprintf("Replaced %d instance%s of %s with %s", instanceCount, extraS, previousSearch, s)
		}
		// Save "searchForBytes" to the search history, if we are on a fast enough system
		if trimmedSearchString := strings.TrimSpace(string(searchForBytes)); trimmedSearchString != "" && !e.slowLoad {
			searchHistory.AddAndSave(trimmedSearchString)
		}
		// Save "replaceWidthBytes" to the replace history, if we are on a fast enough system
		if trimmedReplaceString := strings.TrimSpace(string(replaceWithBytes)); trimmedReplaceString != "" && !e.slowLoad {
			replaceHistory.AddAndSave(trimmedReplaceString)
		}
		// Set up a redraw and return
		e.redraw.Store(true)
		return
	}
	e.SetSearchTermWithTimeout(c, status, s, spellCheckMode, timeout)
	if pressedReturn {
		// Return to the first location before performing the actual search
		e.GoToLineNumber(initialLocation, c, status, false)
		// Save "s" to the search or replace history, if we are on a fast enough system

		// Save "s" to the search history, if we are on a fast enough system
		if trimmedSearchString := strings.TrimSpace(s); trimmedSearchString != "" && !e.slowLoad {
			searchHistory.AddAndSave(trimmedSearchString)
		} else if !searchHistory.Empty() {
			const newestFirst = false
			s = searchHistory.GetIndex(searchHistoryIndex, newestFirst)
			e.SetSearchTerm(c, status, s, false) // no timeout
		}
	}
	if previousSearch == "" {
		// Perform the actual search
		if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
			// If no match was found, and return was not pressed, try again from the top
			// e.GoToTop(c, status)
			// err = e.GoToNextMatch(c, status)
			if err == errNoSearchMatch {
				status.ClearAll(c, true)
				e.redraw.Store(true)
				if foundNoTypos || spellCheckMode {
					status.SetMessageAfterRedraw("No typos found")
					e.spellCheckMode = false
					e.ClearSearch()
				} else if wrap {
					status.SetMessageAfterRedraw(s + " not found")
				} else {
					status.SetMessageAfterRedraw(s + " not found from here")
				}
			}
		}
		e.Center(c)
	}
}
