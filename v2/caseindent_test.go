package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func TestClosingBracketIndentation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		lines   []string
		closing rune
		want    string
	}{
		{"closing a function", []string{"func f() {", "\tfoo()", "\t\t"}, '}', ""},
		{"closing a nested block", []string{"func f() {", "\tif x {", "\t\tfoo()", "\t\t\t"}, '}', "\t"},
		{"closing a call spanning lines", []string{"\tfoo(", "\t\ta,", "\t\t"}, ')', "\t"},
		{"braces in strings and char literals are ignored", []string{"func f() {", "\tif c == '{' || s == \"}\" {", "\t\tfoo(\"{\")", "\t\t\t"}, '}', "\t"},
		{"braces in comments are ignored", []string{"func f() {", "\t// {", "\t\t"}, '}', ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.lines)
			e.mode = mode.Go
			placeCursor(e, len(tc.lines)-1, 0)
			got, found := e.closingBracketIndentation(tc.closing)
			if !found || got != tc.want {
				t.Errorf("got %q (found=%v), want %q", got, found, tc.want)
			}
		})
	}
	e := makeEditor([]string{"foo()", ""})
	e.mode = mode.Go
	placeCursor(e, 1, 0)
	if ws, found := e.closingBracketIndentation('}'); found {
		t.Errorf("found %q, but there is no open bracket", ws)
	}
}

func TestCaseIndentation(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
		want  string
	}{
		{
			"after an empty case, auto-indented one level too deep",
			[]string{"switch (x) {", "    case 1:", "        case"},
			"    ",
		},
		{
			"after an empty case, already dedented by hand",
			[]string{"switch (x) {", "    case 1:", "    case"},
			"    ",
		},
		{
			"after a case body",
			[]string{"switch (x) {", "    case 1:", "        foo();", "        break;", "        case"},
			"    ",
		},
		{
			"after a case with a brace in a char literal",
			[]string{"switch (c) {", "    case '{':", "        depth++;", "        break;", "    case '}':", "        depth--;", "        case"},
			"    ",
		},
		{
			"after a case with braces in a string",
			[]string{"switch (s) {", "    case \"{\":", "        foo(\"}}}\");", "        case"},
			"    ",
		},
		{
			"after a nested switch",
			[]string{"switch (a) {", "    case 1:", "        switch (b) {", "            case 10:", "                foo();", "        }", "        break;", "        case"},
			"    ",
		},
		{
			"after a comment line",
			[]string{"switch (x) {", "    case 1:", "        foo(); // {", "        // }", "        case"},
			"    ",
		},
		{
			"case inside a block comment is ignored",
			[]string{"switch (x) {", "    case 1:", "        /*", "  case 9:", "        */", "        case"},
			"    ",
		},
		{
			"after a default branch",
			[]string{"switch (x) {", "    default:", "        foo();", "        case"},
			"    ",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.lines)
			e.mode = mode.Java
			e.indentation = mode.TabsSpaces{Spaces: true, PerTab: 4}
			placeCursor(e, len(tc.lines)-1, 0)
			got, found := e.caseIndentation()
			if !found {
				t.Fatalf("no case indentation found")
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			e.alignCase()
			if line := e.CurrentLine(); line != tc.want+"case" {
				t.Errorf("aligned line = %q, want %q", line, tc.want+"case")
			}
		})
	}

	for _, tc := range []struct {
		name  string
		lines []string
		want  string
	}{
		{"first case, gofmt style", []string{"func f() {", "\tswitch x {", "\t\tcase"}, "\t"},
		{"second case, gofmt style", []string{"func f() {", "\tswitch x {", "\tcase 1:", "\t\tfoo()", "\t\tcase"}, "\t"},
		{"case in a select", []string{"func f() {", "\tselect {", "\t\tcase"}, "\t"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := makeEditor(tc.lines)
			e.mode = mode.Go
			e.indentation = mode.TabsSpaces{Spaces: false, PerTab: 4}
			placeCursor(e, len(tc.lines)-1, 0)
			got, found := e.caseIndentation()
			if !found || got != tc.want {
				t.Errorf("got %q (found=%v), want %q", got, found, tc.want)
			}
		})
	}

	for _, lines := range [][]string{
		{"int x = 1;", "case"},
		{"switch (x) {", "case"}, // the first case in C or Java may be at the same level as the switch, or one level deeper
	} {
		e := makeEditor(lines)
		e.mode = mode.Java
		placeCursor(e, len(lines)-1, 0)
		if ws, found := e.caseIndentation(); found {
			t.Errorf("%q: found a case indentation %q, but there is no sibling case", lines, ws)
		}
	}
}
