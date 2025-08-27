package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/xyproto/vt"
)

// TableWidget represents a TUI widget for editing a Markdown table
type TableWidget struct {
	contents         *[][]string       // the table contents
	title            string            // title
	cx               int               // current content position
	marginLeft       int               // margin, may be negative?
	marginTop        int               // margin, may be negative?
	oldy             int               // previous position
	cy               int               // current content position
	oldx             int               // previous position
	h                int               // height (number of menu items)
	w                int               // width
	bgColor          vt.AttributeColor // background color
	highlightColor   vt.AttributeColor // selected color (the choice that has been selected after return has been pressed)
	headerColor      vt.AttributeColor // the color of the table header row
	textColor        vt.AttributeColor // text color (the choices that are not highlighted)
	titleColor       vt.AttributeColor // title color (above the choices)
	cursorColor      vt.AttributeColor // color of the "_" cursor
	commentColor     vt.AttributeColor // comment color
	displayQuickHelp bool              // display "quick help" at the bottom
}

// NewTableWidget creates a new TableWidget
func NewTableWidget(title string, contents *[][]string, titleColor, headerColor, textColor, highlightColor, cursorColor, commentColor, bgColor vt.AttributeColor, canvasWidth, _, initialY int, displayQuickHelp bool) *TableWidget {

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
		title:            title,
		w:                widgetWidth,
		h:                widgetHeight,
		cx:               0,
		oldx:             0,
		cy:               initialY,
		oldy:             initialY,
		marginLeft:       10,
		marginTop:        10,
		contents:         contents,
		titleColor:       titleColor,
		headerColor:      headerColor,
		textColor:        textColor,
		highlightColor:   highlightColor,
		cursorColor:      cursorColor,
		commentColor:     commentColor,
		bgColor:          bgColor,
		displayQuickHelp: displayQuickHelp,
	}
}

// Ensure1x1 will ensure that the table is at least 1x1
func (tw *TableWidget) Ensure1x1() {
	cw, ch := tw.ContentsWH()
	if cw < 1 {
		cw = 1
	}
	if ch < 1 {
		ch = 1
	}
	if len(*tw.contents) == 0 {
		*tw.contents = make([][]string, ch)
	}
	for y := 0; y < len(*tw.contents); y++ {
		if len((*tw.contents)[y]) == 0 {
			(*tw.contents)[y] = make([]string, cw)
		}
	}
	if tw.cx < 0 {
		tw.cx = 0
	} else if tw.cx >= cw {
		tw.cx = cw - 1
	}
	if tw.cy < 0 {
		tw.cy = 0
	} else if tw.cy >= ch {
		tw.cy = ch - 1
	}
}

// Expand the table contents to the longest width
func Expand(contents *[][]string) {
	// Ensure that the table is at least 1x1
	if len(*contents) == 0 {
		*contents = make([][]string, 1)
	}
	if len((*contents)[0]) == 0 {
		(*contents)[0] = make([]string, 1)
	}
	// Find the max width
	maxWidth := 0
	for y := 0; y < len(*contents); y++ {
		row := (*contents)[y]
		rowLength := len(row)
		if rowLength > maxWidth {
			maxWidth = rowLength
		}
	}
	// Find all rows less than max width
	for y := 0; y < len(*contents); y++ {
		if (*contents)[y] == nil {
			// Initialize the row
			(*contents)[y] = make([]string, maxWidth)
		} else if len((*contents)[y]) < maxWidth {
			backup := (*contents)[y]
			// Expand the row by creating a blank string slice
			(*contents)[y] = make([]string, maxWidth)
			// Fill in the old data for the first fields of the row
			copy((*contents)[y], backup)
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
func (tw *TableWidget) Draw(c *vt.Canvas) {
	cw, ch := tw.ContentsWH()

	canvasWidth := int(c.W())

	// Height of the title + the size + help text + a blank line
	titleHeight := 3
	titleY := uint(tw.marginTop)

	// Draw the title
	title := tw.title
	for x, r := range title {
		c.PlotColor(uint(tw.marginLeft+x), titleY, tw.titleColor, r)
	}

	// Plot the table position and size below the title
	titleY++
	sizeString := fmt.Sprintf("%d,%d [%dx%d]", tw.cx, tw.cy, cw, ch)
	for x, r := range sizeString {
		c.PlotColor(uint(tw.marginLeft+x), titleY, tw.commentColor, r)
	}

	columnWidths := TableColumnWidths([]string{}, *tw.contents)

	// Draw the headers, with various colors
	// Draw the menu entries, with various colors
	for y := 0; y < ch; y++ {
		xpos := tw.marginLeft
		// First clear this row with spaces
		spaces := strings.Repeat(" ", canvasWidth)
		c.Write(0, uint(tw.marginTop+y+titleHeight), tw.textColor, tw.bgColor, spaces)

		lastX := len((*tw.contents)[y])
		if lastX > cw {
			lastX = cw
		}

		for x := 0; x < lastX; x++ {
			field := (*tw.contents)[y][x]
			color := tw.textColor
			if y == int(tw.cy) && x == int(tw.cx) {
				color = tw.highlightColor
				cursorX := uint(xpos + len(field))
				cursorY := uint(tw.marginTop + y + titleHeight)
				// Draw the "cursor"
				c.Write(cursorX, cursorY, tw.cursorColor, tw.bgColor, "_")
				// Also move the proper cursor, for good measure
				//SetXY(cursorX, cursorY)

			} else if y == 0 {
				color = tw.headerColor
			}
			c.Write(uint(xpos), uint(tw.marginTop+y+titleHeight), color, tw.bgColor, field)
			xpos += columnWidths[x] + 2
		}
	}

	indexY := uint(tw.marginTop + titleHeight + ch)

	// Clear a few extra rows after the table
	spaces := strings.Repeat(" ", canvasWidth)

	for y := uint(0); y < 5; y++ {
		if indexY < c.H() {
			c.Write(0, indexY, tw.textColor, tw.bgColor, spaces)
			indexY++
		}
	}

	indexY -= 3

	if indexY+4 < c.H() {

		if tw.displayQuickHelp {

			var indexX uint
			var helpString string
			var w uint

			// Plot the quick help
			helpString = "Quick instructions"
			indexX = uint(tw.marginLeft)
			for _, r := range helpString {
				//c.Write(0, indexY, tw.textColor, tw.bgColor, spaces)
				c.PlotColor(indexX, indexY, tw.titleColor, r)
				indexX++
			}
			w = uint(canvasWidth) - indexX
			for x := indexX; x < w; x++ {
				c.PlotColor(x, indexY, tw.textColor, ' ')
			}

			indexY++

			helpString = "Just start writing. Press return to insert a row below."
			indexX = uint(tw.marginLeft)
			for _, r := range helpString {
				//c.Write(0, indexY, tw.textColor, tw.bgColor, spaces)
				c.PlotColor(indexX, indexY, tw.commentColor, r)
				indexX++
			}
			w = uint(canvasWidth) - indexX
			for x := indexX; x < w; x++ {
				c.PlotColor(x, indexY, tw.textColor, ' ')
			}

			indexY++

			helpString = "Move with tab and the arrow keys. Add and remove columns with ctrl-n and ctrl-p."
			indexX = uint(tw.marginLeft)
			for _, r := range helpString {
				//c.Write(0, indexY, tw.textColor, tw.bgColor, spaces)
				c.PlotColor(indexX, indexY, tw.commentColor, r)
				indexX++
			}
			w = uint(canvasWidth) - indexX
			for x := indexX; x < w; x++ {
				c.PlotColor(x, indexY, tw.textColor, ' ')
			}

			indexY++

			c.Write(0, indexY, tw.textColor, tw.bgColor, spaces)
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
	if tw.cy >= ch {
		tw.cy = 0
	}
	// just in case rows have differing lengths
	if tw.cx >= cw {
		tw.cx = cw - 1
	}
}

// PageUp moves up 10 rows (or to the top if less than 10 rows available)
func (tw *TableWidget) PageUp() {
	_, ch := tw.ContentsWH()
	tw.oldy = tw.cy
	tw.cy -= 10
	if tw.cy < 0 {
		tw.cy = 0
	}
	// Ensure we don't exceed the table bounds
	if tw.cy >= ch {
		tw.cy = ch - 1
	}
}

// PageDown moves down 10 rows (or to the bottom if less than 10 rows available)
func (tw *TableWidget) PageDown() {
	cw, ch := tw.ContentsWH()
	tw.oldy = tw.cy
	tw.cy += 10
	if tw.cy >= ch {
		tw.cy = ch - 1
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
func (tw *TableWidget) NextOrInsert() {
	cw, ch := tw.ContentsWH()
	tw.oldx = tw.cx
	tw.cx++
	if tw.cx >= cw {
		tw.cx = 0
		tw.cy++
		if tw.cy >= ch {
			newRow := make([]string, cw)
			(*tw.contents) = append((*tw.contents), newRow)
			tw.h++     // Update the widget table height as well (this is not the content height)
			tw.cy = ch // old max index + 1
		}
	}
}

// InsertRowBelow will insert a row below this one
func (tw *TableWidget) InsertRowBelow() {
	tw.Ensure1x1()
	cw, _ := tw.ContentsWH()
	tw.cx = 0
	tw.cy++
	newRow := make([]string, cw)
	// Insert the new row at the cy position
	*tw.contents = append(*tw.contents, nil)
	copy((*tw.contents)[tw.cy+1:], (*tw.contents)[tw.cy:])
	(*tw.contents)[tw.cy] = newRow
	tw.h++ // Update the widget table height as well (this is not the content height)
}

// InsertColumnAfter will insert a column after the current tw.x position
func (tw *TableWidget) InsertColumnAfter() {
	// Iterate through each row in the contents
	for rowIndex, row := range *tw.contents {
		// Create a new row with an additional column
		newRow := make([]string, len(row)+1)
		// Copy the values before the current tw.x position
		copy(newRow, row[:tw.cx+1])
		// Set the new column value to an empty string
		newRow[tw.cx+1] = ""
		// Copy the values after the current tw.x position
		copy(newRow[tw.cx+2:], row[tw.cx+1:])
		// Update the row in the contents
		(*tw.contents)[rowIndex] = newRow
	}
	// Update the widget table width
	tw.w++
}

// CurrentRowIsEmpty checks if the current row is empty
func (tw *TableWidget) CurrentRowIsEmpty() bool {
	row := (*tw.contents)[tw.cy]
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// DeleteCurrentRow deletes the current row
func (tw *TableWidget) DeleteCurrentRow() {
	if tw.cy >= 0 && tw.cy < len(*tw.contents) {
		// Remove the current row from the contents
		*tw.contents = append((*tw.contents)[:tw.cy], (*tw.contents)[tw.cy+1:]...)
		tw.cy--
		tw.h-- // Update the widget table height as well (this is not the content height)
	}
	tw.Ensure1x1()
}

// FieldBelowIsEmpty returns true if the field below exists but is empty
func (tw *TableWidget) FieldBelowIsEmpty() bool {
	_, ch := tw.ContentsWH()
	y := tw.cy + 1
	if y >= ch {
		return false
	}
	row := (*tw.contents)[y]
	if tw.cx >= len(row) {
		return false
	}
	return strings.TrimSpace(row[tw.cx]) == ""
}

// DeleteCurrentColumnIfEmpty will delete the current column if all fields are empty
func (tw *TableWidget) DeleteCurrentColumnIfEmpty() error {
	// Check if all fields in the column are empty
	for _, row := range *tw.contents {
		if row[tw.cx] != "" {
			return errors.New("can only delete column if fields are empty")
		}
	}

	// Iterate through each row in the contents
	for rowIndex, row := range *tw.contents {
		// Create a new row without the current column
		newRow := make([]string, len(row)-1)

		// Copy the values before the current tw.x position
		copy(newRow, row[:tw.cx])

		// Copy the values after the current tw.x position
		copy(newRow[tw.cx:], row[tw.cx+1:])

		// Update the row in the contents
		(*tw.contents)[rowIndex] = newRow
	}

	// Update the widget table width
	tw.w--

	// Adjust the current tw.x position if it's at the last column
	if tw.cx >= tw.w {
		tw.cx = tw.w - 1
	}

	tw.Ensure1x1()

	return nil
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

// SelectStart will select the start of the row
func (tw *TableWidget) SelectStart() bool {
	return tw.SelectIndex(0, tw.cy)
}

// SelectEnd will select the start of the row
func (tw *TableWidget) SelectEnd() bool {
	cw, _ := tw.ContentsWH()
	return tw.SelectIndex(cw-1, tw.cy)
}

// Set will change the field contents of the current position
func (tw *TableWidget) Set(field string) {
	tw.Ensure1x1()
	(*tw.contents)[tw.cy][tw.cx] = field
}

// Get will retrieve the contents of the current field
func (tw *TableWidget) Get() string {
	tw.Ensure1x1()
	return (*tw.contents)[tw.cy][tw.cx]
}

// Add will add a string to the current field
func (tw *TableWidget) Add(s string) {
	tw.Ensure1x1()
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
