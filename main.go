package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xyproto/vt100"
)

const versionString = "o 2.3.4"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.White
		defaultEditorBackground       = vt100.BackgroundDefault
		defaultEditorStatusForeground = vt100.Black
		defaultEditorStatusBackground = vt100.BackgroundGray
		defaultEditorSearchHighlight  = vt100.LightBlue

		version = flag.Bool("version", false, "show version information")
		help    = flag.Bool("help", false, "show simple help")

		statusDuration = 2700 * time.Millisecond

		redraw     bool     // if the contents should be redrawn in the next loop
		copyLine   string   // for the cut/copy/paste functionality
		bookmark   Position // for the bookmark/jump functionality
		statusMode bool     // if information should be shown at the bottom
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
ctrl-f to format the current file with "go fmt"
ctrl-a go to start of line, then start of text
ctrl-e go to end of line
ctrl-p to scroll up 10 lines
ctrl-n to scroll down 10 lines
ctrl-k to delete characters to the end of the line, then delete the line
ctrl-g to toggle filename/line/column/unicode/word count status display
ctrl-d to delete a single character
ctrl-t to toggle syntax highlighting
ctrl-r to toggle text or draw mode (for ASCII graphics)
ctrl-x to cut the current line
ctrl-c to copy the current line
ctrl-v to paste the current line
ctrl-b to bookmark the current position
ctrl-j to jump to the bookmark
ctrl-h to show a minimal help text
ctrl-u to undo
ctrl-l to jump to a specific line
ctrl-w to search (press return to repeat last search)
esc to redraw the screen
`)
		return
	}

	if flag.Arg(0) == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}

	filename, lineNumber := FilenameAndLineNumber(flag.Arg(0), flag.Arg(1))

	defaultHighlight := filepath.Base(filename) == "PKGBUILD" || strings.Contains(filepath.Base(filename), ".")

	tty, err := vt100.NewTTY()
	if err != nil {
		panic(err)
	}

	vt100.Init()

	c := vt100.NewCanvas()
	c.ShowCursor()

	// 4 spaces per tab, scroll 10 lines at a time
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true, 10, defaultEditorSearchHighlight)

	status := NewStatusBar(defaultEditorStatusForeground, defaultEditorStatusBackground, e, statusDuration)

	// Try to load the filename, ignore errors since giving a new filename is also okay
	loaded := e.Load(filename) == nil

	// Draw editor lines from line 0 to h onto the canvas at 0,0
	h := int(c.Height())
	e.WriteLines(c, 0, h, 0, 0)

	// Friendly status message
	statusMessage := "New " + filename
	if loaded {
		if !e.Empty() {
			statusMessage = "Loaded " + filename
		} else {
			statusMessage = "Loaded an empty file: " + filename
		}
	}
	status.SetMessage(statusMessage)
	status.Show(c, e)
	c.Draw()

	// Undo buffer with room for 4096 actions
	undo := NewUndo(4096)

	// Resize handler
	SetUpResizeHandler(c, e, tty)

	tty.SetTimeout(10 * time.Millisecond)

	previousX := -1
	previousY := -1

	if lineNumber > 0 {
		redraw = e.GoToLineNumber(lineNumber, c, status)
	}

	quit := false
	for !quit {
		key := tty.Key()
		switch key {
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			undo.Snapshot(e)
			// Use a globally unique temp file
			if f, err := ioutil.TempFile("/tmp", "_red*.go"); !e.DrawMode() && err == nil {
				// no error, everything is fine
				tempFilename := f.Name()
				err := e.Save(tempFilename, true)
				if err == nil {
					// Run "go fmt" on the temporary file
					cmd := exec.Command("/usr/bin/gofmt", "-w", tempFilename)
					err = cmd.Run()
					if err == nil {
						e.Load(tempFilename)
						// Mark the data as changed, despite just having loaded a file
						e.changed = true
					}
					// Try to remove the temporary file regardless if "gofmt -w" worked out or not
					_ = os.Remove(tempFilename)
				}
				// Try to close the file. f.Close() checks if f is nil before closing.
				_ = f.Close()
				redraw = true
			}
		case 20: // ctrl-t, toggle syntax highlighting
			e.ToggleHighlight()
			redraw = true
		case 23: // ctrl-w, search
			s := e.SearchTerm()
			//e.SetSearchTerm(s, c)
			status.ClearAll(c)
			if s == "" {
				status.SetMessage("Search:")
			} else {
				status.SetMessage("Search: " + s)
			}
			status.ShowNoTimeout(c, e)
			doneCollectingLetters := false
			for !doneCollectingLetters {
				key2 := tty.Key()
				switch key2 {
				case 127: // backspace
					if len(s) > 0 {
						s = s[:len(s)-1]
						e.SetSearchTerm(s, c)
						status.SetMessage("Search: " + s)
						status.ShowNoTimeout(c, e)
					}
				case 27, 17: // esc or ctrl-q
					s = ""
					e.SetSearchTerm(s, c)
					fallthrough
				case 13: // return
					doneCollectingLetters = true
				default:
					if key2 != 0 {
						s += string(rune(key2))
						e.SetSearchTerm(s, c)
						status.SetMessage("Search: " + s)
						status.ShowNoTimeout(c, e)
					}
				}
			}
			status.ClearAll(c)
			if s != "" {
				// Go to the next line with "s"
				foundY := -1
				foundX := -1
				for y := e.DataY(); y < e.Len(); y++ {
					lineContents := e.Line(y)
					if y == e.DataY() {
						x, err := e.DataX()
						if err != nil {
							continue
						}
						// Search from the next position on this line
						x++
						if x >= len(lineContents) {
							continue
						}
						if strings.Contains(lineContents[x:], s) {
							foundX = x + strings.Index(lineContents[x:], s)
							foundY = y
							break
						}
					} else {
						if strings.Contains(lineContents, s) {
							foundX = strings.Index(lineContents, s)
							foundY = y
							break
						}
					}
				}
				if foundY != -1 {
					e.GoTo(foundY, c, status)
					if foundX != -1 {
						tabs := strings.Count(e.Line(foundY), "\t")
						e.pos.sx = foundX + (tabs * (e.spacesPerTab - 1))
					}
					redraw = true
				} else {
					status.SetMessage("Not found")
					status.Show(c, e)
				}
			}
		case 18: // ctrl-r, toggle draw mode
			e.ToggleDrawMode()
			statusMessage := "Text mode"
			if e.DrawMode() {
				statusMessage = "Draw mode"
			}
			status.SetMessage(statusMessage)
			status.Show(c, e)
		case 7: // ctrl-g, status mode
			statusMode = !statusMode
			if statusMode {
				status.ShowLineColWordCount(c, e, filename)
			} else {
				status.ClearAll(c)
			}
		case 252: // left arrow
			if !e.DrawMode() {
				e.Prev(c)
				if e.AfterLineContents() {
					e.End()
				}
				e.SaveX(true)
			} else {
				// Draw mode
				e.pos.Left()
			}
		case 254: // right arrow
			if !e.DrawMode() {
				if e.DataY() < e.Len() {
					e.Next(c)
				}
				if e.AfterLineContents() {
					e.End()
				}
				e.SaveX(true)
			} else {
				// Draw mode
				e.pos.Right(c)
			}
		case 253: // up arrow
			// Move the screen cursor
			if !e.DrawMode() {
				if e.DataY() > 0 {
					// Move the position up in the current screen
					if e.UpEnd(c) != nil {
						// If below the top, scroll the contents up
						if e.DataY() > 0 {
							redraw = e.ScrollUp(c, status, 1)
							e.pos.Down(c)
							e.UpEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if e.AfterLineContents() {
						e.End()
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineContents() {
					e.End()
				}
			} else {
				e.pos.Up()
			}
		case 255: // down arrow
			if !e.DrawMode() {
				if e.DataY() < e.Len() {
					// Move the position down in the current screen
					if e.DownEnd(c) != nil {
						// If at the bottom, don't move down, but scroll the contents
						// Output a helpful message
						if !e.AtOrAfterEndOfDocument() {
							redraw = e.ScrollDown(c, status, 1)
							e.pos.Up()
							e.DownEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if e.AfterLineContents() {
						e.End()
					}
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.AfterLineContents() {
					e.End()
				}
			} else {
				e.pos.Down(c)
			}
		case 14: // ctrl-n, scroll down
			redraw = e.ScrollDown(c, status, e.pos.scrollSpeed)
			if !e.DrawMode() && e.AfterLineContents() {
				e.End()
			}
		case 16: // ctrl-p, scroll up
			redraw = e.ScrollUp(c, status, e.pos.scrollSpeed)
			if !e.DrawMode() && e.AfterLineContents() {
				e.End()
			}
		case 8: // ctrl-h, help
			status.SetMessage("[" + versionString + "] ctrl-s to save, ctrl-q to quit")
			status.Show(c, e)
		case 27: // esc, redraw
			redraw = true
		case 32: // space
			undo.Snapshot(e)
			// Place a space
			if !e.DrawMode() {
				e.InsertRune(' ')
				redraw = true
			} else {
				e.SetRune(' ')
			}
			e.WriteRune(c)
			if e.DrawMode() {
				redraw = true
			} else {
				// Move to the next position
				e.Next(c)
			}
		case 13: // return
			undo.Snapshot(e)
			// if the current line is empty, insert a blank line
			if !e.DrawMode() {
				e.TrimRight(e.DataY())
				lineContents := e.CurrentLine()
				e.FirstScreenPosition(e.DataY())
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
					if leadingWhitespace == "" && (strings.HasSuffix(lineContents, "(") || strings.HasSuffix(lineContents, "{") || strings.HasSuffix(lineContents, "[")) {
						// "smart indentation"
						leadingWhitespace = "\t"
					}
					e.InsertLineBelow()
					h := int(c.Height())
					if e.DataY() >= (h - 1) {
						e.ScrollDown(c, status, 1)
					}
					e.pos.Down(c)
					e.Home()
					// Insert the same leading whitespace for the new line, while moving to the right
					for _, r := range leadingWhitespace {
						e.InsertRune(r)
						e.Next(c)
					}
				} else if e.AtOrAfterEndOfLine() {
					leadingWhitespace := e.LeadingWhitespace()
					if leadingWhitespace == "" && (strings.HasSuffix(lineContents, "(") || strings.HasSuffix(lineContents, "{") || strings.HasSuffix(lineContents, "[")) {
						// "smart indentation"
						leadingWhitespace = "\t"
					}
					e.InsertLineBelow()
					e.Down(c, status)
					e.Home()
					// Insert the same leading whitespace for the new line, while moving to the right
					for _, r := range leadingWhitespace {
						e.InsertRune(r)
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
							e.InsertRune(r)
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
			redraw = true
		case 127: // backspace
			undo.Snapshot(e)
			if !e.DrawMode() && e.CurrentLine() == "" {
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
			redraw = true
		case 9: // tab
			undo.Snapshot(e)
			if !e.DrawMode() {
				// Place a tab
				if !e.DrawMode() {
					e.InsertRune('\t')
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
			redraw = true
		case 1: // ctrl-a, home
			// toggle between start of line and start of non-whitespace
			if e.pos.AtStartOfLine() {
				e.pos.SetX(e.FirstScreenPosition(e.DataY()))
			} else {
				e.Home()
			}
			e.SaveX(true)
		case 5: // ctrl-e, end
			e.End()
			e.SaveX(true)
		case 4: // ctrl-d, delete
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.Delete()
				redraw = true
			}
		case 19: // ctrl-s, save
			if err := e.Save(filename, !e.DrawMode()); err != nil {
				status.SetMessage(err.Error())
				status.Show(c, e)
			} else {
				// TODO: Go to the end of the document at this point, if needed
				// Lines may be trimmed for whitespace, so move to the end, if needed
				if !e.DrawMode() && e.AfterLineContents() {
					e.End()
				}
				// Status message
				status.SetMessage("Saved " + filename)
				status.Show(c, e)
				c.Draw()
			}
		case 21, 26: // ctrl-u or ctrl-z, undo (ctrl-z may background the application)
			if err := undo.Restore(e); err == nil {
				//c.Draw()
				x := e.pos.ScreenX()
				y := e.pos.ScreenY()
				vt100.SetXY(uint(x), uint(y))
				redraw = true
			} else {
				status.SetMessage("Nothing more to undo")
				status.Show(c, e)
			}
		case 12: // ctrl-l, go to line number
			status.ClearAll(c)
			status.SetMessage("Go to line number:")
			status.ShowNoTimeout(c, e)
			lns := ""
			doneCollectingDigits := false
			for !doneCollectingDigits {
				numkey := tty.Key()
				switch numkey {
				case 48, 49, 50, 51, 52, 53, 54, 55, 56, 57: // 0 .. 9
					lns += string('0' + (numkey - 48))
					status.SetMessage("Go to line number: " + lns)
					status.ShowNoTimeout(c, e)
				case 127: // backspace
					if len(lns) > 0 {
						lns = lns[:len(lns)-1]
						status.SetMessage("Go to line number: " + lns)
						status.ShowNoTimeout(c, e)
					}
				case 27, 17: // esc or ctrl-q
					lns = ""
					fallthrough
				case 13: // return
					doneCollectingDigits = true
				}
			}
			status.ClearAll(c)
			if lns != "" {
				if ln, err := strconv.Atoi(lns); err == nil { // no error
					redraw = e.GoToLineNumber(ln, c, status)
					status.SetMessage(lns)
					status.Show(c, e)
				}
			}
		case 11: // ctrl-k, delete to end of line
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
				}
				vt100.Do("Erase End of Line")
				redraw = true
			}
		case 24: // ctrl-x, cut
			undo.Snapshot(e)
			y := e.DataY()
			copyLine = e.Line(y)
			e.DeleteLine(y)
			redraw = true
		case 3: // ctrl-c, copy line
			copyLine = e.Line(e.DataY())
			redraw = true
		case 22: // ctrl-v, paste
			undo.Snapshot(e)
			e.SetLine(e.DataY(), copyLine)
			redraw = true
		case 2: // ctrl-b, bookmark
			bookmark = e.pos
		case 10: // ctrl-j, jump to bookmark
			// TODO: Add a check for if a bookmark exists?
			e.pos = bookmark
			redraw = true
		default:
			if (key >= 'a' && key <= 'z') || (key >= 'A' && key <= 'Z') { // letter
				undo.Snapshot(e)
				// Place a letter
				if !e.DrawMode() {
					e.InsertRune(rune(key))
				} else {
					e.SetRune(rune(key))
				}
				e.WriteRune(c)
				if !e.DrawMode() {
					// Move to the next position
					e.Next(c)
				}
				redraw = true
			} else if key != 0 { // any other key
				// Place *something*
				r := rune(key)
				if !e.DrawMode() {
					e.InsertRune(rune(key))
				} else {
					e.SetRune(rune(key))
				}
				e.WriteRune(c)
				if len(string(r)) > 0 {
					if !e.DrawMode() {
						// Move to the next position
						e.Next(c)
					}
				}
				redraw = true
			} else {
			}
		}
		if statusMode {
			status.ShowLineColWordCount(c, e, filename)
		}
		if redraw {
			// redraw all characters
			h := int(c.Height())
			e.WriteLines(c, e.pos.Offset(), h+e.pos.Offset(), 0, 0)
			c.Draw()
			redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		if redraw || x != previousX || y != previousY {
			vt100.SetXY(uint(x), uint(y))
		}
		previousX = x
		previousY = y
	}
	tty.Close()
	vt100.Clear()
	vt100.Close()
}
