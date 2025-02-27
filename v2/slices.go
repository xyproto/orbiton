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
	var results []string
	for _, e := range sl {
		if f(e) {
			results = append(results, e)
		}
	}
	return results
}

// minMaxLength returns the length of the shortest and longest string
// If the given slice is empty, then 0,0 is returned.
func minMaxLength(xs []string) (int, int) {
	if len(xs) == 0 {
		return 0, 0 // can not find min and max string lengths of an empty slice
	}
	minLen := -1
	maxLen := -1
	for _, s := range xs {
		l := len(s)
		if minLen == -1 || l < minLen {
			minLen = l
		}
		if maxLen == -1 || l > maxLen {
			maxLen = l
		}
	}
	return minLen, maxLen
}
