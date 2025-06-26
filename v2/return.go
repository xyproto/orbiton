package main

import (
	"path/filepath"
	"strings"

	"github.com/xyproto/iferr"
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
	} else if cLikeFor(e.mode) {
		// Add missing parenthesis for "if ... {", "} else if", "} elif", "for", "while" and "when" for C-like languages
		for _, kw := range []string{"for", "foreach", "foreach_reverse", "if", "switch", "when", "while", "while let", "} else if", "} elif"} {
			if strings.HasPrefix(trimmedLine, kw+" ") && !strings.HasPrefix(trimmedLine, kw+" (") {
				kwLenPlus1 := len(kw) + 1
				if kwLenPlus1 < len(trimmedLine) {
					if strings.HasSuffix(trimmedLine, " {") && kwLenPlus1 < len(trimmedLine) && len(trimmedLine) > 3 {
						// Add ( and ), keep the final "{"
						e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[kwLenPlus1:len(trimmedLine)-2] + ") {")
						e.pos.mut.Lock()
						e.pos.sx += 2
						e.pos.mut.Unlock()
					} else if !strings.HasSuffix(trimmedLine, ")") {
						// Add ( and ), there is no final "{"
						e.SetCurrentLine(currentLeadingWhitespace + kw + " (" + trimmedLine[kwLenPlus1:] + ")")
						e.pos.mut.Lock()
						e.pos.sx += 2
						e.pos.mut.Unlock()
						indent = true
						leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
					}
				}
			}
		}
	} else if (e.mode == mode.Go || e.mode == mode.Odin) && trimmedLine == "iferr" {
		oneIndentation := e.indentation.String()
		// default "if err != nil" block if iferr.IfErr can not find a more suitable one
		ifErrBlock := "if err != nil {\n" + oneIndentation + "return nil, err\n" + "}\n"
		// search backwards for "func ", return the full contents, the resulting line index and if it was found
		contents, functionLineIndex, found := e.ContentsAndReverseSearchPrefix("func ")
		if found {
			// count the bytes from the start to the end of the "func " line, since this is what iferr.IfErr uses
			byteCount := 0
			for i := LineIndex(0); i <= functionLineIndex; i++ {
				byteCount += len(e.Line(i))
			}
			// fetch a suitable "if err != nil" block for the current function signature
			if generatedIfErrBlock, err := iferr.IfErr([]byte(contents), byteCount); err == nil { // success
				ifErrBlock = generatedIfErrBlock
			}
		}
		// insert the block of text
		for i, line := range strings.Split(strings.TrimSpace(ifErrBlock), "\n") {
			if i != 0 {
				e.InsertLineBelow()
				e.pos.sy++
			}
			e.SetCurrentLine(currentLeadingWhitespace + line)
		}
		e.End(c)
	} else if (e.mode == mode.XML || e.mode == mode.HTML) && e.expandTags && trimmedLine != "" && !strings.Contains(trimmedLine, "<") && !strings.Contains(trimmedLine, ">") && strings.ToLower(string(trimmedLine[0])) == string(trimmedLine[0]) {
		// Words one a line without < or >? Expand into <tag asdf> above and </tag> below.
		words := strings.Fields(trimmedLine)
		tagName := words[0] // must be at least one word
		// the second word after the tag name needs to be ie. x=42 or href=...,
		// and the tag name must only contain letters a-z A-Z
		if (len(words) == 1 || strings.Contains(words[1], "=")) && onlyAZaz(tagName) {
			above := "<" + trimmedLine + ">"
			if tagName == "img" && !strings.Contains(trimmedLine, "alt=") && strings.Contains(trimmedLine, "src=") {
				// Pick out the image URI from the "src=" declaration
				imageURI := ""
				for _, word := range strings.Fields(trimmedLine) {
					if strings.HasPrefix(word, "src=") {
						imageURI = strings.SplitN(word, "=", 2)[1]
						imageURI = strings.TrimPrefix(imageURI, "\"")
						imageURI = strings.TrimSuffix(imageURI, "\"")
						imageURI = strings.TrimPrefix(imageURI, "'")
						imageURI = strings.TrimSuffix(imageURI, "'")
						break
					}
				}
				// If we got something that looks like and image URI, use the description before "." and capitalize it,
				// then use that as the default "alt=" declaration.
				if strings.Contains(imageURI, ".") {
					imageName := capitalizeWords(strings.TrimSuffix(imageURI, filepath.Ext(imageURI)))
					above = "<" + trimmedLine + " alt=\"" + imageName + "\">"
				}
			}
			// Now replace the current line
			e.SetCurrentLine(currentLeadingWhitespace + above)
			e.End(c)
			// And insert a line below
			e.InsertLineBelow()
			// Then if it's not an img tag, insert the closing tag below the current line
			if tagName != "img" {
				e.pos.mut.Lock()
				e.pos.sy++
				e.pos.mut.Unlock()
				below := "</" + tagName + ">"
				e.SetCurrentLine(currentLeadingWhitespace + below)
				e.pos.mut.Lock()
				e.pos.sy--
				e.pos.sx += 2
				e.pos.mut.Unlock()
				indent = true
				leadingWhitespace = e.indentation.String() + currentLeadingWhitespace
			}
		}
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
