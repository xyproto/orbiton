package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/xyproto/mode"
)

func ExampleEditor_BuildOrExport_goError() {
	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.go")

	os.Chdir("test")
	// The rename is so that "err.go" is not picked up by the CI tests
	os.Rename("err_go", "err.go")
	s, performedAction, compiledOK, outputExecutable := e.BuildOrExport(nil, nil, nil, "err.go", false)
	os.Rename("err.go", "err_go")
	os.Chdir("..")
	fmt.Printf("%s [performed action: %v] [compiled OK: %v] %s\n", s, performedAction, compiledOK, outputExecutable)
	// Output:
	// undefined: asdfasdf [performed action: true] [compiled OK: false]
}

func TestBuildOrExport(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.rs")

	os.Chdir("test")
	_, performedAction, compiledOK, _ := e.BuildOrExport(nil, nil, nil, "err.rs", false)
	os.Chdir("..")

	// fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)

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
