package main

import (
	"testing"

	"github.com/xyproto/mode"
	"github.com/xyproto/vt"
)

func TestReturnBeforeClosingBraceOnLastLine(t *testing.T) {
	for _, tc := range []struct {
		name    string
		initial []string
		y, x    int
		want    []string
		wantY   int
	}{
		{
			"before the final } on the last line",
			[]string{"func main() {", "\tfoo() }"},
			1, 7,
			[]string{"func main() {", "\tfoo()", "}"},
			2,
		},
		{
			"between the braces of an if block",
			[]string{"func main() {", "\tif x {}", "}"},
			1, 7,
			[]string{"func main() {", "\tif x {", "\t}", "}"},
			2,
		},
		{
			"splitting after an opening brace indents the rest",
			[]string{"func main() {", "\tif x {foo()}", "}"},
			1, 7,
			[]string{"func main() {", "\tif x {", "\t\tfoo()}", "}"},
			2,
		},
		{
			"before the final } on the only line",
			[]string{"func main() { foo() }"},
			0, 20,
			[]string{"func main() { foo()", "}"},
			1,
		},
		{
			"before the final } in the middle of the document",
			[]string{"func main() {", "\tfoo() }", "x"},
			1, 7,
			[]string{"func main() {", "\tfoo()", "}", "x"},
			2,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := vt.NewCanvasWithSize(80, 10)
			e := makeEditor(tc.initial)
			e.mode = mode.Go
			e.indentation = e.mode.TabsSpaces()
			e.GoToLineIndexAndColIndex(LineIndex(tc.y), ColIndex(tc.x), c, nil, false, true)
			e.ReturnPressed(c, nil, false)
			got := editorLines(e)
			if len(got) != len(tc.want) {
				t.Fatalf("lines = %q, want %q", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("line %d = %q, want %q (all: %q)", i, got[i], tc.want[i], got)
				}
			}
			if int(e.DataY()) != tc.wantY {
				t.Errorf("cursor line = %d, want %d", e.DataY(), tc.wantY)
			}
		})
	}
}
