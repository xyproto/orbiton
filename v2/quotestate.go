package main

import (
	"errors"
	"strings"

	"github.com/xyproto/mode"
)

// QuoteState keeps track of if we're within a multi-line comment, single quotes, double quotes or multi-line quotes.
// Single line comments are not kept track of in the same way, they can be detected just by checking the current line.
// If one of the ints are > 0, the other ints should not be added to.
// MultiLine comments (/* ... */) are special.
// This could be a flag int instead
type QuoteState struct {
	singleLineCommentMarker            string
	singleLineCommentMarkerRunes       []rune
	doubleQuote                        int
	backtick                           int
	mode                               mode.Mode
	braCount                           int // square bracket count
	parCount                           int // parenthesis count
	singleQuote                        int
	firstRuneInSingleLineCommentMarker rune
	lastRuneInSingleLineCommentMarker  rune
	startedMultiLineComment            bool
	stoppedMultiLineComment            bool
	containsMultiLineComments          bool
	startedMultiLineString             bool
	hasSingleLineComment               bool
	multiLineComment                   bool
	ignoreSingleQuotes                 bool
}

// NewQuoteState takes a singleLineCommentMarker (such as "//" or "#") and returns a pointer to a new QuoteState struct
func NewQuoteState(singleLineCommentMarker string, m mode.Mode, ignoreSingleQuotes bool) (*QuoteState, error) {
	var q QuoteState
	q.singleLineCommentMarker = singleLineCommentMarker
	q.singleLineCommentMarkerRunes = []rune(singleLineCommentMarker)
	lensr := len(q.singleLineCommentMarkerRunes)
	if lensr == 0 {
		return nil, errors.New("single line comment marker is empty")
	}
	q.firstRuneInSingleLineCommentMarker = q.singleLineCommentMarkerRunes[0]
	q.lastRuneInSingleLineCommentMarker = q.singleLineCommentMarkerRunes[lensr-1]
	q.mode = m
	q.ignoreSingleQuotes = ignoreSingleQuotes
	return &q, nil
}

// None returns true if we're not within ', "", `, /* ... */ or a single-line quote right now
func (q *QuoteState) None() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick == 0 && !q.multiLineComment && !q.hasSingleLineComment
}

// ProcessRune is for processing single runes
func (q *QuoteState) ProcessRune(r, prevRune, prevPrevRune rune) {
	switch r {
	case '`':
		if q.None() {
			q.backtick++
			q.startedMultiLineString = true
		} else {
			q.backtick--
			if q.backtick < 0 {
				q.backtick = 0
			}
		}
	case '"':
		if prevPrevRune == '"' && prevRune == '"' {
			q.startedMultiLineString = q.None()
		} else if prevRune != '\\' {
			if q.None() {
				q.doubleQuote++
			} else {
				q.doubleQuote--
				if q.doubleQuote < 0 {
					q.doubleQuote = 0
				}
			}
		}
	case '\'':
		if prevRune != '\\' {
			if q.ignoreSingleQuotes || q.mode == mode.Lisp || q.mode == mode.Clojure {
				return
			}
			if q.None() {
				q.singleQuote++
			} else {
				q.singleQuote--
				if q.singleQuote < 0 {
					q.singleQuote = 0
				}
			}
		}
	case '*': // support multi-line comments
		if q.mode != mode.Shell && q.mode != mode.Make && q.mode != mode.Just && q.firstRuneInSingleLineCommentMarker != '#' && prevRune == '/' && (prevPrevRune == '\n' || prevPrevRune == ' ' || prevPrevRune == '\t') && q.None() {
			// C-style
			q.multiLineComment = true
			q.startedMultiLineComment = true
		} else if (q.mode == mode.StandardML || q.mode == mode.OCaml || q.mode == mode.Haskell) && prevRune == '(' && q.None() {
			q.parCount-- // Not a parenthesis start after all, but the start of a multi-line comment
			q.multiLineComment = true
			q.startedMultiLineComment = true
		} else if (q.mode == mode.Elm || q.mode == mode.Haskell) && prevRune == '{' && q.None() {
			q.parCount-- // Not a parenthesis start after all, but the start of a multi-line comment
			q.multiLineComment = true
			q.startedMultiLineComment = true
		}
	case '{':
		if q.mode == mode.ObjectPascal && q.None() {
			q.multiLineComment = true
			q.startedMultiLineComment = true
		}
	case '-': // support for HTML-style and XML-style multi-line comments
		if q.mode != mode.Shell && q.mode != mode.Make && q.mode != mode.Just && prevRune == '!' && prevPrevRune == '<' && q.None() {
			q.multiLineComment = true
			q.startedMultiLineComment = true
		} else if (q.mode == mode.Elm || q.mode == mode.Haskell) && prevRune == '{' {
			q.multiLineComment = true
			q.startedMultiLineComment = true
		} else if q.mode == mode.Diff && prevRune == '-' && prevPrevRune == '-' {
			// Reset all comment state if we encounter '--' in a diff / patch file
			q.hasSingleLineComment = false
			q.startedMultiLineString = false
			q.stoppedMultiLineComment = false
			q.containsMultiLineComments = false
			q.multiLineComment = false
			q.hasSingleLineComment = false
		}
	case q.lastRuneInSingleLineCommentMarker:
		// TODO: Simplify by checking q.None() first, and assuming that the len of the marker is > 1 if it's not 1 since it's not 0
		if !q.multiLineComment && !q.hasSingleLineComment && !q.startedMultiLineString && prevPrevRune != ':' && q.doubleQuote == 0 && q.singleQuote == 0 && q.backtick == 0 {
			switch {
			case len(q.singleLineCommentMarkerRunes) == 1:
				fallthrough
			case q.mode != mode.Shell && q.mode != mode.Make && q.mode != mode.Just && len(q.singleLineCommentMarkerRunes) > 1 && prevRune == q.firstRuneInSingleLineCommentMarker:
				q.hasSingleLineComment = true
				q.startedMultiLineString = false
				q.stoppedMultiLineComment = false
				q.multiLineComment = false
				q.backtick = 0
				q.doubleQuote = 0
				q.singleQuote = 0
				// We're in a single line comment, nothing more to do for this line
				return
			}
		}
		if r != '/' {
			break
		}
		// r == '/'
		fallthrough
	case '/': // support C-style multi-line comments
		if q.mode != mode.Shell && q.mode != mode.Make && q.mode != mode.Just && q.firstRuneInSingleLineCommentMarker != '#' && prevRune == '*' {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		}
	case '(':
		if q.None() {
			q.parCount++
		}
	case ';':
		if q.mode == mode.Clojure && prevRune == ';' {
			q.hasSingleLineComment = true
		}
	case ')':
		if (q.mode == mode.StandardML || q.mode == mode.OCaml || q.mode == mode.Haskell) && prevRune == '*' {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		} else if q.None() {
			q.parCount--
		}
	case '}':
		if (q.mode == mode.Elm || q.mode == mode.Haskell) && prevRune == '-' {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		} else if q.mode == mode.ObjectPascal {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		}
	case '[':
		if q.None() {
			q.braCount++
		}
	case ']':
		if q.None() {
			q.braCount--
		}
	case '>': // support HTML-style and XML-style multi-line comments
		if prevRune == '-' && (q.mode == mode.HTML || q.mode == mode.XML) {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		}
	}
}

// Process takes a line of text and modifies the current quote state accordingly,
// depending on which runes are encountered.
func (q *QuoteState) Process(line string) (rune, rune) {
	q.hasSingleLineComment = false
	q.startedMultiLineString = false
	q.stoppedMultiLineComment = false
	q.containsMultiLineComments = false
	prevRune := '\n'
	prevPrevRune := '\n'
	for _, r := range line {
		q.ProcessRune(r, prevRune, prevPrevRune)
		prevPrevRune = prevRune
		prevRune = r
	}
	return prevRune, prevPrevRune
}

// ParBraCount will count the parenthesis and square brackets for a single line
// while skipping comments and multi-line strings
// and without modifying the QuoteState.
func (q *QuoteState) ParBraCount(line string) (int, int) {
	qCopy := *q
	qCopy.parCount = 0
	qCopy.braCount = 0
	qCopy.Process(line)
	return qCopy.parCount, qCopy.braCount
}

// checkMultiLineString detects and updates the inCodeBlock state.
// For languages like Nim, Mojo, Python and Starlark.
func checkMultiLineString(trimmedLine string, inCodeBlock bool) (bool, bool) {
	trimmedLine = strings.TrimPrefix(trimmedLine, "return ")
	foundDocstringMarker := false
	// Check for special syntax patterns that indicate the start of a multiline string
	if trimmedLine == "\"\"\"" || trimmedLine == "'''" { // only 3 letters
		inCodeBlock = !inCodeBlock
		foundDocstringMarker = true
	} else if strings.HasSuffix(trimmedLine, " = \"\"\"") || strings.HasSuffix(trimmedLine, " = '''") {
		inCodeBlock = true
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "\"\"\"") && strings.HasSuffix(trimmedLine, "\"\"\"") { // this could be 6 letters
		inCodeBlock = false
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "'''") && strings.HasSuffix(trimmedLine, "'''") { // this could be 6 letters
		inCodeBlock = false
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") { // this is more than 3 ts
		inCodeBlock = !inCodeBlock
		if inCodeBlock {
			foundDocstringMarker = true
		}
	} else if strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''") { // this is more than 3 ts
		if strings.Count(trimmedLine, "\"\"\"")%2 != 0 || strings.Count(trimmedLine, "'''")%2 != 0 {
			inCodeBlock = !inCodeBlock
		}
		if inCodeBlock {
			foundDocstringMarker = true
		}
	}
	return inCodeBlock, foundDocstringMarker
}

// checkMultiLineString2 detects and updates the inCodeBlock state.
// For languages like Nim, Mojo, Python and Starlark.
func checkMultiLineString2(trimmedLine string, inCodeBlock bool) (bool, bool) {
	foundDocstringMarker := false
	if trimmedLine == "return \"\"\"" || trimmedLine == "return '''" {
		inCodeBlock = true
		foundDocstringMarker = false
	} else if trimmedLine == "\"\"\"" || trimmedLine == "'''" { // only 3 letters
		inCodeBlock = !inCodeBlock
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "\"\"\"") && strings.HasSuffix(trimmedLine, "\"\"\"") {
		inCodeBlock = false
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "'''") && strings.HasSuffix(trimmedLine, "'''") {
		inCodeBlock = false
		foundDocstringMarker = true
	} else if strings.HasPrefix(trimmedLine, "\"\"\"") || strings.HasPrefix(trimmedLine, "'''") {
		inCodeBlock = !inCodeBlock
		foundDocstringMarker = true
	} else if strings.HasSuffix(trimmedLine, "\"\"\"") || strings.HasSuffix(trimmedLine, "'''") {
		inCodeBlock = !inCodeBlock
		foundDocstringMarker = true
	}
	return inCodeBlock, foundDocstringMarker
}
