package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func TestStripDiffPrefixes(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode mode.Mode
		in   string
		want string
	}{
		{
			"added lines only",
			mode.Go,
			"+\tif err != nil {\n+\t\treturn err\n+\t}",
			"\tif err != nil {\n\t\treturn err\n\t}",
		},
		{
			"hunk with context",
			mode.C,
			" \tfoo();\n+\tbar();\n \tbaz();",
			"\tfoo();\n\tbar();\n\tbaz();",
		},
		{
			"blank lines are kept",
			mode.Go,
			"+func a() {}\n\n+func b() {}",
			"func a() {}\n\nfunc b() {}",
		},
		{
			// A hunk with removals is left alone rather than silently dropping lines
			"removed lines present",
			mode.Go,
			"+\tnewCall()\n-\toldCall()",
			"+\tnewCall()\n-\toldCall()",
		},
		{
			"context only, nothing added",
			mode.Go,
			" \tfoo();\n \tbar();",
			" \tfoo();\n \tbar();",
		},
		{
			"ordinary code",
			mode.Go,
			"func main() {\n\tprintln(1)\n}",
			"func main() {\n\tprintln(1)\n}",
		},
		{
			"single line",
			mode.Go,
			"+one line only",
			"+one line only",
		},
		{
			// "+ item" is a list, not a diff
			"markdown list",
			mode.Markdown,
			"+ apples\n+ oranges",
			"+ apples\n+ oranges",
		},
		{
			// Editing a patch means the prefixes are the content
			"diff mode",
			mode.Diff,
			"+\tadded();\n \tcontext();",
			"+\tadded();\n \tcontext();",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := NewSimpleEditor(80)
			e.mode = tc.mode
			if got := e.stripDiffPrefixes(tc.in); got != tc.want {
				t.Errorf("got  %q\nwant %q", got, tc.want)
			}
		})
	}
}
