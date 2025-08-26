package main

import "strings"

// The special strings should be as unusual as possible, but short.
// It's important that the various characters will not be syntax highlighted separately.
const (
	escapedLessThan     = "æ010_" + "lt" + "_101æ"
	escapedGreaterThan  = "æ010_" + "gt" + "_101æ"
	escapedCommentStart = "____æ____"
	escapedCommentEnd   = "____ø____"
)

var (
	escapeReplacer     = strings.NewReplacer("<", escapedLessThan, ">", escapedGreaterThan)
	unEscapeReplacer   = strings.NewReplacer(escapedLessThan, "<", escapedGreaterThan, ">")
	shEscapeReplacer   = strings.NewReplacer("<", escapedLessThan, ">", escapedGreaterThan, "/*", escapedCommentStart, "*/", escapedCommentEnd)
	shUnEscapeReplacer = strings.NewReplacer(escapedLessThan, "<", escapedGreaterThan, ">", escapedCommentStart, "/*", escapedCommentEnd, "*/")
)

// Escape escapes < and > by replacing them with specialString1 and specialString2
func Escape(s string) string {
	return escapeReplacer.Replace(s)
}

// UnEscape unescapes specialString1 and specialString2 by replacing them with < and >
func UnEscape(s string) string {
	return unEscapeReplacer.Replace(s)
}

// ShEscape escapes < and > by replacing them with specialString1 and specialString2
// Also escapes /* and */
func ShEscape(s string) string {
	return shEscapeReplacer.Replace(s)
}

// ShUnEscape unescapes specialString1 and specialString2 by replacing them with < and >
// Also unescapes /* and */
func ShUnEscape(s string) string {
	return shUnEscapeReplacer.Replace(s)
}
