package main

import (
	"fmt"
	"os"
)

func ExampleEditor_BuildOrExport_GoError() {
	e := NewSimpleEditor(80)
	os.Chdir("tests")
	s, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err.go")
	os.Chdir("..")

	fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)

	// Output:
	// undefined: asdfasdf [performed action: true] [compiled OK: false]
}

func ExampleEditor_BuildOrExport_RustError() {
	e := NewSimpleEditor(80)
	os.Chdir("tests")
	s, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err.rs")
	os.Chdir("..")

	fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)

	// Output:
	// cannot find macro `rintln` in this scope [performed action: true] [compiled OK: false]
}
