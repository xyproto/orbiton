package main

import "strings"

// hasAnyPrefixWord checks if the given line is prefixed with any one of the given words
func hasAnyPrefixWord(line string, wordList []string) bool {
	for _, word := range wordList {
		if strings.HasPrefix(line, word+" ") {
			return true
		}
	}
	return false
}

// hasAnyPrefix checks if the given line is prefixed with any one of the given strings
func hasAnyPrefix(line string, stringList []string) bool {
	for _, s := range stringList {
		if strings.HasPrefix(line, s) {
			return true
		}
	}
	return false
}

// hasS checks if the given string slice contains the given string
func hasS(sl []string, s string) bool {
	for _, e := range sl {
		if e == s {
			return true
		}
	}
	return false
}

// equalStringSlices checks if two given string string slices are equal or not
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 { // lenb must also be 0 at this point
		return true
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// firstWordContainsOneOf checks if the first word of the given string contains
// any one of the given strings
func firstWordContainsOneOf(s string, sl []string) bool {
	if s == "" {
		return false
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return false
	}
	firstWord := fields[0]
	for _, e := range sl {
		if strings.Contains(firstWord, e) {
			return true
		}
	}
	return false
}

// hasSuffix checks if the given string end with one of the given suffixes
func hasSuffix(s string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(s, suffix) {
			return true
		}
	}
	return false
}

// filterS returns all strings that makes the function f return true
func filterS(sl []string, f func(string) bool) []string {
	results := make([]string, 0, len(sl)/4) // pre-allocate with estimated capacity
	for _, e := range sl {
		if f(e) {
			results = append(results, e)
		}
	}
	return results
}

// maxLength returns the length of the longest string.
// If the given slice is empty, then 0 is returned.
func maxLength(xs []string) int {
	if len(xs) == 0 {
		return 0
	}
	maxLen := len(xs[0]) // use the first string length as the initial max length
	for _, s := range xs[1:] {
		if l := len(s); l > maxLen {
			maxLen = l
		}
	}
	return maxLen
}
