package main

import "testing"

func ExamplePrintln() {
	o := NewTextOutput(true, true)
	o.Println("hello")
	// Output:
	// hello
}

func TestTags(t *testing.T) {
	o := NewTextOutput(true, true)
	a := o.LightTags("<blue>hi</blue>")
	b := o.LightBlue("hi")
	if a != b {
		t.Fatal(a + " != " + b)
	}
}
