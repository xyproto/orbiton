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

// QuoteStateChange represents changes to a QuoteState struct.
// This is used for caching.
type QuoteStateChange struct {
	DeltaBraCount                    int
	DeltaParCount                    int
	DeltaSingleQuote                 int
	ChangedStartedMultiLineComment   bool
	ChangedStoppedMultiLineComment   bool
	ChangedContainsMultiLineComments bool
	ChangedStartedMultiLineString    bool
	ChangedHasSingleLineComment      bool
	ChangedMultiLineComment          bool
}

// ProcessCache can be used for caching the quote state changes, per line of text
type ProcessCache map[string]QuoteStateChange

// Apply can apply this QuoteStateChange to a QuoteState
func (qsc *QuoteStateChange) Apply(qs *QuoteState) {
	qs.braCount += qsc.DeltaBraCount
	qs.parCount += qsc.DeltaParCount
	qs.singleQuote += qsc.DeltaSingleQuote

	if qsc.ChangedStartedMultiLineComment {
		qs.startedMultiLineComment = !qs.startedMultiLineComment
	}
	if qsc.ChangedStoppedMultiLineComment {
		qs.stoppedMultiLineComment = !qs.stoppedMultiLineComment
	}
	if qsc.ChangedContainsMultiLineComments {
		qs.containsMultiLineComments = !qs.containsMultiLineComments
	}
	if qsc.ChangedStartedMultiLineString {
		qs.startedMultiLineString = !qs.startedMultiLineString
	}
	if qsc.ChangedHasSingleLineComment {
		qs.hasSingleLineComment = !qs.hasSingleLineComment
	}
	if qsc.ChangedMultiLineComment {
		qs.multiLineComment = !qs.multiLineComment
	}
}

var qcache = make(ProcessCache)

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

// OnlyBacktick returns true if we're only within a ` quote
func (q *QuoteState) OnlyBacktick() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick > 0 && !q.multiLineComment && !q.hasSingleLineComment
}

// OnlySingleQuote returns true if we're only within a ' quote
func (q *QuoteState) OnlySingleQuote() bool {
	return q.singleQuote > 0 && q.doubleQuote == 0 && q.backtick == 0 && !q.multiLineComment && !q.hasSingleLineComment
}

// OnlyDoubleQuote returns true if we're only within a " quote
func (q *QuoteState) OnlyDoubleQuote() bool {
	return q.singleQuote == 0 && q.doubleQuote > 0 && q.backtick == 0 && !q.multiLineComment && !q.hasSingleLineComment
}

// OnlyMultiLineComment returns true if we're only within a multi-line comment
func (q *QuoteState) OnlyMultiLineComment() bool {
	return q.singleQuote == 0 && q.doubleQuote == 0 && q.backtick == 0 && q.multiLineComment && !q.hasSingleLineComment
}

// String returns info about the current quote state
func (q *QuoteState) String() string {
	return fmt.Sprintf("singleQuote=%v doubleQuote=%v backtick=%v multiLineComment=%v singleLineComment=%v startedMultiLineString=%v\n", q.singleQuote, q.doubleQuote, q.backtick, q.multiLineComment, q.hasSingleLineComment, q.startedMultiLineString)
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

// extractLastRunes extracts the next-to-last and last runes from the string
func extractLastRunes(s string) (rune, rune) {
	if len(s) < 2 {
		return '\n', '\n' // Fallback runes if the string is too short
	}

	runes := []rune(s)
	return runes[len(runes)-2], runes[len(runes)-1]
}

// Process function with caching
func (q *QuoteState) Process(line string) (rune, rune) {
	cacheKey := line

	if change, found := qcache[cacheKey]; found {
		change.Apply(q)
		return extractLastRunes(line)
	}

	originalState := *q

	prevRune := '\n'
	prevPrevRune := '\n'
	for _, r := range line {
		q.ProcessRune(r, prevRune, prevPrevRune)
		prevPrevRune = prevRune
		prevRune = r
	}

	change := calculateStateChange(originalState, *q)
	qcache[cacheKey] = change

	return prevRune, prevPrevRune
}

// calculateStateChange function
func calculateStateChange(original, new QuoteState) QuoteStateChange {
	return QuoteStateChange{
		DeltaBraCount:                    new.braCount - original.braCount,
		DeltaParCount:                    new.parCount - original.parCount,
		DeltaSingleQuote:                 new.singleQuote - original.singleQuote,
		ChangedStartedMultiLineComment:   new.startedMultiLineComment != original.startedMultiLineComment,
		ChangedStoppedMultiLineComment:   new.stoppedMultiLineComment != original.stoppedMultiLineComment,
		ChangedContainsMultiLineComments: new.containsMultiLineComments != original.containsMultiLineComments,
		ChangedStartedMultiLineString:    new.startedMultiLineString != original.startedMultiLineString,
		ChangedHasSingleLineComment:      new.hasSingleLineComment != original.hasSingleLineComment,
		ChangedMultiLineComment:          new.multiLineComment != original.multiLineComment,
	}
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
