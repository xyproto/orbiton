package main

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xyproto/vt"
)

func looksLikeTable(line string) bool {
	return strings.Count(line, "|") > 1 || separatorRow(line)
}

// InTableAt checks if it is likely that the given LineIndex is in a Markdown table
func (e *Editor) InTableAt(i LineIndex) bool {
	// If there is a separation line with no "|" or "-" above or below, then this is not a table
	if !looksLikeTable(e.Line(i-1)) && !looksLikeTable(e.Line(i+1)) {
		return false
	}
	return looksLikeTable(e.Line(i))
}

// InTable checks if we are currently in what appears to be a Markdown table
func (e *Editor) InTable() bool {
	// If there is a separation line with no "|" or "-" above or below, then this is not a table
	if !looksLikeTable(e.PrevLine()) && !looksLikeTable(e.NextLine()) {
		return false
	}
	return looksLikeTable(e.CurrentLine())
}

// TopOfCurrentTable tries to find the first line index of the current Markdown table
func (e *Editor) TopOfCurrentTable() (LineIndex, error) {
	startIndex := e.DataY()

	if !e.InTableAt(startIndex) {
		return -1, errors.New("not in a table")
	}

	index := startIndex
	for index >= 0 && e.InTableAt(index) {
		index--
	}

	return index + 1, nil
}

// GoToTopOfCurrentTable tries to jump to the first line of the current Markdown table
func (e *Editor) GoToTopOfCurrentTable(c *vt.Canvas, status *StatusBar, centerCursor bool) LineIndex {
	topIndex, err := e.TopOfCurrentTable()
	if err != nil {
		return 0
	}
	redraw, _ := e.GoTo(topIndex, c, status)
	e.redraw.Store(redraw)
	if redraw && centerCursor {
		e.Center(c)
	}
	return topIndex
}

// CurrentTableY returns the current Y position within the current Markdown table
func (e *Editor) CurrentTableY() (int, error) {
	topIndex, err := e.TopOfCurrentTable()
	if err != nil {
		return -1, err
	}
	currentIndex := e.DataY()

	indexY := int(currentIndex) - int(topIndex)

	// Count the divider Lines, and subtract those
	s, err := e.CurrentTableString()
	if err != nil {
		return indexY, err
	}
	separatorCounter := 0
	for _, line := range strings.Split(s, "\n") {
		if separatorRow(line) {
			separatorCounter++
		}
	}
	indexY -= separatorCounter

	// just a safeguard
	if indexY < 0 {
		indexY = 0
	}

	return indexY, nil
}

// CurrentTableString returns the current Markdown table as a newline separated string, if possible
func (e *Editor) CurrentTableString() (string, error) {
	index, err := e.TopOfCurrentTable()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for e.InTableAt(index) {
		trimmedLine := strings.TrimSpace(e.Line(index))
		sb.WriteString(trimmedLine + "\n")
		index++
	}

	// Return the collected table lines
	return sb.String(), nil
}

// DeleteCurrentTable will delete the current Markdown table
func (e *Editor) DeleteCurrentTable(c *vt.Canvas, status *StatusBar, bookmark *Position) (LineIndex, error) {
	s, err := e.CurrentTableString()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return 0, errors.New("need at least one line of table to be able to remove it")
	}
	const centerCursor = false
	topOfTable := e.GoToTopOfCurrentTable(c, status, centerCursor)
	for range lines {
		e.DeleteLineMoveBookmark(e.LineIndex(), bookmark)
	}
	return topOfTable, nil
}

// ReplaceCurrentTableWith will try to replace the current table with the given string.
// Also moves the current bookmark, if needed.
func (e *Editor) ReplaceCurrentTableWith(c *vt.Canvas, status *StatusBar, bookmark *Position, tableString string) error {
	topOfTable, err := e.DeleteCurrentTable(c, status, bookmark)
	if err != nil {
		return err
	}
	lines := strings.Split(tableString, "\n")
	addNewLine := false
	e.InsertBlock(c, lines, addNewLine)
	e.GoTo(topOfTable, c, status)
	return nil
}

// separatorRow checks if this string looks like a header/body table separator line in Markdown
func separatorRow(s string) bool {
	notEmpty := false
	for _, r := range s {
		switch r {
		case ' ':
		case '-', '|', '\n', '\t':
			notEmpty = true
		default:
			return false
		}
	}
	return notEmpty
}

// Parse a Markdown table into a slice of header strings and a [][]slice of rows and columns
func parseTable(s string) ([]string, [][]string) {

	var (
		headers []string
		body    [][]string
	)

	// Is it a Markdown table without leading "|" and trailing "|" on every row?
	// see https://tableconvert.com/ for an example of the "Use simple Markdown table" style.
	simpleStyle := strings.Contains(s, "\n--")

	for i, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var fields []string
		if simpleStyle {
			fields = strings.Split(line, "|")
		} else {
			fields = strings.Split(line, "|")
			if len(fields) > 2 {
				if strings.TrimSpace(fields[0]) == "" && strings.TrimSpace(fields[len(fields)-1]) == "" {
					fields = fields[1 : len(fields)-1] // skip the first and last slice entry
				}
			}
		}
		// Is this a separator row?
		if separatorRow(line) {
			// skip
			continue
		}
		// Trim spaces from all the fields
		for i := 0; i < len(fields); i++ {
			fields[i] = strings.TrimSpace(fields[i])
		}

		// Assign the parsed fields into either headers or the table body
		if i == 0 {
			headers = fields
		} else {
			body = append(body, fields)
		}
	}

	return headers, body
}

// TableColumnWidths returns a slice of max widths for all columns in the given headers+body table
func TableColumnWidths(headers []string, body [][]string) []int {
	maxColumns := len(headers)
	for _, row := range body {
		if len(row) > maxColumns {
			maxColumns = len(row)
		}
	}

	columnWidths := make([]int, maxColumns)

	// find the width of the longest string per column
	for i, field := range headers {
		if len(field) > columnWidths[i] {
			columnWidths[i] = len(field)
		}
	}

	// find the width of the longest string per column
	for _, row := range body {
		for i, field := range row {
			if len(field) > columnWidths[i] {
				columnWidths[i] = len(field)
			}
		}
	}
	return columnWidths
}

// RightTrimColumns removes the last column of the table, if it only consists of empty strings
func RightTrimColumns(headers *[]string, body *[][]string) {
	if len(*headers) == 0 || len(*body) == 0 {
		return
	}
	if len((*body)[0]) == 0 {
		return
	}

	// There is at least 1 header, 1 row and 1 column

	// Check if the last header cell is empty
	if strings.TrimSpace((*headers)[len(*headers)-1]) != "" {
		return
	}

	// Check if all the last cells per row are empty
	for _, row := range *body {
		if strings.TrimSpace(row[len(row)-1]) != "" {
			return
		}
	}

	// We now know that the last column is empty, for the headers and for all rows

	// Remove the last column of the headers
	*headers = (*headers)[:len(*headers)-1]

	for i := range *body {
		// Remove the last column of this row
		(*body)[i] = (*body)[i][:len((*body)[i])-1]
	}
}

func tableToString(headers []string, body [][]string) string {

	columnWidths := TableColumnWidths(headers, body)

	var sb strings.Builder

	// First output the headers

	sb.WriteString("|")
	for i, field := range headers {
		sb.WriteString(" ")
		sb.WriteString(field)
		if len(field) < columnWidths[i] {
			neededSpaces := columnWidths[i] - len(field)
			spaces := strings.Repeat(" ", neededSpaces)
			sb.WriteString(spaces)
		}
		sb.WriteString(" |")
	}
	sb.WriteString("\n")

	// Then output the separator line

	sb.WriteString("|")
	for _, neededDashes := range columnWidths {
		dashes := strings.Repeat("-", neededDashes+2)
		sb.WriteString(dashes)
		sb.WriteString("|")
	}
	sb.WriteString("\n")

	// Then add the table body

	for _, row := range body {

		// If all fields are empty, then skip this row
		allEmpty := true
		for _, field := range row {
			if strings.TrimSpace(field) != "" {
				allEmpty = false
				break
			}
		}
		if allEmpty {
			continue
		}

		// Write the fields of this row to the string builder
		sb.WriteString("|")
		for i, field := range row {
			sb.WriteString(" ")
			sb.WriteString(field)
			if len(field) < columnWidths[i] {
				neededSpaces := columnWidths[i] - len(field)
				spaces := strings.Repeat(" ", neededSpaces)
				sb.WriteString(spaces)
			}
			sb.WriteString(" |")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatAllMarkdownTables formats all tables without moving the cursor
func (e *Editor) FormatAllMarkdownTables() {
	content := e.String()
	lines := strings.Split(content, "\n")

	formattedLines := make([]string, 0, len(lines))
	inTable := false
	var tableLines []string

	for i, line := range lines {
		if e.InTableAt(LineIndex(i)) {
			inTable = true
			tableLines = append(tableLines, line)
		} else if inTable {
			// End of the table
			headers, body := parseTable(strings.Join(tableLines, "\n"))

			tableContents := [][]string{}
			tableContents = append(tableContents, headers)
			tableContents = append(tableContents, body...)

			// Make all rows contain as many fields as the longest row
			Expand(&tableContents)

			switch len(tableContents) {
			case 0:
				headers = []string{}
				body = [][]string{}
			case 1:
				headers = tableContents[0]
				body = [][]string{}
			default:
				headers = tableContents[0]
				body = tableContents[1:]
			}

			RightTrimColumns(&headers, &body)
			formattedTableString := tableToString(headers, body)

			// Add the formatted table to the result
			formattedLines = append(formattedLines, strings.Split(formattedTableString, "\n")...)

			// Add the current line (which is not part of the table) to the result if it's not empty
			if strings.TrimSpace(line) != "" {
				formattedLines = append(formattedLines, line)
			}

			// Reset the table state
			inTable = false
			tableLines = nil
		} else {
			formattedLines = append(formattedLines, line)
		}
	}

	// Handle case where the last lines are part of a table
	if inTable {
		headers, body := parseTable(strings.Join(tableLines, "\n"))

		tableContents := [][]string{}
		tableContents = append(tableContents, headers)
		tableContents = append(tableContents, body...)

		// Make all rows contain as many fields as the longest row
		Expand(&tableContents)

		switch len(tableContents) {
		case 0:
			headers = []string{}
			body = [][]string{}
		case 1:
			headers = tableContents[0]
			body = [][]string{}
		default:
			headers = tableContents[0]
			body = tableContents[1:]
		}

		RightTrimColumns(&headers, &body)
		formattedTableString := tableToString(headers, body)

		// Add the formatted table to the result
		formattedLines = append(formattedLines, strings.Split(formattedTableString, "\n")...)
	}

	e.Clear()
	for i, line := range formattedLines {
		e.SetLine(LineIndex(i), line)
	}

	e.changed.Store(true)
	e.redraw.Store(true)
}

// EditMarkdownTable presents the user with a dedicated table editor for the current Markdown table, or just formats it
func (e *Editor) EditMarkdownTable(tty *vt.TTY, c *vt.Canvas, status *StatusBar, bookmark *Position, justFormat, displayQuickHelp bool) {

	initialY, err := e.CurrentTableY()
	if err != nil {
		status.ClearAll(c, true)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
	}

	tableString, err := e.CurrentTableString()
	if err != nil {
		status.ClearAll(c, true)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
	}

	headers, body := parseTable(tableString)

	tableContents := [][]string{}
	tableContents = append(tableContents, headers)
	tableContents = append(tableContents, body...)

	// Make all rows contain as many fields as the longest row
	Expand(&tableContents)

	contentsChanged := false

	if !justFormat {
		contentsChanged, err = e.TableEditor(tty, status, &tableContents, initialY, displayQuickHelp)
		if err != nil {
			status.ClearAll(c, true)
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			return
		}
	}

	if justFormat || contentsChanged {
		switch len(tableContents) {
		case 0:
			headers = []string{}
			body = [][]string{}
		case 1:
			headers = tableContents[0]
			body = [][]string{}
		default:
			headers = tableContents[0]
			body = tableContents[1:]
		}

		RightTrimColumns(&headers, &body)
		newTableString := tableToString(headers, body)

		// Replace the current table with this new string
		if err := e.ReplaceCurrentTableWith(c, status, bookmark, newTableString); err != nil {
			status.ClearAll(c, true)
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			return
		}

	}
}

// TableEditor presents an interface for changing the given headers and body
// initialY is the initial Y position of the cursor in the table
// Returns true if the user changed the contents.
func (e *Editor) TableEditor(tty *vt.TTY, status *StatusBar, tableContents *[][]string, initialY int, displayQuickHelp bool) (bool, error) {

	title := "Markdown Table Editor"
	titleColor := e.Foreground // HeaderBulletColor
	headerColor := e.XColor
	textColor := e.MarkdownTextColor
	highlightColor := e.MenuArrowColor
	cursorColor := e.SearchHighlight
	commentColor := e.CommentColor
	userChangedTheContents := false

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	var (
		c           = vt.NewCanvas()
		tableWidget = NewTableWidget(title, tableContents, titleColor, headerColor, textColor, highlightColor, cursorColor, commentColor, e.Background, int(c.W()), int(c.H()), initialY, displayQuickHelp)
		sigChan     = make(chan os.Signal, 1)
		running     = true
		changed     = true
		cancel      = false
	)

	// Set up a new resize handler
	signal.Notify(sigChan, syscall.SIGWINCH)

	resizeRedrawFunc := func() {
		// Create a new canvas, with the new size
		nc := c.Resized()
		if nc != nil {
			vt.Clear()
			c = nc
			tableWidget.Draw(c)
			c.HideCursorAndRedraw()
			changed = true
		}
	}

	go func() {
		for range sigChan {
			resizeMut.Lock()
			resizeRedrawFunc()
			resizeMut.Unlock()
		}
	}()

	vt.Clear()
	vt.Reset()
	c.HideCursorAndRedraw()

	showMessage := func(msg string, color vt.AttributeColor) {
		msgX := (c.W() - uint(len(msg))) / 2
		msgY := c.H() - 1
		c.Write(msgX, msgY, color, e.Background, msg)
		go func() {
			time.Sleep(1 * time.Second)
			s := strings.Repeat(" ", len(msg))
			c.Write(msgX, msgY, textColor, e.Background, s)
		}()
	}

	for running {

		// Draw elements in their new positions

		if changed {
			resizeMut.RLock()
			tableWidget.Draw(c)
			resizeMut.RUnlock()
			// Update the canvas
			c.HideCursorAndDraw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case upArrow: // Up
			resizeMut.Lock()
			tableWidget.Up()
			changed = true
			resizeMut.Unlock()
		case leftArrow: // Left
			resizeMut.Lock()
			tableWidget.Left()
			changed = true
			resizeMut.Unlock()
		case downArrow: // Down
			resizeMut.Lock()
			tableWidget.Down()
			changed = true
			resizeMut.Unlock()
		case rightArrow: // Right
			resizeMut.Lock()
			tableWidget.Right()
			changed = true
			resizeMut.Unlock()
		case "c:9": // Next, tab
			resizeMut.Lock()
			tableWidget.NextOrInsert()
			changed = true
			resizeMut.Unlock()
		case "c:1": // Start of row, ctrl-a
			resizeMut.Lock()
			tableWidget.SelectStart()
			changed = true
			resizeMut.Unlock()
		case "c:5": // End of row, ctrl-e
			resizeMut.Lock()
			tableWidget.SelectEnd()
			changed = true
			resizeMut.Unlock()
		case "c:27", "q", "c:3", "c:17", "c:15", "c:20": // ESC, q, ctrl-c, ctrl-q, ctrl-o or ctrl-t
			running = false
			changed = true
			cancel = true
		case "c:19": // ctrl-s, save
			resizeMut.Lock()
			// Try to save the file
			if err := e.Save(c, tty); err != nil {
				// TODO: Use a StatusBar instead, then draw it at the end of the loop
				showMessage(err.Error(), cursorColor)
			} else {
				showMessage("Saved", cursorColor)
			}
			changed = true
			resizeMut.Unlock()
		case "c:13": // return, insert a row below
			resizeMut.Lock()
			if tableWidget.FieldBelowIsEmpty() {
				tableWidget.Down()
			} else {
				tableWidget.InsertRowBelow()
				changed = true
				userChangedTheContents = true
			}
			resizeMut.Unlock()
		case "c:14": // ctrl-n, insert column after
			resizeMut.Lock()
			tableWidget.InsertColumnAfter()
			tableWidget.NextOrInsert()
			changed = true
			userChangedTheContents = true
			resizeMut.Unlock()
		case "c:4", "c:16": // ctrl-d or ctrl-p, delete the current column if all its fields are empty
			resizeMut.Lock()
			if err := tableWidget.DeleteCurrentColumnIfEmpty(); err != nil {
				// TODO: Use a StatusBar instead, then draw it at the end of the loop
				showMessage(err.Error(), cursorColor)
			} else {
				changed = true
				userChangedTheContents = true
			}
			resizeMut.Unlock()
		case "c:8", "c:127": // ctrl-h or backspace
			resizeMut.Lock()
			s := tableWidget.Get()
			if len(s) > 0 {
				tableWidget.Set(s[:len(s)-1])
				changed = true
				userChangedTheContents = true
			} else if tableWidget.CurrentRowIsEmpty() {
				tableWidget.DeleteCurrentRow()
				changed = true
				userChangedTheContents = true
			}
			resizeMut.Unlock()
		default:
			resizeMut.Lock()
			if !strings.HasPrefix(key, "c:") {
				tableWidget.Add(key)
				changed = true
				userChangedTheContents = true
			}
			resizeMut.Unlock()
		}

		// If the menu was changed, draw the canvas
		if changed {
			c.HideCursorAndDraw()
		}

		if cancel {
			tableWidget.TrimAll()
			break
		}
	}

	// Restore the signal handlers
	e.SetUpSignalHandlers(c, tty, status, false) // do not only clear the signals

	return userChangedTheContents, nil
}
