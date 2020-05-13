package main

import (
	"fmt"
)

func ExampleEditor_BuildOrExport() {
	e := NewSimpleEditor(80)
	s, err := e.BuildOrExport(nil, nil, "tests/err.go")
	fmt.Println(s)
	fmt.Println(err)

	// Output:
	// Success
	// <nil>
}
