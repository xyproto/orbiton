package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/xyproto/vt100"
)

const versionString = "o 2.0.0"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.Red
		defaultEditorBackground       = vt100.BackgroundBlack
		defaultEditorStatusForeground = vt100.Black
		defaultEditorStatusBackground = vt100.BackgroundGray

		version = flag.Bool("version", false, "show version information")
		help    = flag.Bool("help", false, "show simple help")

		statusDuration = 3000 * time.Millisecond

		redraw   bool
		copyLine string
		bookmark Position
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
ctrl-l to redraw the screen
ctrl-k to delete characters to the end of the line, then delete the line
ctrl-g to show cursor positions, current letter and word count
ctrl-d to delete a single character
ctrl-t to toggle insert mode
ctrl-x to cut the current line
ctrl-c to copy the current line
ctrl-v to paste the current line
ctrl-b to bookmark the current position
ctrl-j to jump to the bookmark
ctrl-h to show a minimal help text
esc to toggle syntax highlighting
`)
		return
	}

	filename := flag.Arg(0)
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Need a filename.")
		os.Exit(1)
	}
	defaultHighlight := strings.Contains(filepath.Base(filename), ".")

	vt100.Init()

	c := vt100.NewCanvas()
	//c.HideCursor()
	c.ShowCursor()

	// 4 spaces per tab, scroll 10 lines at a time
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true, 10)

	status := NewStatusBar(defaultEditorStatusForeground, defaultEditorStatusBackground, e, statusDuration)

	// Try to load the filename, ignore errors since giving a new filename is also okay
	// TODO: Check if the file exists and add proper error reporting
	err := e.Load(filename)
	loaded := err == nil

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

	// Resize handler
	SetUpResizeHandler(c, e)

	// Undo buffer with room for 100 actions
	undo := NewUndo(100)

	tty, err := vt100.NewTTY()
	if err != nil {
		panic(err)
	}

	//tty.SetTimeout(10 * time.Millisecond)

	previousX := -1
	previousY := -1

	quit := false
	for !quit {
		key := tty.Key()
		switch key {
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			undo.Snapshot(c, e)
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
		case 20: // ctrl-t, toggle insert mode
			e.ToggleInsertMode()
			if e.InsertMode() {
				status.SetMessage("Insert mode")
			} else {
				status.SetMessage("Overwrite mode")
			}
			status.Show(c, e)
		case 7: // ctrl-g, status information
			currentRune := e.Rune()
			if !e.DrawMode() {
				status.SetMessage(fmt.Sprintf("line %d col %d unicode %U wordcount: %d undo count: %d", e.DataY(), e.pos.ScreenX(), currentRune, e.WordCount(), undo.Index()+1))
			} else if currentRune > 32 {
				x, _ := e.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %c (%U) wordcount: %d", e.pos.ScreenX(), e.pos.ScreenY(), x, e.DataY(), currentRune, currentRune, e.WordCount()))
			} else {
				x, _ := e.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %U wordcount: %d", e.pos.ScreenX(), e.pos.ScreenY(), x, e.DataY(), currentRune, e.WordCount()))
			}
			status.Show(c, e)
		case 252: // left arrow
			if !e.DrawMode() {
				e.Prev(c)
				if e.AfterLineContents() {
					e.End()
				}
				e.SaveX()
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
				e.SaveX()
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
		case 27: // esc, toggle highlight
			e.ToggleHighlight()
			redraw = true
		case 32: // space
			undo.Snapshot(c, e)
			// Place a space
			if !e.DrawMode() && e.InsertMode() {
				e.InsertRune(' ')
				redraw = true
			} else {
				e.SetRune(' ')
			}
			e.WriteRune(c)
			if e.DrawMode() {
				redraw = true
			}
			// Move to the next position
			if e.InsertMode() {
				e.Next(c)
			}
		case 13: // return
			undo.Snapshot(c, e)
			// if the current line is empty, insert a blank line
			dataCursor := e.DataCursor()
			//emptyLine := 0 == len(strings.TrimSpace(e.Line(dataCursor.Y)))
			if e.InsertMode() {
				e.FirstScreenPosition(e.DataY())
				if e.pos.AtStartOfLine() {
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineBelow()
					// Also move the cursor to the start, since it's now on a new blank line.
					e.pos.Down(c)
					e.Home()
				} else if e.BeforeOrAtStartOfText() {
					x := e.pos.ScreenX()
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineBelow()
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
			undo.Snapshot(c, e)
			if !e.DrawMode() && len(e.CurrentLine()) == 0 {
				e.DeleteLine(e.DataY())
				e.pos.Up()
				e.End()
			} else {
				// Move back
				e.Prev(c)
				// Type a blank
				e.SetRune(' ')
				e.WriteRune(c)
				if !e.DrawMode() {
					// Delete the blank
					e.Delete()
				}
			}
			redraw = true
		case 9: // tab
			undo.Snapshot(c, e)
			if !e.DrawMode() {
				// Place a tab
				if e.InsertMode() && !e.DrawMode() {
					e.InsertRune('\t')
				} else {
					e.SetRune('\t')
				}
				// Write the spaces that represent the tab
				e.WriteTab(c)
				// Move to the next position
				if e.InsertMode() {
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
			e.pos.SaveXRegardless()
		case 5: // ctrl-e, end
			e.End()
			e.pos.SaveXRegardless()
		case 4: // ctrl-d, delete
			undo.Snapshot(c, e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, e)
			} else {
				e.Delete()
				redraw = true
			}
		case 19: // ctrl-s, save
			err := e.Save(filename, !e.DrawMode())
			if err != nil {
				tty.Close()
				vt100.Close()
				fmt.Fprintln(os.Stderr, vt100.Red.Get(err.Error()))
				os.Exit(1)
			}
			// TODO: Go to the end of the document at this point, if needed
			// Lines may be trimmed for whitespace, so move to the end, if needed
			if !e.DrawMode() && e.AfterLineContents() {
				e.End()
			}
			// Status message
			status.SetMessage("Saved " + filename)
			status.Show(c, e)
			c.Draw()
			// Redraw after save, for syntax highlighting
			//redraw = true
		case 26: // ctrl-z, may background the application :/
			redraw = true
		case 21: // ctrl-u, undo
			if err := undo.Restore(c, e); err == nil {
				// no error
				c.Clear()
				c.Redraw()
				c.Draw()
				redraw = true
			} else {
				status.SetMessage("Undo error")
				status.Show(c, e)
			}
		case 12: // ctrl-l, redraw
			redraw = true
		case 11: // ctrl-k, delete to end of line
			undo.Snapshot(c, e)
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
			y := e.DataY()
			copyLine = e.Line(y)
			e.DeleteLine(y)
			redraw = true
		case 3: // ctrl-c, copy line
			copyLine = e.Line(e.DataY())
			redraw = true
		case 22: // ctrl-v, paste
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
				undo.Snapshot(c, e)
				// Place a letter
				if e.InsertMode() {
					e.InsertRune(rune(key))
				} else {
					e.SetRune(rune(key))
				}
				e.WriteRune(c)
				if e.InsertMode() {
					// Move to the next position
					e.Next(c)
				}
				redraw = true
			} else if key != 0 { // any other key
				// Place *something*
				r := rune(key)
				if e.InsertMode() {
					e.InsertRune(rune(key))
				} else {
					e.SetRune(rune(key))
				}
				e.WriteRune(c)
				if len(string(r)) > 0 {
					if e.InsertMode() {
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
			c.Draw()
			redraw = false
		} else if e.Changed() {
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
	vt100.Close()
}
