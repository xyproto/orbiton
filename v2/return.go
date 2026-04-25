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

	// In book mode, never indent and ensure clean paragraphs
	if e.bookMode.Load() {
		indent = false
	}

	afterParagraph := false
	if e.bookMode.Load() && e.AtOrAfterEndOfLine() {
		afterParagraph = true
	}

	// In book mode or Markdown mode, detect list prefixes for auto-continuation.
	var listPrefix string
	if e.bookMode.Load() && e.AtOrAfterEndOfLine() {
		rawLine := e.CurrentLine()
		pfx := rawMarkdownPrefix(strings.ReplaceAll(rawLine, "\t", "    "))
		if pfx != "" {
			body := strings.TrimRight(rawLine, " \t")
			if len(body) <= len(pfx) {
				// The line is just a prefix with no body text — end the list.
				// Clear the prefix and fall through to normal Return handling
				// so a new line is inserted and the cursor moves down.
				e.SetCurrentLine("")
				e.Home()
			} else {
				listPrefix = nextListPrefix(pfx)
			}
		}
	}

	if trimmedLine == "private:" || trimmedLine == "protected:" || trimmedLine == "public:" {
		// De-indent the current line before moving on to the next
		e.SetCurrentLine(trimmedLine)
		leadingWhitespace = currentLeadingWhitespace
	} else if !e.handleReturnAutocomplete(c, trimmedLine, currentLeadingWhitespace, &indent, &leadingWhitespace) && cLikeSwitch(e.mode) {
		currentLine := e.CurrentLine()
		trimmedLine := e.TrimmedLine()
		// De-indent this line by 1 if this line starts with "case " and the next line also starts with "case ", but the current line is indented differently.
		found := strings.Contains(trimmedLine, "case ")
		nextCaseIndex := strings.Index(e.NextTrimmedLine(), "case ")
		if found && nextCaseIndex != -1 && strings.Index(currentLine, "case ") != strings.Index(e.NextLine(), "case ") {
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
		// An empty last line needs InsertLineBelow — InsertLineAbove would
		// trim the trailing empty line back off, resulting in a net no-op
		if e.AtLastLineOfDocument() && e.EmptyLine() {
			e.InsertLineBelow()
			scrollBack = true
		} else {
			e.InsertLineAbove()
			noHome = true
		}
	case e.AtOrAfterEndOfDocument() && !e.AtStartOfTheLine() && !e.AtOrAfterEndOfLine():
		e.InsertLineBelow()
		scrollBack = true
	case e.AfterEndOfLine():
		e.InsertLineBelow()
		scrollBack = true
	case !e.AtFirstLineOfDocument() && e.AtOrAfterLastLineOfDocument() && (e.AtStartOfTheLine() || e.AtOrAfterEndOfLine()):
		// End of the last line: insert a blank line below so the user can
		// actually append lines at the end of the document
		e.InsertLineBelow()
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
	// In graphical book mode the bottom terminal row belongs to the status bar,
	// so the effective editing height is one row shorter.
	editH := h
	if e.bookGraphicalMode() {
		editH--
	}
	if e.bookMode.Load() {
		// In book mode, pos.sy is a data-line offset; the canvas-row
		// scroll math does not account for soft wrap. Advance DataY
		// directly and let bookModeEnsureCursorVisible handle scrolling
		e.pos.Down(c)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
	} else if e.pos.sy > (editH - 1) {
		e.pos.Down(c)
		e.redraw.Store(e.ScrollDown(c, status, 1, editH))
		e.redrawCursor.Store(true)
	} else if e.pos.sy == (editH - 1) {
		e.redraw.Store(e.ScrollDown(c, status, 1, editH))
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

	// Auto-continue Markdown lists: insert the next prefix on the new line
	if listPrefix != "" {
		e.SetCurrentLine(listPrefix + e.LineContentsFromCursorPosition())
		// End() would trim trailing whitespace and strip the space from "* "
		e.pos.SetX(c, len([]rune(listPrefix)))
	}

	// Book-mode: after a heading, insert an extra blank line below so the
	// cursor ends up in a clean paragraph two rows down from the heading.
	if afterParagraph {
		e.InsertLineBelow()
		e.pos.Down(c)
		e.pos.SetX(c, 0)
	}

	e.SaveX(true)
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}
