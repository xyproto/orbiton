package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/xyproto/syntax"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
	"io/ioutil"
	"strings"
	"unicode"
)

type Editor struct {
	lines        map[int][]rune
	eolMode      bool // stop at the end of lines
	changed      bool
	fg           vt100.AttributeColor
	bg           vt100.AttributeColor
	spacesPerTab int // how many spaces per tab character
	scrollSpeed  int // how many lines to scroll, when scrolling
	highlight    bool
}

// Takes:
// * the number of spaces per tab (typically 2, 4 or 8)
// * how many lines the editor should scroll when ctrl-n or ctrl-p are pressed (typically 1, 5 or 10)
// * foreground color attributes
// * background color attributes
func NewEditor(spacesPerTab, scrollSpeed int, fg, bg vt100.AttributeColor, highlight bool) *Editor {
	e := &Editor{}
	e.lines = make(map[int][]rune)
	e.eolMode = true
	e.fg = fg
	e.bg = bg
	e.spacesPerTab = spacesPerTab
	e.scrollSpeed = scrollSpeed
	e.highlight = highlight
	return e
}

func (e *Editor) EOLMode() bool {
	return e.eolMode
}

func (e *Editor) ToggleEOLMode() {
	e.eolMode = !e.eolMode
}

func (e *Editor) Set(x, y int, r rune) {
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[y]
	if !ok {
		e.lines[y] = make([]rune, 0, x+1)
	}
	if x < int(len(e.lines[y])) {
		e.lines[y][x] = r
		e.changed = true
		return
	}
	// If the line is too short, fill it up with spaces
	for x >= int(len(e.lines[y])) {
		e.lines[y] = append(e.lines[y], ' ')
	}
	e.lines[y][x] = r
	e.changed = true
}

func (e *Editor) Get(x, y int) rune {
	if e.lines == nil {
		return ' '
	}
	runes, ok := e.lines[y]
	if !ok {
		return ' '
	}
	if x >= int(len(runes)) {
		return ' '
	}
	return runes[x]
}

func (e *Editor) Changed() bool {
	return e.changed
}

// Line returns the contents of line number N, counting from 0
func (e *Editor) Line(n int) string {
	line, ok := e.lines[n]
	if ok {
		var sb strings.Builder
		for _, r := range line {
			sb.WriteRune(r)
		}
		return sb.String()
	}
	return ""
}

// LastDataPosition returns the last X index for this line, for the data (does not expand tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastDataPosition(n int) int {
	return len(e.Line(n)) - 1
}

// LastScreenPosition returns the last X index for this line, for the screen (expands tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastScreenPosition(n, spacesPerTab int) int {
	extraSpaceBecauseOfTabs := int(e.Count(n, '\t') * (spacesPerTab - 1))
	return e.LastDataPosition(n) + extraSpaceBecauseOfTabs
}

// For a given line index, count the number of given runes
func (e *Editor) Count(n int, r rune) int {
	var counter int
	line, ok := e.lines[n]
	if ok {
		for _, l := range line {
			if l == r {
				counter++
			}
		}
	}
	return counter
}

// Len returns the number of lines
func (e *Editor) Len() int {
	maxy := 0
	for y := range e.lines {
		if y > maxy {
			maxy = y
		}
	}
	return maxy + 1
}

// String returns the contents of the editor
func (e *Editor) String() string {
	var sb strings.Builder
	for i := 0; i < e.Len(); i++ {
		sb.WriteString(e.Line(i) + "\n")
	}
	return sb.String()
}

func (e *Editor) Clear() {
	e.lines = make(map[int][]rune)
}

func (e *Editor) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	datalines := bytes.Split(data, []byte{'\n'})
	e.Clear()
	for y, dataline := range datalines {
		line := string(dataline)
		for x, letter := range line {
			e.Set(int(x), int(y), letter)
		}
	}
	return nil
}

func (e *Editor) Save(filename string, stripTrailingSpaces bool) error {
	var data []byte
	if stripTrailingSpaces {
		// Strip trailing spaces
		for i := 0; i < e.Len(); i++ {
			e.TrimSpaceRight(i)
		}
		// Skip trailing newlines
		data = bytes.TrimRightFunc([]byte(e.String()), unicode.IsSpace)
		// Add a final newline
		data = append(data, '\n')
	} else {
		data = []byte(e.String())
	}
	// Write the data to file
	return ioutil.WriteFile(filename, data, 0664)
}

// Remove spaces from the end of the given line number
func (e *Editor) TrimSpaceRight(n int) {
	_, ok := e.lines[n]
	if !ok {
		return
	}
	lastIndex := len(e.lines[n]) - 1
	// find the last non-space position
	for x := lastIndex; x > 0; x-- {
		if !unicode.IsSpace(e.lines[n][x]) {
			lastIndex = x
			break
		}
	}
	// Remove the trailing spaces
	e.lines[n] = e.lines[n][:(lastIndex + 1)]
}

// Write editor lines from "fromline" to and up to "toline" to the canvas at cx, cy
func (e *Editor) WriteLines(c *vt100.Canvas, fromline, toline, cx, cy int) error {
	o := textoutput.NewTextOutput(true, true)
	tabString := strings.Repeat(" ", e.spacesPerTab)
	w := int(c.Width())
	if fromline >= toline {
		return errors.New("fromline >= toline in WriteLines")
	}
	numlines := toline - fromline
	offset := fromline
	for y := 0; y < numlines; y++ {
		counter := 0
		line := strings.ReplaceAll(e.Line(y+offset), "\t", tabString)
		if len(line) >= w {
			// Shorten the line a bit if it's too wide
			line = line[:w]
		}
		if e.highlight {
			// Output a syntax highlighted line
			vt100.SetXY(uint(cx+counter), uint(cy+y))
			if textWithTags, err := syntax.AsText([]byte(line)); err != nil {
				fmt.Println(line)
				counter += len(line)
			} else {
				// Slice of runes and color attributes
				charactersAndAttributes := o.Extract(o.DarkTags(string(textWithTags)))
				for _, ca := range charactersAndAttributes {
					letter := ca.R
					fg := ca.A
					if letter == ' ' {
						fg = e.fg
					}
					if letter == '\t' {
						c.Write(uint(cx+counter), uint(cy+y), fg, e.bg, tabString)
						counter += 4
					} else {
						c.WriteRune(uint(cx+counter), uint(cy+y), fg, e.bg, letter)
						counter++
					}
				}
			}
		} else {
			// Output a regular line
			c.Write(uint(cx+counter), uint(cy+y), e.fg, e.bg, line)
			counter += len(line)
		}
		// Fill the rest of the line on the canvas with "blanks"
		for x := counter; x < w; x++ {
			c.WriteRune(uint(cx+x), uint(cy+y), e.fg, e.bg, ' ')
		}
	}
	return nil
}

func (e *Editor) DeleteRestOfLine(x, y int) {
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[y]
	if !ok {
		return
	}
	if x >= len(e.lines[y]) {
		return
	}
	e.lines[y] = e.lines[y][:x]
}

func (e *Editor) DeleteLine(y int) {
	endOfDocument := y >= (e.Len() - 1)
	if endOfDocument {
		// Just delete this line
		delete(e.lines, y)
		return
	}
	// Shift all lines after y so that y is overwritten.
	// Then delete the last item.
	maxIndex := 0
	found := false
	for k, _ := range e.lines {
		if k > maxIndex {
			maxIndex = k
			found = true
		}
	}
	if !found {
		// This should never happen
		//panic("IMPOSSIBRUUUU!")
		return
	}
	if _, ok := e.lines[maxIndex]; !ok {
		// The line numbers and the length of e.lines does not match
		return
	}
	// Shift all lines after y one step closer to y, overwriting y
	for i := y; i <= (maxIndex - 1); i++ {
		e.lines[i] = e.lines[i+1]
	}
	// delete the final item
	delete(e.lines, maxIndex)
}

func (e *Editor) Delete(x, y int) {
	if _, ok := e.lines[y]; !ok || len(e.lines[y]) == 0 || (len(e.lines[y]) == 1 && unicode.IsSpace(e.lines[y][0])) {
		// All keys in the map that are > y should be shifted -1.
		// This also overwrites e.lines[y].
		e.DeleteLine(y)
		return
	}
	if x >= len(e.lines[y])-1 {
		// on the last index, just use every element but x
		e.lines[y] = e.lines[y][:x]
		// check if the next line exists
		if _, ok := e.lines[y+1]; ok {
			// then add the contents of the next line, if available
			nextLine, ok := e.lines[y+1]
			if ok && len(nextLine) > 0 {
				e.lines[y] = append(e.lines[y], nextLine...)
				// then delete the next line
				e.DeleteLine(y + 1)
			}
		}
		return
	}
	// Delete just this character
	e.lines[y] = append(e.lines[y][:x], e.lines[y][x+1:]...)
}

func (e *Editor) Empty() bool {
	l := len(e.lines)
	if l == 0 {
		return true
	} else if l == 1 {
		// Check the contents of the 1 remaining line,
		// without specifying a key.
		for _, v := range e.lines {
			if len(strings.TrimSpace(string(v))) == 0 {
				return true
			}
			break
		}
		return false
	} else {
		// > 1 lines
		return false
	}
}

//func (e *Editor) InsertLineAbove(p *Position) {
//	y := p.DataCursor(e).Y
//	newmap := make(map[int][]rune)
//	for i := 0; i < len(e.lines); i++ {
//		if i < (y - 1) {
//			newmap[i] = e.lines[i]
//		} else {
//			newmap[i+1] = e.lines[i]
//		}
//	}
//	// Create a new line
//	newmap[y-1] = make([]rune, 0)
//	e.lines = newmap
//}

func (e *Editor) InsertLineBelow(p *Position) {
	y := p.DataCursor(e).Y
	newLength := len(e.lines) + 1
	newMap := make(map[int][]rune, newLength)
	// i <= len(e.lines) goes up to len(e.lines)+1, on purpose
	for i := 0; i <= newLength; i++ {
		if i < y {
			newMap[i] = e.lines[i]
		} else if i > y {
			newMap[i+1] = e.lines[i]
		}
	}
	// Create a new line
	newMap[y] = make([]rune, 0)
	e.lines = newMap
	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len(e.lines[i]) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}
}

func (e *Editor) Insert(p *Position, r rune) {
	dataCursor := p.DataCursor(e)
	x := dataCursor.X
	y := dataCursor.Y
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[y]
	if !ok {
		// Can only insert in the existing block of text
		return
	}
	if len(e.lines[y]) < x {
		// Can only insert in the existing block of text
		return
	}
	newline := make([]rune, len(e.lines[y])+1)
	for i := 0; i < x; i++ {
		newline[i] = e.lines[y][i]
	}
	newline[x] = r
	for i := x + 1; i < len(newline); i++ {
		newline[i] = e.lines[y][i-1]
	}
	e.lines[y] = newline
}

func (e *Editor) CreateLineIfMissing(n int) {
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[n]
	if !ok {
		e.lines[n] = make([]rune, 0)
	}
}

func (e *Editor) SetColors(fg, bg vt100.AttributeColor) {
	e.fg = fg
	e.bg = bg
}

// WordCount returns the number of spaces in the text + 1
func (e *Editor) WordCount() int {
	return strings.Count(e.String(), " ") + 1
}

func (e *Editor) ToggleHighlight() {
	e.highlight = !e.highlight
}

func (e *Editor) SetHighlight(highlight bool) {
	e.highlight = highlight
}
