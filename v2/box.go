package main

import (
	"strings"

	"github.com/xyproto/vt"
	"github.com/xyproto/wordwrap"
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
	LowerEdge  *vt.AttributeColor
	UpperEdge  *vt.AttributeColor
	Highlight  *vt.AttributeColor
	Text       *vt.AttributeColor
	Background *vt.AttributeColor
	Foreground *vt.AttributeColor
	HB         rune
	HT         rune
	VR         rune
	VL         rune
	BR         rune
	BL         rune
	TR         rune
	TL         rune
	EdgeLeftT  rune
	EdgeRightT rune
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
		EdgeLeftT:  '├',
		EdgeRightT: '┤',
		Foreground: &e.Foreground,
		Background: &e.BoxBackground,
		Text:       &e.BoxTextColor,
		Highlight:  &e.BoxHighlight,
		UpperEdge:  &e.BoxUpperEdge,
		LowerEdge:  &e.BoxTextColor,
	}
}

// NewCanvasBox creates a new box/container for the entire canvas/screen
func NewCanvasBox(c *vt.Canvas) *Box {
	w := int(c.W())
	h := int(c.H())
	return &Box{0, 0, w, h}
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
func (e *Editor) Say(bt *BoxTheme, c *vt.Canvas, x, y int, text string) {
	c.Write(uint(x), uint(y), *bt.Text, *bt.Background, text)
}

// DrawBox can draw a box using "text graphics".
// The given Box struct defines the size and placement.
// If extrude is True, the box looks a bit more like it's sticking out.
// bg is expected to be a background color, for instance e.BoxBackground.
func (e *Editor) DrawBox(bt *BoxTheme, c *vt.Canvas, r *Box) *Box {
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
func (e *Editor) DrawList(bt *BoxTheme, c *vt.Canvas, r *Box, items []string, selected int) {
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
func (e *Editor) DrawTitle(bt *BoxTheme, c *vt.Canvas, r *Box, title string, withSpaces bool) {
	titleWithSpaces := title
	if withSpaces {
		titleWithSpaces = " " + title + " "
	}

	tmp := bt.Text
	bt.Text = bt.UpperEdge
	e.Say(bt, c, r.X+(r.W-len(titleWithSpaces))/2, r.Y, titleWithSpaces)
	bt.Text = tmp
}

// DrawFooter draws text at the bottom of a box, not exactly centered
func (e *Editor) DrawFooter(bt *BoxTheme, c *vt.Canvas, r *Box, text string) {
	textWithSpaces := " " + text + " "
	tmp := bt.Text
	bt.Text = bt.UpperEdge
	e.Say(bt, c, r.X+(r.W-len(textWithSpaces))/2, r.Y+r.H-1, textWithSpaces)
	bt.Text = tmp
}

// DrawText will draw a text widget. Takes a Box struct for the size and position.
// Takes a list of strings. Does not scroll. Uses bt.Foreground and bt.Background.
// The text is wrapped by using the WordWrap function.
// The number of lines that are added as a consequence of wrapping lines is returned as an int.
func (e *Editor) DrawText(bt *BoxTheme, c *vt.Canvas, r *Box, text string, dryRun bool) int {
	maxWidth := int(r.W) - 5 // Adjusted width to account for margins and padding
	x := uint(r.X)
	lineIndex := 0
	addedLines := 0 // Counter for added lines

	// Split the input text into lines
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		// Attempt to wrap the line
		wrappedLines, err := wordwrap.WordWrap(line, maxWidth)
		if err != nil {
			// If an error occurs, chop the line to fit the width
			if len(line) > maxWidth {
				line = line[:maxWidth]
			}
			wrappedLines = []string{line} // Overwrite wrappedLines with the chopped or original line
		} else {
			// Count the additional lines created by wrapping
			addedLines += len(wrappedLines) - 1
		}

		// Draw each wrapped or chopped line to the canvas
		for _, wrappedLine := range wrappedLines {
			y := uint(r.Y + lineIndex)
			if !dryRun {
				c.Write(x, y, *bt.Foreground, *bt.Background, wrappedLine)
			}
			lineIndex++
		}
	}

	return addedLines
}
