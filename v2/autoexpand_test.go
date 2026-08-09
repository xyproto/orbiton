package main

import (
	"strings"
	"testing"

	"github.com/xyproto/mode"
)

// firstLine is a test helper for reading back the first line of an editor
func firstLine(e *Editor) string {
	return strings.SplitN(e.String(), "\n", 2)[0]
}

func TestExpandKotlinConst(t *testing.T) {
	for _, tc := range []struct {
		m        mode.Mode
		line     string
		atEnd    bool
		expected string
	}{
		{mode.Kotlin, "const", true, "const val"},
		{mode.Kotlin, "    const", true, "    const val"},
		{mode.Kotlin, "const val", true, "const val"},
		{mode.Kotlin, "private const", true, "private const"},
		{mode.Kotlin, "const", false, "const"}, // not at the end of the line
		{mode.Java, "const", true, "const"},
	} {
		e := NewSimpleEditor(80)
		e.mode = tc.m
		e.LoadBytes([]byte(tc.line + "\n"))
		e.GoToLineNumber(1, nil, nil, false)
		if tc.atEnd {
			e.End(nil)
		} else {
			e.Home()
		}
		e.expandKotlinConst(nil)
		if got := firstLine(e); got != tc.expected {
			t.Errorf("%v %q (at end: %v) became %q, expected %q", tc.m, tc.line, tc.atEnd, got, tc.expected)
		}
	}
}
