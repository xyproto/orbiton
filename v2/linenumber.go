package main

import (
	"path/filepath"
	"strconv"
	"strings"
)

type (
	// LineNumber is a number from 1 to the last line (counts lines from 1)
	LineNumber int

	// LineIndex is a number from 0 to the last line (counts lines from 0)
	LineIndex int

	// CharacterPosition is a number from 0 to the last character (counts characters from 0)
	CharacterPosition int

	// ColNumber is a number from 1 and up (counts columns from 1)
	ColNumber int

	// ColIndex is a number from 0 and up (counts lines from 0)
	ColIndex int
)

// String converts a LineNumber to a string
func (ln LineNumber) String() string {
	return strconv.Itoa(int(ln))
}

// String converts a LineIndex to a string
func (li LineIndex) String() string {
	return strconv.Itoa(int(li))
}

// String converts a ColIndex to a string
func (ci ColIndex) String() string {
	return strconv.Itoa(int(ci))
}

// LineIndex converts a LineNumber to a LineIndex by subtracting 1
func (ln LineNumber) LineIndex() LineIndex {
	return LineIndex(ln - 1)
}

// ColIndex converts a ColNumber to a ColIndex by subtracting 1
func (cn ColNumber) ColIndex() ColIndex {
	return ColIndex(cn - 1)
}

// LineNumber converts a LineIndex to a LineNumber by adding 1
func (li LineIndex) LineNumber() LineNumber {
	return LineNumber(li + 1)
}

// ColNumber converts a ColIndex to a ColNumber by adding 1
func (ci ColIndex) ColNumber() ColNumber {
	return ColNumber(ci + 1)
}

// FilenameAndLineNumberAndColNumber will take the first two arguments and return a filename and a line number (can be 0)
// If the second argument is a number, that will be used as the line number. Or:
// If the second argument is a number prefixed with a "+", that will be used as the line number. Or:
// If the filename ends with a ":" and a number, that will be used as the line number.
func FilenameAndLineNumberAndColNumber(filename, lineNumberString, colNumberString string) (string, LineNumber, ColNumber) {
	lineNumber := 0
	colNumber := 0
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
	if colNumberConverted, err := strconv.Atoi(colNumberString); err == nil { // no error
		colNumber = colNumberConverted
	} else if strings.HasPrefix(colNumberString, "+") {
		if colNumberConverted, err := strconv.Atoi(colNumberString[1:]); err == nil { // no error
			colNumber = colNumberConverted
		}
	}
	return filename, LineNumber(lineNumber), ColNumber(colNumber)
}
