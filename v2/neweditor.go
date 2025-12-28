package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xyproto/env/v2"
	"github.com/xyproto/files"
	"github.com/xyproto/megafile"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

var (
	specificLetter    bool             // did the editor executable start with a specific letter, or just "o"?
	editTheme         bool             // does the theme have both a dark and a light version?
	inVTEGUI          = env.Bool("OG") // is o running within the VTE GUI application?
	noDrawUntilResize atomic.Bool      // we are running within the VTE GUI application, but SIGWINCH has not been sent yet
	tempDir           = env.Dir("TMPDIR", "/tmp")
	errFileNotFound   = errors.New("file not found")
)

// NewEditor takes a filename and a line number to jump to (may be 0)
// Returns an Editor, a status message for the user, a bool that is true if an image was displayed instead and the finally an error type.
func NewEditor(tty *vt.TTY, c *vt.Canvas, fnord FilenameOrData, lineNumber LineNumber, colNumber ColNumber, theme Theme, origSyntaxHighlight, discoverBGColor, monitorAndReadOnly, nanoMode, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp bool) (*Editor, string, bool, error) {
	if inVTEGUI {
		noDrawUntilResize.Store(true)
	}

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

	ext := strings.ToLower(filepath.Ext(fnord.filename))

	// Check if the given filename is an image
	switch ext {
	case ".png", ".jpg", ".jpeg", ".ico", ".gif", ".bmp", ".webp", ".qoi":
		const waitForKeypress = true
		return nil, "", true, displayImage(c, fnord.filename, waitForKeypress)
	}

	if parentIsMan == nil {
		b := parentProcessIs("man")
		parentIsMan = &b
	}

	if fnord.stdin {
		if *parentIsMan {
			m = mode.ManPage
		} else {
			m = mode.SimpleDetectBytes(fnord.data)
		}
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && m != mode.Blank
	} else {
		if *parentIsMan {
			m = mode.ManPage
		} else {
			m = mode.Detect(stripGZ(fnord.filename)) // Note that mode.Detect can check for the full path, like /etc/fstab
		}
		syntaxHighlight = origSyntaxHighlight && m != mode.Text && (m != mode.Blank || ext != "")
	}

	adjustSyntaxHighlightingKeywords(m) // no theme changes, just language detection and keyword configuration

	indentation := m.TabsSpaces()

	// Additional per-mode considerations, before launching the editor
	rainbowParenthesis := syntaxHighlight // rainbow parenthesis
	switch m {
	case mode.ASCIIDoc, mode.Blank, mode.Email, mode.Markdown, mode.Text, mode.ReStructured, mode.SCDoc:
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
		monitorAndReadOnly,
		createDirectoriesIfMissing,
		displayQuickHelp,
		noDisplayQuickHelp)

	e.highlightCurrentText = !envNoColor

	if readOnly || fnord.stdin || monitorAndReadOnly {
		e.readOnly = true
	}

	// For non-highlighted files, adjust the word wrap
	if !e.syntaxHighlight && c != nil {
		// Adjust the word wrap if the terminal is too narrow
		w := int(c.Width())
		if w < e.wrapWidth {
			e.wrapWidth = w
		}
	}

	// Minor adjustments for some file types
	switch e.mode {
	case mode.Email, mode.Git, mode.ManPage:
		clearOnQuit.Store(true)
	}

	// Set the editor filename
	e.filename = fnord.filename

	// emulate Nano?
	e.nanoMode.Store(nanoMode)
	if nanoMode {
		e.statusMode = true
	}

	// We wish to redraw the canvas and reposition the cursor
	e.redraw.Store(true)
	e.redrawCursor.Store(true)

	// Use os.Stat to check if the file exists, and load the file if it does
	var warningMessage string

	if fnord.stdin && !fnord.Empty() { // we have data from stdin

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {
			return e, "", false, err
		}

		if *parentIsMan {
			e.mode = mode.ManPage
		}

		// Detect the file mode if the current editor mode is blank, or Prolog (since it could be Perl)
		// Markdown is set by default for some files.
		// This corresponds to the check below, and both needs to be updated in sync.
		// Also detect if Assembly could be Go/Plan9-style Assembly.
		if e.mode == mode.Blank || e.mode == mode.Prolog || e.mode == mode.Config || e.mode == mode.FSTAB || e.mode == mode.Nix || e.mode == mode.Markdown || e.mode == mode.Assembly {
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

		// Specifically enable syntax highlighting if the opened file is a configuration file or log file
		if e.mode == mode.Config || e.mode == mode.Log {
			e.syntaxHighlight = true
		}

	} else if e.filename == "" { // no filename, and no data on stdin

		return e, "", false, errFileNotFound

	} else if fileInfo, err := os.Stat(e.filename); err == nil { // no issue

		// Check if this is a directory
		if fileInfo.IsDir() {
			// Check if there is only one file in that directory
			matches, err := filepath.Glob(strings.TrimSuffix(e.filename, "/") + "/" + "*")
			if err == nil && len(matches) == 1 {
				fnord.filename = matches[0]
				e.filename = matches[0]
			} else {
				e.dirMode = true
				startdirs := []string{e.filename, env.HomeDir(), "/tmp"}
				const title = "·-––—==[ Orbiton File Browser ]==—––-·"
				megaFileState := megafile.New(c, tty, startdirs, title, editorExecutable+" -y")

				megaFileState.WrittenTextColor = e.Foreground
				megaFileState.Background = e.Background
				megaFileState.TitleColor = e.HeaderTextColor
				megaFileState.PromptColor = e.LinkColor
				megaFileState.AngleColor = e.JumpToLetterColor
				megaFileState.EdgeBackground = e.Background            // TODO using e.BoxBackground needs some more work in megafile
				megaFileState.HighlightBackground = vt.BackgroundWhite // TODO add to the theme struct

				if _, err := megaFileState.Run(); err != nil && err != megafile.ErrExit {
					return e, "", false, fmt.Errorf("could not browse %s: %v", e.filename, err)
				}
				os.Exit(0)
			}
		}

		warningMessage, err = e.Load(c, tty, fnord)
		if err != nil {
			return e, "", false, err
		}

		if !e.Empty() {
			// Detect the file mode if the current editor mode is blank (or Prolog, since it could be Perl)
			// Markdown is set by default for some files.
			// This corresponds to the check further up, and both needs to be updated in sync.
			if e.mode == mode.Blank || e.mode == mode.Prolog || e.mode == mode.Config || e.mode == mode.FSTAB || e.mode == mode.Nix || (e.mode == mode.Markdown && ext != ".md") {
				firstLine := e.Line(0)
				// The first 100 bytes are enough when trying to detect the contents
				if len(firstLine) > 100 {
					firstLine = firstLine[:100]
				}
				if *parentIsMan {
					e.mode = mode.ManPage
				} else {
					if m, found := mode.DetectFromContents(e.mode, firstLine, e.String); found {
						e.mode = m
					}
				}
			} else if e.mode == mode.Assembly { // Check if it could be Go/Plan9 style Assembly
				if m, found := mode.DetectFromContents(e.mode, e.String(), e.String); found {
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
				// If the read-only file is in /usr/share/doc or /usr/include, the user is likely to want
				// to have syntax highlighting enabled, despite the file being read-only.
				// If not, set the color to red and disable syntax highlighting.
				// Find the absolute path to this filename
				excludeList := []string{"/usr/share/doc", "/usr/include"}
				wantColors := false
				absFilename := e.filename
				if !fnord.stdin {
					if filename, err := e.AbsFilename(); err == nil { // success
						absFilename = filename
					}
				}
				for _, pathPrefix := range excludeList {
					if strings.HasPrefix(absFilename, pathPrefix) {
						wantColors = true
						break
					}
				}
				if !wantColors {
					e.Foreground = vt.LightRed
					// disable syntax highlighting, to make it clear that the text is red
					e.syntaxHighlight = false
				}
			}
			testfile.Close()
		}
	} else {
		if ok, err := e.PrepareEmptySaveAndRemove(c, tty); err != nil {
			return e, "", false, err
		} else if ok {
			createdNewFile = true
		}
	}

	if inVTEGUI && isDarwin {
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
	case mode.ASCIIDoc, mode.Blank, mode.Email, mode.Markdown, mode.Text, mode.ReStructured, mode.SCDoc:
		e.rainbowParenthesis = false
	}

	// If we're editing a git commit message, add a newline and enable word-wrap at 72
	if e.mode == mode.Git {
		e.Git = vt.LightGreen
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
	if !e.readOnly && (!specificLetter || editTheme || nanoMode) {
		assumeLightBackground := env.Bool("O_LIGHT")
		theme := env.StrAlt("O_THEME", "THEME")
		if envNoColor {
			theme = ""
		}
		switch theme {
		case "redblack":
			e.SetTheme(NewRedBlackTheme(), assumeLightBackground)
		case "synthwave":
			e.SetTheme(NewSynthwaveTheme(), assumeLightBackground)
		case "orb":
			e.SetTheme(NewOrbTheme(), assumeLightBackground)
		case "teal":
			e.SetTheme(NewTealTheme(), assumeLightBackground)
		case "vs":
			e.setVSTheme(assumeLightBackground)
		case "litmus":
			e.SetTheme(NewLitmusTheme(), assumeLightBackground)
		case "blueedit":
			e.setBlueEditTheme(assumeLightBackground)
		case "pinetree":
			e.SetTheme(NewPinetreeTheme(), assumeLightBackground)
		case "zulu":
			e.SetTheme(NewZuluTheme(), assumeLightBackground)
		case "graymono":
			envNoColor = false
			e.setGrayTheme()
			e.syntaxHighlight = false
		case "ambermono":
			envNoColor = false
			e.setAmberTheme()
			e.syntaxHighlight = false
		case "greenmono":
			envNoColor = false
			e.setGreenTheme()
			e.syntaxHighlight = false
		case "bluemono":
			envNoColor = false
			e.setBlueTheme()
			e.syntaxHighlight = false
		default:
			if (env.Has("XTERM_VERSION") && !inVTEGUI && env.Str("ALACRITTY_LOG") == "") || env.Str("TERMINAL_EMULATOR") == "JetBrains-JediTerm" {
				b := true
				initialLightBackground = &b
				if editTheme {
					e.setBlueEditTheme(*initialLightBackground)
				} else {
					e.setLightVSTheme()
				}
			} else if isBSD {
				// NetBSD, FreeBSD, OpenBSD or Dragonfly
				e.SetTheme(NewRedBlackTheme())
				DisableQuickHelpScreen(nil)
				clearOnQuit.Store(true)
			} else if shell := env.Str("SHELL"); shell != "/usr/local/bin/fish" && (shell == "/bin/csh" || shell == "/bin/ksh" || strings.HasPrefix(shell, "/usr/local/bin")) && !inVTEGUI && filepath.Base(os.Args[0]) != "default" {
				// This is likely to be FreeBSD or OpenBSD (and the executable/link name is not "default")
				e.SetTheme(NewRedBlackTheme())
				DisableQuickHelpScreen(nil)
				clearOnQuit.Store(true)
			} else if colorString := env.Str("COLORFGBG"); strings.Contains(colorString, ";") {
				fields := strings.Split(colorString, ";")
				backgroundColor := fields[len(fields)-1]
				// 10 (light green), 11 (yellow), 12 (light blue), 13 (light purple), 14 (light cyan) or white
				if backgroundColorNumber, err := strconv.Atoi(backgroundColor); err == nil && backgroundColorNumber >= 10 {
					b := true
					initialLightBackground = &b
					if editTheme {
						e.setBlueEditTheme(*initialLightBackground)
					} else {
						e.setLightVSTheme()
					}
				}
			} else if discoverBGColor {
				// r, g, b is the background color from the current terminal emulator, if available
				// Checke if the combined value of r, g and b (0..1) is larger than 2
				// (a bit arbitrary, but should work for most cases)
				if r, g, b, err := vt.GetBackgroundColor(tty); err == nil && r+g+b > 2 { // success and the background is not dark
					b := true
					initialLightBackground = &b
					if editTheme {
						e.setBlueEditTheme(*initialLightBackground)
					} else {
						e.setLightVSTheme()
					}
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

	// Jump to the correct line number
	switch {
	case lineNumber > 0:
		if colNumber > 0 {
			const center = false
			const handleTabsAsWell = true
			e.GoToLineNumberAndCol(lineNumber, colNumber, c, nil, center, handleTabsAsWell)
		} else {
			e.GoToLineNumber(lineNumber, c, nil, false)
		}
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
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
			e.redraw.Store(true)
			e.redrawCursor.Store(true)
			break
		}
		fallthrough
	default:
		// Draw editor lines from line 0 to h onto the canvas at 0,0
		e.HideCursorDrawLines(c, false, false, false)
		e.redraw.Store(false)
	}

	// Make sure the location history isn't empty (the search history can be empty, it's just a string slice)
	if locationHistory == nil {
		locationHistory = make(LocationHistory, 1)
		locationHistory.Set(absFilename, lineNumber)
	}

	// Redraw the TUI, if needed
	if e.redraw.Load() && c != nil {
		e.Center(c)
		e.HideCursorDrawLines(c, true, false, false)
		e.redraw.Store(false)
	}

	// If SSH_TTY or TMUX is set, redraw everything and then display the status message
	e.sshMode = !inVTEGUI && ((env.Str("SSH_TTY") != "" || env.Str("TMUX") != "" || strings.Contains(env.Str("TERMCAP"), "|screen.")) && !env.Bool("NO_SSH_MODE"))

	// Craft an appropriate status message
	if createdNewFile {
		statusMessage = "New " + e.filename
	} else if e.Empty() && !fnord.stdin {
		statusMessage = "Loaded empty file: " + files.Relative(e.filename) + warningMessage
		if e.readOnly {
			statusMessage += " (read only)"
		}
	} else {
		// If startup is slow (> 100 ms), display the startup time in the status bar
		if fnord.stdin {
			statusMessage += "Read from stdin"
		} else {
			relFilename := files.Relative(e.filename)
			absFilename, err := filepath.Abs(e.filename)
			if err == nil && len(absFilename) < len(relFilename) {
				statusMessage += "Loaded " + absFilename
			} else {
				statusMessage += "Loaded " + relFilename
			}
		}
		if e.binaryFile {
			statusMessage += " (binary)"
		}

		// Take not of the startup duration, in milliseconds
		startupMilliseconds := time.Since(startTime).Milliseconds()

		if startupMilliseconds > 90 {
			statusMessage += fmt.Sprintf(" (%dms)", startupMilliseconds)
		}
		if warningMessage != "" {
			statusMessage += warningMessage
		}
		if e.readOnly && !fnord.stdin && !monitorAndReadOnly {
			statusMessage += " (read only)"
		}
		if e.monitorAndReadOnly {
			statusMessage += " (monitoring)"
		}
	}

	return e, statusMessage, false, nil
}

// NewCustomEditor takes:
// * the number of spaces per tab (typically 2, 4 or 8)
// * if the text should be syntax highlighted
// * if rainbow parenthesis should be enabled
// * if text edit mode is enabled (as opposed to "ASCII draw mode")
// * the current scroll speed, in lines
// * the following colors:
//   - text foreground
//   - text background
//   - search highlight
//   - multi-line comment
//
// * a syntax highlighting scheme
// * a file mode
// * if directories should be created when saving a file if they are missing
func NewCustomEditor(indentation mode.TabsSpaces, scrollSpeed int, m mode.Mode, theme Theme, syntaxHighlight, rainbowParenthesis, monitorAndReadOnly, createDirectoriesIfMissing, displayQuickHelp, noDisplayQuickHelp bool) *Editor {
	e := &Editor{}
	e.SetTheme(theme)
	e.lines = make(map[int][]rune)
	e.indentation = indentation
	e.syntaxHighlight = syntaxHighlight
	e.rainbowParenthesis = rainbowParenthesis
	e.monitorAndReadOnly = monitorAndReadOnly
	e.createDirectoriesIfMissing = createDirectoriesIfMissing
	e.displayQuickHelp = displayQuickHelp
	e.noDisplayQuickHelp = noDisplayQuickHelp
	p := NewPosition(scrollSpeed)
	e.pos = *p
	// If the file is not to be highlighted, set word wrap to 79 (0 to disable)
	if e.syntaxHighlight {
		e.wrapWidth = 79
		e.wrapWhenTyping = false
	}
	switch m {
	case mode.Email, mode.Git:
		// The subject should ideally be maximum 50 characters long, then the body of the
		// git commit message can be 72 characters long. Because e-mail standards.
		e.wrapWidth = 72
		e.wrapWhenTyping = true
	case mode.ASCIIDoc, mode.Blank, mode.Markdown, mode.ReStructured, mode.SCDoc, mode.Text:
		e.wrapWidth = 79
		e.wrapWhenTyping = false
	}
	e.mode = m
	return e
}

// NewSimpleEditor return a new simple editor, where the settings are 4 spaces per tab, white text on black background,
// no syntax highlighting, text edit mode (as opposed to ASCII draw mode), scroll 1 line at a time, color
// search results magenta, use the default syntax highlighting scheme, don't use git mode and don't use markdown mode,
// then set the word wrap limit at the given column width.
func NewSimpleEditor(wordWrapLimit int) *Editor {
	t := NewDefaultTheme()
	e := NewCustomEditor(mode.DefaultTabsSpaces, 1, mode.Blank, t, false, false, false, false, false, false)
	e.wrapWidth = wordWrapLimit
	e.wrapWhenTyping = true
	return e
}

// PrepareEmptySaveAndRemove prepares an empty document, saves a file and then removes it, just to check
func (e *Editor) PrepareEmptySaveAndRemove(c *vt.Canvas, tty *vt.TTY) (bool, error) {
	// Prepare an empty file
	if newMode, err := e.PrepareEmpty(); err != nil {
		return false, err
	} else if newMode != mode.Blank {
		e.mode = newMode
	}
	// Test save, to check if the file can be created and written, or not
	if err := e.Save(c, tty); err != nil {
		// Check if the new file can be saved before the user starts working on the file.
		return false, err
	}
	// Creating a new empty file worked out fine, don't save it until the user saves it
	if os.Remove(e.filename) != nil {
		// This should never happen
		return true, errors.New("could not remove an empty file that was just created: " + e.filename)
	}
	return true, nil
}
