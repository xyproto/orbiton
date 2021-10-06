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
