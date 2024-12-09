package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
)

func ExampleEditor_BuildOrExport_goError() {
	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.go")

	os.Chdir("test")
	// The rename is so that "err.go" is not picked up by the CI tests
	os.Rename("err_go", "err.go")
	e.filename = "err.go"
	outputExecutable, err := e.BuildOrExport(nil, nil, nil)
	os.Rename("err.go", "err_go")
	os.Chdir("..")
	fmt.Printf("err.go [compilation error: %v] %s\n", err, outputExecutable)
	// Output:
	// err.go [compilation error:  undefined: asdfasdf]
}

func TestBuildOrExport(t *testing.T) {
	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.rs")

	os.Chdir("test")
	e.filename = "err.rs"
	_, err := e.BuildOrExport(nil, nil, nil)

	os.Chdir("..")

	// fmt.Printf("err.rs [compilation error: %v] %s\n", err, outputExecutable)

	if files.WhichCached("rustc") != "" {
		// fmt.Println(err)
		if err == nil { // expected to fail, fail on success
			t.Fail()
		}
	} else {
		// fmt.Println(err)
		if err == nil { // expected to fail, fail on success
			t.Fail()
		}
	}
}
