package main

import (
	"github.com/xyproto/vt100"
)

// TableWidget represents a TUI widget for editing a Markdown table
type TableWidget struct {
	title          string               // title
	contents       [][]string           // the table contents
	bgColor        vt100.AttributeColor // background color
	highlightColor vt100.AttributeColor // selected color (the choice that has been selected after return has been pressed)
	headerColor    vt100.AttributeColor // the color of the table header row
	textColor      vt100.AttributeColor // text color (the choices that are not highlighted)
	titleColor     vt100.AttributeColor // title color (above the choices)
	x              int                  // current position
	marginLeft     int                  // margin, may be negative?
	marginTop      int                  // margin, may be negative?
	oldy           int                  // previous position
	y              int                  // current position
	oldx           int                  // previous position
	h              int                  // height (number of menu items)
	w              int                  // width
}

// NewTableWidget creates a new TableWidget
func NewTableWidget(title string, contents [][]string, titleColor, headerColor, textColor, highlightColor, bgColor vt100.AttributeColor, canvasWidth, canvasHeight int) *TableWidget {

	columnWidths := TableColumnWidths([]string{}, contents)

	widgetWidth := 0
	for _, w := range columnWidths {
		widgetWidth += w + 1
	}
	if widgetWidth > int(canvasWidth) {
		widgetWidth = int(canvasWidth)
	}

	widgetHeight := len(contents)

	return &TableWidget{
		title:          title,
		w:              widgetWidth,
		h:              widgetHeight,
		x:              0,
		oldx:           0,
		y:              0,
		oldy:           0,
		marginLeft:     10,
		marginTop:      10,
		contents:       contents,
		titleColor:     titleColor,
		headerColor:    headerColor,
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

	columnWidths := TableColumnWidths([]string{}, tw.contents)

	// Draw the headers, with various colors
	// Draw the menu entries, with various colors
	for y := 0; y < len(tw.contents); y++ {
		row := tw.contents[y]
		xpos := tw.marginLeft
		for x := 0; x < len(row); x++ {
			field := tw.contents[y][x]
			color := tw.textColor
			if y == int(tw.y) && x == int(tw.x) {
				color = tw.highlightColor
			} else if y == 0 {
				color = tw.headerColor
			}
			c.Write(uint(xpos), uint(tw.marginTop+y+titleHeight), color, tw.bgColor, field)
			xpos += columnWidths[x] + 1
		}
	}

}

// Up will move the highlight up (with wrap-around)
func (tw *TableWidget) Up() bool {
	tw.oldy = tw.y
	tw.y--
	if tw.y < 0 {
		tw.y = len(tw.contents) - 1
	}
	l := len(tw.contents[tw.y])
	if tw.x > l {
		tw.x = l - 1
	}
	return true
}

// Down will move the highlight down (with wrap-around)
func (tw *TableWidget) Down() {
	tw.oldy = tw.y
	tw.y++
	if tw.y >= (len(tw.contents) - 1) {
		tw.y = 0
	}
	l := len(tw.contents[tw.y])
	if tw.x > l {
		tw.x = l - 1
	}
}

// Left will move the highlight left (with wrap-around)
func (tw *TableWidget) Left() bool {
	tw.oldx = tw.x
	if tw.x <= 0 {
		row := tw.contents[tw.y]
		tw.x = len(row) - 1
	} else {
		tw.x--
	}
	return true
}

// Right will move the highlight right (with wrap-around)
func (tw *TableWidget) Right() {
	tw.oldx = tw.x
	tw.x++
	row := tw.contents[tw.y]
	if tw.x >= len(row) {
		tw.x = 0
	}
}

// Next will move the highlight to the next cell
func (tw *TableWidget) Next() {
	tw.oldx = tw.x
	tw.x++
	row := tw.contents[tw.y]
	if tw.x >= len(row) {
		tw.x = 0
		tw.y++
	}
	if tw.y >= len(tw.contents) {
		tw.y = 0
	}
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (tw *TableWidget) SelectIndex(x, y int) bool {
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
