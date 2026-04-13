package main

import (
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

// isWordRune returns true if r is a "word" character for word-navigation purposes
func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}

// dataXToScreenX converts a rune index (data X) in a line to a screen X position,
// taking tab expansion into account.
func dataXToScreenX(runes []rune, dataX int, spacesPerTab int) int {
	col := 0
	for i, r := range runes {
		if i >= dataX {
			break
		}
		if r == '\t' {
			col += spacesPerTab
		} else {
			col += runewidth.RuneWidth(r)
		}
	}
	return col
}

// GoToNextWord moves the cursor to the start of the next word on the current or next line
func (e *Editor) GoToNextWord(c *vt.Canvas, status *StatusBar) {
	y := e.DataY()
	runes := e.lines[int(y)]
	x, err := e.DataX()
	if err != nil || x >= len(runes) {
		// At end of line: go to first word on the next line
		if !e.Down(c, status) {
			e.Home()
			e.GoToStartOfTextLine(c)
		}
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return
	}

	// Skip over the current word
	for x < len(runes) && isWordRune(runes[x]) {
		x++
	}
	// Skip non-word characters until the next word (or end of line)
	for x < len(runes) && !isWordRune(runes[x]) {
		x++
	}

	if x >= len(runes) {
		// No next word on this line: move to first word on the next line
		if !e.Down(c, status) {
			e.Home()
			e.GoToStartOfTextLine(c)
		}
	} else {
		e.pos.SetX(c, dataXToScreenX(runes, x, e.indentation.PerTab))
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// GoToPrevWord moves the cursor to the start of the previous word on the current or previous line
func (e *Editor) GoToPrevWord(c *vt.Canvas, status *StatusBar) {
	y := e.DataY()
	runes := e.lines[int(y)]
	x, err := e.DataX()

	atLineStart := err != nil || x == 0
	if atLineStart {
		if y == 0 {
			return
		}
		e.Up(c, status)
		e.End(c)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return
	}

	x-- // step back one character from current position
	// Skip non-word characters backwards
	for x > 0 && !isWordRune(runes[x]) {
		x--
	}
	// If we're still on a non-word character, go to the end of the previous line
	if !isWordRune(runes[x]) {
		if y == 0 {
			e.Home()
			e.redrawCursor.Store(true)
			return
		}
		e.Up(c, status)
		e.End(c)
		e.redraw.Store(true)
		e.redrawCursor.Store(true)
		return
	}
	// Walk back to the start of the word
	for x > 0 && isWordRune(runes[x-1]) {
		x--
	}

	e.pos.SetX(c, dataXToScreenX(runes, x, e.indentation.PerTab))
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// isFunctionLine returns true if the line looks like a function/method/class definition
// for the current editor mode.
func (e *Editor) isFunctionLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	switch e.mode {
	case mode.Go:
		return strings.HasPrefix(trimmed, "func ")
	case mode.Python, mode.Mojo:
		return strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "async def ") || strings.HasPrefix(trimmed, "class ")
	case mode.C, mode.Cpp, mode.ObjC:
		// Non-indented lines containing '(' and ending with '{' or ')' are likely function defs
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			return false
		}
		return strings.Contains(trimmed, "(") && (strings.HasSuffix(trimmed, "{") || strings.HasSuffix(trimmed, ")"))
	case mode.CS, mode.Java, mode.Kotlin, mode.Scala:
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			return false
		}
		return strings.Contains(trimmed, "(") && strings.Contains(trimmed, ")")
	case mode.Rust:
		return strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ") || strings.HasPrefix(trimmed, "async fn ") || strings.HasPrefix(trimmed, "pub async fn ")
	case mode.JavaScript, mode.TypeScript:
		return strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "async function ") ||
			(strings.Contains(trimmed, "=>") && strings.HasSuffix(trimmed, "{"))
	case mode.Ruby:
		return strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "module ")
	case mode.Lua:
		return strings.HasPrefix(trimmed, "function ") || strings.HasPrefix(trimmed, "local function ")
	case mode.Zig:
		return strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "pub fn ")
	case mode.Swift:
		return strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "enum ")
	case mode.Hare:
		return strings.Contains(trimmed, "fn ") && !strings.HasPrefix(trimmed, "//")
	default:
		return false
	}
}

// isHeaderLine returns true if the line is a document header (for Markdown, RST, etc.)
func (e *Editor) isHeaderLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	switch e.mode {
	case mode.Markdown:
		return strings.HasPrefix(trimmed, "#")
	case mode.ReStructured, mode.ASCIIDoc:
		// RST uses underline-style headers; the underline is a line of ====, ----, ~~~~, etc.
		// AsciiDoc uses lines starting with = or ==
		if strings.HasPrefix(trimmed, "=") || strings.HasPrefix(trimmed, "#") {
			return true
		}
		// RST underline: line consists entirely of a single repeated special char
		if len(trimmed) >= 2 {
			ch := trimmed[0]
			if ch == '=' || ch == '-' || ch == '~' || ch == '^' || ch == '"' || ch == '\'' || ch == '`' || ch == '#' || ch == '*' || ch == '+' {
				allSame := true
				for i := 1; i < len(trimmed); i++ {
					if trimmed[i] != ch {
						allSame = false
						break
					}
				}
				if allSame {
					return true
				}
			}
		}
	}
	return false
}

// GoToNextFuncOrSection jumps to the next function definition, document header, or paragraph
func (e *Editor) GoToNextFuncOrSection(c *vt.Canvas, status *StatusBar) {
	l := e.Len()
	startY := int(e.DataY()) + 1

	switch e.mode {
	case mode.Markdown, mode.ReStructured, mode.ASCIIDoc:
		for i := startY; i < l; i++ {
			if e.isHeaderLine(e.Line(LineIndex(i))) {
				e.GoTo(LineIndex(i), c, status)
				e.Home()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return
			}
		}
		// No more headers: fall through to paragraph navigation
		e.GoToNextParagraph(c, status)
	default:
		// Try function navigation for source code modes
		for i := startY; i < l; i++ {
			if e.isFunctionLine(e.Line(LineIndex(i))) {
				e.GoTo(LineIndex(i), c, status)
				e.Home()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return
			}
		}
		// No function found or unknown mode: use paragraph navigation
		e.GoToNextParagraph(c, status)
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}

// GoToPrevFuncOrSection jumps to the previous function definition, document header, or paragraph
func (e *Editor) GoToPrevFuncOrSection(c *vt.Canvas, status *StatusBar) {
	startY := int(e.DataY()) - 1

	switch e.mode {
	case mode.Markdown, mode.ReStructured, mode.ASCIIDoc:
		for i := startY; i >= 0; i-- {
			if e.isHeaderLine(e.Line(LineIndex(i))) {
				e.GoTo(LineIndex(i), c, status)
				e.Home()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return
			}
		}
		// No more headers: fall through to paragraph navigation
		e.GoToPrevParagraph(c, status)
	default:
		for i := startY; i >= 0; i-- {
			if e.isFunctionLine(e.Line(LineIndex(i))) {
				e.GoTo(LineIndex(i), c, status)
				e.Home()
				e.redraw.Store(true)
				e.redrawCursor.Store(true)
				return
			}
		}
		// No function found or unknown mode: use paragraph navigation
		e.GoToPrevParagraph(c, status)
	}
	e.redraw.Store(true)
	e.redrawCursor.Store(true)
}
