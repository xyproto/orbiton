package main

import (
	"fmt"
)

func ExampleEditor_BuildOrExport() {
	e := NewSimpleEditor(80)
	s, err := e.BuildOrExport(nil, nil, "tests/err.go")

	if s != "" {
		fmt.Println("FAIL")
	}

	fmt.Println(err)

	// Output:
	// Could not compile
}
