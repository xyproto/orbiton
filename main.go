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
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/xyproto/syntax"
	"github.com/xyproto/vt100"
)

const version = "o 2.18.1"

var rebaseKeywords = []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground      = vt100.LightGreen // for when syntax highlighting is not in use
		defaultEditorBackground      = vt100.BackgroundDefault
		defaultStatusForeground      = vt100.White
		defaultStatusBackground      = vt100.BackgroundBlack
		defaultStatusErrorForeground = vt100.LightRed
		defaultStatusErrorBackground = vt100.BackgroundDefault
		defaultEditorSearchHighlight = vt100.LightMagenta
		defaultEditorHighlightTheme  = syntax.TextConfig{
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
			Whitespace:    "",
		}

		versionFlag = flag.Bool("version", false, "show version information")
		helpFlag    = flag.Bool("help", false, "show simple help")

		statusDuration = 2700 * time.Millisecond

		copyLine   string   // for the cut/copy/paste functionality
		bookmark   Position // for the bookmark/jump functionality
		statusMode bool     // if information should be shown at the bottom

		firstLetterSinceStart string

		locationHistory map[string]int // remember where we were in each absolute filename
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
           (or if in git interactive rebase mode, cycle the keywords)
ctrl-a     go to start of line, then start of text and then the previous line
ctrl-e     go to end of line and then the next line
ctrl-p     to scroll up 10 lines
ctrl-n     to scroll down 10 lines or go to the next match if a search is active
ctrl-k     to delete characters to the end of the line, then delete the line
ctrl-g     to toggle filename/line/column/unicode/word count status display
ctrl-d     to delete a single character
ctrl-t     to toggle syntax highlighting
ctrl-o     to toggle text or draw mode
ctrl-x     to cut the current line
ctrl-c     to copy the current line
ctrl-v     to paste the current line
ctrl-b     to bookmark the current line
ctrl-j     to jump to the bookmark
ctrl-u     to undo
ctrl-l     to jump to a specific line
ctrl-f     to search for a string
esc        to redraw the screen and clear the last search
ctrl-space to build Go, C++, word wrap
ctrl-r     to render the current text to a PDF document
ctrl-\     to toggle single-line comments

Set NO_COLOR=1 to 1 to disable colors.

`)
		return
	}

	filename, lineNumber := FilenameAndLineNumber(flag.Arg(0), flag.Arg(1))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}

	baseFilename := filepath.Base(filename)
	gitMode :=
		baseFilename == "COMMIT_EDITMSG" ||
			baseFilename == "MERGE_MSG" ||
			(strings.HasPrefix(baseFilename, "git-") &&
				!strings.Contains(baseFilename, ".") &&
				strings.Count(baseFilename, "-") >= 2)

	defaultHighlight := gitMode || baseFilename == "config" || baseFilename == "PKGBUILD" || baseFilename == "BUILD" || baseFilename == "WORKSPACE" || strings.Contains(baseFilename, ".") || strings.HasSuffix(baseFilename, "file") // Makefile, Dockerfile, Jenkinsfile, Vagrantfile

	// TODO: Introduce a separate mode for AsciiDoctor. Use Markdown syntax highlighting, for now.
	docMode := strings.HasSuffix(baseFilename, ".md") || strings.HasSuffix(baseFilename, ".adoc") || strings.HasSuffix(baseFilename, ".rst")

	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}

	vt100.Init()

	c := vt100.NewCanvas()
	c.ShowCursor()

	// 4 spaces per tab, scroll 10 lines at a time, no word wrap
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true, 10, defaultEditorSearchHighlight, defaultEditorHighlightTheme, gitMode, docMode)

	if gitMode {
		// The subject should ideally be maximum 50 characters long, then the body of the
		// git commit message can be 72 characters long. Because e-mail standards.
		e.wordWrapAt = 72
	}

	// For non-highlighted files, adjust the word wrap
	if !defaultHighlight {
		// Adjust the word wrap if the terminal is too narrow
		w := int(c.Width())
		if w < e.wordWrapAt {
			e.wordWrapAt = w
		}
	}

	// Use a theme for light backgrounds if XTERM_VERSION is set,
	// because $COLORFGBG is "15;0" even though the background is white.
	if os.Getenv("XTERM_VERSION") != "" {
		e.lightTheme()
	}

	e.respectNoColorEnvironmentVariable()

	e.gitMode = gitMode

	status := NewStatusBar(defaultStatusForeground, defaultStatusBackground, defaultStatusErrorForeground, defaultStatusErrorBackground, e, statusDuration)
	status.respectNoColorEnvironmentVariable()

	// Try to load the filename, ignore errors since giving a new filename is also okay
	loaded := e.Load(c, tty, filename) == nil

	// If we're editing a git commit message, add a newline and enable word-wrap at 80
	if e.gitMode {
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

	// We wish to redraw the canvas and reposition the cursor
	e.redraw = true
	e.redrawCursor = true

	// Friendly status message
	statusMessage := "New " + filename
	if loaded {
		if !e.Empty() {
			statusMessage = "Loaded " + filename
		} else {
			statusMessage = "Loaded empty file: " + filename
		}
		fileInfo, err := os.Stat(filename)
		if err != nil {
			quitError(tty, err)
		}
		if fileInfo.IsDir() {
			quitError(tty, errors.New(filename+" is a directory"))
		}
		testFile, err := os.OpenFile(filename, os.O_WRONLY, 0664)
		if err != nil {
			// Can not open the file for writing
			statusMessage += " (read only)"
			// Set the color to red when in read-only mode
			e.fg = vt100.Red
			// Disable syntax highlighting, to make it clear that the text is red
			e.highlight = false
			// Do a full reset and redraw
			c = e.FullResetRedraw(c, status)
			// Draw the editor lines again
			e.DrawLines(c, false, true)
			e.redraw = false
		}
		testFile.Close()
	} else if err := e.Save(filename, true); err != nil {
		// Check if the new file can be saved before the user starts working on the file.
		quitError(tty, err)
	} else {
		// Creating a new empty file worked out fine, don't save it until the user saves it
		if os.Remove(filename) != nil {
			// This should never happen
			quitError(tty, errors.New("could not remove an empty file that was just created: "+filename))
		}
	}

	// If the file starts with a hashbang, enable syntax highlighting
	if strings.HasPrefix(strings.TrimSpace(e.Line(0)), "#!") {
		// Enable highlighting and redraw
		e.highlight = true
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
	} else if recordedLineNumber, ok := locationHistory[absFilename]; ok && !gitMode {
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

	dropO := false
	quit := false
	previousKey := ""

	for !quit {
		key := tty.String()
		switch key {
		case "c:17": // ctrl-q, quit
			quit = true
		case "c:23": // ctrl-w, format (or if in git mode, cycle interactive rebase keywords)
			if line := e.CurrentLine(); e.gitMode && hasAnyPrefixWord(line, rebaseKeywords) {
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			}

			status.Clear(c)
			status.SetMessage("Formatting")
			status.Show(c, e)

			// Not in git mode, format Go or C++ code with goimports or clang-format
			undo.Snapshot(e)
			// Map from formatting command to a list of file extensions
			format := map[*exec.Cmd][]string{
				exec.Command("goimports", "-w", "--"):                                             {".go"},
				exec.Command("clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"): {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
				exec.Command("zig", "fmt"):                                                        {".zig"},
				exec.Command("v", "fmt"):                                                          {".v"},
				exec.Command("rustfmt"):                                                           {".rs"},
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
							err := e.Save(tempFilename, true)
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
									// TODO: This error never shows up. Fix it.
									status.SetMessage("Failed to format code: " + errorMessage)
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
									e.Load(c, tty, tempFilename)
									// Mark the data as changed, despite just having loaded a file
									e.changed = true
									formatted = true
								}
								// Try to remove the temporary file regardless if "goimports -w" worked out or not
								_ = os.Remove(tempFilename)
							}
							// Try to close the file. f.Close() checks if f is nil before closing.
							_ = f.Close()
							e.redraw = true
						}
						break OUT
					}
				}
			}
			if !gitMode && !formatted {
				// Check if at least one line is longer than the word wrap limit first
				// word wrap at the current width - 5, with an allowed overshoot of 5 runes
				if e.WrapAllLinesAt(e.wordWrapAt-5, 5) {
					e.redraw = true
					e.redrawCursor = true
				}
			}
		case "c:6": // ctrl-f, search for a string
			e.SearchMode(c, status, tty, true)
		case "c:0": // ctrl-space, build source code to executable, word wrap or convert to PDF, depending on the mode
			if strings.HasSuffix(filepath.Base(filename), ".adoc") {
				asciidoctor := exec.Command("/usr/bin/asciidoctor", "-b", "manpage", "-o", "manpage.1", filename)
				if err := asciidoctor.Run(); err != nil {
					statusMessage = err.Error()
					status.ClearAll(c)
					status.SetMessage(statusMessage)
					status.Show(c, e)
					break // from case
				}
				statusMessage = "Saved manpage.1"
				status.ClearAll(c)
				status.SetMessage(statusMessage)
				status.Show(c, e)
				break // from case
				// Is this a Markdown file? Save to PDF, either by using pandoc or by writing the text file directly
			} else if pandocPath := which("pandoc"); e.markdownMode && strings.HasSuffix(filepath.Base(filename), ".md") && pandocPath != "" {

				go func() {
					pdfFilename := "o.pdf"

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

					err := e.Save(tmpfn, !e.DrawMode())
					if err != nil {
						statusMessage = err.Error()
						status.ClearAll(c)
						status.SetMessage(statusMessage)
						status.Show(c, e)
						return // from goroutine
					}

					pandoc := exec.Command(pandocPath, "-N", "--toc", "-V", "geometry:a4paper", "-o", "o.pdf", tmpfn)
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

			// Is this a .go, .cpp, .cc, .cxx, .h, .hpp, .c++, .h++, .c, .zig or .v file?

			// Map from formatting command to a list of file extensions
			build := map[*exec.Cmd][]string{
				exec.Command("go", "build"):    {".go"},
				exec.Command("cxx"):            {".cpp", ".cc", ".cxx", ".h", ".hpp", ".c++", ".h++", ".c"},
				exec.Command("zig", "build"):   {".zig"},
				exec.Command("v", filename):    {".v"},
				exec.Command("cargo", "build"): {".rs"},
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

						// Use rustc instead of cargo if Cargo.toml is missing and the extension is .rs
						if ext == ".rs" && (!exists("Cargo.toml") && !exists("../Cargo.toml")) {
							cmd = exec.Command("rustc", filename)
						}

						output, err := cmd.CombinedOutput()
						if err != nil || bytes.Contains(output, []byte(": error:")) {
							status.ClearAll(c)
							status.SetErrorMessage("Build error")
							lines := strings.Split(string(output), "\n")
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
												status.SetErrorMessage(strings.Join(fields[3:], " "))
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
							status.SetMessage("Build OK")
							status.Show(c, e)
						}
						break OUT2
					}
				}
			}
			if !foundExtensionToBuild {
				// Building this file extension is not implemented yet.
				// Just display the current time.
				status.ClearAll(c)
				statusMessage := time.Now().Format("15:04") // HH:MM
				status.SetMessage(statusMessage)
				status.Show(c, e)
			}
		case "c:18": // ctrl-r, render to PDF, or if in git mode, cycle rebase keywords

			// Are we in git mode?
			if line := e.CurrentLine(); e.gitMode && hasAnyPrefixWord(line, rebaseKeywords) {
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

				pdfFilename := "o.pdf"

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
		case "c:28": // ctrl-\, toggle comment
			e.ToggleComment()
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
			// Move the screen cursor
			if !e.DrawMode() {
				if e.DataY() > 0 {
					// Move the position up in the current screen
					if e.UpEnd(c) != nil {
						// If below the top, scroll the contents up
						if e.DataY() > 0 {
							e.redraw = e.ScrollUp(c, status, 1)
							e.redrawCursor = true
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
			if !e.DrawMode() {
				if e.DataY() < e.Len() {
					// Move the position down in the current screen
					if e.DownEnd(c) != nil {
						// If at the bottom, don't move down, but scroll the contents
						// Output a helpful message
						if !e.AfterEndOfDocument() {
							e.redraw = e.ScrollDown(c, status, 1)
							e.redrawCursor = true
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
		case "c:14": // ctrl-n, scroll down or jump to next match
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
		case "c:16": // ctrl-p, scroll up
			e.redraw = e.ScrollUp(c, status, e.pos.scrollSpeed)
			e.redrawCursor = true
			if !e.DrawMode() && e.AfterLineScreenContents() {
				e.End()
			}
		case "c:20": // ctrl-t, toggle syntax highlighting or use the next git interactive rebase keyword
			if line := e.CurrentLine(); e.gitMode && hasAnyPrefixWord(line, []string{"p", "pick", "r", "reword", "e", "edit", "s", "squash", "f", "fixup", "x", "exec", "b", "break", "d", "drop", "l", "label", "t", "reset", "m", "merge"}) {
				undo.Snapshot(e)
				newLine := nextGitRebaseKeyword(line)
				e.SetLine(e.DataY(), newLine)
				e.redraw = true
				e.redrawCursor = true
				break
			} else {
				e.ToggleHighlight()
				if e.highlight {
					e.bg = defaultEditorBackground
				} else {
					e.bg = vt100.BackgroundDefault
				}
			}
			// Now do a full reset/redraw
			fallthrough
		case "c:27": // esc, clear search term, reset, clean and redraw
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
			undo.Snapshot(e)
			if !e.DrawMode() {
				// Place a tab
				if !e.DrawMode() {
					e.InsertRune(c, '\t')
				} else {
					e.SetRune('\t')
				}
				// Write the spaces that represent the tab
				e.WriteTab(c)
				// Move to the next position
				if !e.DrawMode() {
					e.Next(c)
				}
			}
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
		case "c:19": // ctrl-s, save
			// Write the file
			status.ClearAll(c)
			// Save the file
			if err := e.Save(filename, !e.DrawMode()); err != nil {
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
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.DeleteRestOfLine()
				if !e.DrawMode() && e.EmptyRightTrimmedLine() {
					// Deleting the rest of the line cleared this line,
					// so just remove it.
					e.DeleteLine(e.DataY())
					// Then go to the end of the line, if needed
					if e.AtOrAfterEndOfLine() {
						e.End()
					}
				}
				vt100.Do("Erase End of Line")
				e.redraw = true
			}
			e.redrawCursor = true
		case "c:24": // ctrl-x, cut line
			undo.Snapshot(e)
			y := e.DataY()
			copyLine = e.Line(y)
			// Copy the line to the clipboard
			_ = clipboard.WriteAll(copyLine)
			e.DeleteLine(y)
			e.redrawCursor = true
			e.redraw = true
		case "c:3": // ctrl-c, copy the stripped contents of the current line
			trimmed := strings.TrimSpace(e.Line(e.DataY()))
			if trimmed != "" {
				copyLine = trimmed
				// Copy the line to the clipboard
				_ = clipboard.WriteAll(copyLine)
			}
			e.redrawCursor = true
			e.redraw = true
		case "c:22": // ctrl-v, paste
			undo.Snapshot(e)
			// Try fetching the line from the clipboard first
			lines, err := clipboard.ReadAll()
			if err == nil { // no error
				if strings.Contains(lines, "\n") {
					copyLine = strings.SplitN(lines, "\n", 2)[0]
				} else {
					copyLine = lines
				}
			}
			if e.EmptyRightTrimmedLine() {
				// If the line is empty, use the existing indentation before pasting
				e.SetLine(e.DataY(), e.LeadingWhitespace()+strings.TrimSpace(copyLine))
			} else {
				// If the line is not empty, insert the trimmed string
				e.InsertString(c, strings.TrimSpace(copyLine))
			}
			// Prepare to redraw the text
			e.redrawCursor = true
			e.redraw = true
		case "c:2": // ctrl-b, bookmark
			bookmark = e.pos
			status.SetMessage("Bookmarked line " + strconv.Itoa(e.LineNumber()))
			status.Show(c, e)
			e.redrawCursor = true
		case "c:10": // ctrl-j, jump to bookmark
			e.GoToPosition(c, status, bookmark)
			// Do the redraw manually before showing the status message
			e.DrawLines(c, true, false)
			e.redraw = false
			// Show the status message.
			status.SetMessage("Jumped to bookmark at line " + strconv.Itoa(e.LineNumber()))
			status.Show(c, e)
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
					// If the first pressed key is "G", then invoke vi-compatible behavior and jump to the end
					if key == "G" {
						// Go to the end of the document
						e.redraw = e.GoToLineNumber(e.Len()+1, c, status, true)
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
				// If the previous letter was an "O" and this letter is uppercase, invoke vi-compatibility for a short moment
				if dropO {
					// This is a one-time operation
					dropO = false
					// Lowercase? Type the O, since it was meant to be typed.
					if len([]rune(key)) > 0 && unicode.IsLower([]rune(key)[0]) {
						e.Prev(c)
						e.SetRune('O')
						e.WriteRune(c)
						e.Next(c)
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
					if e.pos.sx > 0 && len(lineContents) == 0 && len(whitespaceInFront) > 0 {
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
				if len(string(r)) > 0 {
					if !e.DrawMode() {
						// Move to the next position
						e.Next(c)
					}
				}
				e.redrawCursor = true
				e.redraw = true
			}
		}
		previousKey = key
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
	// Quit everything that has to do with the terminal
	vt100.Clear()
	vt100.Close()
	tty.Close()
}
