package main

import (
	"fmt"
	"os"
	"testing"
)

func ExampleEditor_BuildOrExport_goError() {
	e := NewSimpleEditor(80)
	e.mode, _ = detectEditorMode("err.go")

	os.Chdir("test")
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
	e.mode, _ = detectEditorMode("err.rs")

	os.Chdir("test")
	_, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err.rs")
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

func ExampleEditor_BuildOrExport_goTest() {
	e := NewSimpleEditor(80)
	e.mode, _ = detectEditorMode("err.go")

	os.Chdir("test")
	// The rename is so that "err.go" is not picked up by the CI tests
	os.Rename("err_test_go", "err_test.go")
	s, performedAction, compiledOK := e.BuildOrExport(nil, nil, "err_test.go")
	os.Rename("err_test.go", "err_test_go")
	os.Chdir("..")
	fmt.Printf("%s [performed action: %v] [compiled OK: %v]\n", s, performedAction, compiledOK)
	// Output:
	// Test failed: TestTest (0.00s) [performed action: true] [compiled OK: false]
}
