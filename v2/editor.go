package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/cyrus-and/gdb"
	"github.com/xyproto/clip"
	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/megafile"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

var clearOnQuit atomic.Bool // clear the terminal when quitting the editor, or not

// Editor represents the contents and editor settings, but not settings related to the viewport or scrolling
type Editor struct {
	detectedTabs       *bool           // were tab or space indentations detected when loading the data?
	breakpoint         *Position       // for the breakpoint/jump functionality in debug mode
	gdb                *gdb.Gdb        // connection to gdb, if debugMode is enabled
	sameFilePortal     *Portal         // a portal that points to the same file
	lines              map[int][]rune  // the contents of the current document
	macro              *Macro          // the contents of the current macro (will be cleared when esc is pressed)
	filename           string          // the current filename
	searchTerm         string          // the current search term, used when searching
	stickySearchTerm   string          // used when going to the next match with ctrl-n, unless esc has been pressed
	pos                Position        // the current cursor and scroll position
	Theme                              // editor theme, embedded struct
	indentation        mode.TabsSpaces // spaces or tabs, and how many spaces per tab character
	wrapWidth          int             // set to ie. 80 or 100 to trigger word wrap when typing to that column
	mode               mode.Mode       // a filetype mode, like for git, markdown or various programming languages
	debugShowRegisters int             // show no register box, show changed registers, show all changed registers
	previousY          int             // previous cursor position
	previousX          int             // previous cursor position
	lineBeforeSearch   LineIndex       // save the current line number before jumping between search results
	playBackMacroCount int             // number of times the macro should be played back, right now
	nextAction         megafile.Action // send SIGQUIT to the parent PID when quitting
	// atomic.Bool are used for values that might be read when redrawing text asynchronously
	changed                    atomic.Bool // has the contents changed, since last save?
	redraw                     atomic.Bool // if the contents should be redrawn in the next loop
	redrawCursor               atomic.Bool // if the cursor should be moved to the location it is supposed to be
	drawProgress               atomic.Bool // used for drawing the progress character on the right side
	drawFuncName               atomic.Bool // used when drawing the function name in the top right corner
	nanoMode                   atomic.Bool // emulate GNU Nano
	waitWithRedrawing          atomic.Bool // wait with redrawing until a key is pressed
	flaskApplication           atomic.Bool // Python + Flask
	moveLinesMode              atomic.Bool // move lines up and down with ctrl-p and ctrl-n, when enabled
	rainbowParenthesis         bool        // rainbow parenthesis
	sshMode                    bool        // is o used over ssh, tmux or screen, in a way that usually requires extra redrawing?
	debugMode                  bool        // in a mode where ctrl-b toggles breakpoints, ctrl-n steps to the next line and ctrl-space runs the application
	statusMode                 bool        // display a status bar at all times at the bottom of the screen
	showColumnLimit            bool        // show the line where the wrapWidth is (at 79 by default)
	expandTags                 bool        // can be used for XML and HTML
	syntaxHighlight            bool        // syntax highlighting
	quit                       bool        // for indicating if the user wants to end the editor session
	readOnly                   bool        // is the file read-only when initializing o?
	debugHideOutput            bool        // hide the GDB stdout pane when in debug mode?
	binaryFile                 bool        // is this a binary file, or a text file?
	wrapWhenTyping             bool        // wrap text at a certain limit when typing
	addSpace                   bool        // add a space to the editor, once
	debugStepInto              bool        // when stepping to the next instruction, step into instead of over
	slowLoad                   bool        // was the initial file slow to load? (might be an indication of a slow disk or USB stick)
	building                   bool        // currently building code or exporting to a file?
	runAfterBuild              bool        // run the application after building?
	monitorAndReadOnly         bool        // monitor the file for changes and open it as read-only
	primaryClipboard           bool        // use the primary or the secondary clipboard on UNIX?
	jumpToLetterMode           bool        // jump directly to a highlighted letter
	spellCheckMode             bool        // spell check mode?
	createDirectoriesIfMissing bool        // when saving a file, should directories be created if they are missing?
	displayQuickHelp           bool        // display the quick help box?
	noDisplayQuickHelp         bool        // prevent the quick help box from being displayed?
	blockMode                  bool        // toggle if typing should affect the current line or the current block
	dirMode                    bool        // browse a directory and also interact with git
	highlightCurrentLine       bool        // highlight the current line
	highlightCurrentText       bool        // highlight the current text (not the entire line)
	fastInputMode              bool        // reduce input latency for real-time use
}

// Copy makes a copy of an Editor struct, with most fields deep copied
func (e *Editor) Copy(withLines bool) *Editor {
	var e2 Editor
	if e.detectedTabs != nil {
		detectedTabsCopy := *e.detectedTabs
		e2.detectedTabs = &detectedTabsCopy
	}
	e2.breakpoint = e.breakpoint         //.Copy()
	e2.gdb = e.gdb                       //.Copy()
	e2.sameFilePortal = e.sameFilePortal //.Copy()
	if withLines {
		e2.lines = e.CopyLines()
	} else {
		e2.lines = nil
	}
	e2.macro = e.macro //.Copy()
	e2.filename = e.filename
	e2.searchTerm = e.searchTerm
	e2.stickySearchTerm = e.stickySearchTerm
	e2.Theme = e.Theme
	e2.pos = e.pos
	e2.indentation = e.indentation
	e2.wrapWidth = e.wrapWidth
	e2.mode = e.mode
	e2.debugShowRegisters = e.debugShowRegisters
	e2.previousY = e.previousY
	e2.previousX = e.previousX
	e2.lineBeforeSearch = e.lineBeforeSearch
	e2.playBackMacroCount = e.playBackMacroCount
	e2.rainbowParenthesis = e.rainbowParenthesis
	e2.sshMode = e.sshMode
	e2.debugMode = e.debugMode
	e2.statusMode = e.statusMode
	e2.showColumnLimit = e.showColumnLimit
	e2.expandTags = e.expandTags
	e2.syntaxHighlight = e.syntaxHighlight
	e2.nextAction = e.nextAction
	e2.quit = e.quit
	e2.readOnly = e.readOnly
	e2.debugHideOutput = e.debugHideOutput
	e2.binaryFile = e.binaryFile
	e2.wrapWhenTyping = e.wrapWhenTyping
	e2.addSpace = e.addSpace
	e2.debugStepInto = e.debugStepInto
	e2.slowLoad = e.slowLoad
	e2.building = e.building
	e2.runAfterBuild = e.runAfterBuild
	e2.monitorAndReadOnly = e.monitorAndReadOnly
	e2.primaryClipboard = e.primaryClipboard
	e2.jumpToLetterMode = e.jumpToLetterMode
	e2.spellCheckMode = e.spellCheckMode
	e2.createDirectoriesIfMissing = e.createDirectoriesIfMissing
	e2.displayQuickHelp = e.displayQuickHelp
	e2.blockMode = e.blockMode
	e2.dirMode = e.dirMode
	e2.highlightCurrentLine = e.highlightCurrentLine
	e2.highlightCurrentText = e.highlightCurrentText
	e2.fastInputMode = e.fastInputMode
	e2.nanoMode.Store(e.nanoMode.Load())
	e2.changed.Store(e.changed.Load())
	e2.redraw.Store(e.redraw.Load())
	e2.redrawCursor.Store(e.redrawCursor.Load())
	e2.drawProgress.Store(e.drawProgress.Load())
	e2.drawFuncName.Store(e.drawFuncName.Load())
	e2.waitWithRedrawing.Store(e.waitWithRedrawing.Load())
	e2.flaskApplication.Store(e.flaskApplication.Load())
	e2.moveLinesMode.Store(e.moveLinesMode.Load())
	return &e2
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
		e.changed.Store(true)
		return
	}
	// If the line is too short, fill it up with spaces
	if l <= x {
		n := (x + 1) - l
		e.lines[y] = append(e.lines[y], []rune(strings.Repeat(" ", n))...)
	}

	// Set the rune
	e.lines[y][x] = r
	e.changed.Store(true)
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
	return e.changed.Load()
}

// Line returns the contents of line number N, counting from 0
func (e *Editor) Line(n LineIndex) string {
	if n < 0 {
		return ""
	}
	if line, ok := e.lines[int(n)]; ok {
		return string(line)
	}
	return ""
}

// ScreenLine returns the screen contents of line number N, counting from 0.
// The tabs are expanded.
func (e *Editor) ScreenLine(n int) string {
	if line, ok := e.lines[n]; ok {
		var sb strings.Builder
		skipX := e.pos.offsetX
		for _, r := range line {
			if skipX > 0 {
				skipX--
				continue
			}
			sb.WriteRune(r)
		}
		tabSpace := strings.Repeat("\t", e.indentation.PerTab)
		return strings.ReplaceAll(sb.String(), "\t", tabSpace)
	}
	return ""
}

// LastScreenPosition returns the last X index for this line, for the screen (expands tabs)
// Can be negative, if the line is empty.
func (e *Editor) LastScreenPosition(n LineIndex) int {
	extraSpaceBecauseOfTabs := int(e.CountRune('\t', n) * (e.indentation.PerTab - 1))
	return (e.LastDataPosition(n) + extraSpaceBecauseOfTabs)
}

// LastTextPosition returns the last X index for this line, regardless of horizontal scrolling.
// Can be negative if the line is empty. Tabs are expanded.
func (e *Editor) LastTextPosition(n LineIndex) int {
	extraSpaceBecauseOfTabs := int(e.CountRune('\t', n) * (e.indentation.PerTab - 1))
	return (e.LastDataPosition(n) + extraSpaceBecauseOfTabs)
}

// FirstScreenPosition returns the first X index for this line, that is not '\t' or ' '.
// Does not deal with the X offset.
func (e *Editor) FirstScreenPosition(n LineIndex) uint {
	var (
		counter      uint
		spacesPerTab = uint(e.indentation.PerTab)
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
	return len(e.lines)
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

// ContentsAndReverseSearchPrefix returns the contents of the editor,
// and also the LineNumber of the given string, searching for the prefix backwards from the current position.
// Also returns true if the given string was found. Used for the "iferr" feature in keyloop.go.
func (e *Editor) ContentsAndReverseSearchPrefix(prefix string) (string, LineIndex, bool) {
	currentLineIndex := e.LineIndex()
	foundLineIndex := currentLineIndex
	foundIt := false
	var sb strings.Builder
	l := e.Len()
	for i := 0; i < l; i++ {
		lineIndex := LineIndex(i)
		line := e.Line(lineIndex)
		if lineIndex <= currentLineIndex && strings.HasPrefix(strings.TrimSpace(line), prefix) {
			// Found it, and it's above the currentLineIndex
			foundLineIndex = lineIndex
			foundIt = true
		}
		sb.WriteString(line + "\n")
	}
	return sb.String(), foundLineIndex, foundIt
}

// Clear removes all data from the editor
func (e *Editor) Clear() {
	e.lines = make(map[int][]rune)
	e.changed.Store(true)
}

// Load will try to load a file. The file is assumed to be checked to already exist.
// Returns a warning message (possibly empty) and an error type
func (e *Editor) Load(c *vt.Canvas, tty *vt.TTY, fnord FilenameOrData) (string, error) {
	var (
		message string
		err     error
	)

	// Start a spinner, in a short while
	const cursorAfterText = false
	quitChan := e.Spinner(c, tty, fmt.Sprintf("Reading %s... ", fnord.filename), fmt.Sprintf("reading %s: stopped by user", fnord.filename), 200*time.Millisecond, e.ItalicsColor, cursorAfterText)

	// Stop the spinner at the end of the function
	defer func() {
		quitChan <- true
	}()

	start := time.Now()

	// Check if the file extension is ".class" and if "jad" is installed
	if filepath.Ext(fnord.filename) == ".class" && files.WhichCached("jad") != "" && fnord.Empty() {
		if fnord.data, err = e.LoadClass(fnord.filename); err != nil {
			return "Could not run jad", err
		}
		// Load the data (and make opinionated replacements if it's a text file + set e.binaryFile if it's binary)
		e.LoadBytes(fnord.data)
	} else if fnord.stdin {
		// Load the data that has already been read from stdin
		// (and make opinionated replacements if it's a text file + set e.binaryFile if it's binary)
		e.LoadBytes(fnord.data)
	} else if fnord.Empty() {
		// Load the file (and make opinionated replacements if it's a text file + set e.binaryFile if it's binary)
		if err := e.ReadFileAndProcessLines(fnord.filename); err != nil {
			return message, err
		}
	}

	if e.binaryFile {
		e.mode = mode.Blank
	}

	// If enough time passed so that the spinner was shown by now, enter "slow disk mode" where fewer disk-related I/O operations will be performed
	e.slowLoad = time.Since(start) > 400*time.Millisecond

	// Mark the data as "not changed", since this happens when starting the editor
	e.changed.Store(false)

	return message, nil
}

// IndexByteLine represents a single line of text, as bytes and with a line index
type IndexByteLine struct {
	byteLine []byte
	index    int
}

// PrepareEmpty prepares an empty textual representation of a given filename.
// If it's an image, there will be text placeholders for pixels.
// If it's anything else, it will just be blank.
// Returns an editor mode and an error type.
func (e *Editor) PrepareEmpty() (mode.Mode, error) {
	var (
		m    mode.Mode = mode.Blank
		data []byte
		err  error
	)

	// Check if the data could be prepared
	if err != nil {
		return m, err
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
	e.changed.Store(false)

	return m, nil
}

// Save will try to save the current editor contents to file.
// It needs a canvas in case trailing spaces are stripped and the cursor needs to move to the end.
func (e *Editor) Save(c *vt.Canvas, tty *vt.TTY) error {
	return e.SaveAs(c, tty, e.filename)
}

// SaveAs will try to save the current editor contents to given file.
// It needs a canvas in case trailing spaces are stripped and the cursor needs to move to the end.
func (e *Editor) SaveAs(c *vt.Canvas, tty *vt.TTY, filename string) error {

	if e.monitorAndReadOnly {
		return errors.New("file is read-only")
	}

	var (
		bookmark = e.pos.Copy() // Save the current position
		changed  bool
		shebang  bool
		data     []byte
	)

	quitMut.Lock()
	defer quitMut.Unlock()

	if e.createDirectoriesIfMissing {
		if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
			return err
		}
	}

	if e.binaryFile {
		data = []byte(e.String())
	} else {
		// Strip trailing spaces on all lines
		l := e.Len()
		for i := 0; i < l; i++ {
			if e.TrimRight(LineIndex(i)) {
				changed = true
			}
		}

		// Trim away trailing whitespace
		s := trimRightSpace(e.String())

		// Make additional replacements, and add a final newline
		s = opinionatedStringReplacer.Replace(s) + "\n"

		// TODO: Auto-detect tabs/spaces instead of per-language assumptions
		if e.mode.Spaces() {
			// NOTE: This is a hack, that can only replace 10 levels deep.
			for level := 10; level > 0; level-- {
				fromString := "\n" + strings.Repeat("\t", level)
				toString := "\n" + strings.Repeat(" ", level*e.indentation.PerTab)
				s = strings.ReplaceAll(s, fromString, toString)
			}
		} else if e.mode == mode.Make || e.mode == mode.Just {
			var (
				level                          int
				fromString, toString, prevLine string
				lines                          = strings.Split(s, "\n")
			)
		NEXTLINE:
			for i, line := range lines {
				// NOTE: This is a hack, that can only replace 10 levels deep.
				for level = 10; level > 0; level-- {
					if strings.HasPrefix(prevLine, "  ") && len(prevLine) > 2 && prevLine[2] != ' ' {
						// make no replacements for this case, where the previous line has a 2-space indentation
						continue NEXTLINE
					}
					fromString = "\n" + strings.Repeat(" ", level*e.indentation.PerTab)
					toString = "\n" + strings.Repeat("\t", level)
					lines[i] = strings.ReplaceAll(lines[i], fromString, toString)
				}
				prevLine = line
			}
			s = strings.Join(lines, "\n")
		}

		// Should the file be saved with the executable bit enabled?
		// (Does it either start with a shebang or reside in a common bin directory like /usr/bin?)
		shebang = files.BinDirectory(filename) || strings.HasPrefix(s, "#!")

		data = []byte(s)
	}

	// Mark the data as "not changed" if it's not a binary file
	if !e.binaryFile {
		e.changed.Store(false)
	}

	// Default file mode (0644 for regular files, 0755 for executable files)
	var fileMode os.FileMode = 0o644

	// Shell scripts that contains the word "source" on the first three lines often needs to be sourced and should not be "chmod +x"-ed, nor "chmod -x" ed
	containsTheWordSource := containsInTheFirstNLines(data, 3, []byte("source "))

	// Checking the syntax highlighting makes it easy to press `ctrl-t` before saving a script,
	// to toggle the executable bit on or off. This is only for files that start with "#!".
	// Also, if the file is in one of the common bin directories, like "/usr/bin", then assume that it
	// is supposed to be executable. Also skip .install files, even though they are scripts.
	if shebang && e.syntaxHighlight && !containsTheWordSource && !strings.HasSuffix(filename, ".install") {
		// This is a script file, syntax highlighting is enabled and it does not contain the word "source "
		// (typical for shell files that should be sourced and not executed)
		fileMode = 0o755
	}

	// If it's not a binary file OR the file has changed: save the data
	if !e.binaryFile || e.changed.Load() {

		// Check if the user appears to be a quick developer
		if time.Since(editorLaunchTime) < 30*time.Second && e.mode != mode.Text && e.mode != mode.Blank {
			// Disable the quick help at start
			DisableQuickHelpScreen(nil)
		}

		// Start a spinner, in a short while
		const cursorAfterText = false
		quitChan := e.Spinner(c, tty, fmt.Sprintf("Saving %s... ", filename), fmt.Sprintf("saving %s: stopped by user", filename), 200*time.Millisecond, e.ItalicsColor, cursorAfterText)

		// Prepare gzipped data
		if strings.HasSuffix(filename, ".gz") {
			var err error
			data, err = gZipData(data)
			if err != nil {
				quitChan <- true
				return err
			}
		}

		// Save the file and return any errors
		if err := os.WriteFile(filename, data, fileMode); err != nil {
			// Stop the spinner and return
			quitChan <- true
			return err
		}

		// This file should not be considered read-only, since saving went fine
		e.readOnly = false

		// TODO: Consider the previous fileMode of the file when doing chmod +x instead of just setting 0755 or 0644

		// "chmod +x" or "chmod -x". This is needed after saving the file, in order to toggle the executable bit.
		// rust source may start with something like "#![feature(core_intrinsics)]", so avoid that.
		if !containsTheWordSource {
			if shebang && e.mode != mode.Rust && e.mode != mode.Python && e.mode != mode.Mojo && e.mode != mode.Starlark && !e.readOnly {
				// Call Chmod, but ignore errors (since this is just a bonus and not critical)
				os.Chmod(e.filename, fileMode)
				e.syntaxHighlight = true
			} else if e.mode == mode.ASCIIDoc || e.mode == mode.Just || e.mode == mode.Make || e.mode == mode.Markdown || e.mode == mode.ReStructured || e.mode == mode.SCDoc {
				fileMode = 0o644
				os.Chmod(e.filename, fileMode)
			} else if baseFilename := filepath.Base(e.filename); baseFilename == "PKGBUILD" || baseFilename == "APKGBUILD" {
				fileMode = 0o644
				os.Chmod(e.filename, fileMode)
			}
		}

		// Stop the spinner
		quitChan <- true

	}

	e.redrawCursor.Store(true)

	// Trailing spaces may be trimmed, so move to the end, if needed
	if changed {
		e.GoToPosition(c, nil, *bookmark)
		if e.AfterEndOfLine() {
			e.EndNoTrim(c)
		}
		// Do the redraw manually before showing the status message
		respectOffset := true
		redrawCanvas := false
		e.HideCursorDrawLines(c, respectOffset, redrawCanvas, false)
		e.redraw.Store(false)
	}

	// All done
	return nil
}

// TrimRight will remove whitespace from the end of the given line number
// Returns true if the line was trimmed
func (e *Editor) TrimRight(index LineIndex) bool {
	n := int(index)
	line, ok := e.lines[n]
	if !ok {
		return false
	}
	trimmedLine := []rune(trimRightSpace(string(line)))
	if len(trimmedLine) != len(line) {
		e.lines[n] = trimmedLine
		return true
	}
	return false
}

// TrimLeft will remove whitespace from the start of the given line number
// Returns true if the line was trimmed
func (e *Editor) TrimLeft(index LineIndex) bool {
	changed := false
	n := int(index)
	if line, ok := e.lines[n]; ok {
		newRunes := []rune(strings.TrimLeftFunc(string(line), unicode.IsSpace))
		// Compare lengths to detect trimming without full content comparison
		if len(newRunes) != len(line) {
			e.lines[n] = newRunes
			changed = true
		}
	}
	return changed
}

// StripSingleLineComment will strip away trailing single-line comments.
// TODO: Also strip trailing /* ... */ comments
func (e *Editor) StripSingleLineComment(line string) string {
	// Strip trailing block comments (/* ... */) if present
	if start := strings.LastIndex(line, "/*"); start != -1 {
		if idx := strings.Index(line[start:], "*/"); idx != -1 {
			return strings.TrimSpace(line[:start])
		}
	}
	// Strip single-line comments (according to language marker)
	commentMarker := e.SingleLineCommentMarker()
	if count := strings.Count(line, commentMarker); count == 1 {
		if p := strings.Index(line, commentMarker); p != -1 {
			return strings.TrimSpace(line[:p])
		}
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
	if x > len(e.lines[y]) {
		return
	}
	e.lines[y] = e.lines[y][:x]
	e.changed.Store(true)

	// Make sure no lines are nil
	e.MakeConsistent()
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
	e.changed.Store(true)

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
func (e *Editor) Delete(c *vt.Canvas, useBlockMode bool) {

	deleteThisRune := func() bool {
		y := int(e.DataY())
		lineLen := len(e.lines[y])
		if _, ok := e.lines[y]; !ok || lineLen == 0 || (lineLen == 1 && unicode.IsSpace(e.lines[y][0])) {
			// All keys in the map that are > y should be shifted -1.
			// This also overwrites e.lines[y].
			e.DeleteLine(LineIndex(y))
			e.changed.Store(true)
			return true // continue
		}
		x, err := e.DataX()
		if err != nil || x > len(e.lines[y])-1 {
			// on the last index, just use every element but x
			e.lines[y] = e.lines[y][:x]
			// check if the next line exists
			if _, ok := e.lines[y+1]; ok {
				// then add the contents of the next line, if available
				nextLine, ok := e.lines[y+1]
				if ok && len(nextLine) > 0 && !e.blockMode {
					e.lines[y] = append(e.lines[y], nextLine...)
					// then delete the next line
					e.DeleteLine(LineIndex(y + 1))
				}
			}
			e.changed.Store(true)
			return true // continue
		}
		// Delete just this character
		e.lines[y] = append(e.lines[y][:x], e.lines[y][x+1:]...)
		return true // continue
	}

	if useBlockMode {
		e.ForEachLineInBlock(c, deleteThisRune)
	} else {
		deleteThisRune()
	}

	e.changed.Store(true)
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
			return strings.TrimSpace(string(line)) == ""
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
			e.changed.Store(true)
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
			if (splitPosition - spacePosition) <= maxDistance {
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

// WrapAllLines will word wrap all lines that are longer than e.wrapWidth
func (e *Editor) WrapAllLines() bool {
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

			e.changed.Store(true)
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
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	}

	// This appears to be needed as well
	e.MakeConsistent()

	return wrapped
}

// WrapNow is a helper function for changing the word wrap width,
// while also wrapping all lines
func (e *Editor) WrapNow(wrapWith int) {
	e.wrapWidth = wrapWith
	if e.WrapAllLines() {
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	}
}

// InsertLineAbove will attempt to insert a new line above the current position
func (e *Editor) InsertLineAbove() {
	lineIndex := e.DataY()

	if e.sameFilePortal != nil {
		e.sameFilePortal.NewLineInserted(lineIndex)
	}

	y := int(lineIndex)

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
		if len(e.lines[i]) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}
	e.changed.Store(true)
}

// InsertLineBelow will attempt to insert a new line below the current position
func (e *Editor) InsertLineBelow() {
	lineIndex := e.DataY()
	if e.sameFilePortal != nil {
		e.sameFilePortal.NewLineInserted(lineIndex)
	}
	e.InsertLineBelowAt(lineIndex)
}

// InsertLineBelowAt will attempt to insert a new line below the given y position
func (e *Editor) InsertLineBelowAt(index LineIndex) {
	y := int(index)

	// Make sure no lines are nil
	e.MakeConsistent()

	// If we are the the last line, add an empty line at the end and return
	if y == (len(e.lines) - 1) {
		e.lines[int(y)+1] = make([]rune, 0)
		e.changed.Store(true)
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
		if len(e.lines[i]) == 0 {
			delete(e.lines, i)
		} else {
			break
		}
	}

	e.changed.Store(true)
}

// Insert will insert a rune at the given position, with no word wrap,
// and call MakeConsistent at the end.
func (e *Editor) Insert(c *vt.Canvas, r rune) {

	doInsert := func() bool {
		// Ignore it if the current position is out of bounds
		x, _ := e.DataX()
		y := int(e.DataY())
		// If there are no lines, initialize and set the 0th rune to the given one
		if e.lines == nil {
			e.lines = make(map[int][]rune)
			e.lines[0] = []rune{r}
			return true // continue
		}
		// If the current line is empty, initialize it with a line that is just the given rune
		_, ok := e.lines[y]
		if !ok {
			e.lines[y] = []rune{r}
			return true // continue
		}
		if len(e.lines[y]) < x {
			// Can only insert in the existing block of text
			return true // continue
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
		return true // continue
	}

	if e.blockMode {
		e.ForEachLineInBlock(c, doInsert)
	} else {
		doInsert()
	}

	e.changed.Store(true)

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
		e.changed.Store(true)
	}
}

// WordCount returns the number of spaces in the text + 1
func (e *Editor) WordCount() int {
	return len(strings.Fields(e.String()))
}

// ToggleSyntaxHighlight toggles syntax highlighting
func (e *Editor) ToggleSyntaxHighlight() {
	e.syntaxHighlight = !e.syntaxHighlight
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
	leftContents := trimRightSpace(string(runeLine[:x]))
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
	var dataY int
	e.pos.mut.RLock()
	dataY = e.pos.offsetY + e.pos.sy
	e.pos.mut.RUnlock()
	// get the current line of text
	screenCounter := 0 // counter for the characters on the screen
	// loop, while also keeping track of tab expansion
	// add a space to allow to jump to the position after the line and get a valid data position
	found := false
	dataX := 0
	runeCounter := 0
	for _, r := range e.lines[dataY] {
		e.pos.mut.RLock()
		// When we reached the correct screen position, use i as the data position
		if screenCounter == (e.pos.sx + e.pos.offsetX) {
			e.pos.mut.RUnlock()
			dataX = runeCounter
			found = true
			break
		}
		e.pos.mut.RUnlock()
		// Increase the counter, based on the current rune
		if r == '\t' {
			screenCounter += e.indentation.PerTab
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
	e.pos.mut.RLock()
	defer e.pos.mut.RUnlock()
	return LineIndex(e.pos.offsetY + e.pos.sy)
}

// SetRune will set a rune at the current data position
func (e *Editor) SetRune(r rune) {
	// Only set a rune if x is within the current line contents
	if x, err := e.DataX(); err == nil {
		e.Set(x, e.DataY(), r)
	}
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
func (e *Editor) InsertStringAndMove(c *vt.Canvas, s string) {
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

// InsertString will insert a string without newlines at the current data position.
// his will also call e.WriteRune and e.Next, as needed.
func (e *Editor) InsertString(c *vt.Canvas, s string) {
	for _, r := range s {
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

// CurrentLine will get the current data line as a string
func (e *Editor) CurrentLine() string {
	return e.Line(e.DataY())
}

// PrevLine will get the previous data line as a string, or return an empty string
func (e *Editor) PrevLine() string {
	y := e.DataY() - 1
	if y < 0 {
		return ""
	}
	return e.Line(y)
}

// NextLine will get the previous data line as a string, or return an empty string
func (e *Editor) NextLine() string {
	y := e.DataY() + 1
	if y < 0 || int(y) >= e.Len() {
		return ""
	}
	return e.Line(y)
}

// Home will move the cursor the the start of the line (x = 0)
// And also scroll all the way to the left.
func (e *Editor) Home() {
	e.pos.sx = 0
	e.pos.offsetX = 0
	e.redraw.Store(true)
}

// End will move the cursor to the position right after the end of the current line contents,
// and also trim away whitespace from the right side.
func (e *Editor) End(c *vt.Canvas) {
	y := e.DataY()
	e.TrimRight(y)
	x := e.LastTextPosition(y) + 1
	e.pos.SetX(c, x)
	e.redraw.Store(true)
}

// EndNoTrim will move the cursor to the position right after the end of the current line contents
func (e *Editor) EndNoTrim(c *vt.Canvas) {
	x := e.LastTextPosition(e.DataY()) + 1
	e.pos.SetX(c, x)
	e.redraw.Store(true)
}

// AtEndOfLine returns true if the cursor is at exactly the last character of the line, not the one after
func (e *Editor) AtEndOfLine() bool {
	return e.pos.sx+e.pos.offsetX == e.LastTextPosition(e.DataY())
}

// DownEnd will move down and then choose a "smart" X position
func (e *Editor) DownEnd(c *vt.Canvas) error {
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
		expandedRunes := []rune(strings.ReplaceAll(line, "\t", strings.Repeat("\t", e.indentation.PerTab)))
		if e.pos.sx < len(expandedRunes) && expandedRunes[e.pos.sx] == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
		}
	}
	return nil
}

// UpEnd will move up and then choose a "smart" X position
func (e *Editor) UpEnd(c *vt.Canvas) error {
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
		expandedRunes := []rune(strings.ReplaceAll(e.CurrentLine(), "\t", strings.Repeat("\t", e.indentation.PerTab)))
		if e.pos.sx < len(expandedRunes) && expandedRunes[e.pos.sx] == '\t' {
			e.pos.sx = int(e.FirstScreenPosition(e.DataY()))
		}
	}
	return nil
}

// Next will move the cursor to the next position in the contents
func (e *Editor) Next(c *vt.Canvas) error {
	if !e.blockMode {
		// Ignore it if the position is out of bounds
		atTab := e.Rune() == '\t'
		if atTab {
			e.pos.sx += e.indentation.PerTab
		} else {
			e.pos.sx++
		}
		// Did we move too far on this line?
		if e.AfterLineScreenContentsPlusOne() {
			// Undo the move
			if atTab {
				e.pos.sx -= e.indentation.PerTab
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
	// block mode
	// Ignore it if the position is out of bounds
	atTab := e.Rune() == '\t'
	if atTab {
		e.pos.sx += e.indentation.PerTab
	} else {
		e.pos.sx++
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
func (e *Editor) Prev(c *vt.Canvas) error {
	atTab := e.TabToTheLeft() || (e.pos.sx <= e.indentation.PerTab && e.Get(0, e.DataY()) == '\t')
	if e.pos.sx == 0 && e.pos.offsetX > 0 {
		// at left edge, but can scroll to the left
		e.pos.offsetX--
		e.redraw.Store(true)
	} else {
		// If at a tab character, move a few more positions
		if atTab {
			e.pos.sx -= e.indentation.PerTab
		} else {
			e.pos.sx--
		}
	}
	if e.pos.sx < 0 { // Did we move too far and there is no X offset?
		// Undo the move
		if atTab {
			e.pos.sx += e.indentation.PerTab
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

// SaveX will save the current X position, if it's within reason
func (e *Editor) SaveX(regardless bool) {
	if regardless || (!e.AfterLineScreenContentsPlusOne() && e.pos.sx > 1) {
		e.pos.savedX = e.pos.sx
	}
}

// ScrollDown will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollDown(c *vt.Canvas, status *StatusBar, scrollSpeed, canvasHeight int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed

	// Last y position in the canvas
	canvasLastY := canvasHeight - 1

	// Retrieve the current editor scroll offset offset
	mut.RLock()
	e.pos.mut.Lock()
	offset := e.pos.offsetY
	e.pos.mut.Unlock()
	mut.RUnlock()

	// Number of lines in the document
	l := e.Len()

	if offset >= l-canvasLastY {
		c.HideCursorAndDraw()
		// Don't redraw
		return false
	}
	if status != nil {
		status.Clear(c, false)
	}
	if (offset + canScroll) >= (l - canvasLastY) {
		// Almost at the bottom, we can scroll the remaining lines
		canScroll = (l - canvasLastY) - offset
	}

	// Move the scroll offset
	mut.Lock()
	e.pos.mut.Lock()
	e.pos.offsetX = 0
	e.pos.offsetY += canScroll
	e.pos.mut.Unlock()
	mut.Unlock()

	// Prepare to redraw
	return true
}

// ScrollUp will scroll down the given amount of lines given in scrollSpeed
func (e *Editor) ScrollUp(c *vt.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed

	// Retrieve the current editor scroll offset offset
	mut.RLock()
	e.pos.mut.Lock()
	offset := e.pos.offsetY
	e.pos.mut.Unlock()
	mut.RUnlock()

	if offset == 0 {
		return true
	}
	if status != nil {
		status.Clear(c, false)
	}
	if offset-canScroll < 0 {
		// Almost at the top, we can scroll the remaining lines
		canScroll = offset
	}
	// Move the scroll offset
	mut.Lock()
	e.pos.mut.Lock()
	e.pos.offsetX = 0
	e.pos.offsetY -= canScroll
	e.pos.mut.Unlock()
	mut.Unlock()
	// Prepare to redraw
	return true
}

// PgDn will try to scroll down a full page
func (e *Editor) PgDn(c *vt.Canvas, status *StatusBar) bool {
	canvasHeight := int(c.H())
	scrollSpeed := canvasHeight
	return e.ScrollDown(c, status, scrollSpeed, canvasHeight)
}

// AtTopLeftOfDocument is true if the cursor is to the far left of the first line of the document
func (e *Editor) AtTopLeftOfDocument() bool {
	x, err := e.DataX()
	return err == nil && x == 0 && e.DataY() == LineIndex(0)
}

// AtFirstLineOfDocument is true if we're at the first line of the document
func (e *Editor) AtFirstLineOfDocument() bool {
	return e.DataY() == LineIndex(0)
}

// AtSecondLineOfDocumentOrLater is true if we're at the second line of the document, or later
func (e *Editor) AtSecondLineOfDocumentOrLater() bool {
	return e.DataY() >= LineIndex(1)
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
	return e.pos.sy == 0 //&& e.pos.offsetY == 0
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
func (e *Editor) AfterScreenWidth(c *vt.Canvas) bool {
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
func (e *Editor) WriteRune(c *vt.Canvas) {
	if c == nil {
		return
	}
	if !e.blockMode {
		c.WriteRune(uint(e.pos.sx+e.pos.offsetX), uint(e.pos.sy), e.Foreground, e.Background, e.Rune())
	} else {
		e.ForEachLineInBlock(c, func() bool {
			c.WriteRune(uint(e.pos.sx+e.pos.offsetX), uint(e.pos.sy), e.Foreground, e.Background, e.Rune())
			return true // continue
		})
	}
}

// WriteTab writes spaces when there is a tab character, to the canvas
func (e *Editor) WriteTab(c *vt.Canvas) {
	if c == nil {
		return
	}
	spacesPerTab := e.indentation.PerTab
	if !e.blockMode {
		for x := e.pos.sx; x < e.pos.sx+spacesPerTab; x++ {
			c.WriteRune(uint(x+e.pos.offsetX), uint(e.pos.sy), e.Foreground, e.Background, ' ')
		}
	} else {
		e.ForEachLineInBlock(c, func() bool {
			for x := e.pos.sx; x < e.pos.sx+spacesPerTab; x++ {
				c.WriteRune(uint(x+e.pos.offsetX), uint(e.pos.sy), e.Foreground, e.Background, ' ')
			}
			return true // continue
		})
	}
}

// EmptyRightTrimmedLine checks if the current line is empty (and whitespace doesn't count)
func (e *Editor) EmptyRightTrimmedLine() bool {
	return trimRightSpace(e.CurrentLine()) == ""
}

// LineAbove returns the line above, if possible
func (e *Editor) LineAbove() string {
	y := e.DataY() - 1
	if y >= 0 {
		return e.Line(y)
	}
	return ""

}

// LineBelow returns the line below, if possible
func (e *Editor) LineBelow() string {
	y := e.DataY() + 1
	if int(y) < e.Len() {
		return e.Line(y)
	}
	return ""
}

// EmptyRightTrimmedLineBelow checks if the next line is empty (and whitespace doesn't count)
func (e *Editor) EmptyRightTrimmedLineBelow() bool {
	return trimRightSpace(e.Line(e.DataY()+1)) == ""
}

// EmptyLine returns true if the current line is completely empty, no whitespace or anything
func (e *Editor) EmptyLine() bool {
	return e.CurrentLine() == ""
}

// EmptyTrimmedLine returns true if the current line (trimmed) is completely empty
func (e *Editor) EmptyTrimmedLine() bool {
	return e.TrimmedLine() == ""
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

// Up tried to move the cursor up, and also scroll
func (e *Editor) Up(c *vt.Canvas, status *StatusBar) {
	e.GoTo(e.DataY()-1, c, status)
}

// Down tries to move the cursor down, and also scroll
// status is used for clearing status bar messages and can be nil
// returns true if the end is reached
func (e *Editor) Down(c *vt.Canvas, status *StatusBar) bool {
	_, reachedTheEnd := e.GoTo(e.DataY()+1, c, status)
	return reachedTheEnd
}

// LeadingWhitespace returns the leading whitespace for this line
func (e *Editor) LeadingWhitespace() string {
	return e.CurrentLine()[:e.FirstDataPosition(e.DataY())]
}

// WhitespaceAboveAndBelow returns the longest indentation whitespace for the line above or below
func (e *Editor) WhitespaceAboveAndBelow() string {
	prev := getLeadingWhitespace(e.LineAbove())
	next := getLeadingWhitespace(e.LineBelow())
	if len(next) > len(prev) {
		return next
	}
	return prev
}

// WhitespaceOnAboveAndBelow returns the longest indentation whitespace for the current line, the line above or the line below
func (e *Editor) WhitespaceOnAboveAndBelow() string {
	prv := getLeadingWhitespace(e.LineAbove())
	cur := getLeadingWhitespace(e.CurrentLine())
	nxt := getLeadingWhitespace(e.LineBelow())
	if len(nxt) > len(prv) && len(nxt) > len(cur) {
		return nxt
	}
	if len(prv) > len(nxt) && len(prv) > len(cur) {
		return prv
	}
	return cur
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

// GoToPosition can go to the given position struct and use it as the new position
func (e *Editor) GoToPosition(c *vt.Canvas, status *StatusBar, pos Position) {
	e.pos = pos
	redraw, _ := e.GoTo(e.DataY(), c, status)
	e.redraw.Store(redraw)
	e.redrawCursor.Store(true)
}

// GoToStartOfTextLine will go to the start of the non-whitespace text, for this line
func (e *Editor) GoToStartOfTextLine(c *vt.Canvas) {
	e.pos.SetX(c, int(e.FirstScreenPosition(e.DataY())))
	e.redraw.Store(true)
}

// GoToNextParagraph will jump to the next line that has a blank line above it, if possible
// Returns true if the editor should be redrawn, and true if the end has been reached
func (e *Editor) GoToNextParagraph(c *vt.Canvas, status *StatusBar) (bool, bool) {
	var lastFoundBlankLine LineIndex = -1
	l := e.Len()
	for i := e.DataY() + 1; i < LineIndex(l); i++ {
		// Check if this is a blank line
		if strings.TrimSpace(e.Line(i)) == "" {
			lastFoundBlankLine = i
		} else {
			// This is a non-blank line, check if the line above is blank (or before the first line)
			if lastFoundBlankLine == (i - 1) {
				// Yes, this is the line we wish to jump to
				return e.GoTo(i, c, status)
			}
		}
	}
	return false, false
}

// GoToPrevParagraph will jump to the previous line that has a blank line below it, if possible
// Returns true if the editor should be redrawn, and true if the end has been reached
func (e *Editor) GoToPrevParagraph(c *vt.Canvas, status *StatusBar) (bool, bool) {
	lastFoundBlankLine := LineIndex(e.Len())
	for i := e.DataY() - 1; i >= 0; i-- {
		// Check if this is a blank line
		if strings.TrimSpace(e.Line(i)) == "" {
			lastFoundBlankLine = i
		} else {
			// This is a non-blank line, check if the line below is blank (or after the last line)
			if lastFoundBlankLine == (i + 1) {
				// Yes, this is the line we wish to jump to
				return e.GoTo(i, c, status)
			}
		}
	}
	return false, false
}

// Center will scroll the contents so that the line with the cursor ends up in the center of the screen
func (e *Editor) Center(c *vt.Canvas) {
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
	var space string
	switch e.mode {
	// For config files, assume things will be toggled in and out, without a space
	case mode.Config, mode.Ini, mode.FSTAB, mode.Nix:
		break // space == ""
	default:
		space = " "
	}
	e.SetCurrentLine(commentMarker + space + e.CurrentLine())
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

// ForEachLineInBlockWithCommentMarker will move the cursor and run the given function for
// each line in the current block of text (until newline or end of document)
// Also takes a string that will be passed on to the function.
func (e *Editor) ForEachLineInBlockWithCommentMarker(c *vt.Canvas, f func(string), commentMarker string) {
	downCounter := 0
	for !e.EmptyRightTrimmedLine() {
		f(commentMarker)
		if e.AtOrAfterEndOfDocument() {
			break
		}
		if e.Down(c, nil) { // reached the end
			break
		}
		downCounter++
		if e.EmptyLine() {
			break
		}
	}
	// Go up again
	for i := downCounter; i > 0; i-- {
		e.Up(c, nil)
	}
}

// ForEachLineInBlock will move the cursor and run the given function for
// each line in the current block of text (until newline or end of document).
// It will also keep X the same for each line.
// It will then use the X value after the final successful call of the given function.
func (e *Editor) ForEachLineInBlock(c *vt.Canvas, f func() bool) {
	firstX := e.pos.sx
	firstY := e.pos.sy
	firstOffsetX := e.pos.offsetX
	finalX := e.pos.sx
	finalOffsetX := e.pos.offsetX
	downCounter := 0
	for !e.EmptyRightTrimmedLine() {
		e.pos.sx = firstX
		e.pos.offsetX = firstOffsetX
		if !f() {
			break
		}
		finalX = e.pos.sx
		finalOffsetX = e.pos.offsetX
		if e.AtOrAfterEndOfDocument() {
			break
		}
		if e.Down(c, nil) { // reached the end
			break
		}
		downCounter++
		if e.EmptyLine() {
			break
		}
	}
	// Go up again
	for i := downCounter; i > 0; i-- {
		e.Up(c, nil)
	}
	e.pos.sx = finalX
	e.pos.sy = firstY
	e.pos.offsetX = finalOffsetX
	// Make sure no lines are nil
	e.MakeConsistent()
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
		if strings.TrimSpace(s) == "" {
			// Empty trimmed line, end of block
			return bb.String()
		}
		// Save this line to bb
		bb.WriteString(s)
		// And add a newline
		bb.Write([]byte{'\n'})
	}
}

// FunctionBlock will return the text from the given line until
// either the function name changes or the end of the document is reached.
// If no function name is found, an error is returned.
func (e *Editor) FunctionBlock(n LineIndex) (string, error) {
	var (
		sb                    strings.Builder
		line                  []rune
		ok                    bool
		fname, firstLineIndex = e.FunctionNameForLineIndex(n)
	)
	if fname == "" {
		return "", errors.New("not in a function")
	}

	for n := firstLineIndex; ; n++ {
		line, ok = e.lines[int(n)]
		if !ok || e.OnlyFunctionNameForLineIndex(n) != fname {
			// End of document or end of function
			return sb.String(), nil
		}
		// Save this line
		for _, r := range line {
			sb.WriteRune(r)
		}
		sb.WriteRune('\n')
	}
}

// ToggleCommentBlock will toggle comments until a blank line or the end of the document is reached
// The amount of existing commented lines is considered before deciding to comment the block in or out
func (e *Editor) ToggleCommentBlock(c *vt.Canvas) {
	// If most of the lines in the block are comments, comment it out
	// If most of the lines in the block are not comments, comment it in

	var (
		downCounter    = 0
		commentCounter = 0
		commentMarker  = e.SingleLineCommentMarker()
	)

	// Count the commented lines in this block while going down
	for !e.EmptyRightTrimmedLine() {
		if e.CurrentLineCommented(commentMarker) {
			commentCounter++
		}
		if e.AtOrAfterEndOfDocument() {
			break
		}
		if e.Down(c, nil) { // reached the end
			break
		}
		// TODO: Remove the safeguard
		downCounter++
		if downCounter > 10 { // safeguard at the end of the document
			break
		}
	}
	// Go up again
	for i := downCounter; i > 0; i-- {
		e.Up(c, nil)
	}

	// Check if most lines are commented out
	mostLinesAreComments := commentCounter >= (downCounter / 2)

	// Handle the single-line case differently
	if downCounter == 1 && commentCounter == 0 {
		e.CommentOn(commentMarker)
	} else if downCounter == 1 && commentCounter == 1 {
		e.CommentOff(commentMarker)
	} else if mostLinesAreComments {
		e.ForEachLineInBlockWithCommentMarker(c, e.CommentOff, commentMarker)
	} else {
		e.ForEachLineInBlockWithCommentMarker(c, e.CommentOn, commentMarker)
	}
}

// NewLine inserts a new line below and moves down one step
func (e *Editor) NewLine(c *vt.Canvas, status *StatusBar) {
	e.InsertLineBelow()
	e.Down(c, status)
}

// HorizontalScrollIfNeeded will scroll along the X axis, if needed
func (e *Editor) HorizontalScrollIfNeeded(c *vt.Canvas) {
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
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// VerticalScrollIfNeeded will scroll along the X axis, if needed
func (e *Editor) VerticalScrollIfNeeded(c *vt.Canvas) {
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
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// InsertFile inserts the contents of a file at the current location
func (e *Editor) InsertFile(c *vt.Canvas, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	s := opinionatedStringReplacer.Replace(trimRightSpace(string(data)))
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
func (e *Editor) Switch(c *vt.Canvas, tty *vt.TTY, status *StatusBar, fileLock *LockKeeper, filenameToOpen string) error {
	if !files.Exists(filenameToOpen) {
		return errors.New("could not find file to switch to: " + filenameToOpen)
	}

	absFilename, err := e.AbsFilename()
	if err != nil {
		return err
	}

	// About to switch from absFilename to filenameToOpen

	if fileLock != nil {
		// Unlock and save the lock file
		go func() {
			quitMut.Lock()
			defer quitMut.Unlock()
			fileLock.Unlock(absFilename)
			fileLock.Save()
		}()
	}

	// Now open the header filename instead of the current file. Save the current file first.
	e.Save(c, tty)

	// Save the current location in the location history and write it to file
	e.SaveLocationCustom(absFilename, locationHistory)

	var (
		e2             *Editor
		statusMessage  string
		displayedImage bool
	)

	if switchBuffer.Len() == 1 {
		// Load the Editor from the switchBuffer if switchBuffer has length 1, then use that editor.
		switchBuffer.Restore(e)
		undo, switchUndoBackup = switchUndoBackup, undo
	} else {
		fnord := FilenameOrData{filenameToOpen, []byte{}, 0, false}
		e2, statusMessage, displayedImage, _, err = NewEditor(tty, c, fnord, LineNumber(0), ColNumber(0), e.Theme, e.syntaxHighlight, false, e.monitorAndReadOnly, e.nanoMode.Load(), e.createDirectoriesIfMissing, e.displayQuickHelp, e.noDisplayQuickHelp)
		if err == nil { // no issue
			// Save the current Editor to the switchBuffer if switchBuffer if empty, then use the new editor.
			switchBuffer.Snapshot(e)
			// Now use e2 as the current editor
			const withLines = true
			*e = *(e2.Copy(withLines))
			//*e = *e2
			//(*e).lines = (*e2).lines
			//(*e).pos = (*e2).pos
		} else if displayedImage {
			// internal error
			panic("displayed an image while switching from one Editor struct to another")
		} else {
			// internal error when switching from absFilename to filenameToOpen
			panic(err)
		}
		go fnord.SetTitle()
		undo, switchUndoBackup = switchUndoBackup, undo
	}

	if statusMessage != "" {
		status.SetMessageAfterRedraw(statusMessage)
	}

	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	return err
}

// Reload tries to load the current file again
func (e *Editor) Reload(c *vt.Canvas, tty *vt.TTY, status *StatusBar, lk *LockKeeper) error {
	return e.Switch(c, tty, status, lk, e.filename)
}

// TrimmedLine returns the current line, trimmed in both ends
func (e *Editor) TrimmedLine() string {
	return strings.TrimSpace(e.CurrentLine())
}

// PrevTrimmedLine returns the line above, trimmed in both ends
func (e *Editor) PrevTrimmedLine() string {
	return strings.TrimSpace(e.PrevLine())
}

// NextTrimmedLine returns the line below, trimmed in both ends
func (e *Editor) NextTrimmedLine() string {
	return strings.TrimSpace(e.NextLine())
}

// TrimmedLineAt returns the current line, trimmed in both ends
func (e *Editor) TrimmedLineAt(lineIndex LineIndex) string {
	return strings.TrimSpace(e.Line(lineIndex))
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

// CurrentWord returns the current word under the cursor, or an empty string.
// The word may contain numbers or dashes, but not spaces or special characters.
// "." is included.
func (e *Editor) CurrentWord() string {
	y := int(e.DataY())
	runes, ok := e.lines[y]
	if !ok {
		// This should never happen
		return ""
	}

	// Check if there are letters on the current line
	if len(runes) == 0 {
		return ""
	}

	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	qualifies := func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.'
	}

	if x >= len(runes) {
		return ""
	}

	// Check if the cursor is at a word
	if !qualifies(runes[x]) {
		return ""
	}

	// Find the first letter of the word (or the start of the line)
	firstLetterIndex := 0
	for i := x; i >= 0; i-- {
		r := runes[i]
		if !qualifies(r) {
			break
		}
		firstLetterIndex = i
	}

	// Loop from the first letter of the word, to the first non-letter.
	// Gather the letters.
	var word []rune
	for i := firstLetterIndex; i < len(runes); i++ {
		r := runes[i]
		if !qualifies(r) {
			break
		}
		// Gather the letters
		word = append(word, r)
	}

	// Return the word
	return string(word)
}

// AnyTextBeforeCursor checks if there is any text before the cursor, on the same line
func (e *Editor) AnyTextBeforeCursor() bool {
	y := int(e.DataY())
	runes, ok := e.lines[y]
	if !ok {
		// This should never happen
		return false
	}

	// Either find x or use the last index of the line
	x, err := e.DataX()
	if err != nil {
		x = len(runes)
	}

	return strings.TrimSpace(string(runes[:x])) != ""
}

// LastLineNumber returns the last line number (not line index) of the current file
func (e *Editor) LastLineNumber() LineNumber {
	// The last line (by line number, not by index, e.Len() returns an index which is why there is no -1)
	return LineNumber(e.Len())
}

// UserInput asks the user to enter text, then collects the letters. No history.
func (e *Editor) UserInput(c *vt.Canvas, tty *vt.TTY, status *StatusBar, title, defaultValue string, quickList []string, arrowsAreCountedAsLetters bool, tabInsertText string) (string, bool) {
	status.ClearAll(c, false)
	if defaultValue != "" {
		status.SetMessage(title + ": " + defaultValue)
	} else {
		status.SetMessage(title + ":")
	}
	status.ShowNoTimeout(c, e)
	cancel := false
	entered := defaultValue
	doneCollectingLetters := false
	for !doneCollectingLetters {
		if e.debugMode {
			e.DrawWatches(c, false)      // don't reposition cursor
			e.DrawRegisters(c, false)    // don't reposition cursor
			e.DrawInstructions(c, false) // don't reposition cursor
			e.DrawFlags(c, false)        // don't reposition cursor
			e.DrawGDBOutput(c, false)    // don't reposition cursor
		}
		pressed := tty.ReadStringEvent()
		switch pressed {
		case "c:8", "c:127": // ctrl-h or backspace
			if len(entered) > 0 {
				entered = entered[:len(entered)-1]
				status.SetMessage(title + ": " + entered)
				status.ShowNoTimeout(c, e)
			}
		case leftArrow, rightArrow: // left arrow or right arrow
			fallthrough // cancel
		case upArrow, downArrow: // up arrow or down arrow
			if arrowsAreCountedAsLetters {
				entered += pressed
				status.SetMessage(title + ": " + entered)
				status.ShowNoTimeout(c, e)
			}
			// Is this a special quick command, where return is not needed, like "wq"?
			if hasS(quickList, entered) {
				break
			}
			fallthrough // cancel
		case "c:9":
			if tabInsertText != "" {
				entered = tabInsertText
				status.SetMessage(title + ": " + entered)
				status.ShowNoTimeout(c, e)
			}
		case "c:27", "c:3", "c:24": // esc, ctrl-c or ctrl-x
			cancel = true
			entered = ""
			fallthrough // done
		case "c:13": // return
			doneCollectingLetters = true
		default:
			entered += pressed
			status.SetMessage(title + ": " + entered)
			status.ShowNoTimeout(c, e)
		}
		// Is this a special quick command, where return is not needed, like "wq"?
		if hasS(quickList, entered) {
			break
		}
	}
	status.ClearAll(c, true)
	return entered, !cancel
}

// MoveToNumber will try to move to the given line number + column number (given as strings)
func (e *Editor) MoveToNumber(c *vt.Canvas, status *StatusBar, lineNumber, lineColumn string) error {
	// Move to (x, y), line number first and then column number
	i, err := strconv.Atoi(lineNumber)
	if err != nil {
		return err
	}
	foundY := LineNumber(i)
	redraw, _ := e.GoTo(foundY.LineIndex(), c, status)
	e.redraw.Store(redraw)
	e.redrawCursor.Store(redraw)
	x, err := strconv.Atoi(lineColumn)
	if err != nil {
		return err
	}
	foundX := x - 1
	tabs := strings.Count(e.Line(foundY.LineIndex()), "\t")
	e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
	e.Center(c)
	return nil
}

// MoveToLineColumnNumber will try to move to the given line number + column number (given as ints)
func (e *Editor) MoveToLineColumnNumber(c *vt.Canvas, status *StatusBar, lineNumber, lineColumn int, ignoreIndentation bool) error {
	// Move to (x, y), line number first and then column number
	foundY := LineNumber(lineNumber)
	redraw, _ := e.GoTo(foundY.LineIndex(), c, status)
	e.redraw.Store(redraw)
	e.redrawCursor.Store(redraw)
	x := lineColumn
	foundX := x - 1
	tabs := strings.Count(e.Line(foundY.LineIndex()), "\t")
	e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
	if ignoreIndentation {
		e.pos.sx += len(e.LeadingWhitespace())
	}
	e.Center(c)
	return nil
}

// MoveToIndex will try to move to the given line index + column index (given as strings)
func (e *Editor) MoveToIndex(c *vt.Canvas, status *StatusBar, lineIndex, lineColumnIndex string, subtractOne bool) error {
	// Move to (x, y), line number first and then column number
	i, err := strconv.Atoi(lineIndex)
	if err != nil {
		return err
	}
	if subtractOne {
		i--
	}
	foundY := LineIndex(i)
	redraw, _ := e.GoTo(foundY, c, status)
	e.redraw.Store(redraw)
	e.redrawCursor.Store(redraw)
	x, err := strconv.Atoi(lineColumnIndex)
	if err != nil {
		return err
	}
	if subtractOne {
		x--
	}
	foundX := x - 1
	tabs := strings.Count(e.Line(foundY), "\t")
	e.pos.sx = foundX + (tabs * (e.indentation.PerTab - 1))
	e.Center(c)
	return nil
}

// GoToTop jumps and scrolls to the top of the file
func (e *Editor) GoToTop(c *vt.Canvas, status *StatusBar) {
	e.redraw.Store(e.GoToLineNumber(1, c, status, true))
}

// GoToMiddle jumps and scrolls to the middle of the file
func (e *Editor) GoToMiddle(c *vt.Canvas, status *StatusBar) {
	e.GoToLineNumber(LineNumber(e.Len()/2), c, status, true)
}

// GoToEnd jumps and scrolls to the end of the file
func (e *Editor) GoToEnd(c *vt.Canvas, status *StatusBar) {
	// Go to the last line (by line number, not by index, e.Len() returns an index which is why there is no -1)
	e.redraw.Store(e.GoToLineNumber(LineNumber(e.Len()), c, status, true))
}

// SortBlock sorts the a block of lines, at the current position
func (e *Editor) SortBlock(c *vt.Canvas, status *StatusBar, bookmark *Position) {
	if e.CurrentLine() == "" {
		status.SetErrorMessage("no text block at the current position")
		return
	}
	y := e.LineIndex()
	e.Home()
	s := e.Block(y)
	var lines sort.StringSlice
	lines = strings.Split(s, "\n")
	if len(lines) == 0 {
		status.SetErrorMessage("no text block to sort")
		return
	}
	// Remove the last empty line, if it's there
	addEmptyLine := false
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
		addEmptyLine = true
	}
	lines.Sort()
	e.GoTo(y, c, status)
	e.DeleteBlock(bookmark)
	e.GoTo(y, c, status)
	e.InsertBlock(c, lines, addEmptyLine)
	e.GoTo(y, c, status)
}

// SmartSplitLineOnBlanks splits the current line on space, as separate lines, but not if spaces are within brackets, parentheses or curly brackets
func (e *Editor) SmartSplitLineOnBlanks(c *vt.Canvas, status *StatusBar, bookmark *Position) {
	if e.CurrentLine() == "" {
		status.SetErrorMessage("nothing to split on blanks")
		return
	}
	y := e.LineIndex()

	// Split the current (trimmed) line on space
	lines := smartSplit(e.TrimmedLine())

	// Remove empty elements from the slice
	var trimmedLines []string
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		trimmedLines = append(trimmedLines, trimmedLine)
	}

	// Remove the current line and insert the new lines
	e.DeleteCurrentLineMoveBookmark(bookmark)
	const addEmptyLine = false
	e.InsertBlock(c, trimmedLines, addEmptyLine)
	e.GoTo(y, c, status)
}

// ReplaceBlock replaces the current block with the given string, if possible
func (e *Editor) ReplaceBlock(c *vt.Canvas, status *StatusBar, bookmark *Position, s string) {
	if e.CurrentLine() == "" {
		status.SetErrorMessage("no text block at the current position")
		return
	}
	y := e.LineIndex()
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		status.SetErrorMessage("no text block to replace")
		return
	}
	// Remove the last empty line, if it's there
	addEmptyLine := false
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
		addEmptyLine = true
	}
	e.GoTo(y, c, status)
	e.DeleteBlock(bookmark)
	e.GoTo(y, c, status)
	e.InsertBlock(c, lines, addEmptyLine)
	e.GoTo(y, c, status)
}

// DeleteBlock will deletes a block of lines at the current position
func (e *Editor) DeleteBlock(bookmark *Position) {
	s := e.Block(e.LineIndex())
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		// Need at least 1 line to be able to cut "the rest" after the first line has been cut
		return
	}
	for range lines {
		e.DeleteLineMoveBookmark(e.LineIndex(), bookmark)
	}
}

// InsertBlock will insert multiple lines at the current position, without trimming
// If addEmptyLine is true, an empty line will be added at the end
func (e *Editor) InsertBlock(c *vt.Canvas, addLines []string, addEmptyLine bool) {
	e.InsertLineAbove()
	// copyLines contains the lines to be pasted, and they are > 1
	// the first line is skipped since that was already pasted when ctrl-v was pressed the first time
	lastIndex := len(addLines[1:]) - 1
	// If the first line has been pasted, and return has been pressed, paste the rest of the lines differently
	skipFirstLineInsert := e.EmptyRightTrimmedLine()
	// Insert the lines
	for i, line := range addLines {
		if i == lastIndex && strings.TrimSpace(line) == "" {
			// If the last line is blank, skip it
			break
		}
		if skipFirstLineInsert {
			skipFirstLineInsert = false
		} else {
			e.InsertLineBelow()
			e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
		}
		e.InsertStringAndMove(c, line)
	}
	if addEmptyLine {
		e.InsertLineBelow()
		e.Down(c, nil) // no status message if the end of document is reached, there should always be a new line
	}
}

// LineIsBlank checks if the given line index is blank
func (e *Editor) LineIsBlank(y LineIndex) bool {
	return strings.TrimSpace(e.Line(y)) == ""
}

// NextLineIsBlank checks if the next line is blank
func (e *Editor) NextLineIsBlank() bool {
	return e.LineIsBlank(e.DataY() + 1)
}

// OnParenOrBracket checks if we are currently on a parenthesis or bracket
func (e *Editor) OnParenOrBracket() bool {
	switch e.Rune() {
	case '(', ')', '{', '}', '[', ']':
		return true
	default:
		return false
	}
}

// DeleteToEndOfLine (ctrl-k)
func (e *Editor) DeleteToEndOfLine(c *vt.Canvas, status *StatusBar, bookmark *Position, lastCopyY, lastPasteY, lastCutY *LineIndex) {
	if e.Empty() {
		status.SetMessage("Empty file")
		status.Show(c, e)
		return
	}
	// Reset the cut/copy/paste double-keypress detection
	*lastCopyY = -1
	*lastPasteY = -1
	*lastCutY = -1
	if e.EmptyLine() {
		e.DeleteCurrentLineMoveBookmark(bookmark)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return
	}
	e.DeleteRestOfLine()
	if e.EmptyRightTrimmedLine() {
		// Deleting the rest of the line cleared this line,
		// so just remove it.
		e.DeleteCurrentLineMoveBookmark(bookmark)
		// Then go to the end of the line, if needed
		if e.AfterEndOfLine() {
			e.End(c)
		}
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// CutSingleLine can be called when ctrl-x is pressed (or ctrl-k in nano mode)
// returns true if a multi-line cut should happen instead
func (e *Editor) CutSingleLine(status *StatusBar, bookmark *Position, lastCutY, lastCopyY, lastPasteY *LineIndex, copyLines *[]string, firstCopyAction *bool) (y LineIndex, multiLineCut bool) {
	y = e.DataY()
	line := e.Line(y)
	// Now check if there is anything to cut
	if strings.TrimSpace(line) == "" {
		// Nothing to cut, just remove the current line
		e.Home()
		e.DeleteCurrentLineMoveBookmark(bookmark)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		// Check if ctrl-x was pressed once or twice, for this line
		return y, false
	}
	if *lastCutY != y { // Single line cut
		// Also close the portal, if any
		e.ClosePortal()
		*lastCutY = y
		*lastCopyY = -1
		*lastPasteY = -1
		// Copy the line internally
		*copyLines = []string{line}
		var err error
		if isDarwin {
			// Copy the line to the clipboard
			err = pbcopy(line)
		} else {
			// Copy the line to the non-primary clipboard
			err = clip.WriteAll(line, e.primaryClipboard)
		}
		if err != nil && *firstCopyAction {
			if env.Has("WAYLAND_DISPLAY") && files.WhichCached("wl-copy") == "" { // Wayland
				status.SetErrorMessage("The wl-copy utility (from wl-clipboard) is missing!")
			} else if env.Has("DISPLAY") && files.WhichCached("xclip") == "" {
				status.SetErrorMessage("The xclip utility is missing!")
			} else if isDarwin && files.WhichCached("pbcopy") == "" { // pbcopy is missing, on macOS
				status.SetErrorMessage("The pbcopy utility is missing!")
			}
		}
		// Delete the line
		e.DeleteLineMoveBookmark(y, bookmark)
		// No status message is needed for the cut operation, because it's visible that lines are cut
		e.redrawCursor.Store(true)
		e.redraw.Store(true)
		return y, false
	}
	return y, true
}

// CursorForward moves the cursor forward 1 step
func (e *Editor) CursorForward(c *vt.Canvas, status *StatusBar) {
	// If on the last line or before, go to the next character
	if e.DataY() <= LineIndex(e.Len()) {
		e.Next(c)
	}
	if e.AfterScreenWidth(c) {
		e.pos.SetOffsetX(e.pos.OffsetX() + 1)
		e.redraw.Store(true)
		e.pos.sx--
		if e.pos.sx < 0 {
			e.pos.sx = 0
		}
		if e.AfterEndOfLine() {
			e.Down(c, status)
		}
	} else if e.AfterEndOfLine() && e.pos.ScreenY() == int(c.H())-1 {
		e.Down(c, status)
		e.Home()
	} else if e.AfterEndOfLine() {
		e.End(c)
	}

	e.SaveX(true)
	e.redrawCursor.Store(true)
}

// CursorBackward moves the cursor backward 1 step
func (e *Editor) CursorBackward(c *vt.Canvas, status *StatusBar) {
	// movement if there is horizontal scrolling
	if e.pos.OffsetX() > 0 {
		if e.pos.sx > 0 {
			// Move one step left
			if e.TabToTheLeft() {
				e.pos.sx -= e.indentation.PerTab
			} else {
				e.pos.sx--
			}
		} else {
			// Scroll one step left
			e.pos.SetOffsetX(e.pos.OffsetX() - 1)
			e.redraw.Store(true)
		}
		e.SaveX(true)
	} else if e.pos.sx > 0 {
		// no horizontal scrolling going on
		// Move one step left
		if e.TabToTheLeft() {
			e.pos.sx -= e.indentation.PerTab
		} else {
			e.pos.sx--
		}
		e.SaveX(true)
	} else if e.DataY() > 0 {
		// no scrolling or movement to the left going on
		e.Up(c, status)
		e.End(c)
		// e.redraw.Store(true)
	} // else at the start of the document
	e.redrawCursor.Store(true)
	// Workaround for Konsole
	if e.pos.sx <= 2 {
		// Konsole prints "2H" here, but
		// no other terminal emulator does that
		e.redraw.Store(true)
	}
}

// CursorUpward moves the cursor up 1 step
func (e *Editor) CursorUpward(c *vt.Canvas, status *StatusBar) {
	if e.DataY() > 0 {
		// Move the position up in the current screen
		if e.UpEnd(c) != nil {
			// If below the top, scroll the contents up
			if e.DataY() > 0 {
				e.redraw.Store(e.ScrollUp(c, status, 1))
				e.pos.Down(c)
				e.UpEnd(c)
			}
		}
		// If the cursor is after the length of the current line, move it to the end of the current line
		if e.AfterLineScreenContents() {
			e.End(c)
		}
	}
	// If the cursor is after the length of the current line, move it to the end of the current line
	if e.AfterLineScreenContents() || e.AfterEndOfLine() {
		e.End(c)
		// Then, if the rune to the left is '}', move one step to the left
		if r := e.LeftRune(); r == '}' {
			e.Prev(c)
		}
		e.redraw.Store(true)
	}
}

// CursorDownward moves the cursor down 1 step
func (e *Editor) CursorDownward(c *vt.Canvas, status *StatusBar) {
	if e.DataY() < LineIndex(e.Len()) {
		// Move the position down in the current screen
		if e.DownEnd(c) != nil {
			// If at the bottom, don't move down, but scroll the contents
			// Output a helpful message
			if !e.AfterEndOfDocument() {
				canvasHeight := int(c.Height())
				e.redraw.Store(e.ScrollDown(c, status, 1, canvasHeight))
				e.pos.Up()
				e.DownEnd(c)
			}
		}
		// If the cursor is after the length of the current line, move it to the end of the current line
		if e.AfterLineScreenContents() {
			e.End(c)
			// Then, if the rune to the left is '}', move one step to the left
			if r := e.LeftRune(); r == '}' {
				e.Prev(c)
			}
		}
	}
}

// JoinLineWithNext tries to join the current line with the next.
// If the next line is empty, the next line is removed.
// Returns true if this and the next line had text and they were joined to this line.
func (e *Editor) JoinLineWithNext(c *vt.Canvas, bookmark *Position) bool {
	nextLineIndex := e.DataY() + 1
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
	if e.EmptyRightTrimmedLineBelow() {
		// Just delete the line below if it's empty
		e.DeleteLineMoveBookmark(nextLineIndex, bookmark)
		return false
	}
	// Join the line below with this line. Also add a space in between.
	e.TrimLeft(nextLineIndex) // this is unproblematic, even at the end of the document
	e.End(c)
	e.InsertRune(c, ' ')
	e.WriteRune(c)
	e.Next(c)
	e.Delete(c, false)
	return true
}

// EnableAndPlaceCursor first sets the cursor to shown and then places it at the right position
func (e *Editor) EnableAndPlaceCursor(c *vt.Canvas) {
	//e.pos.mut.Lock()
	x := uint(e.pos.ScreenX())
	y := uint(e.pos.ScreenY())
	//e.pos.mut.Unlock()
	c.ShowCursor()
	vt.SetXY(x, y)
}

// OnIncludeLine checks if it looks like the current line is a C or C++ #include
func (e *Editor) OnIncludeLine() bool {
	return strings.HasPrefix(e.TrimmedLine(), "#include")
}

// GetXY tries to fetch the current x and y positions, as uints
func (e *Editor) GetXY() (uint, uint) {
	x, _ := e.DataX()
	y := e.DataY()
	return uint(x), uint(y)
}

// GetXYAfterText tries to fetch the current x and y positions, as uints, at the end of the text
func (e *Editor) GetXYAfterText() (uint, uint) {
	y := e.DataY()
	x := e.LastTextPosition(y) + 1
	return uint(x), uint(y)
}
