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

	"github.com/xyproto/env/v2"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt100"
)

var (
	specificLetter bool             // did the editor executable start with a specific letter, or just "o"?
	editTheme      bool             // does the theme has both a dark and a light version?
	inVTEGUI       = env.Bool("OG") // is o running within the VTE GUI application?
	tempDir        = env.Dir("TMPDIR", "/tmp")
)

// NewEditor takes a filename and a line number to jump to (may be 0)
// Returns an Editor, a status message for the user, a bool that is true if an image was displayed instead and the finally an error type.
func NewEditor(tty *vt100.TTY, c *vt100.Canvas, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, theme Theme, origSyntaxHighlight, discoverBGColor, monitorAndReadOnly bool) (*Editor, string, bool, error) {
	var (
		startTime          = time.Now()
		createdNewFile     bool   // used for indicating that a new file was created
		scrollSpeed        = 10   // number of lines to scroll when using `ctrl-n` and `ctrl-p`
		statusMessage      string // used when loading or creating a file, for the initial status message
		found              bool
		recordedLineNumber LineNumber
		err                error
		readOnly           = fnord.stdin || monitorAndReadOnly
		m                  mode.Mode // mode is what would have been an enum in other languages, for signalling if this file should be in git mode, markdown mode etc
		syntaxHighlight    bool
	)

	baseFilename := filepath.Base(fnord.filename)
	ext := filepath.Ext(baseFilename)

	// Check if the given filename is an image
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".ico", ".gif", ".bmp", ".webp":
		const waitForKeypress = true
		return nil, "", true, displayImage(c, fnord.filename, waitForKeypress)
	}

	if fnord.stdin {
		m = mode.SimpleDetectBytes(fnord.data)
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && m != mode.Blank
	} else {
		m = mode.Detect(stripGZ(fnord.filename)) // Note that mode.Detect can check for the full path, like /etc/fstab
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && (m != mode.Blank || ext != "")
	}

	adjustSyntaxHighlightingKeywords(m) // no theme changes, just language detection and keyword configuration

	indentation := m.TabsSpaces()

	// Additional per-mode considerations, before launching the editor
	rainbowParenthesis := syntaxHighlight // rainbow parenthesis
	switch m {
	case mode.Blank, mode.Doc, mode.Email, mode.Markdown, mode.Text, mode.ReStructured:
		rainbowParenthesis = false
	case mode.ManPage:
		readOnly = true
	}

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e := NewCustomEditor(indentation,
		scrollSpeed,
		m,
		theme,
		syntaxHighlight,
		rainbowParenthesis,
		monitorAndReadOnly)

	if readOnly || fnord.stdin || monitorAndReadOnly {
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

	if fnord.stdin && !fnord.Empty() { // we have data from stdin

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {
			return nil, "", false, err
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
			// A length of 500 should be enough to detect the contents.
			// If not, th rest of the data can be read through the anonymous
			// function given to DetectFromContentBytes.
			if len(firstLine) > 512 {
				firstLine = firstLine[:512]
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

	} else if fileInfo, err := os.Stat(e.filename); err == nil { // no issue

		// TODO: Enter file-rename mode when opening a directory?
		// Check if this is a directory
		if fileInfo.IsDir() {
			return nil, "", false, errors.New(e.filename + " is a directory")
		}

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {
			return nil, "", false, err
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
			testfile, err := os.OpenFile(e.filename, os.O_WRONLY, 0o664)
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
	} else {

		// Prepare an empty file
		if newMode, err := e.PrepareEmpty(); err != nil {
			return nil, "", false, err
		} else if newMode != mode.Blank {
			e.mode = newMode
		}

		// Test save, to check if the file can be created and written, or not
		if err := e.Save(c, tty); err != nil {
			// Check if the new file can be saved before the user starts working on the file.
			return nil, "", false, err
		}

		// Creating a new empty file worked out fine, don't save it until the user saves it
		if os.Remove(e.filename) != nil {
			// This should never happen
			return nil, "", false, errors.New("could not remove an empty file that was just created: " + e.filename)
		}
		createdNewFile = true
	}

	if env.Bool("OG") {
		// Workaround for an issue where opening empty or small files is too quick for the GUI/VTE wrapper
		time.Sleep(500 * time.Millisecond)
	}

	// The editing mode is decided at this point

	// The shebang may have been for bash, make further adjustments
	adjustSyntaxHighlightingKeywords(e.mode)

	// Additional per-mode considerations, before launching the editor
	e.indentation = m.TabsSpaces()
	if e.detectedTabs != nil {
		detectedTabs := *(e.detectedTabs)
		e.indentation.Spaces = !detectedTabs
	}

	switch e.mode {
	case mode.Blank, mode.Doc, mode.Email, mode.Markdown, mode.Text, mode.ReStructured:
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

	// Use a light theme if XTERM_VERSION (and not running with "og") or
	// TERMINAL_EMULATOR is set to "JetBrains-JediTerm",
	// because $COLORFGBG is "15;0" even though the background is white.
	if !e.readOnly && (!specificLetter || editTheme) {
		themeEnv := env.StrAlt("O_THEME", "THEME")
		if themeEnv == "redblack" {
			b := false
			initialLightBackground = &b
			e.setRedBlackTheme()
		} else if themeEnv == "synthwave" {
			b := false
			initialLightBackground = &b
			e.setSynthwaveTheme()
		} else if themeEnv == "blueedit" {
			b := false
			initialLightBackground = &b
			e.setBlueEditTheme()
		} else if themeEnv == "vs" {
			b := false
			initialLightBackground = &b
			e.setVSTheme()
		} else if themeEnv == "ambermono" {
			envNoColor = false
			e.setAmberTheme()
			e.syntaxHighlight = false
		} else if themeEnv == "greenmono" {
			envNoColor = false
			e.setGreenTheme()
			e.syntaxHighlight = false
		} else if themeEnv == "bluemono" {
			envNoColor = false
			e.setBlueTheme()
			e.syntaxHighlight = false
		} else if (env.Has("XTERM_VERSION") && !inVTEGUI && env.Str("ALACRITTY_LOG") == "") || env.Str("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
			b := true
			initialLightBackground = &b
			if editTheme {
				e.setLightBlueEditTheme()
			} else {
				e.setLightVSTheme()
			}
		} else if shell := env.Str("SHELL"); (shell == "/bin/csh" || shell == "/bin/ksh" || strings.HasPrefix(shell, "/usr/local/bin")) && !inVTEGUI && filepath.Base(os.Args[0]) != "default" {
			// This is likely to be FreeBSD or OpenBSD (and the executable/link name is not "default")
			e.setRedBlackTheme()
		} else if colorString := env.Str("COLORFGBG"); strings.Contains(colorString, ";") {
			fields := strings.Split(colorString, ";")
			backgroundColor := fields[len(fields)-1]
			// 10 (light green), 11 (yellow), 12 (light blue), 13 (light purple), 14 (light cyan) or white
			if backgroundColorNumber, err := strconv.Atoi(backgroundColor); err == nil && backgroundColorNumber >= 10 {
				b := true
				initialLightBackground = &b
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
				b := true
				initialLightBackground = &b
				if editTheme {
					e.setLightBlueEditTheme()
				} else {
					e.setLightVSTheme()
				}
			}
		}
	}

	// Find the absolute path to this filename
	absFilename := e.filename
	if !fnord.stdin {
		if filename, err := e.AbsFilename(); err == nil { // success
			absFilename = filename
		}
	}

	// Load the location history. This will be saved again later. Errors are ignored.
	if locationHistory, err = LoadLocationHistory(locationHistoryFilename); err == nil { // success
		recordedLineNumber, found = locationHistory.Get(absFilename)
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
		locationHistory = make(LocationHistory, 1)
		locationHistory.Set(absFilename, lineNumber)
	}

	// Redraw the TUI, if needed
	if e.redraw {
		e.Center(c)
		e.DrawLines(c, true, false)
		e.redraw = false
	}

	// If SSH_TTY or TMUX is set, redraw everything and then display the status message
	e.sshMode = !inVTEGUI && ((env.Str("SSH_TTY") != "" || env.Str("TMUX") != "" || strings.Contains(env.Str("TERMCAP"), "|screen.")) && !env.Bool("NO_SSH_MODE"))

	// Craft an appropriate status message
	if createdNewFile {
		statusMessage = "New " + e.filename
	} else if e.Empty() && !fnord.stdin {
		statusMessage = "Loaded empty file: " + e.filename + warningMessage
		if e.readOnly {
			statusMessage += " (read only)"
		}
	} else {
		// If startup is slow (> 100 ms), display the startup time in the status bar
		if fnord.stdin {
			statusMessage += "Read from stdin"
		} else {
			statusMessage += "Loaded " + e.filename
		}
		if e.binaryFile {
			statusMessage += " (binary)"
		}

		// Take not of the startup duration, in milliseconds
		startupMilliseconds := time.Since(startTime).Milliseconds()

		if startupMilliseconds > 100 {
			statusMessage += fmt.Sprintf(" (%dms)", startupMilliseconds)
		}
		if warningMessage != "" {
			statusMessage += warningMessage
		}
		if e.readOnly {
			statusMessage += " (read only)"
		}
	}

	return e, statusMessage, false, nil
}
