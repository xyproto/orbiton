package main

import (
	"fmt"
	"sort"
	"strings"
)

// Word is a type for a string in a list of strings, that may be quoted
type Word struct {
	s            string
	singleQuoted bool
	doubleQuoted bool
}

// Words is a slice of Word
type Words []Word

// Len helps make Words sortable.
// Len is the number of elements in the collection.
func (ws Words) Len() int {
	return len(ws)
}

// Less helps make Words sortable.
// Less reports whether the element with
// index i should sort before the element with index j.
func (ws Words) Less(i, j int) bool {
	return strings.ToLower(ws[i].s) < strings.ToLower(ws[j].s)
}

// Swap helps make Words sortable.
// Swap swaps the elements with indexes i and j.
func (ws Words) Swap(i, j int) {
	ws[i], ws[j] = ws[j], ws[i]
}

// sortStrings tries to identify and sort a given list of strings in a string.
// The strings are expected to be surrounded by either () or {},
// or that the entire given string is a list of strings.
// The strings may be space separated or comma separated (consistently).
// The strings may be single quoted, double quoted or none (may be non-consistent).
func sortStrings(line string) (string, error) {
	trimmedLine := strings.TrimSpace(line)

	// TODO: Bake the ({[ detection into the rune loop below

	// This check is very basic and will fail at strings like:
	// example=[[ "hello (0)", "hello (1)" ]]
	surroundedByCurly := strings.Count(trimmedLine, "{") == 1 && strings.Count(trimmedLine, "}") == 1
	surroundedByPar := strings.Count(trimmedLine, "(") == 1 && strings.Count(trimmedLine, ")") == 1
	surroundedBySquareBrackets := strings.Count(trimmedLine, "[") == 1 && strings.Count(trimmedLine, "]") == 1

	// Find the "center" string (a list of strings within {} or () on the current line)
	center := ""
	if surroundedByCurly {
		fields := strings.SplitN(trimmedLine, "{", 2)
		if len(fields) != 2 {
			return line, fmt.Errorf("curly brackets have an unusual order: %v", fields)
		}
		fields2 := strings.SplitN(fields[1], "}", 2)
		if len(fields2) != 2 {
			return line, fmt.Errorf("curly brackets have an unusual order: %v", fields2)
		}
		center = fields2[0]
	} else if surroundedByPar {
		fields := strings.SplitN(trimmedLine, "(", 2)
		if len(fields) != 2 {
			return line, fmt.Errorf("parentheses have an unusual order %v", fields)
		}
		fields2 := strings.SplitN(fields[1], ")", 2)
		if len(fields2) != 2 {
			return line, fmt.Errorf("parentheses have an unusual order %v", fields2)
		}
		center = fields2[0]
	} else if surroundedBySquareBrackets {
		fields := strings.SplitN(trimmedLine, "[", 2)
		if len(fields) != 2 {
			return line, fmt.Errorf("square brackets have an unusual order %v", fields)
		}
		fields2 := strings.SplitN(fields[1], "]", 2)
		if len(fields2) != 2 {
			return line, fmt.Errorf("square brackets have an unusual order %v", fields2)
		}
		center = fields2[0]
	} else {
		// Assume that the entire given string is a list of strings, without surrounding (), {} or []
		center = trimmedLine
	}

	// Okay, we have a "center" string containing a list of strings, that may be quoted, may be comma separated
	// The only thing sure to be consistent is either commas or spaces
	inSingleQuote := false
	inDoubleQuote := false
	commaCount := 0
	spaceCount := 0

	// Loop over the runes in the "center" string to count the commas
	// and spaces that are not within single or double quotes.
	for _, r := range center {
		switch r {
		case '\'': // single quote
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"': // double quote
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case ',':
			if !inSingleQuote && !inDoubleQuote {
				commaCount++
			}
		case ' ':
			if !inSingleQuote && !inDoubleQuote {
				spaceCount++
			}
		}
	}

	// Are we dealing with comma-separated or space-separated strings?
	// TODO: This will not work if the strings contains spaces
	commaSeparated := commaCount >= spaceCount

	// Split the string into a []string
	var fields []string
	if commaSeparated {
		fields = strings.Split(center, ",")
	} else {
		fields = strings.Split(center, " ")
	}

	// Convert the string fields to Words
	words := make(Words, len(fields))
	for i, field := range fields {
		trimmedElement := strings.TrimSpace(field)
		// Remove the trailing comma after the word, if any
		if strings.HasSuffix(trimmedElement, ",") {
			trimmedElement = strings.TrimSpace(trimmedElement[:len(trimmedElement)-1])
		}
		var w Word
		// Prepare a Word struct, depending on how this trimmed element is quoted
		if strings.HasPrefix(trimmedElement, "'") && strings.HasSuffix(trimmedElement, "'") {
			w.s = trimmedElement[1 : len(trimmedElement)-1]
			w.singleQuoted = true
			// fmt.Println("SINGLE QUOTED:", w.s)
		} else if strings.HasPrefix(trimmedElement, "\"") && strings.HasSuffix(trimmedElement, "\"") {
			w.s = trimmedElement[1 : len(trimmedElement)-1]
			w.doubleQuoted = true
			// fmt.Println("DOUBLE QUOTED:", w.s)
		} else {
			w.s = trimmedElement
			// fmt.Println("NOT QUOTED:", w.s)
		}
		// Save the Word
		words[i] = w
	}

	// fmt.Println("WORDS", words)

	// Sort the Words
	sort.Sort(words)

	// Join the words to a center string with the same type of separation and quoting as the original string
	var sb strings.Builder
	lastIndex := len(words) - 1
	for i, word := range words {
		if word.singleQuoted { // single quoted
			sb.WriteRune('\'')
			sb.WriteString(word.s)
			sb.WriteRune('\'')
		} else if word.doubleQuoted { // double quoted
			sb.WriteRune('"')
			sb.WriteString(word.s)
			sb.WriteRune('"')
		} else { // bare
			sb.WriteString(word.s)
		}
		if i == lastIndex {
			// Break before adding a ", " or " " suffix to the string
			break
		}
		if commaSeparated {
			sb.WriteRune(',')
		}
		// Write a space after the comma, or just a space if the strings are space separated.
		// This covers both.
		sb.WriteRune(' ')
	}
	newCenter := sb.String()

	// Okay, now replace the old list of strings with the new one, once
	return strings.Replace(line, center, newCenter, 1), nil
}

// SortStrings tries to find and sort a list of strings on the current line.
// The strings are expected to be surrounded by either () or {},
// or that the entire given string is a list of strings.
// The strings may be space separated or comma separated (consistently).
// The strings may be single quoted, double quoted or none (may be non-consistent).
func (e *Editor) SortStrings() error {
	// Sort the strings in the current line, return an error if there are problems
	newCurrentLine, err := sortStrings(e.CurrentLine())
	if err != nil {
		return err
	}
	// Use the sorted line and return
	e.SetLine(e.DataY(), newCurrentLine)
	return nil
}
