package main

import (
	"strings"
)

// reflowLines will attempt to reflow all given lines to the given maximum width.
// Long words may exceed this width.
// Also, given an y position, return the new y position after everything has been reflown
// TODO: Also return a new x position, not just Y.
func reflowLines(document string, givenY, maxWidth int) ([]string, int) {
	lines := make([]string, 0)
	i := 0
	newY := givenY
	for lineY, line := range strings.Split(document, "\n") {
		if strings.TrimSpace(line) == "" {
			lines = append(lines, "")
			i++
			if lineY == givenY {
				newY = i
			}
			lines = append(lines, "")
			i++
			continue
		}
		for _, field := range strings.Fields(line) {
			if len(lines) == i {
				lines = append(lines, field)
				if lineY == givenY {
					newY = i
				}
				continue
			}
			if len(lines[i]+" "+field) < maxWidth {
				lines[i] = lines[i] + " " + field
				continue
			}
			i++
			lines = append(lines, field)
		}
	}
	return lines, newY
}

// Reflow will reflow all lines to a given maximum width.
// Long words may exceed this width.
func (e *Editor) Reflow(maxWidth int) {
	lines, newY := reflowLines(e.String(), e.DataY(), e.wordWrapAt)
	// Clear the current editor contents
	e.Clear()
	for y, line := range lines {
		counter := 0
		for _, letter := range line {
			e.Set(counter, int(y), letter)
			counter++
		}
	}
	e.pos.SetY(newY)
	if e.AtOrAfterEndOfLine() {
		e.End()
	}
	// Mark the data as "changed"
	e.changed = true
}

// WrapAllLinesAt will word wrap all lines that are longer than n,
// with a maximum overshoot of too long words (measured in runes) of maxOvershoot.
// Returns true if any lines were wrapped.
func (e *Editor) WrapAllLinesAt(n, maxOvershoot int) bool {
	wrapped := false
	for i := 0; i < e.Len(); i++ {
		if e.WithinLimit(i) {
			continue
		}
		wrapped = true
		first, second := e.SplitOvershoot(i, false)
		if len(first) > 0 && len(second) > 0 {
			e.InsertLineBelowAt(i)
			e.lines[i] = first
			e.lines[i+1] = second
			e.changed = true
			// Move the cursor as well, so that it is at the same line as before the word wrap
			if i < e.DataY() {
				e.pos.sy++
			}
		}
	}
	return wrapped
}
