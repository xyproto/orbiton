package main

import (
	"errors"
	"strings"

	"github.com/xyproto/vt100"
)

// InTable checks if it is likely that the given LineIndex is in a Markdown table
func (e *Editor) InTable(i LineIndex) bool {
	line := e.Line(i)
	return strings.Count(line, "|") > 1 || strings.Contains(line, "--")
}

// GoToTableTop will move up as long as the current line contains "|", until it can not move further up.
// Can be used for Markdown tables.
func (e *Editor) GoToTableTop(c *vt100.Canvas, status *StatusBar) error {
	startIndex := e.DataY()

	if !e.InTable(startIndex) {
		return errors.New("not in a table")
	}

	index := startIndex
	for ; index >= 0; index-- {
		if !e.InTable(index) {
			index++
			break
		}
	}

	// index is now the first found line of the table

	if index != startIndex {
		e.GoTo(index, c, status)
	}

	return nil
}

// CurrentTable returns the current Markdown table as a newline separated string, if possible
func (e *Editor) CurrentTable() (string, error) {
	startIndex := e.DataY()

	if !e.InTable(startIndex) {
		return "", errors.New("not in a table")
	}

	index := startIndex
	for ; index >= 0; index-- {
		if !e.InTable(index) {
			index++
			break
		}
	}

	// index is now the first found line of the table

	// Now collect all the lines of the table
	var sb strings.Builder
	for i := index; ; i++ {
		if !e.InTable(startIndex) {
			break
		}
		trimmedLine := strings.TrimSpace(e.Line(i))
		sb.WriteString(trimmedLine + "\n")
	}

	// Return the collected table lines
	return sb.String(), nil
}

// EditMarkdownTable presents the user with a dedicated table editor for the current Markdown table
func (e *Editor) EditMarkdownTable() {
	s, err := e.CurrentTable()
	if err != nil {
		logf("ERROR: %v\n", err)
		return
	}
	logf("CURRENT TABLE:\n%s\n", s)
	// 1. Find the entire table
	// 2. Present an UI for editing the table
}
