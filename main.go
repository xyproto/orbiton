package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

const version = "o 2.25.0"

var (
	rebasePrefixes   = []string{"p", "pick", "r", "reword", "d", "drop", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "l", "label", "t", "reset", "m", "merge"}
	checkboxPrefixes = []string{"- [ ]", "- [x]", "- [X]", "* [ ]", "* [x]", "* [X]"}
)

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.LightGreen // for when syntax highlighting is not in use
		defaultEditorBackground       = vt100.BackgroundDefault
		defaultStatusForeground       = vt100.White
		defaultStatusBackground       = vt100.BackgroundBlack
		defaultStatusErrorForeground  = vt100.LightRed
		defaultStatusErrorBackground  = vt100.BackgroundDefault
		defaultEditorSearchHighlight  = vt100.LightMagenta
		defaultEditorMultilineComment = vt100.Gray
		defaultEditorMultilineString  = vt100.Magenta
		defaultEditorHighlightTheme   = syntax.TextConfig{
			String:        "lightyellow",
			Keyword:       "lightred",
			Comment:       "gray",
			Type:          "lightblue",
			Literal:       "lightgreen",
			Punctuation:   "lightblue",
			Plaintext:     "lightgreen",
			Tag:           "lightgreen",
			TextTag:       "lightgreen",
			TextAttrName:  "lightgreen",
			TextAttrValue: "lightgreen",
			Decimal:       "white",
			AndOr:         "lightyellow",
			Star:          "lightyellow",
			Class:         "lightred",
			Private:       "darkred",
			Protected:     "darkyellow",
			Public:        "darkgreen",
			Whitespace:    "",
		}

		versionFlag = flag.Bool("version", false, "show version information")
		helpFlag    = flag.Bool("help", false, "show simple help")

		statusDuration = 2700 * time.Millisecond

		copyLines  []string  // for the cut/copy/paste functionality
		bookmark   *Position // for the bookmark/jump functionality
		statusMode bool      // if information should be shown at the bottom

		firstLetterSinceStart string

		locationHistory map[string]int // remember where we were in each absolute filename

		clearOnQuit bool // clear the terminal when quitting, or not

		spacesPerTab = 4 // default spaces per tab

		mode Mode // an "enum"/int signalling if this file should be in git mode, markdown mode etc

		lastCopyY  = -1 // used for keeping track if ctrl-c is pressed twice on the same line
		lastPasteY = -1 // used for keeping track if ctrl-v is pressed twice on the same line
		lastCutY   = -1 // used for keeping track if ctrl-x is pressed twice on the same line
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
ctrl-w     to format the current file with "go fmt" or "clang-format"
           if in markdown mode: toggle checkboxes
		   if in git mode: cycle git interactive rebase keywords
ctrl-a     go to start of line, then start of text and then the previous line
ctrl-e     go to end of line and then the next line
ctrl-p     to scroll up 10 lines
ctrl-n     to scroll down 10 lines or go to the next match if a search is active
ctrl-k     to delete characters to the end of the line, then delete the line
ctrl-g     to toggle filename/line/column/unicode/word count status display
ctrl-d     to delete a single character
ctrl-t     to toggle syntax highlighting
ctrl-o     to toggle text or draw mode
ctrl-c     to copy the current line, press twice to copy the current block
ctrl-v     to paste one line, press twice to paste the rest
ctrl-x     to cut the current line, press twice to cut the current block
ctrl-b     to bookmark the current line, press again to unbookmark
ctrl-j     to join lines (or jump to the bookmark, if set)
ctrl-u     to undo
ctrl-l     to jump to a specific line
ctrl-f     to search for a string
esc        to redraw the screen and clear the last search
ctrl-space to build Go, C++, word wrap
ctrl-r     to render the current text to a PDF document
ctrl-\     to toggle single-line comments for a block of code
ctrl-~     to save and quit + clear the terminal

Set NO_COLOR=1 to disable colors.

`)
		return
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

	baseFilename := filepath.Base(filename)

	// Check if we should be in a particular mode for a particular type of file
	// TODO: Extract the filename extension first, then perform the switch, to skip the HasSuffix calls
	switch {
	case baseFilename == "COMMIT_EDITMSG" ||
		baseFilename == "MERGE_MSG" ||
		(strings.HasPrefix(baseFilename, "git-") &&
			!strings.Contains(baseFilename, ".") &&
			strings.Count(baseFilename, "-") >= 2):
		// Git mode
		mode = modeGit
	case strings.HasSuffix(baseFilename, ".md"):
		// Markdown mode
		mode = modeMarkdown
	case strings.HasSuffix(baseFilename, ".adoc") || strings.HasSuffix(baseFilename, ".rst") || strings.HasSuffix(baseFilename, ".scdoc") || strings.HasSuffix(baseFilename, ".scd"):
		// Markdown-like syntax highlighting
		// TODO: Introduce a separate mode for these.
		mode = modeMarkdown
	case strings.HasSuffix(baseFilename, ".sh") || strings.HasSuffix(baseFilename, ".bash") || baseFilename == "PKGBUILD":
		mode = modeShell
	case strings.HasSuffix(baseFilename, ".yml") || strings.HasSuffix(baseFilename, ".toml"):
		mode = modeYml
	case baseFilename == "Makefile" || baseFilename == "makefile" || baseFilename == "GNUmakefile":
		mode = modeMakefile
	case strings.HasSuffix(baseFilename, ".asm") || strings.HasSuffix(baseFilename, ".S") || strings.HasSuffix(baseFilename, ".inc"):
		mode = modeAssembly
	case strings.HasSuffix(baseFilename, ".go"):
		mode = modeGo
	case strings.HasSuffix(baseFilename, ".hs"):
		mode = modeHaskell
	}

	// Check if we should enable syntax highlighting by default
	syntaxHighlight := mode == modeGit || baseFilename == "config" || baseFilename == "PKGBUILD" || baseFilename == "BUILD" || baseFilename == "WORKSPACE" || strings.Contains(baseFilename, ".") || strings.HasSuffix(baseFilename, "file") // Makefile, Dockerfile, Jenkinsfile, Vagrantfile

	// Per-language adjustments to highlighting of keywords
	if mode != modeGo {
		delete(syntax.Keywords, "build")
		delete(syntax.Keywords, "package")
	}

	// Additional per-mode considerations
	if mode == modeGit {
		clearOnQuit = true
	} else if mode == modeShell || mode == modeYml {
		spacesPerTab = 2
	} else if mode == modeMakefile {
		spacesPerTab = 1
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

	// scroll 10 lines at a time, no word wrap
	e := NewEditor(spacesPerTab,
		syntaxHighlight,
		true,
		10,
		defaultEditorForeground,
		defaultEditorBackground,
		defaultEditorSearchHighlight,
		defaultEditorMultilineComment,
		defaultEditorMultilineString,
		defaultEditorHighlightTheme,
		mode)

	// For non-highlighted files, adjust the word wrap
	if !syntaxHighlight {
		// Adjust the word wrap if the terminal is too narrow
		w := int(c.Width())
		if w < e.wordWrapAt {
			e.wordWrapAt = w
		}
	}

	// Use a theme for light backgrounds if XTERM_VERSION is set,
	// because $COLORFGBG is "15;0" even though the background is white.
	envTheme := os.Getenv("O_THEME")
	if envTheme == "light" || (envTheme != "dark" && os.Getenv("XTERM_VERSION") != "") {
		e.setLightTheme()
	}

	e.respectNoColorEnvironmentVariable()

	status := NewStatusBar(defaultStatusForeground, defaultStatusBackground, defaultStatusErrorForeground, defaultStatusErrorBackground, e, statusDuration)
	status.respectNoColorEnvironmentVariable()

	// Load a file, or a prepare an empty version of the file (without saving it until the user saves it)
	var (
		statusMessage  string
		warningMessage string
	)

	// We wish to redraw the canvas and reposition the cursor
	e.redraw = true
	e.redrawCursor = true

	// Use os.Stat to check if the file exists, and load the file if it does
	if fileInfo, err := os.Stat(filename); err == nil {

		// TODO: Enter file-rename mode when opening a directory?
		// Check if this is a directory
		if fileInfo.IsDir() {
			quitError(tty, errors.New(filename+" is a directory"))
		}

		warningMessage, err = e.Load(c, tty, filename)
		if err != nil {
			quitError(tty, err)
		}

		if !e.Empty() {
			statusMessage = "Loaded " + filename + warningMessage
		} else {
			statusMessage = "Loaded empty file: " + filename + warningMessage
		}

		// Test write, to check if the file can be written or not
		testfile, err := os.OpenFile(filename, os.O_WRONLY, 0664)
		if err != nil {
			// can not open the file for writing
			statusMessage += " (read only)"
			// set the color to red when in read-only mode
			e.fg = vt100.Red
			// disable syntax highlighting, to make it clear that the text is red
			e.syntaxHighlight = false
			// do a full reset and redraw
			c = e.FullResetRedraw(c, status)
			// draw the editor lines again
			e.DrawLines(c, false, true)
			e.redraw = false
		}
		testfile.Close()
	} else {
		newMode, err := e.PrepareEmpty(c, tty, filename)
		if err != nil {
			quitError(tty, err)
		}

		statusMessage = "New " + filename

		if newMode != modeBlank {
			mode, e.mode = newMode, newMode
		}

		// Test save, to check if the file can be created and written, or not
		if err := e.Save(&filename, true); err != nil {
			// Check if the new file can be saved before the user starts working on the file.
			quitError(tty, err)
		} else {
			// Creating a new empty file worked out fine, don't save it until the user saves it
			if os.Remove(filename) != nil {
				// This should never happen
				quitError(tty, errors.New("could not remove an empty file that was just created: "+filename))
			}
		}
	}

	// The editing mode is decided at this point

	// If we're editing a git commit message, add a newline and enable word-wrap at 80
	if mode == modeGit {
		e.gitColor = vt100.LightGreen
		status.fg = vt100.LightBlue
		status.bg = vt100.BackgroundDefault
		if baseFilename == "MERGE_MSG" {
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
		c = e.FullResetRedraw(c, status)
	}

	// Undo buffer with room for 8192 actions
	undo := NewUndo(8192)

	// Resize handler
	SetUpResizeHandler(c, e, status, tty)

	tty.SetTimeout(2 * time.Millisecond)

	previousX := 1
	previousY := 1

	// Find the absolute path to this filename
	absFilename, err := filepath.Abs(filename)
	if err != nil {
		absFilename = filename
	}

	// Load the location history, if available
	locationHistory = LoadLocationHistory(expandUser(locationHistoryFilename))

	// Check if a line number was given on the command line
	if lineNumber > 0 {
		e.GoToLineNumber(lineNumber, c, status, false)
		e.redraw = true
		e.redrawCursor = true
	} else if recordedLineNumber, ok := locationHistory[absFilename]; ok && mode != modeGit {
		// If this filename exists in the location history, jump there
		lineNumber = recordedLineNumber
		e.GoToLineNumber(lineNumber, c, status, true)
		e.redraw = true
		e.redrawCursor = true
	} else {
		// Draw editor lines from line 0 to h onto the canvas at 0,0
		e.DrawLines(c, false, false)
		e.redraw = false
	}

	if e.redraw {
		e.Center(c)
		e.DrawLines(c, true, false)
		e.redraw = false
	}

	status.SetMessage(statusMessage)
	status.Show(c, e)

	if e.redrawCursor {
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		previousX = x
		previousY = y
		vt100.SetXY(uint(x), uint(y))
		e.redrawCursor = false
	}

	var (
		quit        bool
		previousKey string

		// Used for vi-compatible "O"-mode at start
		dropO bool
	)

	for !quit {
		key := tty.String()
		switch key {
		case "c:17": // ctrl-q, quit
			quit = true
		case "c:23": // ctrl-w, format (or if in git mode, cycle interactive rebase keywords)
			undo.Snapshot(e)

			// Cycle git rebase keywords
			if line := e.CurrentLine(); mode == modeGit && hasAnyPrefixWord(line, rebasePrefixes) {
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			// Toggle Markdown checkboxes
			if line := e.CurrentLine(); mode == modeMarkdown && hasAnyPrefixWord(strings.TrimSpace(line), checkboxPrefixes) {
				if strings.Contains(line, "[ ]") {
					e.SetLine(e.DataY(), strings.Replace(line, "[ ]", "[x]", 1))
					e.redraw = true
				} else if strings.Contains(line, "[x]") {
					e.SetLine(e.DataY(), strings.Replace(line, "[x]", "[ ]", 1))
					e.redraw = true
				} else if strings.Contains(line, "[X]") {
					e.SetLine(e.DataY(), strings.Replace(line, "[X]", "[ ]", 1))
					e.redraw = true
				}
				e.redrawCursor = e.redraw
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
				exec.Command("brittany", "--write-mode=inplace", "--"):                            {".hs"},
			}
			formatted := false
		OUT:
			for cmd, extensions := range format {
				for _, ext := range extensions {
					if strings.HasSuffix(filename, ext) {
						// Use the temporary directory defined in TMPDIR, with fallback to /tmp
						tempdir := os.Getenv("TMPDIR")
						if tempdir == "" {
							tempdir = "/tmp"
						}
						if f, err := ioutil.TempFile(tempdir, "__o*"+ext); err == nil {
							// no error, everything is fine
							tempFilename := f.Name()
							err := e.Save(&tempFilename, true)
							if err == nil {
								// Format the temporary file
								cmd.Args = append(cmd.Args, tempFilename)
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
											e.redraw = e.GoTo(foundY, c, status)
											foundX := -1
											if x, err := strconv.Atoi(fields[2]); err == nil { // no error
												foundX = x - 1
											}
											if foundX != -1 {
												tabs := strings.Count(e.Line(foundY), "\t")
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
									formatted = true
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
			if mode != modeGit && !formatted {
				// Check if at least one line is longer than the word wrap limit first
				// word wrap at the current width - 5, with an allowed overshoot of 5 runes
				if e.WrapAllLinesAt(e.wordWrapAt-5, 5) {
					e.redraw = true
					e.redrawCursor = true
				}
			}
			// Move the cursor if after the end of the line
			if e.AtOrAfterEndOfLine() {
				e.End()
			}
		case "c:6": // ctrl-f, search for a string
			e.SearchMode(c, status, tty, true)
		case "c:0": // ctrl-space, build source code to executable, word wrap, convert to PDF or write to PNG, depending on the mode
			if strings.HasSuffix(baseFilename, ".scd") || strings.HasSuffix(baseFilename, ".scdoc") {
				scdoc := exec.Command("/usr/bin/scdoc")

				// Place the current contents in a buffer, and feed it to stdin to the command
				var buf bytes.Buffer
				buf.WriteString(e.String())
				scdoc.Stdin = &buf

				// Create a new file and use it as stdout
				manpageFile, err := os.Create("out.1")
				if err != nil {
					statusMessage = err.Error()
					status.ClearAll(c)
					status.SetMessage(statusMessage)
					status.Show(c, e)
					break // from case
				}
				scdoc.Stdout = manpageFile

				var errBuf bytes.Buffer
				scdoc.Stderr = &errBuf

				// Run scdoc
				if err := scdoc.Run(); err != nil {
					statusMessage = strings.TrimSpace(errBuf.String())
					status.ClearAll(c)
					status.SetMessage(statusMessage)
					status.Show(c, e)
					break // from case
				}

				statusMessage = "Saved out.1"
				status.ClearAll(c)
				status.SetMessage(statusMessage)
				status.Show(c, e)
				break // from case

			} else if strings.HasSuffix(baseFilename, ".adoc") {
				asciidoctor := exec.Command("/usr/bin/asciidoctor", "-b", "manpage", "-o", "out.1", filename)
				if err := asciidoctor.Run(); err != nil {
					statusMessage = err.Error()
					status.ClearAll(c)
					status.SetMessage(statusMessage)
					status.Show(c, e)
					break // from case
				}
				statusMessage = "Saved out.1"
				status.ClearAll(c)
				status.SetMessage(statusMessage)
				status.Show(c, e)
				break // from case
				// Is this a Markdown file? Save to PDF, either by using pandoc or by writing the text file directly
			} else if pandocPath := which("pandoc"); mode == modeMarkdown && strings.HasSuffix(baseFilename, ".md") && pandocPath != "" {

				go func() {
					pdfFilename := strings.Replace(baseFilename, ".", "_", -1) + ".pdf"

					statusMessage := "Converting to PDF using Pandoc..."
					status.SetMessage(statusMessage)
					status.ShowNoTimeout(c, e)

					tmpfn := "___o___.md"

					if exists(tmpfn) {
						statusMessage = tmpfn + " already exists, please remove it"
						status.ClearAll(c)
						status.SetMessage(statusMessage)
						status.Show(c, e)
						return // from goroutine
					}

					err := e.Save(&tmpfn, !e.DrawMode())
					if err != nil {
						statusMessage = err.Error()
						status.ClearAll(c)
						status.SetMessage(statusMessage)
						status.Show(c, e)
						return // from goroutine
					}

					pandoc := exec.Command(pandocPath, "-N", "--toc", "-V", "geometry:a4paper", "-o", pdfFilename, tmpfn)
					if err = pandoc.Run(); err != nil {
						_ = os.Remove(tmpfn) // Try removing the temporary filename if pandoc fails
						statusMessage = err.Error()
						status.ClearAll(c)
						status.SetMessage(statusMessage)
						status.Show(c, e)
						return // from goroutine
					}

					if err = os.Remove(tmpfn); err != nil {
						statusMessage = err.Error()
						status.ClearAll(c)
						status.SetMessage(statusMessage)
						status.Show(c, e)
						return // from goroutine
					}

					statusMessage = "Saved " + pdfFilename
					status.ClearAll(c)
					status.SetMessage(statusMessage)
					status.Show(c, e)
				}()
				break // from case
			}

			// Map from formatting command to a list of file extensions
			build := map[*exec.Cmd][]string{
				exec.Command("go", "build"):               {".go"},
				exec.Command("cxx"):                       {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
				exec.Command("zig", "build"):              {".zig"},
				exec.Command("v", filename):               {".v"},
				exec.Command("cargo", "build"):            {".rs"},
				exec.Command("ghc", "-dynamic", filename): {".hs"},
			}
			var foundExtensionToBuild bool
		OUT2:
			for cmd, extensions := range build {
				for _, ext := range extensions {
					if strings.HasSuffix(filename, ext) {
						foundExtensionToBuild = true
						status.ClearAll(c)
						status.SetMessage("Building")
						status.ShowNoTimeout(c, e)

						// Save the current line location to file, for later
						e.SaveLocation(absFilename, locationHistory)

						// Special per-language considerations
						if ext == ".rs" && (!exists("Cargo.toml") && !exists("../Cargo.toml")) {
							// Use rustc instead of cargo if Cargo.toml is missing and the extension is .rs
							cmd = exec.Command("rustc", filename)
						} else if (ext == ".cc" || ext == ".h") && exists("BUILD.bazel") {
							// Google-style C++ + Bazel projects
							cmd = exec.Command("bazel", "build")
						} else if ext == ".zig" && !exists("build.zig") {
							// Just build the current file
							cmd = exec.Command("zig", "build-exe", "-lc", filename)
						}

						output, err := cmd.CombinedOutput()
						if err != nil || bytes.Contains(output, []byte("error:")) {
							// Clear all existing status messages and status message clearing goroutines
							status.ClearAll(c)
							// Find the first error message
							lines := strings.Split(string(output), "\n")
							errorMessage := "Build error"
							for _, line := range lines {
								if strings.Contains(line, "error:") {
									parts := strings.SplitN(line, "error:", 2)
									errorMessage = parts[1]
									break
								}
							}
							status.SetErrorMessage(errorMessage)
							status.Show(c, e)
							for i, line := range lines {
								// Jump to the error location, for C++ and Go
								if strings.Count(line, ":") >= 3 {
									fields := strings.SplitN(line, ":", 4)

									// Go to Y:X, if available
									var foundY int
									if y, err := strconv.Atoi(fields[1]); err == nil { // no error
										foundY = y - 1
										e.redraw = e.GoTo(foundY, c, status)
										foundX := -1
										if x, err := strconv.Atoi(fields[2]); err == nil { // no error
											foundX = x - 1
										}
										if foundX != -1 {
											tabs := strings.Count(e.Line(foundY), "\t")
											e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
											e.Center(c)
											// Use the error message as the status message
											if len(fields) >= 4 {
												status.ClearAll(c)
												status.SetErrorMessage(strings.Join(fields[3:], " "))
												status.Show(c, e)
												break OUT2
											}
										}
									}
									e.redrawCursor = true
									break
								} else if (i-1) > 0 && (i-1) < len(lines) {
									if msgLine := lines[i-1]; strings.Contains(line, " --> ") && strings.Count(line, ":") == 2 && strings.Count(msgLine, ":") >= 1 {
										// Jump to the error location, for Rust
										errorFields := strings.SplitN(msgLine, ":", 2)                  // Already checked for 2 colons
										errorMessage := strings.TrimSpace(errorFields[1])               // There will always be 3 elements in errorFields, so [1] is fine
										locationFields := strings.SplitN(line, ":", 3)                  // Already checked for 2 colons in line
										filenameFields := strings.SplitN(locationFields[0], " --> ", 2) // [0] is fine, already checked for " ---> "
										errorFilename := strings.TrimSpace(filenameFields[1])           // [1] is fine
										if filename != errorFilename {
											status.ClearAll(c)
											status.SetMessage("Error in " + errorFilename + ": " + errorMessage)
											status.Show(c, e)
											break OUT2
										}
										errorY := locationFields[1]
										errorX := locationFields[2]

										// Go to Y:X, if available
										var foundY int
										if y, err := strconv.Atoi(errorY); err == nil { // no error
											foundY = y - 1
											e.redraw = e.GoTo(foundY, c, status)
											foundX := -1
											if x, err := strconv.Atoi(errorX); err == nil { // no error
												foundX = x - 1
											}
											if foundX != -1 {
												tabs := strings.Count(e.Line(foundY), "\t")
												e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
												e.Center(c)
												// Use the error message as the status message
												if errorMessage != "" {
													status.SetErrorMessage(errorMessage)
													status.Show(c, e)
													break OUT2
												}
											}
										}
										e.redrawCursor = true
										break
									}
								}
							}
						} else {
							status.ClearAll(c)
							status.SetMessage("Success")
							status.Show(c, e)
						}
						break OUT2
					}
				}
			}
			if !foundExtensionToBuild {
				// Building this file extension is not implemented yet.
				status.ClearAll(c)
				// Just display the current time and word count.
				statusMessage := fmt.Sprintf("%d words, %s", e.WordCount(), time.Now().Format("15:04")) // HH:MM
				status.SetMessage(statusMessage)
				status.Show(c, e)
			}
		case "c:18": // ctrl-r, render to PDF, or if in git mode, cycle rebase keywords

			// Are we in git mode?
			if line := e.CurrentLine(); mode == modeGit && hasAnyPrefixWord(line, rebasePrefixes) {
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

				pdfFilename := strings.Replace(baseFilename, ".", "_", -1) + ".pdf"

				// Show a status message while writing
				statusMessage := "Saving PDF..."
				status.SetMessage(statusMessage)
				status.ShowNoTimeout(c, e)

				// TODO: Only overwrite if the previous PDF file was also rendered by "o".
				_ = os.Remove(pdfFilename)
				// Write the file
				if err := e.SavePDF(filename, pdfFilename); err != nil {
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
		case "c:15": // ctrl-o, toggle ASCII draw mode
			e.ToggleDrawMode()
			statusMessage := "Text mode"
			if e.DrawMode() {
				statusMessage = "Draw mode"
			}
			status.Clear(c)
			status.SetMessage(statusMessage)
			status.Show(c, e)
		case "c:7": // ctrl-g, status mode
			statusMode = !statusMode
			if statusMode {
				status.ShowLineColWordCount(c, e, filename)
			} else {
				status.ClearAll(c)
			}
		case "←": // left arrow
			if !e.DrawMode() {
				e.Prev(c)
				if e.AfterLineScreenContents() {
					e.End()
				}
				e.SaveX(true)
			} else {
				// Draw mode
				e.pos.Left()
			}
			e.redrawCursor = true
		case "→": // right arrow
			if !e.DrawMode() {
				if e.DataY() < e.Len() {
					e.Next(c)
				}
				if e.AfterLineScreenContents() {
					e.End()
				}
				e.SaveX(true)
			} else {
				// Draw mode
				e.pos.Right(c)
			}
			e.redrawCursor = true
		case "↑": // up arrow
			// Disregard the curren copy/cut/paste state
			lastCutY = -1
			lastCopyY = -1
			lastPasteY = -1
			// Move the screen cursor
			if !e.DrawMode() {
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
						e.End()
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineScreenContents() {
					e.End()
				}
			} else {
				e.pos.Up()
			}
			e.redrawCursor = true
		case "↓": // down arrow
			// Disregard the curren copy/cut/paste state
			lastCutY = -1
			lastCopyY = -1
			lastPasteY = -1
			if !e.DrawMode() {
				if e.DataY() < e.Len() {
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
						e.End()
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineScreenContents() {
					e.End()
				}
			} else {
				e.pos.Down(c)
			}
			e.redrawCursor = true
		case "c:14": // ctrl-n, scroll down or jump to next match, using the sticky search term
			e.UseStickySearchTerm()
			if e.SearchTerm() != "" {
				// Go to next match
				e.GoToNextMatch(c, status)
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
				if !e.DrawMode() && e.AfterLineScreenContents() {
					e.End()
				}
			}
		case "c:16": // ctrl-p, scroll up, and clear the sticky search term
			e.redraw = e.ScrollUp(c, status, e.pos.scrollSpeed)
			e.redrawCursor = true
			if !e.DrawMode() && e.AfterLineScreenContents() {
				e.End()
			}
			// Additional way to clear the sticky search term, like with Esc
		case "c:20": // ctrl-t, toggle syntax highlighting or use the next git interactive rebase keyword
			if line := e.CurrentLine(); mode == modeGit && hasAnyPrefixWord(line, []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			} else {
				e.ToggleHighlight()
				if e.syntaxHighlight {
					e.bg = defaultEditorBackground
				} else {
					e.bg = vt100.BackgroundDefault
				}
			}
			// Now do a full reset/redraw
			fallthrough
		case "c:27": // esc, clear search term (but not the sticky search term), reset, clean and redraw
			c = e.FullResetRedraw(c, status)
		case " ": // space
			undo.Snapshot(e)
			// Place a space
			if !e.DrawMode() {
				e.InsertRune(c, ' ')
				e.redraw = true
			} else {
				e.SetRune(' ')
			}
			e.WriteRune(c)
			if e.DrawMode() {
				e.redraw = true
			} else {
				// Move to the next position
				e.Next(c)
			}
		case "c:13": // return
			undo.Snapshot(e)
			// if the current line is empty, insert a blank line
			if !e.DrawMode() {
				e.TrimRight(e.DataY())
				lineContents := e.CurrentLine()
				if e.pos.AtStartOfLine() {
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineAbove()
					// Also move the cursor to the start, since it's now on a new blank line.
					e.pos.Down(c)
					e.Home()
				} else if e.AtOrBeforeStartOfTextLine() {
					x := e.pos.ScreenX()
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineAbove()
					// Also move the cursor to the start, since it's now on a new blank line.
					e.pos.Down(c)
					e.pos.SetX(x)
				} else if e.AtOrAfterEndOfLine() && e.AtLastLineOfDocument() {
					leadingWhitespace := e.LeadingWhitespace()
					if len(lineContents) > 0 && (strings.HasSuffix(lineContents, "(") || strings.HasSuffix(lineContents, "{") || strings.HasSuffix(lineContents, "[")) {
						// "smart indentation"
						leadingWhitespace += "\t"
					}
					e.InsertLineBelow()
					h := int(c.Height())
					if e.pos.sy >= (h - 1) {
						e.ScrollDown(c, status, 1)
						e.redrawCursor = true
					}
					e.pos.Down(c)
					e.Home()
					// Insert the same leading whitespace for the new line, while moving to the right
					for _, r := range leadingWhitespace {
						e.InsertRune(c, r)
						e.Next(c)
					}
				} else if e.AfterEndOfLine() {
					leadingWhitespace := e.LeadingWhitespace()
					if len(lineContents) > 0 && (strings.HasSuffix(lineContents, "(") || strings.HasSuffix(lineContents, "{") || strings.HasSuffix(lineContents, "[")) {
						// "smart indentation"
						leadingWhitespace += "\t"
					}
					e.InsertLineBelow()
					e.Down(c, status)
					e.Home()
					// Insert the same leading whitespace for the new line, while moving to the right
					for _, r := range leadingWhitespace {
						e.InsertRune(c, r)
						e.Next(c)
					}
				} else {
					// Split the current line in two
					if !e.SplitLine() {
						// Grab the leading whitespace from the current line
						leadingWhitespace := e.LeadingWhitespace()
						// Insert a line below, then move down and to the start of it
						e.InsertLineBelow()
						e.Down(c, status)
						e.Home()
						// Insert the same leading whitespace for the new line, while moving to the right
						for _, r := range leadingWhitespace {
							e.InsertRune(c, r)
							e.Next(c)
						}
					} else {
						e.Down(c, status)
						e.Home()
					}
				}
			} else {
				if e.AtLastLineOfDocument() {
					e.CreateLineIfMissing(e.DataY() + 1)
				}
				e.pos.Down(c)
			}
			e.redraw = true
		case "c:8", "c:127": // ctrl-h or backspace
			undo.Snapshot(e)
			if !e.DrawMode() && e.EmptyLine() {
				e.DeleteLine(e.DataY())
				e.pos.Up()
				e.TrimRight(e.DataY())
				e.End()
			} else if !e.DrawMode() && e.pos.AtStartOfLine() {
				if e.DataY() > 0 {
					e.pos.Up()
					e.End()
					e.TrimRight(e.DataY())
					e.Delete()
				}
			} else {
				// Move back
				e.Prev(c)
				// Type a blank
				e.SetRune(' ')
				e.WriteRune(c)
				if !e.DrawMode() && !e.AtOrAfterEndOfLine() {
					// Delete the blank
					e.Delete()
				}
			}
			e.redrawCursor = true
			e.redraw = true
		case "c:9": // tab
			if e.DrawMode() {
				// Do nothing
				break
			}
			y := e.DataY()
			leftRune := e.LeftRune()
			ext := filepath.Ext(filename)
			if leftRune == '.' && ext != "" {
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

				// Now the preceeding word before the "." has been found

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
			// Auto-indentation or regular indentation
			if !e.AtOrBeforeStartOfTextLine() && ext != "" {
				// If in the middle of the text and the character to the left is not a ".", then autoindent
				if strings.TrimSpace(e.Line(y-1)) == "" {
					// The line above is empty, do nothing
					break
				}
				// Move the current indentation to the same as the line above
				undo.Snapshot(e)

				var (
					//spaceHere         = e.LeadingWhitespace()
					spaceAbove        = e.LeadingWhitespaceAt(y - 1)
					trimmedLine       = strings.TrimSpace(e.Line(y))
					strippedLineAbove = e.StripSingleLineComment(strings.TrimSpace(e.Line(y - 1)))
					newLeadingSpace   = spaceAbove
					oneIndentation    = "\t"
				)
				if e.mode == modeShell {
					oneIndentation = strings.Repeat(" ", spacesPerTab)
				}

				// Smart-ish indentation
				if (strings.HasPrefix(trimmedLine, "case ") && !strings.HasPrefix(trimmedLine, "switch ")) || trimmedLine == "}" || trimmedLine == "]" || trimmedLine == ")" {
					//strings.HasSuffix(strippedLineAbove, "}") || strings.HasSuffix(strippedLineAbove, "]") || strings.HasSuffix(strippedLineAbove, ")") ||
					// Use one less indentation than the line above
					if (len(spaceAbove) - len(oneIndentation)) <= 0 {
						// Do nothing
						break
					}
					newLeadingSpace = spaceAbove[:len(spaceAbove)-len(oneIndentation)]
				} else if strings.HasSuffix(strippedLineAbove, "{") || strings.HasSuffix(strippedLineAbove, "[") || strings.HasSuffix(strippedLineAbove, "(") || strings.HasSuffix(strippedLineAbove, ":") {
					newLeadingSpace = spaceAbove + oneIndentation
				}

				e.SetLine(y, newLeadingSpace+trimmedLine)
				e.redrawCursor = true
				e.redraw = true
				break
			} else if e.mode == modeShell {
				undo.Snapshot(e)
				// For shell and PKGBUILD files, insert spaces instead of tab
				for i := 0; i < spacesPerTab; i++ {
					e.InsertRune(c, ' ')
				}
			} else {
				undo.Snapshot(e)
				// Insert a tab character to the file
				e.InsertRune(c, '\t')
			}
			// Write the spaces that represent the tab to the canvas
			e.WriteTab(c)
			// Move to the next position
			e.Next(c)
			// Prepare to redraw
			e.redrawCursor = true
			e.redraw = true
		case "c:1", "c:25": // ctrl-a, home (or ctrl-y for scrolling up in the st terminal)
			// First check if we just moved to this line with the arrow keys
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑"
			// If at an empty line, go up one line
			if !justMovedUpOrDown && e.EmptyRightTrimmedLine() {
				e.Up(c, status)
				//e.GoToStartOfTextLine()
				e.End()
			} else if x, err := e.DataX(); err == nil && x == 0 && !justMovedUpOrDown {
				// If at the start of the line,
				// go to the end of the previous line
				e.Up(c, status)
				e.End()
			} else if e.AtStartOfTextLine() {
				// If at the start of the text, go to the start of the line
				e.Home()
			} else {
				// If none of the above, go to the start of the text
				e.GoToStartOfTextLine()
			}
			e.redrawCursor = true
			e.SaveX(true)
		case "c:5": // ctrl-e, end
			// First check if we just moved to this line with the arrow keys
			justMovedUpOrDown := previousKey == "↓" || previousKey == "↑"
			// If we didn't just move here, and are at the end of the line,
			// move down one line and to the end, if not,
			// just move to the end.
			if !justMovedUpOrDown && e.AfterEndOfLine() {
				e.Down(c, status)
				e.End()
			} else {
				e.End()
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
		case "c:30": // ctrl-~, save and quit + clear the terminal
			clearOnQuit = true
			quit = true
			fallthrough
		case "c:19": // ctrl-s, save
			status.ClearAll(c)
			// Save the file
			if err := e.Save(&filename, !e.DrawMode()); err != nil {
				status.SetMessage(err.Error())
				status.Show(c, e)
			} else {
				// TODO: Go to the end of the document at this point, if needed
				// Lines may be trimmed for whitespace, so move to the end, if needed
				if !e.DrawMode() && e.AfterLineScreenContents() {
					e.End()
				}
				// Save the current location in the location history and write it to file
				e.SaveLocation(absFilename, locationHistory)
				// Status message
				status.SetMessage("Saved " + filename)
				status.Show(c, e)
				c.Draw()
			}
		case "c:21", "c:26": // ctrl-u or ctrl-z, undo (ctrl-z may background the application)
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
				case "c:27", "c:17": // esc or ctrl-q
					lns = ""
					fallthrough
				case "c:13": // return
					doneCollectingDigits = true
				}
			}
			status.ClearAll(c)
			if lns != "" {
				if ln, err := strconv.Atoi(lns); err == nil { // no error
					e.redraw = e.GoToLineNumber(ln, c, status, true)
				}
			}
			e.redrawCursor = true
		case "c:11": // ctrl-k, delete to end of line
			if e.Empty() {
				status.SetMessage("Empty file")
				status.Show(c, e)
				break
			}
			undo.Snapshot(e)
			e.DeleteRestOfLine()
			if !e.DrawMode() && e.EmptyRightTrimmedLine() {
				// Deleting the rest of the line cleared this line,
				// so just remove it.
				e.DeleteLine(e.DataY())
				// Then go to the start or end of the line, if needed
				if len(e.CurrentLine()) == 1 {
					e.Home()
				} else if e.AtOrAfterEndOfLine() {
					e.End()
				}
			}
			// TODO: Is this one needed/useful?
			vt100.Do("Erase End of Line")
			e.redraw = true
			e.redrawCursor = true
		case "c:24": // ctrl-x, cut line
			y := e.DataY()
			line := e.Line(y)
			// Now check if there is anything to cut
			if len(strings.TrimSpace(line)) == 0 {
				break
			}
			// Prepare to cut
			undo.Snapshot(e)
			// Check if ctrl-x was pressed once or twice, for this line
			if lastCutY != y { // Single line cut
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
			// No status message is needed for the cut operation, because it's visible that lines are cut
			e.redrawCursor = true
			e.redraw = true
		case "c:3": // ctrl-c, copy the stripped contents of the current line
			y := e.DataY()
			if lastCopyY != y { // Single line copy
				lastCopyY = y
				lastPasteY = -1
				lastCutY = -1
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
				}
				// Go to the end of the line, for easy line duplication with ctrl-c, enter, ctrl-v
				e.End()
			} else { // Multi line copy
				lastCopyY = y
				lastPasteY = -1
				lastCutY = -1
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
			// Try fetching the lines from the clipboard first
			lines, err := clipboard.ReadAll()
			if err == nil { // no error
				// Fix nonbreaking spaces first
				lines = strings.Replace(lines, string([]byte{0xc2, 0xa0}), string([]byte{0x20}), -1)
				// Split the text into lines and store it in "copyLines"
				copyLines = strings.Split(lines, "\n")
			}
			// Now check if there is anything to paste
			if len(copyLines) == 0 {
				break
			}
			// Prepare to paste
			undo.Snapshot(e)
			y := e.DataY()
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
				lastPasteY = y
				// Pressed the second time for this line number, paste multiple lines without trimming

				// copyLines contains the lines to be pasted, and they are > 1
				// the first line is skipped since that was already pasted when ctrl-v was pressed the first time
				lastIndex := len(copyLines[1:]) - 1
				// Start by pasting (and overwriting) an untrimmed version of this line
				e.SetLine(y, copyLines[0])
				// The paste the rest of the lines, also untrimmed
				for i, line := range copyLines[1:] {
					if i == lastIndex && len(strings.TrimSpace(line)) == 0 {
						// If the last line is blank, skip it
						break
					}
					e.InsertLineBelow()
					e.Down(c, nil) // no status message if the end of ducment is reached, there should always be a new line
					e.InsertString(c, line)
				}
			}
			// Prepare to redraw the text
			e.redrawCursor = true
			e.redraw = true
		case "c:2": // ctrl-b, bookmark
			if bookmark == nil {
				tmpBookmark := e.pos
				bookmark = &tmpBookmark
				status.SetMessage("Bookmarked line " + strconv.Itoa(e.LineNumber()))
			} else {
				bookmarkLine := bookmark.LineNumber()
				bookmark = nil
				status.SetMessage("Unbookmark line " + strconv.Itoa(bookmarkLine))
			}
			status.Show(c, e)
			e.redrawCursor = true
		case "c:10": // ctrl-j, jump to bookmark (or join line if bookmark is not set)
			if bookmark != nil {
				undo.Snapshot(e)
				e.GoToPosition(c, status, *bookmark)
				// Do the redraw manually before showing the status message
				e.DrawLines(c, true, false)
				e.redraw = false
				// Show the status message.
				status.SetMessage("Jumped to bookmark at line " + strconv.Itoa(e.LineNumber()))
				status.Show(c, e)
				e.redrawCursor = true
				break
			}
			// Join line
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
					e.End()
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
			if len([]rune(key)) > 0 && unicode.IsLetter([]rune(key)[0]) { // letter
				undo.Snapshot(e)
				// Check for if a special "first letter" has been pressed, which triggers vi-like behavior
				if firstLetterSinceStart == "" {
					firstLetterSinceStart = key
					// If the first pressed key is "G" and this is not git mode, then invoke vi-compatible behavior and jump to the end
					if key == "G" && (mode != modeGit) {
						// Go to the end of the document
						e.redraw = e.GoToLineNumber(e.Len(), c, status, true)
						e.redrawCursor = true
						firstLetterSinceStart = "x"
						break
					}
				}
				if firstLetterSinceStart == "O" {
					// If the first typed letter since starting this editor was 'O', and this is also uppercase,
					// then disregard the initial 'O'. This is to help vim-users.
					dropO = true
					// Set the first letter since start to something that will not trigger this branch any more.
					firstLetterSinceStart = "x"
					// ignore the O
					break
				}
				// If the previous letter was an "O" and this letter is lowercase, invoke vi-compatibility for a short moment
				if dropO {
					// This is a one-time operation
					dropO = false
					// Lowercase? Type the O, since it was meant to be typed.
					if len([]rune(key)) > 0 && unicode.IsLower([]rune(key)[0]) {
						e.Prev(c)
						e.SetRune('O')
						e.WriteRune(c)
						e.Next(c)
					} else if !e.DrawMode() {
						// Was this a special case of "OK" as the first thing written?
						if key == "K" {
							e.InsertRune(c, 'O')
							e.WriteRune(c)
							e.Next(c)
						}
					}
				}
				// Type the letter that was pressed
				if len([]rune(key)) > 0 {
					if !e.DrawMode() {
						// Insert a letter. This is what normally happens.
						e.InsertRune(c, []rune(key)[0])
						e.WriteRune(c)
						e.Next(c)
					} else {
						// Replace this letter.
						e.SetRune([]rune(key)[0])
						e.WriteRune(c)
					}
					e.redraw = true
				}
			} else if len([]rune(key)) > 0 && unicode.IsGraphic([]rune(key)[0]) { // any other key that can be drawn
				undo.Snapshot(e)

				// Place *something*
				r := []rune(key)[0]

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

				if !e.DrawMode() {
					e.InsertRune(c, []rune(key)[0])
				} else {
					e.SetRune([]rune(key)[0])
				}
				e.WriteRune(c)
				if len(string(r)) > 0 && !e.DrawMode() {
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
			status.ShowLineColWordCount(c, e, filename)
		} else if status.isError {
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
	}
	// Save the current location in the location history and write it to file
	e.SaveLocation(absFilename, locationHistory)

	// Clear all status bar messages
	status.ClearAll(c)

	// Quit everything that has to do with the terminal
	if clearOnQuit {
		vt100.Clear()
		vt100.Close()
	} else {
		c.Draw()
		fmt.Println()
	}
}
