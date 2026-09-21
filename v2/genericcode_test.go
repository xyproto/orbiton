package main

import (
	"testing"

	"github.com/xyproto/mode"
)

func TestGenericCode(t *testing.T) {
	for _, tc := range []struct {
		filename string
		m        mode.Mode
		binary   bool
		want     bool
	}{
		{"code.xyz", mode.Blank, false, true},
		{"code.groovy", mode.Blank, false, true},
		{"noext", mode.Blank, false, false},
		{"-", mode.Blank, false, false},
		{"notes.txt", mode.Text, false, false},
		{"README", mode.Markdown, false, false},
		{"main.go", mode.Go, false, false},
		{"image.bin", mode.Blank, true, false},
	} {
		e := NewSimpleEditor(80)
		e.filename = tc.filename
		e.mode = tc.m
		e.binaryFile = tc.binary
		if got := e.GenericCode(); got != tc.want {
			t.Errorf("GenericCode() for %q (%v) = %v, want %v", tc.filename, tc.m, got, tc.want)
		}
		wantProgramming := tc.want || ProgrammingLanguage(tc.m)
		if got := e.ProgrammingLanguage(); got != wantProgramming {
			t.Errorf("ProgrammingLanguage() for %q (%v) = %v, want %v", tc.filename, tc.m, got, wantProgramming)
		}
		if tc.want && e.NoSmartIndentation() {
			t.Errorf("NoSmartIndentation() for %q should be false", tc.filename)
		}
	}
}
