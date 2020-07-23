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

// func ExampleEditor_Brackets() {
// 	e := NewSimpleEditor(80)
// 	e.InsertStringAndMove(nil, "if 1 + 1 == 3 {")
// 	e.InsertRune(nil, '\n')
// 	e.InsertRune(nil, '!')
// 	e.InsertRune(nil, '\n')
// 	e.InsertRune(nil, '}')
// 	e.InsertRune(nil, '\n')
// 	fmt.Println(e)
// 	// Output:
// 	// if 1 + 1 == 3 {
// 	//     !
// 	// }
// }

/*

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

	//lines := strings.Split(e.String(), "\n")
	//for i, line := range lines {
	//	fmt.Println(i, "|"+line+"|")
	//}

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
	e := NewSimpleEditor(5)
	e.InsertString(nil, "This is")

	//lines := strings.Split(e.String(), "\n")
	//for i, line := range lines {
	//	fmt.Println(i, "|"+line+"|")
	//}

	fmt.Println(e)
	// Output:
	// This
	// is
}

func ExampleEditor_InsertString_wrap8() {
	e := NewSimpleEditor(7)
	e.InsertString(nil, "and my name is")

	fmt.Println(e)
	// Output:
	// and
	// my name
	// is
}

func ExampleEditor_InsertString_wrap9() {
	e := NewSimpleEditor(7)
	e.InsertString(nil, "seem to")
	e.Prev(nil)
	e.Delete()
	e.InsertRune(nil, 'o')
	e.NewLine(nil, nil)
	e.InsertString(nil, "be")

	fmt.Println(e)
	// Output:
	// seem to
	// be
}

func ExampleEditor_InsertString_wrap10() {
	e := NewSimpleEditor(40)
	e.InsertString(nil, "hello there")
	e.Home()
	e.Next(nil)
	e.InsertString(nil, "disturbance")

	fmt.Println(e)
	// Output:
	// hdisturbanceello there
}

*/
