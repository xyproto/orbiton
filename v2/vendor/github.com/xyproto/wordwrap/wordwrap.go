package wordwrap

import (
	"errors"
	"strings"
	"unicode"
)

// NoLineStart returns true for punctuation that must not begin a wrapped line.
// These characters in good typography always follow the preceding word.
func NoLineStart(r rune) bool {
	switch r {
	case '.', ',', '!', '?', ':', ';', ')', ']', '}', '\'', '"',
		'\u2019', '\u201D', '\u00BB', '\u2014', '\u2013':
		// ' " » — –
		return true
	}
	return false
}

// WrapResult holds the result of a line wrap operation.
type WrapResult struct {
	// Left is the text that stays on the current line (the break space excluded).
	Left string
	// Right is the text that moves to the next line (leading spaces removed).
	Right string
	// BreakAt is the rune index of the space where the line was broken.
	// -1 if no wrapping was performed.
	BreakAt int
	// RightStart is the rune index in the original line where Right begins
	// (past the break space and any consecutive whitespace).
	RightStart int
	// Wrapped is true if the line was successfully wrapped.
	Wrapped bool
}

// WrapLine finds the best position to break a single line that exceeds limit.
//
// It searches backwards from the limit position for a whitespace character
// that produces a typographically valid break — one where the resulting new
// line does not begin with punctuation that must follow the preceding word
// (. , ! ? : ; ) ] } ' " etc.).
//
// maxBacktrack limits how far back from limit to search for a break point.
// When maxBacktrack is 0 the search extends all the way to position 1.
//
// If the line does not exceed limit, or no suitable break point is found,
// the result has Wrapped=false, Left equals the full line, and BreakAt is -1.
func WrapLine(line string, limit, maxBacktrack int) WrapResult {
	runes := []rune(line)
	if len(runes) <= limit || limit <= 0 {
		return WrapResult{Left: line, BreakAt: -1}
	}

	minPos := 1 // don't break at position 0 (would produce empty left part)
	if maxBacktrack > 0 && limit-maxBacktrack > minPos {
		minPos = limit - maxBacktrack
	}

	breakAt := -1
	for i := limit; i >= minPos; i-- {
		if i >= len(runes) {
			continue
		}
		if !unicode.IsSpace(runes[i]) {
			continue
		}
		// Check that the next non-space char doesn't violate typography rules
		next := i + 1
		for next < len(runes) && unicode.IsSpace(runes[next]) {
			next++
		}
		if next < len(runes) && NoLineStart(runes[next]) {
			continue // bad split: punctuation would start the new line
		}
		breakAt = i
		break
	}

	if breakAt < 0 {
		return WrapResult{Left: line, BreakAt: -1}
	}

	// Skip any run of whitespace after the break point
	rightStart := breakAt + 1
	for rightStart < len(runes) && unicode.IsSpace(runes[rightStart]) {
		rightStart++
	}

	// Trim trailing whitespace from the left part
	leftEnd := breakAt
	for leftEnd > 0 && unicode.IsSpace(runes[leftEnd-1]) {
		leftEnd--
	}

	return WrapResult{
		Left:       string(runes[:leftEnd]),
		Right:      string(runes[rightStart:]),
		BreakAt:    breakAt,
		RightStart: rightStart,
		Wrapped:    true,
	}
}

// WordWrap wraps the input text to the specified maxWidth (in runes).
// It returns a slice of strings, each of which is a line of the wrapped text,
// or an error if maxWidth is not valid.
//
// The wrapping respects typographic conventions: no line will begin with
// punctuation that must follow the preceding word (. , ! ? : ; ) ] } etc.).
func WordWrap(text string, maxWidth int) ([]string, error) {
	if maxWidth <= 0 {
		return nil, errors.New("maxWidth must be greater than 0")
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	for _, line := range lines {
		for len([]rune(line)) > maxWidth {
			result := WrapLine(line, maxWidth, 0)
			if !result.Wrapped {
				// No suitable break point; hard-break at maxWidth.
				runes := []rune(line)
				wrappedLines = append(wrappedLines, string(runes[:maxWidth]))
				line = string(runes[maxWidth:])
				continue
			}
			wrappedLines = append(wrappedLines, result.Left)
			line = result.Right
		}
		wrappedLines = append(wrappedLines, line)
	}

	return wrappedLines, nil
}
