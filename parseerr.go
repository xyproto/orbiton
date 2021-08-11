package main

import (
	"strconv"
	"strings"
)

// ParsePythonError parses a Python error message and returns the line number and first error message.
// If no error message is found, -1 and an empty string will be returned.
func ParsePythonError(msg, filename string) (int, string) {
	var (
		foundLineNumber bool   // ... ", line N"
		foundHat        bool   // ^
		errorMessage    string // Typically after "SyntaxError: "
		lineNumber      = -1   // The line number with the Python error, if any
		err             error  // Only used within the loop below
	)
	for _, line := range strings.Split(msg, "\n") {
		if foundHat && strings.Contains(line, ": ") {
			errorMessage = strings.SplitN(line, ": ", 2)[1]
			break
		} else if foundLineNumber {
			if strings.Contains(line, "^") {
				foundHat = true
			} else {
				continue
			}
		} else if strings.Contains(line, "\""+filename+"\"") {
			fields := strings.Split(line, ", line ")
			if len(fields) < 2 {
				continue
			}
			lineNumber, err = strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			foundLineNumber = true
		}
	}
	return lineNumber, errorMessage
}
