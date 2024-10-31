package main

import ()

// SymbolWidget represents a TUI widget for presenting a menu with choices for the user
type SymbolWidget struct {
	title          string         // title
	choices        [][]string     // a slice of a slice of menu items
	bgColor        AttributeColor // background color
	highlightColor AttributeColor // selected color (the choice that has been selected after return has been pressed)
	textColor      AttributeColor // text color (the choices that are not highlighted)
	titleColor     AttributeColor // title color (above the choices)
	x              int            // current position
	marginLeft     int            // margin, may be negative?
	marginTop      int            // margin, may be negative?
	oldy           int            // previous position
	y              int            // current position
	oldx           int            // previous position
	h              int            // height (number of menu items)
	w              int            // width
}

// NewSymbolWidget creates a new SymbolWidget
func NewSymbolWidget(title string, choices [][]string, titleColor, textColor, highlightColor, bgColor AttributeColor, canvasWidth, canvasHeight int) *SymbolWidget {
	maxlen := 0
	for _, choice := range choices {
		if len(choice) > maxlen {
			maxlen = len(choice)
		}
	}
	marginLeft := 10
	if canvasWidth-(maxlen+marginLeft) <= 0 {
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
		w:              marginLeft + maxlen,
		h:              len(choices),
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
func (sw *SymbolWidget) Draw(c *Canvas) {
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
			// SetXY(0, uint(sw.marginTop+y+titleHeight))
			if y == int(sw.y) && x == int(sw.x) {
				c.Write(uint(sw.marginLeft+x*2), uint(sw.marginTop+y+titleHeight), sw.highlightColor, sw.bgColor, symbol)
			} else {
				c.Write(uint(sw.marginLeft+x*2), uint(sw.marginTop+y+titleHeight), sw.textColor, sw.bgColor, symbol)
			}
		}

	}
}

// Up will move the highlight up (with wrap-around)
func (sw *SymbolWidget) Up() {
	sw.oldy = sw.y
	if sw.y == 0 {
		sw.y = len(sw.choices) - 1
	} else {
		sw.y--
	}
	// just in case rows have differing lengths
	l := len(sw.choices[sw.y])
	if sw.x >= l {
		sw.x = l - 1
	}
}

// Down will move the highlight down (with wrap-around)
func (sw *SymbolWidget) Down() {
	sw.oldy = sw.y
	sw.y++
	if sw.y >= len(sw.choices) {
		sw.y = 0
	}
	l := len(sw.choices[sw.y])
	if sw.x >= l {
		sw.x = l - 1
	}
}

// Left will move the highlight left (with wrap-around)
func (sw *SymbolWidget) Left() bool {
	sw.oldx = sw.x
	sw.x--
	if sw.x < 0 {
		row := sw.choices[sw.y]
		sw.x = len(row) - 1
	}
	return true
}

// Right will move the highlight right (with wrap-around)
func (sw *SymbolWidget) Right() {
	sw.oldx = sw.x
	sw.x++
	row := sw.choices[sw.y]
	if sw.x >= len(row) {
		sw.x = 0
	}
}

// Next will move the highlight to the next cell
func (sw *SymbolWidget) Next() {
	sw.oldx = sw.x
	sw.x++
	row := sw.choices[sw.y]
	if sw.x >= len(row) {
		sw.x = 0
		sw.y++
	}
	row = sw.choices[sw.y]
	if sw.x >= len(row) {
		sw.x = 0
		sw.y++
	}
	if sw.y >= len(sw.choices) {
		sw.y = 0
	}
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (sw *SymbolWidget) SelectIndex(x, y int) bool {
	if y >= sw.h || x >= sw.w {
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
