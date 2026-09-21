package main

import (
	"strings"

	"github.com/xyproto/mode"
)

// smartIndentation takes the leading whitespace for a line, and the trimmed contents of a line
// it tries to indent or dedent in a smart way, intended for use on the following line,
// and returns a new string of leading whitespace.
func (e *Editor) smartIndentation(leadingWhitespace, trimmedLine string, alsoDedent bool) string {
	// Grab the whitespace for this new line
	// "smart indentation", add one indentation from the line above
	if (endsWithOpeningBracket(trimmedLine) || strings.HasSuffix(trimmedLine, ":")) && !strings.HasPrefix(trimmedLine, e.SingleLineCommentMarker()) {
		leadingWhitespace += e.indentation.String()
	}
	if alsoDedent && endsWithClosingBracket(trimmedLine) {
		// "smart dedentation", subtract one indentation from the line above
		indentation := e.indentation.String()
		if len(leadingWhitespace) > len(indentation) {
			leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-len(indentation)]
		}
	}
	return leadingWhitespace
}

// bracketPair returns the opening and closing bracket for the given rune,
// if it is a parenthesis, curly bracket or square bracket
func bracketPair(r rune) (opening, closing rune, ok bool) {
	switch r {
	case '(', ')':
		return '(', ')', true
	case '{', '}':
		return '{', '}', true
	case '[', ']':
		return '[', ']', true
	}
	return 0, 0, false
}

// endsWithOpeningBracket checks if the string ends with "(", "{" or "["
func endsWithOpeningBracket(s string) bool {
	if s == "" {
		return false
	}
	r := rune(s[len(s)-1])
	opening, _, ok := bracketPair(r)
	return ok && r == opening
}

// endsWithClosingBracket checks if the string ends with ")", "}" or "]"
func endsWithClosingBracket(s string) bool {
	if s == "" {
		return false
	}
	r := rune(s[len(s)-1])
	_, closing, ok := bracketPair(r)
	return ok && r == closing
}

// startsWithClosingBracket checks if the string starts with ")", "}" or "]"
func startsWithClosingBracket(s string) bool {
	if s == "" {
		return false
	}
	r := rune(s[0])
	_, closing, ok := bracketPair(r)
	return ok && r == closing
}

// splitIndentation returns the leading whitespace for the right half of a line that was split in two.
// If the right half starts with a closing bracket, it keeps the indentation of the line when the
// bracket was opened at the end of the left half, and is dedented one level if not.
// Otherwise, the indentation follows the left half.
func (e *Editor) splitIndentation(leadingWhitespace, left, right string) string {
	if !startsWithClosingBracket(right) {
		return e.smartIndentation(leadingWhitespace, left, false)
	}
	if !endsWithOpeningBracket(left) {
		if indentation := e.indentation.String(); len(leadingWhitespace) >= len(indentation) {
			leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-len(indentation)]
		}
	}
	return leadingWhitespace
}

// isCaseLine checks if the trimmed line is a "case" or "default" branch of a switch statement
func isCaseLine(trimmedLine string) bool {
	return strings.HasPrefix(trimmedLine, "case ") || strings.HasPrefix(trimmedLine, "default:") || strings.HasPrefix(trimmedLine, "default ->")
}

// The maximum number of lines to scan upward when looking for the enclosing block
const maxBlockScanLines = 10000

// enclosingBlock scans the lines above the current line and returns the line index of the innermost
// unclosed opening bracket, together with the leading whitespace of the nearest "case" line at that
// level (if any). Brackets within strings and comments are ignored.
func (e *Editor) enclosingBlock(opening, closing rune) (openingLine LineIndex, caseWhitespace string, hasCase, found bool) {
	q, err := e.NewQuoteState()
	if err != nil {
		return 0, "", false, false
	}
	type block struct {
		caseWhitespace string
		line           LineIndex
		hasCase        bool
	}
	var stack []block
	y := e.DataY()
	for i := max(y-maxBlockScanLines, 0); i < y; i++ {
		line := e.Line(i)
		q.startLine()
		if len(stack) > 0 && q.None() && isCaseLine(strings.TrimSpace(line)) {
			top := &stack[len(stack)-1]
			top.caseWhitespace = getLeadingWhitespace(line)
			top.hasCase = true
		}
		q.ForEachCodeRune(line, func(r rune) bool {
			switch r {
			case opening:
				stack = append(stack, block{line: i})
			case closing:
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
			}
			return true
		})
	}
	if len(stack) == 0 {
		return 0, "", false, false
	}
	top := stack[len(stack)-1]
	return top.line, top.caseWhitespace, top.hasCase, true
}

// caseIndentation returns the leading whitespace that a "case" on the current line should have,
// by looking for a sibling "case" in the same switch block. For Go, where gofmt aligns "case" with
// "switch" and "select", the indentation of the "switch" or "select" line is used if there is no sibling.
func (e *Editor) caseIndentation() (string, bool) {
	openingLine, caseWhitespace, hasCase, found := e.enclosingBlock('{', '}')
	if !found {
		return "", false
	}
	if hasCase {
		return caseWhitespace, true
	}
	if line := e.Line(openingLine); e.mode == mode.Go && (strings.Contains(line, "switch") || strings.Contains(line, "select")) {
		return getLeadingWhitespace(line), true
	}
	return "", false
}

// alignCase re-indents the current "case" line to match the sibling cases in the same switch block.
// Returns true if the indentation of the current line was changed.
func (e *Editor) alignCase() bool {
	leadingWhitespace, found := e.caseIndentation()
	if !found || leadingWhitespace == e.LeadingWhitespace() {
		return false
	}
	e.SetCurrentLine(leadingWhitespace + e.TrimmedLine())
	return true
}

// closingBracketIndentation returns the leading whitespace of the line above the current line that has
// the unmatched opening bracket that the given closing bracket would close
func (e *Editor) closingBracketIndentation(closing rune) (string, bool) {
	opening, _, ok := bracketPair(closing)
	if !ok {
		return "", false
	}
	openingLine, _, _, found := e.enclosingBlock(opening, closing)
	if !found {
		return "", false
	}
	return getLeadingWhitespace(e.Line(openingLine)), true
}
