package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

// EditorColors is a collection of a few selected colors, for setting up a new editor
type EditorColors struct {
	fg               vt100.AttributeColor // default foreground color
	bg               vt100.AttributeColor // default background color
	searchFg         vt100.AttributeColor // search highlight color
	gitColor         vt100.AttributeColor // git commit message color
	multiLineComment vt100.AttributeColor // color for multiLine comments
	multiLineString  vt100.AttributeColor // colo for multiLine strings
}

// Editor represents the contents and editor settings, but not settings related to the viewport or scrolling
type Editor struct {
	lines              map[int][]rune        // the contents of the current document
	changed            bool                  // has the contents changed, since last save?
	tabsSpaces         TabsSpaces            // spaces or tabs, and how many spaces per tab character
	syntaxHighlight    bool                  // syntax highlighting
	rainbowParenthesis bool                  // rainbow parenthesis
	pos                Position              // the current cursor and scroll position
	searchTerm         string                // the current search term, used when searching
	stickySearchTerm   string                // for going to the next match with ctrl-n, unless esc has been pressed
	redraw             bool                  // if the contents should be redrawn in the next loop
	redrawCursor       bool                  // if the cursor should be moved to the location it is supposed to be
	lineBeforeSearch   LineIndex             // save the current line when jumping between search results
	wrapWidth          int                   // set to 80 or 100 to trigger word wrap when typing to that column
	mode               Mode                  // a filetype mode, like for git or markdown
	filename           string                // the current filename
	locationHistory    map[string]LineNumber // location history, for jumping to the last location when opening a file
	quit               bool                  // for indicating if the user wants to end the editor session
	clearOnQuit        bool                  // clear the terminal when quitting the editor, or not
	wrapWhenTyping     bool                  // wrap text at a certain limit when typing
	lightTheme         bool                  // using a light theme? (the XTERM_VERSION environment variable is set)
	noColor            bool                  // should no color be used?
	firstLineHash      bool                  // is the first line starting with "#"?
	slowLoad           bool                  // was the initial file slow to load? (might be an indication of a slow disk or USB stick)
	readOnly           bool                  // is the file read-only when initializing o?
	EditorColors
}

// Used when switching between a .c or .cpp file to the corresponding .h file
var (
	switchBuffer     *Undo = NewUndo(1)               // Save the contents of one switch
	switchUndoBackup *Undo = NewUndo(defaultUndoSize) // Save a copy of the undo stack when switching between files
)

// NewCustomEditor takes:
// * the number of spaces per tab (typically 2, 4 or 8)
// * if the text should be syntax highlighted
// * if rainbow parenthesis should be enabled
// * if text edit mode is enabled (as opposed to "ASCII draw mode")
// * the current scroll speed, in lines
// * the following colors:
//    - text foreground
//    - text background
//    - search highlight
//    - multiline comment
// * a syntax highlighting scheme
// * a file mode
func NewCustomEditor(tabsSpaces TabsSpaces, syntaxHighlight, rainbowParenthesis bool, scrollSpeed int, fg, bg, searchFg, multiLineComment, multiLineString vt100.AttributeColor, scheme syntax.TextConfig, mode Mode, theme Theme) *Editor {
	syntax.DefaultTextConfig = scheme
	e := &Editor{}
	e.lines = make(map[int][]rune)
	e.fg = fg
	e.bg = bg
	e.tabsSpaces = tabsSpaces
	e.syntaxHighlight = syntaxHighlight
	e.rainbowParenthesis = rainbowParenthesis
	p := NewPosition(scrollSpeed)
	e.pos = *p
	e.searchFg = searchFg
	// If the file is not to be highlighted, set word wrap to 99 (0 to disable)
	if !syntaxHighlight {
		e.wrapWidth = 99
		e.wrapWhenTyping = true
	}
	if mode == modeGit {
		// The subject should ideally be maximum 50 characters long, then the body of the
		// git commit message can be 72 characters long. Because e-mail standards.
		e.wrapWidth = 72
		e.wrapWhenTyping = true
	} else if mode == modeMarkdown {
		e.wrapWidth = 99
		e.wrapWhenTyping = false
	} else if mode == modeText || mode == modeBlank {
		e.wrapWidth = 99
		e.wrapWhenTyping = false
	}
	e.mode = mode
	e.multiLineComment = multiLineComment
	e.multiLineString = multiLineString

	// Check if NO_COLOR is set
	e.respectNoColorEnvironmentVariable()

	// Check if a specific theme is set
	switch theme {
	case redBlackTheme:
		e.setRedBlackTheme()
	case lightTheme:
		e.setLightTheme()
	case defaultTheme:
		break
	}

	return e
}

// NewSimpleEditor return a new simple editor, where the settings are 4 spaces per tab, white text on black background,
// no syntax highlighting, text edit mode (as opposed to ASCII draw mode), scroll 1 line at a time, color
// search results magenta, use the default syntax highlighting scheme, don't use git mode and don't use markdown mode,
// then set the word wrap limit at the given column width.
func NewSimpleEditor(wordWrapLimit int) *Editor {
	e := NewCustomEditor(defaultTabsSpaces, false, false, 1, vt100.White, vt100.Black, vt100.Magenta, vt100.Gray, vt100.Magenta, syntax.DefaultTextConfig, modeBlank, defaultTheme)
	e.wrapWidth = wordWrapLimit
	e.wrapWhenTyping = true
	return e
}

// CopyLines will create a new map[int][]rune struct that is the copy of all the lines in the editor
func (e *Editor) CopyLines() map[int][]rune {
	lines2 := make(map[int][]rune)
	for key, runes := range e.lines {
		runes2 := make([]rune, len(runes))
		copy(runes2, runes)
		lines2[key] = runes2
	}
	return lines2
}

// Set will store a rune in the editor data, at the given data coordinates
func (e *Editor) Set(x int, index LineIndex, r rune) {
	y := int(index)
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[y]
	if !ok {
		e.lines[y] = make([]rune, 0, x+1)
	}
	l := len(e.lines[y])
	if x < l {
		e.lines[y][x] = r
		e.changed = true
		return
	}
	// If the line is too short, fill it up with spaces
	if l <= x {
		n := (x + 1) - l
		e.lines[y] = append(e.lines[y], []rune(strings.Repeat(" ", n))...)
	}

	// Set the rune
	e.lines[y][x] = r
	e.changed = true
}

// Get will retrieve a rune from the editor data, at the given coordinates
func (e *Editor) Get(x int, y LineIndex) rune {
	if e.lines == nil {
		return ' '
	}
	runes, ok := e.lines[int(y)]
	if !ok {
		return ' '
	}
	if x >= len(runes) {
		return ' '
	}
	return runes[x]
}

// Changed will return true if the contents were changed since last time this function was called
func (e *Editor) Changed() bool {
	return e.changed
}

// Line returns the contents of line number N, counting from 0
func (e *Editor) Line(n LineIndex) string {
	line, ok := e.lines[int(n)]
	if ok {
		var sb strings.Builder
		for _, r := range line {
			sb.WriteRune(r)
		}
		return sb.String()
	}
	return ""
}

// ScreenLine returns the screen contents of line number N, counting from 0.
// The tabs are expanded.
func (e *Editor) ScreenLine(n int) string {
	line, ok := e.lines[n]
	if ok {
		var sb strings.Builder
		skipX := e.pos.offsetX
		for _, r := range line {
			if skipX > 0 {
				skipX--
				continue
			}
			sb.WriteRune(r)
		}
		tabSpace := strings.Repeat("\t", e.tabsSpaces.perTab)
		return strings.Replace(sb.String(), "\t", tabSpace, -1)
	}
	return ""
}

// LastDataPosition returns the last X index for this line, for the data (does not expand tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastDataPosition(n LineIndex) int {
	return len([]rune(e.Line(n))) - 1
}

// LastScreenPosition returns the last X index for this line, for the screen (expands tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastScreenPosition(n LineIndex) int {
	extraSpaceBecauseOfTabs := int(e.CountRune('\t', n) * (e.tabsSpaces.perTab - 1))
	return (e.LastDataPosition(n) + extraSpaceBecauseOfTabs)
}

// LastTextPosition returns the last X index for this line, regardless of horizontal scrolling.
// Can be negative if the line is empty. Tabs are expanded.
func (e *Editor) LastTextPosition(n LineIndex) int {
	extraSpaceBecauseOfTabs := int(e.CountRune('\t', n) * (e.tabsSpaces.perTab - 1))
	return (e.LastDataPosition(n) + extraSpaceBecauseOfTabs)
}

// FirstScreenPosition returns the first X index for this line, that is not '\t' or ' '.
// Does not deal with the X offset.
func (e *Editor) FirstScreenPosition(n LineIndex) uint {
	var (
		counter      uint
		spacesPerTab = uint(e.tabsSpaces.perTab)
	)
	for _, r := range e.Line(n) {
		if r == '\t' {
			counter += spacesPerTab
		} else if r == ' ' {
			counter++
		} else {
			break
		}
	}
	return counter
}

// FirstDataPosition returns the first X index for this line, that is not whitespace.
func (e *Editor) FirstDataPosition(n LineIndex) int {
	counter := 0
	for _, r := range e.Line(n) {
		if !unicode.IsSpace(r) {
			break
		}
		counter++
	}
	return counter
}

// CountRune will count the number of instances of the rune r in the line n
func (e *Editor) CountRune(r rune, n LineIndex) int {
	var counter int
	line, ok := e.lines[int(n)]
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
	l := e.Len()
	for i := 0; i < l; i++ {
		sb.WriteString(e.Line(LineIndex(i)) + "\n")
	}
	return sb.String()
}

// Clear removes all data from the editor
func (e *Editor) Clear() {
	e.lines = make(map[int][]rune)
	e.changed = true
}

// Load will try to load a file. The file is assumed to be checked to already exist.
// Returns a warning message (possibly empty) and an error type
func (e *Editor) Load(c *vt100.Canvas, tty *vt100.TTY, filename string) (string, error) {
	var message string

	// Start a spinner, in a short while
	quitChan := Spinner(c, tty, fmt.Sprintf("Reading %s... ", filename), fmt.Sprintf("reading %s: stopped by user", filename), e.noColor, 200*time.Millisecond)

	start := time.Now()

	// Read the file and check if it could be read
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		// Stop the spinner and return
		quitChan <- true
		return message, err
	}

	// If enough time passed so that the spinner was shown by now, enter "slow disk mode" where fewer disk-related I/O operations will be performed
	e.slowLoad = time.Since(start) > 400*time.Millisecond

	// Opinonated replacements
	data = opinionatedByteReplacer(data)

	// Load the data
	e.LoadBytes(data)

	// Stop the spinner
	quitChan <- true

	// Mark the data as "not changed"
	e.changed = false

	return message, nil
}

// LoadBytes replaces the current editor contents with the given bytes
func (e *Editor) LoadBytes(data []byte) {
	e.Clear()

	byteLines := bytes.Split(data, []byte{'\n'})

	lb := len(byteLines)

	// If the last line is empty, skip it
	if len(byteLines) > 0 && len(byteLines[lb-1]) == 0 {
		byteLines = byteLines[:lb-1]
		lb--
	}

	// One allocation for all the lines
	e.lines = make(map[int][]rune, lb)

	// Place the lines into the editor
	for y, byteLine := range byteLines {
		e.lines[y] = []rune(string(byteLine))
	}

	// Mark the editor contents as "changed"
	e.changed = true
}

// PrepareEmpty prepares an empty textual representation of a given filename.
// If it's an image, there will be text placeholders for pixels.
// If it's anything else, it will just be blank.
// Returns an editor mode and an error type.
func (e *Editor) PrepareEmpty(c *vt100.Canvas, tty *vt100.TTY, filename string) (Mode, error) {
	var (
		mode Mode = modeBlank
		data []byte
		err  error
	)

	// Check if the data could be prepared
	if err != nil {
		return mode, err
	}

	lines := strings.Split(string(data), "\n")
	e.Clear()
	for y, line := range lines {
		counter := 0
		for _, letter := range line {
			e.Set(counter, LineIndex(y), letter)
			counter++
		}
	}
	// Mark the data as "not changed"
	e.changed = false

	return mode, nil
}

// Save will try to save the current editor contents to file.
// It needs a canvas in case trailing spaces are stripped and the cursor needs to move to the end.
func (e *Editor) Save(c *vt100.Canvas, tty *vt100.TTY) error {

	// Save the current position
	bookmark := e.pos.Copy()

	// Strip trailing spaces on all lines
	l := e.Len()
	changed := false
	for i := 0; i < l; i++ {
		if e.TrimRight(LineIndex(i)) {
			changed = true
		}
	}

	// Trim away trailing whitespace
	s := strings.TrimRightFunc(e.String(), unicode.IsSpace)

	// Make additional replacements, and add a final newline
	s = opinionatedStringReplacer.Replace(s) + "\n"

	// TODO: Auto-detect tabs/spaces instead of per-language assumptions
	if Spaces(e.mode) {
		// NOTE: This is a hack, that can only replace 10 levels deep.
		for level := 10; level > 0; level-- {
			fromString := "\n" + strings.Repeat("\t", level)
			toString := "\n" + strings.Repeat(" ", level*e.tabsSpaces.perTab)
			s = strings.Replace(s, fromString, toString, -1)
		}
	}

	// Should the file be saved with the executable bit enabled?
	// (Does it either start with a shebang or reside in a common bin directory like /usr/bin?)
	shebang := aBinDirectory(e.filename) || strings.HasPrefix(s, "#!")

	// Mark the data as "not changed"
	e.changed = false

	// Default file mode (0644 for regular files, 0755 for executable files)
	var fileMode os.FileMode = 0644

	// Checking the syntax highlighting makes it easy to press `ctrl-t` before saving a script,
	// to toggle the executable bit on or off. This is only for files that start with "#!".
	// Also, if the file is in one of the common bin directories, like "/usr/bin", then assume that it
	// is supposed to be executable.
	if shebang && e.syntaxHighlight {
		// This is both a script file and the syntax highlight is enabled.
		fileMode = 0755
	}

	// Start a spinner, in a short while
	quitChan := Spinner(c, tty, fmt.Sprintf("Saving %s... ", e.filename), fmt.Sprintf("saving %s: stopped by user", e.filename), e.noColor, 200*time.Millisecond)

	// Save the file and return any errors
	if err := ioutil.WriteFile(e.filename, []byte(s), fileMode); err != nil {
		// Stop the spinner and return
		quitChan <- true
		return err
	}

	// Apparently, this file isn't read-only, since saving went fine
	e.readOnly = false

	// "chmod +x" or "chmod -x". This is needed after saving the file, in order to toggle the executable bit.
	// rust source may start with something like "#![feature(core_intrinsics)]", so avoid that.
	if shebang && e.mode != modeRust && !e.readOnly {
		// Call Chmod, but ignore errors (since this is just a bonus and not critical)
		os.Chmod(e.filename, fileMode)
		e.SetSyntaxHighlight(true)
	}

	// Stop the spinner
	quitChan <- true

	e.redrawCursor = true

	// Trailing spaces may be trimmed, so move to the end, if needed
	if changed {
		e.GoToPosition(c, nil, *bookmark)
		if e.AfterEndOfLine() {
			e.EndNoTrim(c)
		}
		// Do the redraw manually before showing the status message
		e.DrawLines(c, true, false)
		e.redraw = false
	}

	// All done
	return nil
}

// TrimRight will remove whitespace from the end of the given line number
// Returns true if the line was trimmed
func (e *Editor) TrimRight(index LineIndex) bool {
	changed := false
	n := int(index)
	if line, ok := e.lines[n]; ok {
		newRunes := []rune(strings.TrimRightFunc(string(line), unicode.IsSpace))
		// TODO: Just compare lengths instead of contents?
		if string(newRunes) != string(line) {
			e.lines[n] = newRunes
			changed = true
		}
	}
	return changed
}

// TrimLeft will remove whitespace from the start of the given line number
// Returns true if the line was trimmed
func (e *Editor) TrimLeft(index LineIndex) bool {
	changed := false
	n := int(index)
	if line, ok := e.lines[n]; ok {
		newRunes := []rune(strings.TrimLeftFunc(string(line), unicode.IsSpace))
		// TODO: Just compare lengths instead of contents?
		if string(newRunes) != string(line) {
			e.lines[n] = newRunes
			changed = true
		}
	}
	return changed
}

// StripSingleLineComment will strip away trailing single-line comments.
// TODO: Also strip trailing /* ... */ comments
func (e *Editor) StripSingleLineComment(line string) string {
	commentMarker := e.SingleLineCommentMarker()
	if strings.Count(line, commentMarker) == 1 {
		p := strings.Index(line, commentMarker)
		return strings.TrimSpace(line[:p])
	}
	return line
}

// DeleteRestOfLine will delete the rest of the line, from the given position
func (e *Editor) DeleteRestOfLine() {
	x, err := e.DataX()
	if err != nil {
		// position is after the data, do nothing
		return
	}
	y := int(e.DataY())
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	v, ok := e.lines[y]
	if !ok {
		return
	}
	if v == nil {
		e.lines[y] = make([]rune, 0)
	}
	if x > len([]rune(e.lines[y])) {
		return
	}
	e.lines[y] = e.lines[y][:x]
	e.changed = true
}

// DeleteLine will delete the given line index
func (e *Editor) DeleteLine(n LineIndex) {
	if n < 0 {
		// This should never happen
		return
	}
	lastLineIndex := LineIndex(e.Len() - 1)
	endOfDocument := n >= lastLineIndex
	if endOfDocument {
		// Just delete this line
		delete(e.lines, int(n))
		return
	}
	// TODO: Rely on the length of the hash map for finding the index instead of
	//       searching through each line number key.
	var maxIndex LineIndex
	found := false
	for k := range e.lines {
		if LineIndex(k) > maxIndex {
			maxIndex = LineIndex(k)
			found = true
		}
	}
	if !found {
		// This should never happen
		return
	}
	if _, ok := e.lines[int(maxIndex)]; !ok {
		// The line numbers and the length of e.lines does not match
		return
	}
	// Shift all lines after y:
	// shift all lines after n one step closer to n, overwriting e.lines[n]
	for index := n; index <= (maxIndex - 1); index++ {
		i := int(index)
		e.lines[i] = e.lines[i+1]
	}
	// Then delete the final item
	delete(e.lines, int(maxIndex))

	// This changes the document
	e.changed = true

	// Make sure no lines are nil
	e.MakeConsistent()
}

// DeleteLineMoveBookmark will delete the given line index and also move the bookmark if it's after n
func (e *Editor) DeleteLineMoveBookmark(n LineIndex, bookmark *Position) {
	if bookmark != nil && bookmark.LineIndex() > n {
		bookmark.DecY()
	}
	e.DeleteLine(n)
}

// DeleteCurrentLineMoveBookmark will delete the current line and also move the bookmark one up
// if it's after the current line.
func (e *Editor) DeleteCurrentLineMoveBookmark(bookmark *Position) {
	e.DeleteLineMoveBookmark(e.DataY(), bookmark)
}

// Delete will delete a character at the given position
func (e *Editor) Delete() {
	y := int(e.DataY())
	lineLen := len([]rune(e.lines[y]))
	if _, ok := e.lines[y]; !ok || lineLen == 0 || (lineLen == 1 && unicode.IsSpace(e.lines[y][0])) {
		// All keys in the map that are > y should be shifted -1.
		// This also overwrites e.lines[y].
		e.DeleteLine(LineIndex(y))
		e.changed = true
		return
	}
	x, err := e.DataX()
	if err != nil || x > len([]rune(e.lines[y]))-1 {
		// on the last index, just use every element but x
		e.lines[y] = e.lines[y][:x]
		// check if the next line exists
		if _, ok := e.lines[y+1]; ok {
			// then add the contents of the next line, if available
			nextLine, ok := e.lines[y+1]
			if ok && len([]rune(nextLine)) > 0 {
				e.lines[y] = append(e.lines[y], nextLine...)
				// then delete the next line
				e.DeleteLine(LineIndex(y + 1))
			}
		}
		e.changed = true
		return
	}
	// Delete just this character
	e.lines[y] = append(e.lines[y][:x], e.lines[y][x+1:]...)

	e.changed = true

	// Make sure no lines are nil
	e.MakeConsistent()
}

// Empty will check if the current editor contents are empty or not.
// If there's only one line left and it is only whitespace, that will be considered empty as well.
func (e *Editor) Empty() bool {
	l := len(e.lines)
	if l == 0 {
		return true
	}
	if l == 1 {
		// Regardless of line number key, check the contents of the one remaining trimmed line
		for _, line := range e.lines {
			return len(strings.TrimSpace(string(line))) == 0
		}
	}
	// > 1 lines
	return false
}

// MakeConsistent creates an empty slice of runes for any empty lines,
// to make sure that no line number below e.Len() points to a nil map.
func (e *Editor) MakeConsistent() {
	// Check if the keys in the map are consistent
	for i := 0; i < len(e.lines); i++ {
		if _, found := e.lines[i]; !found {
			e.lines[i] = make([]rune, 0)
			e.changed = true
		}
	}
}

// WithinLimit will check if a line is within the word wrap limit,
// given a Y position.
func (e *Editor) WithinLimit(y LineIndex) bool {
	return len(e.lines[int(y)]) < e.wrapWidth
}

// LastWord will return the last word of a line,
// given a Y position. Returns an empty string if there is no last word.
func (e *Editor) LastWord(y int) string {
	// TODO: Use a faster method
	words := strings.Fields(strings.TrimSpace(string(e.lines[y])))
	if len(words) > 0 {
		return words[len(words)-1]
	}
	return ""
}

// SplitOvershoot will split the line into a first part that is within the
// word wrap length and a second part that is the overshooting part.
// y is the line index (y position, counting from 0).
// isSpace is true if a space has just been inserted on purpose at the current position.
// returns true if there was a space at the split point.
func (e *Editor) SplitOvershoot(index LineIndex, isSpace bool) ([]rune, []rune, bool) {
	hasSpace := false

	y := int(index)

	// Maximum word length to not keep as one word
	maxDistance := e.wrapWidth / 2
	if e.WithinLimit(index) {
		return e.lines[y], make([]rune, 0), false
	}
	splitPosition := e.wrapWidth
	if isSpace {
		splitPosition, _ = e.DataX()
	} else {
		// Starting at the split position, move left until a space is reached (or the start of the line).
		// If a space is reached, check if it is too far away from n to be used as a split position, or not.
		spacePosition := -1
		for i := splitPosition; i >= 0; i-- {
			if i < len(e.lines[y]) && unicode.IsSpace(e.lines[y][i]) {
				// Found a space at position i
				spacePosition = i
				break
			}
		}
		// Found a better position to split, at a nearby space?
		if spacePosition != -1 {
			hasSpace = true
			distance := splitPosition - spacePosition
			if distance > maxDistance {
				// To far away, don't use this as a split point,
				// stick to the hard split.
			} else {
				// Okay, we found a better split point.
				splitPosition = spacePosition
			}
		}
	}

	// Split the line into two parts

	n := splitPosition
	// Make space for the two parts
	first := make([]rune, len(e.lines[y][:n]))
	second := make([]rune, len(e.lines[y][n:]))
	// Copy the line into first and second
	copy(first, e.lines[y][:n])
	copy(second, e.lines[y][n:])

	// If the second part starts with a space, remove it
	if len(second) > 0 && unicode.IsSpace(second[0]) {
		second = second[1:]
		hasSpace = true
	}

	return first, second, hasSpace
}

// WrapAllLinesAt will word wrap all lines that are longer than n,
// with a maximum overshoot of too long words (measured in runes) of maxOvershoot.
// Returns true if any lines were wrapped.
func (e *Editor) WrapAllLinesAt(n, maxOvershoot int) bool {
	// This is not even called when the problematic insert behavior occurs

	wrapped := false
	insertedLines := 0

	y := e.DataY()

	for i := 0; i < e.Len(); i++ {
		if e.WithinLimit(LineIndex(i)) {
			continue
		}
		wrapped = true

		first, second, spaceBetween := e.SplitOvershoot(LineIndex(i), false)

		if len(first) > 0 && len(second) > 0 {

			e.lines[i] = first
			if spaceBetween {
				second = append(second, ' ')
			}
			e.lines[i+1] = append(second, e.lines[i+1]...)
			e.InsertLineBelowAt(LineIndex(i + 1))

			// This isn't perfect, but it helps move the cursor somewhere in
			// the vicinity of where the line was before word wrapping.
			// TODO: Make the cursor placement exact.
			if LineIndex(i) < y {
				insertedLines++
			}

			e.changed = true
		}
	}

	// Move the cursor as well, after wrapping
	if insertedLines > 0 {
		e.pos.sy += insertedLines
		if e.pos.sy < 0 {
			e.pos.sy = 0
		} else if e.pos.sy >= len(e.lines) {
			e.pos.sy = len(e.lines) - 1
		}
		e.redraw = true
		e.redrawCursor = true
	}

	return wrapped
}

// InsertLineAbove will attempt to insert a new line above the current position
func (e *Editor) InsertLineAbove() {
	y := int(e.DataY())

	// Create new set of lines
	lines2 := make(map[int][]rune)

	// If at the first line, just add a line at the top
	if y == 0 {

		// Insert a blank line
		lines2[0] = make([]rune, 0)
		// Then insert all the other lines, shifted by 1
		for k, v := range e.lines {
			lines2[k+1] = v
		}
		y++

	} else {

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

	}

	// Use the new set of lines
	e.lines = lines2

	// Make sure no lines are nil
	e.MakeConsistent()

	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len([]rune(e.lines[i])) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}
	e.changed = true
}

// InsertLineBelow will attempt to insert a new line below the current position
func (e *Editor) InsertLineBelow() {
	e.InsertLineBelowAt(e.DataY())
}

// InsertLineBelowAt will attempt to insert a new line below the given y position
func (e *Editor) InsertLineBelowAt(index LineIndex) {
	y := int(index)

	// Make sure no lines are nil
	e.MakeConsistent()

	// If we are the the last line, add an empty line at the end and return
	if y == (len(e.lines) - 1) {
		e.lines[int(y)+1] = make([]rune, 0)
		e.changed = true
		return
	}

	// Create new set of lines, with room for one more
	lines2 := make(map[int][]rune, len(e.lines)+1)

	// For each line in the old map, if at y, insert a blank line
	// (insert a blank line below)
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

	// Skip trailing newlines after this line
	for i := len(e.lines); i > y; i-- {
		if len([]rune(e.lines[i])) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}

	e.changed = true
}

// Insert will insert a rune at the given position, with no word wrap,
// but MakeConsisten will be called.
func (e *Editor) Insert(r rune) {
	// Ignore it if the current position is out of bounds
	x, _ := e.DataX()

	y := int(e.DataY())

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

	e.changed = true

	// Make sure no lines are nil
	e.MakeConsistent()
}

// CreateLineIfMissing will create a line at the given Y index, if it's missing
func (e *Editor) CreateLineIfMissing(n LineIndex) {
	if e.lines == nil {
		e.lines = make(map[int][]rune)
	}
	_, ok := e.lines[int(n)]
	if !ok {
		e.lines[int(n)] = make([]rune, 0)
		e.changed = true
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

// ToggleSyntaxHighlight toggles syntax highlighting
func (e *Editor) ToggleSyntaxHighlight() {
	e.syntaxHighlight = !e.syntaxHighlight
}

// SetSyntaxHighlight enables or disables syntax highlighting
func (e *Editor) SetSyntaxHighlight(syntaxHighlight bool) {
	e.syntaxHighlight = syntaxHighlight
}

// ToggleRainbow toggles rainbow parenthesis
func (e *Editor) ToggleRainbow() {
	e.rainbowParenthesis = !e.rainbowParenthesis
}

// SetRainbow enables or disables rainbow parenthesis
func (e *Editor) SetRainbow(rainbowParenthesis bool) {
	e.rainbowParenthesis = rainbowParenthesis
}

// SetLine will fill the given line index with the given string.
// Any previous contents of that line is removed.
func (e *Editor) SetLine(n LineIndex, s string) {
	e.CreateLineIfMissing(n)
	e.lines[int(n)] = make([]rune, 0)
	counter := 0
	// It's important not to use the index value when looping over a string,
	// unless the byte index is what one's after, as opposed to the rune index.
	for _, letter := range s {
		e.Set(counter, n, letter)
		counter++
	}
}

// SetCurrentLine will replace the current line with the given string
func (e *Editor) SetCurrentLine(s string) {
	e.SetLine(e.DataY(), s)
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
	runeLine := e.lines[int(y)]
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
	// the y position in the data is the lines scrolled + current screen cursor Y position
	dataY := e.pos.offsetY + e.pos.sy
	// get the current line of text
	screenCounter := 0 // counter for the characters on the screen
	// loop, while also keeping track of tab expansion
	// add a space to allow to jump to the position after the line and get a valid data position
	found := false
	dataX := 0
	runeCounter := 0
	for _, r := range e.lines[dataY] {
		// When we reached the correct screen position, use i as the data position
		if screenCounter == (e.pos.sx + e.pos.offsetX) {
			dataX = runeCounter
			found = true
			break
		}
		// Increase the counter, based on the current rune
		if r == '\t' {
			screenCounter += e.tabsSpaces.perTab
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
func (e *Editor) DataY() LineIndex {
	return LineIndex(e.pos.offsetY + e.pos.sy)
}

// SetRune will set a rune at the current data position
func (e *Editor) SetRune(r rune) {
	// Only set a rune if x is within the current line contents
	if x, err := e.DataX(); err == nil {
		e.Set(x, e.DataY(), r)
	}
}

// NextLine will go to the start of the next line, with scrolling
func (e *Editor) NextLine(y LineIndex, c *vt100.Canvas, status *StatusBar) {
	e.pos.sx = 0
	e.pos.offsetX = 0
	e.GoTo(y+1, c, status)
}

// InsertBelow will insert the given rune at the start of the line below,
// starting a new line if required.
func (e *Editor) InsertBelow(y int, r rune) {
	if _, ok := e.lines[y+1]; !ok {
		// If the next line does not exist, create one containing just "r"
		e.lines[y+1] = []rune{r}
	} else if len(e.lines[y+1]) > 0 {
		// If the next line is non-empty, insert "r" at the start
		e.lines[y+1] = append([]rune{r}, e.lines[y+1][:]...)
	} else {
		// The next line exists, but is of length 0, should not happen, just replace it
		e.lines[y+1] = []rune{r}
	}
}

// InsertStringBelow will insert the given string at the start of the line below,
// starting a new line if required.
func (e *Editor) InsertStringBelow(y int, s string) {
	if _, ok := e.lines[y+1]; !ok {
		// If the next line does not exist, create one containing the string
		e.lines[y+1] = []rune(s)
	} else if len(e.lines[y+1]) > 0 {
		// If the next line is non-empty, insert the string at the start
		e.lines[y+1] = append([]rune(s), e.lines[y+1][:]...)
	} else {
		// The next line exists, but is of length 0, should not happen, just replace it
		e.lines[y+1] = []rune(s)
	}
}

// InsertStringAndMove will insert a string at the current data position
// and possibly move down. This will also call e.WriteRune, e.Down and e.Next, as needed.
func (e *Editor) InsertStringAndMove(c *vt100.Canvas, s string) {
	for _, r := range s {
		if r == '\n' {
			e.InsertLineBelow()
			e.Down(c, nil)
			continue
		}
		e.InsertRune(c, r)
		e.WriteRune(c)
		e.Next(c)
	}
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

// LeftRune will get the rune to the left of the current data position
func (e *Editor) LeftRune() rune {
	y := e.DataY()
	x, err := e.DataX()
	if err != nil {
		// This is after the line contents, return the last rune
		runes, ok := e.lines[int(y)]
		if !ok || len(runes) == 0 {
			return rune(0)
		}
		// Return the last rune
		return runes[len(runes)-1]
	}
	if x <= 0 {
		// Nothing to the left of this
		return rune(0)
	}
	// Return the rune to the left
	return e.Get(x-1, e.DataY())
}

// CurrentLine will get the current data line, as a string
func (e *Editor) CurrentLine() string {
	return e.Line(e.DataY())
}

// Home will move the cursor the the start of the line (x = 0)
// And also scroll all the way to the left.
func (e *Editor) Home() {
	e.pos.sx = 0
	e.pos.offsetX = 0
	e.redraw = true
}

// End will move the cursor to the position right after the end of the current line contents,
// and also trim away whitespace from the right side.
func (e *Editor) End(c *vt100.Canvas) {
	y := e.DataY()
	e.TrimRight(y)
	x := e.LastTextPosition(y) + 1
	e.pos.SetX(c, x)
	e.redraw = true
}

// EndNoTrim will move the cursor to the position right after the end of the current line contents
func (e *Editor) EndNoTrim(c *vt100.Canvas) {
	x := e.LastTextPosition(e.DataY()) + 1
	e.pos.SetX(c, x)
	e.redraw = true
}

// AtEndOfLine returns true if the cursor is at exactly the last character of the line, not the one after
func (e *Editor) AtEndOfLine() bool {
	return e.pos.sx+e.pos.offsetX == e.LastTextPosition(e.DataY())
}

// DownEnd will move down and then choose a "smart" X position
func (e *Editor) DownEnd(c *vt100.Canvas) error {
	tmpx := e.pos.sx
	err := e.pos.Down(c)
	if err != nil {
		return err
	}
	line := e.CurrentLine()
	if len(strings.TrimSpace(line)) == 1 {
		e.TrimRight(e.DataY())
		e.End(c)
	} else if e.AfterLineScreenContentsPlusOne() && tmpx > 1 {
		e.End(c)
		if e.pos.sx != tmpx && e.pos.sx > e.pos.savedX {
			e.pos.savedX = tmpx
		}
	} else {
		e.pos.sx = e.pos.savedX

		if e.pos.sx < 0 {
			e.pos.sx = 0
		}
		if e.AfterLineScreenContentsPlusOne() {
			e.End(c)
		}

		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if e.Rune() == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
		}

		// Expand the line, then check if e.pos.sx falls on a tab character ("\t" is expanded to several tabs ie. "\t\t\t\t")
		expandedRunes := []rune(strings.Replace(line, "\t", strings.Repeat("\t", e.tabsSpaces.perTab), -1))
		if e.pos.sx < len(expandedRunes) && expandedRunes[e.pos.sx] == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
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
	if e.AfterLineScreenContentsPlusOne() && tmpx > 1 {
		e.End(c)
		if e.pos.sx != tmpx && e.pos.sx > e.pos.savedX {
			e.pos.savedX = tmpx
		}
	} else {
		e.pos.sx = e.pos.savedX

		if e.pos.sx < 0 {
			e.pos.sx = 0
		}
		if e.AfterLineScreenContentsPlusOne() {
			e.End(c)
		}

		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if e.Rune() == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
		}

		// Expand the line, then check if e.pos.sx falls on a tab character ("\t" is expanded to several tabs ie. "\t\t\t\t")
		expandedRunes := []rune(strings.Replace(e.CurrentLine(), "\t", strings.Repeat("\t", e.tabsSpaces.perTab), -1))
		if e.pos.sx < len(expandedRunes) && expandedRunes[e.pos.sx] == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
		}
	}
	return nil
}

// Next will move the cursor to the next position in the contents
func (e *Editor) Next(c *vt100.Canvas) error {
	// Ignore it if the position is out of bounds
	atTab := e.Rune() == '\t'
	if atTab {
		e.pos.sx += e.tabsSpaces.perTab
	} else {
		e.pos.sx++
	}
	// Did we move too far on this line?
	if e.AfterLineScreenContentsPlusOne() {
		// Undo the move
		if atTab {
			e.pos.sx -= e.tabsSpaces.perTab
		} else {
			e.pos.sx--
		}
		// Move down
		err := e.pos.Down(c)
		if err != nil {
			return err
		}
		// Move to the start of the line
		e.pos.sx = 0
	}
	return nil
}

// LeftRune2 returns the rune to the left of the current position, or an error
func (e *Editor) LeftRune2() (rune, error) {
	x, err := e.DataX()
	if err != nil {
		return rune(0), err
	}
	x--
	if x <= 0 {
		return rune(0), errors.New("no runes to the left")
	}
	return e.Get(x, e.DataY()), nil
}

// TabToTheLeft returns true if there is a '\t' to the left of the current position
func (e *Editor) TabToTheLeft() bool {
	r, err := e.LeftRune2()
	if err != nil {
		return false
	}
	return r == '\t'
}

// Prev will move the cursor to the previous position in the contents
func (e *Editor) Prev(c *vt100.Canvas) error {

	atTab := e.TabToTheLeft() || (e.pos.sx <= e.tabsSpaces.perTab && e.Get(0, e.DataY()) == '\t')
	if e.pos.sx == 0 && e.pos.offsetX > 0 {
		// at left edge, but can scroll to the left
		e.pos.offsetX--
		e.redraw = true
	} else {
		// If at a tab character, move a few more positions
		if atTab {
			e.pos.sx -= e.tabsSpaces.perTab
		} else {
			e.pos.sx--
		}
	}
	if e.pos.sx < 0 { // Did we move too far and there is no X offset?
		// Undo the move
		if atTab {
			e.pos.sx += e.tabsSpaces.perTab
		} else {
			e.pos.sx++
		}
		// Move up, and to the end of the line above, if in EOL mode
		err := e.pos.Up()
		if err != nil {
			return err
		}
		e.End(c)
	}
	return nil
}

// Right will move the cursor to the right, if possible.
// It will not move the cursor up or down.
func (p *Position) Right(c *vt100.Canvas) {
	w := 80 // default width
	if c != nil {
		w = int(c.Width())
	}
	if p.sx < (w - 1) {
		p.sx++
	} else {
		p.sx = 0
		p.offsetX += (w - 1)
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
	if regardless || (!e.AfterLineScreenContentsPlusOne() && e.pos.sx > 1) {
		e.pos.savedX = e.pos.sx
	}
}

// ScrollDown will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollDown(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed

	// Last y position in the canvas
	canvasLastY := int(c.H() - 1)

	// Retrieve the current editor scroll offset offset
	mut.RLock()
	offset := e.pos.offsetY
	mut.RUnlock()

	// Number of lines in the document
	l := e.Len()

	if offset >= l-canvasLastY {
		c.Draw()
		// Don't redraw
		return false
	}

	if status != nil {
		status.Clear(c)
	}

	if (offset + canScroll) >= (l - canvasLastY) {
		// Almost at the bottom, we can scroll the remaining lines
		canScroll = (l - canvasLastY) - offset
	}

	// Move the scroll offset
	mut.Lock()
	e.pos.offsetX = 0
	e.pos.offsetY += canScroll
	mut.Unlock()

	// Prepare to redraw
	return true
}

// ScrollUp will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollUp(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed

	// Retrieve the current editor scroll offset offset
	mut.RLock()
	offset := e.pos.offsetY
	mut.RUnlock()

	if offset == 0 {
		// Can't scroll further up
		// Status message
		//status.SetMessage("Start of text")
		//status.Show(c, p)
		//c.Draw()
		// Redraw
		return true
	}
	status.Clear(c)
	if offset-canScroll < 0 {
		// Almost at the top, we can scroll the remaining lines
		canScroll = offset
	}
	// Move the scroll offset
	mut.Lock()
	e.pos.offsetX = 0
	e.pos.offsetY -= canScroll
	mut.Unlock()
	// Prepare to redraw
	return true
}

// AtFirstLineOfDocument is true if we're at the first line of the document
func (e *Editor) AtFirstLineOfDocument() bool {
	return e.DataY() == LineIndex(0)
}

// AtLastLineOfDocument is true if we're at the last line of the document
func (e *Editor) AtLastLineOfDocument() bool {
	return e.DataY() == LineIndex(e.Len()-1)
}

// AfterLastLineOfDocument is true if we're after the last line of the document
func (e *Editor) AfterLastLineOfDocument() bool {
	return e.DataY() > LineIndex(e.Len()-1)
}

// AtOrAfterLastLineOfDocument is true if we're at or after the last line of the document
func (e *Editor) AtOrAfterLastLineOfDocument() bool {
	return e.DataY() >= LineIndex(e.Len()-1)
}

// AtOrAfterEndOfDocument is true if the cursor is at or after the end of the last line of the document
func (e *Editor) AtOrAfterEndOfDocument() bool {
	return (e.AtLastLineOfDocument() && e.AtOrAfterEndOfLine()) || e.AfterLastLineOfDocument()
}

// AfterEndOfDocument is true if the cursor is after the end of the last line of the document
func (e *Editor) AfterEndOfDocument() bool {
	return e.AfterLastLineOfDocument() // && e.AtOrAfterEndOfLine()
}

// AtEndOfDocument is true if the cursor is at the end of the last line of the document
func (e *Editor) AtEndOfDocument() bool {
	return e.AtLastLineOfDocument() && e.AtEndOfLine()
}

// AtStartOfDocument is true if we're at the first line of the document
func (e *Editor) AtStartOfDocument() bool {
	return e.pos.sy == 0 && e.pos.offsetY == 0
}

// AtStartOfScreenLine is true if the cursor is a the start of the screen line.
// The line may be scrolled all the way to the end, and the cursor moved to the left of the screen, for instance.
func (e *Editor) AtStartOfScreenLine() bool {
	return e.pos.AtStartOfScreenLine()
}

// AtStartOfTheLine is true if the cursor is a the start of the screen line, and the line is not scrolled.
func (e *Editor) AtStartOfTheLine() bool {
	return e.pos.AtStartOfTheLine()
}

// AtLeftEdgeOfDocument is true if we're at the first column at the document. Same as AtStarOfTheLine.
func (e *Editor) AtLeftEdgeOfDocument() bool {
	return e.pos.sx == 0 && e.pos.offsetX == 0
}

// AtOrAfterEndOfLine returns true if the cursor is at or after the contents of this line
func (e *Editor) AtOrAfterEndOfLine() bool {
	if e.EmptyLine() {
		return true
	}
	x, err := e.DataX()
	if err != nil {
		// After end of data
		return true
	}
	return x >= e.LastDataPosition(e.DataY())
}

// AfterEndOfLine returns true if the cursor is after the contents of this line
func (e *Editor) AfterEndOfLine() bool {
	if e.EmptyLine() {
		return true
	}
	x, err := e.DataX()
	if err != nil {
		// After end of data
		return true
	}
	return x > e.LastDataPosition(e.DataY())
}

// AfterLineScreenContents will check if the cursor is after the current line contents
func (e *Editor) AfterLineScreenContents() bool {
	return e.pos.sx > e.LastScreenPosition(e.DataY())
}

// AfterScreenWidth checks if the current cursor position has moved after the terminal/canvas width
func (e *Editor) AfterScreenWidth(c *vt100.Canvas) bool {
	w := 80 // default width
	if c != nil {
		w = int(c.W())
	}
	return e.pos.sx >= w
}

// AfterLineScreenContentsPlusOne will check if the cursor is after the current line contents, with a margin of 1
func (e *Editor) AfterLineScreenContentsPlusOne() bool {
	return e.pos.sx > (e.LastScreenPosition(e.DataY()) + 1)
}

// WriteRune writes the current rune to the given canvas
func (e *Editor) WriteRune(c *vt100.Canvas) {
	if c != nil {
		c.WriteRune(uint(e.pos.sx+e.pos.offsetX), uint(e.pos.sy), e.fg, e.bg, e.Rune())
	}
}

// WriteTab writes spaces when there is a tab character, to the canvas
func (e *Editor) WriteTab(c *vt100.Canvas) {
	spacesPerTab := e.tabsSpaces.perTab
	for x := e.pos.sx; x < e.pos.sx+spacesPerTab; x++ {
		c.WriteRune(uint(x+e.pos.offsetX), uint(e.pos.sy), e.fg, e.bg, ' ')
	}
}

// EmptyRightTrimmedLine checks if the current line is empty (and whitespace doesn't count)
func (e *Editor) EmptyRightTrimmedLine() bool {
	return len(strings.TrimRightFunc(e.CurrentLine(), unicode.IsSpace)) == 0
}

// EmptyRightTrimmedLineBelow checks if the next line is empty (and whitespace doesn't count)
func (e *Editor) EmptyRightTrimmedLineBelow() bool {
	return len(strings.TrimRightFunc(e.Line(e.DataY()+1), unicode.IsSpace)) == 0
}

// EmptyLine returns true if the current line is completely empty, no whitespace or anything
func (e *Editor) EmptyLine() bool {
	return len(e.CurrentLine()) == 0
}

// EmptyTrimmedLine returns true if the current line (trimmed) is completely empty
func (e *Editor) EmptyTrimmedLine() bool {
	return len(e.TrimmedLine()) == 0
}

// AtStartOfTextScreenLine returns true if the position is at the start of the text for this screen line
func (e *Editor) AtStartOfTextScreenLine() bool {
	return uint(e.pos.sx) == e.FirstScreenPosition(e.DataY())
}

// BeforeStartOfTextScreenLine returns true if the position is before the start of the text for this screen line
func (e *Editor) BeforeStartOfTextScreenLine() bool {
	return uint(e.pos.sx) < e.FirstScreenPosition(e.DataY())
}

// AtOrBeforeStartOfTextScreenLine returns true if the position is before or at the start of the text for this screen line
func (e *Editor) AtOrBeforeStartOfTextScreenLine() bool {
	return uint(e.pos.sx) <= e.FirstScreenPosition(e.DataY())
}

// GoTo will go to a given line index, counting from 0
// Returns true if the editor should be redrawn
// status is used for clearing status bar messages and can be nil
func (e *Editor) GoTo(dataY LineIndex, c *vt100.Canvas, status *StatusBar) bool {
	if dataY == e.DataY() {
		// Already at the correct line, but still trigger a redraw
		return true
	}
	reachedEnd := false
	// Out of bounds checking for y
	if dataY < 0 {
		dataY = 0
	} else if dataY >= LineIndex(e.Len()) {
		dataY = LineIndex(e.Len() - 1)
		reachedEnd = true
	}

	h := 25
	if c != nil {
		// Get the current terminal height
		h = int(c.Height())
	}

	// Is the place we want to go within the current scroll window?
	topY := LineIndex(e.pos.offsetY)
	botY := LineIndex(e.pos.offsetY + h)

	if dataY >= topY && dataY < botY {
		// No scrolling is needed, just move the screen y position
		e.pos.sy = int(dataY) - e.pos.offsetY
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if int(dataY) < h {
		// No scrolling is needed, just move the screen y position
		e.pos.offsetY = 0
		e.pos.sy = int(dataY)
		if e.pos.sy < 0 {
			e.pos.sy = 0
		}
	} else if reachedEnd {
		// To the end of the text
		e.pos.offsetY = e.Len() - h
		e.pos.sy = h - 1
	} else {
		prevY := e.pos.sy
		// Scrolling is needed
		e.pos.sy = 0
		e.pos.offsetY = int(dataY)
		lessJumpY := prevY
		lessJumpOffset := int(dataY) - prevY
		if (lessJumpY + lessJumpOffset) < e.Len() {
			e.pos.sy = lessJumpY
			e.pos.offsetY = lessJumpOffset
		}
	}

	// The Y scrolling is done, move the X position according to the contents of the line
	e.pos.SetX(c, int(e.FirstScreenPosition(e.DataY())))

	// Clear all status messages
	if status != nil {
		status.ClearAll(c)
	}

	// Trigger cursor redraw
	e.redrawCursor = true

	// Should also redraw the text
	return true
}

// GoToLineNumber will go to a given line number, but counting from 1, not from 0!
func (e *Editor) GoToLineNumber(lineNumber LineNumber, c *vt100.Canvas, status *StatusBar, center bool) bool {
	if lineNumber < 1 {
		lineNumber = 1
	}
	redraw := e.GoTo(lineNumber.LineIndex(), c, status)
	if redraw && center {
		e.Center(c)
	}
	return redraw
}

// GoToLineNumberAndCol will go to a given line number and column number, but counting from 1, not from 0!
func (e *Editor) GoToLineNumberAndCol(lineNumber LineNumber, colNumber ColNumber, c *vt100.Canvas, status *StatusBar, center bool) bool {
	if colNumber < 1 {
		colNumber = 1
	}
	if lineNumber < 1 {
		lineNumber = 1
	}
	xIndex := colNumber.ColIndex()
	yIndex := lineNumber.LineIndex()

	// Go to the correct line
	redraw := e.GoTo(yIndex, c, status)

	// Go to the correct column as well
	tabs := strings.Count(e.Line(yIndex), "\t")
	newScreenX := int(xIndex) + (tabs * (e.tabsSpaces.perTab - 1))
	if e.pos.sx != newScreenX {
		redraw = true
	}
	e.pos.sx = newScreenX

	if redraw && center {
		e.Center(c)
	}
	return redraw
}

// Up tried to move the cursor up, and also scroll
func (e *Editor) Up(c *vt100.Canvas, status *StatusBar) {
	e.GoTo(e.DataY()-1, c, status)
}

// Down tries to move the cursor down, and also scroll
// status is used for clearing status bar messages and can be nil
func (e *Editor) Down(c *vt100.Canvas, status *StatusBar) {
	e.GoTo(e.DataY()+1, c, status)
}

// LeadingWhitespace returns the leading whitespace for this line
func (e *Editor) LeadingWhitespace() string {
	return e.CurrentLine()[:e.FirstDataPosition(e.DataY())]
}

// LeadingWhitespaceAt returns the leading whitespace for a given line index
func (e *Editor) LeadingWhitespaceAt(y LineIndex) string {
	return e.Line(y)[:e.FirstDataPosition(y)]
}

// LineNumber will return the current line number (data y index + 1)
func (e *Editor) LineNumber() LineNumber {
	return LineNumber(e.DataY() + 1)
}

// LineIndex will return the current line index (data y index)
func (e *Editor) LineIndex() LineIndex {
	return e.DataY()
}

// ColNumber will return the current column number (data x index + 1)
func (e *Editor) ColNumber() ColNumber {
	x, _ := e.DataX()
	return ColNumber(x + 1)
}

// ColIndex will return the current column index (data x index)
func (e *Editor) ColIndex() ColIndex {
	x, _ := e.DataX()
	return ColIndex(x)
}

// StatusMessage returns a status message, intended for being displayed at the bottom
func (e *Editor) StatusMessage() string {
	return fmt.Sprintf("line %d col %d rune %U words %d [%s]", e.LineNumber(), e.ColNumber(), e.Rune(), e.WordCount(), e.mode)
}

// DrawLines will draw a screen full of lines on the given canvas
func (e *Editor) DrawLines(c *vt100.Canvas, respectOffset, redraw bool) error {
	var err error
	h := int(c.Height())
	if respectOffset {
		offsetY := e.pos.OffsetY()
		err = e.WriteLines(c, LineIndex(offsetY), LineIndex(h+offsetY), 0, 0)
	} else {
		err = e.WriteLines(c, LineIndex(0), LineIndex(h), 0, 0)
	}
	if redraw {
		c.Redraw()
	} else {
		c.Draw()
	}
	return err
}

// FullResetRedraw will completely reset and redraw everything, including creating a brand new Canvas struct
func (e *Editor) FullResetRedraw(c *vt100.Canvas, status *StatusBar, drawLines bool) {
	if status != nil {
		status.ClearAll(c)
		e.SetSearchTerm(c, status, "")
	}
	savePos := e.pos

	vt100.Close()
	vt100.Reset()
	vt100.Clear()
	vt100.Init()

	newC := vt100.NewCanvas()
	newC.ShowCursor()
	w := int(newC.Width())
	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}
	e.pos = savePos

	if drawLines {
		e.DrawLines(c, true, false)
	}

	// Assign the new canvas to the current canvas
	*c = *newC

	// TODO: Find out why the following lines are needed to properly handle the SIGWINCH resize signal

	newC = vt100.NewCanvas()
	newC.ShowCursor()
	w = int(newC.Width())
	if w < e.wrapWidth {
		e.wrapWidth = w
	} else if e.wrapWidth < 80 && w >= 80 {
		e.wrapWidth = w
	}

	if drawLines {
		e.DrawLines(c, true, false)
	}

	e.redraw = true
	e.redrawCursor = true
}

// GoToPosition can go to the given position struct and use it as the new position
func (e *Editor) GoToPosition(c *vt100.Canvas, status *StatusBar, pos Position) {
	e.pos = pos
	e.redraw = e.GoTo(e.DataY(), c, status)
	e.redrawCursor = true
}

// GoToStartOfTextLine will go to the start of the non-whitespace text, for this line
func (e *Editor) GoToStartOfTextLine(c *vt100.Canvas) {
	e.pos.SetX(c, int(e.FirstScreenPosition(e.DataY())))
	e.redraw = true
}

// GoToNextParagraph will jump to the next line that has a blank line above it, if possible
// Returns true if the editor should be redrawn
func (e *Editor) GoToNextParagraph(c *vt100.Canvas, status *StatusBar) bool {
	var lastFoundBlankLine LineIndex = -1
	l := e.Len()
	for i := e.DataY() + 1; i < LineIndex(l); i++ {
		// Check if this is a blank line
		if len(strings.TrimSpace(e.Line(i))) == 0 {
			lastFoundBlankLine = i
		} else {
			// This is a non-blank line, check if the line above is blank (or before the first line)
			if lastFoundBlankLine == (i - 1) {
				// Yes, this is the line we wish to jump to
				return e.GoTo(i, c, status)
			}
		}
	}
	return false
}

// GoToPrevParagraph will jump to the previous line that has a blank line below it, if possible
// Returns true if the editor should be redrawn
func (e *Editor) GoToPrevParagraph(c *vt100.Canvas, status *StatusBar) bool {
	var lastFoundBlankLine LineIndex = LineIndex(e.Len())
	for i := e.DataY() - 1; i >= 0; i-- {
		// Check if this is a blank line
		if len(strings.TrimSpace(e.Line(i))) == 0 {
			lastFoundBlankLine = i
		} else {
			// This is a non-blank line, check if the line below is blank (or after the last line)
			if lastFoundBlankLine == (i + 1) {
				// Yes, this is the line we wish to jump to
				return e.GoTo(i, c, status)
			}
		}
	}
	return false
}

// Center will scroll the contents so that the line with the cursor ends up in the center of the screen
func (e *Editor) Center(c *vt100.Canvas) {
	// Find the terminal height
	h := 25
	if c != nil {
		h = int(c.Height())
	}

	// General information about how the positions and offsets relate:
	//
	// offset + screen y = data y
	//
	// offset = e.pos.offset
	// screen y = e.pos.sy
	// data y = e.DataY()
	//
	// offset = data y - screen y

	// Plan:
	// 1. offset = data y - (h / 2)
	// 2. screen y = data y - offset

	// Find the center line
	centerY := h / 2
	y := int(e.DataY())
	if y < centerY {
		// Not enough room to adjust
		return
	}

	// Find the new offset and y position
	newOffset := y - centerY
	newScreenY := y - newOffset

	// Assign the new values to the editor
	e.pos.offsetY = newOffset
	e.pos.sy = newScreenY
}

// CommentOn will insert a comment marker (like # or //) in front of a line
func (e *Editor) CommentOn(commentMarker string) {
	e.SetCurrentLine(commentMarker + " " + e.CurrentLine())
}

// CommentOff will remove "//" or "// " from the front of the line if "//" is given
func (e *Editor) CommentOff(commentMarker string) {
	var (
		changed      bool
		newContents  string
		contents     = e.CurrentLine()
		trimContents = strings.TrimSpace(contents)
	)
	commentMarkerPlusSpace := commentMarker + " "
	if strings.HasPrefix(trimContents, commentMarkerPlusSpace) {
		// toggle off comment
		newContents = strings.Replace(contents, commentMarkerPlusSpace, "", 1)
		changed = true
	} else if strings.HasPrefix(trimContents, commentMarker) {
		// toggle off comment
		newContents = strings.Replace(contents, commentMarker, "", 1)
		changed = true
	}
	if changed {
		e.SetCurrentLine(newContents)
		// If the line was shortened and the cursor ended up after the line, move it
		if e.AfterEndOfLine() {
			e.End(nil)
		}
	}
}

// CurrentLineCommented checks if the current trimmed line starts with "//", if "//" is given
func (e *Editor) CurrentLineCommented(commentMarker string) bool {
	return strings.HasPrefix(e.TrimmedLine(), commentMarker)
}

// ForEachLineInBlock will move the cursor and run the given function for
// each line in the current block of text (until newline or end of document)
// Also takes a string that will be passed on to the function.
func (e *Editor) ForEachLineInBlock(c *vt100.Canvas, f func(string), commentMarker string) {
	downCounter := 0
	for !e.EmptyRightTrimmedLine() && !e.AtOrAfterEndOfDocument() {
		f(commentMarker)
		e.Down(c, nil)
		downCounter++
	}
	// Go up again
	for i := downCounter; i > 0; i-- {
		e.Up(c, nil)
	}
}

// Block will return the text from the given line until
// either a newline or the end of the document.
func (e *Editor) Block(n LineIndex) string {
	var (
		bb, lb strings.Builder // block string builder and line string builder
		line   []rune
		ok     bool
		s      string
	)
	for {
		line, ok = e.lines[int(n)]
		n++
		if !ok || len(line) == 0 {
			// End of document, empty line or invalid line: end of block
			return bb.String()
		}
		lb.Reset()
		for _, r := range line {
			lb.WriteRune(r)
		}
		s = lb.String()
		if len(strings.TrimSpace(s)) == 0 {
			// Empty trimmed line, end of block
			return bb.String()
		}
		// Save this line to bb
		bb.WriteString(s)
		// And add a newline
		bb.Write([]byte{'\n'})
	}
}

// ToggleCommentBlock will toggle comments until a blank line or the end of the document is reached
// The amount of existing commented lines is considered before deciding to comment the block in or out
func (e *Editor) ToggleCommentBlock(c *vt100.Canvas) {
	// If most of the lines in the block are comments, comment it out
	// If most of the lines in the block are not comments, comment it in

	var (
		downCounter    = 0
		commentCounter = 0
		commentMarker  = e.SingleLineCommentMarker()
	)

	// Count the commented lines in this block while going down
	for !e.EmptyRightTrimmedLine() && !e.AtOrAfterEndOfDocument() {
		if e.CurrentLineCommented(commentMarker) {
			commentCounter++
		}
		e.Down(c, nil)
		downCounter++
	}
	// Go up again
	for i := downCounter; i > 0; i-- {
		e.Up(c, nil)
	}

	// Check if most lines are commented out
	mostLinesAreComments := commentCounter >= (downCounter / 2)

	if mostLinesAreComments {
		e.ForEachLineInBlock(c, e.CommentOff, commentMarker)
	} else {
		e.ForEachLineInBlock(c, e.CommentOn, commentMarker)
	}
}

// NewLine inserts a new line below and moves down one step
func (e *Editor) NewLine(c *vt100.Canvas, status *StatusBar) {
	e.InsertLineBelow()
	e.Down(c, status)
}

// ChopLine takes a string where the tabs have been expanded
// and scrolls it + chops it up for display in the current viewport.
// e.pos.offsetX and the given viewportWidth are respected.
func (e *Editor) ChopLine(line string, viewportWidth int) string {
	var screenLine string
	// Shorten the screen line to account for the X offset
	if len([]rune(line)) > e.pos.offsetX {
		screenLine = line[e.pos.offsetX:]
	}
	// Shorten the screen line to account for the terminal width
	if len(string(screenLine)) >= viewportWidth {
		screenLine = screenLine[:viewportWidth]
	}
	return screenLine
}

// HorizontalScrollIfNeeded will scroll along the X axis, if needed
func (e *Editor) HorizontalScrollIfNeeded(c *vt100.Canvas) {
	x := e.pos.sx
	w := 80
	if c != nil {
		w = int(c.W())
	}
	if x < w {
		e.pos.offsetX = 0
	} else {
		e.pos.offsetX = (x - w) + 1
		e.pos.sx -= e.pos.offsetX
	}
	e.redraw = true
	e.redrawCursor = true
}

// VerticalScrollIfNeeded will scroll along the X axis, if needed
func (e *Editor) VerticalScrollIfNeeded(c *vt100.Canvas, status *StatusBar) {
	y := e.pos.sy
	h := 25
	if c != nil {
		h = int(c.H())
	}
	if y < h {
		e.pos.offsetY = 0
	} else {
		e.pos.offsetY = (y - h) + 1
		e.pos.sy -= e.pos.offsetY
	}
	e.redraw = true
	e.redrawCursor = true
}

// InsertFile inserts the contents of a file at the current location
func (e *Editor) InsertFile(c *vt100.Canvas, filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	s := opinionatedStringReplacer.Replace(strings.TrimRightFunc(string(data), unicode.IsSpace))
	e.InsertStringAndMove(c, s)
	return nil
}

// AbsFilename returns the absolute filename for this editor,
// cleaned with filepath.Clean.
func (e *Editor) AbsFilename() (string, error) {
	absFilename, err := filepath.Abs(e.filename)
	if err != nil {
		return "", err
	}
	return filepath.Clean(absFilename), nil
}

// Switch replaces the current editor with a new Editor that opens the given file.
// The undo stack is also swapped.
// Only works for switching to one file, and then back again.
func (e *Editor) Switch(c *vt100.Canvas, tty *vt100.TTY, status *StatusBar, lk *LockKeeper, filenameToOpen string, forceOpen bool) error {
	absFilename, err := e.AbsFilename()
	if err != nil {
		return err
	}
	// Unlock and save the lock file
	lk.Unlock(absFilename)
	lk.Save()
	// Now open the header filename instead of the current file. Save the current file first.
	e.Save(c, tty)
	// Save the current location in the location history and write it to file
	e.SaveLocation(absFilename, e.locationHistory)

	var (
		e2            *Editor
		statusMessage string
	)

	if switchBuffer.Len() == 1 {
		// Load the Editor from the switchBuffer if switchBuffer has length 1, then use that editor.
		switchBuffer.Restore(e)
		undo, switchUndoBackup = switchUndoBackup, undo
	} else {
		e2, statusMessage, err = NewEditor(tty, c, filenameToOpen, LineNumber(0), ColNumber(0), defaultTheme)
		if err == nil { // no issue
			// Save the current Editor to the switchBuffer if switchBuffer if empty, then use the new editor.
			switchBuffer.Snapshot(e)

			// Now use e2 as the current editor
			*e = *e2
			(*e).lines = (*e2).lines
			(*e).pos = (*e2).pos

		} else {
			panic(err)
		}
		undo, switchUndoBackup = switchUndoBackup, undo
	}

	e.redraw = true
	e.redrawCursor = true

	if statusMessage != "" {
		status.Clear(c)
		status.SetMessage(statusMessage)
		status.Show(c, e)
	}

	return err
}

// TrimmedLine returns the current line, trimmed in both ends
func (e *Editor) TrimmedLine() string {
	return strings.TrimSpace(e.CurrentLine())
}

// LineContentsFromCursorPosition returns the rest of the line,
// from the current cursor position, trimmed.
func (e *Editor) LineContentsFromCursorPosition() string {
	x, err := e.DataX()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(e.CurrentLine()[x:])
}

// LettersBeforeCursor returns the current word up until the cursor (for autocompletion)
func (e *Editor) LettersBeforeCursor() string {
	y := int(e.DataY())
	runes, ok := e.lines[y]
	if !ok {
		// This should never happen
		return ""
	}
	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	var word []rune

	// Loop from the position before the current one and then leftwards on the current line
	for i := x - 1; i >= 0; i-- {
		r := runes[i]
		if !unicode.IsLetter(r) {
			break
		}
		// Gather the letters in reverse
		word = append([]rune{r}, word...)
	}
	return string(word)
}

// LettersOrDotBeforeCursor returns the current word up until the cursor (for autocompletion).
// Will also include ".".
func (e *Editor) LettersOrDotBeforeCursor() string {
	y := int(e.DataY())
	runes, ok := e.lines[y]
	if !ok {
		// This should never happen
		return ""
	}
	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	var word []rune

	// Loop from the position before the current one and then leftwards on the current line
	for i := x - 1; i >= 0; i-- {
		r := runes[i]
		if !(r == '.' || unicode.IsLetter(r)) {
			break
		}
		// Gather the letters in reverse
		word = append([]rune{r}, word...)
	}
	return string(word)
}
