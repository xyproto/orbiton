package main

import "fmt"

// QuoteState keeps track of if we're within a multiline comment, single quotes, double quotes or multiline quotes.
// Single line comments are not kept track of in the same way, they can be detected just by checking the current line.
// If one of the ints are > 0, the other ints should not be added to.
// Multiline comments (/* ... */) are special.
// This could be a flag int instead
type QuoteState struct {
	singleQuote             int
	doubleQuote             int
	backtick                int
	multiLineComment        bool
	singleLineComment       bool
	singleLineCommentMarker string
	startedMultiLineString  bool
}

// NewQuoteState takes a singleLineCommentMarker (such as "//" or "#") and returns a pointer to a new QuoteState struct
func NewQuoteState(singleLineCommentMarker string) *QuoteState {
	var q QuoteState
	q.singleLineCommentMarker = singleLineCommentMarker
	return &q
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

// OnlyMultiLineComment returns true if we're only within a multiline comment
func (q *QuoteState) OnlyMultiLineComment() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick == 0 && q.multiLineComment && !q.singleLineComment
}

// String returns info about the current quote state
func (q *QuoteState) String() string {
	return fmt.Sprintf("singleQuote=%v doubleQuote=%v backtick=%v multiLineComment=%v singleLineComment=%v startedMultiLineString=%v\n", q.singleQuote, q.doubleQuote, q.backtick, q.multiLineComment, q.singleLineComment, q.startedMultiLineString)
}

// Process takes a line of text and modifies the current quote state accordingly,
// depending on which runes are encountered.
func (q *QuoteState) Process(line string) {
	var prevRune rune
	q.singleLineComment = false
	q.startedMultiLineString = false
	for _, r := range line {
		switch r {
		case '`':
			if q.None() {
				q.backtick++
				q.startedMultiLineString = true
			} else {
				q.backtick--
			}
		case '"':
			if q.None() {
				q.doubleQuote++
			} else {
				q.doubleQuote--
			}
		case '\'':
			if q.None() {
				q.singleQuote++
			} else {
				q.singleQuote--
			}
		case '*':
			if prevRune == '/' && q.None() {
				q.multiLineComment = true
			}
		case []rune(q.singleLineCommentMarker)[0]:
			if len(q.singleLineCommentMarker) > 1 && prevRune == []rune(q.singleLineCommentMarker)[1] && q.None() {
				q.singleLineComment = true
				// We're in a single line comment, nothing more to do for this line
				return
			}
			if r != '/' {
				break
			}
			// r == '/'
			fallthrough
		case '/':
			if prevRune == '*' && !q.None() {
				q.multiLineComment = false
			}
		}
		prevRune = r
	}
}
