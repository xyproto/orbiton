package main

import (
	"path/filepath"
	"strconv"
	"strings"
)

// FilenameAndLineNumber will take the first two arguments and return a filename and a line number (can be 0)
// If the second argument is a number, that will be used as the line number. Or:
// If the second argument is a number prefixed with a "+", that will be used as the line number. Or:
// If the filename ends with a ":" and a number, that will be used as the line number.
func FilenameAndLineNumber(filename, lineNumberString string) (string, int) {
	lineNumber := 0
	if lineNumberConverted, err := strconv.Atoi(lineNumberString); err == nil { // no error
		lineNumber = lineNumberConverted
	} else if strings.HasPrefix(lineNumberString, "+") {
		if lineNumberConverted, err := strconv.Atoi(lineNumberString[1:]); err == nil { // no error
			lineNumber = lineNumberConverted
		}
	} else if strings.Contains(filepath.Base(filename), ":") {
		fields := strings.SplitN(filename, ":", 2)
		if lineNumberConverted, err := strconv.Atoi(fields[1]); err == nil { // no error
			lineNumber = lineNumberConverted
			filename = fields[0]
		}
	} else if strings.Contains(filepath.Base(filename), "+") {
		fields := strings.SplitN(filename, "+", 2)
		if lineNumberConverted, err := strconv.Atoi(fields[1]); err == nil { // no error
			lineNumber = lineNumberConverted
			filename = fields[0]
		}
	}
	return filename, lineNumber
}
