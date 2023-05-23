package main

import (
	"fmt"
	"strings"

	"github.com/xyproto/vt100"
)

// TableWidget represents a TUI widget for editing a Markdown table
type TableWidget struct {
	title          string               // title
	contents       *[][]string          // the table contents
	bgColor        vt100.AttributeColor // background color
	highlightColor vt100.AttributeColor // selected color (the choice that has been selected after return has been pressed)
	headerColor    vt100.AttributeColor // the color of the table header row
	textColor      vt100.AttributeColor // text color (the choices that are not highlighted)
	titleColor     vt100.AttributeColor // title color (above the choices)
	cursorColor    vt100.AttributeColor // color of the "_" cursor
	cx             int                  // current content position
	marginLeft     int                  // margin, may be negative?
	marginTop      int                  // margin, may be negative?
	oldy           int                  // previous position
	cy             int                  // current content position
	oldx           int                  // previous position
	h              int                  // height (number of menu items)
	w              int                  // width
}

// NewTableWidget creates a new TableWidget
func NewTableWidget(title string, contents *[][]string, titleColor, headerColor, textColor, highlightColor, cursorColor, bgColor vt100.AttributeColor, canvasWidth, canvasHeight, initialY int) *TableWidget {

	columnWidths := TableColumnWidths([]string{}, *contents)

	widgetWidth := 0
	for _, w := range columnWidths {
		widgetWidth += w + 1
	}
	if widgetWidth > int(canvasWidth) {
		widgetWidth = int(canvasWidth)
	}

	widgetHeight := len(*contents)

	return &TableWidget{
		title:          title,
		w:              widgetWidth,
		h:              widgetHeight,
		cx:             0,
		oldx:           0,
		cy:             initialY,
		oldy:           initialY,
		marginLeft:     10,
		marginTop:      10,
		contents:       contents,
		titleColor:     titleColor,
		headerColor:    headerColor,
		textColor:      textColor,
		highlightColor: highlightColor,
		cursorColor:    cursorColor,
		bgColor:        bgColor,
	}
}

// Expand the table contents to the longest width
func Expand(contents *[][]string) {
	// Find the max width
	maxWidth := 0
	for y := 0; y < len(*contents); y++ {
		if len((*contents)[y]) > maxWidth {
			maxWidth = len(*contents)
		}
	}
	// Find all rows less than max width
	for y := 0; y < len(*contents); y++ {
		if len((*contents)[y]) < maxWidth {
			backup := (*contents)[y]
			// Expand the row by creating a blank string slice
			(*contents)[y] = make([]string, maxWidth)
			// Fill in the old data for the first fields of the row
			for x := 0; x < len(backup); x++ {
				(*contents)[y][x] = backup[x]
			}
		}
	}
}

// ContentsWH returns the width and the height of the table contents
func (tw *TableWidget) ContentsWH() (int, int) {
	rowCount := len(*tw.contents)
	if rowCount == 0 {
		return 0, 0
	}
	return len((*tw.contents)[0]), rowCount
}

// Draw will draw this menu widget on the given canvas
func (tw *TableWidget) Draw(c *vt100.Canvas) {
	cw, ch := tw.ContentsWH()

	// Draw the title
	titleHeight := 2
	title := tw.title
	if len(*tw.contents) > 0 {
		title = tw.title + fmt.Sprintf("(%d, %d) [%d, %d]", tw.w, tw.h, cw, ch)
	}
	for x, r := range title {
		c.PlotColor(uint(tw.marginLeft+x), uint(tw.marginTop), tw.titleColor, r)
	}

	columnWidths := TableColumnWidths([]string{}, *tw.contents)

	// Draw the headers, with various colors
	// Draw the menu entries, with various colors
	for y := 0; y < ch; y++ {
		xpos := tw.marginLeft
		// First clear this row with spaces
		spaces := strings.Repeat(" ", int(c.W()))
		c.Write(0, uint(tw.marginTop+y+titleHeight), tw.textColor, tw.bgColor, spaces)
		for x := 0; x < len((*tw.contents)[y]); x++ {
			field := (*tw.contents)[y][x]
			color := tw.textColor
			if y == int(tw.cy) && x == int(tw.cx) {
				color = tw.highlightColor
				// Draw the "cursor"
				c.Write(uint(xpos+len(field)), uint(tw.marginTop+y+titleHeight), tw.cursorColor, tw.bgColor, "_")
			} else if y == 0 {
				color = tw.headerColor
			}
			c.Write(uint(xpos), uint(tw.marginTop+y+titleHeight), color, tw.bgColor, field)
			xpos += columnWidths[x] + 2
		}
	}

}

// Up will move the highlight up (with wrap-around)
func (tw *TableWidget) Up() {
	cw, ch := tw.ContentsWH()

	tw.oldy = tw.cy
	tw.cy--
	if tw.cy < 0 {
		tw.cy = ch - 1
	}
	// just in case rows have differing lengths
	if tw.cx >= cw {
		tw.cx = cw - 1
	}
}

// Down will move the highlight down (with wrap-around)
func (tw *TableWidget) Down() {
	cw, ch := tw.ContentsWH()

	tw.oldy = tw.cy
	tw.cy++
	if tw.cy >= ch-1 {
		tw.cy = 0
	}
	// just in case rows have differing lengths
	if tw.cx >= cw {
		tw.cx = cw - 1
	}
}

// Left will move the highlight left (with wrap-around)
func (tw *TableWidget) Left() {
	cw, _ := tw.ContentsWH()

	tw.oldx = tw.cx
	tw.cx--
	if tw.cx < 0 {
		tw.cx = cw - 1
	}
}

// Right will move the highlight right (with wrap-around)
func (tw *TableWidget) Right() {
	cw, _ := tw.ContentsWH()

	tw.oldx = tw.cx
	tw.cx++
	if tw.cx >= cw {
		tw.cx = 0
	}
}

// NextOrInsert will move the highlight to the next cell, or insert a new row
func (tw *TableWidget) NextOrInsert() bool {
	cw, ch := tw.ContentsWH()

	tw.oldx = tw.cx
	tw.cx++
	if tw.cx >= cw {
		tw.cx = 0
		tw.cy++
		if tw.cy >= ch {
			newRow := make([]string, cw, cw)
			(*tw.contents) = append((*tw.contents), newRow)
			tw.h++     // Update the widget table height as well (this is not the content height)
			tw.cy = ch // old max index + 1
		}
	}
	return true // redraw
}

// SelectIndex will select a specific index. Returns false if it was not possible.
func (tw *TableWidget) SelectIndex(x, y int) bool {
	cw, ch := tw.ContentsWH()

	if x >= cw || y >= ch {
		return false
	}
	tw.oldx = tw.cx
	tw.oldy = tw.cy
	tw.cx = x
	tw.cy = y
	return true
}

// SelectFirst will select the first menu choice
func (tw *TableWidget) SelectFirst() bool {
	return tw.SelectIndex(0, 0)
}

// SelectLast will select the last menu choice
func (tw *TableWidget) SelectLast() bool {
	cw, ch := tw.ContentsWH()

	tw.oldx = tw.cx
	tw.oldy = tw.cy
	tw.cx = cw - 1
	tw.cy = ch - 1
	return true
}

// Set will change the field contents of the current position
func (tw *TableWidget) Set(field string) {
	(*tw.contents)[tw.cy][tw.cx] = field
}

// Get will retrieve the contents of the current field
func (tw *TableWidget) Get() string {
	return (*tw.contents)[tw.cy][tw.cx]
}

// Add will add a string to the current field
func (tw *TableWidget) Add(s string) {
	(*tw.contents)[tw.cy][tw.cx] += s
}

// TrimAll will trim the leading and trailing spaces from all fields in this table
func (tw *TableWidget) TrimAll() {
	for y := 0; y < len(*tw.contents); y++ {
		for x := 0; x < len((*tw.contents)[y]); x++ {
			(*tw.contents)[y][x] = strings.TrimSpace((*tw.contents)[y][x])
		}
	}
}
