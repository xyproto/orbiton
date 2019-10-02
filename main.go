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

const versionString = "red 1.2.1"

func main() {
	var (
		// Color scheme for the "text edit" mode
		defaultEditorForeground       = vt100.Red
		defaultEditorBackground       = vt100.BackgroundBlack
		defaultEditorStatusForeground = vt100.Black
		defaultEditorStatusBackground = vt100.BackgroundGray

		// Color scheme for the "ASCII graphics" mode
		defaultASCIIGraphicsForeground       = vt100.LightBlue
		defaultASCIIGraphicsBackground       = vt100.BackgroundDefault
		defaultASCIIGraphicsStatusForeground = vt100.White
		defaultASCIIGraphicsStatusBackground = vt100.BackgroundMagenta

		statusDuration = 2200 * time.Millisecond

		//offset = 0
		redraw = false

		version = flag.Bool("version", false, "show version information")
		help    = flag.Bool("help", false, "show simple help")
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
ctrl-h to toggle syntax highlighting for Go.
ctrl-f to format the current file with "go fmt" (but not save the result).
ctrl-a go to start of line
ctrl-e go to end of line
ctrl-p scroll up 10 lines
ctrl-n scroll down 10 lines
ctrl-l to redraw the screen
ctrl-k to delete characters to the end of the line, then delete the line
ctrl-g to show cursor positions, current letter and word count
ctrl-d to delete a single character
ctrl-j to toggle insert mode
ctrl-z to undo
esc to toggle "text edit mode" and "ASCII graphics mode"

`)
		return
	}

	filename := flag.Arg(0)
	if filename == "" {
		fmt.Fprintln(os.Stderr, "Please supply a filename.")
		os.Exit(1)
	}

	vt100.Init()
	vt100.ShowCursor(true)

	c := vt100.NewCanvas()

	defaultHighlight := strings.Contains(filename, ".")

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

	// Undo buffer with room for 1000 actions
	undo := NewUndo(1000)

	tty, err := vt100.NewTTY()
	if err != nil {
		panic(err)
	}
	tty.SetTimeout(10 * time.Millisecond)
	quit := false
	for !quit {
		key := tty.Key()
		switch key {
		case 27: // esc
			e.ToggleEOLMode()
			if e.EOLMode() {
				e.SetColors(defaultEditorForeground, defaultEditorBackground)
				status.SetColors(defaultEditorStatusForeground, defaultEditorStatusBackground)
				c.FillBackground(e.bg)
				e.SetHighlight(defaultHighlight)
				e.SetInsertMode(true)
				status.SetMessage("Text edit mode")
				redraw = true
			} else {
				e.SetColors(defaultASCIIGraphicsForeground, defaultASCIIGraphicsBackground)
				status.SetColors(defaultASCIIGraphicsStatusForeground, defaultASCIIGraphicsStatusBackground)
				c.FillBackground(e.bg)
				e.SetHighlight(false)
				e.SetInsertMode(false)
				status.SetMessage("ASCII graphics mode")
				redraw = true
			}
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			undo.Snapshot(c, p, e)
			if e.eolMode {
				// Use a globally unique tempfile
				f, err := ioutil.TempFile("/tmp", "_red*.go")
				if err == nil {
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
			}
		case 10: // ctrl-j, toggle insert mode
			e.ToggleInsertMode()
		case 7: // ctrl-g, status information
			currentRune := p.Rune()
			if e.EOLMode() {
				status.SetMessage(fmt.Sprintf("line %d col %d unicode %U wordcount: %d undo index: %d", p.DataY(), p.ViewX(), currentRune, e.WordCount(), undo.Index()))
			} else {
				if currentRune > 32 {
					status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %c (%U) wordcount: %d", p.ViewX(), p.ViewY(), p.DataX(), p.DataY(), currentRune, currentRune, e.WordCount()))
				} else {
					status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %U wordcount: %d", p.ViewX(), p.ViewY(), p.DataX(), p.DataY(), currentRune, e.WordCount()))
				}
			}
			status.Show(c, p)
		case 252: // left arrow
			p.Prev(c)
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
			p.SaveX()
		case 254: // right arrow
			if !e.EOLMode() || (e.EOLMode() && p.DataY() < e.Len()) {
				p.Next(c)
			}
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
			p.SaveX()
		case 253: // up arrow
			// Move the screen cursor
			if !e.EOLMode() || (e.EOLMode() && p.DataY() > 0) {
				// Move the position up in the current screen
				if p.UpEnd(c) != nil {
					// If at the top, don't move up, but scroll the contents
					// Output a helpful message
					if p.DataY() == 0 {
						status.SetMessage("Start of text")
					} else {
						//status.SetMessage("Top of screen, scroll with ctrl-p")
						redraw = p.ScrollUp(c, status, 1)
						p.Down(c)
						p.UpEnd(c)
					}
					status.Show(c, p)
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.EOLMode() && p.AfterLineContents() {
					p.End()
				}
			} else if e.EOLMode() {
				status.SetMessage("Start of text")
				status.Show(c, p)
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
		case 255: // down arrow
			if !e.EOLMode() || (e.EOLMode() && p.DataY() < e.Len()) {
				// Move the position down in the current screen
				if p.DownEnd(c) != nil {
					// If at the bottom, don't move down, but scroll the contents
					// Output a helpful message
					if p.EndOfDocument() {
						status.SetMessage("End of text")
					} else {
						//status.SetMessage("Bottom of screen, scroll with ctrl-n")
						redraw = p.ScrollDown(c, status, 1)
						p.Up()
						p.DownEnd(c)
					}
					status.Show(c, p)
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.EOLMode() && p.AfterLineContents() {
					p.End()
				}
			} else if e.EOLMode() {
				status.SetMessage("End of text")
				status.Show(c, p)
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
		case 14: // ctrl-n, scroll down
			redraw = p.ScrollDown(c, status, p.scrollSpeed)
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
		case 16: // ctrl-p, scroll up
			redraw = p.ScrollUp(c, status, p.scrollSpeed)
			if e.EOLMode() && p.AfterLineContents() {
				p.End()
			}
		case 8: // ctrl-h, toggle highlight
			e.ToggleHighlight()
			redraw = true
		case 32: // space
			undo.Snapshot(c, p, e)
			// Place a space
			if e.InsertMode() {
				p.InsertRune(' ')
				redraw = true
			} else {
				p.SetRune(' ')
			}
			p.WriteRune(c)
			// Move to the next position
			p.Next(c)
		case 13: // return
			undo.Snapshot(c, p, e)
			// if the current line is empty, insert a blank line
			dataCursor := p.DataCursor()
			//emptyLine := 0 == len(strings.TrimSpace(e.Line(dataCursor.Y)))
			if e.EOLMode() {
				if dataCursor.X >= (len(e.Line(dataCursor.Y)) - 1) {
					// Insert a new line at the current y position, then shift the rest down.
					p.Down(c)
					e.InsertLineBelow(p)
					// Also move the cursor to the start, since it's now on a new blank line
					p.Home()
				} else {
					// Insert a new line a the current y position, then shift the rest down.
					e.InsertLineBelow(p)
					// Also move the cursor to the start, since it's now on a new blank line.
					p.Down(c)
					p.Home()
				}
			} else {
				e.CreateLineIfMissing(dataCursor.Y + 1)
				p.Down(c)
			}
			redraw = true
		case 127: // backspace
			undo.Snapshot(c, p, e)
			if e.EOLMode() && len(p.Line()) == 0 {
				e.DeleteLine(p.DataY())
				p.Up()
				p.End()
			} else {
				// Move back
				p.Prev(c)
				// Type a blank
				p.SetRune(' ')
				p.WriteRune(c)
				if e.EOLMode() {
					// Delete the blank
					e.Delete(p)
				}
			}
			redraw = true
		case 9: // tab
			undo.Snapshot(c, p, e)
			if e.EOLMode() {
				// Place a tab
				if e.InsertMode() {
					p.InsertRune('\t')
				} else {
					p.SetRune('\t')
				}
				// Write the spaces that represent the tab
				p.WriteTab(c)
				// Move to the next position
				p.Next(c)
			}
			redraw = true
		case 1: // ctrl-a, home
			p.Home()
		case 5: // ctrl-e, end
			p.End()
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
			err := e.Save(filename, e.eolMode)
			if err != nil {
				tty.Close()
				vt100.Close()
				fmt.Fprintln(os.Stderr, vt100.Red.Get(err.Error()))
				os.Exit(1)
			}
			// TODO: Go to the end of the document at this point, if needed
			// Lines may be trimmed for whitespace, so move to the end, if needed
			if e.EOLMode() && p.AfterLineContents() {
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
				if e.EOLMode() && p.EmptyLine() {
					// Deleting the rest of the line cleared this line,
					// so just remove it.
					e.DeleteLine(p.DataY())
				}
				vt100.Do("Erase End of Line")
				redraw = true
			}
		default:
			if (key >= 'a' && key <= 'z') || (key >= 'A' && key <= 'Z') { // letter
				undo.Snapshot(c, p, e)
				// Place a letter
				if e.InsertMode() {
					p.InsertRune(rune(key))
					redraw = true
				} else {
					p.SetRune(rune(key))
				}
				p.WriteRune(c)
				// Move to the next position
				p.Next(c)
			} else if key != 0 { // any other key
				// Place *something*
				r := rune(key)
				if e.InsertMode() {
					p.InsertRune(rune(key))
					redraw = true
				} else {
					p.SetRune(rune(key))
				}
				p.WriteRune(c)
				if len(string(r)) > 0 {
					// Move to the next position
					p.Next(c)
				}
			}
		}
		if redraw {
			// redraw all characters
			h := int(c.Height())
			e.WriteLines(c, 0+p.Offset(), h+p.Offset(), 0, 0)
			c.Draw()
			//status.Show(c, p)
			redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		vt100.SetXY(uint(p.ViewX()), uint(p.ViewY()))
	}
	tty.Close()
	vt100.Close()
}
