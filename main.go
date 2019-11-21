package main

import (
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

const versionString = "o 2.10.1"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.LightGreen // for when syntax highlighting is not in use
		defaultEditorBackground       = vt100.BackgroundDefault
		defaultEditorStatusForeground = vt100.White
		defaultEditorStatusBackground = vt100.BackgroundBlack
		defaultEditorSearchHighlight  = vt100.LightMagenta
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
			Whitespace:    "",
		}

		version = flag.Bool("version", false, "show version information")
		help    = flag.Bool("help", false, "show simple help")

		statusDuration = 2700 * time.Millisecond

		copyLine   string   // for the cut/copy/paste functionality
		bookmark   Position // for the bookmark/jump functionality
		statusMode bool     // if information should be shown at the bottom

		firstLetterSinceStart string

		locationHistory map[string]int // remember where we were in each absolute filename
	)

	flag.Parse()

	if *version {
		fmt.Println(versionString)
		return
	}

	if *help {
		fmt.Println(versionString + " - simple and limited text editor")
		fmt.Print(`
Hotkeys

ctrl-q to quit
ctrl-s to save
ctrl-o to format the current file with "go fmt"
ctrl-a go to start of line, then start of text, then previous paragraph
ctrl-e go to end of line, then next paragraph
ctrl-p to scroll up 10 lines
ctrl-n to scroll down 10 lines or go to the next match if a search is active
ctrl-k to delete characters to the end of the line, then delete the line
ctrl-g to toggle filename/line/column/unicode/word count status display
ctrl-d to delete a single character
ctrl-t to toggle syntax highlighting
ctrl-y to toggle text or draw mode (for ASCII graphics)
ctrl-x to cut the current line
ctrl-c to copy the current line
ctrl-v to paste the current line
ctrl-b to bookmark the current line
ctrl-j to jump to the bookmark
ctrl-u to undo
ctrl-l to jump to a specific line
ctrl-f to search for a string
esc to redraw the screen and clear the last search
ctrl-w to toggle single-line comments
ctrl-space to build
ctrl-r to render the current text to a PNG image
`)
		return
	}

	filename, lineNumber := FilenameAndLineNumber(flag.Arg(0), flag.Arg(1))
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}

	baseFilename := filepath.Base(filename)
	gitMode := baseFilename == "COMMIT_EDITMSG" || (strings.HasPrefix(baseFilename, "git-") && !strings.Contains(baseFilename, ".") && strings.Count(baseFilename, "-") >= 2)
	defaultHighlight := gitMode || baseFilename == "PKGBUILD" || strings.Contains(baseFilename, ".") || baseFilename == "Makefile"

	tty, err := vt100.NewTTY()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: "+err.Error())
		os.Exit(1)
	}

	vt100.Init()

	c := vt100.NewCanvas()
	c.ShowCursor()

	// 4 spaces per tab, scroll 10 lines at a time, no word wrap
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true, 10, defaultEditorSearchHighlight, defaultEditorHighlightTheme)

	// Use a theme for light backgrounds if XTERM_VERSION is set,
	// because $COLORFGBG is "15;0" even though the background is white.
	if os.Getenv("XTERM_VERSION") != "" {
		e.lightTheme()
	}

	e.respectNoColorEnvironmentVariable()

	e.gitMode = gitMode

	status := NewStatusBar(defaultEditorStatusForeground, defaultEditorStatusBackground, e, statusDuration)
	status.respectNoColorEnvironmentVariable()

	// Try to load the filename, ignore errors since giving a new filename is also okay
	loaded := e.Load(c, tty, filename) == nil

	// If we're editing a git commit message, add a newline and enable word-wrap at 80
	if e.gitMode {
		e.gitColor = vt100.LightGreen
		status.fg = vt100.LightBlue
		status.bg = vt100.BackgroundDefault
		e.InsertLineBelow()
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
	} else if recordedLineNumber, ok := locationHistory[absFilename]; ok {
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
	for !quit {
		key := tty.String()
		switch key {
		case "c:17": // ctrl-q, quit
			quit = true
		case "c:23": // ctrl-o, format
			undo.Snapshot(e)
			// Map from formatting command to a list of file extensions
			format := map[*exec.Cmd][]string{
				exec.Command("/usr/bin/goimports", "-w", "--"):                                             []string{".go"},
				exec.Command("/usr/bin/clang-format", "-fallback-style=WebKit", "-style=file", "-i", "--"): []string{".cpp", ".cxx", ".h", ".hpp", ".c++", ".h++"},
			}
			formatted := false
		OUT:
			for cmd, extensions := range format {
				for _, ext := range extensions {
					if strings.HasSuffix(filename, ext) {
						// Use a globally unique temp file
						if f, err := ioutil.TempFile("/tmp", "__o*"+ext); err == nil {
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
			if !formatted {
				status.SetMessage("Can only format Go or C++ code.")
				status.Show(c, e)
			}
		case "c:6": // ctrl-f, search for a string
			e.SearchMode(c, status, tty, true)
		case "c:0": // ctrl-space, "cxx" or "go build"
			// Map from formatting command to a list of file extensions
			build := map[*exec.Cmd][]string{
				exec.Command("go", "build"): []string{".go"},
				exec.Command("cxx"):         []string{".cpp", ".cxx", ".h", ".hpp", ".c++", ".h++"},
			}
		OUT2:
			for cmd, extensions := range build {
				for _, ext := range extensions {
					if strings.HasSuffix(filename, ext) {
						status.ClearAll(c)
						status.SetMessage("Building")
						status.Show(c, e)

						output, err := cmd.CombinedOutput()
						if err != nil {
							lines := strings.Split(string(output), "\n")
							for _, line := range lines {
								if strings.Count(line, ":") >= 3 {
									fields := strings.SplitN(line, ":", 4)

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
									break
								}
							}
						} else {
							// TODO: This is not correct for cxx / C++, fix
							status.ClearAll(c)
							status.SetMessage("Build OK")
							status.Show(c, e)
						}
						break OUT2
					}
				}
			}
		case "c:18": // ctrl-r, screen recording or render as PNG
			imageFilename := filename + ".png"
			// Show a status message while writing
			statusMessage := "Rendering image..."
			status.SetMessage(statusMessage)
			status.Show(c, e)
			// Write the image
			if err := e.Render(imageFilename); err != nil {
				statusMessage = err.Error()
			} else {
				statusMessage = "Saved " + imageFilename
			}
			// Show a status message after writing
			status.SetMessage(statusMessage)
			status.Show(c, e)
		case "c:15": // ctrl-w, toggle comment
			e.ToggleComment()
			e.redraw = true
			e.redrawCursor = true
		case "c:25": // ctrl-y, toggle ASCII draw mode
			e.ToggleDrawMode()
			statusMessage := "Text mode"
			if e.DrawMode() {
				statusMessage = "Draw mode"
			}
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
		case "c:20": // ctrl-t, toggle syntax highlighting
			e.ToggleHighlight()
			if e.highlight {
				e.bg = defaultEditorBackground
			} else {
				e.bg = vt100.BackgroundDefault
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
		case "c:1": // ctrl-a, home
			// If at an empty line, go up one line
			if e.EmptyRightTrimmedLine() {
				e.Up(c, status)
				e.GoToStartOfTextLine()
			} else if x, err := e.DataX(); err == nil && x == 0 {
				// If at the start of the line,
				// go to the end of the previous paragraph
				if e.GoToPrevParagraph(c, status) {
					e.redraw = true
					e.End()
				} else {
					// if a previous paragraph was not found, go to the start of the text above
					e.Up(c, status)
					e.GoToStartOfTextLine()
				}
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
			if e.AfterEndOfLine() {
				// go to the start of the next paragraph
				e.redraw = e.GoToNextParagraph(c, status)
				e.GoToStartOfTextLine()
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
			if err := e.Save(filename, !e.DrawMode()); err != nil {
				status.SetMessage(err.Error())
				status.Show(c, e)
			} else {
				// TODO: Go to the end of the document at this point, if needed
				// Lines may be trimmed for whitespace, so move to the end, if needed
				if !e.DrawMode() && e.AfterLineScreenContents() {
					e.End()
				}
				// Status message
				status.SetMessage("Saved " + filename)
				status.Show(c, e)
				c.Draw()
			}
			// Save the current location in the location history and write it to file
			e.SaveLocation(absFilename, locationHistory)
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
					if unicode.IsLower([]rune(key)[0]) {
						e.Prev(c)
						e.SetRune('O')
						e.WriteRune(c)
						e.Next(c)
					}
				}
				// Type the letter that was pressed
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
			} else if key != "" { // any other key
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
