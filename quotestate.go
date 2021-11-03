package main

import (
	"errors"
	"fmt"

	"github.com/xyproto/mode"
)

// QuoteState keeps track of if we're within a multi-line comment, single quotes, double quotes or multi-line quotes.
// Single line comments are not kept track of in the same way, they can be detected just by checking the current line.
// If one of the ints are > 0, the other ints should not be added to.
// MultiLine comments (/* ... */) are special.
// This could be a flag int instead
type QuoteState struct {
	singleQuote                        int
	doubleQuote                        int
	backtick                           int
	multiLineComment                   bool
	singleLineComment                  bool
	singleLineCommentMarker            string
	singleLineCommentMarkerRunes       []rune
	firstRuneInSingleLineCommentMarker rune
	lastRuneInSingleLineCommentMarker  rune
	startedMultiLineString             bool
	startedMultiLineComment            bool
	stoppedMultiLineComment            bool
	containsMultiLineComments          bool
	parCount                           int // Parenthesis count
	braCount                           int // Square bracket count
	mode                               mode.Mode
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
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick == 0 && !q.multiLineComment && !q.singleLineComment
}

// OnlyBacktick returns true if we're only within a ` quote
func (q *QuoteState) OnlyBacktick() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick > 0 && !q.multiLineComment && !q.singleLineComment
}

// OnlySingleQuote returns true if we're only within a ' quote
func (q *QuoteState) OnlySingleQuote() bool {
	return q.singleQuote > 0 && q.doubleQuote == 0 && q.backtick == 0 && !q.multiLineComment && !q.singleLineComment
}

// OnlyDoubleQuote returns true if we're only within a " quote
func (q *QuoteState) OnlyDoubleQuote() bool {
	return q.singleQuote == 0 && q.doubleQuote > 0 && q.backtick == 0 && !q.multiLineComment && !q.singleLineComment
}

// OnlyMultiLineComment returns true if we're only within a multi-line comment
func (q *QuoteState) OnlyMultiLineComment() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick == 0 && q.multiLineComment && !q.singleLineComment
}

// String returns info about the current quote state
func (q *QuoteState) String() string {
	return fmt.Sprintf("singleQuote=%v doubleQuote=%v backtick=%v multiLineComment=%v singleLineComment=%v startedMultiLineString=%v\n", q.singleQuote, q.doubleQuote, q.backtick, q.multiLineComment, q.singleLineComment, q.startedMultiLineString)
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
	case '*': // support C-style, StandardML-style and multi-line comments
		if q.firstRuneInSingleLineCommentMarker != '#' && (prevRune == '/' || prevRune == '(') && (prevPrevRune == '\n' || prevPrevRune == ' ' || prevPrevRune == '\t') && q.None() {
			q.multiLineComment = true
			q.startedMultiLineComment = true
		}
	case '-': // support for HTML-style and XML-style multi-line comments
		if prevRune == '!' && prevPrevRune == '<' && q.None() {
			q.multiLineComment = true
			q.startedMultiLineComment = true
		}
	case q.lastRuneInSingleLineCommentMarker:
		// TODO: Simplify by checking q.None() first, and assuming that the len of the marker is > 1 if it's not 1 since it's not 0
		if !q.multiLineComment && !q.singleLineComment && !q.startedMultiLineString && prevPrevRune != ':' && q.doubleQuote == 0 && q.singleQuote == 0 && q.backtick == 0 {
			switch {
			case len(q.singleLineCommentMarkerRunes) == 1:
				fallthrough
			case len(q.singleLineCommentMarkerRunes) > 1 && prevRune == q.firstRuneInSingleLineCommentMarker:
				q.singleLineComment = true
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
		if q.firstRuneInSingleLineCommentMarker != '#' && prevRune == '*' {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}

		}
	case '(':
		if q.mode == mode.StandardML && prevRune == '*' {
			q.stoppedMultiLineComment = true
			q.multiLineComment = false
			if q.startedMultiLineComment {
				q.containsMultiLineComments = true
			}
		} else if q.None() {
			q.parCount++
		}
	case ')':
		if q.None() {
			q.parCount--
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
	q.singleLineComment = false
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
// while skipping comments and multiline strings
// and without modifying the QuoteState.
func (q *QuoteState) ParBraCount(line string) (int, int) {
	qCopy := *q
	qCopy.parCount = 0
	qCopy.braCount = 0
	qCopy.Process(line)
	return qCopy.parCount, qCopy.braCount
}
