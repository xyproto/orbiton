package main

import (
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/xyproto/vt100"
)

// InTableAt checks if it is likely that the given LineIndex is in a Markdown table
func (e *Editor) InTableAt(i LineIndex) bool {
	line := e.Line(i)
	return strings.Count(line, "|") > 1 || separatorRow(line)
}

// InTable checks if we are currently in what appears to be a Markdown table
func (e *Editor) InTable() bool {
	line := e.CurrentLine()
	return strings.Count(line, "|") > 1 || separatorRow(line)
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
func (e *Editor) GoToTopOfCurrentTable(c *vt100.Canvas, status *StatusBar, centerCursor bool) LineIndex {
	topIndex, err := e.TopOfCurrentTable()
	if err != nil {
		return 0
	}
	e.redraw, _ = e.GoTo(topIndex, c, status)
	if e.redraw && centerCursor {
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
func (e *Editor) DeleteCurrentTable(c *vt100.Canvas, status *StatusBar, bookmark *Position) (LineIndex, error) {
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
func (e *Editor) ReplaceCurrentTableWith(c *vt100.Canvas, status *StatusBar, bookmark *Position, tableString string) error {
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

// EditMarkdownTable presents the user with a dedicated table editor for the current Markdown table
func (e *Editor) EditMarkdownTable(tty *vt100.TTY, c *vt100.Canvas, status *StatusBar, bookmark *Position, justFormat bool) {

	initialY, err := e.CurrentTableY()
	if err != nil {
		status.ClearAll(c)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
	}

	tableString, err := e.CurrentTableString()
	if err != nil {
		status.ClearAll(c)
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

	if !justFormat {
		if err := e.TableEditor(tty, status, &tableContents, initialY); err != nil {
			status.ClearAll(c)
			status.SetError(err)
			status.ShowNoTimeout(c, e)
			return
		}
	}

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

	newTableString := tableToString(headers, body)

	// Replace the current table with this new string
	if err := e.ReplaceCurrentTableWith(c, status, bookmark, newTableString); err != nil {
		status.ClearAll(c)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
	}
}

// TableEditor presents an interface for changing the given headers and body
// initialY is the initial Y position of the cursor in the table
func (e *Editor) TableEditor(tty *vt100.TTY, status *StatusBar, tableContents *[][]string, initialY int) error {

	title := "Markdown Table Editor"
	titleColor := e.MenuArrowColor
	headerColor := e.HeaderTextColor
	textColor := e.MenuTextColor
	highlightColor := e.MenuSelectedColor
	cursorColor := e.SearchHighlight
	commentColor := e.CommentColor

	// Clear the existing handler
	signal.Reset(syscall.SIGWINCH)

	var (
		c           = vt100.NewCanvas()
		tableWidget = NewTableWidget(title, tableContents, titleColor, headerColor, textColor, highlightColor, cursorColor, commentColor, e.Background, int(c.W()), int(c.H()), initialY)
		sigChan     = make(chan os.Signal, 1)
		running     = true
		changed     = true
		cancel      = false
	)

	// Set up a new resize handler
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		for range sigChan {
			resizeMut.Lock()

			// Create a new canvas, with the new size
			nc := c.Resized()
			if nc != nil {
				vt100.Clear()
				c = nc
				tableWidget.Draw(c)
				c.Redraw()
				changed = true
			}

			// Inform all elements that the terminal was resized
			resizeMut.Unlock()
		}
	}()

	vt100.Clear()
	vt100.Reset()
	c.Redraw()

	for running {

		// Draw elements in their new positions

		if changed {
			resizeMut.RLock()
			tableWidget.Draw(c)
			resizeMut.RUnlock()
			// Update the canvas
			c.Draw()
		}

		// Handle events
		key := tty.String()
		switch key {
		case "↑", "c:16": // Up or ctrl-p
			resizeMut.Lock()
			tableWidget.Up()
			changed = true
			resizeMut.Unlock()
		case "←": // Left
			resizeMut.Lock()
			tableWidget.Left()
			changed = true
			resizeMut.Unlock()
		case "↓", "c:14": // Down or ctrl-n
			resizeMut.Lock()
			tableWidget.Down()
			changed = true
			resizeMut.Unlock()
		case "→": // Right
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
		case "c:27", "q", "c:3", "c:17", "c:15", "c:19", "c:20": // ESC, q, ctrl-c, ctrl-q, ctrl-o, ctrl-s or ctrl-t
			running = false
			changed = true
			cancel = true
		case "c:13": // return, insert a row below
			resizeMut.Lock()
			tableWidget.InsertRowBelow()
			changed = true
			resizeMut.Unlock()
		case "c:8", "c:127": // ctrl-h or backspace
			resizeMut.Lock()
			s := tableWidget.Get()
			if len(s) > 0 {
				tableWidget.Set(s[:len(s)-1])
				changed = true
			} else if tableWidget.CurrentRowIsEmpty() {
				tableWidget.DeleteCurrentRow()
				changed = true
			}
			resizeMut.Unlock()
		default:
			resizeMut.Lock()
			if !strings.HasPrefix(key, "c:") {
				tableWidget.Add(key)
				changed = true
			}
			resizeMut.Unlock()
		}

		// If the menu was changed, draw the canvas
		if changed {
			c.Draw()
		}

		if cancel {
			tableWidget.TrimAll()
			break
		}

	}

	// Restore the signal handlers
	e.SetUpSignalHandlers(c, tty, status)

	return nil
}
