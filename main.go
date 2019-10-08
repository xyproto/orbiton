package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/xyproto/vt100"
)

const versionString = "red 1.3.0"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.Red
		defaultEditorBackground       = vt100.BackgroundBlack
		defaultEditorStatusForeground = vt100.Black
		defaultEditorStatusBackground = vt100.BackgroundGray

		// Color scheme for the "ASCII graphics" mode
		//defaultASCIIGraphicsForeground       = vt100.LightBlue
		//defaultASCIIGraphicsBackground       = vt100.BackgroundDefault
		//defaultASCIIGraphicsStatusForeground = vt100.White
		//defaultASCIIGraphicsStatusBackground = vt100.BackgroundMagenta

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
		fmt.Println(versionString + " - a very simple and limited text editor")
		fmt.Print(`
Hotkeys

ctrl-q to quit
ctrl-s to save
ctrl-h to toggle syntax highlighting for Go code
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
ctrl-z to undo
ctrl-x to cut the current line
ctrl-c to copy the current line
ctrl-v to paste the current line
ctrl-b to bookmark the current position
ctrl-j to jump to the bookmark

`)
		return
	}

	filename := flag.Arg(0)
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Please supply a filename.")
		os.Exit(1)
	}
	defaultHighlight := strings.Contains(filename, ".")

	vt100.Init()

	c := vt100.NewCanvas()
	//c.HideCursor()
	c.ShowCursor()

	// 4 spaces per tab, scroll 10 lines at a time
	e := NewEditor(4, defaultEditorForeground, defaultEditorBackground, defaultHighlight, true)

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
		status.SetMessage(versionString)
	}
	p := NewPosition(10, e)
	status.Show(c, p)
	c.Draw()

	// Resize handler
	SetUpResizeHandler(c, e, p)

	// Undo buffer with room for 1000 actions
	undo := NewUndo(1000)

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
		//case 27: // esc
		//	e.ToggleDrawMode()
		//	if !e.DrawMode() {
		//		e.SetColors(defaultEditorForeground, defaultEditorBackground)
		//		status.SetColors(defaultEditorStatusForeground, defaultEditorStatusBackground)
		//		c.FillBackground(e.bg)
		//		c.Draw()
		//		e.SetHighlight(defaultHighlight)
		//		e.SetInsertMode(true)
		//		status.SetMessage("Text edit mode")
		//		redraw = true
		//	} else {
		//		e.SetColors(defaultASCIIGraphicsForeground, defaultASCIIGraphicsBackground)
		//		status.SetColors(defaultASCIIGraphicsStatusForeground, defaultASCIIGraphicsStatusBackground)
		//		c.FillBackground(e.bg)
		//		c.Draw()
		//		e.SetHighlight(false)
		//		e.SetInsertMode(false)
		//		status.SetMessage("ASCII graphics mode")
		//		redraw = true
		//	}
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			undo.Snapshot(c, p, e)
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
			status.Show(c, p)
		case 7: // ctrl-g, status information
			currentRune := p.Rune()
			if !e.DrawMode() {
				status.SetMessage(fmt.Sprintf("line %d col %d unicode %U wordcount: %d undo count: %d", p.DataY(), p.ScreenX(), currentRune, e.WordCount(), undo.Index()+1))
			} else if currentRune > 32 {
				x, _ := p.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %c (%U) wordcount: %d", p.ScreenX(), p.ScreenY(), x, p.DataY(), currentRune, currentRune, e.WordCount()))
			} else {
				x, _ := p.DataX()
				status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %U wordcount: %d", p.ScreenX(), p.ScreenY(), x, p.DataY(), currentRune, e.WordCount()))
			}
			status.Show(c, p)
		case 252: // left arrow
			if !e.DrawMode() {
				p.Prev(c)
				if p.AfterLineContents() {
					p.End()
				}
				p.SaveX()
			} else {
				// Draw mode
				p.Left()
			}
		case 254: // right arrow
			if !e.DrawMode() {
				if p.DataY() < e.Len() {
					p.Next(c)
				}
				if p.AfterLineContents() {
					p.End()
				}
				p.SaveX()
			} else {
				// Draw mode
				p.Right(c)
			}
		case 253: // up arrow
			// Move the screen cursor
			if !e.DrawMode() {
				if p.DataY() > 0 {
					// Move the position up in the current screen
					if p.UpEnd(c) != nil {
						// If at the top, don't move up, but scroll the contents
						// Output a helpful message
						if p.DataY() == 0 {
							status.SetMessage("Start of text")
							status.Show(c, p)
						} else {
							//status.SetMessage("Top of screen, scroll with ctrl-p")
							//status.Show(c, p)
							redraw = p.ScrollUp(c, status, 1)
							p.Down(c)
							p.UpEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if p.AfterLineContents() {
						p.End()
					}
				} else {
					status.SetMessage("Start of text")
					status.Show(c, p)
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if p.AfterLineContents() {
					p.End()
				}
			} else {
				p.Up()
			}
		case 255: // down arrow
			if !e.DrawMode() {
				if p.DataY() < e.Len() {
					// Move the position down in the current screen
					if p.DownEnd(c) != nil {
						// If at the bottom, don't move down, but scroll the contents
						// Output a helpful message
						if p.EndOfDocument() {
							status.SetMessage("End of text")
							status.Show(c, p)
						} else {
							//status.SetMessage("Bottom of screen, scroll with ctrl-n")
							//status.Show(c, p)
							redraw = p.ScrollDown(c, status, 1)
							p.Up()
							p.DownEnd(c)
						}
					}
					// If the cursor is after the length of the current line, move it to the end of the current line
					if p.AfterLineContents() {
						p.End()
					}
				} else {
					status.SetMessage("End of text")
					status.Show(c, p)
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if p.AfterLineContents() {
					p.End()
				}
			} else {
				p.Down(c)
			}
		case 14: // ctrl-n, scroll down
			redraw = p.ScrollDown(c, status, p.scrollSpeed)
			if !e.DrawMode() && p.AfterLineContents() {
				p.End()
			}
		case 16: // ctrl-p, scroll up
			redraw = p.ScrollUp(c, status, p.scrollSpeed)
			if !e.DrawMode() && p.AfterLineContents() {
				p.End()
			}
		case 8: // ctrl-h, toggle highlight
			e.ToggleHighlight()
			redraw = true
		case 32: // space
			undo.Snapshot(c, p, e)
			// Place a space
			if !e.DrawMode() && e.InsertMode() {
				p.InsertRune(' ')
				redraw = true
			} else {
				p.SetRune(' ')
			}
			p.WriteRune(c)
			if e.DrawMode() {
				redraw = true
			}
			// Move to the next position
			if e.InsertMode() {
				p.Next(c)
			}
		case 13: // return
			undo.Snapshot(c, p, e)
			// if the current line is empty, insert a blank line
			dataCursor := p.DataCursor()
			//emptyLine := 0 == len(strings.TrimSpace(e.Line(dataCursor.Y)))
			if e.InsertMode() {
				p.e.FirstScreenPosition(p.DataY())
				if p.AtStartOfLine() {
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineBelow(p)
					// Also move the cursor to the start, since it's now on a new blank line.
					p.Down(c)
					p.Home()
				} else if p.BeforeOrAtStartOfText(e) {
					x := p.ScreenX()
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineBelow(p)
					// Also move the cursor to the start, since it's now on a new blank line.
					p.Down(c)
					p.SetX(x)
				} else {
					// Split the current line in two
					e.SplitLine(p)
					// Move to the start of the next line
					p.Down(c)
					p.Home()
				}
			} else {
				e.CreateLineIfMissing(dataCursor.Y + 1)
				p.Down(c)
				if !e.DrawMode() {
					p.Home()
				}
			}
			redraw = true
		case 127: // backspace
			undo.Snapshot(c, p, e)
			if !e.DrawMode() && len(p.Line()) == 0 {
				e.DeleteLine(p.DataY())
				p.Up()
				p.End()
			} else {
				// Move back
				p.Prev(c)
				// Type a blank
				p.SetRune(' ')
				p.WriteRune(c)
				if !e.DrawMode() {
					// Delete the blank
					e.Delete(p)
				}
			}
			redraw = true
		case 9: // tab
			undo.Snapshot(c, p, e)
			if !e.DrawMode() {
				// Place a tab
				if e.InsertMode() && !e.DrawMode() {
					p.InsertRune('\t')
				} else {
					p.SetRune('\t')
				}
				// Write the spaces that represent the tab
				p.WriteTab(c)
				// Move to the next position
				if e.InsertMode() {
					p.Next(c)
				}
			}
			redraw = true
		case 1: // ctrl-a, home
			// toggle between start of line and start of non-whitespace
			if p.AtStartOfLine() {
				p.SetX(p.e.FirstScreenPosition(p.DataY()))
			} else {
				p.Home()
			}
			p.SaveXRegardless()
		case 5: // ctrl-e, end
			p.End()
			p.SaveXRegardless()
		case 4: // ctrl-d, delete
			undo.Snapshot(c, p, e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, p)
			} else {
				e.Delete(p)
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
			if !e.DrawMode() && p.AfterLineContents() {
				p.End()
			}
			// Status message
			status.SetMessage("Saved " + filename)
			status.Show(c, p)
			c.Draw()
			// Redraw after save, for syntax highlighting
			//redraw = true
		case 26: // ctrl-z, undo
			if undoCanvas, undoPosition, undoEditor, err := undo.Back(); err == nil {
				// no error
				*c = *(undoCanvas)
				*p = *(undoPosition)
				*e = *(undoEditor)
				// link the position and editor structs
				p.e = e
				// redraw everything
				redraw = true
			}
		case 12: // ctrl-l, redraw
			redraw = true
		case 11: // ctrl-k, delete to end of line
			undo.Snapshot(c, p, e)
			if e.Empty() {
				status.SetMessage("Empty")
				status.Show(c, p)
			} else {
				e.DeleteRestOfLine(p)
				if !e.DrawMode() && p.EmptyLine() {
					// Deleting the rest of the line cleared this line,
					// so just remove it.
					e.DeleteLine(p.DataY())
				}
				vt100.Do("Erase End of Line")
				redraw = true
			}
		case 24: // ctrl-x, cut
			y := p.DataY()
			copyLine = e.Line(y)
			e.DeleteLine(y)
			redraw = true
		case 3: // ctrl-c, copy line
			copyLine = e.Line(p.DataY())
			redraw = true
		case 22: // ctrl-v, paste
			e.SetLine(p.DataY(), copyLine)
			redraw = true
		case 2: // ctrl-b, bookmark
			bookmark = *p
		case 10: // ctrl-j, jump to bookmark
			if bookmark.e != nil {
				*p = bookmark
				redraw = true
			}
		default:
			if (key >= 'a' && key <= 'z') || (key >= 'A' && key <= 'Z') { // letter
				undo.Snapshot(c, p, e)
				// Place a letter
				if e.InsertMode() {
					p.InsertRune(rune(key))
				} else {
					p.SetRune(rune(key))
				}
				p.WriteRune(c)
				if e.InsertMode() {
					// Move to the next position
					p.Next(c)
				}
				redraw = true
			} else if key != 0 { // any other key
				// Place *something*
				r := rune(key)
				if e.InsertMode() {
					p.InsertRune(rune(key))
				} else {
					p.SetRune(rune(key))
				}
				p.WriteRune(c)
				if len(string(r)) > 0 {
					if e.InsertMode() {
						// Move to the next position
						p.Next(c)
					}
				}
				redraw = true
			} else {
			}
		}
		if redraw {
			// redraw all characters
			h := int(c.Height())
			e.WriteLines(c, 0+p.Offset(), h+p.Offset(), 0, 0)
			c.Draw()
			redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		x := p.ScreenX()
		y := p.ScreenY()
		if x != previousX || y != previousY {
			vt100.SetXY(uint(x), uint(y))
		}
		previousY = x
		previousY = y
	}
	tty.Close()
	vt100.Close()
}
