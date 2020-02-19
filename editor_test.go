package main

import (
	"fmt"
	"testing"
)

func TestEditor(t *testing.T) {
	e := NewSimpleEditor(80)
	e.InsertRune(nil, 'a')
	if e.String() != "a\n" {
		fmt.Println("Expected \"a\" and a newline, got:", e.String())
		t.Fail()
	}
}

func ExampleEditor_InsertString() {
	e := NewSimpleEditor(80)
	e.InsertString(nil, "hello")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_Home() {
	e := NewSimpleEditor(80)
	e.InsertString(nil, "llo")
	e.Home()
	e.InsertString(nil, "he")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_End() {
	e := NewSimpleEditor(80)
	e.InsertString(nil, "el")
	e.Home()
	e.InsertRune(nil, 'h')
	e.End()
	e.InsertString(nil, "lo")

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_Next() {
	e := NewSimpleEditor(80)
	e.InsertString(nil, "hllo")
	e.Home()
	e.Next(nil)
	e.InsertRune(nil, 'e')

	fmt.Println(e)
	// Output:
	// hello
}

func ExampleEditor_InsertString_wrap1() {
	e := NewSimpleEditor(12)
	e.InsertString(nil, "hello there")

	fmt.Println(e)
	// Output:
	// hello there
}

func ExampleEditor_InsertString_wrap2() {
	e := NewSimpleEditor(7)
	e.InsertString(nil, "hello there")

	fmt.Println(e)
	// Output:
	// hello
	// there
}

func ExampleEditor_InsertString_wrap3() {
	e := NewSimpleEditor(11)
	e.InsertString(nil, "hello there")

	fmt.Println(e)
	// Output:
	// hello there
}

func ExampleEditor_InsertString_wrap4() {
	e := NewSimpleEditor(5)
	e.InsertString(nil, "hello there")

	fmt.Println(e)
	// Output:
	// hello
	// there
}

func ExampleEditor_InsertString_wrap5() {
	e := NewSimpleEditor(9)
	e.InsertString(nil, "Hello odd")
	e.Home()
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.InsertRune(nil, 'T')

	fmt.Println(e)
	// Output:
	// Hello
	// Todd
}

func ExampleEditor_InsertString_wrap6() {
	e := NewSimpleEditor(12)
	e.InsertString(nil, "Hello there")
	e.NewLine(nil, nil)
	e.InsertString(nil, "This is text")

	fmt.Println(e)
	// Output:
	// Hello there
	// This is text
}

func ExampleEditor_InsertString_wrap7() {
	e := NewSimpleEditor(12)
	e.InsertString(nil, "Hello there")
	e.NewLine(nil, nil)
	e.InsertString(nil, "Yoda")
	e.Up(nil, nil)
	e.Home()
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.Next(nil)
	e.InsertRune(nil, 'y')
	e.Next(nil)
	e.InsertRune(nil, 'o')
	e.Next(nil)
	e.InsertRune(nil, 'u')
	e.Next(nil)
	e.InsertRune(nil, ' ')

	fmt.Println(e)
	// Output:
	// Hello you
	// there
	// Yoda
}
