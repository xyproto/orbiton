package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/xyproto/vt100"
)

const version = "o 2.30.2"

func main() {
	var (
		// Record the time when the program starts
		startTime = time.Now()

		versionFlag = flag.Bool("version", false, "version information")
		helpFlag    = flag.Bool("help", false, "quick overview of hotkeys")
		forceFlag   = flag.Bool("f", false, "open even if already open")
		cpuprofile  = flag.String("cpuprofile", "", "write cpu profile to `file`")
		memprofile  = flag.String("memprofile", "", "write memory profile to `file`")

		statusDuration = 2700 * time.Millisecond

		copyLines  []string  // for the cut/copy/paste functionality
		bookmark   *Position // for the bookmark/jump functionality
		statusMode bool      // if information should be shown at the bottom

		firstLetterSinceStart string
		firstPasteAction      bool = true

		spacesPerTab = 4 // default spaces per tab

		lastCopyY  LineIndex = -1 // used for keeping track if ctrl-c is pressed twice on the same line
		lastPasteY LineIndex = -1 // used for keeping track if ctrl-v is pressed twice on the same line
		lastCutY   LineIndex = -1 // used for keeping track if ctrl-x is pressed twice on the same line

		createdNewFile bool // used for indicating that a new file was created
		readOnly       bool // used for indicating that a loaded file is read-only

		statusMessage  string // used when loading or creating a file, for the initial status message
		warningMessage string // used when loading or creating a file, for the initial status message

		previousKey string // keep track of the previous key press

		lastCommandMenuIndex int // for the command menu
	)

	flag.Parse()

	if *versionFlag {
		fmt.Println(version)
		return
	}

	if *helpFlag {
		fmt.Println(version + " - simple and limited text editor")
		fmt.Print(`
Hotkeys

ctrl-q     to quit
ctrl-s     to save
ctrl-w     for Zig, Rust, V and Go, format with the "... fmt" command
           for C++, format the current file with "clang-format"
           for markdown, toggle checkboxes
           for git interactive rebases, cycle the rebase keywords
ctrl-a     go to start of line, then start of text and then the previous line
ctrl-e     go to end of line and then the next line
ctrl-n     to scroll down 10 lines or go to the next match if a search is active
ctrl-p     to scroll up 10 lines or go to the previous match
ctrl-k     to delete characters to the end of the line, then delete the line
ctrl-g     to toggle filename/line/column/unicode/word count status display
ctrl-d     to delete a single character
ctrl-t     to toggle syntax highlighting
ctrl-o     to open the command menu, where the first option is always "Save and quit"
ctrl-c     to copy the current line, press twice to copy the current block
ctrl-v     to paste one line, press twice to paste the rest
ctrl-x     to cut the current line, press twice to cut the current block
ctrl-b     to toggle a bookmark for the current line, or jump to a bookmark
ctrl-j     to join lines
ctrl-u     to undo (ctrl-z is also possible, but may background the application)
ctrl-l     to jump to a specific line (or press return to jump to the top)
ctrl-f     to find a string
esc        to redraw the screen and clear the last search
ctrl-space to build Go, C++, Zig, V, Rust, Haskell, Markdown, Adoc or Sdoc
ctrl-r     to render the current text to a PDF document
ctrl-\     to toggle single-line comments for a block of code
ctrl-~     to jump to matching parenthesis

See the man page for more information.

Set NO_COLOR=1 to disable colors.

`)
		return
	}

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	filename, lineNumber := FilenameAndLineNumber(flag.Arg(0), flag.Arg(1))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}

	// If the filename ends with "." and the file does not exist, assume this was an attempt at tab-completion gone wrong.
	// If there are multiple files that exist that start with the given filename, open the one first in the alphabet (.cpp before .o)
	if strings.HasSuffix(filename, ".") && !exists(filename) {
		// Glob
		matches, err := filepath.Glob(filename + "*")
		if err == nil && len(matches) > 0 { // no error and at least 1 match
			sort.Strings(matches)
			filename = matches[0]
		}
	}

	// mode is what would have been an enum in other languages, for signalling if this file should be in git mode, markdown mode etc
	mode, syntaxHighlight := detectEditorMode(filename)

	adjustSyntaxHighlightingKeywords(mode)
	// Additional per-mode considerations, before launching the editor
	rainbowParenthesis := syntaxHighlight // rainbow parenthesis
	switch mode {
	case modeMakefile, modePython, modeCMake:
		spacesPerTab = 4
	case modeShell, modeConfig, modeHaskell, modeVim:
		spacesPerTab = 2
	case modeMarkdown, modeText, modeBlank:
		rainbowParenthesis = false
	}

	// Initialize the terminal
	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}
	defer tty.Close()
	vt100.Init()

	// Create a Canvas for drawing onto the terminal
	c := vt100.NewCanvas()
	c.ShowCursor()

	// How many lines to scroll at the time when using `ctrl-n` and `ctrl-p`
	scrollSpeed := 10

	// New editor struct. Scroll 10 lines at a time, no word wrap.
	e := NewEditor(spacesPerTab,
		syntaxHighlight,
		rainbowParenthesis,
		scrollSpeed,
		defaultEditorForeground,
		defaultEditorBackground,
		defaultEditorSearchHighlight,
		defaultEditorMultilineComment,
		defaultEditorMultilineString,
		defaultEditorHighlightTheme,
		mode)

	// Set the editor filename
	e.filename = filename

	// Per file mode editor adjustments
	if e.mode == modeGit {
		e.clearOnQuit = true
	}

	// For non-highlighted files, adjust the word wrap
	if !e.syntaxHighlight {
		// Adjust the word wrap if the terminal is too narrow
		w := int(c.Width())
		if w < e.wordWrapAt {
			e.wordWrapAt = w
		}
	}

	// Use a theme for light backgrounds if XTERM_VERSION is set,
	// because $COLORFGBG is "15;0" even though the background is white.
	if os.Getenv("XTERM_VERSION") != "" {
		e.setLightTheme()
	}
	e.respectNoColorEnvironmentVariable()

	// Prepare a status bar
	status := NewStatusBar(defaultStatusForeground, defaultStatusBackground, defaultStatusErrorForeground, defaultStatusErrorBackground, e, statusDuration)
	status.respectNoColorEnvironmentVariable()

	// We wish to redraw the canvas and reposition the cursor
	e.redraw = true
	e.redrawCursor = true

	// Use os.Stat to check if the file exists, and load the file if it does
	if fileInfo, err := os.Stat(e.filename); err == nil {

		// TODO: Enter file-rename mode when opening a directory?
		// Check if this is a directory
		if fileInfo.IsDir() {
			quitError(tty, errors.New(e.filename+" is a directory"))
		}

		warningMessage, err = e.Load(c, tty, e.filename)
		if err != nil {
			quitError(tty, err)
		}

		if !e.Empty() {
			// Check if the first line is special
			firstLine := e.Line(0)
			if strings.HasPrefix(firstLine, "#!") { // The line starts with a shebang
				words := strings.Split(firstLine, " ")
				lastWord := words[len(words)-1]
				if strings.Contains(lastWord, "/") {
					words = strings.Split(lastWord, "/")
					lastWord = words[len(words)-1]
				}
				switch lastWord {
				case "python":
					e.mode = modePython
				case "bash", "fish", "zsh", "tcsh", "ksh", "sh", "ash":
					e.mode = modeShell
				}
			}
			// If more lines start with "# " than "// " or "/* ", and mode is blank,
			// set the mode to modeConfig and enable syntax highlighting.
			if e.mode == modeBlank {
				hashComment := 0
				slashComment := 0
				for _, line := range strings.Split(e.String(), "\n") {
					if strings.HasPrefix(line, "# ") {
						hashComment++
					} else if strings.HasPrefix(line, "/") { // Count all lines starting with "/" as a comment, for this purpose
						slashComment++
					}
				}
				if hashComment > slashComment {
					e.mode = modeConfig
					e.syntaxHighlight = true
				}
			}
			// If the mode is modeOCaml and there are no ";;" strings, switch to Standard ML
			if e.mode == modeOCaml {
				if !strings.Contains(e.String(), ";;") {
					e.mode = modeStandardML
				}
			}
		}

		// Test write, to check if the file can be written or not
		testfile, err := os.OpenFile(e.filename, os.O_WRONLY, 0664)
		if err != nil {
			// can not open the file for writing
			readOnly = true
			// set the color to red when in read-only mode
			e.fg = vt100.Red
			// disable syntax highlighting, to make it clear that the text is red
			e.syntaxHighlight = false
			// do a full reset and redraw
			e.FullResetRedraw(c, status, false)
			// draw the editor lines again
			e.DrawLines(c, false, true)
			e.redraw = false
		}
		testfile.Close()
	} else {
		// Prepare an empty file
		if newMode, err := e.PrepareEmpty(c, tty, e.filename); err != nil {
			quitError(tty, err)
		} else if newMode != modeBlank {
			e.mode = newMode
		}

		// Test save, to check if the file can be created and written, or not
		if err := e.Save(c); err != nil {
			// Check if the new file can be saved before the user starts working on the file.
			quitError(tty, err)
		} else {
			// Creating a new empty file worked out fine, don't save it until the user saves it
			if os.Remove(e.filename) != nil {
				// This should never happen
				quitError(tty, errors.New("could not remove an empty file that was just created: "+e.filename))
			}
		}
		createdNewFile = true
	}

	// The editing mode is decided at this point

	// The shebang may have been for bash, make further adjustments
	adjustSyntaxHighlightingKeywords(e.mode)

	// Additional per-mode considerations, before launching the editor
	switch e.mode {
	case modeMakefile, modePython, modeCMake:
		e.spacesPerTab = 4
	case modeShell, modeConfig, modeHaskell, modeVim:
		e.spacesPerTab = 2
	case modeMarkdown, modeText, modeBlank:
		e.rainbowParenthesis = false
	}

	// If we're editing a git commit message, add a newline and enable word-wrap at 80
	if e.mode == modeGit {
		e.gitColor = vt100.LightGreen
		status.fg = vt100.LightBlue
		status.bg = vt100.BackgroundDefault
		if filepath.Base(e.filename) == "MERGE_MSG" {
			e.InsertLineBelow()
		} else if e.EmptyLine() {
			e.InsertLineBelow()
		}
		e.wordWrapAt = 80
	}

	// If the file starts with a hash bang, enable syntax highlighting
	if strings.HasPrefix(strings.TrimSpace(e.Line(0)), "#!") {
		// Enable styntax highlighting and redraw
		e.syntaxHighlight = true
		e.bg = defaultEditorBackground
		// Now do a full reset/redraw
		e.FullResetRedraw(c, status, false)
	}

	// Circular undo buffer with room for N actions
	undo := NewUndo(8192)

	// Terminal resize handler
	e.SetUpResizeHandler(c, status, tty)

	// ctrl-c handler
	e.SetUpTerminateHandler(c, status, tty)

	tty.SetTimeout(2 * time.Millisecond)

	previousX := 1
	previousY := 1

	// Find the absolute path to this filename
	absFilename, err := filepath.Abs(e.filename)
	if err != nil {
		// This should never happen, just use the given filename
		absFilename = e.filename
	}
	absFilename = filepath.Clean(absFilename)

	if !(*forceFlag) && ProbablyAlreadyOpen(absFilename) {
		quitError(tty, fmt.Errorf("\"%s\" is locked by another instance of o. Use -f to force open", absFilename))
	}

	var (
		found              bool
		recordedLineNumber LineNumber
	)

	// Load the location history. This will be saved again later. Errors are ignored.
	e.locationHistory, err = LoadLocationHistory(expandUser(locationHistoryFilename))
	if err == nil { // no error
		recordedLineNumber, found = e.locationHistory[absFilename]
	}

	// Load the search history. This will be saved again later. Errors are ignored.
	searchHistory, _ = LoadSearchHistory(expandUser(searchHistoryFilename))

	// Jump to the correct line number
	switch {
	case lineNumber > 0:
		e.GoToLineNumber(lineNumber, c, status, false)
		e.redraw = true
		e.redrawCursor = true
	case lineNumber == 0 && mode != modeGit:
		// Load the o location history, if a line number was not given on the command line (and if available)
		if !found {
			// Try to load the NeoVim location history, then
			recordedLineNumber, err = FindInNvimLocationHistory(expandUser(nvimLocationHistoryFilename), absFilename)
			found = err == nil
		}
		if !found {
			// Try to load the ViM location history, then
			recordedLineNumber, err = FindInVimLocationHistory(expandUser(vimLocationHistoryFilename), absFilename)
			found = err == nil
		}
		// Check if an existing line number was found
		if found {
			lineNumber = recordedLineNumber
			e.GoToLineNumber(lineNumber, c, status, true)
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

	// Make sure the location history isn't empty
	if e.locationHistory == nil {
		e.locationHistory = make(map[string]LineNumber, 1)
		e.locationHistory[absFilename] = lineNumber
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
		if readOnly {
			statusMessage += " (read only)"
		}
	} else {
		// If startup is slow (> 100 ms), display the startup time in the status bar
		if startupMilliseconds >= 100 {
			statusMessage = fmt.Sprintf("Loaded %s%s (%dms)", e.filename, warningMessage, startupMilliseconds)
		} else {
			statusMessage = fmt.Sprintf("Loaded %s%s", e.filename, warningMessage)
		}
		if readOnly {
			statusMessage += " (read only)"
		}
	}

	// Display the status message
	status.SetMessage(statusMessage)
	status.Show(c, e)

	// Redraw the cursor, if needed
	if e.redrawCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		previousX = x
		previousY = y
		vt100.SetXY(uint(x), uint(y))
		e.redrawCursor = false
	}

	var key string

	// This is the main loop for the editor
	for !e.quit {

		// Read the next key
		key = tty.String()

		switch key {
		case "c:17": // ctrl-q, quit
			e.quit = true
		case "c:23": // ctrl-w, format (or if in git mode, cycle interactive rebase keywords)
			undo.Snapshot(e)

			// Clear the search term
			e.ClearSearchTerm()

			// Cycle git rebase keywords
			if line := e.CurrentLine(); e.mode == modeGit && hasAnyPrefixWord(line, gitRebasePrefixes) {
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			if e.mode == modeMarkdown {
				e.ToggleCheckboxCurrentLine()
				break
			}

			// Not in git mode, format Go or C++ code with goimports or clang-format
			// Map from formatting command to a list of file extensions
			format := map[*exec.Cmd][]string{
				exec.Command("goimports", "-w", "--"):                                             {".go"},
				exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"): {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
				exec.Command("zig", "fmt"):                                                        {".zig"},
				exec.Command("v", "fmt"):                                                          {".v"},
				exec.Command("rustfmt"):                                                           {".rs"},
				exec.Command("brittany", "--write-mode=inplace"):                                  {".hs"},
				exec.Command("autopep8", "-i", "--max-line-length", "120"):                        {".py"},
				exec.Command("ocamlformat"):                                                       {".ml"},
				exec.Command("crystal", "tool", "format"):                                         {".cr"},
				exec.Command("ktlint", "-F"):                                                      {".kt", ".kts"},
				exec.Command("guessica"):                                                          {"PKGBUILD"},
				exec.Command("google-java-format", "-i"):                                          {".java"},
			}
		OUT:
			for cmd, extensions := range format {
				for _, ext := range extensions {
					if strings.HasSuffix(e.filename, ext) {
						if which(cmd.Path) == "" { // Does the formatting tool even exist?
							status.ClearAll(c)
							status.SetErrorMessage(cmd.Path + " is missing")
							status.Show(c, e)
							break OUT
						}
						utilityName := filepath.Base(cmd.Path)
						status.Clear(c)
						status.SetMessage("Calling " + utilityName)
						status.Show(c, e)
						// Use the temporary directory defined in TMPDIR, with fallback to /tmp
						tempdir := os.Getenv("TMPDIR")
						if tempdir == "" {
							tempdir = "/tmp"
						}
						if f, err := ioutil.TempFile(tempdir, "__o*"+ext); err == nil {
							// no error, everything is fine
							tempFilename := f.Name()

							// TODO: Implement e.SaveAs
							oldFilename := e.filename
							e.filename = tempFilename
							err := e.Save(c)
							e.filename = oldFilename

							if err == nil {
								// Add the filename of the temporary file to the command
								cmd.Args = append(cmd.Args, tempFilename)
								// Format the temporary file
								output, err := cmd.CombinedOutput()
								if err != nil {
									// Only grab the first error message
									errorMessage := strings.TrimSpace(string(output))
									if strings.Count(errorMessage, "\n") > 0 {
										errorMessage = strings.TrimSpace(strings.SplitN(errorMessage, "\n", 2)[0])
									}
									if errorMessage == "" {
										status.SetErrorMessage("Failed to format code")
									} else {
										status.SetErrorMessage("Failed to format code: " + errorMessage)
									}
									if strings.Count(errorMessage, ":") >= 3 {
										fields := strings.Split(errorMessage, ":")
										// Go To Y:X, if available
										var foundY int
										if y, err := strconv.Atoi(fields[1]); err == nil { // no error
											foundY = y - 1
											e.redraw = e.GoTo(LineIndex(foundY), c, status)
											foundX := -1
											if x, err := strconv.Atoi(fields[2]); err == nil { // no error
												foundX = x - 1
											}
											if foundX != -1 {
												tabs := strings.Count(e.Line(LineIndex(foundY)), "\t")
												e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
												e.Center(c)
											}
										}
										e.redrawCursor = true
									}
									status.Show(c, e)
									break OUT
								} else {
									if _, err := e.Load(c, tty, tempFilename); err != nil {
										status.ClearAll(c)
										status.SetMessage(err.Error())
										status.Show(c, e)
									}
									// Mark the data as changed, despite just having loaded a file
									e.changed = true
									e.redrawCursor = true
								}
								// Try to remove the temporary file regardless if "goimports -w" worked out or not
								_ = os.Remove(tempFilename)
							}
							// Try to close the file. f.Close() checks if f is nil before closing.
							_ = f.Close()
							e.redraw = true
							e.redrawCursor = true
						}
						break OUT
					}
				}
			}

			// Move the cursor if after the end of the line
			if e.AtOrAfterEndOfLine() {
				e.End(c)
			}
		case "c:6": // ctrl-f, search for a string
			e.SearchMode(c, status, tty, true)
		case "c:0": // ctrl-space, build source code to executable, convert to PDF or write to PNG, depending on the mode

			if e.mode == modeMarkdown {
				undo.Snapshot(e)
				e.ToggleCheckboxCurrentLine()
				break
			}

			// Save the current file, but only if it has changed
			if e.changed {
				if err := e.Save(c); err != nil {
					status.ClearAll(c)
					status.SetErrorMessage(err.Error())
					status.Show(c, e)
					break
				}
			}

			// Clear the current search term
			e.ClearSearchTerm()

			// Build or export the current file
			statusMessage, performedAction, compiled := e.BuildOrExport(c, status, e.filename)

			//logf("status message %s performed action %v compiled %v filename %s\n", statusMessage, performedAction, compiled, e.filename)

			// Could an action be performed for this file extension?
			if !performedAction {
				status.ClearAll(c)
				// Building this file extension is not implemented yet.
				// Just display the current time and word count.
				// TODO: status.ClearAll() should have cleared the status bar first, but this is not always true,
				//       which is why the message is hackily surrounded by spaces. Fix.
				statusMessage := fmt.Sprintf("    %d words, %s    ", e.WordCount(), time.Now().Format("15:04")) // HH:MM
				status.SetMessage(statusMessage)
				status.Show(c, e)
			} else if performedAction && !compiled {
				status.ClearAll(c)
				// Performed an action, but it did not work out
				if statusMessage != "" {
					status.SetErrorMessage(statusMessage)
				} else {
					// This should never happen, failed compilations should return a message
					status.SetErrorMessage("Compilation failed")
				}
				status.ShowNoTimeout(c, e)
			} else if performedAction && compiled {
				// Everything worked out
				if statusMessage != "" {
					// Got a status message (this may not be the case for build/export processes running in the background)
					// NOTE: Do not clear the status message first here!
					status.SetMessage(statusMessage)
					status.ShowNoTimeout(c, e)
				}
			}
		case "c:18": // ctrl-r, render to PDF, or if in git mode, cycle rebase keywords

			// Are we in git mode?
			if line := e.CurrentLine(); e.mode == modeGit && hasAnyPrefixWord(line, gitRebasePrefixes) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			// Save the current text to .pdf directly (without using pandoc)

			// Write to PDF in a goroutine
			go func() {

				pdfFilename := strings.Replace(filepath.Base(e.filename), ".", "_", -1) + ".pdf"

				// Show a status message while writing
				status.SetMessage("Saving PDF...")
				status.ShowNoTimeout(c, e)

				// TODO: Only overwrite if the previous PDF file was also rendered by "o".
				_ = os.Remove(pdfFilename)
				// Write the file
				if err := e.SavePDF(e.filename, pdfFilename); err != nil {
					statusMessage = err.Error()
				} else {
					statusMessage = "Saved " + pdfFilename
				}
				// Show a status message after writing
				status.ClearAll(c)
				status.SetMessage(statusMessage)
				status.Show(c, e)
			}()
		case "c:28": // ctrl-\, toggle comment for this block
			undo.Snapshot(e)
			e.ToggleCommentBlock(c)
			e.redraw = true
			e.redrawCursor = true
		case "c:15": // ctrl-o, launch the command menu
			status.ClearAll(c)
			lastCommandMenuIndex = e.CommandMenu(c, status, tty, undo, lastCommandMenuIndex)
		case "c:7": // ctrl-g, status mode
			statusMode = !statusMode
			if statusMode {
				status.ShowLineColWordCount(c, e, e.filename)
			} else {
				status.ClearAll(c)
			}
		case "←": // left arrow
			// movement if there is horizontal scrolling
			if e.pos.offsetX > 0 {
				if e.pos.sx > 0 {
					// Move one step left
					if e.TabToTheLeft() {
						e.pos.sx -= e.spacesPerTab
					} else {
						e.pos.sx--
					}
				} else {
					// Scroll one step left
					e.pos.offsetX--
					e.redraw = true
				}
				e.SaveX(true)
			} else if e.pos.sx > 0 {
				// no horizontal scrolling going on
				// Move one step left
				if e.TabToTheLeft() {
					e.pos.sx -= e.spacesPerTab
				} else {
					e.pos.sx--
				}
				e.SaveX(true)
			} else if e.DataY() > 0 {
				// no scrolling or movement to the left going on
				e.Up(c, status)
				e.End(c)
				//e.redraw = true
			} // else at the start of the document
			e.redrawCursor = true
			// Workaround for Konsole
			if e.pos.sx <= 2 {
				// Konsole prints "2H" here, but
				// no other terminal emulator does that
				e.redraw = true
				e.redrawCursor = false
			}
		case "→": // right arrow
			// If on the last line or before, go to the next character
			if e.DataY() < LineIndex(e.Len()) {
				e.Next(c)
			}
			if e.AfterScreenWidth(c) {
				e.pos.offsetX++
				e.redraw = true
				e.pos.sx--
				if e.pos.sx < 0 {
					e.pos.sx = 0
				}
				if e.AfterEndOfLine() {
					e.Down(c, status)
				}
			} else if e.AfterEndOfLine() {
				e.End(c)
			}
			e.SaveX(true)
			e.redrawCursor = true
		case "↑": // up arrow
			// Move the screen cursor

			// TODO: Stay at the same X offset when moving up in the document?
			if e.pos.offsetX > 0 {
				e.pos.offsetX = 0
			}

			if e.DataY() > 0 {
				// Move the position up in the current screen
				if e.UpEnd(c) != nil {
					// If below the top, scroll the contents up
					if e.DataY() > 0 {
						e.redraw = e.ScrollUp(c, status, 1)
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
			if e.AfterLineScreenContents() {
				e.End(c)
			}
			e.redrawCursor = true
		case "↓": // down arrow

			// TODO: Stay at the same X offset when moving down in the document?
			if e.pos.offsetX > 0 {
				e.pos.offsetX = 0

			}

			if e.DataY() < LineIndex(e.Len()) {
				// Move the position down in the current screen
				if e.DownEnd(c) != nil {
					// If at the bottom, don't move down, but scroll the contents
					// Output a helpful message
					if !e.AfterEndOfDocument() {
						e.redraw = e.ScrollDown(c, status, 1)
						e.pos.Up()
						e.DownEnd(c)
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineScreenContents() {
					e.End(c)
					// Then move one step to the left
					if strings.TrimSpace(e.CurrentLine()) != "" {
						e.Prev(c)
					}
				}
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.AfterLineScreenContents() {
				e.End(c)
			}
			e.redrawCursor = true
		case "c:14": // ctrl-n, scroll down or jump to next match, using the sticky search term
			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to next match
				wrap := true
				forward := true
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					if wrap {
						status.SetMessage(e.SearchTerm() + " not found")
					} else {
						status.SetMessage(e.SearchTerm() + " not found from here")
					}
					status.Show(c, e)
				}
			} else {
				// Scroll down
				e.redraw = e.ScrollDown(c, status, e.pos.scrollSpeed)
				// If e.redraw is false, the end of file is reached
				if !e.redraw {
					status.Clear(c)
					status.SetMessage("EOF")
					status.Show(c, e)
				}
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}
			}
		case "c:16": // ctrl-p, scroll up or jump to the previous match, using the sticky search term
			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to previous match
				wrap := true
				forward := false
				if err := e.GoToNextMatch(c, status, wrap, forward); err == errNoSearchMatch {
					status.Clear(c)
					if wrap {
						status.SetMessage(e.SearchTerm() + " not found")
					} else {
						status.SetMessage(e.SearchTerm() + " not found from here")
					}
					status.Show(c, e)
				}
			} else {
				e.redraw = e.ScrollUp(c, status, e.pos.scrollSpeed)
				e.redrawCursor = true
				if e.AfterLineScreenContents() {
					e.End(c)
				}
			}
			// Additional way to clear the sticky search term, like with Esc
		case "c:20": // ctrl-t, toggle syntax highlighting or use the next git interactive rebase keyword
			if line := e.CurrentLine(); e.mode == modeGit && hasAnyPrefixWord(line, gitRebasePrefixes) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}
			// Regular syntax highlight toggle
			e.ToggleSyntaxHighlight()
			if e.syntaxHighlight {
				e.bg = defaultEditorBackground
			} else {
				e.bg = vt100.BackgroundDefault
			}
			e.redraw = true
			e.redrawCursor = true
			// Now do a full reset/redraw
			fallthrough
		case "c:27": // esc, clear search term (but not the sticky search term), reset, clean and redraw
			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1
			// Do a full clear and redraw
			e.FullResetRedraw(c, status, true)
		case " ": // space
			undo.Snapshot(e)
			// Place a space
			e.InsertRune(c, ' ')
			e.redraw = true
			e.WriteRune(c)
			// Move to the next position
			e.Next(c)
		case "c:13": // return

			// Modify the paste double-keypress detection to allow for a manual return before pasting the rest
			if lastPasteY != -1 && previousKey != "c:13" {
				lastPasteY++
			}

			undo.Snapshot(e)

			e.TrimRight(e.DataY())
			lineContents := e.CurrentLine()
			trimmedLine := strings.TrimSpace(lineContents)
			if e.pos.AtStartOfLine() && !e.AtOrAfterLastLineOfDocument() {
				// Insert a new line a the current y position, then shift the rest down.
				e.InsertLineAbove()
				// Also move the cursor to the start, since it's now on a new blank line.
				e.pos.Down(c)
				e.Home()
			} else if len(trimmedLine) > 0 && e.AtOrBeforeStartOfTextLine() {
				x := e.pos.ScreenX()
				// Insert a new line a the current y position, then shift the rest down.
				e.InsertLineAbove()
				// Also move the cursor to the start, since it's now on a new blank line.
				e.pos.Down(c)
				e.pos.SetX(c, x)
			} else if e.AtOrAfterEndOfLine() && e.AtOrAfterLastLineOfDocument() {

				// Grab the leading whitespace from the current line, and indent depending on the end of trimmedLine
				const alsoDedent = false
				leadingWhitespace := e.smartIndentation(e.LeadingWhitespace(), trimmedLine, alsoDedent)

				e.InsertLineBelow()
				h := int(c.Height())
				if e.pos.sy >= (h - 1) {
					e.redraw = e.ScrollDown(c, status, 1)
					e.redrawCursor = true
				}
				e.pos.Down(c)
				e.Home()

				// Insert the same leading whitespace for the new line, while moving to the right
				e.InsertString(c, leadingWhitespace)

			} else if e.AtOrAfterEndOfLine() {

				// Grab the leading whitespace from the current line, and indent depending on the end of trimmedLine
				const alsoDedent = false
				leadingWhitespace := e.smartIndentation(e.LeadingWhitespace(), trimmedLine, alsoDedent)

				e.InsertLineBelow()
				e.pos.Down(c)
				e.Home()

				// Insert the same leading whitespace for the new line, while moving to the right
				e.InsertString(c, leadingWhitespace)
			} else {
				const alsoDedent = true

				// Split the current line in two
				if !e.SplitLine() {

					// Grab the leading whitespace from the current line, and indent or dedent depending on the end of trimmedLine
					leadingWhitespace := e.smartIndentation(e.LeadingWhitespace(), trimmedLine, alsoDedent)

					// Insert a line below, then move down and to the start of it
					e.InsertLineBelow()
					e.pos.Down(c)
					e.Home()

					// Insert the same leading whitespace for the new line, while moving to the right
					e.InsertString(c, leadingWhitespace)

				} else {
					leadingWhitespace := e.smartIndentation(e.LeadingWhitespace(), trimmedLine, alsoDedent)

					e.pos.Down(c)
					e.Home()

					// Insert the same leading whitespace for the new line, while moving to the right
					e.InsertString(c, leadingWhitespace)
				}
			}
			e.redraw = true
		case "c:8", "c:127": // ctrl-h or backspace
			// Just clear the search term, if there is an active search
			if len(e.SearchTerm()) > 0 {
				e.ClearSearchTerm()
				e.redraw = true
				e.redrawCursor = true
				break
			}
			undo.Snapshot(e)
			// Delete the character to the left
			if e.EmptyLine() {
				e.DeleteLine(e.DataY())
				e.pos.Up()
				e.TrimRight(e.DataY())
				e.End(c)
			} else if e.pos.AtStartOfLine() {
				if e.DataY() > 0 {
					e.pos.Up()
					e.End(c)
					e.TrimRight(e.DataY())
					e.Delete()
				}
			} else if (e.mode == modeShell || e.mode == modePython || e.mode == modeCMake) && e.AtStartOfTextLine() && len(e.LeadingWhitespace()) >= e.spacesPerTab {
				// Delete several spaces
				for i := 0; i < e.spacesPerTab; i++ {
					// Move back
					e.Prev(c)
					// Type a blank
					e.SetRune(' ')
					e.WriteRune(c)
					e.Delete()
				}
			} else {
				// Move back
				e.Prev(c)
				// Type a blank
				e.SetRune(' ')
				e.WriteRune(c)
				if !e.AtOrAfterEndOfLine() {
					// Delete the blank
					e.Delete()
				}
			}
			e.redrawCursor = true
			e.redraw = true
		case "c:9": // tab
			y := int(e.DataY())
			r := e.Rune()
			leftRune := e.LeftRune()
			ext := filepath.Ext(e.filename)
			if leftRune == '.' && !unicode.IsLetter(r) && e.mode != modeBlank {
				// Autocompletion
				undo.Snapshot(e)

				runes, ok := e.lines[y]
				if !ok {
					// This should never happen
					break
				}

				// Either find x or use the last index of the line
				x, err := e.DataX()
				if err != nil {
					x = len(runes) - 1
				}

				if x <= 0 {
					// This should never happen
					break
				}

				word := make([]rune, 0)
				// Loop from the current location (at the ".") and to the left in the current line
				for i := x - 1; i >= 0; i-- {
					r := e.lines[y][i]
					if r == '.' {
						continue
					}
					if !unicode.IsLetter(r) {
						break
					}
					// Gather the letters in reverse
					word = append([]rune{r}, word...)
				}

				if len(word) == 0 {
					// No word before ".", nothing to complete
					break
				}

				// Now the preceding word before the "." has been found

				// Grep all files in this directory with the same extension as the currently edited file
				// for what could follow the word and a "."
				suggestions := corpus(string(word), "*"+ext)

				// Choose a suggestion (tab cycles to the next suggestion)
				chosen := e.SuggestMode(c, status, tty, suggestions)
				e.redrawCursor = true
				e.redraw = true

				if chosen != "" {
					// Insert the chosen word
					e.InsertString(c, chosen)
					break
				}
			}

			// Enable auto indent if the extension is not "" and either:
			// * The mode is set to Go and the position is not at the very start of the line (empty or not)
			// * Syntax highlighting is enabled and the cursor is not at the start of the line (or before)
			trimmedLine := strings.TrimSpace(e.Line(LineIndex(y)))
			//emptyLine := len(trimmedLine) == 0
			//almostEmptyLine := len(trimmedLine) <= 1

			// Check if a line that is more than just a '{', '(', '[' or ':' ends with one of those
			endsWithSpecial := len(trimmedLine) > 1 && r == '{' || r == '(' || r == '[' || r == ':'

			// Smart indent if:
			// * the rune to the left is not a blank character or the line ends with {, (, [ or :
			// * and also if it the cursor is not to the very left
			// * and also if this is not a text file or a blank file
			if (!unicode.IsSpace(leftRune) || endsWithSpecial) && e.pos.sx > 0 && e.mode != modeBlank {
				lineAbove := 1
				if strings.TrimSpace(e.Line(LineIndex(y-lineAbove))) == "" {
					// The line above is empty, use the indendation before the line above that
					lineAbove--
				}
				indexAbove := LineIndex(y - lineAbove)
				// If we have a line (one or two lines above) as a reference point for the indentation
				if strings.TrimSpace(e.Line(indexAbove)) != "" {

					// Move the current indentation to the same as the line above
					undo.Snapshot(e)

					var (
						spaceAbove        = e.LeadingWhitespaceAt(indexAbove)
						strippedLineAbove = e.StripSingleLineComment(strings.TrimSpace(e.Line(indexAbove)))
						newLeadingSpace   string
						oneIndentation    string
					)

					switch e.mode {
					case modeShell, modePython, modeCMake:
						// If this is a shell script, use 2 spaces (or however many spaces are defined in e.spacesPerTab)
						oneIndentation = strings.Repeat(" ", e.spacesPerTab)
					default:
						// For anything else, use real tabs
						oneIndentation = "\t"
					}

					// Smart-ish indentation
					if !strings.HasPrefix(strippedLineAbove, "switch ") && (strings.HasPrefix(strippedLineAbove, "case ") ||
						strings.HasSuffix(strippedLineAbove, "{") || strings.HasSuffix(strippedLineAbove, "[") ||
						strings.HasSuffix(strippedLineAbove, "(") || strings.HasSuffix(strippedLineAbove, ":") ||
						strings.HasSuffix(strippedLineAbove, " \\") ||
						strings.HasPrefix(strippedLineAbove, "if ")) {
						// Use one more indentation than the line above
						newLeadingSpace = spaceAbove + oneIndentation
					} else if ((len(spaceAbove) - len(oneIndentation)) > 0) && strings.HasSuffix(trimmedLine, "}") {
						// Use one less indentation than the line above
						newLeadingSpace = spaceAbove[:len(spaceAbove)-len(oneIndentation)]
					} else {
						// Use the same indentation as the line above
						newLeadingSpace = spaceAbove
					}

					e.SetLine(LineIndex(y), newLeadingSpace+trimmedLine)
					if e.AtOrAfterEndOfLine() {
						e.End(c)
					}
					e.redrawCursor = true
					e.redraw = true

					// job done
					break

				}
			}

			undo.Snapshot(e)
			switch e.mode {
			case modeShell, modePython, modeCMake:
				for i := 0; i < spacesPerTab; i++ {
					e.InsertRune(c, ' ')
					// Write the spaces that represent the tab to the canvas
					e.WriteTab(c)
					// Move to the next position
					e.Next(c)
				}
			default:
				// Insert a tab character to the file
				e.InsertRune(c, '\t')
				// Write the spaces that represent the tab to the canvas
				e.WriteTab(c)
				// Move to the next position
				e.Next(c)
			}

			// Prepare to redraw
			e.redrawCursor = true
			e.redraw = true
		case "c:1", "c:25": // ctrl-a, home (or ctrl-y for scrolling up in the st terminal)

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑"
			// If at an empty line, go up one line
			if !justMovedUpOrDown && e.EmptyRightTrimmedLine() && e.SearchTerm() == "" {
				e.Up(c, status)
				//e.GoToStartOfTextLine()
				e.End(c)
			} else if x, err := e.DataX(); err == nil && x == 0 && !justMovedUpOrDown && e.SearchTerm() == "" {
				// If at the start of the line,
				// go to the end of the previous line
				e.Up(c, status)
				e.End(c)
			} else if e.AtStartOfTextLine() {
				// If at the start of the text, go to the start of the line
				e.Home()
			} else {
				// If none of the above, go to the start of the text
				e.GoToStartOfTextLine(c)
			}

			e.redrawCursor = true
			e.SaveX(true)
		case "c:5": // ctrl-e, end

			// Do not reset cut/copy/paste status

			// First check if we just moved to this line with the arrow keys, or just cut a line with ctrl-x
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑" || previousKey == "c:24"
			if e.AtEndOfDocument() {
				e.End(c)
				break
			}
			// If we didn't just move here, and are at the end of the line,
			// move down one line and to the end, if not,
			// just move to the end.
			if !justMovedUpOrDown && e.AfterEndOfLine() && e.SearchTerm() == "" {
				e.Down(c, status)
				e.Home()
			} else {
				e.End(c)
			}

			e.redrawCursor = true
			e.SaveX(true)
		case "c:4": // ctrl-d, delete
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.Delete()
				e.redraw = true
			}
			e.redrawCursor = true
		case "c:30": // ctrl-~, jump to matching parenthesis or curly bracket
			r := e.Rune()

			if e.AfterEndOfLine() {
				e.Prev(c)
				r = e.Rune()
			}

			// Find which opening and closing parenthesis/curly brackets to look for
			opening, closing := rune(0), rune(0)
			switch r {
			case '(', ')':
				opening = '('
				closing = ')'
			case '{', '}':
				opening = '{'
				closing = '}'
			case '[', ']':
				opening = '['
				closing = ']'
			}

			if opening == rune(0) {
				status.Clear(c)
				status.SetMessage("No matching (, ), [, ], { or }")
				status.Show(c, e)
				break
			}

			// Search either forwards or backwards to find a matching rune
			switch r {
			case '(', '{', '[':
				parcount := 0
				for !e.AtOrAfterEndOfDocument() {
					if r := e.Rune(); r == closing {
						if parcount == 1 {
							// FOUND, STOP
							break
						} else {
							parcount--
						}
					} else if r == opening {
						parcount++
					}
					e.Next(c)
				}
			case ')', '}', ']':
				parcount := 0
				for !e.AtStartOfDocument() {
					if r := e.Rune(); r == opening {
						if parcount == 1 {
							// FOUND, STOP
							break
						} else {
							parcount--
						}
					} else if r == closing {
						parcount++
					}
					e.Prev(c)
				}
			}

			e.redrawCursor = true
			e.redraw = true
		case "c:19": // ctrl-s, save
			// TODO: Call a Save method directly, not via a string
			e.UserCommand(c, status, "save")
		case "c:21", "c:26": // ctrl-u or ctrl-z, undo (ctrl-z may background the application)
			// Forget the cut, copy and paste line state
			lastCutY = -1
			lastPasteY = -1
			lastCopyY = -1

			// Try to restore the previous editor state in the undo buffer
			if err := undo.Restore(e); err == nil {
				//c.Draw()
				x := e.pos.ScreenX()
				y := e.pos.ScreenY()
				vt100.SetXY(uint(x), uint(y))
				e.redrawCursor = true
				e.redraw = true
			} else {
				status.SetMessage("Nothing more to undo")
				status.Show(c, e)
			}
		case "c:12": // ctrl-l, go to line number
			status.ClearAll(c)
			status.SetMessage("Go to line number:")
			status.ShowNoTimeout(c, e)
			lns := ""
			cancel := false
			doneCollectingDigits := false
			for !doneCollectingDigits {
				numkey := tty.String()
				switch numkey {
				case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9": // 0 .. 9
					lns += numkey // string('0' + (numkey - 48))
					status.SetMessage("Go to line number: " + lns)
					status.ShowNoTimeout(c, e)
				case "c:8", "c:127": // ctrl-h or backspace
					if len(lns) > 0 {
						lns = lns[:len(lns)-1]
						status.SetMessage("Go to line number: " + lns)
						status.ShowNoTimeout(c, e)
					}
				case "↑", "↓": // up arrow or down arrow
					fallthrough
				case "c:27", "c:17": // esc or ctrl-q
					cancel = true
					lns = ""
					fallthrough
				case "c:13": // return
					doneCollectingDigits = true
				}
			}
			if !cancel {
				e.ClearSearchTerm()
			}
			status.ClearAll(c)
			if lns == "" && !cancel {
				if e.DataY() > 0 {
					// If not at the top, go to the first line (by line number, not by index)
					e.redraw = e.GoToLineNumber(1, c, status, true)
				} else {
					// Go to the last line (by line number, not by index, e.Len() returns an index which is why there is no -1)
					e.redraw = e.GoToLineNumber(LineNumber(e.Len()), c, status, true)
				}
			} else {
				// Go to the specified line
				if ln, err := strconv.Atoi(lns); err == nil { // no error
					e.redraw = e.GoToLineNumber(LineNumber(ln), c, status, true)
				}
			}
			e.redrawCursor = true
		case "c:24": // ctrl-x, cut line
			y := e.DataY()
			line := e.Line(y)
			// Prepare to cut
			undo.Snapshot(e)
			// Now check if there is anything to cut
			if len(strings.TrimSpace(line)) == 0 {
				// Nothing to cut, just remove the current line
				e.Home()
				e.DeleteLine(e.DataY())
				// Check if ctrl-x was pressed once or twice, for this line
			} else if lastCutY != y { // Single line cut
				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1
				// Copy the line internally
				copyLines = []string{line}
				// Copy the line to the clipboard
				_ = clipboard.WriteAll(line)
				// Delete the line
				e.DeleteLine(y)
			} else { // Multi line cut (add to the clipboard, since it's the second press)
				lastCutY = y
				lastCopyY = -1
				lastPasteY = -1

				s := e.Block(y)
				lines := strings.Split(s, "\n")
				if len(lines) < 1 {
					// Need at least 1 line to be able to cut "the rest" after the first line has been cut
					break
				}
				copyLines = append(copyLines, lines...)
				s = strings.Join(copyLines, "\n")
				// Place the block of text in the clipboard
				_ = clipboard.WriteAll(s)
				// Delete the corresponding number of lines
				for range lines {
					e.DeleteLine(y)
				}
			}
			// Go to the end of the current line
			e.End(c)
			// No status message is needed for the cut operation, because it's visible that lines are cut
			e.redrawCursor = true
			e.redraw = true
		case "c:11": // ctrl-k, delete to end of line
			if e.Empty() {
				status.SetMessage("Empty file")
				status.Show(c, e)
				break
			}

			// Reset the cut/copy/paste double-keypress detection
			lastCopyY = -1
			lastPasteY = -1
			lastCutY = -1

			undo.Snapshot(e)
			e.DeleteRestOfLine()
			if e.EmptyRightTrimmedLine() {
				// Deleting the rest of the line cleared this line,
				// so just remove it.
				e.DeleteLine(e.DataY())
				// Then go to the end of the line, if needed
				if e.AfterEndOfLine() {
					e.End(c)
				}
			}
			// TODO: Is this one needed/useful?
			vt100.Do("Erase End of Line")
			e.redraw = true
			e.redrawCursor = true
		case "c:3": // ctrl-c, copy the stripped contents of the current line
			y := e.DataY()

			// Forget the cut and paste line state
			lastCutY = -1
			lastPasteY = -1

			// check if this operation is done on the same line as last time
			singleLineCopy := lastCopyY != y
			lastCopyY = y

			if singleLineCopy { // Single line copy
				// Pressed for the first time for this line number
				trimmed := strings.TrimSpace(e.Line(y))
				if trimmed != "" {
					// Copy the line to the internal clipboard
					copyLines = []string{trimmed}
					// Copy the line to the clipboard
					if err := clipboard.WriteAll(strings.Join(copyLines, "\n")); err != nil {
						// The copy did not work out, only copying internally
						status.SetMessage("Copied 1 line")
					} else {
						// The copy operation worked out
						status.SetMessage("Copied 1 line (clipboard)")
					}
					status.Show(c, e)
					// Go to the end of the line, for easy line duplication with ctrl-c, enter, ctrl-v,
					// but only if the copied line is shorter than the terminal width.
					if uint(len(trimmed)) < c.Width() {
						e.End(c)
					}
				}
			} else { // Multi line copy
				// Pressed multiple times for this line number, copy the block of text starting from this line
				s := e.Block(y)
				if s != "" {
					copyLines = strings.Split(s, "\n")
					// Prepare a status message
					plural := ""
					lineCount := strings.Count(s, "\n")
					if lineCount > 1 {
						plural = "s"
					}
					// Place the block of text in the clipboard
					err := clipboard.WriteAll(s)
					if err != nil {
						status.SetMessage(fmt.Sprintf("Copied %d line%s", lineCount, plural))
					} else {
						status.SetMessage(fmt.Sprintf("Copied %d line%s (clipboard)", lineCount, plural))
					}
					status.Show(c, e)
				}
			}
		case "c:22": // ctrl-v, paste

			// This may only work for the same user, and not with sudo/su

			// Try fetching the lines from the clipboard first
			s, err := clipboard.ReadAll()
			if err == nil { // no error
				// Fix nonbreaking spaces first
				s = strings.Replace(s, string([]byte{0xc2, 0xa0}), string([]byte{0x20}), -1)
				// Fix annoying tildes
				s = strings.Replace(s, string([]byte{0xcc, 0x88}), string([]byte{'~'}), -1)
				// And \r\n
				s = strings.Replace(s, string([]byte{'\r', '\n'}), string([]byte{'\n'}), -1)
				// Then \r
				s = strings.Replace(s, string([]byte{'\r'}), string([]byte{'\n'}), -1)

				// Note that control characters are not replaced, they are just not printed.

				// Split the text into lines and store it in "copyLines"
				copyLines = strings.Split(s, "\n")
			} else if firstPasteAction {
				firstPasteAction = false
				hasXclip := which("xclip") != ""
				hasWclip := which("wl-paste") != ""
				noBreak := false
				status.Clear(c)
				if !hasXclip && !hasWclip {
					status.SetErrorMessage("Either xclip or wl-paste (wl-clipboard) are missing!")
				} else if !hasXclip {
					status.SetErrorMessage("The xclip utility is missing!")
				} else if !hasWclip {
					status.SetErrorMessage("The wl-paste utility (from wl-clipboard) is missing!")
				} else {
					noBreak = true
				}
				if !noBreak {
					status.Show(c, e)
					break // Break instead of pasting from the internal buffer, but only the first time
				}
			} else {
				status.Clear(c)
				e.redrawCursor = true
			}

			// Now check if there is anything to paste
			if len(copyLines) == 0 {
				break
			}
			// Prepare to paste
			undo.Snapshot(e)
			y := e.DataY()

			// Forget the cut and copy line state
			lastCutY = -1
			lastCopyY = -1

			// Redraw after pasting
			e.redraw = true

			if lastPasteY != y { // Single line paste
				lastPasteY = y
				// Pressed for the first time for this line number, paste only one line

				// copyLines[0] is the line to be pasted, and it exists

				if e.EmptyRightTrimmedLine() {
					// If the line is empty, use the existing indentation before pasting
					e.SetLine(y, e.LeadingWhitespace()+strings.TrimSpace(copyLines[0]))
				} else {
					// If the line is not empty, insert the trimmed string
					e.InsertString(c, strings.TrimSpace(copyLines[0]))
				}
			} else { // Multi line paste (the rest of the lines)
				// Pressed the second time for this line number, paste multiple lines without trimming
				var (
					// copyLines contains the lines to be pasted, and they are > 1
					// the first line is skipped since that was already pasted when ctrl-v was pressed the first time
					lastIndex = len(copyLines[1:]) - 1

					// If the first line has been pasted, and return has been pressed, paste the rest of the lines differently
					skipFirstLineInsert bool
				)

				if previousKey != "c:13" {
					// Start by pasting (and overwriting) an untrimmed version of this line,
					// if the previous key was not return.
					e.SetLine(y, copyLines[0])
				} else if e.EmptyRightTrimmedLine() {
					skipFirstLineInsert = true
				}

				// The paste the rest of the lines, also untrimmed
				for i, line := range copyLines[1:] {
					if i == lastIndex && len(strings.TrimSpace(line)) == 0 {
						// If the last line is blank, skip it
						break
					}
					if skipFirstLineInsert {
						skipFirstLineInsert = false
					} else {
						e.InsertLineBelow()
						e.Down(c, nil) // no status message if the end of ducment is reached, there should always be a new line
					}
					e.InsertString(c, line)
				}
			}
			// Prepare to redraw the text
			e.redrawCursor = true
			e.redraw = true
		case "c:2": // ctrl-b, bookmark, unbookmark or jump to bookmark
			if bookmark == nil {
				// no bookmark, create a bookmark at the current line
				bookmark = e.pos.Copy()
				// TODO: Modify the statusbar implementation so that extra spaces are not needed here.
				status.SetMessage("  Bookmarked line " + e.LineNumber().String() + "  ")
			} else if bookmark.LineNumber() == e.LineNumber() {
				// bookmarking the same line twice: remove the bookmark
				status.SetMessage("Removed bookmark for line " + bookmark.LineNumber().String())
				bookmark = nil
			} else {
				// jumping to a bookmark
				undo.Snapshot(e)
				e.GoToPosition(c, status, *bookmark)
				// Do the redraw manually before showing the status message
				e.DrawLines(c, true, false)
				e.redraw = false
				// Show the status message.
				status.SetMessage("Jumped to bookmark at line " + e.LineNumber().String())
			}
			status.Show(c, e)
			e.redrawCursor = true
		case "c:10": // ctrl-j, join line
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				undo.Snapshot(e)
				nextLineIndex := e.DataY() + 1
				if e.EmptyRightTrimmedLineBelow() {
					// Just delete the line below if it's empty
					e.DeleteLine(nextLineIndex)
				} else {
					// Join the line below with this line. Also add a space in between.
					e.TrimLeft(nextLineIndex) // this is unproblematic, even at the end of the document
					e.End(c)
					e.InsertRune(c, ' ')
					e.WriteRune(c)
					e.Next(c)
					e.Delete()
				}
				e.redraw = true
			}
			e.redrawCursor = true

		case "/": // check if this is was the first pressed letter or not
			if firstLetterSinceStart == "" {
				// Set the first letter since start to something that will not trigger this branch any more.
				firstLetterSinceStart = "x"
				// If the first typed letter since starting this editor was '/', go straight to search mode.
				e.SearchMode(c, status, tty, true)
				// Case handled
				break
			}
			// This was not the first pressed letter, continue handling this key in the default case
			fallthrough
		default:
			//panic(fmt.Sprintf("PRESSED KEY: %v", []rune(key)))
			if len([]rune(key)) > 0 && unicode.IsLetter([]rune(key)[0]) { // letter

				undo.Snapshot(e)
				// Check for if a special "first letter" has been pressed, which triggers vi-like behavior
				if firstLetterSinceStart == "" {
					firstLetterSinceStart = key
					// If the first pressed key is "G" and this is not git mode, then invoke vi-compatible behavior and jump to the end
					if key == "G" && (e.mode != modeGit) {
						// Go to the end of the document
						e.redraw = e.GoToLineNumber(LineNumber(e.Len()), c, status, true)
						e.redrawCursor = true
						firstLetterSinceStart = "x"
						break
					}
				}
				// Type the letter that was pressed
				if len([]rune(key)) > 0 {
					// Insert a letter. This is what normally happens.
					e.InsertRune(c, []rune(key)[0])
					e.WriteRune(c)
					e.Next(c)
					e.redraw = true
				}
			} else if len([]rune(key)) > 0 && unicode.IsGraphic([]rune(key)[0]) { // any other key that can be drawn
				undo.Snapshot(e)

				// Place *something*
				r := []rune(key)[0]

				if r == 160 {
					// This is a nonbreaking space that may be inserted with altgr+space that is HORRIBLE.
					// Set r to a regular space instead.
					r = ' '
				}

				// "smart dedent"
				if r == '}' || r == ']' || r == ')' {
					lineContents := strings.TrimSpace(e.CurrentLine())
					whitespaceInFront := e.LeadingWhitespace()
					currentX, _ := e.DataX()
					nextLineContents := e.Line(e.DataY() + 1)
					foundCurlyBracketBelow := currentX-1 == strings.Index(nextLineContents, "}")
					foundSquareBracketBelow := currentX-1 == strings.Index(nextLineContents, "]")
					foundParenthesisBelow := currentX-1 == strings.Index(nextLineContents, ")")
					noDedent := foundCurlyBracketBelow || foundSquareBracketBelow || foundParenthesisBelow

					if e.pos.sx > 0 && len(lineContents) == 0 && len(whitespaceInFront) > 0 && !noDedent {
						// move one step left
						e.Prev(c)
						// trim trailing whitespace
						e.TrimRight(e.DataY())
					}
				}

				e.InsertRune(c, r)
				e.WriteRune(c)
				if len(string(r)) > 0 {
					// Move to the next position
					e.Next(c)
				}
				e.redrawCursor = true
				e.redraw = true
			}
		}
		previousKey = key
		// Clear status, if needed
		if statusMode && e.redrawCursor {
			status.ClearAll(c)
		}
		// Redraw, if needed
		if e.redraw {
			// Draw the editor lines on the canvas, respecting the offset
			e.DrawLines(c, true, false)
			e.redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		// Drawing status messages should come after redrawing, but before cursor positioning
		if statusMode {
			status.ShowLineColWordCount(c, e, e.filename)
		} else if status.IsError() {
			// Show the status message
			status.Show(c, e)
		}
		// Position the cursor
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		if e.redrawCursor || x != previousX || y != previousY {
			vt100.SetXY(uint(x), uint(y))
			e.redrawCursor = false
		}
		previousX = x
		previousY = y
		// The first letter was not O or /, which invokes special vi-compatible behavior
		firstLetterSinceStart = "x"

	} // end of main loop

	// Save the current location in the location history and write it to file
	e.SaveLocation(absFilename, e.locationHistory)

	// Clear all status bar messages
	status.ClearAll(c)

	// Quit everything that has to do with the terminal
	if e.clearOnQuit {
		vt100.Clear()
		vt100.Close()
	} else {
		c.Draw()
		fmt.Println()
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		runtime.GC()    // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
