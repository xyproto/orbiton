package main

import (
	"errors"
	"strings"
	"unicode"

	"github.com/xyproto/vt100"
)

// Position represents a position on the screen, including how far down the view has scrolled
type Position struct {
	sx          int     // the position of the cursor in the current scrollview
	sy          int     // the position of the cursor in the current scrollview
	scroll      int     // how far one has scrolled
	scrollSpeed int     // how many lines to scroll, when scrolling
	e           *Editor // needed for examining the underlying data
	savedX      int     // for smart down cursor movement
}

// NewPosition returns a new Position struct
func NewPosition(scrollSpeed int, e *Editor) *Position {
	return &Position{0, 0, 0, scrollSpeed, e, 0}
}

// DataX will return the X position in the data (as opposed to the X position in the viewport)
func (p *Position) DataX() (int, error) {
	if p.e.DrawMode() {
		return p.sx, nil //+ strings.Count(p.e.Line(p.sy), "\t")*(p.e.spacesPerTab-1)
	}
	// the y position in the data is the lines scrolled + current screen cursor Y position
	dataY := p.scroll + p.sy
	// get the current line of text
	screenCounter := 0 // counter for the characters on the screen
	// loop, while also keeping track of tab expansion
	// add a space to allow to jump to the position after the line and get a valid data position
	found := false
	dataX := 0
	runeCounter := 0
	for _, r := range p.e.lines[dataY] {
		// When we reached the correct screen position, use i as the data position
		if screenCounter == p.sx {
			dataX = runeCounter
			found = true
			break
		}
		// Increase the counter, based on the current rune
		if r == '\t' {
			screenCounter += p.e.spacesPerTab
		} else {
			screenCounter++
		}
		runeCounter++
	}
	if !found {
		return runeCounter, errors.New("position is after data")
	}
	// Return the data cursor
	return dataX, nil
}

// DataY will return the Y position in the data (as opposed to the Y position in the viewport)
func (p *Position) DataY() int {
	return p.scroll + p.sy
}

// ScreenX returns the screen X position in the current view
func (p *Position) ScreenX() int {
	return p.sx
}

// ScreenY returns the screen Y position in the current view
func (p *Position) ScreenY() int {
	return p.sy
}

// Offset returns the scroll offset for the current view
func (p *Position) Offset() int {
	return p.scroll
}

// DataCursor returns the (x,y) position in the underlying data
func (p *Position) DataCursor() *Cursor {
	x, _ := p.DataX()
	return &Cursor{x, p.DataY()}
}

// SetX will set the screen X position
func (p *Position) SetX(x int) {
	p.sx = x
}

// SetY will set the screen Y position
func (p *Position) SetY(y int) {
	p.sy = y
}

// SetOffset will set the screen scolling offset
func (p *Position) SetOffset(offset int) {
	p.scroll = offset
}

// SetRune will set a rune at the current data position
func (p *Position) SetRune(r rune) {
	dataCursor := p.DataCursor()
	p.e.Set(dataCursor.X, dataCursor.Y, r)
}

// InsertRune will insert a rune at the current data position
func (p *Position) InsertRune(r rune) {
	p.e.Insert(p, r)
}

// Rune will get the rune at the current data position
func (p *Position) Rune() rune {
	dataCursor := p.DataCursor()
	return p.e.Get(dataCursor.X, dataCursor.Y)
}

// Line will get the current data line, as a string
func (p *Position) Line() string {
	dataCursor := p.DataCursor()
	return p.e.Line(dataCursor.Y)
}

// Home will move the cursor the the start of the line (x = 0)
func (p *Position) Home() {
	p.sx = 0
}

// End will move the cursor to the position right after the end of the cirrent line contents
func (p *Position) End() {
	p.sx = p.e.LastScreenPosition(p.DataY()) + 1
}

// DownEnd will move down and then choose a "smart" X position
func (p *Position) DownEnd(c *vt100.Canvas) error {
	tmpx := p.sx
	err := p.Down(c)
	if err != nil {
		return err
	}
	if p.AfterLineContentsPlusOne() && tmpx > 1 {
		p.End()
		if p.sx != tmpx && p.sx > p.savedX {
			p.savedX = tmpx
		}
	} else {
		p.sx = p.savedX
		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if p.Rune() == '\t' {
			p.sx = p.e.FirstScreenPosition(p.DataY())
		}
	}
	return nil
}

// UpEnd will move up and then choose a "smart" X position
func (p *Position) UpEnd(c *vt100.Canvas) error {
	tmpx := p.sx
	err := p.Up()
	if err != nil {
		return err
	}
	if p.AfterLineContentsPlusOne() && tmpx > 1 {
		p.End()
		if p.sx != tmpx && p.sx > p.savedX {
			p.savedX = tmpx
		}
	} else {
		p.sx = p.savedX
		// Also checking if e.Rune() is ' ' is nice for code, but horrible for regular text files
		if p.Rune() == '\t' {
			p.sx = p.e.FirstScreenPosition(p.DataY())
		}
	}
	return nil
}

// Next will move the cursor to the next position in the contents
func (p *Position) Next(c *vt100.Canvas) error {
	dataCursor := p.DataCursor()
	atTab := '\t' == p.e.Get(dataCursor.X, dataCursor.Y)
	if atTab && !p.e.DrawMode() {
		p.sx += p.e.spacesPerTab
	} else {
		p.sx++
	}
	// Did we move too far on this line?
	w := int(c.W())
	if (!p.e.DrawMode() && p.AfterLineContentsPlusOne()) || (p.e.DrawMode() && p.sx >= w) {
		// Undo the move
		if atTab && !p.e.DrawMode() {
			p.sx -= p.e.spacesPerTab
		} else {
			p.sx--
		}
		// Move down
		if !p.e.DrawMode() {
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

// Prev will move the cursor to the previous position in the contents
func (p *Position) Prev(c *vt100.Canvas) error {
	dataCursor := p.DataCursor()
	atTab := false
	if dataCursor.X > 0 {
		atTab = '\t' == p.e.Get(dataCursor.X-1, dataCursor.Y)
	}
	// If at a tab character, move a few more posisions
	if atTab && !p.e.DrawMode() {
		p.sx -= p.e.spacesPerTab
	} else {
		p.sx--
	}
	// Did we move too far?
	if p.sx < 0 {
		// Undo the move
		if atTab && !p.e.DrawMode() {
			p.sx += p.e.spacesPerTab
		} else {
			p.sx++
		}
		// Move up, and to the end of the line above, if in EOL mode
		if !p.e.DrawMode() {
			err := p.Up()
			if err != nil {
				return err
			}
			p.End()
		}
	}
	return nil
}

// Right will move the cursor to the right, if possible.
// It will not move the cursor up or down.
func (p *Position) Right(c *vt100.Canvas) {
	lastX := int(c.Width() - 1)
	if p.sx < lastX {
		p.sx++
	}
}

// Left will move the cursor to the left, if possible.
// It will not move the cursor up or down.
func (p *Position) Left() {
	if p.sx > 0 {
		p.sx--
	}
}

// SaveX will save the current X position, if it's within reason
func (p *Position) SaveX() {
	if !p.AfterLineContentsPlusOne() && p.sx > 1 {
		p.savedX = p.sx
	}
}

// SaveXRegardless will save the current X position, regardless
func (p *Position) SaveXRegardless() {
	p.savedX = p.sx
}

// Up will move the cursor up
func (p *Position) Up() error {
	if p.sy <= 0 {
		return errors.New("already at the top of the canvas")
	}
	p.sy--
	return nil
}

// Down will move the cursor down
func (p *Position) Down(c *vt100.Canvas) error {
	if p.sy >= int(c.H()-1) {
		return errors.New("already at the bottom of the canvas")
	}
	p.sy++
	return nil
}

// ScrollDown will scroll down the given amount of lines given in scrollSpeed
func (p *Position) ScrollDown(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
	// Find out if we can scroll scrollSpeed, or less
	canScroll := scrollSpeed
	// last y posision in the canvas
	canvasLastY := int(c.H() - 1)
	// number of lines in the document
	l := p.e.Len()
	if p.scroll >= p.e.Len()-canvasLastY {
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

// ScrollUp will scroll down the given amount of lines given in scrollSpeed
func (p *Position) ScrollUp(c *vt100.Canvas, status *StatusBar, scrollSpeed int) bool {
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
func (p *Position) EndOfDocument() bool {
	dataCursor := p.DataCursor()
	return dataCursor.Y >= (p.e.Len() - 1)
}

// StartOfDocument is true if we're at the first line of the document
func (p *Position) StartOfDocument() bool {
	return p.sy == 0 && p.scroll == 0
}

// AfterLineContents will check if the cursor is after the current line contents
func (p *Position) AfterLineContents() bool {
	dataCursor := p.DataCursor()
	return p.sx > p.e.LastScreenPosition(dataCursor.Y)
	//return dataCursor.X > e.LastDataPosition(dataCursor.Y)
}

// AfterLineContentsPlusOne will check if the cursor is after the current line contents, with a margin of 1
func (p *Position) AfterLineContentsPlusOne() bool {
	dataCursor := p.DataCursor()
	return p.sx > (p.e.LastScreenPosition(dataCursor.Y) + 1)
	//return dataCursor.X > e.LastDataPosition(dataCursor.Y)
}

// WriteRune writes the current rune to the given canvas
func (p *Position) WriteRune(c *vt100.Canvas) {
	c.WriteRune(uint(p.sx), uint(p.sy), p.e.fg, p.e.bg, p.Rune())
}

// WriteTab writes spaces when there is a tab character, to the canvas
func (p *Position) WriteTab(c *vt100.Canvas) {
	spacesPerTab := p.e.spacesPerTab
	if p.e.DrawMode() {
		spacesPerTab = 1
	}
	for x := p.sx; x < p.sx+spacesPerTab; x++ {
		c.WriteRune(uint(x), uint(p.sy), p.e.fg, p.e.bg, ' ')
	}
}

// EmptyLine checks if the current line is empty (and whitespace doesn't count)
func (p *Position) EmptyLine() bool {
	return 0 == len(strings.TrimRightFunc(p.e.Line(p.DataY()), unicode.IsSpace))
}

// AtStartOfLine returns true if the position is at the very start of the line, regardless of whitespace
func (p *Position) AtStartOfLine() bool {
	return p.sx == 0
}

// AtStartOfText returns true if the position is at the start of the text for this line
func (p *Position) AtStartOfText(e *Editor) bool {
	return p.sx == p.e.FirstScreenPosition(p.DataY())
}

// BeforeStartOfText returns true if the position is before the start of the text for this line
func (p *Position) BeforeStartOfText(e *Editor) bool {
	return p.sx < p.e.FirstScreenPosition(p.DataY())
}

// BeforeOrAtStartOfText returns true if the position is before or at the start of the text for this line
func (p *Position) BeforeOrAtStartOfText(e *Editor) bool {
	return p.sx <= p.e.FirstScreenPosition(p.DataY())
}
