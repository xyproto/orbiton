package main

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

// InTable checks if it is likely that the given LineIndex is in a Markdown table
func (e *Editor) InTable(i LineIndex) bool {
	line := e.Line(i)
	return strings.Count(line, "|") > 1 || strings.Contains(line, "---") || strings.Count(line, "-") > 4
}

// TopOfCurrentTable tries to find the first line index of the current Markdown table
func (e *Editor) TopOfCurrentTable() (LineIndex, error) {
	startIndex := e.DataY()

	if !e.InTable(startIndex) {
		return -1, errors.New("not in a table")
	}

	index := startIndex
	for index >= 0 && e.InTable(index) {
		index--
	}

	return index + 1, nil
}

// GoToTopOfCurrentTable tries to jump to the first line of the current Markdown table
func (e *Editor) GoToTopOfCurrentTable(c *vt100.Canvas, status *StatusBar) {
	topIndex, err := e.TopOfCurrentTable()
	if err != nil {
		return
	}
	e.redraw, _ = e.GoTo(topIndex, c, status)
	if e.redraw {
		e.Center(c)
	}
}

// CurrentTableString returns the current Markdown table as a newline separated string, if possible
func (e *Editor) CurrentTableString() (string, error) {
	index, err := e.TopOfCurrentTable()
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	for e.InTable(index) {
		trimmedLine := strings.TrimSpace(e.Line(index))
		sb.WriteString(trimmedLine + "\n")
		index++
	}

	// Return the collected table lines
	return sb.String(), nil
}

// DeleteCurrentTable will delete the current Markdown table
func (e *Editor) DeleteCurrentTable(c *vt100.Canvas, status *StatusBar, bookmark *Position) error {
	s, err := e.CurrentTableString()
	if err != nil {
		return err
	}
	lines := strings.Split(s, "\n")
	if len(lines) == 0 {
		return errors.New("need at least one line of table to be able to remove it")
	}
	e.GoToTopOfCurrentTable(c, status)
	for range lines {
		e.DeleteLineMoveBookmark(e.LineIndex(), bookmark)
	}
	return nil
}

// ReplaceCurrentTableWith will try to replace the current table with the given string.
// Also moves the current bookmark, if needed.
func (e *Editor) ReplaceCurrentTableWith(c *vt100.Canvas, status *StatusBar, bookmark *Position, s string) error {
	if err := e.DeleteCurrentTable(c, status, bookmark); err != nil {
		return err
	}
	lines := strings.Split(s, "\n")
	addNewLine := false
	e.InsertBlock(c, lines, addNewLine)
	e.Up(c, status)
	e.GoToTopOfCurrentTable(c, status)
	return nil
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
		if strings.Contains(line, "---") || strings.Count(line, "-") > 5 {
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

func tableToString(headers []string, body [][]string) string {
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
func (e *Editor) EditMarkdownTable(c *vt100.Canvas, status *StatusBar, bookmark *Position) {
	tableString, err := e.CurrentTableString()
	if err != nil {
		status.ClearAll(c)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
	}

	headers, body := parseTable(tableString)

	if err := e.TableEditorMode(headers, body); err != nil {
		status.ClearAll(c)
		status.SetError(err)
		status.ShowNoTimeout(c, e)
		return
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

// TableEditorMode presents an interface for changing the given headers and body
func (e *Editor) TableEditorMode(headers []string, body [][]string) error {
	// TODO: Change the headers and body
	// TODO: Look at the symbol selection UI

	// No editor mode just yet
	return nil
}
