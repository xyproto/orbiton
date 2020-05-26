package main

import (
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

// InsertRune will insert a rune at the current data position, with word wrap
func (e *Editor) InsertRune(c *vt100.Canvas, r rune) {
	y := int(e.DataY())

	// If it's not a word-wrap situation, just insert and return
	if e.wordWrapAt == 0 || e.WithinLimit(LineIndex(y)) {
		e.Insert(r)
		return
	}

	// Insert a regular space instead of a nonbreaking space.
	// Nobody likes nonbreaking spaces.
	if r == 0xc2a0 {
		r = ' '
	}

	// --- Repaint, afterwards ---

	e.changed = true
	e.redrawCursor = true
	e.redraw = true

	// --- Gather some facts ---

	isSpace := unicode.IsSpace(r)
	x, err := e.DataX()
	if err != nil {
		x = e.pos.sx
	}
	x += e.pos.offsetX

	prevAtSpace := false
	if x > 0 && x <= len(e.lines[y]) {
		prevAtSpace = unicode.IsSpace(e.lines[y][x-1])
	}
	atSpace := false
	if x >= 0 && x < len(e.lines[y]) {
		atSpace = unicode.IsSpace(e.lines[y][x])
	}

	//panic(fmt.Sprintf("x=%d, y=%d, line=%s, atSpace=%v, prevAtSpace=%v\n", x, y, e.Line(y), atSpace, prevAtSpace))
	EOL := e.AtOrAfterEndOfLine()

	lastWord := []rune(strings.TrimSpace(e.LastWord(y)))
	shortWord := (len(string(lastWord)) < 10) && (len(string(lastWord)) < e.wordWrapAt)

	// logf("InsertRune, isSpace=%v, atSpace=%v, prevAtSpace=%v, EOL=%v, r=%s, lastWord=%s, shortWord=%v\n", isSpace, atSpace, prevAtSpace, EOL, string(r), string(lastWord), shortWord)

	// --- A large switch/case for catching all cases ---

	switch {
	case !EOL:
		// The line is full. Move everything one line down and continue writing.
		right := e.lines[y][x:]
		e.lines[y] = e.lines[y][:x]
		e.TrimRight(LineIndex(y))
		e.insertBelow(y, r)
		e.lines[y+1] = append([]rune{r}, right...)
		// Go to the len(lastWord)-1 of the next line
		e.GoTo(LineIndex(y+1), c, nil)
		e.pos.sx = 0
	case !isSpace && !atSpace && EOL:
		// Pressing letters, producing a short word that overflows
		lastWord = append(lastWord, r)
		// Remove the last r of the current line
		pos := len(e.lines[y]) - len(lastWord)
		if pos > 0 {
			e.lines[y] = e.lines[y][:pos]
			e.TrimRight(LineIndex(y))
		} else {
			// This would leave the current line empty!
			// Typing a letter at the end of a line, breaking a word
			if _, ok := e.lines[y+1]; !ok {
				// If the next line does not exist, create one containing just "r"
				e.lines[y+1] = []rune{r}
			} else if len(e.lines[y+1]) > 0 {
				// If the next line is non-empty, insert "r" at the start
				e.lines[y+1] = append([]rune{r}, e.lines[y+1][:]...)
			}
			// Go to the start of the next line
			e.nextLine(LineIndex(y), c, nil)
			break
		}
		// Insert the last word of the above line on the next line
		if _, ok := e.lines[y+1]; !ok {
			// If the next line does not exist, create one containing just "lastWord" + "r"
			if prevAtSpace {
				lastpos := len(lastWord) - 1
				lastWord = append(lastWord[:lastpos], ' ')
				lastWord = append(lastWord, r)
			}
			e.lines[y+1] = lastWord
		} else if len(e.lines[y+1]) > 0 {
			// If the next line is non-empty, insert "lastWord" + "r" at the start
			e.lines[y+1] = append(lastWord, e.lines[y+1][:]...)
		}
		// Go to the len(lastWord)-1 of the next line
		e.GoTo(LineIndex(y+1), c, nil)
		e.pos.sx = len(lastWord) - 1
	case isSpace && EOL:
		// Space at the end of a long word
		e.InsertLineBelowAt(LineIndex(y))
	case !isSpace && EOL && !shortWord:
		fallthrough
	case !isSpace && prevAtSpace && EOL:
		fallthrough
	case !isSpace && atSpace && !prevAtSpace && EOL && shortWord:
		fallthrough
	case !isSpace && !atSpace && !prevAtSpace && EOL && !shortWord:
		// Pressing a single letter
		e.insertBelow(y, r)
		e.nextLine(LineIndex(y), c, nil)
	case !isSpace && !atSpace && !prevAtSpace && EOL && shortWord:
		// Typing a letter or a space, and the word is short, so it should be moved down
		if !isSpace {
			lastWord = append(lastWord, r)
		}
		// Remove the last r of the current line
		pos := len(e.lines[y]) - len(lastWord)
		if pos > 0 {
			e.lines[y] = e.lines[y][:pos]
			e.TrimRight(LineIndex(y))
		} else {
			// This would leave the current line empty!
			// Typing a letter at the end of a line, breaking a word
			if _, ok := e.lines[y+1]; !ok {
				// If the next line does not exist, create one containing just "r"
				e.lines[y+1] = []rune{r}
			} else if len(e.lines[y+1]) > 0 {
				// If the next line is non-empty, insert "r" at the start
				e.lines[y+1] = append([]rune{r}, e.lines[y+1][:]...)
			}
			// Go to the start of the next line
			e.nextLine(LineIndex(y), c, nil)
			break
		}
		// Insert the last word of the above line on the next line
		if _, ok := e.lines[y+1]; !ok {
			// If the next line does not exist, create one containing just "lastWord" + "r"
			if prevAtSpace {
				lastpos := len(lastWord) - 1
				lastWord = append(lastWord[:lastpos], ' ')
				lastWord = append(lastWord, r)
			}
			e.lines[y+1] = lastWord
		} else if len(e.lines[y+1]) > 0 {
			// If the next line is non-empty, insert "lastWord" + "r" at the start
			e.lines[y+1] = append(lastWord, e.lines[y+1][:]...)
		}
		// Go to the len(lastWord)-1 of the next line
		e.GoTo(LineIndex(y+1), c, nil)
		e.pos.sx = len(lastWord) - 1
	default:
		e.Insert(r)
	}
	e.TrimRight(LineIndex(y))
}
