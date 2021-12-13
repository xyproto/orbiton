package main

import (
	"github.com/xyproto/vt100"
)

// SymbolWidget represents a TUI widget for presenting a menu with choices for the user
type SymbolWidget struct {
	title          string               // title
	w              uint                 // width
	h              uint                 // height (number of menu items)
	x              uint                 // current position
	oldx           uint                 // previous position
	y              uint                 // current position
	oldy           uint                 // previous position
	marginLeft     int                  // margin, may be negative?
	marginTop      int                  // margin, may be negative?
	choices        [][]string           // a slice of a slice of menu items
	titleColor     vt100.AttributeColor // title color (above the choices)
	textColor      vt100.AttributeColor // text color (the choices that are not highlighted)
	highlightColor vt100.AttributeColor // selected color (the choice that has been selected after return has been pressed)
	bgColor        vt100.AttributeColor // background color
}

// NewSymbolWidget creates a new SymbolWidget
func NewSymbolWidget(title string, choices [][]string, titleColor, textColor, highlightColor, bgColor vt100.AttributeColor, canvasWidth, canvasHeight uint) *SymbolWidget {
	maxlen := uint(0)
	for _, choice := range choices {
		if uint(len(choice)) > uint(maxlen) {
			maxlen = uint(len(choice))
		}
	}
	marginLeft := 10
	if int(canvasWidth)-(int(maxlen)+marginLeft) <= 0 {
		marginLeft = 0
	}
	marginTop := 8
	if int(canvasHeight)-(len(choices)+marginTop) <= 8 {
		marginTop = 2
	} else if int(canvasHeight)-(len(choices)+marginTop) <= 0 {
		marginTop = 0
	}
	return &SymbolWidget{
		title:          title,
		w:              uint(marginLeft + int(maxlen)),
		h:              uint(len(choices)),
		x:              0,
		oldx:           0,
		y:              0,
		oldy:           0,
		marginLeft:     marginLeft,
		marginTop:      marginTop,
		choices:        choices,
		titleColor:     titleColor,
		textColor:      textColor,
		highlightColor: highlightColor,
		bgColor:        bgColor,
	}
}

// Selected returns the currently selected item
func (sw *SymbolWidget) Selected() (int, int) {
	return int(sw.x), int(sw.y)
}

// Draw will draw this menu widget on the given canvas
func (sw *SymbolWidget) Draw(c *vt100.Canvas) {
	// Draw the title
	titleHeight := 2
	for x, r := range sw.title {
		c.PlotColor(uint(sw.marginLeft+x), uint(sw.marginTop), sw.titleColor, r)
	}
	// Draw the menu entries, with various colors
	for y := 0; y < len(sw.choices); y++ {
		row := sw.choices[y]
		for x := 0; x < len(row); x++ {
			symbol := sw.choices[y][x]
			if y == int(sw.y) && x == int(sw.x) {
				c.Write(uint(sw.marginLeft+x*2), uint(sw.marginTop+y+titleHeight), sw.highlightColor, sw.bgColor, symbol)
			} else {
				c.Write(uint(sw.marginLeft+x*2), uint(sw.marginTop+y+titleHeight), sw.textColor, sw.bgColor, symbol)
			}
		}

	}
}

// Up will move the highlight up (with wrap-around)
func (sw *SymbolWidget) Up(c *vt100.Canvas) bool {
	sw.oldy = sw.y
	if sw.y <= 0 {
		sw.y = uint(len(sw.choices)) - 1
	} else {
		sw.y--
	}
	row := sw.choices[sw.y]
	if sw.x > uint(len(row)) {
		sw.x = uint(len(row) - 1)
	}
	return true
}

// Left will move the highlight left (with wrap-around)
func (sw *SymbolWidget) Left(c *vt100.Canvas) bool {
	sw.oldx = sw.x
	if sw.x <= 0 {
		row := sw.choices[sw.y]
		sw.x = uint(len(row)) - 1
	} else {
		sw.x--
	}
	return true
}

// Down will move the highlight down (with wrap-around)
func (sw *SymbolWidget) Down(c *vt100.Canvas) bool {
	sw.oldy = sw.y
	sw.y++
	if sw.y >= uint(len(sw.choices)) {
		sw.y = 0
	}
	row := sw.choices[sw.y]
	if sw.x > uint(len(row)) {
		sw.x = uint(len(row) - 1)
	}
	return true
}

// Right will move the highlight right (with wrap-around)
func (sw *SymbolWidget) Right(c *vt100.Canvas) bool {
	sw.oldx = sw.x
	sw.x++
	row := sw.choices[sw.y]
	if sw.x >= uint(len(row)) {
		sw.x = 0
	}
	return true
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (sw *SymbolWidget) SelectIndex(x, y uint) bool {
	if y >= sw.h {
		return false
	}
	if x >= sw.w {
		return false
	}
	sw.oldx = sw.x
	sw.oldy = sw.y
	sw.x = x
	sw.y = y
	return true
}

// SelectFirst will select the first menu choice
func (sw *SymbolWidget) SelectFirst() bool {
	return sw.SelectIndex(0, 0)
}

// SelectLast will select the last menu choice
func (sw *SymbolWidget) SelectLast() bool {
	sw.oldx = sw.x
	sw.oldy = sw.y
	sw.x = sw.w - 1
	sw.y = sw.h - 1
	return true
}
