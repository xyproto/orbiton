package main

import (
	"strconv"
	"strings"
)

// ParsePythonError parses a Python error message and returns the line number and first error message.
// If no error message is found, -1 and an empty string will be returned.
func ParsePythonError(msg, filename string) (int, int, string) {
	var (
		foundLineNumber bool   // ... ", line N"
		foundHat        bool   // ^
		errorMessage    string // Typically after "SyntaxError: "
		lineNumber      = -1   // The line number with the Python error, if any
		columnNumber    = -1   // The column number, from the position of the "^" in the error message, if any
		err             error  // Only used within the loop below
	)
	for line := range strings.SplitSeq(msg, "\n") {
		if foundHat && strings.Contains(line, ": ") {
			errorMessage = strings.SplitN(line, ": ", 2)[1]
			// break since this is usually the end of the approximately 5 line error message from Python
			break
		} else if foundLineNumber && len(line) > 4 {
			// de-indent the line before finding the hat column number
			if hatPos := strings.Index(line[4:], "^"); hatPos != -1 {
				foundHat = true
				// this is the column number (not index),
				columnNumber = hatPos + 1
			} else {
				continue
			}
		} else if strippedLine := strings.TrimSpace(line); strings.Contains(line, "\""+filename+"\"") || (strings.HasPrefix(strippedLine, "File ") && strings.Contains(line, "\", line ")) {
			fields := strings.Split(strippedLine, ", line ")
			if len(fields) < 2 {
				continue
			}
			lineNumber, err = strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			foundLineNumber = true
		} else if strippedLine := strings.TrimSpace(line); strings.Contains(line, "("+filename+", ") && strings.Contains(line, "Error: ") {
			fields := strings.SplitN(strippedLine, "Error: ", 2)
			errorMessageFileAndLine := fields[1]
			fields = strings.SplitN(errorMessageFileAndLine, "("+filename+", ", 2)
			errorMessage = fields[0]
			lineNumberString := fields[1]
			lineNumberString = strings.TrimPrefix(lineNumberString, "line ")
			lineNumberString = strings.TrimSuffix(lineNumberString, ")")
			if n, err := strconv.Atoi(lineNumberString); err == nil {
				lineNumber = n
			}
		}
	}

	// Strip the "(detected at line N)" message at the end
	if strings.HasSuffix(errorMessage, ")") && strings.Contains(errorMessage, "(detected at line ") {
		fields := strings.SplitN(errorMessage, "(detected at line ", 2)
		errorMessage = strings.TrimSpace(fields[0])
	}

	return lineNumber, columnNumber, errorMessage
}
