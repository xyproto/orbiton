package main

import "testing"

func ExamplePrintln() {
	o := NewTextOutput(true, true)
	o.Println("hello")
	// Output:
	// hello
}

//func TestColorOn(t *testing.T) {
//	o := NewTextOutput(true, true)
//	a := o.ColorOn(1, 36) + "x" + o.ColorOff()
//	b := o.Cyan("x")
//	if a != b {
//		t.Fatal(a + " != " + b)
//	}
//}

func TestTags(t *testing.T) {
	o := NewTextOutput(true, true)
	a := o.LightTags("<blue>hi</blue>")
	b := o.LightBlue("hi")
	if a != b {
		t.Fatal(a + " != " + b)
	}
}
