package main

import "strings"

// The special strings should be as unusual as possible, but short.
// It's important that the various characters will not be syntax highlighted separately.
const (
	escapedLessThan    = "æ010_" + "lt" + "_101æ"
	escapedGreaterThan = "æ010_" + "gt" + "_101æ"
)

var (
	escapeReplacer   = strings.NewReplacer("<", escapedLessThan, ">", escapedGreaterThan)
	unEscapeReplacer = strings.NewReplacer(escapedLessThan, "<", escapedGreaterThan, ">")
)

// Escape escapes < and > by replacing them with specialString1 and specialString2
func Escape(s string) string {
	return escapeReplacer.Replace(s)
}

// UnEscape escapes specialString1 and specialString2 by replacing them with < and >
func UnEscape(s string) string {
	return unEscapeReplacer.Replace(s)
}
