package main

import (
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/xyproto/vt"
)

// Selection tracks an active text selection using display coordinates.
// displayX values are screen X positions after tab expansion.
// Y values are data (document) line indices
type Selection struct {
	anchorY        LineIndex // anchor line (where selection started)
	anchorDisplayX int       // anchor display X (screen column after tab expansion)
	activeY        LineIndex // active end line (moves with cursor)
	activeDisplayX int       // active end display X
}

// start returns the normalized start of the selection (earlier position in document order)
func (s *Selection) start() (LineIndex, int) {
	if s.anchorY < s.activeY || (s.anchorY == s.activeY && s.anchorDisplayX <= s.activeDisplayX) {
		return s.anchorY, s.anchorDisplayX
	}
	return s.activeY, s.activeDisplayX
}

// end returns the normalized end of the selection (later position, exclusive)
func (s *Selection) end() (LineIndex, int) {
	if s.anchorY < s.activeY || (s.anchorY == s.activeY && s.anchorDisplayX <= s.activeDisplayX) {
		return s.activeY, s.activeDisplayX
	}
	return s.anchorY, s.anchorDisplayX
}

// ContainsPos returns true if the given (dataY, displayX) falls within this selection.
// displayX is the rune index into the tab-expanded line (same as runeIndex in WriteLines)
func (s *Selection) ContainsPos(dataY LineIndex, displayX int) bool {
	startY, startX := s.start()
	endY, endX := s.end()
	if dataY < startY || dataY > endY {
		return false
	}
	if startY == endY {
		return displayX >= startX && displayX < endX
	}
	if dataY == startY {
		return displayX >= startX
	}
	if dataY == endY {
		return displayX < endX
	}
	return true // middle line — fully selected
}

// IsEmpty returns true if the selection has no extent
func (s *Selection) IsEmpty() bool {
	return s.anchorY == s.activeY && s.anchorDisplayX == s.activeDisplayX
}

// Text returns the selected text extracted from the editor's document lines
func (s *Selection) Text(e *Editor) string {
	startY, startX := s.start()
	endY, endX := s.end()
	var sb strings.Builder
	for y := startY; y <= endY; y++ {
		if y > startY {
			sb.WriteRune('\n')
		}
		runes, ok := e.lines[int(y)]
		if !ok {
			continue
		}
		var lineStartX, lineEndX int
		if y == startY {
			lineStartX = e.displayXToDataX(y, startX)
		}
		if y == endY {
			lineEndX = e.displayXToDataX(y, endX)
		} else {
			lineEndX = len(runes)
		}
		if lineStartX > len(runes) {
			lineStartX = len(runes)
		}
		if lineEndX > len(runes) {
			lineEndX = len(runes)
		}
		if lineStartX < lineEndX {
			sb.WriteString(string(runes[lineStartX:lineEndX]))
		}
	}
	return sb.String()
}

// displayXToDataX converts a display X (tab-expanded column) to a rune index in the given line
func (e *Editor) displayXToDataX(y LineIndex, displayX int) int {
	runes, ok := e.lines[int(y)]
	if !ok {
		return 0
	}
	col := 0
	for i, r := range runes {
		if col >= displayX {
			return i
		}
		if r == '\t' {
			col += e.indentation.PerTab
		} else {
			col += runewidth.RuneWidth(r)
		}
	}
	return len(runes)
}

// currentDisplayX returns the cursor's total display X (sx + offsetX)
func (e *Editor) currentDisplayX() int {
	return e.pos.sx + e.pos.offsetX
}

// StartSelection starts a new selection anchored at the current cursor position
func (e *Editor) StartSelection() {
	e.selection = &Selection{
		anchorY:        e.DataY(),
		anchorDisplayX: e.currentDisplayX(),
		activeY:        e.DataY(),
		activeDisplayX: e.currentDisplayX(),
	}
}

// UpdateSelection moves the active end of the selection to the current cursor position
func (e *Editor) UpdateSelection() {
	if e.selection == nil {
		return
	}
	e.selection.activeY = e.DataY()
	e.selection.activeDisplayX = e.currentDisplayX()
}

// ClearSelection clears any active selection
func (e *Editor) ClearSelection() {
	e.selection = nil
}

// HasSelection returns true if there is a non-empty selection
func (e *Editor) HasSelection() bool {
	return e.selection != nil && !e.selection.IsEmpty()
}

// DeleteSelection removes the selected text from the document and moves the cursor
// to the start of the (now-deleted) selection
func (e *Editor) DeleteSelection(c *vt.Canvas, status *StatusBar) {
	if e.selection == nil || e.selection.IsEmpty() {
		return
	}
	startY, startDisplayX := e.selection.start()
	endY, endDisplayX := e.selection.end()

	startDataX := e.displayXToDataX(startY, startDisplayX)
	endDataX := e.displayXToDataX(endY, endDisplayX)

	if startY == endY {
		// Single-line deletion: remove runes from startDataX to endDataX
		runes, ok := e.lines[int(startY)]
		if ok {
			newRunes := make([]rune, 0, len(runes))
			newRunes = append(newRunes, runes[:startDataX]...)
			if endDataX < len(runes) {
				newRunes = append(newRunes, runes[endDataX:]...)
			}
			e.lines[int(startY)] = newRunes
		}
	} else {
		// Multi-line deletion:
		// 1. Merge the prefix of startY with the suffix of endY
		// 2. Delete all lines from startY+1 to endY using DeleteLine (which handles shifting)
		startRunes := e.lines[int(startY)]
		endRunes := e.lines[int(endY)]

		var prefix []rune
		if startDataX <= len(startRunes) {
			prefix = append([]rune{}, startRunes[:startDataX]...)
		}
		var suffix []rune
		if endDataX <= len(endRunes) {
			suffix = endRunes[endDataX:]
		}

		merged := make([]rune, len(prefix)+len(suffix))
		copy(merged, prefix)
		copy(merged[len(prefix):], suffix)
		e.lines[int(startY)] = merged

		// Delete lines from endY down to startY+1 (in reverse so indices stay valid)
		for y := endY; y > startY; y-- {
			e.DeleteLine(y)
		}
		e.MakeConsistent()
	}

	// Move cursor to the start of the deleted selection
	e.GoTo(startY, c, status)
	e.pos.SetX(c, startDisplayX)
	e.changed.Store(true)
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}
