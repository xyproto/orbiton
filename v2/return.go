package main

import (
	"strings"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// ReturnPressed is called when the user pressed return while editing text
func (e *Editor) ReturnPressed(c *vt.Canvas, status *StatusBar) {
	var (
		trimmedLine              = e.TrimmedLine()
		currentLeadingWhitespace = e.LeadingWhitespace()
		// Grab the leading whitespace from the current line, and indent depending on the end of trimmedLine
		leadingWhitespace = e.smartIndentation(currentLeadingWhitespace, trimmedLine, false) // the last parameter is "also dedent"

		noHome = false
		indent = true
	)

	// TODO: add and use something like "e.shouldAutoIndent" for these file types
	if e.mode == mode.Markdown || e.mode == mode.Text || e.mode == mode.Blank {
		indent = false
	}

	if trimmedLine == "private:" || trimmedLine == "protected:" || trimmedLine == "public:" {
		// De-indent the current line before moving on to the next
		e.SetCurrentLine(trimmedLine)
		leadingWhitespace = currentLeadingWhitespace
	} else if e.handleReturnAutocomplete(c, trimmedLine, currentLeadingWhitespace, &indent, &leadingWhitespace) {
		// handled by autocomplete helpers
	} else if cLikeSwitch(e.mode) {
		currentLine := e.CurrentLine()
		trimmedLine := e.TrimmedLine()
		// De-indent this line by 1 if this line starts with "case " and the next line also starts with "case ", but the current line is indented differently.
		currentCaseIndex := strings.Index(trimmedLine, "case ")
		nextCaseIndex := strings.Index(e.NextTrimmedLine(), "case ")
		if currentCaseIndex != -1 && nextCaseIndex != -1 && strings.Index(currentLine, "case ") != strings.Index(e.NextLine(), "case ") {
			oneIndentation := e.indentation.String()
			deIndented := strings.Replace(currentLine, oneIndentation, "", 1)
			e.SetCurrentLine(deIndented)
			e.End(c)
			leadingWhitespace = currentLeadingWhitespace
		}
	}

	scrollBack := false

	// TODO: Collect the criteria that trigger the same behavior

	switch {
	case e.AtOrAfterLastLineOfDocument() && (e.AtStartOfTheLine() || e.AtOrBeforeStartOfTextScreenLine()):
		e.InsertLineAbove()
		noHome = true
	case e.AtOrAfterEndOfDocument() && !e.AtStartOfTheLine() && !e.AtOrAfterEndOfLine():
		e.InsertStringAndMove(c, "")
		e.InsertLineBelow()
		scrollBack = true
	case e.AfterEndOfLine():
		e.InsertLineBelow()
		scrollBack = true
	case !e.AtFirstLineOfDocument() && e.AtOrAfterLastLineOfDocument() && (e.AtStartOfTheLine() || e.AtOrAfterEndOfLine()):
		e.InsertStringAndMove(c, "")
		scrollBack = true
	case e.AtStartOfTheLine():
		e.InsertLineAbove()
		noHome = true
	default:
		// Split the current line in two
		if !e.SplitLine() {
			e.InsertLineBelow()
		}
		scrollBack = true
		// Indent the next line if at the end, not else
		if !e.AfterEndOfLine() {
			indent = false
		}
	}
	e.MakeConsistent()

	h := int(c.Height())
	if e.pos.sy > (h - 1) {
		e.pos.Down(c)
		e.redraw.Store(e.ScrollDown(c, status, 1, h))
		e.redrawCursor.Store(true)
	} else if e.pos.sy == (h - 1) {
		e.redraw.Store(e.ScrollDown(c, status, 1, h))
		e.redrawCursor.Store(true)
	} else {
		e.pos.Down(c)
	}

	if !noHome {
		e.pos.mut.Lock()
		e.pos.sx = 0
		e.pos.mut.Unlock()
		// e.Home()
		if scrollBack {
			e.pos.SetX(c, 0)
		}
	}

	if indent && len(leadingWhitespace) > 0 {
		// If the leading whitespace starts with a tab and ends with a space, remove the final space
		if strings.HasPrefix(leadingWhitespace, "\t") && strings.HasSuffix(leadingWhitespace, " ") {
			leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-1]
		}
		if !noHome {
			// Insert the same leading whitespace for the new line
			e.SetCurrentLine(leadingWhitespace + e.LineContentsFromCursorPosition())
			// Then move to the start of the text
			e.GoToStartOfTextLine(c)
		}
	}

	e.SaveX(true)
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}
