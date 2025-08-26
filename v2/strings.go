package main

import (
	"regexp"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

// ansiRegex matches ANSI escape sequences
var (
	ansiRegex     *regexp.Regexp
	ansiRegexOnce sync.Once
)

// hasKey checks if the given string map contains the given key
func hasKey(m map[string]string, key string) bool {
	_, found := m[key]
	return found
}

// hasWords checks if a range of more than one letter is found
func hasWords(s string) bool {
	letterCount := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letterCount++
		} else {
			letterCount = 0
		}
		if letterCount > 1 {
			return true
		}
	}
	return false
}

// containsSubstring checks if the given string contains one of the given substrings
func containsSubstring(haystack string, substrings []string) bool {
	for _, ss := range substrings {
		if strings.Contains(haystack, ss) {
			return true
		}
	}
	return false
}

// allUpper checks if all letters in a string are uppercase
func allUpper(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// allLower checks if all letters in a string are lowercase
func allLower(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) && !unicode.IsLower(r) {
			return false
		}
	}
	return true
}

// capitalizeWords can change "john bob" to "John Bob"
func capitalizeWords(s string) string {
	words := strings.Fields(s)
	newWords := make([]string, len(words))
	for i, word := range words {
		if len(word) > 1 {
			capitalizedWord := strings.ToUpper(string(word[0])) + word[1:]
			newWords[i] = capitalizedWord
		} else {
			newWords[i] = word
		}
	}
	return strings.Join(newWords, " ")
}

// onlyAZaz checks if the given string only contains letters a-z and A-Z
func onlyAZaz(s string) bool {
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return true
}

// smartSplit will split a string on spaces, but only spaces that are not within [], () or {}
func smartSplit(s string) []string {
	// Define constants for states.
	const (
		Outside = iota
		InParentheses
		InBrackets
		InBraces
	)

	state := Outside
	var result []string
	var word strings.Builder

	for _, ch := range s {
		switch ch {
		case '(':
			if state == Outside {
				state = InParentheses
			}
			word.WriteRune(ch)
		case ')':
			if state == InParentheses {
				state = Outside
			}
			word.WriteRune(ch)
		case '[':
			if state == Outside {
				state = InBrackets
			}
			word.WriteRune(ch)
		case ']':
			if state == InBrackets {
				state = Outside
			}
			word.WriteRune(ch)
		case '{':
			if state == Outside {
				state = InBraces
			}
			word.WriteRune(ch)
		case '}':
			if state == InBraces {
				state = Outside
			}
			word.WriteRune(ch)
		case ' ':
			if state == Outside {
				// Only split on space if outside of any brackets, braces, or parentheses.
				result = append(result, word.String())
				word.Reset()
			} else {
				word.WriteRune(ch)
			}
		default:
			word.WriteRune(ch)
		}
	}

	// Append the last word.
	if word.Len() > 0 {
		result = append(result, word.String())
	}

	return result
}

// isAllowedFilenameChar checks if the given rune is allowed in a typical cross-platform filename
func isAllowedFilenameChar(r rune) bool {
	if unicode.IsLetter(r) || unicode.IsDigit(r) || isEmoji(r) {
		return true
	}
	switch r {
	case '.', ',', ':', '-', '+', '_', '/', '\\':
		return true
	default:
		return false
	}
}

// sanitizeFilename removes any character from the input string that is not part of a typical cross-platform filename
func sanitizeFilename(input string) string {
	result := make([]rune, 0, len(input))
	for _, r := range input {
		if isAllowedFilenameChar(r) {
			result = append(result, r)
		}
	}
	return string(result)
}

// getLeadingWhitespace returns the leading whitespace of the given string
func getLeadingWhitespace(line string) string {
	whitespace := make([]rune, 0, 8) // pre-allocate for the expected common case
	for _, char := range line {
		if unicode.IsSpace(char) {
			whitespace = append(whitespace, char)
		} else {
			break
		}
	}
	return string(whitespace)
}

func withinBackticks(line, what string) bool {
	first := []rune(what)[0]
	within := false
	lineRunes := []rune(line)
	whatRunes := []rune(what)

	for i, r := range lineRunes {
		if r == '`' { // `
			within = !within
			continue
		}
		if within && r == first {
			// check if the following runes also matches "what"
			// if they do, return true
			match := true
			for whatIndex, whatRune := range whatRunes {
				lineIndex := i + whatIndex
				if lineIndex >= len(lineRunes) {
					match = false
					break
				}
				if lineRunes[lineIndex] != whatRune {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// isEmoji checks if a rune is likely to be an emoji.
func isEmoji(r rune) bool {
	// Check if the rune falls within ranges that are likely to be used by emojis.
	return unicode.Is(unicode.S, r) || // Symbols
		unicode.Is(unicode.P, r) || // Punctuation
		r >= utf8.RuneSelf // Emojis are typically multi-byte characters in UTF-8
}

// trimRightSpace trims space but only at the right side of a string
func trimRightSpace(str string) string {
	return strings.TrimRightFunc(str, unicode.IsSpace)
}

func stripTerminalCodes(msg string) string {
	// Regular expression to match ANSI escape sequences
	ansiRegexOnce.Do(func() {
		ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
	})
	// Replace all occurrences with an empty string
	return ansiRegex.ReplaceAllString(msg, "")
}
