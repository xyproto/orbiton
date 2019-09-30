package main

import (
	"flag"
	"fmt"
	"github.com/xyproto/vt100"
	"os"
	"os/exec"
	"time"
)

const versionString = "red 1.0.0"

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
ctrl-a go to start of line
ctrl-e go to end of line
ctrl-p scroll up 10 lines
ctrl-n scroll down 10 lines
ctrl-l to redraw the screen
ctrl-k to delete characters to the end of the line
ctrl-s to save (don't use this on files you care about!)
ctrl-g to show cursor positions, current letter and word count
ctrl-d to delete a single character
ctrl-f to format the current file with "go fmt" (but not save the result).
ctrl-h to toggle syntax highlighting for Go.
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

	// 4 spaces per tab, scroll 10 lines at a time
	e := NewEditor(4, 10, defaultEditorForeground, defaultEditorBackground, false)

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
	p := &Position{}
	status.Show(c, p)
	c.Draw()

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
				status.SetMessage("Text edit mode")
				redraw = true
			} else {
				e.SetColors(defaultASCIIGraphicsForeground, defaultASCIIGraphicsBackground)
				status.SetColors(defaultASCIIGraphicsStatusForeground, defaultASCIIGraphicsStatusBackground)
				c.FillBackground(e.bg)
				status.SetMessage("ASCII graphics mode")
				redraw = true
			}
		case 17: // ctrl-q, quit
			quit = true
		case 6: // ctrl-f
			if e.eolMode {
				err := e.Save("/tmp/_tmp.go", true)
				if err == nil {
					cmd := exec.Command("/usr/bin/gofmt", "-w", "/tmp/_tmp.go")
					err = cmd.Run()
					if err == nil {
						e.Load("/tmp/_tmp.go")
					}
					cmd = exec.Command("/usr/bin/rm", "-f", "/tmp/_tmp.go")
					_ = cmd.Run()
				}
				redraw = true
			}
		case 7: // ctrl-g, status information
			dataCursor := p.DataCursor(e)
			currentRune := p.Rune(e)
			if e.EOLMode() {
				status.SetMessage(fmt.Sprintf("line %d col %d unicode %U wordcount: %d", dataCursor.Y, p.X(), currentRune, e.WordCount()))
			} else {
				if currentRune > 32 {
					status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %c (%U) wordcount: %d", p.X(), p.Y(), dataCursor.X, dataCursor.Y, currentRune, currentRune, e.WordCount()))
				} else {
					status.SetMessage(fmt.Sprintf("%d,%d (data %d,%d) %U wordcount: %d", p.X(), p.Y(), dataCursor.X, dataCursor.Y, currentRune, e.WordCount()))
				}
			}
			status.Show(c, p)
		case 252: // left arrow
			p.Prev(c, e)
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 254: // right arrow
			p.Next(c, e)
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 253: // up arrow
			dataCursor := p.DataCursor(e)
			// Move the screen cursor
			if p.Y() == 0 {
				// If at the top, don't move up, but scroll the contents
				// Output a helpful message
				if dataCursor.Y == 0 {
					status.SetMessage("Start of text")
				} else {
					//status.SetMessage("Top of screen, scroll with ctrl-p")
					redraw = p.ScrollUp(c, status, e, 1)
				}
				status.Show(c, p)
			} else {
				// Move the data cursor
				p.Up()
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 255: // down arrow
			dataCursor := p.DataCursor(e)
			if !e.EOLMode() || (e.EOLMode() && dataCursor.Y < e.Len()) {
				// Move the position down in the current screen
				err := p.Down(c)
				if err != nil {
					// If at the bottom, don't move down, but scroll the contents
					// Output a helpful message
					if p.EndOfDocument(e) {
						status.SetMessage("End of text")
					} else {
						//status.SetMessage("Bottom of screen, scroll with ctrl-n")
						redraw = p.ScrollDown(c, status, e, 1)
					}
					status.Show(c, p)
				}
				// If the cursor is after the length of the current line, move it to the end of the current line
				if e.EOLMode() && p.AfterLineContents(e) {
					p.End(e)
				}
			} else if e.EOLMode() {
				status.SetMessage("End of text")
				status.Show(c, p)
			}
			// If the cursor is after the length of the current line, move it to the end of the current line
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 14: // ctrl-n, scroll down
			redraw = p.ScrollDown(c, status, e, e.scrollSpeed)
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 16: // ctrl-p, scroll up
			redraw = p.ScrollUp(c, status, e, e.scrollSpeed)
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
		case 8: // ctrl-h, toggle highlight
			e.ToggleHighlight()
			redraw = true
		case 32: // space
			// Place a space
			p.SetRune(e, ' ')
			p.WriteRune(c, e)
			// Move to the next position
			p.Next(c, e)
		case 13: // return
			dataCursor := p.DataCursor(e)
			e.CreateLineIfMissing(dataCursor.Y + 1)
			// Move down and home
			p.Down(c)
			p.Home(e)
		case 127: // backspace
			// Move back
			p.Prev(c, e)
			// Type a blank
			p.SetRune(e, ' ')
			p.WriteRune(c, e)
			// Delete the blank
			dataCursor := p.DataCursor(e)
			e.Delete(dataCursor.X, dataCursor.Y)
		case 9: // tab
			// Place a tab
			p.SetRune(e, '\t')
			// Write the spaces that represent the tab
			p.WriteTab(c, e)
			// Move to the next position
			p.Next(c, e)
		case 1: // ctrl-a, home
			p.Home(e)
		case 5: // ctrl-e, end
			p.End(e)
		case 4: // ctrl-d, delete
			dataCursor := p.DataCursor(e)
			e.Delete(dataCursor.X, dataCursor.Y)
			redraw = true
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
			if e.EOLMode() && p.AfterLineContents(e) {
				p.End(e)
			}
			// Status message
			status.SetMessage("Saved " + filename)
			status.Show(c, p)
			c.Draw()
			// Redraw after save, for syntax highlighting
			redraw = true
		case 12: // ctrl-l, redraw
			redraw = true
		case 11: // ctrl-k, delete to end of line
			dataCursor := p.DataCursor(e)
			e.DeleteRestOfLine(dataCursor.X, dataCursor.Y)
			vt100.Do("Erase End of Line")
			redraw = true
		default:
			if (key >= 'a' && key <= 'z') || (key >= 'A' && key <= 'Z') { // letter
				// Place a letter
				//e.Insert(p, rune(key))
				p.SetRune(e, rune(key))
				p.WriteRune(c, e)
				// Move to the next position
				p.Next(c, e)
			} else if key != 0 { // any other key
				// Place *something*
				r := rune(key)
				p.SetRune(e, r)
				p.WriteRune(c, e)
				if len(string(r)) > 0 {
					// Move to the next position
					p.Next(c, e)
				}
			}
		}
		if redraw {
			// redraw all characters
			h := int(c.Height())
			e.WriteLines(c, 0+p.Offset(), h+p.Offset(), 0, 0)
			c.Draw()
			status.Show(c, p)
			redraw = false
		} else if e.Changed() {
			c.Draw()
		}
		vt100.SetXY(uint(p.X()), uint(p.Y()))
	}
	tty.Close()
	vt100.Close()
}
