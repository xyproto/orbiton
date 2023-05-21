package main

import (
	"github.com/xyproto/vt100"
)

// TableWidget represents a TUI widget for editing a Markdown table
type TableWidget struct {
	title          string               // title
	headers        []string             // a slice of column headers
	body           [][]string           // a slice of a slice of menu items
	bgColor        vt100.AttributeColor // background color
	highlightColor vt100.AttributeColor // selected color (the choice that has been selected after return has been pressed)
	textColor      vt100.AttributeColor // text color (the choices that are not highlighted)
	titleColor     vt100.AttributeColor // title color (above the choices)
	x              uint                 // current position
	marginLeft     int                  // margin, may be negative?
	marginTop      int                  // margin, may be negative?
	oldy           uint                 // previous position
	y              uint                 // current position
	oldx           uint                 // previous position
	h              uint                 // height (number of menu items)
	w              uint                 // width
}

// NewTableWidget creates a new TableWidget
func NewTableWidget(title string, headers []string, body [][]string, titleColor, textColor, highlightColor, bgColor vt100.AttributeColor, canvasWidth, canvasHeight uint) *TableWidget {

	columnWidths := TableColumnWidths(headers, body)

	widgetWidth := 0
	for _, w := range columnWidths {
		widgetWidth += w + 1
	}

	widgetHeight := 1 + len(body)

	return &TableWidget{
		title:          title,
		w:              uint(widgetWidth),
		h:              uint(widgetHeight),
		x:              0,
		oldx:           0,
		y:              0,
		oldy:           0,
		marginLeft:     10,
		marginTop:      10,
		headers:        headers,
		body:           body,
		titleColor:     titleColor,
		textColor:      textColor,
		highlightColor: highlightColor,
		bgColor:        bgColor,
	}
}

// Draw will draw this menu widget on the given canvas
func (tw *TableWidget) Draw(c *vt100.Canvas) {
	// Draw the title
	titleHeight := 2
	for x, r := range tw.title {
		c.PlotColor(uint(tw.marginLeft+x), uint(tw.marginTop), tw.titleColor, r)
	}
	// Draw the menu entries, with various colors
	for y := 0; y < len(tw.body); y++ {
		row := tw.body[y]
		for x := 0; x < len(row); x++ {
			field := tw.body[y][x]
			if y == int(tw.y) && x == int(tw.x) {
				c.Write(uint(tw.marginLeft+x*2), uint(tw.marginTop+y+titleHeight), tw.highlightColor, tw.bgColor, field)
			} else {
				c.Write(uint(tw.marginLeft+x*2), uint(tw.marginTop+y+titleHeight), tw.textColor, tw.bgColor, field)
			}
		}

	}
}

// Up will move the highlight up (with wrap-around)
func (tw *TableWidget) Up() bool {
	tw.oldy = tw.y
	if tw.y <= 0 {
		tw.y = uint(len(tw.body)) - 1
	} else {
		tw.y--
	}
	row := tw.body[tw.y]
	if tw.x > uint(len(row)) {
		tw.x = uint(len(row) - 1)
	}
	return true
}

// Left will move the highlight left (with wrap-around)
func (tw *TableWidget) Left() bool {
	tw.oldx = tw.x
	if tw.x <= 0 {
		row := tw.body[tw.y]
		tw.x = uint(len(row)) - 1
	} else {
		tw.x--
	}
	return true
}

// Down will move the highlight down (with wrap-around)
func (tw *TableWidget) Down() {
	tw.oldy = tw.y
	tw.y++
	if tw.y >= uint(len(tw.body)) {
		tw.y = 0
	}
	row := tw.body[tw.y]
	if tw.x > uint(len(row)) {
		tw.x = uint(len(row) - 1)
	}
}

// Right will move the highlight right (with wrap-around)
func (tw *TableWidget) Right() {
	tw.oldx = tw.x
	tw.x++
	row := tw.body[tw.y]
	if tw.x >= uint(len(row)) {
		tw.x = 0
	}
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (tw *TableWidget) SelectIndex(x, y uint) bool {
	if y >= tw.h || x >= tw.w {
		return false
	}
	tw.oldx = tw.x
	tw.oldy = tw.y
	tw.x = x
	tw.y = y
	return true
}

// SelectFirst will select the first menu choice
func (tw *TableWidget) SelectFirst() bool {
	return tw.SelectIndex(0, 0)
}

// SelectLast will select the last menu choice
func (tw *TableWidget) SelectLast() bool {
	tw.oldx = tw.x
	tw.oldy = tw.y
	tw.x = tw.w - 1
	tw.y = tw.h - 1
	return true
}
