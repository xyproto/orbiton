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
	changed      bool // has the contents changed, since last save?
	fg           vt100.AttributeColor
	bg           vt100.AttributeColor
	spacesPerTab int  // how many spaces per tab character
	highlight    bool // syntax highlighting
	drawMode     bool // text or draw mode (for ASCII graphics)?
	pos          Position
}

// NewEditor takes:
// * the number of spaces per tab (typically 2, 4 or 8)
// * foreground color attributes
// * background color attributes
// * if syntax highlighting is enabled
// * if "insert mode" is enabled (as opposed to "draw mode")
func NewEditor(spacesPerTab int, fg, bg vt100.AttributeColor, highlight, textEditMode bool, scrollSpeed int) *Editor {
	e := &Editor{}
	e.lines = make(map[int][]rune)
	e.fg = fg
	e.bg = bg
	e.spacesPerTab = spacesPerTab
	e.highlight = highlight
	e.drawMode = !textEditMode
	p := NewPosition(scrollSpeed)
	e.pos = *p
	return e
}

// CopyLines will create a new map[int][]rune struct that is the copy of all the lines in the editor
func (e *Editor) CopyLines() map[int][]rune {
	lines2 := make(map[int][]rune)
	for key, runes := range e.lines {
		runes2 := make([]rune, len(runes), len(runes))
		for i, r := range runes {
			runes2[i] = r
		}
		lines2[key] = runes2
	}
	return lines2
}

// Copy will create a new Editor struct that is a copy of this one
func (e *Editor) Copy() Editor {
	var e2 Editor
	e2.lines = e.CopyLines()
	e2.changed = e.changed
	e2.fg = e.fg
	e2.bg = e.bg
	e2.spacesPerTab = e.spacesPerTab
	e2.highlight = e.highlight
	e2.drawMode = e.drawMode
	e2.pos = e.pos
	return e2
}

// DrawMode returns true if the editor is in "text edit mode" and the cursor should not float around
func (e *Editor) DrawMode() bool {
	return e.drawMode
}

// ToggleDrawMode toggles if the editor is in "text edit mode" or "ASCII graphics mode"
func (e *Editor) ToggleDrawMode() {
	e.drawMode = !e.drawMode
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
	if x < int(len([]rune(e.lines[y]))) {
		e.lines[y][x] = r
		e.changed = true
		return
	}
	// If the line is too short, fill it up with spaces
	for x >= int(len([]rune(e.lines[y]))) {
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
		tabSpace := "\t"
		if !e.DrawMode() {
			tabSpace = strings.Repeat("\t", e.spacesPerTab)
		}
		//return strings.ReplaceAll(sb.String(), "\t", tabSpace)
		return strings.Replace(sb.String(), "\t", tabSpace, -1)
	}
	return ""
}

// LastDataPosition returns the last X index for this line, for the data (does not expand tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastDataPosition(n int) int {
	return len([]rune(e.Line(n))) - 1
}

// LastScreenPosition returns the last X index for this line, for the screen (expands tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastScreenPosition(n int) int {
	if e.DrawMode() {
		return e.LastDataPosition(n)
	}
	// TODO: THIS IS WRONG, it does not account for unicode characters
	extraSpaceBecauseOfTabs := int(e.Count('\t', n) * (e.spacesPerTab - 1))
	return e.LastDataPosition(n) + extraSpaceBecauseOfTabs
}

// FirstScreenPosition returns the first X index for this line, that is not whitespace.
func (e *Editor) FirstScreenPosition(n int) int {
	spacesPerTab := e.spacesPerTab
	if e.DrawMode() {
		spacesPerTab = 1
	}
	counter := 0
	for _, r := range e.Line(n) {
		if unicode.IsSpace(r) {
			if r == '\t' {
				counter += spacesPerTab
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

// FirstDataPosition returns the first X index for this line, that is not whitespace.
func (e *Editor) FirstDataPosition(n int) int {
	counter := 0
	for _, r := range e.Line(n) {
		if !unicode.IsSpace(r) {
			break
		}
		counter++
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
	lastIndex := len([]rune(e.lines[n])) - 1
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
	tabString := " "
	if !e.DrawMode() {
		tabString = strings.Repeat(" ", e.spacesPerTab)
	}
	w := int(c.Width())
	if fromline >= toline {
		return errors.New("fromline >= toline in WriteLines")
	}
	numlines := toline - fromline
	offset := fromline
	for y := 0; y < numlines; y++ {
		counter := 0
		//line := strings.ReplaceAll(e.Line(y+offset), "\t", tabString)
		line := strings.Replace(e.Line(y+offset), "\t", tabString, -1)
		screenLine := strings.TrimRightFunc(line, unicode.IsSpace)
		if len([]rune(screenLine)) >= w {
			screenLine = screenLine[:w]
		}
		if e.highlight {
			// Output a syntax highlighted line
			//vt100.SetXY(uint(cx+counter), uint(cy+y))
			if textWithTags, err := syntax.AsText([]byte(line)); err != nil {
				// Only output the line up to the width of the canvas
				fmt.Println(screenLine)
				counter += len([]rune(screenLine))
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
						if e.DrawMode() {
							counter++
						} else {
							counter += e.spacesPerTab
						}
					} else {
						c.WriteRune(uint(cx+counter), uint(cy+y), fg, e.bg, letter)
						counter++
					}
				}
			}
		} else {
			// Output a regular line
			c.Write(uint(cx+counter), uint(cy+y), e.fg, e.bg, screenLine)
			counter += len([]rune(screenLine))
		}
		//length := len([]rune(screenLine)) + strings.Count(screenLine, "\t")*(e.spacesPerTab-1)
		// Fill the rest of the line on the canvas with "blanks"
		for x := counter; x < w; x++ {
			c.WriteRune(uint(cx+x), uint(cy+y), e.fg, e.bg, ' ')
		}
	}
	return nil
}

// DeleteRestOfLine will delete the rest of the line, from the given position
func (e *Editor) DeleteRestOfLine() {
	x, err := e.DataX()
	if err != nil {
		// position is after the data, do nothing
		return
	}
	y := e.DataY()
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[y]
	if !ok {
		return
	}
	if x >= len([]rune(e.lines[y])) {
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
		// This should never happen
		vt100.Clear()
		vt100.Close()
		panic(err)
	}

	e.changed = true
}

// Delete will delete a character at the given position
func (e *Editor) Delete() {
	y := e.DataY()
	llen := len([]rune(e.lines[y]))
	if _, ok := e.lines[y]; !ok || llen == 0 || llen == 1 && unicode.IsSpace(e.lines[y][0]) {
		// All keys in the map that are > y should be shifted -1.
		// This also overwrites e.lines[y].
		e.DeleteLine(y)
		e.changed = true
		return
	}
	x, err := e.DataX()
	if err != nil || x >= len([]rune(e.lines[y]))-1 {
		// on the last index, just use every element but x
		e.lines[y] = e.lines[y][:x]
		// check if the next line exists
		if _, ok := e.lines[y+1]; ok {
			// then add the contents of the next line, if available
			nextLine, ok := e.lines[y+1]
			if ok && len([]rune(nextLine)) > 0 {
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
		// This should never happen
		vt100.Clear()
		vt100.Close()
		panic(err)
	}

	e.changed = true
}

// Empty will check if the current editor contents are empty or not.
// If there's only one line left and it is only whitespace, that will be considered empty as well.
func (e *Editor) Empty() bool {
	l := len(e.lines)
	switch l {
	case 0:
		return true
	case 1:
		// Check the contents of the 1 remaining line,
		// without specifying a key.
		for _, v := range e.lines {
			if len(strings.TrimSpace(string(v))) == 0 {
				return true
			}
			break
		}
		fallthrough
	default:
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

// InsertLineAbove will attempt to insert a new line below the current position
func (e *Editor) InsertLineAbove() {
	y := e.DataY()

	// Create new set of lines
	lines2 := make(map[int][]rune)

	// For each line in the old map, if at (y-1), insert a blank line
	// (insert a blank line above)
	for k, v := range e.lines {
		if k < (y - 1) {
			lines2[k] = v
		} else if k == (y - 1) {
			lines2[k] = v
			lines2[k+1] = make([]rune, 0)
		} else if k > (y - 1) {
			lines2[k+1] = v
		}
	}
	// Use the new set of lines
	e.lines = lines2

	e.MakeConsistent()

	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len([]rune(e.lines[i])) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		// This should never happen
		vt100.Clear()
		vt100.Close()
		panic(err)
	}

	e.changed = true
}

// InsertLineBelow will attempt to insert a new line below the current position
func (e *Editor) InsertLineBelow() {
	y := e.DataY()

	// Create new set of lines
	lines2 := make(map[int][]rune)

	// For each line in the old map, if at (y-1), insert a blank line
	// (insert a blank line above)
	for k, v := range e.lines {
		if k < y {
			lines2[k] = v
		} else if k == y {
			lines2[k] = v
			lines2[k+1] = make([]rune, 0)
		} else if k > y {
			lines2[k+1] = v
		}
	}
	// Use the new set of lines
	e.lines = lines2

	e.MakeConsistent()

	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len([]rune(e.lines[i])) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		// This should never happen
		vt100.Clear()
		vt100.Close()
		panic(err)
	}

	e.changed = true
}

// Insert will insert a rune at the given position
func (e *Editor) Insert(r rune) {
	// Ignore it if the current position is out of bounds
	x, _ := e.DataX()

	y := e.DataY()

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
	if len([]rune(e.lines[y])) < x {
		// Can only insert in the existing block of text
		return
	}
	newlineLength := len(e.lines[y]) + 1
	newline := make([]rune, newlineLength)
	for i := 0; i < x; i++ {
		newline[i] = e.lines[y][i]
	}
	newline[x] = r
	for i := x + 1; i < newlineLength; i++ {
		newline[i] = e.lines[y][i-1]
	}
	e.lines[y] = newline

	// Check if the keys in the map are consistent
	if err := e.MakeConsistent(); err != nil {
		// This should never happen
		vt100.Clear()
		vt100.Close()
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
		// This should never happen
		vt100.Clear()
		vt100.Close()
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
	return len(strings.Fields(e.String()))
}

// ToggleHighlight toggles syntax highlighting
func (e *Editor) ToggleHighlight() {
	e.highlight = !e.highlight
}

// SetHighlight enables or disables syntax highlighting
func (e *Editor) SetHighlight(highlight bool) {
	e.highlight = highlight
}

// SetLine will fill the given line index with the given string.
// Any previous contents of that line is removed.
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

// SplitLine will, at the given position, split the line in two.
// The right side of the contents is moved to a new line below.
func (e *Editor) SplitLine() bool {
	x, err := e.DataX()
	if err != nil {
		// After contents, this should not happen, do nothing
		return false
	}

	y := e.DataY()

	// Get the contents of this line
	runeLine := e.lines[y]
	if len(runeLine) < 2 {
		// Did not split
		return false
	}
	leftContents := strings.TrimRightFunc(string(runeLine[:x]), unicode.IsSpace)
	rightContents := string(runeLine[x:])
	// Insert a new line above this one
	e.InsertLineAbove()
	// Replace this line with the left contents
	e.SetLine(y, leftContents)
	e.SetLine(y+1, rightContents)
	// Splitted
	return true
}

// DataX will return the X position in the data (as opposed to the X position in the viewport)
func (e *Editor) DataX() (int, error) {
	if e.drawMode {
		return e.pos.sx, nil
	}
	// the y position in the data is the lines scrolled + current screen cursor Y position
	dataY := e.pos.scroll + e.pos.sy
	// get the current line of text
	screenCounter := 0 // counter for the characters on the screen
	// loop, while also keeping track of tab expansion
	// add a space to allow to jump to the position after the line and get a valid data position
	found := false
	dataX := 0
	runeCounter := 0
	for _, r := range e.lines[dataY] {
		// When we reached the correct screen position, use i as the data position
		if screenCounter == e.pos.sx {
			dataX = runeCounter
			found = true
			break
		}
		// Increase the counter, based on the current rune
		if r == '\t' {
			screenCounter += e.spacesPerTab
		} else {
			screenCounter++
		}
		runeCounter++
	}
	if !found {
		return runeCounter, errors.New("position is after data")
	}
	// Return the data cursor
	return dataX, nil
}

// DataY will return the Y position in the data (as opposed to the Y position in the viewport)
func (e *Editor) DataY() int {
	if e.drawMode {
		return e.pos.sy
	}
	return e.pos.scroll + e.pos.sy
}

// SetRune will set a rune at the current data position
func (e *Editor) SetRune(r rune) {
	// Only set a rune if x is within the current line contents
	if x, err := e.DataX(); err == nil {
		e.Set(x, e.DataY(), r)
	}
}

// InsertRune will insert a rune at the current data position
func (e *Editor) InsertRune(r rune) {
	e.Insert(r)
}

// Rune will get the rune at the current data position
func (e *Editor) Rune() rune {
	x, err := e.DataX()
	if err != nil {
		// after line contents, return a zero rune
		return rune(0)
	}
	return e.Get(x, e.DataY())
}

// CurrentLine will get the current data line, as a string
func (e *Editor) CurrentLine() string {
	return e.Line(e.DataY())
}

// Home will move the cursor the the start of the line (x = 0)
func (e *Editor) Home() {
	e.pos.sx = 0
}

// End will move the cursor to the position right after the end of the cirrent line contents
func (e *Editor) End() {
	e.pos.sx = e.LastScreenPosition(e.DataY()) + 1
}

func (e *Editor) AtEndOfLine() bool {
	return e.pos.sx == e.LastScreenPosition(e.DataY())
}

//func (e *Editor) AtOrAfterEndOfLine() bool {
//	return e.pos.sx >= e.LastScreenPosition(e.DataY())
//}

// DownEnd will move down and then choose a "smart" X position
func (e *Editor) DownEnd(c *vt100.Canvas) error {
	tmpx := e.pos.sx
	err := e.pos.Down(c)
	if err != nil {
		return err
	}
	if e.AfterLineContentsPlusOne() && tmpx > 1 {
		e.End()
		if e.pos.sx != tmpx && e.pos.sx > e.pos.savedX {
			e.pos.savedX = tmpx
		}
	} else {
		e.pos.sx = e.pos.savedX
		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if e.Rune() == '\t' {
			e.pos.sx = e.FirstScreenPosition(e.DataY())
		}
	}
	return nil
}

// UpEnd will move up and then choose a "smart" X position
func (e *Editor) UpEnd(c *vt100.Canvas) error {
	tmpx := e.pos.sx
	err := e.pos.Up()
	if err != nil {
		return err
	}
	if e.AfterLineContentsPlusOne() && tmpx > 1 {
		e.End()
		if e.pos.sx != tmpx && e.pos.sx > e.pos.savedX {
			e.pos.savedX = tmpx
		}
	} else {
		e.pos.sx = e.pos.savedX
		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if e.Rune() == '\t' {
			e.pos.sx = e.FirstScreenPosition(e.DataY())
		}
	}
	return nil
}

// Next will move the cursor to the next position in the contents
func (e *Editor) Next(c *vt100.Canvas) error {
	// Ignore it if the position is out of bounds
	x, _ := e.DataX()
	atTab := '\t' == e.Get(x, e.DataY())
	if atTab && !e.DrawMode() {
		e.pos.sx += e.spacesPerTab
	} else {
		e.pos.sx++
	}
	// Did we move too far on this line?
	w := int(c.W())
	if (!e.DrawMode() && e.AfterLineContentsPlusOne()) || (e.DrawMode() && e.pos.sx >= w) {
		// Undo the move
		if atTab && !e.DrawMode() {
			e.pos.sx -= e.spacesPerTab
		} else {
			e.pos.sx--
		}
		// Move down
		if !e.DrawMode() {
			err := e.pos.Down(c)
			if err != nil {
				return err
			}
			// Move to the start of the line
			e.pos.sx = 0
		}
	}
	return nil
}

// Prev will move the cursor to the previous position in the contents
func (e *Editor) Prev(c *vt100.Canvas) error {
	atTab := false
	// Ignore it if the position is out of bounds
	x, _ := e.DataX()
	if x > 0 {
		atTab = '\t' == e.Get(x-1, e.DataY())
	}
	// If at a tab character, move a few more posisions
	if atTab && !e.DrawMode() {
		e.pos.sx -= e.spacesPerTab
	} else {
		e.pos.sx--
	}
	// Did we move too far?
	if e.pos.sx < 0 {
		// Undo the move
		if atTab && !e.DrawMode() {
			e.pos.sx += e.spacesPerTab
		} else {
			e.pos.sx++
		}
		// Move up, and to the end of the line above, if in EOL mode
		if !e.DrawMode() {
			err := e.pos.Up()
			if err != nil {
				return err
			}
			e.End()
		}
	}
	return nil
}

// Right will move the cursor to the right, if possible.
// It will not move the cursor up or down.
func (p *Position) Right(c *vt100.Canvas) {
	lastX := int(c.Width() - 1)
	if p.sx < lastX {
		p.sx++
	}
}

// Left will move the cursor to the left, if possible.
// It will not move the cursor up or down.
func (p *Position) Left() {
	if p.sx > 0 {
		p.sx--
	}
}

// SaveX will save the current X position, if it's within reason
func (e *Editor) SaveX(regardless bool) {
	if regardless || (!e.AfterLineContentsPlusOne() && e.pos.sx > 1) {
		e.pos.savedX = e.pos.sx
	}
}

// ScrollDown will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollDown(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed
	// last y posision in the canvas
	canvasLastY := int(c.H() - 1)
	// number of lines in the document
	l := e.Len()
	if e.pos.scroll >= e.Len()-canvasLastY {
		// Status message
		//status.SetMessage("End of text")
		//status.Show(c, p)
		c.Draw()
		// Don't redraw
		return false
	}
	status.Clear(c)
	if (e.pos.scroll + canScroll) >= (l - canvasLastY) {
		// Almost at the bottom, we can scroll the remaining lines
		canScroll = (l - canvasLastY) - e.pos.scroll
	}
	// Move the scroll offset
	e.pos.scroll += canScroll
	// Prepare to redraw
	return true
}

// ScrollUp will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollUp(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed
	if e.pos.scroll == 0 {
		// Can't scroll further up
		// Status message
		//status.SetMessage("Start of text")
		//status.Show(c, p)
		c.Draw()
		// Don't redraw
		return false
	}
	status.Clear(c)
	if e.pos.scroll-canScroll < 0 {
		// Almost at the top, we can scroll the remaining lines
		canScroll = e.pos.scroll
	}
	// Move the scroll offset
	e.pos.scroll -= canScroll
	// Prepare to redraw
	return true
}

// AtLastLineOfDocument is true if we're at the last line of the document (or beyond)
func (e *Editor) AtLastLineOfDocument() bool {
	return e.DataY() >= (e.Len() - 1)
}

// AtOrAfterEndOfDocument is true if the cursor is at or after the end of the last line of the documebnt
func (e *Editor) AtOrAfterEndOfDocument() bool {
	return e.AtLastLineOfDocument() && e.AtOrAfterEndOfLine()
}

// AtEndOfDocument is true if the cursor is at the end of the last line of the documebnt
func (e *Editor) AtEndOfDocument() bool {
	return e.AtLastLineOfDocument() && e.AtEndOfLine()
}

// StartOfDocument is true if we're at the first line of the document
func (e *Editor) AtStartOfDocument() bool {
	return e.pos.sy == 0 && e.pos.scroll == 0
}

// Is the cursor at or after the contents of this line?
func (e *Editor) AtOrAfterEndOfLine() bool {
	x, err := e.DataX()
	if err != nil {
		// After end of data
		return true
	}
	return x >= e.LastDataPosition(e.DataY())
}

// AfterLineContents will check if the cursor is after the current line contents
func (e *Editor) AfterLineContents() bool {
	return e.pos.sx > e.LastScreenPosition(e.DataY())
}

// AfterLineContentsPlusOne will check if the cursor is after the current line contents, with a margin of 1
func (e *Editor) AfterLineContentsPlusOne() bool {
	return e.pos.sx > (e.LastScreenPosition(e.DataY()) + 1)
}

// WriteRune writes the current rune to the given canvas
func (e *Editor) WriteRune(c *vt100.Canvas) {
	c.WriteRune(uint(e.pos.sx), uint(e.pos.sy), e.fg, e.bg, e.Rune())
}

// WriteTab writes spaces when there is a tab character, to the canvas
func (e *Editor) WriteTab(c *vt100.Canvas) {
	spacesPerTab := e.spacesPerTab
	if e.DrawMode() {
		spacesPerTab = 1
	}
	for x := e.pos.sx; x < e.pos.sx+spacesPerTab; x++ {
		c.WriteRune(uint(x), uint(e.pos.sy), e.fg, e.bg, ' ')
	}
}

// EmptyLine checks if the current line is empty (and whitespace doesn't count)
func (e *Editor) EmptyLine() bool {
	return 0 == len(strings.TrimRightFunc(e.CurrentLine(), unicode.IsSpace))
}

// AtStartOfText returns true if the position is at the start of the text for this line
func (e *Editor) AtStartOfTextLine() bool {
	return e.pos.sx == e.FirstScreenPosition(e.DataY())
}

// BeforeStartOfText returns true if the position is before the start of the text for this line
func (e *Editor) BeforeStartOfTextLine() bool {
	return e.pos.sx < e.FirstScreenPosition(e.DataY())
}

// BeforeOrAtStartOfText returns true if the position is before or at the start of the text for this line
func (e *Editor) AtOrBeforeStartOfTextLine() bool {
	return e.pos.sx <= e.FirstScreenPosition(e.DataY())
}

// GoToLineNumber will go to a given line number, but counting from 1, not from 0!
func (e *Editor) GoToLineNumber(lineNumber int, c *vt100.Canvas, status *StatusBar) bool {
	return e.GoTo(lineNumber-1, c, status)
}

// GoTo will go to a given line index, counting from 0
func (e *Editor) GoTo(y int, c *vt100.Canvas, status *StatusBar) bool {
	redraw := false
	h := int(c.Height())
	if y >= h {
		e.pos.sy = 0
		e.pos.scroll = 0
		e.ScrollDown(c, status, y)
		e.pos.SetX(e.FirstScreenPosition(e.DataY()))
		redraw = true
	} else {
		e.pos.sy = 0
		e.pos.scroll = 0
		// Jump to the given line number, if > 0
		for i := 0; i < y; i++ {
			e.pos.Down(c)
		}
		e.pos.SetX(e.FirstScreenPosition(e.DataY()))
		redraw = true
	}
	return redraw
}

// Up tried to move the cursor up, and also scroll
func (e *Editor) Up(c *vt100.Canvas, status *StatusBar) {
	e.GoTo(e.DataY()-1, c, status)
}

// Down tries to move the cursor down, and also scroll
func (e *Editor) Down(c *vt100.Canvas, status *StatusBar) {
	e.GoTo(e.DataY()+1, c, status)
}

// LeadingWhitespace returns the leading whitespace for this line
func (e *Editor) LeadingWhitespace() string {
	return e.CurrentLine()[:e.FirstDataPosition(e.DataY())]
}

// LineNumber will return the current line number (data y index + 1)
func (e *Editor) LineNumber() int {
	return e.DataY() + 1
}

// ColumNumber will return the current column number (data x index + 1)
func (e *Editor) ColumnNumber() int {
	x, _ := e.DataX()
	return x + 1
}

// StatusMessage returns a status message, intended for being displayed at the bottom
func (e *Editor) StatusMessage() string {
	return fmt.Sprintf("line %d col %d rune %U words %d", e.LineNumber(), e.ColumnNumber(), e.Rune(), e.WordCount())
}
