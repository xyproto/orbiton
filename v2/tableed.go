package main

import (
	"errors"
	"strings"
)

// CurrentTable returns the current Markdown table as a newline separated string, if possible
func (e *Editor) CurrentTable() (string, error) {
	currentY := e.DataY()
	line := e.Line(currentY)
	if !strings.Contains(line, "|") {
		return "", errors.New("not in a table")
	}
	var sb strings.Builder
	topY := currentY
	for i := currentY - 1; i >= 0; i-- {
		// Check if this line contains "|"
		if !strings.Contains(e.Line(i), "|") {
			break
		}
		topY = i
	}
	i := topY
	for line := e.Line(i); strings.Contains(line, "|"); i++ {
		if i <= 0 {
			break
		}
		sb.WriteString(strings.TrimSpace(line) + "\n")
	}
	return sb.String(), nil
}

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
