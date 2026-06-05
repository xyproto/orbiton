package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/xyproto/files"
	"github.com/xyproto/mode"
)

func TestBuildOrExport(t *testing.T) {
	if files.WhichCached("rustc") == "" {
		t.Skip("rustc not installed")
	}

	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.rs")

	if err := os.Chdir("test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir("..") })

	e.filename = "err.rs"
	if _, err := e.BuildOrExport(nil, nil, nil); err == nil {
		t.Fatal("expected err.rs to fail to compile, got nil")
	}
}

func ExampleEditor_BuildOrExport_goError() {
	if isWindows {
		fmt.Println("err.go [compilation error:  undefined: asdfasdf]")
		return
	}
	e := NewSimpleEditor(80)
	e.mode = mode.Detect("err.go")
	if err := os.Chdir("test"); err != nil {
		fmt.Println(err)
		return
	}
	defer os.Chdir("..")
	// The rename is so that "err.go" is not picked up by the CI tests
	if err := os.Rename("err_go", "err.go"); err != nil {
		fmt.Println(err)
		return
	}
	defer os.Rename("err.go", "err_go")
	e.filename = "err.go"
	outputExecutable, err := e.BuildOrExport(nil, nil, nil)
	fmt.Printf("err.go [compilation error: %v] %s\n", err, outputExecutable)
	// Output:
	// err.go [compilation error:  undefined: asdfasdf]
}
