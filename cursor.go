package main

import (
	"errors"
	"github.com/xyproto/vt100"
)

type Cursor struct {
	X, Y int
}

type Position struct {
	sx     int // the position of the cursor in the current scrollview
	sy     int // the position of the cursor in the current scrollview
	scroll int // how far one has scrolled
}

func position2datacursor(p *Position, e *Editor) *Cursor {
	if !e.eolMode {
		return &Cursor{p.sx, p.scroll + p.sy}
	}
	var dataX int
	// the y position in the data is the lines scrolled + current screen cursor Y position
	dataY := p.scroll + p.sy
	// get the current line of text
	line := e.Line(dataY)
	screenCounter := 0 // counter for the characters on the screen
	// loop, while also keeping track of tab expansion
	// add a space to allow to jump to the position after the line and get a valid data position
	for i, r := range line + " " {
		// When we reached the correct screen position, use i as the data position
		if screenCounter == p.sx {
			dataX = i
			break
		}
		// Increase the conter, based on the current rune
		if r == '\t' {
			screenCounter += e.spacesPerTab
		} else {
			screenCounter++
		}
	}
	// Return the data cursor
	return &Cursor{dataX, dataY}
}

func (p *Position) X() int {
	return p.sx
}

func (p *Position) Y() int {
	return p.sy
}

// Scroll offset
func (p *Position) Offset() int {
	return p.scroll
}

func (p *Position) DataCursor(e *Editor) *Cursor {
	return position2datacursor(p, e)
}

func (p *Position) ScreenCursor() *Cursor {
	return &Cursor{p.sx, p.sy}
}

func (p *Position) SetScreenScursor(c *Cursor) {
	p.sx = c.X
	p.sy = c.Y
}

func (p *Position) SetX(x int) {
	p.sx = x
}

func (p *Position) SetY(y int) {
	p.sy = y
}

func (p *Position) SetOffset(offset int) {
	p.scroll = offset
}

// Set the rune at the current data position
func (p *Position) SetRune(e *Editor, r rune) {
	dataCursor := p.DataCursor(e)
	e.Set(dataCursor.X, dataCursor.Y, r)
}

// Get the rune at the current data position
func (p *Position) Rune(e *Editor) rune {
	dataCursor := p.DataCursor(e)
	return e.Get(dataCursor.X, dataCursor.Y)
}

// Get the current line
func (p *Position) Line(e *Editor) string {
	dataCursor := p.DataCursor(e)
	return e.Line(dataCursor.Y)
}

// Move the cursor the the start of the line (x 0)
func (p *Position) Home(e *Editor) {
	p.sx = 0
}

// Move the cursor to the position right after the end of the cirrent line contents
func (p *Position) End(e *Editor) {
	dataCursor := p.DataCursor(e)
	p.sx = e.LastScreenPosition(dataCursor.Y, e.spacesPerTab) + 1
}

// Move to the next position in the contents
func (p *Position) Next(c *vt100.Canvas, e *Editor) error {
	dataCursor := p.DataCursor(e)
	atTab := e.eolMode && ('\t' == e.Get(dataCursor.X, dataCursor.Y))
	if atTab {
		p.sx += e.spacesPerTab
	} else {
		p.sx++
	}
	// Did we move too far on this line?
	w := int(c.W())
	if (e.eolMode && p.AfterLineContentsPlusOne(e)) || (!e.eolMode && p.sx >= w) {
		// Undo the move
		if atTab {
			p.sx -= e.spacesPerTab
		} else {
			p.sx--
		}
		// Move down
		if e.eolMode {
			err := p.Down(c)
			if err != nil {
				return err
			}
			// Move to the start of the line
			p.sx = 0
		}
	}
	return nil
}

// Move to the previous position in the contents
func (p *Position) Prev(c *vt100.Canvas, e *Editor) error {
	dataCursor := p.DataCursor(e)
	atTab := false
	if dataCursor.X > 0 {
		atTab = e.eolMode && ('\t' == e.Get(dataCursor.X-1, dataCursor.Y))
	}
	// If at a tab character, move a few more posisions
	if atTab {
		p.sx -= e.spacesPerTab
	} else {
		p.sx--
	}
	// Did we move too far?
	if p.sx < 0 {
		// Undo the move
		if atTab {
			p.sx += e.spacesPerTab
		} else {
			p.sx++
		}
		// Move up, and to the end of the line above, if in EOL mode
		if e.eolMode {
			err := p.Up()
			if err != nil {
				return err
			}
			p.End(e)
		}
	}
	return nil
}

func (p *Position) Up() error {
	if p.sy <= 0 {
		return errors.New("already at the top of the canvas")
	}
	p.sy--
	return nil
}

func (p *Position) Down(c *vt100.Canvas) error {
	if p.sy >= int(c.H()-1) {
		return errors.New("already at the bottom of the canvas")
	}
	p.sy++
	return nil
}

func (p *Position) ScrollDown(c *vt100.Canvas, status *StatusBar, e *Editor, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed
	// last y posision in the canvas
	canvasLastY := int(c.H() - 1)
	// number of lines in the document
	l := e.Len()
	if p.scroll >= e.Len()-canvasLastY {
		// Status message
		status.SetMessage("End of text")
		status.Show(c, p)
		c.Draw()
		// Don't redraw
		return false
	}
	status.Clear(c)
	if (p.scroll + canScroll) >= (l - canvasLastY) {
		// Almost at the bottom, we can scroll the remaining lines
		canScroll = (l - canvasLastY) - p.scroll
	}
	// Move the scroll offset
	p.scroll += canScroll
	// Prepare to redraw
	return true
}

func (p *Position) ScrollUp(c *vt100.Canvas, status *StatusBar, e *Editor, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed
	if p.scroll == 0 {
		// Can't scroll further up
		// Status message
		status.SetMessage("Start of text")
		status.Show(c, p)
		c.Draw()
		// Don't redraw
		return false
	}
	status.Clear(c)
	if p.scroll-canScroll < 0 {
		// Almost at the top, we can scroll the remaining lines
		canScroll = p.scroll
	}
	// Move the scroll offset
	p.scroll -= canScroll
	// Prepare to redraw
	return true
}

// EndOfDocument is true if we're at the last line of the document (or beyond)
func (p *Position) EndOfDocument(e *Editor) bool {
	dataCursor := p.DataCursor(e)
	return dataCursor.Y >= (e.Len() - 1)
}

// StartOfDocument is true if we're at the first line of the document
func (p *Position) StartOfDocument() bool {
	return p.sy == 0 && p.scroll == 0
}

// Check if the cursor is after the current line contents
func (p *Position) AfterLineContents(e *Editor) bool {
	dataCursor := p.DataCursor(e)
	return p.sx > e.LastScreenPosition(dataCursor.Y, e.spacesPerTab)
	//return dataCursor.X > e.LastDataPosition(dataCursor.Y)
}

func (p *Position) AfterLineContentsPlusOne(e *Editor) bool {
	dataCursor := p.DataCursor(e)
	return p.sx > (e.LastScreenPosition(dataCursor.Y, e.spacesPerTab) + 1)
	//return dataCursor.X > e.LastDataPosition(dataCursor.Y)
}

// WriteRune writes the current rune to the given canvas
func (p *Position) WriteRune(c *vt100.Canvas, e *Editor) {
	c.WriteRune(uint(p.sx), uint(p.sy), e.fg, e.bg, p.Rune(e))
}

// WriteTab writes spaces when there is a tab character, to the canvas
func (p *Position) WriteTab(c *vt100.Canvas, e *Editor) {
	for x := p.sx; x < p.sx+e.spacesPerTab; x++ {
		c.WriteRune(uint(x), uint(p.sy), e.fg, e.bg, ' ')
	}
}
