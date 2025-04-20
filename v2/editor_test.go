package main

import (
	"fmt"
	"testing"

	"github.com/xyproto/mode"
)

func TestEditor(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertRune(nil, 'a')
	if e.String() != "a\n" {
		fmt.Println("Expected \"a\" and a newline, got:", e.String())
		t.Fail()
	}
}

// TestTrimRight tests the TrimRight method for removing trailing spaces.
func TestTrimRight(t *testing.T) {
	e := NewSimpleEditor(0)
	// Trim trailing spaces
	e.lines = map[int][]rune{0: []rune("foo   ")}
	changed := e.TrimRight(LineIndex(0))
	if !changed {
		t.Errorf("Expected TrimRight to report change for trailing spaces")
	}
	if got := string(e.lines[0]); got != "foo" {
		t.Errorf("TrimRight: expected 'foo', got '%s'", got)
	}
	// No trimming when no trailing spaces
	e.lines = map[int][]rune{1: []rune("bar")}
	changed = e.TrimRight(LineIndex(1))
	if changed {
		t.Errorf("Expected TrimRight to report no change for 'bar'")
	}
}

// TestFilenameLineColNumber tests parsing filenames with embedded line and column specifiers.
func TestFilenameLineColNumber(t *testing.T) {
	var fn string
	var ln LineNumber
	var cn ColNumber
	// Simple case
	fn, ln, cn = FilenameLineColNumber("file.go", "12", "3")
	if fn != "file.go" || ln != 12 || cn != 3 {
		t.Errorf("Expected ('file.go',12,3), got (%q,%d,%d)", fn, ln, cn)
	}
	// Prefixed plus
	fn, ln, cn = FilenameLineColNumber("file.go", "+7", "+9")
	if fn != "file.go" || ln != 7 || cn != 9 {
		t.Errorf("Expected ('file.go',7,9), got (%q,%d,%d)", fn, ln, cn)
	}
	// Filename with colon
	fn, ln, cn = FilenameLineColNumber("test.txt:34", "", "")
	if fn != "test.txt" || ln != 34 || cn != 0 {
		t.Errorf("Expected ('test.txt',34,0), got (%q,%d,%d)", fn, ln, cn)
	}
	// Filename with plus
	fn, ln, cn = FilenameLineColNumber("abc+5", "", "")
	if fn != "abc" || ln != 5 || cn != 0 {
		t.Errorf("Expected ('abc',5,0), got (%q,%d,%d)", fn, ln, cn)
	}
}

func ExampleEditor_InsertStringAndMove() {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "hello")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_Home() {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "llo")
	e.Home()
	e.InsertStringAndMove(nil, "he")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_End() {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "el")
	e.Home()
	e.InsertRune(nil, 'h')
	e.End(nil)
	e.InsertStringAndMove(nil, "lo")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_Next() {
	e := NewSimpleEditor(80)
	e.InsertStringAndMove(nil, "hllo")
	e.Home()
	e.Next(nil)
	e.InsertRune(nil, 'e')

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_InsertRune() {
	e := NewSimpleEditor(80)
	e.mode = mode.SQL

	e.InsertStringAndMove(nil, "text -- comment")
	e.Home()
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.InsertRune(nil, '\n')

	fmt.Println(e)
	// Output:
	// text
	//  -- comment
}

// TestTrimLeft tests the TrimLeft method for removing leading spaces.
func TestTrimLeft(t *testing.T) {
	e := NewSimpleEditor(0)
	// Test trimming leading spaces
	e.lines = map[int][]rune{0: []rune("   foo")}
	changed := e.TrimLeft(LineIndex(0))
	if !changed {
		t.Errorf("Expected TrimLeft to report change for leading spaces")
	}
	if got := string(e.lines[0]); got != "foo" {
		t.Errorf("TrimLeft: expected 'foo', got '%s'", got)
	}
	// Test no trimming when no leading spaces
	e.lines = map[int][]rune{1: []rune("bar")}
	changed = e.TrimLeft(LineIndex(1))
	if changed {
		t.Errorf("Expected TrimLeft to report no change for 'bar'")
	}
}

// TestStripSingleLineComment tests stripping of block and single-line comments.
func TestStripSingleLineComment(t *testing.T) {
	e := NewSimpleEditor(0)
	// Block comment stripping
	input := "code /* comment */"
	want := "code"
	if got := e.StripSingleLineComment(input); got != want {
		t.Errorf("StripSingleLineComment block: want '%s', got '%s'", want, got)
	}
	// Single-line comment stripping (default marker '//')
	input = "line // comment"
	want = "line"
	if got := e.StripSingleLineComment(input); got != want {
		t.Errorf("StripSingleLineComment line: want '%s', got '%s'", want, got)
	}
	// No comment remains unchanged
	input = "clean line"
	want = "clean line"
	if got := e.StripSingleLineComment(input); got != want {
		t.Errorf("StripSingleLineComment no comment: want '%s', got '%s'", want, got)
	}
}
