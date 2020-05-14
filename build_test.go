package main

import (
	"fmt"
	"os"
	"testing"
)

func ExampleEditor_BuildOrExport_goError() {
	e := NewSimpleEditor(80)
	os.Chdir("tests")

	// The rename is so that "err.go" is not picked up by the CI tests
	os.Rename("err_go", "err.go")
	s, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err.go")
	os.Rename("err.go", "err_go")

	os.Chdir("..")
	fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)

	// Output:
	// undefined: asdfasdf [performed action: true] [compiled OK: false]
}

func TestBuildOrExport(t *testing.T) {
	e := NewSimpleEditor(80)
	os.Chdir("tests")
	_, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err.rs")
	os.Chdir("..")

	//fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)

	if which("rustc") != "" {
		//fmt.Println(s)
		if !performedAction {
			t.Fail()
		}
		if compiledOK {
			t.Fail()
		}

	} else {
		//fmt.Println(s)
		// silent compiler
		if performedAction {
			t.Fail()
		}
		if compiledOK {
			t.Fail()
		}
	}
}
