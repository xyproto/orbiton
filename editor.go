package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"unicode"

	"github.com/xyproto/syntax"
	"github.com/xyproto/textoutput"
	"github.com/xyproto/vt100"
)

// Editor represents the contents and editor settings, but not settings related to the viewport or scrolling
type Editor struct {
	lines        map[int][]rune
	eolMode      bool // stop at the end of lines, or float around?
	changed      bool // has the contents changed, since last save?
	fg           vt100.AttributeColor
	bg           vt100.AttributeColor
	spacesPerTab int  // how many spaces per tab character
	highlight    bool // syntax highlighting
	insertMode   bool // insert or overwrite mode?
}

// NewEditor takes:
// * the number of spaces per tab (typically 2, 4 or 8)
// * foreground color attributes
// * background color attributes
// * if syntax highlighting is enabled
// * if "insert mode" is enabled (as opposed to "overwrite mode")
func NewEditor(spacesPerTab int, fg, bg vt100.AttributeColor, highlight, insertMode bool) *Editor {
	e := &Editor{}
	e.lines = make(map[int][]rune)
	e.eolMode = true
	e.fg = fg
	e.bg = bg
	e.spacesPerTab = spacesPerTab
	e.highlight = highlight
	e.insertMode = insertMode
	return e
}

// EOLMode returns true if the editor is in "text edit mode" and the cursor should not float around
func (e *Editor) EOLMode() bool {
	return e.eolMode
}

// ToggleEOLMode toggles if the editor is in "text edit mode" or "ASCII graphics mode"
func (e *Editor) ToggleEOLMode() {
	e.eolMode = !e.eolMode
}

// Set will store a rune in the editor data, at the given data coordinates
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

// Get will retrieve a rune from the editor data, at the given coordinates
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

// Changed will return true if the contents were changed since last time this function was called
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

// ScreenLine returns the screen contents of line number N, counting from 0
func (e *Editor) ScreenLine(n int) string {
	line, ok := e.lines[n]
	if ok {
		var sb strings.Builder
		for _, r := range line {
			sb.WriteRune(r)
		}
		tabSpace := strings.Repeat("\t", e.spacesPerTab)
		return strings.ReplaceAll(sb.String(), "\t", tabSpace)
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
func (e *Editor) LastScreenPosition(n int) int {
	extraSpaceBecauseOfTabs := int(e.Count('\t', n) * (e.spacesPerTab - 1))
	return e.LastDataPosition(n) + extraSpaceBecauseOfTabs
}

// FirstScreenPosition returns the first X index for this line, that is not whitespace.
func (e *Editor) FirstScreenPosition(n int) int {
	counter := 0
	for _, r := range e.Line(n) {
		if unicode.IsSpace(r) {
			if r == '\t' {
				counter += e.spacesPerTab
			} else {
				counter++
			}
			continue
		} else {
			break
		}
	}
	return counter
}

// Count the number of instances of the rune r in the line n
func (e *Editor) Count(r rune, n int) int {
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

// Clear removes all data from the editor
func (e *Editor) Clear() {
	e.lines = make(map[int][]rune)
	e.changed = true
}

// Load will try to load a file
func (e *Editor) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	datalines := bytes.Split(data, []byte{'\n'})
	e.Clear()
	for y, dataline := range datalines {
		line := string(dataline)
		counter := 0
		for _, letter := range line {
			e.Set(counter, int(y), letter)
			counter++
		}
	}
	// Mark the data as "not changed"
	e.changed = false
	return nil
}

// Save will try to save a file
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
	// Mark the data as "not changed"
	e.changed = false
	// Write the data to file
	return ioutil.WriteFile(filename, data, 0664)
}

// TrimSpaceRight will remove spaces from the end of the given line number
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
	e.changed = true
}

// WriteLines will draw editor lines from "fromline" to and up to "toline" to the canvas, at cx, cy
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
		lastIsBlank := false
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
						lastIsBlank = letter == ' ' || letter == rune(0)
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
		if lastIsBlank {
			counter--
		}
		for x := counter; x < w; x++ {
			c.WriteRune(uint(cx+x), uint(cy+y), e.fg, e.bg, ' ')
		}
	}
	return nil
}

// DeleteRestOfLine will delete the rest of the line, from the given position
func (e *Editor) DeleteRestOfLine(p *Position) {
	x := p.DataX()
	y := p.DataY()
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
	e.changed = true
}

// DeleteLine will delete the given line index
func (e *Editor) DeleteLine(n int) {
	endOfDocument := n >= (e.Len() - 1)
	if endOfDocument {
		// Just delete this line
		delete(e.lines, n)
		return
	}
	// Shift all lines after y so that y is overwritten.
	// Then delete the last item.
	maxIndex := 0
	found := false
	for k := range e.lines {
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
	// Shift all lines after n one step closer to n, overwriting e.lines[n]
	for i := n; i <= (maxIndex - 1); i++ {
		e.lines[i] = e.lines[i+1]
	}
	// delete the final item
	delete(e.lines, maxIndex)

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}

	e.changed = true
}

// Delete will delete a character at the given position
func (e *Editor) Delete(p *Position) {
	x := p.DataX()
	y := p.DataY()
	if _, ok := e.lines[y]; !ok || len(e.lines[y]) == 0 || (len(e.lines[y]) == 1 && unicode.IsSpace(e.lines[y][0])) {
		// All keys in the map that are > y should be shifted -1.
		// This also overwrites e.lines[y].
		e.DeleteLine(y)
		e.changed = true
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
		e.changed = true
		return
	}
	// Delete just this character
	e.lines[y] = append(e.lines[y][:x], e.lines[y][x+1:]...)

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}

	e.changed = true
}

// Empty will check if the current editor contents are empty or not.
// If there's only one line left and it is only whitespace, that will be considered empty as well.
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

// MakeConsistent makes sure all the keys in the map that should be there are present, and removes all keys that should not be there
func (e *Editor) MakeConsistent() error {
	// Check if the keys in the map are consistent
	for i := 0; i < len(e.lines); i++ {
		if _, found := e.lines[i]; !found {
			e.lines[i] = make([]rune, 0)
			e.changed = true
		}
	}
	i := len(e.lines)
	if _, found := e.lines[i]; found {
		return fmt.Errorf("line number %d should not be there", i)
	}
	return nil
}

// InsertLineBelow will attempt to insert a new line below the current position
func (e *Editor) InsertLineBelow(p *Position) {
	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}

	y := p.DataY()
	newLength := len(e.lines) + 1
	newMap := make(map[int][]rune, newLength)
	for i := 0; i < newLength; i++ {
		if i < y {
			newMap[i] = e.lines[i]
		} else if i == y {
			// Create a new line
			newMap[i] = make([]rune, 0)
		} else if i > y {
			newMap[i] = e.lines[i-1]
		}
	}
	// Assign the new map
	e.lines = newMap

	e.MakeConsistent()

	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len(e.lines[i]) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}

	e.changed = true
}

// Insert will insert a rune at the given position
func (e *Editor) Insert(p *Position, r rune) {
	dataCursor := p.DataCursor()
	x := dataCursor.X
	y := dataCursor.Y

	// If there are no lines, initialize and set the 0th rune to the given one
	if e.lines == nil {
		e.lines = make(map[int][]rune)
		e.lines[0] = []rune{r}
		return
	}

	// If the current line is empty, initialize it with a line that is just the given rune
	_, ok := e.lines[y]
	if !ok {
		e.lines[y] = []rune{r}
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

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}

	e.changed = true
}

// CreateLineIfMissing will create a line at the given Y index, if it's missing
func (e *Editor) CreateLineIfMissing(n int) {
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[n]
	if !ok {
		e.lines[n] = make([]rune, 0)
		e.changed = true
	}

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		vt100.Reset()
		vt100.Clear()
		panic(err)
	}
}

// SetColors will set the current editor theme (foreground, background).
// The background color should be a background attribute (like vt100.BackgroundBlue).
func (e *Editor) SetColors(fg, bg vt100.AttributeColor) {
	e.fg = fg
	e.bg = bg
}

// WordCount returns the number of spaces in the text + 1
func (e *Editor) WordCount() int {
	return strings.Count(e.String(), " ") + 1
}

// ToggleHighlight toggles syntax highlighting
func (e *Editor) ToggleHighlight() {
	e.highlight = !e.highlight
}

// SetHighlight enables or disables syntax highlighting
func (e *Editor) SetHighlight(highlight bool) {
	e.highlight = highlight
}

// ToggleInsertMode toggles insert mode
func (e *Editor) ToggleInsertMode() {
	e.insertMode = !e.insertMode
}

// SetInsertMode enables or disables insert mode
func (e *Editor) SetInsertMode(insertMode bool) {
	e.insertMode = insertMode
}

// InsertMode returns the current state for the insert mode
func (e *Editor) InsertMode() bool {
	return e.insertMode
}

func (e *Editor) SetLine(n int, s string) {
	e.CreateLineIfMissing(n)
	e.lines[n] = []rune{}
	counter := 0
	// It's important not to use the index value when looping over a string,
	// unless the byte index is what one's after, as opposed to the rune index.
	for _, letter := range s {
		e.Set(counter, n, letter)
		counter++
	}
}

// At the given position, split the line in two, then place the right side of the contents on a new line below
func (e *Editor) SplitLine(p *Position) {
	dataCursor := p.DataCursor()
	x := dataCursor.X
	y := dataCursor.Y
	// Get the contents of this line
	line := e.Line(y)
	leftContents := strings.TrimRightFunc(line[:x], unicode.IsSpace)
	rightContents := line[x:]
	// Insert a new line below this one
	e.InsertLineBelow(p)
	// Replace this line with the left contents
	e.SetLine(y, leftContents)
	e.SetLine(y+1, rightContents)
}
