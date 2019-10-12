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

const versionString = "o 2.2.0"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.Red
		defaultEditorBackground       = vt100.BackgroundBlack
		defaultEditorStatusForeground = vt100.Blue
		defaultEditorStatusBackground = vt100.BackgroundGray

		version = flag.Bool("version", false, "show version information")
		help    = flag.Bool("help", false, "show simple help")

		statusDuration = 2700 * time.Millisecond

		redraw    bool     // if the contents should be redrawn in the next loop
		copyLine  string   // for the cut/copy/paste functionality
		bookmark  Position // for the bookmark/jump functionality
		wordcount bool     // always show wordcount at the bottom?
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
ctrl-g to show cursor positions, current letter and word count
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
ctrl-w to show a word counter
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
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true, 10)

	status := NewStatusBar(defaultEditorStatusForeground, defaultEditorStatusBackground, e, statusDuration)

	// Try to load the filename, ignore errors since giving a new filename is also okay
	loaded := e.Load(filename) == nil

	// Draw editor lines from line 0 uE to h onto the canvas at 0,0
	h := int(c.Height())
	e.WriteLines(c, 0, h, 0, 0)

	// Friendly status message
	if loaded {
		status.SetMessage("Loaded " + filename)
	} else {
		status.SetMessage("New " + filename)
	}
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
		redraw = e.GoTo(lineNumber, c, status)
	}

	quit := false
	for !quit {
		key := tty.Key()
		switch key {
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			undo.Snapshot(e)
			// Use a globally unique tempfile
			if f, err := ioutil.TempFile("/tmp", "_red*.go"); !e.DrawMode() && err == nil {
				// no error, everyting is fine
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
		case 23: // ctrl-w, always show word count
			// Enter writers mode. There is no escape.
			wordcount = true
			status.ShowWordCount(c, e)
			// Writers mode, green on black
			e.fg = vt100.LightGreen
			e.bg = vt100.BackgroundDefault
			redraw = true
		case 18: // ctrl-r, toggle draw mode
			e.ToggleDrawMode()
			if e.DrawMode() {
				status.SetMessage("Draw mode")
			} else {
				status.SetMessage("Text mode")
			}
			status.Show(c, e)
		case 7: // ctrl-g, status information
			currentRune := e.Rune()
			if !e.DrawMode() {
				status.SetMessage(fmt.Sprintf("line %d col %d rune %U word count: %d", e.DataY(), e.pos.ScreenX(), currentRune, e.WordCount()))
			} else if currentRune > 32 {
				x, _ := e.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %c (%U) word count: %d", e.pos.ScreenX(), e.pos.ScreenY(), x, e.DataY(), currentRune, currentRune, e.WordCount()))
			} else {
				x, _ := e.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %U word count: %d", e.pos.ScreenX(), e.pos.ScreenY(), x, e.DataY(), currentRune, e.WordCount()))
			}
			status.Show(c, e)
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
						// If at the top, don't move up, but scroll the contents
						// Output a helpful message
						if e.DataY() == 0 {
							//status.SetMessage("Start of text")
							//status.Show(c, e)
						} else {
							//status.SetMessage("Top of screen, scroll with ctrl-p")
							//status.Show(c, e)
							redraw = e.ScrollUp(c, status, 1)
							e.pos.Down(c)
							e.UpEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if e.AfterLineContents() {
						e.End()
					}
				} else {
					//status.SetMessage("Start of text")
					//status.Show(c, e)
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
						if e.EndOfDocument() {
							//status.SetMessage("End of text")
							//status.Show(c, e)
						} else {
							redraw = e.ScrollDown(c, status, 1)
							e.pos.Up()
							e.DownEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if e.AfterLineContents() {
						e.End()
					}
				} else {
					//status.SetMessage("End of text")
					//status.Show(c, e)
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
			dataCursor := e.DataCursor()
			//emptyLine := 0 == len(strings.TrimSpace(e.Line(dataCursor.Y)))
			if !e.DrawMode() {
				e.FirstScreenPosition(e.DataY())
				if e.pos.AtStartOfLine() {
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineAbove()
					// Also move the cursor to the start, since it's now on a new blank line.
					e.pos.Down(c)
					e.Home()
				} else if e.BeforeOrAtStartOfText() {
					x := e.pos.ScreenX()
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineAbove()
					// Also move the cursor to the start, since it's now on a new blank line.
					e.pos.Down(c)
					e.pos.SetX(x)
				} else {
					// Split the current line in two
					e.SplitLine()
					// Move to the start of the next line
					e.pos.Down(c)
					e.Home()
				}
			} else {
				e.CreateLineIfMissing(dataCursor.Y + 1)
				e.pos.Down(c)
				if !e.DrawMode() {
					e.Home()
				}
			}
			redraw = true
		case 127: // backspace
			undo.Snapshot(e)
			if !e.DrawMode() && e.EmptyLine() {
				e.DeleteLine(e.DataY())
				e.pos.Up()
				e.End()
			} else if !e.DrawMode() && e.pos.AtStartOfLine() {
				if e.DataY() > 0 {
					e.pos.Up()
					e.End()
					e.Delete()
				}
			} else {
				// Move back
				e.Prev(c)
				// Type a blank
				e.SetRune(' ')
				e.WriteRune(c)
				if !e.DrawMode() && !e.AtEndOfLine() {
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
		case 21, 26: // ctrl-u or ctrl-z, undo (ctrl-z may beckground the application)
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
					redraw = e.GoTo(ln, c, status)
					status.SetMessage(lns)
					status.Show(c, e)
					redraw = true
				}
			}
		case 11: // ctrl-k, delete to end of line
			undo.Snapshot(e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.DeleteRestOfLine()
				if !e.DrawMode() && e.EmptyLine() {
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
		if redraw {
			// redraw all characters
			h := int(c.Height())
			e.WriteLines(c, e.pos.Offset(), h+e.pos.Offset(), 0, 0)
			if wordcount {
				status.ShowWordCount(c, e)
			}
			c.Draw()
			redraw = false
		} else if e.Changed() {
			if wordcount {
				status.ShowWordCount(c, e)
			}
			c.Draw()
		}
		x := e.pos.ScreenX()
		y := e.pos.ScreenY()
		if x != previousX || y != previousY {
			vt100.SetXY(uint(x), uint(y))
		}
		previousY = x
		previousY = y
	}
	tty.Close()
	vt100.Clear()
	vt100.Close()
}
