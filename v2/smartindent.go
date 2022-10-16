package main

import "strings"

// smartIndentation takes the leading whitespace for a line, and the trimmed contents of a line
// it tries to indent or dedent in a smart way, intended for use on the following line,
// and returns a new string of leading whitespace.
func (e *Editor) smartIndentation(leadingWhitespace, trimmedLine string, alsoDedent bool) string {
	// Grab the whitespace for this new line
	// "smart indentation", add one indentation from the line above
	if len(trimmedLine) > 0 &&
		(strings.HasSuffix(trimmedLine, "(") || strings.HasSuffix(trimmedLine, "{") || strings.HasSuffix(trimmedLine, "[") ||
			strings.HasSuffix(trimmedLine, ":")) && !strings.HasPrefix(trimmedLine, e.SingleLineCommentMarker()) {
		leadingWhitespace += e.indentation.String()
	}
	if alsoDedent {
		// "smart dedentation", subtract one indentation from the line above
		if len(trimmedLine) > 0 &&
			(strings.HasSuffix(trimmedLine, ")") || strings.HasSuffix(trimmedLine, "}") || strings.HasSuffix(trimmedLine, "]")) {
			indentation := e.indentation.String()
			if len(leadingWhitespace) > len(indentation) {
				leadingWhitespace = leadingWhitespace[:len(leadingWhitespace)-len(indentation)]
			}
		}
	}
	return leadingWhitespace
}
