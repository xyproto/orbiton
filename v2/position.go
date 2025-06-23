package main

import (
	"errors"
	"sync"

	"github.com/xyproto/vt"
)

// Position represents a position on the screen, including how far down the view has scrolled
type Position struct {
	mut         *sync.RWMutex // for the position
	sx          int           // the position of the cursor in the current scrollview
	sy          int           // the position of the cursor in the current scrollview
	offsetX     int           // how far one has scrolled along the X axis
	offsetY     int           // how far one has scrolled along the Y axis
	scrollSpeed int           // how many lines to scroll, when scrolling up and down
	savedX      int           // for smart down cursor movement
}

// NewPosition returns a new Position struct
func NewPosition(scrollSpeed int) *Position {
	return &Position{&sync.RWMutex{}, 0, 0, 0, 0, scrollSpeed, 0}
}

// Copy will create a new Position struct that is a copy of this one
func (p *Position) Copy() *Position {
	return &Position{p.mut, p.sx, p.sy, p.offsetX, p.offsetY, p.scrollSpeed, p.savedX}
}

// ScreenX returns the screen X position in the current view
func (p *Position) ScreenX() int {
	var x int
	p.mut.RLock()
	x = p.sx
	p.mut.RUnlock()
	return x
}

// ScreenY returns the screen Y position in the current view
func (p *Position) ScreenY() int {
	var y int
	p.mut.RLock()
	y = p.sy
	p.mut.RUnlock()
	return y
}

// OffsetX returns the X scroll offset for the current view
func (p *Position) OffsetX() int {
	var x int
	p.mut.RLock()
	x = p.offsetX
	p.mut.RUnlock()
	return x
}

// OffsetY returns the Y scroll offset for the current view
func (p *Position) OffsetY() int {
	var y int
	p.mut.RLock()
	y = p.offsetY
	p.mut.RUnlock()
	return y
}

// SetX will set the screen X position
func (p *Position) SetX(c *vt.Canvas, x int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.sx = x
	w := 80 // default width
	if c != nil {
		w = int(c.W())
	}
	if x < w {
		p.offsetX = 0
	} else {
		p.offsetX = (x - w) + 1
		p.sx -= p.offsetX
	}
}

// SetY will set the screen Y position
func (p *Position) SetY(y int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.sy = y
}

// DecY will decrease Y by 1
func (p *Position) DecY() {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.sy--
	if p.sy < 0 {
		p.sy = 0
	}
}

// IncY will increase Y by 1
func (p *Position) IncY(c *vt.Canvas) {
	p.mut.Lock()
	defer p.mut.Unlock()

	h := 25 // default height
	if c != nil {
		h = int(c.H())
	}

	p.sy++
	if p.sy > (h - 1) {
		p.sy = (h - 1)
	}
}

// SetOffsetX will set the screen X scrolling offset
func (p *Position) SetOffsetX(offsetX int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.offsetX = offsetX
}

// SetOffsetY will set the screen Y scrolling offset
func (p *Position) SetOffsetY(offsetY int) {
	p.mut.Lock()
	defer p.mut.Unlock()

	p.offsetY = offsetY
}

// Up will move the cursor up
func (p *Position) Up() error {
	p.mut.Lock()
	defer p.mut.Unlock()
	if p.sy <= 0 {
		return errors.New("already at the top of the canvas")
	}
	p.sy--
	return nil
}

// Down will move the cursor down
func (p *Position) Down(c *vt.Canvas) error {
	p.mut.Lock()
	defer p.mut.Unlock()
	h := 25 // default height
	if c != nil {
		h = int(c.H())
	}
	if p.sy >= h-1 {
		return errors.New("already at the bottom of the canvas")
	}
	p.sy++
	return nil
}

// AtStartOfScreenLine returns true if the position is at the very start of the line, regardless of whitespace and scrolling
func (p *Position) AtStartOfScreenLine() bool {
	p.mut.RLock()
	defer p.mut.RUnlock()

	return p.sx == 0
}

// AtStartOfTheLine returns true if the position is at the very start of the line, and the line is not scrolled
func (p *Position) AtStartOfTheLine() bool {
	p.mut.RLock()
	defer p.mut.RUnlock()

	return p.sx == 0 && p.offsetX == 0
}

// LineIndex returns the current line index this position is at
func (p *Position) LineIndex() LineIndex {
	p.mut.RLock()
	defer p.mut.RUnlock()

	return LineIndex(p.offsetY + p.sy)
}

// LineNumber returns the current line number this Position is at
func (p *Position) LineNumber() LineNumber {
	p.mut.RLock()
	defer p.mut.RUnlock()

	return LineIndex(p.offsetY + p.sy).LineNumber()
}

// ColNumber returns the current column number this Position is at
func (p *Position) ColNumber() ColNumber {
	p.mut.RLock()
	defer p.mut.RUnlock()

	return ColIndex(p.offsetX + p.sx).ColNumber()
}

// Right will move the cursor to the right, if possible.
// It will not move the cursor up or down.
func (p *Position) Right(c *vt.Canvas) {
	p.mut.Lock()
	defer p.mut.Unlock()

	w := 80 // default width
	if c != nil {
		w = int(c.Width())
	}
	if p.sx < (w - 1) {
		p.sx++
	} else {
		p.sx = 0
		p.offsetX += (w - 1)
	}
}

// Left will move the cursor to the left, if possible.
// It will not move the cursor up or down.
func (p *Position) Left() {
	p.mut.Lock()
	defer p.mut.Unlock()

	if p.sx > 0 {
		p.sx--
	}
}
