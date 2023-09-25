package main

import (
	"github.com/xyproto/vt100"
)

// Box is a position, width and height
type Box struct {
	X int
	Y int
	W int
	H int
}

// BoxTheme contains the runes used to draw boxes
type BoxTheme struct {
	LowerEdge  *vt100.AttributeColor
	UpperEdge  *vt100.AttributeColor
	Highlight  *vt100.AttributeColor
	Text       *vt100.AttributeColor
	Background *vt100.AttributeColor
	Foreground *vt100.AttributeColor
	HB         rune
	HT         rune
	VR         rune
	VL         rune
	BR         rune
	BL         rune
	TR         rune
	TL         rune
}

// NewBox creates a new box/container
func NewBox() *Box {
	return &Box{0, 0, 0, 0}
}

// NewBoxTheme creates a new theme/style for a box/container, based on the editor theme
func (e *Editor) NewBoxTheme() *BoxTheme {
	return &BoxTheme{
		TL:         '╭', // top left
		TR:         '╮', // top right
		BL:         '╰', // bottom left
		BR:         '╯', // bottom right
		VL:         '│', // vertical line, left side
		VR:         '│', // vertical line, right side
		HT:         '─', // horizontal line
		HB:         '─', // horizontal bottom line
		Foreground: &e.Foreground,
		Background: &e.BoxBackground,
		Text:       &e.BoxTextColor,
		Highlight:  &e.BoxHighlight,
		UpperEdge:  &e.BoxUpperEdge,
		LowerEdge:  &e.BoxTextColor,
	}
}

// NewCanvasBox creates a new box/container for the entire canvas/screen
func NewCanvasBox(c *vt100.Canvas) *Box {
	w := int(c.W())
	h := int(c.H())
	return &Box{0, 0, w, h}
}

// Center will place a Box at the center of the given container.
func (b *Box) Center(container *Box) {
	widthleftover := container.W - b.W
	heightleftover := container.H - b.H
	b.X = container.X + widthleftover/2
	b.Y = container.Y + heightleftover/2
}

// Fill will place a Box so that it fills the entire given container.
func (b *Box) Fill(container *Box) {
	b.X = container.X
	b.Y = container.Y
	b.W = container.W
	b.H = container.H
}

// FillWithMargins will place a Box inside a given container, with the given margins.
// Margins are given in number of characters.
func (b *Box) FillWithMargins(container *Box, xmargins, ymargins int) {
	b.Fill(container)
	b.X += xmargins
	b.Y += ymargins
	b.W -= xmargins * 2
	b.H -= ymargins * 2
}

// UpperRightPlacement will place a box in the upper right corner of a container, like a little window
func (b *Box) UpperRightPlacement(container *Box, minWidth int) {
	w := float64(container.W)
	h := float64(container.H)
	b.X = int(w * 0.6)
	b.Y = int(h * 0.1)
	b.W = int(w * 0.3)
	if b.W < minWidth {
		b.W = minWidth
	}
	b.H = int(h * 0.25)
	if (b.X + b.W) >= int(w) {
		b.W = int(w) - b.X
	}
}

// LowerRightPlacement will place a box in the lower right corner of a container, like a little window
func (b *Box) LowerRightPlacement(container *Box, minWidth int) {
	w := float64(container.W)
	h := float64(container.H)
	b.X = int(w * 0.6)
	b.Y = int(h * 0.37)
	b.W = int(w * 0.3)
	if b.W < minWidth {
		b.W = minWidth
	}
	b.H = int(h * 0.45)
	if (b.X + b.W) >= int(w) {
		b.W = int(w) - b.X
	}
}

// LowerLeftPlacement will place a box in the lower left corner of a container, like a little window
func (b *Box) LowerLeftPlacement(container *Box, minWidth int) {
	w := float64(container.W)
	h := float64(container.H)
	b.X = int(w * 0.05)
	b.Y = int(h * 0.37)
	b.W = int(w * 0.5)
	if b.W < minWidth {
		b.W = minWidth
	}
	b.H = int(h * 0.45)
	if (b.X + b.W) >= int(w) {
		b.W = int(w) - b.X
	}
}

// EvenLowerRightPlacement will place the box even lower
func (b *Box) EvenLowerRightPlacement(container *Box, minWidth int) {
	w := float64(container.W)
	h := float64(container.H)
	b.X = int(w * 0.3)
	b.Y = int(h * 0.83)
	b.W = int(w * 0.62)
	if b.W < minWidth {
		b.W = minWidth
	}
	b.H = int(h * 0.18)
	if (b.X + b.W) >= int(w) {
		b.W = int(w) - b.X
	}
}

// LowerPlacement will place a box in the lower right corner of a container, like a little window
func (b *Box) LowerPlacement(container *Box, minWidth int) {
	w := float64(container.W)
	h := float64(container.H)
	b.X = int(w * 0.1)
	b.Y = int(h * 0.3)
	b.W = int(w * 0.8)
	if b.W < minWidth {
		b.W = minWidth
	}
	b.H = int(h * 0.7)
	if (b.X + b.W) >= int(w) {
		b.W = int(w) - b.X
	}
}

// Say will output text at the given coordinates, with the configured theme
func (e *Editor) Say(bt *BoxTheme, c *vt100.Canvas, x, y int, text string) {
	c.Write(uint(x), uint(y), *bt.Text, *bt.Background, text)
}

// DrawBox can draw a box using "text graphics".
// The given Box struct defines the size and placement.
// If extrude is True, the box looks a bit more like it's sticking out.
// bg is expected to be a background color, for instance e.BoxBackground.
func (e *Editor) DrawBox(bt *BoxTheme, c *vt100.Canvas, r *Box) *Box {
	var (
		bg     = bt.Background
		FG1    = bt.UpperEdge
		FG2    = bt.LowerEdge
		x      = uint(r.X)
		y      = uint(r.Y)
		width  = uint(r.W)
		height = uint(r.H)
	)
	c.WriteRune(x, y, *FG1, *bg, bt.TL)
	for i := x + 1; i < x+(width-1); i++ {
		c.WriteRune(i, y, *FG1, *bg, bt.HT)
	}
	c.WriteRune(x+width-1, y, *FG1, *bg, bt.TR)
	for i := y + 1; i < y+height; i++ {
		c.WriteRune(x, i, *FG1, *bg, bt.VL)
		c.Write(x+1, i, *FG1, *bg, repeatRune(' ', width-2))
		c.WriteRune(x+width-1, i, *FG2, *bg, bt.VR)
	}
	c.WriteRune(x, y+height-1, *FG1, *bg, bt.BL)
	for i := x + 1; i < x+(width-1); i++ {
		c.WriteRune(i, y+height-1, *FG2, *bg, bt.HB)
	}
	c.WriteRune(x+width-1, y+height-1, *FG2, *bg, bt.BR)
	return &Box{int(x), int(y), int(width), int(height)}
}

// DrawList will draw a list widget. Takes a Box struct for the size and position.
// Takes a list of strings to be listed and an int that represents
// which item is currently selected. Does not scroll or wrap.
// Set selected to -1 to skip highlighting one of the items.
// Uses bt.Highlight, bt.Text and bt.Background.
func (e *Editor) DrawList(bt *BoxTheme, c *vt100.Canvas, r *Box, items []string, selected int) {
	x := uint(r.X)
	for i, s := range items {
		y := uint(r.Y + i)
		if i == selected {
			c.Write(x, y, *bt.Highlight, *bt.Background, s)
		} else {
			c.Write(x, y, *bt.Text, *bt.Background, s)
		}
	}
}

// DrawTitle draws a title at the top of a box, not exactly centered
func (e *Editor) DrawTitle(bt *BoxTheme, c *vt100.Canvas, r *Box, title string) {
	titleWithSpaces := " " + title + " "
	tmp := bt.Text
	bt.Text = bt.UpperEdge
	e.Say(bt, c, r.X+(r.W-len(titleWithSpaces))/2, r.Y, titleWithSpaces)
	bt.Text = tmp
}

// DrawText will draw a text widget. Takes a Box struct for the size and position.
// Takes a list of strings. Does not scroll or wrap. Uses bt.Foreground and bt.Background.
func (e *Editor) DrawText(bt *BoxTheme, c *vt100.Canvas, r *Box, lines []string) {
	x := uint(r.X)
	for i, s := range lines {
		y := uint(r.Y + i)
		// TODO: Make it possible to output colored text without ruining the box edges and text alignment.
		//       Look at highlight.go
		c.Write(x, y, *bt.Foreground, *bt.Background, s)
	}
}
