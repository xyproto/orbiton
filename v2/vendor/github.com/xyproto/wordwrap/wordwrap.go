package wordwrap

import (
	"errors"
	"strings"
)

// noLineStart returns true for punctuation that must not begin a wrapped line.
// These characters in good typography always follow the preceding word.
func noLineStart(r rune) bool {
	switch r {
	case '.', ',', '!', '?', ':', ';', ')', ']', '}', '\'', '"',
		'\u2019', '\u201D', '\u00BB', '\u2014', '\u2013':
		// ' " » — –
		return true
	}
	return false
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
		runes := []rune(line)
		for len(runes) > maxWidth {
			// Find the best split position: scan left from maxWidth for a space
			// that does not leave punctuation at the start of the next segment.
			splitPos := -1
			for i := maxWidth; i > 0; i-- {
				if runes[i-1] != ' ' {
					continue
				}
				// The prospective new line would start at runes[i].
				// Skip over any leading spaces to find the first real character.
				next := i
				for next < len(runes) && runes[next] == ' ' {
					next++
				}
				if next < len(runes) && noLineStart(runes[next]) {
					continue // bad split
				}
				splitPos = i - 1 // index of the space
				break
			}
			if splitPos == -1 {
				// No valid space found; hard-break at maxWidth.
				wrappedLines = append(wrappedLines, string(runes[:maxWidth]))
				runes = runes[maxWidth:]
			} else {
				wrappedLines = append(wrappedLines, string(runes[:splitPos]))
				// Skip the space itself and any following spaces.
				runes = runes[splitPos+1:]
				for len(runes) > 0 && runes[0] == ' ' {
					runes = runes[1:]
				}
			}
		}
		wrappedLines = append(wrappedLines, string(runes))
	}

	return wrappedLines, nil
}
