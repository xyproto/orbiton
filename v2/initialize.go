package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/env"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	specificLetter bool // did the editor executable start with a specific letter, or just "o"?
	editTheme      bool // does the theme has both a dark and a light version?
)

// NewEditor takes a filename and a line number to jump to (may be 0)
// Returns an Editor, a status message and an error type
func NewEditor(tty *vt100.TTY, c *vt100.Canvas, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, theme Theme, origSyntaxHighlight, discoverBGColor bool) (*Editor, string, error) {

	var (
		startTime          = time.Now()
		createdNewFile     bool   // used for indicating that a new file was created
		scrollSpeed        = 10   // number of lines to scroll when using `ctrl-n` and `ctrl-p`
		statusMessage      string // used when loading or creating a file, for the initial status message
		found              bool
		recordedLineNumber LineNumber
		err                error
		readOnly           bool
		m                  mode.Mode // mode is what would have been an enum in other languages, for signalling if this file should be in git mode, markdown mode etc
		syntaxHighlight    bool
	)

	baseFilename := filepath.Base(fnord.filename)
	ext := filepath.Ext(baseFilename)

	if fnord.Empty() {
		m = mode.Detect(fnord.filename) // Note that mode.Detect can check for the full path, like /etc/fstab
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && (m != mode.Blank || ext != "")
	} else {
		m = mode.SimpleDetectBytes(fnord.data)
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && m != mode.Blank
	}

	adjustSyntaxHighlightingKeywords(m) // no theme changes, just language detection and keyword configuration

	tabsSpaces := m.TabsSpaces()

	// Additional per-mode considerations, before launching the editor
	rainbowParenthesis := syntaxHighlight // rainbow parenthesis
	switch m {
	case mode.Blank, mode.Doc, mode.Email, mode.Markdown, mode.Text:
		rainbowParenthesis = false
	case mode.ManPage:
		readOnly = true
	}

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e := NewCustomEditor(tabsSpaces,
		scrollSpeed,
		m,
		theme,
		syntaxHighlight,
		rainbowParenthesis)

	if readOnly {
		e.readOnly = true
	}

	// For non-highlighted files, adjust the word wrap
	if !e.syntaxHighlight {
		// Adjust the word wrap if the terminal is too narrow
		w := int(c.Width())
		if w < e.wrapWidth {
			e.wrapWidth = w
		}
	}

	// Minor adjustments for some file types
	switch e.mode {
	case mode.Email, mode.Git, mode.ManPage:
		e.clearOnQuit = true
	}

	// Set the editor filename
	e.filename = fnord.filename

	// We wish to redraw the canvas and reposition the cursor
	e.redraw = true
	e.redrawCursor = true

	// Use os.Stat to check if the file exists, and load the file if it does
	var warningMessage string

	if !fnord.Empty() { // we have data from stdin

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {
			return nil, "", err
		}

		// Detect the file mode if the current editor mode is blank, or Prolog (since it could be Perl)
		// Markdown is set by default for some files.
		// This corresponds to the check below, and both needs to be updated in sync.
		if e.mode == mode.Blank || e.mode == mode.Prolog || e.mode == mode.Config || e.mode == mode.Markdown {
			var firstLine []byte
			byteLines := bytes.SplitN(fnord.data, []byte{'\n'}, 2)
			if len(byteLines) > 0 {
				firstLine = byteLines[0]
			} else {
				firstLine = fnord.data
			}

			// If the first line is too long, shorten it a bit.
			// A length of 100 should be enough to detect the contents.
			// If not, th rest of the data can be read through the anonymous
			// function given to DetectFromContentBytes.
			if len(firstLine) > 100 {
				firstLine = firstLine[:100]
			}

			// fnord.data is is wrapped in a function, since some types of data may be streamed
			// if the first line is not enough to determine the content type.
			if m, found := mode.DetectFromContentBytes(e.mode, firstLine, func() []byte { return fnord.data }); found {
				e.mode = m
			}
		}

		// Specifically enable syntax highlighting if the opened file is a configuration file
		if e.mode == mode.Config {
			e.syntaxHighlight = true
		}

	} else {

		// This is possibly a directory, or an attempt to create a new and empty file

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {

			// Could not load a file, this is possibly a directory

			// Prepare an empty file
			if newMode, err := e.PrepareEmpty(c, tty, e.filename); err != nil {
				return nil, "", err
			} else if newMode != mode.Blank {
				e.mode = newMode
			}
			// Test save, to check if the file can be created and written, or not
			if err := e.Save(c, tty); err != nil {
				// Check if the new file can be saved before the user starts working on the file.
				return nil, "", err
			}
			// Creating a new empty file worked out fine, don't save it until the user saves it
			if os.Remove(e.filename) != nil {
				// This should never happen
				return nil, "", errors.New("could not remove an empty file that was just created: " + e.filename)
			}
			createdNewFile = true

			//return nil, "", err
		}

		if !e.Empty() {
			// Detect the file mode if the current editor mode is blank (or Prolog, since it could be Perl)
			// Markdown is set by default for some files.
			// This corresponds to the check furthe up, and both needs to be updated in sync.
			if e.mode == mode.Blank || e.mode == mode.Prolog || e.mode == mode.Config || (e.mode == mode.Markdown && ext != ".md") {
				firstLine := e.Line(0)
				// The first 100 bytes are enough when trying to detect the contents
				if len(firstLine) > 100 {
					firstLine = firstLine[:100]
				}
				if m, found := mode.DetectFromContents(e.mode, firstLine, e.String); found {
					e.mode = m
				}
			}
			// Specifically enable syntax highlighting if the opened file is a configuration file or a Man page
			if e.mode == mode.Config {
				e.syntaxHighlight = true
			}
		}

		if !e.slowLoad {
			// Test open, to check if the file can be written or not
			testfile, err := os.OpenFile(e.filename, os.O_WRONLY, 0664)
			if err != nil {
				// can not open the file for writing
				e.readOnly = true
				// Set the color to red when in read-only mode, unless the file is not in "/usr/share/doc",
				// in which case the user is unlikely to want to save the file,
				// and is likely to want a better reading experience.
				if !strings.HasPrefix(e.filename, "/usr/share/doc") {
					e.Foreground = vt100.LightRed
					// disable syntax highlighting, to make it clear that the text is red
					e.syntaxHighlight = false
				}
			}
			testfile.Close()
		}
	}

	// The editing mode is decided at this point

	// The shebang may have been for bash, make further adjustments
	adjustSyntaxHighlightingKeywords(e.mode)

	// Additional per-mode considerations, before launching the editor
	e.tabsSpaces = m.TabsSpaces()

	switch e.mode {
	case mode.Blank, mode.Doc, mode.Email, mode.Markdown, mode.Text:
		e.rainbowParenthesis = false
	}

	// If we're editing a git commit message, add a newline and enable word-wrap at 72
	if e.mode == mode.Git {
		e.Git = vt100.LightGreen
		if filepath.Base(e.filename) == "MERGE_MSG" {
			e.InsertLineBelow()
		} else if e.EmptyLine() {
			e.InsertLineBelow()
		}
		// TODO: Are these two needed, or covered by NewCustomEditor?
		e.wrapWidth = 72
		e.wrapWhenTyping = true
	} else if e.mode == mode.Email {
		// TODO: Are these two needed, or covered by NewCustomEditor?
		e.wrapWidth = 72
		e.wrapWhenTyping = true
		e.GoToEnd(c, nil)
	}

	// If the file starts with a hash bang, enable syntax highlighting
	if strings.HasPrefix(strings.TrimSpace(e.Line(0)), "#!") && !e.readOnly {
		// Enable syntax highlighting and redraw
		e.syntaxHighlight = true
	}

	// Use a light theme if XTERM_VERSION (and not running with "ko") or
	// TERMINAL_EMULATOR is set to "JetBrains-JediTerm",
	// because $COLORFGBG is "15;0" even though the background is white.
	if !e.readOnly && (!specificLetter || editTheme) {
		inKO := env.Bool("KO")
		if (env.Has("XTERM_VERSION") && !inKO && env.Str("ALACRITTY_LOG") == "") || env.Str("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
			if editTheme {
				e.setLightBlueEditTheme()
			} else {
				e.setLightVSTheme()
			}
		} else if shell := env.Str("SHELL"); (shell == "/bin/csh" || shell == "/bin/ksh" || strings.HasPrefix(shell, "/usr/local/bin")) && !inKO && filepath.Base(os.Args[0]) != "default" {
			// This is likely to be FreeBSD or OpenBSD (and the executable/link name is not "default")
			e.setRedBlackTheme()
		} else if colorString := env.Str("COLORFGBG"); strings.Contains(colorString, ";") {
			fields := strings.Split(colorString, ";")
			backgroundColor := fields[len(fields)-1]
			// 10 (light green), 11 (yellow), 12 (light blue), 13 (light purple), 14 (light cyan) or white
			if backgroundColorNumber, err := strconv.Atoi(backgroundColor); err == nil && backgroundColorNumber >= 10 {
				if editTheme {
					e.setLightBlueEditTheme()
				} else {
					e.setLightVSTheme()
				}
			}
		} else if discoverBGColor {
			// r, g, b is the background color from the current terminal emulator, if available
			// Checke if the combined value of r, g and b (0..1) is larger than 2
			// (a bit arbitrary, but should work for most cases)
			if r, g, b, err := vt100.GetBackgroundColor(tty); err == nil && r+g+b > 2 { // success and the background is not dark
				if editTheme {
					e.setLightBlueEditTheme()
				} else {
					e.setLightVSTheme()
				}
			}
		}
	}

	// Find the absolute path to this filename
	absFilename, err := e.AbsFilename()
	if err != nil {
		// This should never happen, just use the given filename
		absFilename = e.filename
	}

	// Load the location history. This will be saved again later. Errors are ignored.
	if locationHistory, err = LoadLocationHistory(locationHistoryFilename); err == nil { // success
		recordedLineNumber, found = locationHistory[absFilename]
	}

	if !e.slowLoad {
		// Load the search history. This will be saved again later. Errors are ignored.
		searchHistory, _ = LoadSearchHistory(searchHistoryFilename)
	}

	// Jump to the correct line number
	switch {
	case lineNumber > 0:
		if colNumber > 0 {
			e.GoToLineNumberAndCol(lineNumber, colNumber, c, nil, false)
		} else {
			e.GoToLineNumber(lineNumber, c, nil, false)
		}
		e.redraw = true
		e.redrawCursor = true
	case lineNumber == 0 && e.mode != mode.Git && e.mode != mode.Email:
		// Load the o location history, if a line number was not given on the command line (and if available)
		if !found && !e.slowLoad {
			// Try to load the NeoVim location history, then
			recordedLineNumber, err = FindInNvimLocationHistory(nvimLocationHistoryFilename, absFilename)
			found = err == nil
		}
		if !found && !e.slowLoad {
			// Try to load the ViM location history, then
			recordedLineNumber, err = FindInVimLocationHistory(vimLocationHistoryFilename, absFilename)
			found = err == nil
		}
		// Check if an existing line number was found
		if found {
			lineNumber = recordedLineNumber
			e.GoToLineNumber(lineNumber, c, nil, true)
			e.redraw = true
			e.redrawCursor = true
			break
		}
		fallthrough
	default:
		// Draw editor lines from line 0 to h onto the canvas at 0,0
		e.DrawLines(c, false, false)
		e.redraw = false
	}

	// Make sure the location history isn't empty (the search history can be empty, it's just a string slice)
	if locationHistory == nil {
		locationHistory = make(map[string]LineNumber, 1)
		locationHistory[absFilename] = lineNumber
	}

	// Redraw the TUI, if needed
	if e.redraw {
		e.Center(c)
		e.DrawLines(c, true, false)
		e.redraw = false
	}

	// Record the startup duration, in milliseconds
	//startupMilliseconds := time.Since(startTime).Milliseconds() // Go 1.11 and above only
	startupMilliseconds := int64(time.Since(startTime)) / 1e6

	// Craft an appropriate status message
	if createdNewFile {
		statusMessage = "New " + e.filename
	} else if e.Empty() {
		statusMessage = "Loaded empty file: " + e.filename + warningMessage
		if e.readOnly {
			statusMessage += " (read only)"
		}
	} else {
		// If startup is slow (> 100 ms), display the startup time in the status bar
		if e.filename == "-" || e.filename == "/dev/stdin" {
			if startupMilliseconds >= 100 {
				statusMessage = fmt.Sprintf("Read from stdin%s (%dms)", warningMessage, startupMilliseconds)
			} else {
				statusMessage = fmt.Sprintf("Read from stdin%s", warningMessage)
			}
		} else {
			if startupMilliseconds >= 100 {
				statusMessage = fmt.Sprintf("Loaded %s%s (%dms)", e.filename, warningMessage, startupMilliseconds)
			} else {
				statusMessage = fmt.Sprintf("Loaded %s%s", e.filename, warningMessage)
			}
		}
		if e.readOnly {
			statusMessage += " (read only)"
		}
	}

	// If SSH_TTY or TMUX is set, redraw everything and then display the status message
	e.sshMode = env.Str("SSH_TTY") != "" || env.Str("TMUX") != "" || strings.Contains(env.Str("TERMCAP"), "|screen.")

	return e, statusMessage, nil
}
