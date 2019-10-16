package main

import "strings"

// The special strings should be as unusual as possible, but short.
// It's important that the various characters will not be syntax highlighted separately.
const (
	specialString1 = "æøå_lt_æøå"
	specialString2 = "æøå_gt_æøå"
)

// Escape escapes < and > by replacing them with specialString1 and specialString2
func Escape(s string) string {
	return strings.Replace(strings.Replace(s, "<", specialString1, -1), ">", specialString2, -1)
}

// UnEscape escapes specialString1 and specialString2 by replacing them with < and >
func UnEscape(s string) string {
	return strings.Replace(strings.Replace(s, specialString1, "<", -1), specialString2, ">", -1)
}
