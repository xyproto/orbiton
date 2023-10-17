package wordwrap

import (
	"errors"
	"strings"
)

// WordWrap wraps the input text to the specified maxWidth.
// It returns a slice of strings, each of which is a line
// of the wrapped text, or an error if maxWidth is not valid.
func WordWrap(text string, maxWidth int) ([]string, error) {
	if maxWidth <= 0 {
		return nil, errors.New("maxWidth must be greater than 0")
	}

	lines := strings.Split(text, "\n") // Split input text into lines
	var wrappedLines []string

	for _, line := range lines {
		for len(line) > maxWidth {
			splitPos := strings.LastIndex(line[:maxWidth], " ")
			if splitPos == -1 { // no space found to split at
				splitPos = maxWidth // split at max width
			}
			wrappedLines = append(wrappedLines, line[:splitPos])
			line = line[splitPos:]
			line = strings.TrimLeft(line, " ") // trim leading spaces of the new line
		}
		wrappedLines = append(wrappedLines, line)
	}

	return wrappedLines, nil
}
