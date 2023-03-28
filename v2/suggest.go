package main

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/xyproto/vt100"
)

// corpus will grep all files matching the glob for "searchword.*" and return a list of what matched "*".
// I thought this might be slow, but initial tests shows that this appears to be fast enough for interactive usage.
func corpus(searchword, glob string) []string {
	wordCount := make(map[string]int)

	filenames, err := filepath.Glob(glob)
	if err != nil {
		return []string{}
	}

	corpusRegex := regexp.MustCompile(`[[:^alpha:]]` + searchword + `\.([[:alpha:]]*)`)

	var data []byte
	var highestCount int
	for _, filename := range filenames {
		data, err = os.ReadFile(filename)
		if err != nil {
			continue
		}
		submatches := corpusRegex.FindAllStringSubmatch(string(data), -1)
		for _, submatch := range submatches {
			word := submatch[1]
			if _, ok := wordCount[word]; ok {
				wordCount[word]++
				if wordCount[word] > highestCount {
					highestCount = wordCount[word]
				}
			} else {
				wordCount[word] = 1
				if wordCount[word] > highestCount {
					highestCount = wordCount[word]
				}
			}
		}
	}

	// Copy the words from the map to a string slice, such
	// that the most frequent words appear first.
	sl := make([]string, len(wordCount))
	slIndex := 0
	for i := highestCount; i >= 0; i-- {
		for word, count := range wordCount {
			if count == i && len(word) > 0 {
				sl[slIndex] = word
				slIndex++
			}
		}
	}

	return sl
}

// SuggestMode lets the user tab through the suggested words
func (e *Editor) SuggestMode(c *vt100.Canvas, status *StatusBar, tty *vt100.TTY, suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}

	suggestIndex := 0
	s := suggestions[suggestIndex]

	status.ClearAll(c)
	status.SetMessage("Suggest: " + s)
	status.ShowNoTimeout(c, e)

	var doneChoosing bool
	for !doneChoosing {
		key := tty.String()
		switch key {
		case "c:9", "↓", "→": // tab, down arrow or right arrow
			// Cycle suggested words
			suggestIndex++
			if suggestIndex == len(suggestions) {
				suggestIndex = 0
			}
			s = suggestions[suggestIndex]
			status.ClearAll(c)
			status.SetMessage("Suggest: " + s)
			status.ShowNoTimeout(c, e)
		case "↑", "←": // up arrow or left arrow
			// Cycle suggested words (one back)
			suggestIndex--
			if suggestIndex < 0 {
				suggestIndex = len(suggestions) - 1
			}
			s = suggestions[suggestIndex]
			status.ClearAll(c)
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
	status.ClearAll(c)
	// The chosen word
	return s
}
